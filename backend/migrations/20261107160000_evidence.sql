-- migrate:up

create extension if not exists vector;

create table evidence_items (
  workspace_id uuid not null references workspaces (id),
  id uuid not null,
  text text not null,
  source_ref text,
  tags text[] not null default '{}',
  do_not_use boolean not null default false,
  confirmed_at timestamptz,
  embedding vector(8),
  embedding_model text,
  embedded_at timestamptz,
  created_at timestamptz not null default now(),
  primary key (workspace_id, id)
);

alter table evidence_items enable row level security;
alter table evidence_items force row level security;
create policy evidence_items_isolation on evidence_items
  using (workspace_id = nullif(current_setting('app.workspace_id', true), '')::uuid);

grant select, insert, update, delete on evidence_items to outreach_app;

-- migrate:down

revoke select, insert, update, delete on evidence_items from outreach_app;
drop table if exists evidence_items;
drop extension if exists vector;
