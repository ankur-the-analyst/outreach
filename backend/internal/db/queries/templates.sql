-- name: ListTemplates :many
select workspace_id, id, name, subject, body_text, body_html, design_json, created_at
from templates
where workspace_id = $1
order by created_at desc;

-- name: GetTemplate :one
select workspace_id, id, name, subject, body_text, body_html, design_json, created_at
from templates
where workspace_id = $1 and id = $2;

-- name: InsertTemplate :one
insert into templates (
  workspace_id, id, name, subject, body_text, body_html, design_json
) values (
  $1, $2, $3, $4, $5, $6, $7
)
returning workspace_id, id, name, subject, body_text, body_html, design_json, created_at;

-- name: UpdateTemplate :one
update templates
set name = $3, subject = $4, body_text = $5, body_html = $6, design_json = $7
where workspace_id = $1 and id = $2
returning workspace_id, id, name, subject, body_text, body_html, design_json, created_at;

-- name: DeleteTemplatesForWorkspace :exec
delete from templates where workspace_id = $1;
