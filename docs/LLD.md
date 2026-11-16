# Outreach: Low-Level Design (LLD)

Version 0.6 · 12 Nov 2026 · Builds on PRD.md v0.6 and HLD.md v0.6

**Frozen for M9–M11.** Additive SQL: `docs/SCHEMA.md`. §14 (v0.5) stays closed. §15 is this cut.

Specifies **Now** plus the operator-loop tables (fields, templates, structured CV). Does not specify catalogue crawl or ATS apply.

## 1. Conventions

- Go 1.23+, module `github.com/<you>/outreach/backend`. Errors wrapped with `%w`. `context.Context` first.
- IDs: UUIDv7. API JSON snake_case.
- Times: `timestamptz` UTC. Mailbox windows use an IANA zone. Model names and timezone **defaults** live in config, not in `DEFAULT` clauses that encode product policy (except harmless operational defaults such as `now()`).
- Cost: micro-USD `bigint`.
- **dbmate:** `backend/migrations/YYYYMMDDHHMMSS_name.sql` with `-- migrate:up` / `-- migrate:down`, **in git**. `DATABASE_URL` Postgres. `make migrate` → `dbmate up`.
- **Composite keys:** every workspace-scoped table `PRIMARY KEY (workspace_id, id)` (or natural key that **starts with** `workspace_id`). Child FKs are `(workspace_id, parent_id) REFERENCES parent (workspace_id, id)`.
- Queries take `workspace_id` as the first argument. RLS: `USING (workspace_id = current_setting('app.workspace_id', true)::uuid)` and **`FORCE ROW LEVEL SECURITY`**. DB roles: `outreach` (owner, migrations only), `outreach_app` (api + worker, no BYPASSRLS, no table ownership).
- River job payload always includes `workspace_id`. Worker: `SET app.workspace_id` on the conn at job start; `RESET` (or set empty) when the job ends, before the pool reuse.

## 2. Packages (`backend/internal`)

| Package | Now |
|---|---|
| `config` | envconfig |
| `db` | pgx, sqlc, `InTx`, set workspace GUC |
| `crypto` | envelope `Seal`/`Open`, HMAC tokens |
| `auth` | sessions, argon2id, Google OIDC identity |
| `workspace` | workspaces, members, kill switch, sender identity |
| `mailbox` | CRUD, test, `Operations` (send + fetch), `Registry` |
| `mailbox/smtpimap` | SMTP + IMAP |
| `mailbox/gmailapi` | JWT + `Subject`; one SA per domain |
| `mailbox/oauth` | Google start/callback, HMAC state |
| `fields` | workspace field defs; resolve builtins + custom JSON |
| `domain` | `sending_domains`, DNS check helper (manual/on-connect; no 6h monitor job) |
| `pacing` | ledger, lock protocol, strategies, warmup `ramp` |
| `mime` | build message |
| `contacts` | contacts, companies, CSV, provenance |
| `suppression` | checks + hashed erasure leftover |
| `jobsjd` | profiles, resumes, optional JDs, analyze, fit |
| `cv` | structured spec → PDF render (M8) |
| `evidence` | items, propose, confirm, embed, `retrieve` |
| `compose` | templates, drafts, gates, review |
| `campaign` | campaigns, steps, enrolments, `campaign_senders` |
| `sender` | hold, reserve, send, errors |
| `sync` | IMAP supervisor, Gmail poll |
| `inbound` | thread, DSN, policy table |
| `agent` | task runners |
| `ai` | `Provider` (`Complete`, `CompleteJSON`, `Embed`) |
| `health` | samples, pause, warmup day |
| `tracking` | unsubscribe Now; pixel + click wrap M11 |
| `ssrf` | JD fetch |
| `events` | SSE from Postgres NOTIFY on a small set of tables (optional); not a generic bus |
| `audit` | log |
| `httpapi` | oapi-codegen |
| `worker` | River |

## 3. Schema (Now)

Abbreviated **applied** Now plus tables that LLD always listed (`templates` was specified before it was migrated). **v0.5 additive DDL to run via dbmate is only in `docs/SCHEMA.md`.** Every workspace table includes `workspace_id` and composite FKs as in §1.

