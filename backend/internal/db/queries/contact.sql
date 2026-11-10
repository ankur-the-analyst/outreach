-- name: InsertCompany :one
insert into companies (workspace_id, id, domain, name)
values ($1, $2, $3, $4)
returning workspace_id, id, domain, name, last_contacted_at, cooldown_days, cooldown_override_reason, created_at;

-- name: GetCompanyByDomain :one
select workspace_id, id, domain, name, last_contacted_at, cooldown_days, cooldown_override_reason, created_at
from companies
where workspace_id = $1 and domain = $2;

-- name: GetCompany :one
select workspace_id, id, domain, name, last_contacted_at, cooldown_days, cooldown_override_reason, created_at
from companies
where workspace_id = $1 and id = $2;

-- name: InsertContact :one
insert into contacts (
  workspace_id, id, email, first_name, last_name, title, company_id,
  linkedin_url, timezone, region, role_class, provenance, readiness, custom
) values (
  $1, $2, $3, $4, $5, $6, $7,
  $8, $9, $10, $11, $12, $13, $14
)
returning
  workspace_id, id, email, first_name, last_name, title, company_id,
  linkedin_url, timezone, region, role_class, provenance, readiness, created_at;

-- name: ListContacts :many
select
  workspace_id, id, email, first_name, last_name, title, company_id,
  linkedin_url, timezone, region, role_class, provenance, readiness, created_at
from contacts
where workspace_id = $1
  and (sqlc.narg('after')::uuid is null or id < sqlc.narg('after'))
order by id desc
limit sqlc.arg('n');

-- name: GetContact :one
select
  workspace_id, id, email, first_name, last_name, title, company_id,
  linkedin_url, timezone, region, role_class, provenance, readiness, created_at
from contacts
where workspace_id = $1 and id = $2;

-- name: UpdateContact :one
update contacts
set first_name = $3, last_name = $4, title = $5, linkedin_url = $6, timezone = $7, region = $8
where workspace_id = $1 and id = $2
returning
  workspace_id, id, email, first_name, last_name, title, company_id,
  linkedin_url, timezone, region, role_class, provenance, readiness, created_at;

-- name: DeleteContact :exec
delete from contacts
where workspace_id = $1 and id = $2;

-- name: DeleteContactsForWorkspace :exec
delete from contacts where workspace_id = $1;

-- name: DeleteCompaniesForWorkspace :exec
delete from companies where workspace_id = $1;
