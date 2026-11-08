-- name: UpsertSenderIdentity :one
insert into sender_identities (workspace_id, legal_name, reply_address, postal_address)
values ($1, $2, $3, $4)
on conflict (workspace_id) do update
set legal_name = excluded.legal_name, reply_address = excluded.reply_address, postal_address = excluded.postal_address
returning workspace_id, legal_name, reply_address, postal_address;

-- name: GetSenderIdentity :one
select workspace_id, legal_name, reply_address, postal_address
from sender_identities
where workspace_id = $1;

-- name: InsertCampaign :one
insert into campaigns (
  workspace_id, id, name, profile_id, intent, sender_strategy, status,
  list_unsubscribe, new_contacts_per_day, ai_instructions, undo_hold_s
) values (
  $1, $2, $3, $4, $5, $6, $7,
  $8, $9, $10, $11
)
returning
  workspace_id, id, name, profile_id, intent, sender_strategy, status,
  list_unsubscribe, new_contacts_per_day, ai_instructions, undo_hold_s, created_at;

-- name: ListCampaigns :many
select
  workspace_id, id, name, profile_id, intent, sender_strategy, status,
  list_unsubscribe, new_contacts_per_day, ai_instructions, undo_hold_s, created_at
from campaigns
where workspace_id = $1
order by created_at desc;

-- name: GetCampaign :one
select
  workspace_id, id, name, profile_id, intent, sender_strategy, status,
  list_unsubscribe, new_contacts_per_day, ai_instructions, undo_hold_s, created_at
from campaigns
where workspace_id = $1 and id = $2;

-- name: LockCampaign :one
select id
from campaigns
where workspace_id = $1 and id = $2
for update;

-- name: CountCampaignFirstTouchToday :one
select count(*)::bigint as n
from drafts d
join enrolments e
  on e.workspace_id = d.workspace_id and e.id = d.enrolment_id
where d.workspace_id = $1
  and e.campaign_id = $2
  and d.step_position = 0
  and d.status in ('pending', 'approved', 'sent')
  and d.created_at >= sqlc.arg('from_at')::timestamptz
  and d.created_at < sqlc.arg('to_at')::timestamptz;

-- name: UpdateCampaignStatus :one
update campaigns
set status = $3, list_unsubscribe = $4, starts_at = coalesce(starts_at, $5)
where workspace_id = $1 and id = $2
returning
  workspace_id, id, name, profile_id, intent, sender_strategy, status,
  list_unsubscribe, new_contacts_per_day, ai_instructions, undo_hold_s, created_at;

-- name: InsertCampaignSender :exec
insert into campaign_senders (workspace_id, campaign_id, mailbox_id, position, weight)
values ($1, $2, $3, $4, $5);

-- name: ListCampaignSenders :many
select workspace_id, campaign_id, mailbox_id, position, weight
from campaign_senders
where workspace_id = $1 and campaign_id = $2
order by position;

-- name: InsertCampaignStep :one
insert into campaign_steps (
  workspace_id, id, campaign_id, position, kind, wait_days, content, ai_instructions
) values (
  $1, $2, $3, $4, $5, $6, $7, $8
)
returning workspace_id, id, campaign_id, position, kind, wait_days, content, ai_instructions;

-- name: ListCampaignSteps :many
select workspace_id, id, campaign_id, position, kind, wait_days, content, ai_instructions
from campaign_steps
where workspace_id = $1 and campaign_id = $2
order by position;

-- name: InsertEnrolment :one
insert into enrolments (
  workspace_id, id, campaign_id, contact_id, job_id, profile_id, intent, context, status
) values (
  $1, $2, $3, $4, $5, $6, $7, $8, $9
)
returning
  workspace_id, id, campaign_id, contact_id, job_id, profile_id, intent, context, status, created_at;

-- name: GetEnrolment :one
select
  workspace_id, id, campaign_id, contact_id, job_id, profile_id, intent, context,
  mailbox_id, current_step, status, next_action_at, created_at
