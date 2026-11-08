-- migrate:up

create table workspace_ai (
  workspace_id uuid primary key references workspaces (id),
  providers jsonb not null default '[]',
  models jsonb not null default '{}',
  fallback jsonb,
  updated_at timestamptz not null default now()
);

alter table workspace_ai enable row level security;
alter table workspace_ai force row level security;
create policy workspace_ai_isolation on workspace_ai
  using (workspace_id = nullif(current_setting('app.workspace_id', true), '')::uuid);

grant select, insert, update, delete on workspace_ai to outreach_app;

-- migrate:down

revoke select, insert, update, delete on workspace_ai from outreach_app;
drop table if exists workspace_ai;
