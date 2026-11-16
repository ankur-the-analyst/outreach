# Outreach: High-Level Design (HLD)

Version 0.6 · 12 Nov 2026 · Builds on PRD v0.6 · Supersedes HLD v0.5

**Frozen for M9–M11** (file on the mail, HTML templates, open/click). M7–M8 shipped. Behavior: this file. Tables: `docs/SCHEMA.md`. Tradeoffs: LLD v0.6 §15.

v0.5 remains the operator loop (tenancy, Gmail user OAuth, ledger, review, campaigns, structured CV). v0.6 does not add a job crawler, ATS form-fill, or auto-apply.

The **PRD release cut** is the authority for scope. This document is the technical design for **Now**, plus the interfaces Next/Later plug into. It does not require MCP, region-rules tables, extra DB roles, document KB, or reply auto-send in the first build.

## 1. Overview

Monorepo: Next.js + TypeScript web app; Go **api** (HTTP, auth, CRUD, public unsubscribe) and **worker** (schedule, send, sync, AI, health). PostgreSQL is the source of truth, the River queue, and pgvector. Object storage for resumes and attachments. Redis unused in v1.

```
  Browser --HTTPS--> apps/web (Next.js)
                         |
                    REST + SSE (OpenAPI)
                         |
                    cmd/api (Go)
                         |  SQL + River enqueue in the same transaction
                    PostgreSQL (+ pgvector)
                         |  workers claim jobs
                    cmd/worker (Go)
                         |
            SMTP/IMAP   Gmail user OAuth   Workspace SA (send stub)   LLM (BYOK)
```

**Jobs, not an event bus.** A write that needs work inserts a River job in the same transaction. No `events_outbox` dispatcher in Now. If subscribers multiply, an outbox is Next; handlers should stay idempotent so that swap is easy.

**Provider abstraction.** Core never switches on a vendor name. Adapters register capabilities (chat, JSON, embeddings). Workspace maps roles (`compose`, `analyze`, `classify`, `reply`, `embed`) to a provider + model, with one fallback. Launch: `openai`, `google` (AI Studio / Gemini), `anthropic`, `openrouter` (OpenAI-compatible API at a fixed host; the operator types the model id), `openai_compat`. `embed` cannot resolve to Anthropic.

## 2. Key decisions

| Decision | Choice |
|---|---|
| Tenancy | `workspace_id` everywhere; composite `(workspace_id, id)` FKs; RLS + `FORCE ROW LEVEL SECURITY`; api/worker DB users are not table owner |
| Pacing | `send_ledger` rolling 24h; mailbox then domain lock; domain budgets |
| Send safety | `held → reserved → sending → sent \| unknown \| failed`; Message-ID first; `FindSent` before retry |
| Follow-up | Sticky mailbox required; reschedule if unavailable |
| Rotation | Strategy registry; round robin from ledger (oldest last **first-touch**), not a counter |
| Warmup | `warmup_profiles` data; `ramp(profile, day, health)`; idle days ignored |
| Writing | Confirmed `evidence_items` + IDs; include-all if ≤ ~30; else exact pgvector under workspace filter |
| Replies | Classify → policy code → `reply.draft` → review. No auto-send. `reply_mode` column exists, allowed value in Now is `draft` (and `off`) |
| Transports | `Operations` (send + fetch); Gmail user OAuth live; Workspace SA send still stub |
| Fields | One catalog per object: built-in columns (read-only) plus workspace `field_defs`. Values for extras stay JSON on the row. Kinds are text, number, link, date, yes/no, dropdown. `{{user.*}}` merges the workspace owner's saved values at draft time. Not EAV. Not columns on global `users` |
| CV | JSON spec without email; `cv_renders` per mailbox (and optional job); never rewrite uploaded PDF bytes |
| AI | Propose-only tools; no send tool |
| Secrets | Envelope encryption; server master key; UI discloses the host can decrypt |
| Migrations | dbmate, SQL in git |
| Queue | River |

## 3. Processes

### 3.1 Web

Inbox, tracker, contacts/companies, campaigns, review, evidence, profiles, mailboxes/domains, suppression, settings (AI keys, sender identity, members, audit). No business logic in the web app.

### 3.2 API

`/api/v1`, public `/u/{token}` (and tracking later). Session cookie, active workspace on the session. Does not talk to SMTP/IMAP/LLMs except mailbox test (short timeout).

### 3.3 Worker