```sql
create table workspaces (
  id uuid primary key, name text not null,
  sends_paused boolean not null default false,  -- kill switch
  attrs jsonb not null default '{}', -- M7.2 SCHEMA.md
  created_at timestamptz not null default now()
);
create table users (
  id uuid primary key, email citext unique not null, name text,
  password_hash text, google_sub text unique, created_at timestamptz not null default now()
);
create table memberships (
  workspace_id uuid references workspaces (id),
  user_id uuid references users (id),
  role text not null check (role in ('owner','member','viewer')),
  attrs jsonb not null default '{}', -- M7.2 SCHEMA.md; not on users
  primary key (workspace_id, user_id)
);
create table sessions (
  id text primary key, user_id uuid not null references users (id),
  workspace_id uuid not null references workspaces (id),
  expires_at timestamptz not null, created_at timestamptz not null default now()
);

create table secrets (
  workspace_id uuid not null, id uuid not null,
  kind text not null, -- app_password | service_account_key | llm_key | oauth_token
  key_version int not null, wrapped_dek bytea not null, nonce bytea not null, ciphertext bytea not null,
  hint text, created_at timestamptz not null default now(),
  primary key (workspace_id, id)
);

create table google_service_accounts (
  workspace_id uuid not null, id uuid not null,
  domain citext not null, client_email text not null, client_id text not null,
  secret_id uuid not null, scopes text[] not null, created_at timestamptz not null default now(),
  primary key (workspace_id, id), unique (workspace_id, domain),
  foreign key (workspace_id, secret_id) references secrets (workspace_id, id)
);

create table workspace_ai (
  workspace_id uuid primary key references workspaces (id),
  providers jsonb not null default '[]', -- [{id, kind, secret_id, base_url, hint}]
  models jsonb not null default '{}',    -- role → "kind:model"; empty means config defaults
  fallback jsonb,
  updated_at timestamptz not null default now()
);

create table sending_domains (
  workspace_id uuid not null, id uuid not null,
  domain citext not null, daily_limit int not null,
  first_seen_at timestamptz not null default now(),
  primary key (workspace_id, id), unique (workspace_id, domain)
);

create table warmup_profiles (
  workspace_id uuid not null, id uuid not null,
  name text not null,
  kind text not null check (kind in ('standard','conservative','fast','off','custom')),
  spec jsonb not null, -- start, ramp table, max_growth, idle_counts, bounce_gate, min_sample
  version int not null default 1,
  primary key (workspace_id, id)
);
-- Seed four presets per workspace on workspace create (copy from bundled JSON).

create table mailboxes (
  workspace_id uuid not null, id uuid not null,
  email citext not null, display_name text, signature_html text,
  auth_kind text not null check (auth_kind in ('app_password','service_account','oauth','smtp_imap')),
  provider text not null, -- gmail | workspace | other
  smtp_host text, smtp_port int, smtp_tls text, imap_host text, imap_port int, username text,
  secret_id uuid, service_account_id uuid, sending_domain_id uuid not null,
  timezone text not null, -- required at connect; no silent product default in schema
  window_days int[] not null, window_start time not null, window_end time not null,
  daily_limit int not null, hourly_limit int not null, min_gap_s int not null, max_gap_s int not null,
  hard_daily_ceiling int not null,
  warmup_profile_id uuid not null, warmup_sending_day int not null default 0,
  effective_daily_limit int not null, -- init from profile start cap, typically 2
  reply_mode text not null default 'draft' check (reply_mode in ('draft','off')), -- 'auto' Next
  status text not null check (status in ('healthy','warming','throttled','needs_reconnect','paused')),
  status_reason text, throttled_until timestamptz,
  last_sent_at timestamptz, next_free_at timestamptz,
  created_at timestamptz not null default now(),
  primary key (workspace_id, id), unique (workspace_id, email),
  foreign key (workspace_id, secret_id) references secrets (workspace_id, id),
  foreign key (workspace_id, service_account_id) references google_service_accounts (workspace_id, id),
  foreign key (workspace_id, sending_domain_id) references sending_domains (workspace_id, id),
  foreign key (workspace_id, warmup_profile_id) references warmup_profiles (workspace_id, id)
);

create table mailbox_pools (
  workspace_id uuid not null, id uuid not null, name text not null,
  primary key (workspace_id, id)
);
create table mailbox_pool_members (
  workspace_id uuid not null, pool_id uuid not null, mailbox_id uuid not null,
  primary key (workspace_id, pool_id, mailbox_id),
  foreign key (workspace_id, pool_id) references mailbox_pools (workspace_id, id),
  foreign key (workspace_id, mailbox_id) references mailboxes (workspace_id, id)
);

create table sender_identities (
  workspace_id uuid primary key references workspaces (id),
  legal_name text not null, reply_address citext not null, postal_address text not null
);

create table companies (
  workspace_id uuid not null, id uuid not null,
  domain citext, name text not null,
  last_contacted_at timestamptz, cooldown_days int not null default 30,
  cooldown_override_reason text,
  custom jsonb not null default '{}', -- M7.2; not in applied contacts migration yet
  primary key (workspace_id, id), unique (workspace_id, domain)
);
create table contacts (
  workspace_id uuid not null, id uuid not null,
  email citext not null, first_name text, last_name text, title text,
  company_id uuid, linkedin_url text, timezone text, region text, timezone_confidence real,
  role_class text, -- hiring_manager | recruiter | teammate | exec | generic_mailbox | unknown
  provenance jsonb not null, -- {source, added_by, at} required
  readiness text not null default 'ready' check (readiness in ('ready','risky','blocked')),
  custom jsonb not null default '{}', tags text[] not null default '{}',
  sticky_mailbox_id uuid,
  created_at timestamptz not null default now(),
  primary key (workspace_id, id), unique (workspace_id, email),
  foreign key (workspace_id, company_id) references companies (workspace_id, id),
  foreign key (workspace_id, sticky_mailbox_id) references mailboxes (workspace_id, id)
);
create table suppression (
  workspace_id uuid not null,
  kind text not null check (kind in ('email','domain')),
  value_hash bytea not null, -- hmac of normalized email/domain; plaintext omitted after erasure
  value_citext citext,       -- null after erasure
  reason text not null, created_at timestamptz not null default now(),
  primary key (workspace_id, kind, value_hash)
);

create table profiles (
  workspace_id uuid not null, id uuid not null,
  name text not null, about text, links jsonb not null default '{}',
  primary key (workspace_id, id)
);
create table resumes (
  workspace_id uuid not null, id uuid not null,
  profile_id uuid not null, version int not null,
  file_key text not null, file_name text not null, mime text not null, text_content text not null,
  created_at timestamptz not null default now(),
  primary key (workspace_id, id),
  foreign key (workspace_id, profile_id) references profiles (workspace_id, id)
);
create table evidence_items (
  workspace_id uuid not null, id uuid not null,
  profile_id uuid, text text not null, source_ref text, tags text[] not null default '{}',
  do_not_use boolean not null default false, confirmed_at timestamptz, -- null = unconfirmed
  embedding vector, embedding_model text, embedded_at timestamptz,
  created_at timestamptz not null default now(),
  primary key (workspace_id, id),
  foreign key (workspace_id, profile_id) references profiles (workspace_id, id)
);
-- vector dimension: set in the first migration from config (e.g. vector(1536)). Not mixed.

create table jobs (
  workspace_id uuid not null, id uuid not null,
  source_url text, raw_text text not null, company_id uuid, title text, analysis jsonb,
  analysis_status text not null default 'pending',
  created_at timestamptz not null default now(),
  primary key (workspace_id, id)
);
create table fits (
  workspace_id uuid not null, id uuid not null,
  job_id uuid not null, profile_id uuid not null, resume_id uuid, score int, result jsonb,
  created_at timestamptz not null default now(),
  primary key (workspace_id, id), unique (workspace_id, job_id, profile_id)
);

create table templates (
  workspace_id uuid not null, id uuid not null,
  name text not null, subject text not null, body_html text not null, body_text text,
  variables text[] not null default '{}',
  primary key (workspace_id, id)
);

create table campaigns (
  workspace_id uuid not null, id uuid not null,
  name text not null, pool_id uuid, profile_id uuid,
  intent text not null check (intent in ('job','referral','cold','freelance','informational','custom')),
  sender_strategy text not null check (sender_strategy in
    ('round_robin','sequential','capacity_first','weighted','domain_spread','manual')),
  status text not null check (status in ('draft','running','paused','stopped','completed')),
  track_opens boolean not null default false, track_clicks boolean not null default false,
  list_unsubscribe boolean not null default false, -- true when intent pack is commercial
  new_contacts_per_day int not null, starts_at timestamptz,
  company_cooldown_days int not null default 30, ai_instructions text,
  undo_hold_s int not null default 300,
  created_at timestamptz not null default now(),
  primary key (workspace_id, id)
);
create table campaign_senders (
  workspace_id uuid not null, campaign_id uuid not null, mailbox_id uuid not null,
  position int not null, weight int not null default 1,
  primary key (workspace_id, campaign_id, mailbox_id),
  foreign key (workspace_id, campaign_id) references campaigns (workspace_id, id),
  foreign key (workspace_id, mailbox_id) references mailboxes (workspace_id, id)
);
create table campaign_steps (
  workspace_id uuid not null, id uuid not null, campaign_id uuid not null, position int not null,
  kind text not null check (kind in ('email','wait')), wait_days int,
  content text check (content in ('ai','template')), template_id uuid, ai_instructions text,
  primary key (workspace_id, id), unique (workspace_id, campaign_id, position)
);

create table enrolments (
  workspace_id uuid not null, id uuid not null,
  campaign_id uuid not null, contact_id uuid not null,
  job_id uuid, profile_id uuid,
  intent text not null check (intent in ('job','referral','cold','freelance','informational','custom')),
  context jsonb not null default '{}',
  mailbox_id uuid, thread_id uuid, current_step int not null default 0,
  status text not null check (status in
    ('pending','active','waiting_review','held','replied','bounced','unsubscribed','completed','stopped','failed')),
  next_action_at timestamptz, hold_until timestamptz, stop_reason text,
  outcome text, -- positive_reply | referral | interview | offer | declined | no_response | bounced | unsubscribed
  created_at timestamptz not null default now(),
  primary key (workspace_id, id), unique (workspace_id, campaign_id, contact_id)
);

create table applications (
  workspace_id uuid not null, id uuid not null,
  enrolment_id uuid not null unique,
  contact_id uuid not null, job_id uuid, profile_id uuid,
  stage text not null check (stage in
    ('drafted','sent','followed_up','replied','interview','offer','rejected','closed')),
  stage_changed_at timestamptz not null default now(),
  primary key (workspace_id, id)
);

create table drafts (
  workspace_id uuid not null, id uuid not null,
  enrolment_id uuid, step_position int,
  kind text not null check (kind in ('outbound','reply')),
  subject text not null, body_html text not null, body_text text not null,
  facts_used jsonb not null default '[]', -- [{claim, evidence_id}]
  retrieval jsonb, -- reserved for Next
  new_recipient_from_inbound boolean not null default false,
  status text not null check (status in ('pending','approved','rejected','sent','superseded','cancelled')),
  approved_by uuid, created_at timestamptz not null default now(),
  primary key (workspace_id, id)
);

create table threads (
  workspace_id uuid not null, id uuid not null,
  mailbox_id uuid not null, contact_id uuid, subject text, gmail_thread_id text,
  last_message_at timestamptz, unread boolean not null default false,
  primary key (workspace_id, id)
);
create table messages (
  workspace_id uuid not null, id uuid not null,
  mailbox_id uuid not null, thread_id uuid,
  direction text not null check (direction in ('out','in')),
  rfc_message_id text not null,
  in_reply_to text, references_ids text[] not null default '{}',
  gmail_id text, imap_folder text, imap_uid bigint,
  from_addr citext, to_addrs citext[], subject text, body_text text, body_html_key text,
  sent_at timestamptz, received_at timestamptz,
  status text, kind text, classification jsonb,
  enrolment_id uuid, draft_id uuid,
  primary key (workspace_id, id), unique (workspace_id, mailbox_id, rfc_message_id)
);
create table attachments (
  workspace_id uuid not null, id uuid not null,
  message_id uuid not null, file_key text not null, file_name text not null, mime text not null, size int not null,
  primary key (workspace_id, id)
);
create table sync_state (
  workspace_id uuid not null, mailbox_id uuid not null, folder text not null,
  uid_validity bigint, last_uid bigint, gmail_history_id bigint,
  last_synced_at timestamptz, last_error text,
  primary key (workspace_id, mailbox_id, folder)
);

create table send_ledger (
  workspace_id uuid not null, id uuid not null,
  mailbox_id uuid not null, sending_domain_id uuid not null,
  enrolment_id uuid, draft_id uuid not null, step_position int,
  is_first_touch boolean not null,
  rfc_message_id text not null,
  state text not null check (state in
    ('held','reserved','sending','sent','unknown','released','cancelled','failed')),
  reserved_at timestamptz, release_at timestamptz, sent_at timestamptz,
  created_at timestamptz not null default now(),
  primary key (workspace_id, id)
);
create unique index send_ledger_enrolment_step
  on send_ledger (workspace_id, enrolment_id, step_position)
  where enrolment_id is not null and step_position is not null
    and state not in ('cancelled','released','failed');
create table send_attempts (
  workspace_id uuid not null, id uuid not null,
  ledger_id uuid not null, attempt int not null,
  outcome text not null, provider_response text,
  created_at timestamptz not null default now(),
  primary key (workspace_id, id)
);

create table health_samples (
  workspace_id uuid not null, id bigserial,
  mailbox_id uuid, sending_domain_id uuid,
  metric text not null, value double precision not null, window text not null,
  created_at timestamptz not null default now(),
  primary key (workspace_id, id)
);

create table ai_runs (
  workspace_id uuid not null, id uuid not null,
  task text not null, model text not null, prompt_version text,
  tokens_in int, tokens_out int, cost_micro_usd bigint, latency_ms int, status text,
  created_at timestamptz not null default now(),
  primary key (workspace_id, id)
);
create table audit_log (
  workspace_id uuid not null, id bigserial,
  user_id uuid, action text not null, target text, detail jsonb,
  created_at timestamptz not null default now(),
  primary key (workspace_id, id)
);
```

