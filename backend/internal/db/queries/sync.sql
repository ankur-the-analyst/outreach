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
  from_addr, to_addrs, subject, body_text, body_html, sent_at, received_at
) values (
  $1, $2, $3, $4, $5, $6, $7,
  $8, $9, $10, $11, $12, $13, $14
)
on conflict (workspace_id, mailbox_id, rfc_message_id) do update
set body_html = case
      when excluded.body_html is not null and excluded.body_html <> '' then excluded.body_html
      else messages.body_html
    end,
    body_text = case
      when excluded.body_html is not null and excluded.body_html <> ''
        and excluded.body_text is not null and excluded.body_text <> ''
        then excluded.body_text
      else messages.body_text
    end;

-- name: GetMessageByRFC :one
select id, direction, classification, from_addr, subject, body_text
from messages
where workspace_id = $1 and mailbox_id = $2 and rfc_message_id = $3;

-- name: GetMessage :one
select workspace_id, id, mailbox_id, direction, from_addr, subject, body_text, classification
from messages
where workspace_id = $1 and id = $2;

-- name: UpdateMessageClassification :exec
update messages
set classification = $3, kind = $4
where workspace_id = $1 and id = $2;

-- name: MarkThreadRead :execrows
update threads
set unread = false
where workspace_id = $1 and id = $2;

-- name: ListThreadsForMailbox :many
select workspace_id, id, mailbox_id, contact_id, subject, gmail_thread_id, last_message_at, unread
from threads
where workspace_id = $1 and mailbox_id = $2
order by last_message_at desc nulls last;

-- name: ListMessagesForThread :many
select workspace_id, id, mailbox_id, thread_id, direction, rfc_message_id, in_reply_to,
  from_addr, to_addrs, subject, body_text, body_html, sent_at, received_at, classification
from messages
where workspace_id = $1 and thread_id = $2
order by coalesce(sent_at, received_at, now());

-- name: MissingHTMLSpan :one
select
  min(coalesce(received_at, sent_at))::timestamptz as start_at,
  max(coalesce(received_at, sent_at))::timestamptz as end_at,
  count(*)::bigint as n
from messages
where workspace_id = $1 and mailbox_id = $2
  and (
    coalesce(body_html, '') = ''
    or body_html like '%cid:%'
  );

-- name: GetSyncState :one
select gmail_history_id, last_synced_at, last_error
from sync_state
where workspace_id = $1 and mailbox_id = $2 and folder = $3;

-- name: UpsertSyncState :exec
insert into sync_state (workspace_id, mailbox_id, folder, last_synced_at, last_error, gmail_history_id)
values ($1, $2, $3, $4, $5, sqlc.narg('gmail_history_id'))
on conflict (workspace_id, mailbox_id, folder) do update
set last_synced_at = excluded.last_synced_at,
    last_error = excluded.last_error,
    gmail_history_id = coalesce(excluded.gmail_history_id, sync_state.gmail_history_id);

-- name: OldestSyncAt :one
select min(last_synced_at)::timestamptz as last_at
from sync_state
where workspace_id = $1;

-- name: DeleteMessagesForWorkspace :exec
delete from messages where workspace_id = $1;

-- name: DeleteMessagesForContact :exec
delete from messages
where messages.workspace_id = sqlc.arg('workspace_id')
  and (
    from_addr = sqlc.arg('email')
    or sqlc.arg('email')::text = any(to_addrs)
    or enrolment_id in (
      select id from enrolments
      where workspace_id = sqlc.arg('workspace_id') and contact_id = sqlc.arg('contact_id')
    )
  );

-- name: DeleteThreadsForContact :exec
delete from threads
where workspace_id = $1 and contact_id = $2;

-- name: DeleteThreadsForWorkspace :exec
delete from threads where workspace_id = $1;

-- name: DeleteSyncStateForWorkspace :exec
delete from sync_state where workspace_id = $1;