| Component | Now |
|---|---|
| Scheduler | Due enrolments/steps, recipient windows, skip if kill switch |
| Hold releaser | `held` past `release_at` → reserve (same checks as sender) |
| Sender | Lock, recompute strategy, reserve, render, transport, record |
| Reconciler | `unknown` → `FindSent`; sent or one retry same Message-ID |
| IMAP / Gmail sync | IDLE + UID; `history.list` |
| Inbound | Thread, rules, classify job, policy, reply.draft |
| AI | evidence.propose, analyze, fit, compose |
| Health | Samples; pause; warmup advance |
| Housekeeping | Stale reservations (&gt;10 min), UID reconcile, erasure |
| DNS / seed / MCP / tasks | Not Now |

## 4. Send pipeline

### 4.1 Selection protocol (avoids deadlock and double-pick)

Never hold two mailbox locks.

1. Read eligible pool members (healthy/warming, in window, under budgets from ledger states that count — see 4.2). No lock.
2. Order by campaign `sender_strategy`.
3. For each candidate in order: `BEGIN`; `SELECT mailbox FOR UPDATE`; `SELECT sending_domain FOR UPDATE`; **recompute** eligibility and whether this mailbox is still the strategy winner; if yes, insert ledger `reserved`; `COMMIT`; if no, `ROLLBACK` and try the next.
4. Sticky / manual pin overrides strategy. Follow-ups only use the sticky mailbox (reschedule, never rotate).

Round robin: eligible mailbox whose last **first-touch** ledger row (counting states in 4.2) is oldest. Sequential: configured order, first still eligible. Capacity first: most remaining budget. Weighted: weights on `campaign_senders`. Domain spread: alternate domains, then RR inside. Manual: `campaign_senders` assignment or contact pin.

### 4.2 Ledger states that count

Toward rolling 24h mailbox and domain budgets **and** toward round-robin “last first-touch”: `held`, `reserved`, `sending`, `sent`, `unknown`.

`released` and `cancelled` and `failed` do not count.

### 4.3 State machine

```
draft approved → held (release_at = now + undo) → reserved → sending → sent
                      |                         |            |
                   cancelled                  released    unknown → sent if FindSent
                   (back to review)                         → retry once, same Message-ID
                                                             → failed
```

- Message-ID generated and stored **before** `Send`.
- River unique on `(enrolment_id, step_position)`; retries mutate the **same** ledger row.
- **`Send` OK + `AppendSent` fail = `sent`.** Append is repair. Log it.
- `unknown`: timeout or connection loss after the body was handed to the transport. Never retry blindly. Reconciler searches by Message-ID (Sent and, for Gmail, a wider search) for **15 minutes**, then one retry, then `failed`.
- Kill switch: workspace flag checked in hold releaser and reservation. In-flight `sending` may finish; nothing new reserves.

### 4.4 MIME

Message-ID, threading headers, text then HTML, signature. `List-Unsubscribe` on commercial class. Sender identity footer on commercial class. Never `Precedence: bulk`. Optional `Feedback-ID` with a non-personal campaign id (no email in the header).

**Attachments (v0.5):** outer `multipart/mixed`; inner `multipart/alternative` for text/HTML; then one file part. Built **after** the ledger has a mailbox (`reserved` / `sending`), never at first compose. The attached CV’s visible contact email is the sending mailbox’s `From`, not the profile’s personal address. Job pack default: attach. Other packs: no file unless the campaign opts in. Gmail raw payload must stay under provider size (fail closed in review, do not send a truncated PDF).

### 4.5 Warmup

`effective_cap = min(ramp(profile, sending_day, health), mailbox.daily_limit, hard_ceiling, remaining_domain_budget)`.

A sending day counts only if the mailbox used ≥ half of that day’s cap. Gates: bounce / deferral / auth over last three sending days (PRD §6). Profiles cannot disable ceilings, domain budgets, auto-pause, or the kill switch.

## 5. Transports and sync

```go
type Transport interface {
    Test(ctx context.Context) error
    Send(ctx context.Context, m *mime.Built) (SendResult, error)
    AppendSent(ctx context.Context, m *mime.Built) error
    FindSent(ctx context.Context, messageID string) (bool, error)
}
// Read/IDLE live in package sync, not on this interface.
```

Gmail user OAuth implements send + inbox poll via `Operations`. Workspace domain service-account send remains unwired (`ErrNotConfigured` / not wired). Do not switch core on provider names; register `(authKind, provider)`.