Intent packs: JSON files in `backend/internal/campaign/packs/*.json`, loaded in process. Not a DB table in Now.

`enrolments.context` must be non-empty notes **or** a `job_id` **or** campaign `ai_instructions`.

Commercial intents (`cold`, `freelance`, or `custom` marked commercial in the pack): starting the campaign requires `sender_identities` and sets `list_unsubscribe = true`.

## 4. Transports

```go
var ErrNotConfigured = errors.New("oauth transport not configured")

type Transport interface {
    Test(ctx context.Context) error
    Send(ctx context.Context, m *mime.Built) (SendResult, error)
    AppendSent(ctx context.Context, m *mime.Built) error
    FindSent(ctx context.Context, messageID string) (bool, error)
}
```

| auth_kind | Send | FindSent / read |
|---|---|---|
| app_password | SMTP | IMAP SEARCH header Message-ID; Gmail files Sent on Send — AppendSent no-op |
| smtp_imap | SMTP | AppendSent after **Send success** if server does not file; FindSent IMAP |
| service_account | Gmail API raw send | Gmail API get/list by rfc822 msgid |
| oauth | Gmail API user | Gmail history / get |

`Send` success always wins over `AppendSent` failure: ledger `sent`, attempt row records append error.

## 5. Pacing and send

### 5.1 Counting query

