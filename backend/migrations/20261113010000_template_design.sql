-- migrate:up

alter table templates
  add column design_json jsonb not null default '{}';

-- migrate:down

alter table templates drop column if exists design_json;
