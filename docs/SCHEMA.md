# Schema (v0.6 additive)

Authority: HLD.md v0.6, LLD.md v0.6 §15. **Do not apply this file.** New tables go through dbmate in `backend/migrations/` with `-- migrate:up` / `-- migrate:down`, FORCE RLS, grant `outreach_app`.

Applied Now (already in git migrations): tenancy, secrets, mailboxes, pools, warmup, contacts (`custom jsonb` on contacts only), companies (no `custom` yet), profiles, resumes, jobs, fits, drafts (`mailbox_id NOT NULL`, no attachment), campaigns + `campaign_steps.template_id` **without** `templates`, ledger, sync, etc.

M7–M8 DDL in this file is **already applied** (keep as history). **M9–M11 below is what still has to migrate.**

Conventions: `PRIMARY KEY (workspace_id, id)` unless noted. Child FKs `(workspace_id, parent_id)`. Policy:

```sql
using (workspace_id = nullif(current_setting('app.workspace_id', true), '')::uuid)
```

`memberships` already has `memberships_by_user` plus workspace isolation. Adding `attrs` does not change that.

---

## M7.2 — Field catalog and JSON bags

```sql
create table field_defs (
  workspace_id uuid not null references workspaces (id),
  id uuid not null,
  object text not null check (object in ('contact', 'company', 'member', 'workspace')),
  key text not null check (key ~ '^[a-z][a-z0-9_]{0,39}$'),
  label text not null,
  kind text not null check (kind in ('text', 'number', 'url', 'date', 'boolean', 'select')),
  required boolean not null default false,
  options jsonb not null default '[]',
  position int not null default 0,
  primary key (workspace_id, id),
  unique (workspace_id, object, key)
);

alter table companies add column custom jsonb not null default '{}';
alter table memberships add column attrs jsonb not null default '{}';
alter table workspaces add column attrs jsonb not null default '{}';
-- contacts.custom already exists
```

App enforces: ≤50 defs per `(workspace_id, object)`; JSON keys ⊆ defs; builtins are columns, never JSON keys (LLD §14.2).

---

## M7.3 — Templates

```sql
create table templates (
  workspace_id uuid not null references workspaces (id),
  id uuid not null,
  name text not null,
  subject text not null,
  body_text text not null,
  body_html text,
  created_at timestamptz not null default now(),
  primary key (workspace_id, id)
);

alter table campaign_steps
  add foreign key (workspace_id, template_id)
  references templates (workspace_id, id);
```

`campaign_steps.template_id` is already nullable. FK is composite so a step cannot point at another workspace’s template.

---

## M8.2 — Structured CV and renders

```sql
create table cv_documents (
  workspace_id uuid not null references workspaces (id),
  id uuid not null,
  profile_id uuid not null,
  version int not null,
  spec jsonb not null,
  created_at timestamptz not null default now(),
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
  created_at timestamptz not null default now(),
  primary key (workspace_id, id),
  foreign key (workspace_id, cv_id) references cv_documents (workspace_id, id),
  foreign key (workspace_id, mailbox_id) references mailboxes (workspace_id, id)
);

create unique index cv_renders_untailored
  on cv_renders (workspace_id, cv_id, mailbox_id) where job_id is null;
create unique index cv_renders_tailored
  on cv_renders (workspace_id, cv_id, mailbox_id, job_id) where job_id is not null;

alter table drafts add column attachment_render_id uuid;
alter table drafts
  add foreign key (workspace_id, attachment_render_id)
  references cv_renders (workspace_id, id);
```

Optional later: `cv_renders.job_id` → `jobs`. Not required to send an untailored stamp (`job_id` null).

Check in app (not a cheap SQL CHECK across tables): `from_email` equals `mailboxes.email` for `mailbox_id`.

`resumes` unchanged (upload → evidence). `spec` has no email field.

---

## M9.3 — Original file on the draft

```sql
alter table campaigns
  add column attach text not null default 'none'
  check (attach in ('none', 'original', 'render'));

alter table drafts add column attachment_resume_id uuid;
alter table drafts
  add foreign key (workspace_id, attachment_resume_id)
  references resumes (workspace_id, id);
```

Job pack **insert** sets `attach='original'` in the app (existing rows stay `none` until patched).

---

## M10.1 — Canvas JSON

```sql
alter table templates
  add column design_json jsonb not null default '{}';
```

`body_html` / `body_text` / `subject` already exist. Empty `{}` = text-only template.

---

## M11.1 — Open / click events

`campaigns.track_opens` / `track_clicks` already exist.

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

create index mail_events_enrolment
  on mail_events (workspace_id, enrolment_id, kind, at);

alter table enrolments add column opened_at timestamptz;
alter table enrolments add column clicked_at timestamptz;
```

FORCE RLS + grant `outreach_app` as every other workspace table. Public `/t/*` sets `app.workspace_id` from the token, same idea as `/u/{token}`.

---

## No DDL

M7.1 compose bind, M7.4 start-needs-From, M8.1 MIME, M8.3 tailor (insert render with `job_id`).

---

## Down (intent)

M11.1: drop enrolment columns, drop `mail_events`.  
M10.1: drop `templates.design_json`.  
M9.3: drop `drafts.attachment_resume_id`, drop `campaigns.attach`.  
M8.2: drop FK on drafts, drop column, drop `cv_renders` then `cv_documents`.  
M7.3: drop FK on `campaign_steps`, drop `templates`.  
M7.2: drop `companies.custom`, `memberships.attrs`, `workspaces.attrs`, drop `field_defs`.