Rolling usage for mailbox or domain:

```sql
select count(*) from send_ledger
where workspace_id = $ws and mailbox_id = $mb
  and state in ('held','reserved','sending','sent','unknown')
  and coalesce(sent_at, reserved_at, created_at) > now() - interval '24 hours';
```

Hourly: same with `1 hour`. First-touch RR: `is_first_touch` and those states, order by that timestamp desc.

### 5.2 `ramp(profile, sending_day, health) int`

Pure function from `warmup_profiles.spec`. Standard bundled spec matches PRD §6. Returns 0 if status is paused / needs_reconnect. Caller still `min` with limits and domain remainder.

### 5.3 Selection

Interface:

```go
type Strategy interface {
    Order(eligible []MailboxView, ledger LedgerView, senders []CampaignSender) []uuid.UUID
}
```

Registry keyed by `campaigns.sender_strategy`. Protocol: HLD §4.1 (one mailbox lock at a time, recompute after lock).

### 5.4 Sender

On approve: insert `send_ledger` `held`, `release_at = now()+undo_hold_s`, `rfc_message_id` already on draft.

Hold releaser (15s): rows with `state=held` and `release_at<=now` and not `workspaces.sends_paused` → run reservation.

Reservation transaction: kill switch, draft still approved, suppression, readiness, company cooldown, already replied, budgets, gap; set `reserved`, `next_free_at`.

Send job: `state=sending`; `Transport.Send`; on OK `sent` + `AppendSent` best effort; on timeout after write `unknown`; on 5xx/4xx classify (HLD: auth, throttle, bounce, content).

Reconciler (1 min): `unknown` younger than 15 min → `FindSent`; found → `sent`; older than 15 min and no find → one `Send` with same Message-ID; still fail → `failed`.

Stale `reserved`/`sending` &gt; 10 min without a provider result → `released` if no FindSent, else `sent`.

## 6. Sync

