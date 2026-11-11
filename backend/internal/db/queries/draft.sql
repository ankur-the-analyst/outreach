-- name: InsertDraft :one
insert into drafts (
  workspace_id, id, enrolment_id, step_position, mailbox_id, to_addr, rfc_message_id, kind,
  subject, body_html, body_text, facts_used, status, created_at
) values (
  $1, $2, $3, $4, $5, $6, $7, $8,
  $9, $10, $11, $12, $13, $14
)
returning
  workspace_id, id, enrolment_id, step_position, mailbox_id, to_addr, rfc_message_id, kind,
  subject, body_html, body_text, facts_used, status, approved_by, created_at;

-- name: GetDraftByEnrolmentStep :one
select
  workspace_id, id, enrolment_id, step_position, mailbox_id, to_addr, rfc_message_id, kind,
  subject, body_html, body_text, facts_used, status, approved_by, created_at
from drafts
where workspace_id = $1
  and enrolment_id = $2
  and step_position = $3
  and status in ('pending', 'approved')
order by created_at desc
limit 1;

-- name: GetDraft :one
select
  workspace_id, id, mailbox_id, to_addr, rfc_message_id, kind,
  subject, body_html, body_text, facts_used, status, approved_by, created_at
from drafts
where workspace_id = $1 and id = $2;

-- name: UpdateDraftStatus :one
update drafts
set status = $3, approved_by = $4
where workspace_id = $1 and id = $2
returning
  workspace_id, id, mailbox_id, to_addr, rfc_message_id, kind,
  subject, body_html, body_text, status, approved_by, created_at;

-- name: UpdateDraftBody :one
update drafts
set subject = $3, body_text = $4, body_html = $5
where workspace_id = $1 and id = $2 and status = 'pending'
returning
  workspace_id, id, mailbox_id, to_addr, rfc_message_id, kind,
  subject, body_html, body_text, facts_used, status, approved_by, created_at;

-- name: ListReviewDrafts :many
select
  d.workspace_id, d.id, d.mailbox_id, d.to_addr, d.rfc_message_id,
  d.kind, d.subject, d.body_html, d.body_text, d.facts_used, d.status, d.approved_by, d.created_at,
  l.id as ledger_id, l.state as ledger_state, l.release_at,
  coalesce(rs.file_name, r.file_name, '') as attachment_name, r.from_email as attachment_from
from drafts d
left join send_ledger l
  on l.workspace_id = d.workspace_id
 and l.draft_id = d.id
 and l.state in ('held', 'reserved', 'sending', 'sent', 'unknown')
left join cv_renders r
  on r.workspace_id = d.workspace_id
 and r.id = d.attachment_render_id
left join resumes rs
  on rs.workspace_id = d.workspace_id
 and rs.id = d.attachment_resume_id
where d.workspace_id = $1
  and d.status in ('pending', 'approved')
  and (sqlc.narg('after')::uuid is null or d.id < sqlc.narg('after'))
order by d.id desc
limit sqlc.arg('n');

-- name: CountReviewDrafts :one
select count(*)::bigint as n
from drafts
where workspace_id = $1
  and status in ('pending', 'approved');

-- name: GetLedgerByDraft :one
select
  workspace_id, id, mailbox_id, sending_domain_id, enrolment_id, draft_id,
  step_position, is_first_touch, rfc_message_id, state, reserved_at, release_at, sent_at, created_at
from send_ledger
where workspace_id = $1 and draft_id = $2
order by created_at desc
limit 1;

-- name: DeleteDraftsForWorkspace :exec
delete from drafts
where workspace_id = $1;

-- name: DeleteDraftsForContact :exec
delete from drafts
where drafts.workspace_id = sqlc.arg('workspace_id')
  and (
    to_addr = sqlc.arg('email')
    or enrolment_id in (
      select id from enrolments
      where workspace_id = sqlc.arg('workspace_id') and contact_id = sqlc.arg('contact_id')
    )
  );
