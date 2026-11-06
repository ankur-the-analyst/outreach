-- migrate:up

create table mailboxes (
  workspace_id uuid not null references workspaces (id),
  id uuid not null,
  email citext not null,
  display_name text,
  signature_html text,
  auth_kind text not null check (auth_kind in ('app_password', 'service_account', 'oauth', 'smtp_imap')),
  provider text not null,
  smtp_host text,
  smtp_port int,
  smtp_tls text,
  imap_host text,
  imap_port int,
  username text,
  secret_id uuid,
  service_account_id uuid,
  sending_domain_id uuid not null,
  timezone text not null,
  window_days int[] not null,
  window_start time not null,
  window_end time not null,
  daily_limit int not null,
  hourly_limit int not null,
  min_gap_s int not null,
  max_gap_s int not null,
  hard_daily_ceiling int not null,
  warmup_profile_id uuid not null,
  warmup_sending_day int not null default 0,
  effective_daily_limit int not null,
  reply_mode text not null default 'draft' check (reply_mode in ('draft', 'off')),
  status text not null check (status in ('healthy', 'warming', 'throttled', 'needs_reconnect', 'paused')),
  status_reason text,
  throttled_until timestamptz,
  last_sent_at timestamptz,
  next_free_at timestamptz,
  created_at timestamptz not null default now(),
  primary key (workspace_id, id),
  unique (workspace_id, email),
  foreign key (workspace_id, secret_id) references secrets (workspace_id, id),
  foreign key (workspace_id, sending_domain_id) references sending_domains (workspace_id, id),
  foreign key (workspace_id, warmup_profile_id) references warmup_profiles (workspace_id, id)
);

alter table mailboxes enable row level security;
alter table mailboxes force row level security;
create policy mailboxes_isolation on mailboxes
  using (workspace_id = nullif(current_setting('app.workspace_id', true), '')::uuid);

grant select, insert, update, delete on mailboxes to outreach_app;

-- migrate:down

revoke select, insert, update, delete on mailboxes from outreach_app;
drop table if exists mailboxes;
