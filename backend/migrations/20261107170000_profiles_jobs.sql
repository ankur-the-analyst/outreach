-- migrate:up

create table profiles (
  workspace_id uuid not null references workspaces (id),
  id uuid not null,
  name text not null,
  about text,
  links jsonb not null default '{}',
  created_at timestamptz not null default now(),
  primary key (workspace_id, id)
);

create table resumes (
  workspace_id uuid not null references workspaces (id),
  id uuid not null,
  profile_id uuid not null,
  version int not null,
  file_key text not null,
  file_name text not null,
  mime text not null,
  text_content text not null,
  created_at timestamptz not null default now(),
  primary key (workspace_id, id),
  unique (workspace_id, profile_id, version),
  foreign key (workspace_id, profile_id) references profiles (workspace_id, id)
);

create table jobs (
  workspace_id uuid not null references workspaces (id),
  id uuid not null,
  source_url text,
  raw_text text not null,
  title text,
  analysis jsonb,
  analysis_status text not null default 'pending',
  created_at timestamptz not null default now(),
  primary key (workspace_id, id)
);

create table fits (
  workspace_id uuid not null references workspaces (id),
  id uuid not null,
  job_id uuid not null,
  profile_id uuid not null,
  resume_id uuid,
  score int,
  result jsonb,
  created_at timestamptz not null default now(),
  primary key (workspace_id, id),
  unique (workspace_id, job_id, profile_id),
  foreign key (workspace_id, job_id) references jobs (workspace_id, id),
  foreign key (workspace_id, profile_id) references profiles (workspace_id, id),
  foreign key (workspace_id, resume_id) references resumes (workspace_id, id)
);

alter table profiles enable row level security;
alter table profiles force row level security;
create policy profiles_isolation on profiles
  using (workspace_id = nullif(current_setting('app.workspace_id', true), '')::uuid);

alter table resumes enable row level security;
alter table resumes force row level security;
create policy resumes_isolation on resumes
  using (workspace_id = nullif(current_setting('app.workspace_id', true), '')::uuid);

alter table jobs enable row level security;
alter table jobs force row level security;
create policy jobs_isolation on jobs
  using (workspace_id = nullif(current_setting('app.workspace_id', true), '')::uuid);

alter table fits enable row level security;
alter table fits force row level security;
create policy fits_isolation on fits
  using (workspace_id = nullif(current_setting('app.workspace_id', true), '')::uuid);

grant select, insert, update, delete on profiles, resumes, jobs, fits to outreach_app;

-- migrate:down

revoke select, insert, update, delete on profiles, resumes, jobs, fits from outreach_app;
drop table if exists fits;
drop table if exists resumes;
drop table if exists jobs;
drop table if exists profiles;
