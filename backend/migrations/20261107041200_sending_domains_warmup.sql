-- migrate:up

create table sending_domains (
  workspace_id uuid not null references workspaces (id),
  id uuid not null,
  domain citext not null,
  daily_limit int not null,
  first_seen_at timestamptz not null default now(),
  primary key (workspace_id, id),
  unique (workspace_id, domain)
);

create table warmup_profiles (
  workspace_id uuid not null references workspaces (id),
  id uuid not null,
  name text not null,
  kind text not null check (kind in ('standard', 'conservative', 'fast', 'off', 'custom')),
  spec jsonb not null,
  version int not null default 1,
  primary key (workspace_id, id)
);

alter table sending_domains enable row level security;
alter table sending_domains force row level security;
create policy sending_domains_isolation on sending_domains
  using (workspace_id = nullif(current_setting('app.workspace_id', true), '')::uuid);

alter table warmup_profiles enable row level security;
alter table warmup_profiles force row level security;
create policy warmup_profiles_isolation on warmup_profiles
  using (workspace_id = nullif(current_setting('app.workspace_id', true), '')::uuid);

grant select, insert, update, delete on sending_domains, warmup_profiles to outreach_app;

-- migrate:down

revoke select, insert, update, delete on sending_domains, warmup_profiles from outreach_app;
drop table if exists warmup_profiles;
drop table if exists sending_domains;
