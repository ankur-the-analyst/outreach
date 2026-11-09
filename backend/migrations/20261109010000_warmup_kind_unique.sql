-- migrate:up

create unique index warmup_profiles_workspace_kind
  on warmup_profiles (workspace_id, kind);

-- migrate:down

drop index if exists warmup_profiles_workspace_kind;
