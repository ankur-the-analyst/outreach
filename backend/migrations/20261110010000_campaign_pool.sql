-- migrate:up

alter table campaigns
  add column pool_id uuid;

alter table campaigns
  add constraint campaigns_pool_fk
  foreign key (workspace_id, pool_id)
  references mailbox_pools (workspace_id, id);

-- migrate:down

alter table campaigns drop constraint if exists campaigns_pool_fk;
alter table campaigns drop column if exists pool_id;
