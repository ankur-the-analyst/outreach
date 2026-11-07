-- migrate:up

create table sender_identities (
  workspace_id uuid primary key references workspaces (id),
  legal_name text not null,
  reply_address citext not null,
  postal_address text not null
);

create table campaigns (
  workspace_id uuid not null references workspaces (id),
  id uuid not null,
  name text not null,
  profile_id uuid,
  intent text not null check (intent in ('job','referral','cold','freelance','informational','custom')),
  sender_strategy text not null check (sender_strategy in
    ('round_robin','sequential','capacity_first','weighted','domain_spread','manual')),
  status text not null check (status in ('draft','running','paused','stopped','completed')),
  track_opens boolean not null default false,
  track_clicks boolean not null default false,
  list_unsubscribe boolean not null default false,
  new_contacts_per_day int not null,
  starts_at timestamptz,
  company_cooldown_days int not null default 30,
  ai_instructions text,
  undo_hold_s int not null default 300,
  created_at timestamptz not null default now(),
  primary key (workspace_id, id),
  foreign key (workspace_id, profile_id) references profiles (workspace_id, id)
);

create table campaign_senders (
  workspace_id uuid not null,
  campaign_id uuid not null,
  mailbox_id uuid not null,
  position int not null,
  weight int not null default 1,
  primary key (workspace_id, campaign_id, mailbox_id),
  foreign key (workspace_id, campaign_id) references campaigns (workspace_id, id),
  foreign key (workspace_id, mailbox_id) references mailboxes (workspace_id, id)
);

create table campaign_steps (
  workspace_id uuid not null,
  id uuid not null,
  campaign_id uuid not null,
  position int not null,
  kind text not null check (kind in ('email','wait')),
  wait_days int,
  content text check (content in ('ai','template')),
  template_id uuid,
  ai_instructions text,
  primary key (workspace_id, id),
  unique (workspace_id, campaign_id, position),
  foreign key (workspace_id, campaign_id) references campaigns (workspace_id, id)
);

create table enrolments (
  workspace_id uuid not null references workspaces (id),
  id uuid not null,
  campaign_id uuid not null,
  contact_id uuid not null,
  job_id uuid,
  profile_id uuid,
  intent text not null check (intent in ('job','referral','cold','freelance','informational','custom')),
  context jsonb not null default '{}',
  mailbox_id uuid,
  thread_id uuid,
  current_step int not null default 0,
  status text not null check (status in
    ('pending','active','waiting_review','held','replied','bounced','unsubscribed','completed','stopped','failed')),
  next_action_at timestamptz,
  hold_until timestamptz,
  stop_reason text,
  outcome text,
  created_at timestamptz not null default now(),
  primary key (workspace_id, id),
  unique (workspace_id, campaign_id, contact_id),
  foreign key (workspace_id, campaign_id) references campaigns (workspace_id, id),
  foreign key (workspace_id, contact_id) references contacts (workspace_id, id),
  foreign key (workspace_id, job_id) references jobs (workspace_id, id),
  foreign key (workspace_id, profile_id) references profiles (workspace_id, id),
  foreign key (workspace_id, mailbox_id) references mailboxes (workspace_id, id)
);

alter table sender_identities enable row level security;
alter table sender_identities force row level security;
create policy sender_identities_isolation on sender_identities
  using (workspace_id = nullif(current_setting('app.workspace_id', true), '')::uuid);

alter table campaigns enable row level security;
alter table campaigns force row level security;
create policy campaigns_isolation on campaigns
  using (workspace_id = nullif(current_setting('app.workspace_id', true), '')::uuid);

alter table campaign_senders enable row level security;
alter table campaign_senders force row level security;
create policy campaign_senders_isolation on campaign_senders
  using (workspace_id = nullif(current_setting('app.workspace_id', true), '')::uuid);

alter table campaign_steps enable row level security;
alter table campaign_steps force row level security;
create policy campaign_steps_isolation on campaign_steps
  using (workspace_id = nullif(current_setting('app.workspace_id', true), '')::uuid);

alter table enrolments enable row level security;
alter table enrolments force row level security;
create policy enrolments_isolation on enrolments
  using (workspace_id = nullif(current_setting('app.workspace_id', true), '')::uuid);

grant select, insert, update, delete on sender_identities, campaigns, campaign_senders, campaign_steps, enrolments to outreach_app;

-- migrate:down

revoke select, insert, update, delete on sender_identities, campaigns, campaign_senders, campaign_steps, enrolments from outreach_app;
drop table if exists enrolments;
drop table if exists campaign_steps;
drop table if exists campaign_senders;
drop table if exists campaigns;
drop table if exists sender_identities;
