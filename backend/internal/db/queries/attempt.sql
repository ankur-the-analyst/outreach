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
