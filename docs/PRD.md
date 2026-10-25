# Outreach: Product Requirements (PRD)

Version 0.4 · 25 Oct 2026 · Owner: Anmol Madan · Pairs with HLD.md and LLD.md

The **release cut** (section 0) is the authority for what we build first. Later sections describe the product; they do not expand Now.

## 0. Release cut

First release: a send path, a read path, and a small writing path. It prevents four failures: a **limited mailbox**, a **duplicate or embarrassing email**, an **invented claim**, and a **missed reply**. Everything else is an interface, a column, or a documented skip so adding it later is an extension, not a rewrite.

| Area | Now | Next | Later |
|---|---|---|---|
| Product | Campaigns of enrolments; intent packs as data; contact + context required; JD optional | Angles picker; tasks in sequences | Catalogue import; tailored PDFs |
| Pacing | Rolling 24-hour `send_ledger`; mailbox and domain budgets | Domain stagger UI rules | — |
| Send | State machine with `unknown`; Message-ID before transport; reconciler; 5-minute undo hold; kill switch; sticky mailbox on follow-up; sender strategies | — | — |
| Health / warmup | Warmup profiles; health-gated ramp; idle days do not count | Complaint-like trend; domain score slowdown | Seed tests; Postmaster |
| Transports | SMTP/IMAP; Workspace Gmail API (one SA per domain); `AppendSent` / `FindSent`; OAuth stub | — | OAuth implementation; Outlook |
| Inbound | Cursor sync; rules; classify; policy in code; **reply drafts always to review** | Interview prep tasks | Calendar |
| Writing | Evidence library (user-confirmed); compose cites evidence IDs; embed evidence (pgvector); include-all when the set is small; deterministic gates; review at autonomy level 1 | `quality.review` model pass; voice from edits; document KB | — |
| Scheduling | Recipient timezone windows | Tasks engine | — |
| Compliance | Sender identity for commercial intents; RFC 8058 on commercial mail; provenance on contacts; erasure job | — | Region-rules engine; LIA records |
| Tenancy | Workspace per user (or invited team); composite FKs; RLS; server master key + disclosure that the host can decrypt | — | Extra DB roles; passphrase custody |
| Agents | — | — | MCP, tokens, webhooks |
| Observability | Ledger usage; unknown count; sync lag; health samples | DNS monitor | Seed placement |

**Not in Now:** reply auto-send, peer warmup pools, scraping, MCP, a job-board crawler, the CRM AI Workforce canvas, tools that send.

## 1. What this is

A one-to-one outreach product for a few hundred good conversations, not hundreds of thousands of sends. The operator picks people, the product drafts from **confirmed evidence**, sends from the operator’s own mailboxes at a human pace, follows up with something new, catches every reply, and records what happened.

It serves job search first (`job`, `referral`, cold to a hiring manager) and the same engine serves freelance pitches, intros, and custom asks through **intent packs**.

Positioning: the outreach layer job trackers lack and cold-email platforms are not built for. Small volume, high intent, every claim traceable, every send reversible for a few minutes, no way for the model to send.

Stack: Next.js + TypeScript web app, Go api and worker, PostgreSQL (River, pgvector). See HLD.md and LLD.md.

## 2. Users and tenancy

| User | Job |
|---|---|
| Operator (primary) | Get in front of the person who can say yes, without burning mailboxes |
| Freelancer / founder / networker | Same engine, different intent pack |
| Friend on the same server | Own workspace: own mailboxes, contacts, keys |
| Small team (roles already in the model) | Invite into a workspace (owner / member / viewer) |

A friend is a **new workspace**, not a member of yours, unless invited. Each workspace connects its own mailboxes. No shared Gmail farm.

## 3. Intent packs

A pack is data: writer rules, default sequence, attachment behaviour, compliance class, success definition. New packs do not change the engine.