IMAP supervisor: advisory lock `hashtext(mailbox_id::text)`; LOGIN; SELECT INBOX and `\Sent`; UIDVALIDITY change → SEARCH SINCE 30 days; UID FETCH batches of 100; IDLE INBOX (reissue 25 min); Sent catch-up every 5 min. Auth error → `needs_reconnect`, stop supervisor.

Gmail: poll 60–120s jitter; historyId 404 → list `newer_than:7d`.

## 7. Inbound policy

Rules first, then classify JSON:

```json
{ "class": "interested|interview|question|not_now|rejection|referral|out_of_office|unsubscribe|complaint|other",
  "confidence": 0.0, "return_date": null, "referral": {"name":"", "email":""}, "summary": "" }
```

| class | enrolment | application.stage | extra |
|---|---|---|---|
| interested / question | replied | replied | `reply.draft` |
| interview | replied | interview | `reply.draft` + notify |
| not_now / rejection | stopped | rejected | — |
| referral | replied | replied | `reply.draft`; `new_recipient_from_inbound` if new email |
| out_of_office | held | — | `hold_until` |
| unsubscribe / complaint | unsubscribed | closed | suppress; complaint also samples health; ≥2 complaints in 24h pause campaign |
| other | replied | replied | `reply.draft` if confidence ≥ 0.6 |

`reply.draft` status `pending`. API rejects `reply_mode=auto`.

## 8. AI

```go
type Role string // compose | analyze | classify | reply | embed

type Provider interface {
    Complete(ctx context.Context, req Request) (TextResult, error)
    CompleteJSON(ctx context.Context, req Request, schema json.RawMessage) (JSONResult, error)
    Embed(ctx context.Context, texts []string) ([][]float32, error) // ErrNotSupported if none
}
```

Prompts in `internal/ai/prompts/*.tmpl` with `prompt_version`. Temperature 0.2 analyze/classify, 0.6 compose/reply. Wrap JD/email as data.

`retrieve(ctx, ws, profileID, queryEmbedding) []Evidence`: if confirmed usable items ≤ 30, all; else `ORDER BY embedding <=> $q LIMIT 20` filtered `workspace_id`, `confirmed_at is not null`, `not do_not_use`, `embedding_model = current`. Confirm/edit enqueues embed. Composer may only cite returned IDs.

Gates: no `{{`; max words; banned phrases file; each `\d+` and proper names in body must occur in cited evidence text or `enrolment.context`. One regenerate then flag.

No send tool. No `quality.review` job in Now.

## 9. HTTP (Now)

Auth, workspaces/members, mailboxes (test, reconnect, pause), google-service-accounts, pools, `sending-domains`, warmup-profiles, contacts/import (preview + `columns` map), `field_defs`, suppression, profiles/resumes, cv documents/renders, evidence (propose, confirm, list), jobs, templates, campaigns (steps, senders, enrol, start/pause/stop; start requires From), drafts (patch, approve, reject, regenerate), threads/reply/read, applications, settings/ai, settings/sender-identity, settings/kill-switch, `/events/stream`, public `POST/GET /u/{token}`.

Errors `{error:{code,message,details}}`. Cursor pagination.

Reject: `reply_mode` other than `draft`|`off`; empty provenance on contact; campaign start without sender identity when pack is commercial; campaign start with no senders and empty pool; compose with no AI key unless `kind=fake`.

## 10. Web routes (Now)

`/inbox`, `/tracker`, `/contacts`, `/campaigns`, `/review`, `/evidence`, `/jobs`, `/mailboxes`, `/profiles`, `/settings`. Tomorrow’s queue can be a filter on `/review` + ledger `held`.

## 11. Config (env)

`DATABASE_URL`, `MASTER_KEYS`, `MASTER_KEY_ACTIVE`, `TOKEN_KEY`, `SESSION_KEY`, `PUBLIC_APP_URL`, `PUBLIC_TRACK_HOST`, `STORAGE_DRIVER`, `S3_*`, optional platform `OPENAI_API_KEY` / `GEMINI_API_KEY` (workspaces BYOK), `GOOGLE_OAUTH_CLIENT_ID/SECRET` / `GOOGLE_OAUTH_REDIRECT_URI` (Gmail mailbox OAuth; names must be tagged `envconfig` so `OAuth` does not split to `O_AUTH`), `EMBED_DIM` (must match migration), `WORKER_CONCURRENCY_SEND`, `WORKER_CONCURRENCY_AI`, `UNKNOWN_TIMEOUT_S` default 900, `UNDO_HOLD_S` default 300.

## 12. Tests (Now, blocking)

- Lock protocol: two parallel first-touches on a two-mailbox RR pool never double-pick the same mailbox while both are eligible; never deadlock.
- Budget: `held` counts; releasing hold without send frees the slot.
- `Send` OK `AppendSent` fail → `sent`, no retry.
- `unknown` + FindSent true → `sent`, no second recipient delivery in Mailpit.
- Warmup: idle day does not increment `warmup_sending_day`; day-1 cap 2.
- Evidence: unconfirmed item cannot appear in `facts_used`; number not in evidence fails gate.
- Policy: classify `unsubscribe` never enqueues a send job.
- RLS: `outreach_app` with wrong `app.workspace_id` reads 0 rows.
- dbmate up/down on testcontainers Postgres.

## 13. Build order

Shipped through M6 + Gmail user OAuth. §14 objects are **closed**. Next: ACTION_PLAN **M7.1**.

## 14. Fields, templates, CV (v0.5)

DDL to migrate: **`docs/SCHEMA.md`**. This section is why those tables look like that.

### 14.0 Current scenario (constraints we are not free to ignore)

