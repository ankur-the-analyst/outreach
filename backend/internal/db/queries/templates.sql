-- name: ListTemplates :many
select workspace_id, id, name, subject, body_text, body_html, created_at
from templates
where workspace_id = $1
order by created_at desc;

-- name: GetTemplate :one
select workspace_id, id, name, subject, body_text, body_html, created_at
from templates
where workspace_id = $1 and id = $2;

-- name: InsertTemplate :one
insert into templates (
  workspace_id, id, name, subject, body_text, body_html
) values (
  $1, $2, $3, $4, $5, $6
)
returning workspace_id, id, name, subject, body_text, body_html, created_at;

-- name: DeleteTemplatesForWorkspace :exec
delete from templates where workspace_id = $1;
