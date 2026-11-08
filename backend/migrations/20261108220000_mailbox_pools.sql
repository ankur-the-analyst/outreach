-- migrate:up

create table mailbox_pools (
  workspace_id uuid not null references workspaces (id),
  id uuid not null,
  name text not null,
  primary key (workspace_id, id)
);

create table mailbox_pool_members (
  workspace_id uuid not null,
  pool_id uuid not null,
  mailbox_id uuid not null,
  primary key (workspace_id, pool_id, mailbox_id),
  foreign key (workspace_id, pool_id) references mailbox_pools (workspace_id, id),
  foreign key (workspace_id, mailbox_id) references mailboxes (workspace_id, id)
);

alter table mailbox_pools enable row level security;
alter table mailbox_pools force row level security;
create policy mailbox_pools_isolation on mailbox_pools
  using (workspace_id = nullif(current_setting('app.workspace_id', true), '')::uuid);

alter table mailbox_pool_members enable row level security;
alter table mailbox_pool_members force row level security;
create policy mailbox_pool_members_isolation on mailbox_pool_members
  using (workspace_id = nullif(current_setting('app.workspace_id', true), '')::uuid);

grant select, insert, update, delete on mailbox_pools, mailbox_pool_members to outreach_app;

-- migrate:down

revoke select, insert, update, delete on mailbox_pools, mailbox_pool_members from outreach_app;
drop table if exists mailbox_pool_members;
drop table if exists mailbox_pools;
