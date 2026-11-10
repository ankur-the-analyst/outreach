-- migrate:up

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

alter table templates enable row level security;
alter table templates force row level security;
create policy templates_isolation on templates
  using (workspace_id = nullif(current_setting('app.workspace_id', true), '')::uuid);

grant select, insert, update, delete on templates to outreach_app;

-- migrate:down

revoke select, insert, update, delete on templates from outreach_app;
alter table campaign_steps drop constraint if exists campaign_steps_workspace_id_template_id_fkey;
drop table if exists templates;
