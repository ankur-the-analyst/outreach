-- name: CountWorkspaces :one
select count(*)::bigint as n
from workspaces;

-- name: GetWorkspace :one
select id, name, sends_paused, created_at
from workspaces
where id = $1;

-- name: UpdateWorkspaceName :exec
update workspaces
set name = $2
where id = $1;

-- name: TouchWorkspace :exec
update workspaces
set name = name
where id = $1;

-- name: SetSendsPaused :exec
update workspaces
set sends_paused = $2
where id = $1;

-- name: GetContactDedup :one
select contact_dedup
from workspaces
where id = $1;

-- name: SetContactDedup :exec
update workspaces
set contact_dedup = $2
where id = $1;

-- name: GetMembership :one
select workspace_id, user_id, role
from memberships
where workspace_id = $1 and user_id = $2;

-- name: GetMembershipAttrs :one
select attrs
from memberships
where workspace_id = $1 and user_id = $2;

-- name: GetOwnerAttrs :one
select attrs
from memberships
where workspace_id = $1 and role = 'owner'
order by user_id
limit 1;

-- name: GetOwner :one
select coalesce(u.name, '') as name, u.email::text as email, m.attrs
from memberships m
join users u on u.id = m.user_id
where m.workspace_id = $1 and m.role = 'owner'
order by m.user_id
limit 1;

-- name: UpdateMembershipAttrs :exec
update memberships
set attrs = $3
where workspace_id = $1 and user_id = $2;

-- name: ListMembers :many
select m.workspace_id, m.user_id, m.role, u.email, u.name
from memberships m
join users u on u.id = m.user_id
where m.workspace_id = $1
order by m.role, u.email;

-- name: DeleteSessionsForWorkspace :exec
delete from sessions
where workspace_id = $1;

-- name: DeleteMembershipsForWorkspace :exec
delete from memberships
where workspace_id = $1;

-- name: DeleteWorkspace :exec
delete from workspaces
where id = $1;

-- name: DeleteUser :exec
delete from users
where id = $1;
