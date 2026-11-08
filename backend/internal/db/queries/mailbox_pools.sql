-- name: InsertMailboxPool :one
insert into mailbox_pools (workspace_id, id, name)
values ($1, $2, $3)
returning workspace_id, id, name;

-- name: ListMailboxPools :many
select workspace_id, id, name
from mailbox_pools
where workspace_id = $1
order by name;

-- name: GetMailboxPool :one
select workspace_id, id, name
from mailbox_pools
where workspace_id = $1 and id = $2;

-- name: InsertMailboxPoolMember :exec
insert into mailbox_pool_members (workspace_id, pool_id, mailbox_id)
values ($1, $2, $3);

-- name: ListMailboxPoolMembers :many
select m.pool_id, m.mailbox_id, mb.email
from mailbox_pool_members m
join mailboxes mb
  on mb.workspace_id = m.workspace_id and mb.id = m.mailbox_id
where m.workspace_id = $1
order by mb.email;

-- name: DeleteMailboxPoolMembersForWorkspace :exec
delete from mailbox_pool_members
where workspace_id = $1;

-- name: DeleteMailboxPoolsForWorkspace :exec
delete from mailbox_pools
where workspace_id = $1;
