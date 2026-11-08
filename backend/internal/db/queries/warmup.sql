-- name: InsertWarmupProfile :exec
insert into warmup_profiles (workspace_id, id, name, kind, spec, version)
values ($1, $2, $3, $4, $5, $6);

-- name: CountWarmupProfiles :one
select count(*)::bigint as n
from warmup_profiles;

-- name: ListWarmupProfiles :many
select workspace_id, id, name, kind, spec, version
from warmup_profiles
where workspace_id = $1
order by kind;

-- name: DeleteWarmupProfilesForWorkspace :exec
delete from warmup_profiles
where workspace_id = $1;

-- name: InsertSendingDomain :exec
insert into sending_domains (workspace_id, id, domain, daily_limit)
values ($1, $2, $3, $4);

-- name: DeleteSendingDomainsForWorkspace :exec
delete from sending_domains
where workspace_id = $1;