IMAP: UID + UIDVALIDITY; IDLE on INBOX; Sent catch-up; one supervisor per mailbox (advisory lock); max two connections per Gmail mailbox. Gmail API: `history.list` from stored id; 404 → bounded resync. Dedup `(mailbox_id, rfc_message_id)`.

## 6. Inbound

1. Thread match (In-Reply-To / References, Gmail thread, subject+contact 30 days).
2. Rules: DSN (hard suppress; soft ×3 → hard), OOO hold, unsubscribe, complaint-like → suppress and pause campaign if clustered.
3. Else `inbound.classify` (JSON).
4. **Policy table in code** (not the model): stage, stop, hold, suppress, maybe enqueue `reply.draft`.
5. Reply drafts always `pending` review. No path from inbound text to `Send` except a human approval of that draft, then the same hold → ledger path as outbound.

Inbound bodies wrapped as data. A proposal that names a **new** recipient is never auto-accepted (column on the draft; Now there is no auto anyway).

`mailboxes.reply_mode`: `draft` (default) or `off`. Value `auto` is rejected by the API until Next.

## 7. Evidence and retrieval (Now)

```
resume/links → evidence.propose → user confirms → evidence_items
enrolment → (JD? analyze + fit vs evidence IDs) → compose (claims + evidence IDs) → gates → review
```

`evidence_items.embedding` filled at confirm/edit via `embed` role. `retrieve(ctx)`: if confirmed count ≤ 30, return all; else exact cosine (`<=>`) `WHERE workspace_id = $1 AND embedding_model = $current`. No HNSW in Now. Re-embed job on model change writes a new column then swaps; queries filter `embedding_model`. Do not index replies or JDs.

Deterministic claim check: each body number/name/date must appear in a cited item or enrolment context. Composer may only cite IDs `retrieve` returned.

Document tables (`kb_sources`, chunks, RRF) are **not created** in Now. `drafts.retrieval` JSON exists so a later retriever can record passage IDs.

## 8. Security and tenancy

- Composite FKs so a child row cannot point at another workspace’s parent.
- RLS on `app.workspace_id` with **FORCE**. Migrations and dbmate run as owner; **api and worker roles do not own tables** and have no `BYPASSRLS`.
- Each River job carries `workspace_id`, sets it on the connection at start, and resets it before the connection returns to the pool.
- Envelope encryption; keys never returned (hint last 4).
- SSRF guard on JD URLs (5s, 2MB, re-check IP on redirect).
- HMAC unsubscribe tokens; public POST idempotent and rate-limited.

Scheduler in Now: worker iterates due work **per job already scoped to a workspace** (River payload includes `workspace_id`). Do not run a global “select all enrolments” as table owner.

## 9. Observability

JSON logs with request/job ids. Metrics: sends by ledger state, budget usage, unknown count and time-to-resolve, IMAP lag, AI latency/tokens/cost, empty retrieve rate. Daily UID reconcile.

## 10. Scale

Tens of workspaces on one Compose stack; ~50 mailboxes per workspace; tens to a few hundred sends/day/workspace. Verify Gmail API quota before Workspace launch.

## 11. Risks (Now)

| Risk | Mitigation |
|---|---|
| Duplicate after timeout | `unknown`, FindSent, same Message-ID, AppendSent ≠ delivery |
| RR double-pick | Recompute after lock; count held/reserved |
| RLS bypass | FORCE RLS; non-owner DB roles; job-scoped GUC |
| Invented claims | Evidence IDs + deterministic match + review |
| Injection | Data wrapping; no send tool; replies to review |
| Host can decrypt friends | Disclosure in settings; custody Later |
| App passwords restricted | Gmail user OAuth; Workspace SA later |
| Silent fake compose | No binding / no key → operator error, not `ai.Fake` copy |
| Wrong CSV column | Mapping required when headers are not aliases; preview shows sample cells |
| Template leftover `{{` | Reject unknown vars on save; gates on render |
| Start with empty From | Campaign start requires ≥1 mailbox (senders or pool members) |
| CV email ≠ From | Render PDF from structured CV after mailbox is known |
| Tailor invents jobs | Only confirmed evidence + enrolment context; operator sees the PDF |

## 12. What the LLD must specify

Now schema and jobs: see applied migrations. v0.5 additive DDL is **`docs/SCHEMA.md`**. Tradeoffs LLD §14. No `kb_chunks`, no `region_rules`, no MCP, no ATS crawl.

