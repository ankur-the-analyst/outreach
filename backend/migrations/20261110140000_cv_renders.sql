-- migrate:up

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

alter table cv_documents enable row level security;
alter table cv_documents force row level security;
create policy cv_documents_isolation on cv_documents
  using (workspace_id = nullif(current_setting('app.workspace_id', true), '')::uuid);

alter table cv_renders enable row level security;
alter table cv_renders force row level security;
create policy cv_renders_isolation on cv_renders
  using (workspace_id = nullif(current_setting('app.workspace_id', true), '')::uuid);

grant select, insert, update, delete on cv_documents, cv_renders to outreach_app;

-- migrate:down

alter table drafts drop constraint if exists drafts_workspace_id_attachment_render_id_fkey;
alter table drafts drop column if exists attachment_render_id;
revoke select, insert, update, delete on cv_documents, cv_renders from outreach_app;
drop table if exists cv_renders;
drop table if exists cv_documents;
