-- name: CountWorkspaces :one
select count(*)::bigint as n
from workspaces;

-- name: DeleteWorkspace :exec
delete from workspaces
where id = $1;
