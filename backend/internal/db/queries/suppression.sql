-- name: InsertSuppression :exec
insert into suppression (workspace_id, kind, value_hash, value_citext, reason)
values ($1, $2, $3, $4, $5)
on conflict (workspace_id, kind, value_hash) do update
set reason = excluded.reason, value_citext = excluded.value_citext;

-- name: SuppressionExists :one
select exists(
  select 1 from suppression
  where workspace_id = $1 and kind = $2 and value_hash = $3
)::bool as present;

-- name: DeleteSuppressionForWorkspace :exec
delete from suppression
where workspace_id = $1;