from enrolments
where workspace_id = $1 and id = $2;

-- name: UpdateEnrolmentStatus :one
update enrolments
set status = $3
where workspace_id = $1 and id = $2
returning
  workspace_id, id, campaign_id, contact_id, job_id, profile_id, intent, context,
  mailbox_id, current_step, status, next_action_at, created_at;

-- name: ListOpenEnrolmentsByEmail :many
select e.id, e.status
from enrolments e
join contacts c
  on c.workspace_id = e.workspace_id and c.id = e.contact_id
where e.workspace_id = $1
  and c.email = sqlc.arg('email')
  and e.status in ('pending', 'active', 'waiting_review', 'held');

-- name: UpdateEnrolmentMailbox :exec
update enrolments
set mailbox_id = $3
where workspace_id = $1 and id = $2;

-- name: ListEnrolments :many
select
  workspace_id, id, campaign_id, contact_id, job_id, profile_id, intent, context, status, created_at
from enrolments
where workspace_id = $1 and campaign_id = $2
order by created_at desc;

-- name: ListTrackerRows :many
select
  e.id,
  e.status as enrolment_status,
  e.created_at,
  camp.id as campaign_id,
  camp.name as campaign_name,
  ct.email,
  ct.first_name,
  ct.last_name,
  co.name as company_name,
  coalesce(d.status, '') as draft_status,
  d.created_at as draft_created_at,
  coalesce(l.state, '') as ledger_state
from enrolments e
join campaigns camp
  on camp.workspace_id = e.workspace_id and camp.id = e.campaign_id
join contacts ct
  on ct.workspace_id = e.workspace_id and ct.id = e.contact_id
left join companies co
  on co.workspace_id = ct.workspace_id and co.id = ct.company_id
left join lateral (
  select status, created_at
  from drafts
  where workspace_id = e.workspace_id and enrolment_id = e.id
  order by created_at desc
  limit 1
) d on true
left join lateral (
  select state
  from send_ledger
  where workspace_id = e.workspace_id and enrolment_id = e.id
  order by created_at desc
  limit 1
) l on true
where e.workspace_id = $1
order by coalesce(d.created_at, e.created_at) desc;

-- name: CampaignWantsListUnsubscribe :one
select c.list_unsubscribe
from send_ledger l
join drafts d
  on d.workspace_id = l.workspace_id and d.id = l.draft_id
join enrolments e
  on e.workspace_id = d.workspace_id and e.id = d.enrolment_id
join campaigns c
  on c.workspace_id = e.workspace_id and c.id = e.campaign_id
where l.workspace_id = $1 and l.id = $2;

-- name: ListDueEmailSteps :many
select e.id as enrolment_id, s.position as step_position
from enrolments e
join campaigns c
  on c.workspace_id = e.workspace_id and c.id = e.campaign_id
join campaign_steps s
  on s.workspace_id = e.workspace_id
 and s.campaign_id = e.campaign_id
 and s.position = e.current_step
where e.workspace_id = $1
  and c.status = 'running'
  and e.status in ('pending', 'active')
  and s.kind = 'email'
  and (e.next_action_at is null or e.next_action_at <= sqlc.arg('now')::timestamptz)
order by e.created_at, e.id;

-- name: DeleteEnrolmentsForWorkspace :exec
delete from enrolments where workspace_id = $1;

-- name: DeleteEnrolmentsForContact :exec
delete from enrolments
where workspace_id = $1 and contact_id = $2;

-- name: DeleteCampaignSendersForWorkspace :exec
delete from campaign_senders where workspace_id = $1;

-- name: DeleteCampaignStepsForWorkspace :exec
delete from campaign_steps where workspace_id = $1;

-- name: DeleteCampaignsForWorkspace :exec
delete from campaigns where workspace_id = $1;

-- name: DeleteSenderIdentityForWorkspace :exec
delete from sender_identities where workspace_id = $1;
