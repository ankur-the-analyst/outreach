-- migrate:up

create table drafts (
  workspace_id uuid not null references workspaces (id),
  id uuid not null,
  enrolment_id uuid,
  step_position int,
  mailbox_id uuid not null,
  to_addr citext not null,
  rfc_message_id text not null,
  kind text not null check (kind in ('outbound', 'reply')),
  subject text not null,
  body_html text not null,
  body_text text not null,
  facts_used jsonb not null default '[]',
  retrieval jsonb,
  new_recipient_from_inbound boolean not null default false,
  status text not null check (status in ('pending', 'approved', 'rejected', 'sent', 'superseded', 'cancelled')),
  approved_by uuid,
  created_at timestamptz not null default now(),
  primary key (workspace_id, id),
  foreign key (workspace_id, mailbox_id) references mailboxes (workspace_id, id)
);

alter table drafts enable row level security;
alter table drafts force row level security;
create policy drafts_isolation on drafts
  using (workspace_id = nullif(current_setting('app.workspace_id', true), '')::uuid);

grant select, insert, update, delete on drafts to outreach_app;

-- migrate:down

revoke select, insert, update, delete on drafts from outreach_app;
drop table if exists drafts;
