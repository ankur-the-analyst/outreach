# Outreach: High-Level Design (HLD)

Version 0.4 · 25 Oct 2026 · Builds on PRD v0.4 · Supersedes HLD v0.2 / v0.3 drafts

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
            SMTP/IMAP   Gmail API (1 SA / Workspace domain)   LLM (BYOK)
```

**Jobs, not an event bus.** A write that needs work inserts a River job in the same transaction. No `events_outbox` dispatcher in Now. If subscribers multiply, an outbox is Next; handlers should stay idempotent so that swap is easy.

**Provider abstraction.** Core never switches on a vendor name. Adapters register capabilities (chat, JSON, embeddings). Workspace maps roles (`compose`, `analyze`, `classify`, `reply`, `embed`) to a provider + model, with one fallback. Launch: `openai`, `google` (AI Studio / Gemini), `anthropic`, `openai_compat`. `embed` cannot resolve to Anthropic.

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
| Transports | `Test`, `Send`, `AppendSent`, `FindSent`; OAuth stub returns not-configured |
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

Message-ID, threading headers, text then HTML, signature, resume attached for `job` else link unless chosen. `List-Unsubscribe` on commercial class. Sender identity footer on commercial class. Never `Precedence: bulk`. Optional `Feedback-ID` with a non-personal campaign id (no email in the header).

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

OAuth implementation is a stub that implements the interface and returns `ErrNotConfigured`.

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
| App passwords restricted | OAuth stub; Workspace SA |

## 12. What the LLD must specify

Now schema and jobs only, plus: `campaign_senders`, `warmup_profiles`, `sender_strategy` registry, `reply_mode` (`draft`/`off`), vector columns on `evidence_items`, `Embed` + `embed` role, `retrieve`, OAuth stub, kill switch, hold `release_at`, ledger states, selection protocol, `FORCE RLS` and connection GUC rules. No `kb_chunks`, no `region_rules`, no MCP, no three-role scheduler split.
