-- name: InsertHealthSample :one
insert into health_samples (
  workspace_id, mailbox_id, sending_domain_id, metric, value, "window"
) values (
  $1, $2, $3, $4, $5, $6
)
returning workspace_id, id, mailbox_id, sending_domain_id, metric, value, "window", created_at;

-- name: ListHealthSamples :many
select workspace_id, id, mailbox_id, sending_domain_id, metric, value, "window", created_at
from health_samples
where workspace_id = $1
order by id desc
limit 50;

-- name: DeleteHealthSamplesForWorkspace :exec
delete from health_samples
where workspace_id = $1;
