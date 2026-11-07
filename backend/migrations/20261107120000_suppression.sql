-- migrate:up

create table suppression (
  workspace_id uuid not null references workspaces (id),
  kind text not null check (kind in ('email', 'domain')),
  value_hash bytea not null,
  value_citext citext,
  reason text not null,
  created_at timestamptz not null default now(),
  primary key (workspace_id, kind, value_hash)
);

alter table suppression enable row level security;
alter table suppression force row level security;
create policy suppression_isolation on suppression
  using (workspace_id = nullif(current_setting('app.workspace_id', true), '')::uuid);

grant select, insert, update, delete on suppression to outreach_app;

-- migrate:down

revoke select, insert, update, delete on suppression from outreach_app;
drop table if exists suppression;
