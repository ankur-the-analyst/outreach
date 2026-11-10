-- name: ListFieldDefs :many
select workspace_id, id, object, key, label, kind, required, options, position
from field_defs
where workspace_id = $1
  and (sqlc.narg('object')::text is null or object = sqlc.narg('object'))
order by object, position, key;

-- name: CountFieldDefs :one
select count(*)::int as n
from field_defs
where workspace_id = $1 and object = $2;

-- name: InsertFieldDef :one
insert into field_defs (
  workspace_id, id, object, key, label, kind, required, options, position
) values (
  $1, $2, $3, $4, $5, $6, $7, $8, $9
)
returning workspace_id, id, object, key, label, kind, required, options, position;

-- name: DeleteFieldDefsForWorkspace :exec
delete from field_defs where workspace_id = $1;
