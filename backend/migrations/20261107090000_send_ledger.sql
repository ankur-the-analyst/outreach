-- migrate:up

create table send_ledger (
  workspace_id uuid not null references workspaces (id),
  id uuid not null,
  mailbox_id uuid not null,
  sending_domain_id uuid not null,
  enrolment_id uuid,
  draft_id uuid not null,
  step_position int,
  is_first_touch boolean not null,
  rfc_message_id text not null,
  state text not null check (state in
    ('held','reserved','sending','sent','unknown','released','cancelled','failed')),
  reserved_at timestamptz,
  release_at timestamptz,
  sent_at timestamptz,
  created_at timestamptz not null default now(),
  primary key (workspace_id, id),
  foreign key (workspace_id, mailbox_id) references mailboxes (workspace_id, id),
  foreign key (workspace_id, sending_domain_id) references sending_domains (workspace_id, id)
);

create unique index send_ledger_enrolment_step
  on send_ledger (workspace_id, enrolment_id, step_position)
  where enrolment_id is not null and step_position is not null
    and state not in ('cancelled','released','failed');

alter table send_ledger enable row level security;
alter table send_ledger force row level security;
create policy send_ledger_isolation on send_ledger
  using (workspace_id = nullif(current_setting('app.workspace_id', true), '')::uuid);

grant select, insert, update, delete on send_ledger to outreach_app;

-- migrate:down

revoke select, insert, update, delete on send_ledger from outreach_app;
drop table if exists send_ledger;