Shipped and in production-shaped local use:

- Tenancy is **workspace**, not user. RLS `FORCE` on `app.workspace_id`. `users.email` is **globally** unique; `memberships` is the per-workspace join (`memberships_by_user` exists so login can list workspaces **before** a GUC).
- `contacts.custom jsonb` already exists; CSV import only aliases `email` / `first_name` / `last_name` / `title` / `company`. Provenance required. Dedup `(workspace_id, email)`.
- `campaign_steps.template_id` and `content in ('ai','template')` already exist **without** a `templates` table or FK.
- `drafts.mailbox_id` is **NOT NULL**. One-off compose already picks a From. Campaign first-touch still goes through ledger reserve; do not assume mailbox is unknown at draft insert, but **do not attach a CV until that mailbox is the one that will send**.
- MIME is `multipart/alternative` only (no file parts). Gmail send is **raw RFC2822**.
- Compose binder falls back to `ai.Fake` on missing key. Settings already store kind + sealed key.
- `resumes` is text + `file_key` for evidence propose, not a send artifact.
- HLD scale: tens of workspaces, ~50 mailboxes, hundreds of sends/day — not a warehouse. We optimize for **correct merge and one PDF per send**, not analytics over custom fields.
- Operator still approves every outbound. Replies never auto-send.

Everything below is chosen against that, not against a greenfield CRM.

### 14.1 Objects: why these four tables

| Operator word | Table | Decision | Rejected |
|---|---|---|---|
| Lead / contact | `contacts` + `custom` | Person we email. Custom bag already shipped; keep it. | New `leads` table (split from contacts) — we have one person record and readiness/suppression already on contacts. |
| Account (their company) | `companies` + `custom` (add) | Domain is the natural key we already unique on. | Treating workspace as “account” — that is *our* tenant, not Acme Inc. |
| User (teammate) | **`memberships.attrs`** | Fields are **workspace-scoped** (title, signature, calendar in *this* product). | Columns on `users` — same login in two workspaces would share or leak a signature; also `users` is not RLS-by-workspace (it is the login row). |
| Our tenant | `workspaces.attrs` + existing `sender_identities` | CAN-SPAM name/address stay real columns (required, queried, not optional JSON). | Putting legal_name only in attrs — start-campaign gate would scrape JSON. |

`profiles` stay persona + CV, not a field object. Enrolment `context` stays notes + optional job — not a place to hide custom lead fields (those must survive across campaigns).

**Contact and member fields are necessary in this cut** because CSV/templates/`{{user.signature}}`/`{{contact.*}}` read them. Company and workspace attrs are the same *mechanism* so we do not add a second pattern later; they are not on the critical send path.

**Merge order** (first hit wins): `contact` → `company` → `user` (membership of approver, else enroler) → `workspace` / `sender` → `profile` → `job` → `mailbox`. Builtins always win over custom of the same key.

### 14.2 Catalog + JSON values, not EAV

**Choice:** `field_defs` (schema) + JSON map on the row (data).

| Alternative | Why not now |
|---|---|
| EAV `field_values (object_id, def_id, value)` | We do not `WHERE custom_x =` in the scheduler. Every compose would join 20 rows. RLS on a values table is more surface. Promote later if we filter lists by custom field. |
| Only JSON, no catalog | Templates cannot reject `{{contact.nope}}` on save; CSV mapper has nothing to offer but free text. |
| JSON schema in `workspaces.settings` | No row-level list for UI; harder unique `(object, key)`; same RLS as a table anyway. |
| Postgres tables per workspace / `ALTER TABLE` | Migrations are global dbmate; cannot let tenants add columns. |
| Put every field as a real column (`linkedin_url` already is) | We already promoted the ones send/readiness need. The long tail is tenant-specific. |

```sql
create table field_defs (
  workspace_id uuid not null references workspaces (id),
  id uuid not null,
  object text not null check (object in ('contact','company','member','workspace')),
  key text not null check (key ~ '^[a-z][a-z0-9_]{0,39}$'),
  label text not null,
  kind text not null check (kind in ('text','number','url','date','boolean','select')),
  required boolean not null default false,
  options jsonb not null default '[]',
  position int not null default 0,
  primary key (workspace_id, id),
  unique (workspace_id, object, key)
);
```

Also: `companies.custom jsonb not null default '{}'`, `memberships.attrs jsonb not null default '{}'`, `workspaces.attrs jsonb not null default '{}'`. Contacts already have `custom`.

**Write rule:** keys in JSON ⊆ defs for that object. Unknown key → 400. Builtin keys are **not** stored in JSON (they have columns). Delete def → leave orphan JSON (ignore on read); no table rewrite. Rename = new key. Cap **50 defs / object / workspace** so merge stays bounded.

**Builtins (forbidden as `field_defs.key`):**

- contact: `email`, `first_name`, `last_name`, `title`, `linkedin_url`, `timezone`, `region`, `role_class`, `tags`
- company: `domain`, `name`
- member: `email`, `name`, `role` (from `users` + membership)
- workspace/sender: `name`, `legal_name`, `reply_address`, `postal_address`

**Catalog the operator sees.** `GET /api/v1/fields?object=contact&catalog=1` (same for `member`) returns the built-in columns first, marked `builtin`, then `field_defs`. Built-ins are not inserted. The contact list the operator fills is `email`, `first_name`, `last_name`, `title`, `linkedin_url`, `timezone`, `region`. Reserved keys that stay off that list (`role_class`, `tags`) still cannot be reused as a custom key. Member built-ins on the list are `name`, `email`, `role`.

