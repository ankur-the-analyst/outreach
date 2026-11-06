-- migrate:up

create table secrets (
  workspace_id uuid not null references workspaces (id),
  id uuid not null,
  kind text not null check (kind in ('app_password', 'service_account_key', 'llm_key', 'oauth_token')),
  key_version int not null,
  wrapped_dek bytea not null,
  nonce bytea not null,
  ciphertext bytea not null,
  hint text,
  created_at timestamptz not null default now(),
  primary key (workspace_id, id)
);

alter table secrets enable row level security;
alter table secrets force row level security;
create policy secrets_isolation on secrets
  using (workspace_id = nullif(current_setting('app.workspace_id', true), '')::uuid);

grant select, insert, update, delete on secrets to outreach_app;

-- migrate:down

revoke select, insert, update, delete on secrets from outreach_app;
drop table if exists secrets;
