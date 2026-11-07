-- name: AdvisoryLockMailbox :exec
select pg_advisory_xact_lock(hashtext($1::text));

-- name: InsertThread :one
insert into threads (workspace_id, id, mailbox_id, subject, last_message_at, unread)
values ($1, $2, $3, $4, $5, $6)
returning workspace_id, id, mailbox_id, contact_id, subject, gmail_thread_id, last_message_at, unread;

-- name: GetThreadByRFC :one
select t.workspace_id, t.id, t.mailbox_id, t.contact_id, t.subject, t.gmail_thread_id, t.last_message_at, t.unread
from threads t
join messages m on m.workspace_id = t.workspace_id and m.thread_id = t.id
where m.workspace_id = $1 and m.mailbox_id = $2 and m.rfc_message_id = $3
limit 1;

-- name: GetThreadBySubject :one
select workspace_id, id, mailbox_id, contact_id, subject, gmail_thread_id, last_message_at, unread
from threads
where workspace_id = $1 and mailbox_id = $2
  and trim(regexp_replace(lower(coalesce(subject, '')), '((re|fw|fwd):\s*)+', '', 'g')) = $3
order by last_message_at desc nulls last
limit 1;

-- name: TouchThread :exec
update threads
set last_message_at = $4, unread = unread or $5
where workspace_id = $1 and id = $2 and mailbox_id = $3;

-- name: InsertMessage :exec
insert into messages (
  workspace_id, id, mailbox_id, thread_id, direction, rfc_message_id, in_reply_to,
  from_addr, to_addrs, subject, body_text, sent_at, received_at
) values (
  $1, $2, $3, $4, $5, $6, $7,
  $8, $9, $10, $11, $12, $13
)
on conflict (workspace_id, mailbox_id, rfc_message_id) do nothing;

-- name: ListThreadsForMailbox :many
select workspace_id, id, mailbox_id, contact_id, subject, gmail_thread_id, last_message_at, unread
from threads
where workspace_id = $1 and mailbox_id = $2
order by last_message_at desc nulls last;

-- name: ListMessagesForThread :many
select workspace_id, id, mailbox_id, thread_id, direction, rfc_message_id, in_reply_to,
  from_addr, to_addrs, subject, body_text, sent_at, received_at
from messages
where workspace_id = $1 and thread_id = $2
order by coalesce(sent_at, received_at, now());

-- name: UpsertSyncState :exec
insert into sync_state (workspace_id, mailbox_id, folder, last_synced_at, last_error)
values ($1, $2, $3, $4, $5)
on conflict (workspace_id, mailbox_id, folder) do update
set last_synced_at = excluded.last_synced_at, last_error = excluded.last_error;

-- name: DeleteMessagesForWorkspace :exec
delete from messages where workspace_id = $1;

-- name: DeleteThreadsForWorkspace :exec
delete from threads where workspace_id = $1;

-- name: DeleteSyncStateForWorkspace :exec
delete from sync_state where workspace_id = $1;
