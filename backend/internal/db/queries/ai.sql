-- name: GetWorkspaceAI :one
select workspace_id, providers, models, fallback, updated_at
from workspace_ai
where workspace_id = $1;

-- name: UpsertWorkspaceAI :one
insert into workspace_ai (workspace_id, providers, models, fallback, updated_at)
values ($1, $2, $3, $4, now())
on conflict (workspace_id) do update
set providers = excluded.providers,
    models = excluded.models,
    fallback = excluded.fallback,
    updated_at = now()
returning workspace_id, providers, models, fallback, updated_at;

-- name: DeleteWorkspaceAIForWorkspace :exec
delete from workspace_ai
where workspace_id = $1;
