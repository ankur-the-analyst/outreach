-- name: InsertSendAttempt :exec
insert into send_attempts (workspace_id, id, ledger_id, attempt, outcome, provider_response)
values ($1, $2, $3, $4, $5, $6);

-- name: CountAttempts :one
select count(*)::bigint as n
from send_attempts
where workspace_id = $1 and ledger_id = $2;

-- name: CountAttemptsByOutcome :one
select count(*)::bigint as n
from send_attempts
where workspace_id = $1 and ledger_id = $2 and outcome = $3;

-- name: DeleteAttemptsForWorkspace :exec
delete from send_attempts
where workspace_id = $1;

-- name: DeleteAttemptsForContact :exec
delete from send_attempts
where send_attempts.workspace_id = sqlc.arg('workspace_id')
  and ledger_id in (
    select l.id
    from send_ledger l
    left join enrolments e
      on e.workspace_id = l.workspace_id and e.id = l.enrolment_id
    left join drafts d
      on d.workspace_id = l.workspace_id and d.id = l.draft_id
    where l.workspace_id = sqlc.arg('workspace_id')
      and (
        e.contact_id = sqlc.arg('contact_id')
        or d.to_addr = sqlc.arg('email')
      )
  );
