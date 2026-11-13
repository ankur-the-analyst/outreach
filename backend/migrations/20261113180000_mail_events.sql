-- migrate:up

create table mail_events (
  workspace_id uuid not null references workspaces (id),
  id uuid not null,
  enrolment_id uuid not null,
  draft_id uuid not null,
  kind text not null check (kind in ('open', 'click')),
  url text,
  at timestamptz not null default now(),
  primary key (workspace_id, id),
  foreign key (workspace_id, enrolment_id) references enrolments (workspace_id, id),
  foreign key (workspace_id, draft_id) references drafts (workspace_id, id)
);

create index mail_events_enrolment
  on mail_events (workspace_id, enrolment_id, kind, at);

alter table enrolments add column opened_at timestamptz;
alter table enrolments add column clicked_at timestamptz;

alter table mail_events enable row level security;
alter table mail_events force row level security;
create policy mail_events_isolation on mail_events
  using (workspace_id = nullif(current_setting('app.workspace_id', true), '')::uuid);

grant select, insert, update, delete on mail_events to outreach_app;

-- migrate:down

revoke select, insert, update, delete on mail_events from outreach_app;
alter table enrolments drop column if exists opened_at;
alter table enrolments drop column if exists clicked_at;
drop table if exists mail_events;
