-- migrate:up

create table companies (
  workspace_id uuid not null references workspaces (id),
  id uuid not null,
  domain citext,
  name text not null,
  last_contacted_at timestamptz,
  cooldown_days int not null default 30,
  cooldown_override_reason text,
  created_at timestamptz not null default now(),
  primary key (workspace_id, id),
  unique (workspace_id, domain)
);

create table contacts (
  workspace_id uuid not null references workspaces (id),
  id uuid not null,
  email citext not null,
  first_name text,
  last_name text,
  title text,
  company_id uuid,
  linkedin_url text,
  timezone text,
  region text,
  timezone_confidence real,
  role_class text,
  provenance jsonb not null,
  readiness text not null default 'ready' check (readiness in ('ready', 'risky', 'blocked')),
  custom jsonb not null default '{}',
  tags text[] not null default '{}',
  sticky_mailbox_id uuid,
  created_at timestamptz not null default now(),
  primary key (workspace_id, id),
  unique (workspace_id, email),
  foreign key (workspace_id, company_id) references companies (workspace_id, id),
  foreign key (workspace_id, sticky_mailbox_id) references mailboxes (workspace_id, id)
);

alter table companies enable row level security;
alter table companies force row level security;
create policy companies_isolation on companies
  using (workspace_id = nullif(current_setting('app.workspace_id', true), '')::uuid);

alter table contacts enable row level security;
alter table contacts force row level security;
create policy contacts_isolation on contacts
  using (workspace_id = nullif(current_setting('app.workspace_id', true), '')::uuid);

grant select, insert, update, delete on companies, contacts to outreach_app;

-- migrate:down

revoke select, insert, update, delete on companies, contacts from outreach_app;
drop table if exists contacts;
drop table if exists companies;