**Types.** `kind` is already constrained. `select` requires a non-empty `options` JSON array of strings. Other kinds ignore options.

**`{{user.key}}` at draft time.** Enrolment has no author. Merge reads `memberships.attrs` for `role = owner` (one row) into `user.<key>`. Approver-specific values are a later pass, when approve re-renders the draft. Contact custom values stay on `contact.<key>` and in the compose notes. Do not add CRM pipeline fields (status, priority, source) as contact builtins.

One person per email per workspace stays a unique key. The workspace chooses what a repeat does: `prevent` (default) refuses the add, `update` fills empty name, title, company, and contact fields on that person. `overwrite` replaces values that are already filled. A second row with the same email is never inserted. Every contact read and write filters `workspace_id` from the session and runs under the `app.workspace_id` GUC, so another tenant's rows stay invisible. Match is email only. Name and phone are not duplicate keys.

### 14.3 CSV map

**Choice:** client sends `columns: {field_key: csv_header}`. Server does not persist mapping (HLD: session only). Guess aliases + def labels; **never invent `email`**.

Rejected: persist `import_presets` now — one more table before we know if operators reuse the same sheet. Rejected: “first column is email” — silent corruption.

Import still requires provenance. Company create/find by name/domain only in this slice; `company.<custom>` map is a follow-on (same `columns` shape, prefix), not a second importer.

### 14.4 Templates

**Choice:** workspace `templates` table + FK from existing `campaign_steps.template_id`.

Rejected: body on the step only — cannot reuse “job first touch” across campaigns. Rejected: Liquid/Mustache engine — we need a **closed** var set (builtins ∪ defs) so save-time reject works; a full engine invites `{{#each}}` and injection. Small `{{path}}` / `{{path | fallback}}` parser.

```sql
create table templates (
  workspace_id uuid not null references workspaces (id),
  id uuid not null,
  name text not null,
  subject text not null,
  body_text text not null,
  body_html text,
  primary key (workspace_id, id)
);
alter table campaign_steps
  add foreign key (workspace_id, template_id) references templates (workspace_id, id);
```

(`template_id` is already nullable on steps.) Render at draft create. Leftover `{{` still gated.

### 14.5 CV: structured spec + render rows

**Choice:** canonical CV is JSON (`cv_documents.spec`) **without email**. Bytes we attach are `cv_renders` produced **for a mailbox** (and optional `job_id`). Upload (`resumes`) stays evidence input.

| Alternative | Why not |
|---|---|
| Attach the uploaded PDF | Cannot reliably change the contact email; recruiter replies off-thread when From ≠ CV. |
| pdfcpu/qpdf stamp on upload | Layout-fragile; fonts/XObject; two mailboxes still need two files anyway. |
| One PDF column on `profiles` | Round-robin would send the wrong From. |
| Render at compose always | Fine when `drafts.mailbox_id` is already the sender; **invalid if reserve later picks another mailbox**. Rule: `from_email` on the render **must equal** `mailboxes.email` of the ledger mailbox at `sending`. If draft mailbox ≠ reserved mailbox, re-render or fail. |
| Store PDF on `drafts` as bytea | Blows up WAL; we already have `storage` + `file_key` on resumes. |

```sql
create table cv_documents (
  workspace_id uuid not null references workspaces (id),
  id uuid not null,
  profile_id uuid not null,
  version int not null,
  spec jsonb not null, -- name, headline, sections; never email
  primary key (workspace_id, id),
  unique (workspace_id, profile_id, version),
  foreign key (workspace_id, profile_id) references profiles (workspace_id, id)
);

create table cv_renders (
  workspace_id uuid not null references workspaces (id),
  id uuid not null,
  cv_id uuid not null,
  mailbox_id uuid not null,
  job_id uuid,
  file_key text not null,
  file_name text not null,
  mime text not null default 'application/pdf',
  size int not null,
  from_email citext not null,
  primary key (workspace_id, id),
  foreign key (workspace_id, cv_id) references cv_documents (workspace_id, id),
  foreign key (workspace_id, mailbox_id) references mailboxes (workspace_id, id)
);
-- Postgres UNIQUE treats NULLs as distinct; two "untailored" rows would pass a
-- unique (cv_id, mailbox_id, job_id). Use partial indexes instead:
create unique index cv_renders_untailored
  on cv_renders (workspace_id, cv_id, mailbox_id) where job_id is null;
create unique index cv_renders_tailored
  on cv_renders (workspace_id, cv_id, mailbox_id, job_id) where job_id is not null;

alter table drafts add column attachment_render_id uuid;
```

Job pack: no render at `sending` → do not `Transport.Send`. Tailor (M8.3) = new render with `job_id`, same `cv_id` (spec unchanged unless operator saves a new version). Check: `from_email = mailboxes.email`.

MIME: `Files []` on `mime.Message`; outer mixed, inner alternative. Fail closed on size.

### 14.6 Compose bind / start (no DDL)

Fake fallback is a product bug in the current binder, not a missing table. Fail if no key. `kind=fake` only when settings say so.

Start: 400 if no `campaign_senders` and pool empty — empty group is already a UI state.

### 14.7 Slice ↔ DDL

| Slice | DDL |
|---|---|
| M7.1 | none |
| M7.2 | `field_defs`; `companies.custom`; `memberships.attrs`; `workspaces.attrs` |
| M7.3 | `templates` + FK `campaign_steps` |
| M7.4 | none |
| M8.1 | none (mime) |
| M8.2 | `cv_documents`, `cv_renders` (+ partial uniques), `drafts.attachment_render_id` |
| M8.3 | none (row with `job_id`) |

