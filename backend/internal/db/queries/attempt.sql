-- name: InsertSendAttempt :exec
insert into send_attempts (workspace_id, id, ledger_id, attempt, outcome, provider_response)
values ($1, $2, $3, $4, $5, $6);

-- name: DeleteAttemptsForWorkspace :exec
delete from send_attempts
where workspace_id = $1;
