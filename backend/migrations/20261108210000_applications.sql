-- migrate:up

create table applications (
  workspace_id uuid not null,
  id uuid not null,
  enrolment_id uuid not null,
  contact_id uuid not null,
  job_id uuid,
  profile_id uuid,
  stage text not null check (stage in
    ('drafted','sent','followed_up','replied','interview','offer','rejected','closed')),
  stage_changed_at timestamptz not null default now(),
  primary key (workspace_id, id),
  unique (workspace_id, enrolment_id),
  foreign key (workspace_id, enrolment_id) references enrolments (workspace_id, id) on delete cascade,
  foreign key (workspace_id, contact_id) references contacts (workspace_id, id),
  foreign key (workspace_id, job_id) references jobs (workspace_id, id),
  foreign key (workspace_id, profile_id) references profiles (workspace_id, id)
);

alter table applications enable row level security;
alter table applications force row level security;
create policy applications_isolation on applications
  using (workspace_id = nullif(current_setting('app.workspace_id', true), '')::uuid);

grant select, insert, update, delete on applications to outreach_app;

insert into applications (workspace_id, id, enrolment_id, contact_id, job_id, profile_id, stage)
select e.workspace_id, gen_random_uuid(), e.id, e.contact_id, e.job_id, e.profile_id, 'drafted'
from enrolments e;

-- migrate:down

revoke select, insert, update, delete on applications from outreach_app;
drop table if exists applications;
