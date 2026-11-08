-- name: InsertSecret :exec
insert into secrets (workspace_id, id, kind, key_version, wrapped_dek, nonce, ciphertext, hint)
values ($1, $2, $3, $4, $5, $6, $7, $8);

-- name: GetSecret :one
select key_version, wrapped_dek, nonce, ciphertext
from secrets
where workspace_id = $1 and id = $2;

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
  secret_id, service_account_id, sending_domain_id, timezone,
  window_days, window_start, window_end,
  daily_limit, hourly_limit, min_gap_s, max_gap_s, hard_daily_ceiling,
  warmup_profile_id, effective_daily_limit, status
) values (
  $1, $2, $3, $4, $5, $6,
  $7, $8, $9, $10, $11, $12,
  $13, $14, $15, $16,
  $17, $18, $19,
  $20, $21, $22, $23, $24,
  $25, $26, $27
)
returning
  workspace_id, id, email, display_name, auth_kind, provider,
  smtp_host, smtp_port, smtp_tls, imap_host, imap_port, username,
  sending_domain_id, timezone, warmup_profile_id, warmup_sending_day,
  effective_daily_limit, status, created_at;

-- name: ListMailboxes :many
select
  m.workspace_id, m.id, m.email, m.display_name, m.auth_kind, m.provider,
  m.smtp_host, m.smtp_port, m.smtp_tls, m.imap_host, m.imap_port, m.username,
  m.sending_domain_id, m.timezone, m.warmup_profile_id, m.warmup_sending_day,
  m.effective_daily_limit, m.status, m.reply_mode, m.created_at,
  w.kind as warmup_kind,
  d.domain, d.daily_limit as domain_daily_limit
from mailboxes m
join warmup_profiles w
  on w.workspace_id = m.workspace_id and w.id = m.warmup_profile_id
join sending_domains d
  on d.workspace_id = m.workspace_id and d.id = m.sending_domain_id
where m.workspace_id = $1
order by m.email;

-- name: ListSendingDomains :many
select workspace_id, id, domain, daily_limit, first_seen_at
from sending_domains
where workspace_id = $1
order by domain;

-- name: GetMailboxDial :one
select
  m.email, m.auth_kind, m.smtp_host, m.smtp_port, m.username,
  m.imap_host, m.imap_port,
  s.key_version, s.wrapped_dek, s.nonce, s.ciphertext
from mailboxes m
left join secrets s on s.workspace_id = m.workspace_id and s.id = m.secret_id
where m.workspace_id = $1 and m.id = $2;

-- name: GetMailbox :one
select
  workspace_id, id, email, display_name, auth_kind, provider,
  smtp_host, smtp_port, smtp_tls, imap_host, imap_port, username,
  sending_domain_id, timezone, warmup_profile_id, warmup_sending_day,
  effective_daily_limit, status, created_at
from mailboxes
where workspace_id = $1 and id = $2;

-- name: GetMailboxWindow :one
select timezone, window_days, window_start, window_end, status
from mailboxes
where workspace_id = $1 and id = $2;

-- name: LockMailbox :one
select
  workspace_id, id, email, display_name, auth_kind, provider,
  smtp_host, smtp_port, smtp_tls, imap_host, imap_port, username,
  sending_domain_id, timezone, warmup_profile_id, warmup_sending_day,
  effective_daily_limit, status, created_at
from mailboxes
where workspace_id = $1 and id = $2
for update;

-- name: ListMailboxesByIDs :many
select
  workspace_id, id, email, display_name, auth_kind, provider,
  smtp_host, smtp_port, smtp_tls, imap_host, imap_port, username,
  sending_domain_id, timezone, warmup_profile_id, warmup_sending_day,
  effective_daily_limit, status, created_at
from mailboxes
where workspace_id = $1 and id = any(sqlc.arg('ids')::uuid[]);

-- name: GetSendingDomain :one
select workspace_id, id, domain, daily_limit, first_seen_at
from sending_domains
where workspace_id = $1 and id = $2;

-- name: LockSendingDomain :one
select workspace_id, id, domain, daily_limit, first_seen_at
from sending_domains
where workspace_id = $1 and id = $2
for update;

-- name: SetMailboxStatus :exec
update mailboxes
set status = $3, status_reason = $4
where workspace_id = $1 and id = $2;

-- name: SetMailboxWarmup :exec
update mailboxes
set warmup_profile_id = $3, effective_daily_limit = $4
where workspace_id = $1 and id = $2;

-- name: GetMailboxReplyMode :one
select reply_mode
from mailboxes
where workspace_id = $1 and id = $2;

-- name: SetMailboxReplyMode :exec
update mailboxes
set reply_mode = $3
where workspace_id = $1 and id = $2;

-- name: SetMailboxDomain :exec
update mailboxes
set sending_domain_id = $3
where workspace_id = $1 and id = $2;

-- name: DeleteMailboxesForWorkspace :exec
delete from mailboxes
where workspace_id = $1;

-- name: InsertGoogleServiceAccount :one
insert into google_service_accounts (
  workspace_id, id, domain, client_email, client_id, secret_id, scopes
) values ($1, $2, $3, $4, $5, $6, $7)
returning workspace_id, id, domain, client_email, client_id, secret_id, scopes, created_at;

-- name: GetGoogleServiceAccountByDomain :one
select workspace_id, id, domain, client_email, client_id, secret_id, scopes, created_at
from google_service_accounts
where workspace_id = $1 and domain = $2;

-- name: ListGoogleServiceAccounts :many
select workspace_id, id, domain, client_email, client_id, secret_id, scopes, created_at
from google_service_accounts
where workspace_id = $1
order by domain;

-- name: DeleteGoogleServiceAccountsForWorkspace :exec
delete from google_service_accounts
where workspace_id = $1;

-- name: DeleteSecretsForWorkspace :exec
delete from secrets
where workspace_id = $1;
