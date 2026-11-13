-- name: InsertMailEvent :one
insert into mail_events (
  workspace_id, id, enrolment_id, draft_id, kind, url, at
) values (
  $1, $2, $3, $4, $5, $6, $7
)
returning workspace_id, id, enrolment_id, draft_id, kind, url, at;

-- name: ListMailEventsForEnrolment :many
select workspace_id, id, enrolment_id, draft_id, kind, url, at
from mail_events
where workspace_id = $1 and enrolment_id = $2
order by at, id;

-- name: TouchEnrolmentOpened :exec
update enrolments
set opened_at = coalesce(opened_at, sqlc.arg('at'))
where workspace_id = sqlc.arg('workspace_id') and id = sqlc.arg('id');

-- name: TouchEnrolmentClicked :exec
update enrolments
set clicked_at = coalesce(clicked_at, sqlc.arg('at'))
where workspace_id = sqlc.arg('workspace_id') and id = sqlc.arg('id');

-- name: GetEnrolmentTracking :one
select opened_at, clicked_at
from enrolments
where workspace_id = $1 and id = $2;

-- name: GetDraftEnrolment :one
select enrolment_id
from drafts
where workspace_id = $1 and id = $2;

-- name: DeleteMailEventsForWorkspace :exec
delete from mail_events
where workspace_id = $1;
