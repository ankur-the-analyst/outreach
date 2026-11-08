-- migrate:up

create table health_samples (
  workspace_id uuid not null references workspaces (id),
  id bigserial,
  mailbox_id uuid,
  sending_domain_id uuid,
  metric text not null,
  value double precision not null,
  "window" text not null,
  created_at timestamptz not null default now(),
  primary key (workspace_id, id)
);

alter table health_samples enable row level security;
alter table health_samples force row level security;
create policy health_samples_isolation on health_samples
  using (workspace_id = nullif(current_setting('app.workspace_id', true), '')::uuid);

grant select, insert, update, delete on health_samples to outreach_app;
grant usage, select on sequence health_samples_id_seq to outreach_app;

-- migrate:down

revoke usage, select on sequence health_samples_id_seq from outreach_app;
revoke select, insert, update, delete on health_samples from outreach_app;
drop table if exists health_samples;
