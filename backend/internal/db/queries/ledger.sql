-- name: InsertLedger :one
insert into send_ledger (
  workspace_id, id, mailbox_id, sending_domain_id, enrolment_id, draft_id,
  step_position, is_first_touch, rfc_message_id, state, reserved_at, release_at
) values (
  $1, $2, $3, $4, $5, $6,
  $7, $8, $9, $10, $11, $12
)
returning
  workspace_id, id, mailbox_id, sending_domain_id, enrolment_id, draft_id,
  step_position, is_first_touch, rfc_message_id, state, reserved_at, release_at, sent_at, created_at;

-- name: GetLedger :one
select
  workspace_id, id, mailbox_id, sending_domain_id, enrolment_id, draft_id,
  step_position, is_first_touch, rfc_message_id, state, reserved_at, release_at, sent_at, created_at
from send_ledger
where workspace_id = $1 and id = $2;

-- name: UpdateLedgerState :one
update send_ledger
set
  state = sqlc.arg('next_state'),
  reserved_at = case when sqlc.arg('next_state') = 'reserved' then now() else reserved_at end,
  sent_at = case when sqlc.arg('next_state') = 'sent' then now() else sent_at end
where workspace_id = sqlc.arg('workspace_id') and id = sqlc.arg('id') and state = sqlc.arg('from_state')
returning
  workspace_id, id, mailbox_id, sending_domain_id, enrolment_id, draft_id,
  step_position, is_first_touch, rfc_message_id, state, reserved_at, release_at, sent_at, created_at;

-- name: CountMailboxUsage :one
select count(*)::bigint as n
from send_ledger
where workspace_id = $1 and mailbox_id = $2
  and state in ('held', 'reserved', 'sending', 'sent', 'unknown')
  and coalesce(sent_at, reserved_at, created_at) > now() - interval '24 hours';

-- name: CountDomainUsage :one
select count(*)::bigint as n
from send_ledger
where workspace_id = $1 and sending_domain_id = $2
  and state in ('held', 'reserved', 'sending', 'sent', 'unknown')
  and coalesce(sent_at, reserved_at, created_at) > now() - interval '24 hours';

-- name: ListFirstTouchAt :many
select mailbox_id,
  max(coalesce(sent_at, reserved_at, created_at))::timestamptz as last_at
from send_ledger
where workspace_id = $1
  and is_first_touch = true
  and state in ('held', 'reserved', 'sending', 'sent', 'unknown')
group by mailbox_id;

-- name: DeleteLedgerForWorkspace :exec
delete from send_ledger
where workspace_id = $1;

-- name: DeleteLedgerForContact :exec
delete from send_ledger l
where l.workspace_id = sqlc.arg('workspace_id')
  and (
    exists (
      select 1 from enrolments e
      where e.workspace_id = l.workspace_id and e.id = l.enrolment_id
        and e.contact_id = sqlc.arg('contact_id')
    )
    or exists (
      select 1 from drafts d
      where d.workspace_id = l.workspace_id and d.id = l.draft_id
        and d.to_addr = sqlc.arg('email')
    )
  );
