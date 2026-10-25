-- migrate:up

create extension if not exists citext;

do $$
begin
  if not exists (select 1 from pg_roles where rolname = 'outreach_app') then
    create role outreach_app login password 'outreach_app'
      nosuperuser inherit nocreatedb nocreaterole nobypassrls;
  end if;
end
$$;

grant connect on database outreach to outreach_app;
grant usage on schema public to outreach_app;

create table workspaces (
  id uuid primary key,
  name text not null,
  sends_paused boolean not null default false,
  created_at timestamptz not null default now()
);

create table users (
  id uuid primary key,
  email citext unique not null,
  name text,
  password_hash text,
  google_sub text unique,
  created_at timestamptz not null default now()
);

create table memberships (
  workspace_id uuid not null references workspaces (id),
  user_id uuid not null references users (id),
  role text not null check (role in ('owner', 'member', 'viewer')),
  primary key (workspace_id, user_id)
);

create table sessions (
  id text primary key,
  user_id uuid not null references users (id),
  workspace_id uuid not null references workspaces (id),
  expires_at timestamptz not null,
  created_at timestamptz not null default now()
);

-- Tenant tables: even the table owner is subject to RLS.
-- users is instance-global (login by email) and has no workspace_id.
alter table workspaces enable row level security;
alter table workspaces force row level security;
create policy workspaces_isolation on workspaces
  using (id = nullif(current_setting('app.workspace_id', true), '')::uuid);

alter table memberships enable row level security;
alter table memberships force row level security;
create policy memberships_isolation on memberships
  using (workspace_id = nullif(current_setting('app.workspace_id', true), '')::uuid);

alter table sessions enable row level security;
alter table sessions force row level security;
create policy sessions_isolation on sessions
  using (workspace_id = nullif(current_setting('app.workspace_id', true), '')::uuid);

grant select, insert, update, delete on
  workspaces, users, memberships, sessions
  to outreach_app;

-- migrate:down

revoke select, insert, update, delete on
  workspaces, users, memberships, sessions
  from outreach_app;

drop table if exists sessions;
drop table if exists memberships;
drop table if exists users;
drop table if exists workspaces;

revoke usage on schema public from outreach_app;
revoke connect on database outreach from outreach_app;

drop extension if exists citext;
-- outreach_app remains: cluster roles are not owned by this database.
