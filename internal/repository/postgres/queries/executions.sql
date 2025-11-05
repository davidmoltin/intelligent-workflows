-- name: CreateExecution :one
INSERT INTO workflow_executions (
    workflow_id,
    execution_id,
    trigger_event,
    trigger_payload,
    context,
    status
) VALUES (
    $1, $2, $3, $4, $5, $6
) RETURNING *;

-- name: GetExecution :one
SELECT * FROM workflow_executions
WHERE id = $1;

-- name: GetExecutionByExecutionID :one
SELECT * FROM workflow_executions
WHERE execution_id = $1;

-- name: ListExecutions :many
SELECT * FROM workflow_executions
WHERE ($1::uuid IS NULL OR workflow_id = $1)
AND ($2::text IS NULL OR status = $2)
ORDER BY started_at DESC
LIMIT $3 OFFSET $4;

-- name: CountExecutions :one
SELECT COUNT(*) FROM workflow_executions
WHERE ($1::uuid IS NULL OR workflow_id = $1)
AND ($2::text IS NULL OR status = $2);

-- name: UpdateExecutionStatus :exec
UPDATE workflow_executions
SET
    status = $2,
    result = $3,
    completed_at = CASE WHEN $2 IN ('completed', 'failed', 'blocked', 'cancelled') THEN NOW() ELSE completed_at END,
    duration_ms = CASE WHEN $2 IN ('completed', 'failed', 'blocked', 'cancelled') THEN EXTRACT(EPOCH FROM (NOW() - started_at)) * 1000 ELSE duration_ms END,
    error_message = $4
WHERE id = $1;

-- name: CreateStepExecution :one
INSERT INTO step_executions (
    execution_id,
    step_id,
    step_type,
    status,
    input
) VALUES (
    $1, $2, $3, $4, $5
) RETURNING *;

-- name: UpdateStepExecution :exec
UPDATE step_executions
SET
    status = $2,
    output = $3,
    completed_at = CASE WHEN $2 IN ('completed', 'failed', 'skipped') THEN NOW() ELSE completed_at END,
    duration_ms = CASE WHEN $2 IN ('completed', 'failed', 'skipped') THEN EXTRACT(EPOCH FROM (NOW() - started_at)) * 1000 ELSE duration_ms END,
    error_message = $4
WHERE id = $1;

-- name: ListStepExecutions :many
SELECT * FROM step_executions
WHERE execution_id = $1
ORDER BY started_at ASC;

-- name: GetStepExecution :one
SELECT * FROM step_executions
WHERE id = $1;