## 13. Operator loop (v0.5)

Shipped: auth, RLS, Gmail OAuth send/sync, ledger, review, contacts CSV (header aliases only), campaigns, fake/default compose writer, profile **paste** resume, MIME `multipart/alternative` with no file parts.

Goal: one job-outreach send from the connected Gmail that is cited, approved, and (for job) carries a CV whose contact email is that Gmail.

```
CSV (mapped) → contacts
profile + structured CV
campaign (job) + From (mailbox or group)
  ├ template render  OR  compose(BYOK) + gates
  ├ review (body + which file)
  └ reserve mailbox → render PDF(From) → MIME mixed → Send
```

### 13.1 Compose bind

Workspace settings already store kind + sealed key. `cmd/api` and `cmd/worker` still construct `ai.Fake` as the default writer; `writer()` falls back to Fake on `ErrNoBinding` / `ErrNoKey`.

**Rule:** if the workspace has no compose binding or no key, compose **fails** with an operator error (“set a model and key in Settings”). Fake is only for tests and an explicit `kind=fake` binding. Campaign AI steps enqueue compose; same rule. Do not send Fake copy in production.

**Breaks:** operator thinks Gmail is the problem; drafts all identical. **Fix:** fail closed; log `google_oauth` already proves config at boot — compose should log `compose_provider` kind, never the key.

### 13.2 Import mapping

Preview returns header names + first row samples. Import body includes `columns` mapping CSV headers → field keys (builtins or `field_defs.key`). Email column required. Unmapped fields empty. Persist mapping in the browser for the session only (Now). Contact + member defs are required in this cut. See LLD §14 and SCHEMA.md.

**Breaks:** `Work Email` never matches `email` → all rows blocked, or the first column is treated as email. **Fix:** if no email alias matches, do not guess; UI must map. Import still requires provenance.

### 13.3 Templates

`templates` workspace table: name, subject, body. Vars: `{{contact.first_name}}`, `{{company.name}}`, `{{job.title}}`, `{{profile.name}}`, `{{contact.first_name | there}}`. Unknown vars rejected **on save**. Campaign email step: `content=template` + `template_id`, or `content=ai`. Render at draft-create; leftover `{{` still hits existing gates.

**Breaks:** template saved, contact missing first name → empty greeting; AI and template both ignored. **Fix:** fallbacks in the pipe syntax; empty required var is a gate failure on that draft, not a campaign-wide crash.

### 13.8 One field list

The operator sees built-in columns and the fields they added as one list. Built-ins are not rows in `field_defs` and cannot be renamed or deleted. Adding a field picks a type the schema already allows (`text`, `number`, `url`, `date`, `boolean`, `select`). A dropdown needs at least one choice. Contact extras go into the draft notes. Member extras are filled into `{{user.key}}` from the workspace owner's saved values when the draft is written (there is no approver yet). Pipeline fields such as status, priority, and source are not part of this list. See LLD §14.2.

### 13.4 From on start

Start is illegal if `campaign_senders` is empty **and** the pool has no members. UI already creates a campaign with a mailbox or group; groups can exist with zero mailboxes.

**Breaks:** “Job Application / No mailboxes” then Start queues work that never reserves. **Fix:** API 400 on start; operator copy: add a mailbox to this campaign or group.

### 13.5 CV as structured data, PDF at send

Do **not** byte-edit an uploaded PDF to change the email (layout breaks, text is not a field).

Canonical CV is JSON (`cv_documents.spec`) **without email**. PDF is `cv_renders` for that mailbox (optional `job_id` when tailored). Upload is evidence input (`resumes`), not the attached bytes. `drafts.attachment_render_id` points at the render. At `sending`, `from_email` must equal the ledger mailbox. If draft mailbox ≠ reserved mailbox, re-render or fail. Review shows filename + From.

**Breaks:** attach a PDF that still lists a personal Gmail; two NULLs sneak past a UNIQUE on `job_id`. **Fix:** render after mailbox is the one that will send; partial unique indexes (SCHEMA.md). Job campaign without a render does not call `Send`.

### 13.6 Tailor per JD

Same JSON, different **emphasis**: reorder/drop bullets that are in confirmed evidence; do not add employers, dates, or metrics. JD is enrolment context. Model proposes a CV JSON patch; gates: every number/name/date in the PDF text must appear in evidence or context (same idea as body gates). Operator reviews body **and** PDF. One regen.