| Intent | Writer | Default sequence | Class | Success |
|---|---|---|---|---|
| `job` | JD + fit if present; evidence; name the role | First mail; follow-up 5–7 days; one ~day 14 | Non-commercial; opt-out line | Interview |
| `referral` | Mutual connection; tiny ask | First; follow-up 4–5 days | Non-commercial | Introduction |
| `cold` | Why this person; one proof; ask | As `job` | Commercial if it pitches | Positive reply |
| `freelance` | Offer; one relevant piece of work | As `job` | Commercial | Call booked |
| `informational` | Curiosity; no job ask | One follow-up | Non-commercial | Conversation |
| `custom` | Context + profile | User-defined | Chosen by user | User-defined |

## 4. Principles

1. Evidence over eloquence. Unmapped claims do not send.
2. Few, right people. List quality beats volume.
3. A human decides the send (level 1 in Now) and can undo for a few minutes. **Replies never auto-send in Now.**
4. Deliverability is a feature: own health signals, domain budgets, refuse a launch that would damage a domain (preflight can wait until Next if DNS UI is thin; reservation still enforces caps).
5. Reputation is shared on a domain.
6. Honest numbers: sample sizes; no fake winners.
7. Provenance, opt-out, sender identity, erasure.
8. BYOK models. The model cannot send.
9. Tenant isolation is absolute.

## 5. Now capabilities

### 5.1 Contacts and companies

Typed, pasted, CSV (preview + mapping), emails found in a pasted JD. **Provenance required** (source, who, when). Dedup: one contact per email per workspace; one company per domain. Role class flagged (`hiring_manager`, `recruiter`, `generic_mailbox`, …). Generic `careers@` / `hr@` warned. Readiness: `ready` / `risky` / `blocked` (syntax, MX later as Next; suppression always). Company cooldown 30 days, override with reason. Timezone and region stored when known (drive send windows; region engine is Later).

### 5.2 Evidence and writing

Resume (PDF/DOCX) and links → model **proposes** evidence items → operator confirms. Unconfirmed items cannot be cited. Compose returns body + claim→evidence IDs. Deterministic gates: leftover `{{`, banned phrases, length (~125 words first mail), link count, attachment rules. Numbers, names, dates in the body must appear in a cited evidence item or in enrolment context. One regeneration; then show failures.

If the workspace has ~30 or fewer confirmed items, **include them all** in the prompt (no retrieval miss). Otherwise retrieve by embedding similarity (pgvector) under `workspace_id`.

Review queue, autonomy **level 1**: every outbound draft is approved by a human. Then a 5-minute undo hold (configurable; zero only by explicit choice).

Follow-ups: new value, same thread, **same mailbox**. If that mailbox is down, wait.

### 5.3 Campaigns

Enrolments: contact + context + intent; optional job and profile. Sequence: AI or template email, wait N days. Stop on reply, bounce, unsubscribe, OOO hold, manual stop. Cap new contacts per day. Sender strategy on the campaign (round robin, sequential, capacity first, weighted, domain spread, manual). Recipient local window when timezone is known; else mailbox window.

Templates: `{{contact.first_name}}`, `{{company.name}}`, `{{job.title}}`, `{{profile.name}}`, fallbacks `{{contact.first_name | there}}`. Invalid variables rejected on save.

### 5.4 Mailboxes and domains

- Personal Gmail: app password, SMTP + IMAP (Google may restrict this; transport is swappable).
- Workspace: **one service account per domain**, impersonate each user. Cannot impersonate `@gmail.com`.
- Other SMTP/IMAP; `AppendSent` when the provider does not file Sent.
- OAuth: empty slot in Now.

Sending domain is an object: caps, age, mailboxes. Rolling 24-hour mailbox **and** domain budgets. Health samples from bounces, deferrals, auth failures. Warmup is a **profile** (standard / conservative / fast / off / custom). Idle days do not count. Never start at the provider ceiling (500 personal / 500 trial Workspace / 2,000 Workspace). Default first days: 2, 3, 5, … Kill switch pauses all sends in the workspace.

### 5.5 Inbox

Cursor sync (IMAP UID, Gmail `historyId`) for inbox and sent. Threading by Message-ID / In-Reply-To / References, else Gmail thread id, else contact+subject. Rules: DSN, OOO, unsubscribe, complaint-like language. Else classify → **policy in code** (stage, stop, hold, suppress). Warranted replies → `reply.draft` → review. Notify on interview / interested.