### 14.8 Tests (add)

- Builtin key cannot be inserted into `field_defs`; contact JSON with unknown key rejected; membership attrs do not appear when the same user is queried in another workspace.
- CSV: `Work Email` mapped → contact; no email map and no alias → error.
- Template `{{contact.nope}}` rejected; `{{contact.first_name \| there}}` ok.
- Start empty From → 400.
- Two mailboxes → two renders, same spec, different `from_email`; second untailored insert for same `(cv, mailbox)` conflicts.
- Tailor bullet with a number not in confirmed evidence fails; Fake string never used when no key.

## 15. Original file, canvas, tracking (v0.6)

HLD §15. Packages: `storage` (disk), `jobsjd` upload bytes, `cv`/`sender` Files, `templates` design_json, `tracking` (new) inject + public handlers. Operator copy: filename, “Opened”, “Clicked”; never “pixel”, “ledger”, “enrolment.”

### 15.1 M9 original attach

**Choice:** attach `resumes` bytes. Keep `cv_renders` as `attach=render`. Do not pdf-stamp the upload.

| Alternative | Why not |
|---|---|
| Only stamped JSON PDF | Operator asked to send the file they uploaded. |
| Stamp email onto the upload | Layout-fragile (v0.5 already rejected). |
| bytea on drafts | WAL; `file_key` already on resumes. |
| Keep Memory store | Original cannot be rebuilt; api/worker split 404s. |

`campaigns.attach text not null default 'none' check (attach in ('none','original','render'))`. Job pack create sets `original`. Start with `original` and no resume bytes → 400 “Upload a PDF on this profile.”

`drafts.attachment_resume_id` nullable, composite FK to `resumes`. App: at most one of `attachment_resume_id` / `attachment_render_id`.

Disk: `STORAGE_DRIVER=disk` (default Now), `STORAGE_PATH` (both binaries). Keys stay `{workspace_id}/resumes/{id}`. Put **multipart bytes**; extract text into `text_content` as today. Empty text still allowed if PDF extract fails? **No:** evidence path needs text; upload without extractable text is 400 “Could not read this file” unless they paste. Prefer: bytes required for attach; text required for propose.

`Files`: if draft has resume id, Get those bytes, name `file_name`, pdf/docx mime. `ErrTooLarge` → Review fail closed.

### 15.2 M10 canvas

**Choice:** `design_json jsonb not null default '{}'` on existing `templates`. Compile to `body_html` / `body_text` on PUT. Empty `{}` means text-only template (today’s editor still valid).

Walk `design_json` strings for `{{` on save (same parser as M7.3). Images: `file_key` in element data, bytes in disk under `{workspace_id}/templates/{id}/…`; HTML gets `cid:` or absolute `PUBLIC_APP_URL` — **Now: inline cid** if small, else reject. Keep it thin: image blocks store https URL the operator pasted, no media library in M10.1.

Reference implementation: `leads_crm_1/partner-frontend/components/email-builder/` (`design_json` `{settings,elements}`, `generateHTML.js`). Port generateHTML into `apps/web` (TS) **or** Go; **pick web compile, API stores the three columns** so Go does not own dnd types. API still re-validates vars.

### 15.3 M11 tracking

Columns `track_opens` / `track_clicks` already on `campaigns` (default false). New campaigns: default **true** in the app when creating (migration does not rewrite old rows).

```sql
create table mail_events (
  workspace_id uuid not null references workspaces (id),
  id uuid not null,
  enrolment_id uuid not null,
  draft_id uuid not null,
  kind text not null check (kind in ('open', 'click')),
  url text,
  at timestamptz not null default now(),
  primary key (workspace_id, id),
  foreign key (workspace_id, enrolment_id) references enrolments (workspace_id, id),
  foreign key (workspace_id, draft_id) references drafts (workspace_id, id)
);
create index mail_events_enrolment on mail_events (workspace_id, enrolment_id, kind, at);
alter table enrolments add column opened_at timestamptz;
alter table enrolments add column clicked_at timestamptz;
```

Token: HMAC over `workspace_id|draft_id|kind|url`, public `GET /t/o/{token}` and `GET /t/c/{token}` on **api** (no session). Handler uses a **bypass that still sets GUC** from the token’s workspace (same pattern as `/u/{token}`). Invalid token → 204 GIF / 404, no leak.

Inject in `mime` or `sender` after HTML exists. `PUBLIC_TRACK_HOST` empty + flags true → do not send, operator error.

No `download` kind.

### 15.4 Slice ↔ DDL

| Slice | DDL |
|---|---|
| M9.1 | none (disk driver) |
| M9.2 | none if `file_key` already stores bytes |
| M9.3 | `campaigns.attach`; `drafts.attachment_resume_id` |
| M10.1 | `templates.design_json` |
| M10.2 | none (UI) |
| M11.1 | `mail_events`; `enrolments.opened_at/clicked_at` |
| M11.2 | none (inject + public routes) |

### 15.5 Tests

- Disk Put in api process, Get in a second process (or new Memory-incompatible test: temp dir).
- Job start `attach=original` without resume bytes → 400.
- Mailpit shows the uploaded PDF filename, not a stamped `ada-lovelace.pdf`, when `original`.
- Template save with `{{contact.nope}}` in a button href → 400.
- Open pixel: second fetch does not clear `opened_at`; click records url; `/u/` href unchanged.