**Breaks:** model invents a title; PDF and body disagree. **Fix:** fail the draft; do not attach.

## 14. Out of this cut

JD URL fetch / catalogue API, Workspace SA send, erasure-vs-evidence polish, public OAuth verification, column-mapping saved on the workspace, HTML/DOCX **from Word**. Freehire is a possible **JD source later**, not a dependency. Attachment **download** receipts (clients do not report them).

## 15. Files, HTML templates, tracking (v0.6)

Build **in this order**. Campaigns as the daily loop wait until M9 can put a real file on the mail.

```
M9  upload PDF bytes → durable store → MIME attach (original)
M10 design_json canvas → body_html ({{vars}} still gated)
M11 inject pixel + click wraps on HTML → public /t/* → tracker
```

M10 HTML is what M11 rewrites. M9 does not need HTML. Do not start M11 before M10 saves `body_html`.

### 15.1 M9 — Original resume on the mail

**Operator:** upload a PDF (or DOCX) on the profile; job campaigns attach **that file**. Review shows the filename. We still **do not rewrite** the PDF (HLD 13.5). If the file lists another email, that is visible to the operator; it is not a silent stamp.

**Not:** replace structured CV. `cv_renders` stay for `attach=render` (mailbox From on the file). New campaign field `attach`: `none` | `original` | `render`. Job create defaults to `original`. Other types default `none` (opt in).

**Storage:** `Memory` cannot serve original bytes across api/worker. Disk (both processes, same `STORAGE_PATH`) is Now; S3 later behind the same `storage.Store`. Upload stores **file bytes** at `resumes.file_key`, not UTF-8 of the extracted text. Text stays in `text_content` for evidence.

**Send:** after mailbox is known, `Files` loads original by `drafts.attachment_resume_id` (latest profile resume with bytes). Missing blob → no send (fail closed). Size still `mime.MaxBytes`. Tailor/gates apply to the **body**, not to rewriting the attached PDF.

**Breaks:** worker 404 on the PDF because it lived in api RAM. **Fix:** disk before attach. **Breaks:** paste-only resume has no bytes. **Fix:** operator copy “upload a PDF to attach”; `attach=original` without bytes is 400 on start / skip file with Review warning — prefer 400 on start.

### 15.2 M10 — Template canvas

**Operator:** drag blocks (heading, text, button, image, row, footer) into a canvas; save a named email; campaign step “saved email” uses it. Merge tags stay `{{contact.first_name}}` etc. Unknown tags fail **on save**.

**Source of truth:** `templates.design_json` `{settings, elements}` (same shape as `leads_crm_1` email-builder). `body_html` is compiled on save (`generateHTML`). `body_text` is a plain-text fallback generated from the same tree (or operator-edited). Theme: `docs/design/theme.css` on the **app chrome**; email HTML uses inline styles from the canvas, not our CSS file.

**Web:** port the CRM canvas (dnd-kit), not Unlayer. Do not take SweetAlert / Phosphor as a new system; use existing shadcn + tokens.

**Draft:** render vars into html+text at draft-create; leftover `{{` still gated. AI compose unchanged (may still emit simple HTML).

**Breaks:** canvas saves `{{nope}}` inside a button URL. **Fix:** walk all string fields in `design_json` with the same var parser as body.

### 15.3 M11 — Open and click

**Operator:** on Tracker / Review, this person **opened** or **clicked** (time). Campaign can turn tracking off. Unsubscribe links are never rewritten. We do **not** claim “they downloaded the CV.”

**How:** if the campaign has `track_opens` / `track_clicks` and the message has HTML: HMAC token (same family as `/u/{token}`), `PUBLIC_TRACK_HOST`. `GET /t/o/{token}` → 1×1 GIF, insert `mail_events` kind `open`. `GET /t/c/{token}` → 302 to original URL, kind `click`. Unique first open/click denorm on the person in this campaign (`opened_at` / `clicked_at`). Prefetch and privacy proxies may inflate opens — show as “Opened”, not “read.”

**Inject** at MIME build (worker), after template/AI HTML exists, never at first compose. Skip `mailto`, `#`, `/u/`, existing `/t/`.

**Breaks:** tracking host is localhost in production mail. **Fix:** `PUBLIC_TRACK_HOST` required when flags on; fail closed at send, operator error. **Breaks:** Gmail image proxy. **Fix:** still record the proxy fetch as opened; do not invent a second signal.
