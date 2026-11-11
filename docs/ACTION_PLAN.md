# Action plan

Authority: PRD/HLD/LLD **v0.6** (frozen for M9–M11). DDL: `docs/SCHEMA.md`. Theme: `docs/design/theme.css`. Cadence: one **slice** at a time (learn Go on each), then stop. M7–M8 shipped.

Not in this plan: auto-reply, Outlook, MCP, region-rules, document KB, `quality.review`, peer warmup, scraping, ATS crawl, billing, dark mode. Gmail **user** OAuth send/read is shipped. Workspace SA send is still out.

---

## How a slice works

1. Name the slice and the Go it will teach.
2. Implement only that slice until it runs.
3. If the slice added behavior or schema: **then** unit tests, then automated tests. Scaffold-only slices (Makefile, empty page) skip this.
4. Stop. You run it, we talk through the code, then the next slice.

A **milestone** is not done until its “Done when” holds **and** its tests are green (`make test`).

### Tests

| Kind | When | What |
|---|---|---|
| Unit | After a slice that added logic | Table-driven, same package, no network. Crypto, `ramp()`, gates, policy, MIME, strategies. |
| Automated | After a slice that touched DB, HTTP, or mail | testcontainers or Compose: dbmate up/down, RLS, lock protocol, ledger, Mailpit send/unknown. LLD §12 are blocking before M3 is closed. |

`make test` runs unit. `make test-int` runs automated. CI later runs both.

### Migrations (dbmate)

- Every schema change is a new file `backend/migrations/YYYYMMDDHHMMSS_name.sql`.
- Both `-- migrate:up` and `-- migrate:down`. Down must leave the DB as it was.
- Applied migrations are immutable. Fix forward with a new file.
- App and worker never migrate on boot; `make migrate` (dbmate up) is explicit.
- sqlc generates from the current schema after migrate, it does not own DDL.
- First real migration is M0.2 (roles, workspaces, users, sessions, FORCE RLS).

---

## M0 — Repo and empty shell

**Goal:** One git history, something that runs.

| Slice | Build | Go to notice | Done when |
|---|---|---|---|
| **M0.1** Tooling | `git init`, `.gitignore`, `docker-compose` Postgres 16 + pgvector, `Makefile`, `backend/go.mod`, `cmd/api` `GET /health` (JSON, no DB yet) | `package main`, `net/http`, `log/slog`, modules | `make up` then curl health returns 200 |
| **M0.2** dbmate | Roles `outreach` / `outreach_app`; migration 1: `workspaces`, `users`, `workspace_members`, `sessions`; `FORCE RLS`; `make migrate` | SQL in git, not GORM auto-migrate | `dbmate up` and `down` on Compose |
| **M0.3** sqlc + health DB | sqlc config, ping on `/health` (`SELECT 1`), config from env | `database/sql` or pgx, `context`, envconfig | Health fails if Postgres is down |
| **M0.4** Web shell | `apps/web` Next App Router, shadcn, copy `docs/design/theme.css` into `globals.css`, empty layout (no dashboard chrome yet) | — | Dev server shows themed blank page |

**Out:** River, auth UI, mailboxes.

---

## M1 — Auth, tenancy, secrets

**Goal:** A user owns a workspace; the DB will not leak another workspace’s rows.

| Slice | Build | Go to notice | Done when |
|---|---|---|---|
| **M1.1** Crypto | Envelope `Seal`/`Open`; `MASTER_KEYS`; table `secrets` | `crypto/aes`, interfaces at the caller | Round-trip test; wrong key fails |
| **M1.2** Auth | Register/login (argon2id), session cookie, `cmd/api` middleware | wrapping errors `%w`, `http.Handler` | Session required on `/api/v1/*` except health |
| **M1.3** Workspace GUC | `InTx` sets `app.workspace_id`; reset on conn return; app role ≠ table owner | `pgx` pool, defer | RLS test: wrong GUC → 0 rows |
| **M1.4** Members + kill switch | Workspace CRUD, invite member, `sends_paused` | constructors, no globals | Owner can pause; member cannot delete workspace |
| **M1.5** Shell chrome | App layout: sidebar nav stubs, theme tokens, login page | — | Logged-in empty Overview; nav goes nowhere useful yet |

**Out:** Google OIDC can wait until after password works. OIDC is a later M1 slice if needed.

---

## M2 — Mailboxes, domains, warmup (no send yet)

**Goal:** Connect a mailbox, test it, store domain + warmup profile. Transport is an interface.

| Slice | Build | Go to notice | Done when |
|---|---|---|---|
| **M2.1** Domain + warmup tables | `sending_domains`, `warmup_profiles` seed (standard / conservative / fast / off), `ramp()` pure function | table tests, `json.Unmarshal` spec | Idle day does not increment; day-1 cap 2 |
| **M2.2** Transport interface | `Test`/`Send`/`AppendSent`/`FindSent`; SMTP/IMAP adapter; Mailpit in Compose | accept interface, return struct | `Test` against Mailpit succeeds |
| **M2.3** Mailbox CRUD | Connect, secret seal, pause, reconnect, assign warmup + domain | composite FKs in sqlc | Cannot attach SA to `@gmail.com` |
| **M2.4** Workspace Gmail SA | One SA per domain; impersonation adapter; OAuth stub `ErrNotConfigured` | `errors.Is` | Stub returns not-configured; SA unique(workspace, domain) |
| **M2.5** Mailboxes UI | List + connect form (shadcn) | — | Test button hits api |

