-- name: InsertSecret :exec
insert into secrets (workspace_id, id, kind, key_version, wrapped_dek, nonce, ciphertext, hint)
values ($1, $2, $3, $4, $5, $6, $7, $8);

-- name: GetSendingDomainByName :one
select workspace_id, id, domain, daily_limit, first_seen_at
from sending_domains
where workspace_id = $1 and domain = $2;

-- name: GetWarmupProfileByKind :one
select workspace_id, id, name, kind, spec, version
from warmup_profiles
where workspace_id = $1 and kind = $2
limit 1;

-- name: GetWarmupProfile :one
select workspace_id, id, name, kind, spec, version
from warmup_profiles
where workspace_id = $1 and id = $2;

-- name: InsertMailbox :one
insert into mailboxes (
  workspace_id, id, email, display_name, auth_kind, provider,
  smtp_host, smtp_port, smtp_tls, imap_host, imap_port, username,
  secret_id, sending_domain_id, timezone,
  window_days, window_start, window_end,
  daily_limit, hourly_limit, min_gap_s, max_gap_s, hard_daily_ceiling,
  warmup_profile_id, effective_daily_limit, status
) values (
  $1, $2, $3, $4, $5, $6,
  $7, $8, $9, $10, $11, $12,
  $13, $14, $15,
  $16, $17, $18,
  $19, $20, $21, $22, $23,
  $24, $25, $26
)
returning
  workspace_id, id, email, display_name, auth_kind, provider,
  smtp_host, smtp_port, smtp_tls, imap_host, imap_port, username,
  sending_domain_id, timezone, warmup_profile_id, warmup_sending_day,
  effective_daily_limit, status, created_at;

-- name: ListMailboxes :many
select
  workspace_id, id, email, display_name, auth_kind, provider,
  smtp_host, smtp_port, smtp_tls, imap_host, imap_port, username,
  sending_domain_id, timezone, warmup_profile_id, warmup_sending_day,
  effective_daily_limit, status, created_at
from mailboxes
where workspace_id = $1
order by email;

-- name: GetMailbox :one
select
  workspace_id, id, email, display_name, auth_kind, provider,
  smtp_host, smtp_port, smtp_tls, imap_host, imap_port, username,
  sending_domain_id, timezone, warmup_profile_id, warmup_sending_day,
  effective_daily_limit, status, created_at
from mailboxes
where workspace_id = $1 and id = $2;

-- name: SetMailboxStatus :exec
update mailboxes
set status = $3, status_reason = $4
where workspace_id = $1 and id = $2;

-- name: SetMailboxWarmup :exec
update mailboxes
set warmup_profile_id = $3, effective_daily_limit = $4
where workspace_id = $1 and id = $2;

-- name: SetMailboxDomain :exec
update mailboxes
set sending_domain_id = $3
where workspace_id = $1 and id = $2;

-- name: DeleteMailboxesForWorkspace :exec
delete from mailboxes
where workspace_id = $1;

-- name: DeleteSecretsForWorkspace :exec
delete from secrets
where workspace_id = $1;
