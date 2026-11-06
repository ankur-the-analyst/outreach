-- name: InsertUser :exec
insert into users (id, email, name, password_hash)
values ($1, $2, $3, $4);

-- name: GetUserByEmail :one
select id, email, name, password_hash, google_sub, created_at
from users
where email = $1;

-- name: InsertWorkspace :exec
insert into workspaces (id, name)
values ($1, $2);

-- name: InsertMembership :exec
insert into memberships (workspace_id, user_id, role)
values ($1, $2, $3);

-- name: ListMembershipsByUser :many
select workspace_id, user_id, role
from memberships
where user_id = $1;

-- name: InsertSession :exec
insert into sessions (id, user_id, workspace_id, expires_at)
values ($1, $2, $3, $4);

-- name: GetSession :one
select id, user_id, workspace_id, expires_at, created_at
from sessions
where id = $1;

-- name: DeleteSession :exec
delete from sessions
where id = $1;