**Out:** Actual campaign send. Gmail API can be a thin adapter tested with a fake.

---

## M3 — Safe send

**Goal:** One approved draft can become one email, never two, never over cap.

| Slice | Build | Go to notice | Done when |
|---|---|---|---|
| **M3.1** River | `cmd/worker`, unique jobs, payload `workspace_id`, GUC set/reset | River kinds, idempotent handlers | Job in same TX as the write |
| **M3.2** Ledger SM | `held → reserved → sending → sent\|unknown\|failed`; 5 min hold | explicit state, no stringly enums | Hold counts toward budget; cancel frees slot |
| **M3.3** Lock protocol | Mailbox then domain `FOR UPDATE`; recompute after lock; strategies registry | `FOR UPDATE`, deadlock avoidance | Parallel RR test never double-picks; no deadlock |
| **M3.4** MIME + one-off send | Build Message-ID first; Mailpit send; `AppendSent` failure still `sent` | `net/mail`, MIME | Mailpit has one message; Message-ID stable |
| **M3.5** Unknown reconciler | 15 min `FindSent`; one same-ID retry | time + row, not pub/sub | FindSent true → `sent`, no second delivery |
| **M3.6** Suppression + kill switch | Hashed suppression; reservation checks pause | — | Kill switch: no new reserve; in-flight sending may finish |
| **M3.7** Review → hold UI | Approve/undo on a one-off draft (campaigns still thin) | — | Undo within hold does not send |

**Blocking tests:** LLD §12 lock, budget, AppendSent, unknown, warmup, RLS.

---

## M4 — Read path

**Goal:** Every reply is seen. Nothing auto-sends.

| Slice | Build | Go to notice | Done when |
|---|---|---|---|
| **M4.1** Sync | IMAP UID cursor / Gmail `history.list`; threads table | supervisor loop, backoff | Mailpit reply appears as a thread |
| **M4.2** Rules + policy | DSN, OOO, unsub; classify job; **policy in code** | lookup table, no model send | `unsubscribe` never enqueues send |
| **M4.3** Reply draft | `reply.draft` + review only; `reply_mode` `draft`\|`off` | — | Reply sits in review |
| **M4.4** Inbox UI | `/inbox` | — | Open thread, mark read |

---

## M5 — Write path (evidence + compose)

**Goal:** No unmapped claim leaves the building.

| Slice | Build | Go to notice | Done when |
|---|---|---|---|
| **M5.1** AI provider | `Complete` / `CompleteJSON` / `Embed`; BYOK; Anthropic ≠ embed | small interface at use site | Fake provider in tests |
| **M5.2** Evidence | Propose → confirm; unconfirmed cannot be cited; pgvector; include-all if ≤ ~30 | sqlc + vector | Gate test: number not in evidence fails |
| **M5.3** Compose + gates | Body + `facts_used`; leftover `{{`, length, links | pure gate functions | One regen then show failures |
| **M5.4** Profiles / JD | Resume upload, optional JD, fit | object storage interface | Enrolment can attach a job later |
| **M5.5** Evidence + Review UI | `/evidence`, `/review` | — | Confirm a claim; approve a draft |

---

## M6 — Campaigns and tracker

**Goal:** Enrol people, run a sequence, see outcomes. Sticky mailbox on follow-up.

| Slice | Build | Go to notice | Done when |
|---|---|---|---|
| **M6.1** Contacts | CRUD, CSV preview, provenance required, dedup | — | Empty provenance rejected |
| **M6.2** Campaigns | Packs as JSON; steps; `campaign_senders`; enrol (contact+context); start/pause/stop | pack data not code | Commercial start requires sender identity |
| **M6.3** Scheduler | Due steps, recipient windows, cap new/day, sticky follow-up | unique `(enrolment, step)` | Follow-up waits if sticky mailbox down |
| **M6.4** Tracker + settings | `/tracker`, `/contacts`, `/campaigns`; sender identity; erasure job; RFC 8058 on commercial | — | One enrolment row moves Drafted → Sent → Replied |

---

## M7 — Writing loop (bind, lists, templates, From)

**Goal:** A real model writes or a saved template fills; a CSV with odd headers becomes contacts; Start has a mailbox. HLD §13.1–13.4.

