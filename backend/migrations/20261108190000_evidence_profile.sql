-- migrate:up

alter table evidence_items
  add column profile_id uuid;

alter table evidence_items
  add constraint evidence_items_profile_fk
    foreign key (workspace_id, profile_id) references profiles (workspace_id, id);

-- migrate:down

alter table evidence_items drop constraint if exists evidence_items_profile_fk;
alter table evidence_items drop column if exists profile_id;
