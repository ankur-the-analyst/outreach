-- migrate:up

create table send_attempts (
  workspace_id uuid not null,
  id uuid not null,
  ledger_id uuid not null,
  attempt int not null,
  outcome text not null,
  provider_response text,
  created_at timestamptz not null default now(),
  primary key (workspace_id, id),
  foreign key (workspace_id, ledger_id) references send_ledger (workspace_id, id)
);

alter table send_attempts enable row level security;
alter table send_attempts force row level security;
create policy send_attempts_isolation on send_attempts
  using (workspace_id = nullif(current_setting('app.workspace_id', true), '')::uuid);

grant select, insert, update, delete on send_attempts to outreach_app;

-- migrate:down

revoke select, insert, update, delete on send_attempts from outreach_app;
drop table if exists send_attempts;