| Slice | Build | Go to notice | Done when |
|---|---|---|---|
| **M7.1** Compose bind | `writer()` does not fall back to `ai.Fake` unless `kind=fake`. No key / no binding → HTTP error, not stub copy. api/worker keep Fake **only** as test/default when settings say fake. Log compose kind at call site | `errors.Is`, binder already on postgres | Compose with a saved OpenAI/Gemini key returns that model’s text; empty settings return a clear error |
| **M7.2** Fields + CSV map | dbmate `field_defs` + `companies.custom` / `memberships.attrs` / `workspaces.attrs`. Preview headers + samples. Import `columns` map to builtin or def key. Email mapping required if header is not an alias | `encoding/csv` already; map lookup | `Work Email` imports when mapped; unmapped email refuses import; member attrs do not leak across workspaces |
| **M7.3** Templates | dbmate `templates`; step `content=template` + `template_id`; render with `\|` fallbacks; unknown vars rejected on save; leftover `{{` still gated | `strings.Replacer` / small parser, table tests | Save `{{nope}}` fails; draft for Ada uses first name |
| **M7.4** Start needs From | `Start` 400 if no `campaign_senders` and pool has 0 members. Mailbox group UI already exists — use it | — | Empty group cannot start; adding the Gmail mailbox unblocks |

**Out:** PDF, JD fetch, crawler.

---

## M8 — File on the mail

**Goal:** Job sends attach a CV whose contact email is the sending mailbox. Tailor is a second slice on the same JSON. HLD §13.5–13.6, §4.4.

| Slice | Build | Go to notice | Done when |
|---|---|---|---|
| **M8.1** MIME mixed | `mime.Message` files; outer mixed + inner alternative. Unit test bytes contain `Content-Disposition: attachment` | `mime/multipart` nested writers | Mailpit (or fixture) shows one attachment; no attachment ⇒ same as today |
| **M8.2** Structured CV + stamp | Profile CV JSON (not PDF rewrite). After ledger has a mailbox, render PDF with that `From`, store `file_key`, attach on `job` (and campaign opt-in). Review shows filename + From. Render fail → no send | `storage` interface; render behind a small `cv.Render` | Two mailboxes ⇒ two PDFs, same experience, different email on the file |
| **M8.3** Tailor per JD | Propose JSON patch from confirmed evidence + JD; same claim gates on PDF text; operator reviews PDF; one regen | Completer JSON already | Invented employer fails the draft; dropped bullet that was unconfirmed is OK |

**Out:** Catalogue crawl, auto-apply, editing arbitrary uploaded PDFs in place.

---

## M9 — Original resume on the mail

**Goal:** The PDF the operator uploaded is the file on the mail. Disk behind `storage.Store`. HLD §15.1, LLD §15.1.

| Slice | Build | Go to notice | Done when |
|---|---|---|---|
| **M9.1** Disk store | `storage.Disk` (`STORAGE_PATH`); api + worker same dir; Memory remains tests | `os.Open` / `MkdirAll`; interface already `Put`/`Get` | Put in one process, Get in another against a temp dir |
| **M9.2** Upload bytes | Profile upload stores **file bytes** at `file_key`; `text_content` is extract | `multipart.Reader` or existing JSON+bytes | Mailpit/Get returns PDF magic, not UTF-8 of the text |
| **M9.3** Attach original | `campaigns.attach`; job default `original`; `drafts.attachment_resume_id`; Review filename; start 400 if no bytes | sqlc narg; MIME `Files` already | Job send attachment filename = upload; missing blob does not Send |

**Out:** Rewrite email inside the PDF; S3 (same interface later).

---

## M10 — Template canvas

**Goal:** Drag-drop HTML email; `{{vars}}` still fail on save. HLD §15.2. Reference: `leads_crm_1/partner-frontend/components/email-builder/`.

| Slice | Build | Go to notice | Done when |
|---|---|---|---|
| **M10.1** `design_json` | Column; PUT validates vars in JSON strings; compile html/text in the web, API stores | `encoding/json` walk | `{{contact.nope}}` in a button URL → 400 |
| **M10.2** Canvas UI | Port CRM canvas onto theme tokens; saved emails page | — | Drag heading + `{{contact.first_name}}`, save, reopen |
| **M10.3** Draft HTML | Template step renders html+text into the draft | existing template render | Review shows HTML; leftover `{{` still gated |

**Out:** Media library, Unlayer, Word/DOCX import.

---

## M11 — Open and click

**Goal:** Tracker shows Opened / Clicked. Pixel + link wrap. HLD §15.3.

| Slice | Build | Go to notice | Done when |
|---|---|---|---|
| **M11.1** Events | `mail_events`; `opened_at` / `clicked_at`; HMAC token | same token idea as `/u/` | RLS: other workspace GUC → 0 rows |
| **M11.2** Inject + public `/t/o` `/t/c` | GIF; 302; skip `/u/`; `PUBLIC_TRACK_HOST` required if flags on | `html` rewrite tests, no regex-on-the-whole-MIME | Second open keeps first `opened_at`; unsub href unchanged |

**Out:** Attachment download tracking; heatmaps.

---

## After this cut (not scheduled)

JD URL / catalogue as a **source**, Workspace SA send, erasure-vs-evidence, public OAuth verification, DNS monitor, angles, `quality.review`, KB/RRF, S3.

---

## Next

**M9.1** — Disk `storage.Store` so original PDFs survive api vs worker.