### 5.6 Tracker

One row per enrolment. Stages: Drafted, Sent, Followed up, Replied, Closed; job-only Interview, Offer, Rejected. Outcome label on the enrolment (classifier guess, operator can confirm).

### 5.7 Compliance (thin)

Sender identity (name, reply address, postal address) required before **commercial** campaigns start; footer on those mails. Non-commercial: plain opt-out line. RFC 8058 on commercial class. All opt-out routes suppress within a minute. Erasure: delete contact, messages, attachments, evidence; keep hashed suppression. Audit approvals, overrides, sends, erasures.

### 5.8 AI

BYOK per workspace. Launch adapters: OpenAI, Google AI Studio, Anthropic, OpenAI-compatible base URL. Roles: `compose`, `analyze`, `classify`, `reply`, `embed` (`act` / `review` unused until Next). Anthropic cannot be mapped to `embed`. Tools: read and propose only. **No send tool.** JDs and replies wrapped as data.

## 6. Deliverability (Now numbers)

Caps are ceilings. Typical job outreach needs 30–50/day/mailbox at most.

| Type | Steady cap | Hourly | Gap | Provider ceiling (rolling 24h) |
|---|---|---|---|---|
| New personal Gmail | 20–40 | 10 | 4–9 min | ~500 recipients |
| Warmed personal Gmail | 50–80 | 15 | 2–6 min | ~500 |
| New Workspace | 20–40 | 15 | 2–6 min | 500 trial / 2,000 |
| Warmed Workspace | 80–150 | 25 | 1–4 min | 2,000 |
| Other SMTP | 50 | 15 | 2–6 min | Provider |

Domain default cap: 40/day if domain age &lt; 30 days, else 100 (editable). Auto-pause mailbox if bounce rate &gt; 3% with at least 20 sent. Auth failure → needs reconnect.

Warmup default curve (standard profile): day 1→2, 2→3, 3→5, then grow at most ~20%/day toward the mailbox limit, gated on bounces / deferrals / auth. Provider ceilings, domain budgets, auto-pause, and the kill switch are never turned off by a profile.

## 7. Autonomy (Now)

| Level | Behaviour |
|---|---|
| 1 Review each (**only level in Now**) | Operator approves every outbound draft. Replies always review. |
| 2–3 | Next (auto follow-ups / gated first touch). Not implemented. |

Every approved send enters the undo hold, then the ledger. Kill switch cancels queued work and blocks new reservations.

## 8. Measures

| Measure | Target |
|---|---|
| Hard bounce rate / mailbox | Under 2% |
| Replies detected vs mailbox | 100%; weekly UID reconcile |
| Sent mail with a claim and no evidence ID | 0 |
| Reply drafts sent without approval | 0 |
| Duplicate sends after timeout | 0 (unknown + reconciler) |
| Enrolment with JD → draft ready | Under 20 s |
| Mailboxes limited / blocked | Tracked, never hidden |

## 9. Build order

1. Foundations: monorepo, dbmate, auth, workspaces, secrets, mailboxes (SMTP/IMAP + test), domains.
2. Safe send: ledger, lock order, strategies, warmup profiles, hold, kill switch, suppression, one-off send.
3. Read: IMAP/Gmail sync, threading, bounces, inbox UI, reply from the product.
4. Write: evidence, compose, deterministic gates, review, classify + policy.
5. Campaigns: sequences, enrolments, recipient windows, tracker.
6. Workspace Gmail API, sender identity gate, erasure, commercial unsubscribe headers.

## 10. Non-goals

Mass mail; peer warmup pools; automating LinkedIn; scraping; owning a contact database; unattended AI send; mobile app; billing; public OAuth verification / CASA.

## 11. Open questions

1. First mailboxes: aged personal Gmail, a new outreach domain, or both?
2. Which intent packs ship on day one (`job` + `cold` vs all)?
3. Hosted-for-friends only, or OSS README at first release?
4. Postal address acceptable for commercial footers (home vs virtual)?
