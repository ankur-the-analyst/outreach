-- migrate:up

create table threads (
  workspace_id uuid not null references workspaces (id),
  id uuid not null,
  mailbox_id uuid not null,
  contact_id uuid,
  subject text,
  gmail_thread_id text,
  last_message_at timestamptz,
  unread boolean not null default false,
  primary key (workspace_id, id),
  foreign key (workspace_id, mailbox_id) references mailboxes (workspace_id, id)
);

create table messages (
  workspace_id uuid not null,
  id uuid not null,
  mailbox_id uuid not null,
  thread_id uuid not null,
  direction text not null check (direction in ('out', 'in')),
  rfc_message_id text not null,
  in_reply_to text,
  references_ids text[] not null default '{}',
  gmail_id text,
  imap_folder text,
  imap_uid bigint,
  from_addr citext,
  to_addrs text[] not null default '{}',
  subject text,
  body_text text,
  body_html_key text,
  sent_at timestamptz,
  received_at timestamptz,
  status text,
  kind text,
  classification jsonb,
  enrolment_id uuid,
  draft_id uuid,
  primary key (workspace_id, id),
  unique (workspace_id, mailbox_id, rfc_message_id),
  foreign key (workspace_id, mailbox_id) references mailboxes (workspace_id, id),
  foreign key (workspace_id, thread_id) references threads (workspace_id, id)
);

create table sync_state (
  workspace_id uuid not null,
  mailbox_id uuid not null,
  folder text not null,
  uid_validity bigint,
  last_uid bigint,
  gmail_history_id bigint,
  last_synced_at timestamptz,
  last_error text,
  primary key (workspace_id, mailbox_id, folder),
  foreign key (workspace_id, mailbox_id) references mailboxes (workspace_id, id)
);

alter table threads enable row level security;
alter table threads force row level security;
create policy threads_isolation on threads
  using (workspace_id = nullif(current_setting('app.workspace_id', true), '')::uuid);

alter table messages enable row level security;
alter table messages force row level security;
create policy messages_isolation on messages
  using (workspace_id = nullif(current_setting('app.workspace_id', true), '')::uuid);

alter table sync_state enable row level security;
alter table sync_state force row level security;
create policy sync_state_isolation on sync_state
  using (workspace_id = nullif(current_setting('app.workspace_id', true), '')::uuid);

grant select, insert, update, delete on threads, messages, sync_state to outreach_app;

-- migrate:down

revoke select, insert, update, delete on threads, messages, sync_state from outreach_app;
drop table if exists messages;
drop table if exists sync_state;
drop table if exists threads;
