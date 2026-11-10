-- name: NextCvVersion :one
select coalesce(max(version), 0)::int + 1
from cv_documents
where workspace_id = $1 and profile_id = $2;

-- name: InsertCvDocument :one
insert into cv_documents (workspace_id, id, profile_id, version, spec)
values ($1, $2, $3, $4, $5)
returning workspace_id, id, profile_id, version, spec, created_at;

-- name: LatestCvByProfile :one
select workspace_id, id, profile_id, version, spec, created_at
from cv_documents
where workspace_id = $1 and profile_id = $2
order by version desc
limit 1;

-- name: GetUntailoredRender :one
select workspace_id, id, cv_id, mailbox_id, job_id, file_key, file_name, mime, size, from_email, created_at
from cv_renders
where workspace_id = $1
  and cv_id = $2
  and mailbox_id = $3
  and job_id is null;

-- name: InsertCvRender :one
insert into cv_renders (
  workspace_id, id, cv_id, mailbox_id, file_key, file_name, mime, size, from_email
) values (
  $1, $2, $3, $4, $5, $6, $7, $8, $9
)
returning workspace_id, id, cv_id, mailbox_id, job_id, file_key, file_name, mime, size, from_email, created_at;

-- name: UpdateDraftAttachment :exec
update drafts
set attachment_render_id = $3
where workspace_id = $1 and id = $2;

-- name: GetDraftAttach :one
select id, mailbox_id, enrolment_id, attachment_render_id
from drafts
where workspace_id = $1 and id = $2;

-- name: DeleteCvRendersForWorkspace :exec
delete from cv_renders where workspace_id = $1;

-- name: DeleteCvDocumentsForWorkspace :exec
delete from cv_documents where workspace_id = $1;
