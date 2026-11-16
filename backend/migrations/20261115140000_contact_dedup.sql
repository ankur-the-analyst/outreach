-- migrate:up
alter table workspaces
  add column contact_dedup jsonb not null default '{"action":"prevent","overwrite":false}';

-- migrate:down
alter table workspaces drop column contact_dedup;
