-- migrate:up

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

alter table field_defs enable row level security;
alter table field_defs force row level security;
create policy field_defs_isolation on field_defs
  using (workspace_id = nullif(current_setting('app.workspace_id', true), '')::uuid);

grant select, insert, update, delete on field_defs to outreach_app;

-- migrate:down

revoke select, insert, update, delete on field_defs from outreach_app;
drop table if exists field_defs;
alter table companies drop column if exists custom;
alter table memberships drop column if exists attrs;
alter table workspaces drop column if exists attrs;
