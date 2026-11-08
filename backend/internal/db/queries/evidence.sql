-- name: InsertEvidence :one
insert into evidence_items (
  workspace_id, id, profile_id, text, source_ref, tags
) values (
  $1, $2, $3, $4, $5, $6
)
returning workspace_id, id, profile_id, text, source_ref, tags, do_not_use, confirmed_at, embedding_model, created_at;

-- name: GetEvidence :one
select workspace_id, id, profile_id, text, source_ref, tags, do_not_use, confirmed_at, embedding_model, created_at
from evidence_items
where workspace_id = $1 and id = $2;

-- name: ListEvidence :many
select workspace_id, id, profile_id, text, source_ref, tags, do_not_use, confirmed_at, embedding_model, created_at
from evidence_items
where workspace_id = $1
order by created_at desc;

-- name: GetEvidenceByIDs :many
select workspace_id, id, profile_id, text, source_ref, tags, do_not_use, confirmed_at, embedding_model, created_at
from evidence_items
where workspace_id = $1 and id = any(sqlc.arg('ids')::uuid[]);

-- name: ConfirmEvidence :one
update evidence_items
set confirmed_at = now(),
    embedding = @embedding::vector,
    embedding_model = @embedding_model,
    embedded_at = now()
where workspace_id = @workspace_id and id = @id
returning workspace_id, id, profile_id, text, source_ref, tags, do_not_use, confirmed_at, embedding_model, created_at;

-- name: CountConfirmedEvidence :one
select count(*)
from evidence_items
where workspace_id = $1
  and confirmed_at is not null
  and not do_not_use
  and embedding_model = $2;

-- name: ListConfirmedEvidence :many
select workspace_id, id, profile_id, text, source_ref, tags, do_not_use, confirmed_at, embedding_model, created_at
from evidence_items
where workspace_id = $1
  and confirmed_at is not null
  and not do_not_use
  and embedding_model = $2
order by created_at desc;

-- name: RetrieveEvidence :many
select workspace_id, id, profile_id, text, source_ref, tags, do_not_use, confirmed_at, embedding_model, created_at
from evidence_items
where workspace_id = $1
  and confirmed_at is not null
  and not do_not_use
  and embedding_model = $2
order by embedding <=> @query::vector
limit 20;

-- name: DeleteEvidenceForWorkspace :exec
delete from evidence_items where workspace_id = $1;
