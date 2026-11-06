-- migrate:up

create table google_service_accounts (
  workspace_id uuid not null references workspaces (id),
  id uuid not null,
  domain citext not null,
  client_email text not null,
  client_id text not null,
  secret_id uuid not null,
  scopes text[] not null,
  created_at timestamptz not null default now(),
  primary key (workspace_id, id),
  unique (workspace_id, domain),
  foreign key (workspace_id, secret_id) references secrets (workspace_id, id)
);

alter table google_service_accounts enable row level security;
alter table google_service_accounts force row level security;
create policy google_service_accounts_isolation on google_service_accounts
  using (workspace_id = nullif(current_setting('app.workspace_id', true), '')::uuid);

grant select, insert, update, delete on google_service_accounts to outreach_app;

alter table mailboxes
  add constraint mailboxes_service_account_fk
  foreign key (workspace_id, service_account_id)
  references google_service_accounts (workspace_id, id);

-- migrate:down

alter table mailboxes drop constraint if exists mailboxes_service_account_fk;
revoke select, insert, update, delete on google_service_accounts from outreach_app;
drop table if exists google_service_accounts;
