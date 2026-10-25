# Outreach: Low-Level Design (LLD)

Version 0.4 · 25 Oct 2026 · Builds on PRD.md and HLD.md v0.4

Specifies **Now** plus extension columns/interfaces. Does not specify Next/Later tables.

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
| `mailbox` | CRUD, test, `Transport` |
| `mailbox/smtpimap` | SMTP + IMAP |
| `mailbox/gmailapi` | JWT + `Subject`; one SA per domain |
| `mailbox/oauth` | stub, `ErrNotConfigured` |
| `domain` | `sending_domains`, DNS check helper (manual/on-connect; no 6h monitor job) |
| `pacing` | ledger, lock protocol, strategies, warmup `ramp` |
| `mime` | build message |
| `contacts` | contacts, companies, CSV, provenance |
| `suppression` | checks + hashed erasure leftover |
| `jobsjd` | profiles, resumes, optional JDs, analyze, fit |
| `evidence` | items, propose, confirm, embed, `retrieve` |
| `compose` | templates, drafts, gates, review |
| `campaign` | campaigns, steps, enrolments, `campaign_senders` |
| `sender` | hold, reserve, send, errors |
| `sync` | IMAP supervisor, Gmail poll |
| `inbound` | thread, DSN, policy table |
| `agent` | task runners |
| `ai` | `Provider` (`Complete`, `CompleteJSON`, `Embed`) |
| `health` | samples, pause, warmup day |
| `tracking` | unsubscribe Now; pixel Later |
| `ssrf` | JD fetch |
| `events` | SSE from Postgres NOTIFY on a small set of tables (optional); not a generic bus |
| `audit` | log |
| `httpapi` | oapi-codegen |
| `worker` | River |

## 3. Schema (Now)

Abbreviated; every workspace table includes `workspace_id` and composite FKs as in §1.

```sql
create table workspaces (
  id uuid primary key, name text not null,
  sends_paused boolean not null default false,  -- kill switch
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
| oauth | stub | stub |

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

Auth, workspaces/members, mailboxes (test, reconnect, pause), google-service-accounts, pools, `sending-domains`, warmup-profiles, contacts/import, suppression, profiles/resumes, evidence (propose, confirm, list), jobs, campaigns (steps, senders, enrol, start/pause/stop), drafts (patch, approve, reject, regenerate), threads/reply/read, applications, settings/ai, settings/sender-identity, settings/kill-switch, `/events/stream`, public `POST/GET /u/{token}`.

Errors `{error:{code,message,details}}`. Cursor pagination.

Reject: `reply_mode` other than `draft`|`off`; empty provenance on contact; campaign start without sender identity when pack is commercial.

## 10. Web routes (Now)

`/inbox`, `/tracker`, `/contacts`, `/campaigns`, `/review`, `/evidence`, `/jobs`, `/mailboxes`, `/profiles`, `/settings`. Tomorrow’s queue can be a filter on `/review` + ledger `held`.

## 11. Config (env)

`DATABASE_URL`, `MASTER_KEYS`, `MASTER_KEY_ACTIVE`, `TOKEN_KEY`, `SESSION_KEY`, `PUBLIC_APP_URL`, `PUBLIC_TRACK_HOST`, `STORAGE_DRIVER`, `S3_*`, optional platform `OPENAI_API_KEY` / `GEMINI_API_KEY` (workspaces BYOK), `GOOGLE_OAUTH_CLIENT_ID/SECRET` (login only), `EMBED_DIM` (must match migration), `WORKER_CONCURRENCY_SEND`, `WORKER_CONCURRENCY_AI`, `UNKNOWN_TIMEOUT_S` default 900, `UNDO_HOLD_S` default 300.

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

Matches PRD §9. First mergeable slice: repo (pnpm + Go module + docker-compose postgres), dbmate migration 1 (workspaces/users/sessions), auth, empty web shell.
