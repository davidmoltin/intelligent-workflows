-- name: CreateWorkflow :one
INSERT INTO workflows (
    workflow_id,
    version,
    name,
    description,
    definition,
    enabled,
    created_by,
    tags
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8
) RETURNING *;

-- name: GetWorkflow :one
SELECT * FROM workflows
WHERE id = $1;

-- name: GetWorkflowByWorkflowID :one
SELECT * FROM workflows
WHERE workflow_id = $1
ORDER BY created_at DESC
LIMIT 1;

-- name: GetWorkflowByWorkflowIDAndVersion :one
SELECT * FROM workflows
WHERE workflow_id = $1 AND version = $2;

-- name: ListWorkflows :many
SELECT * FROM workflows
WHERE ($1::boolean IS NULL OR enabled = $1)
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: CountWorkflows :one
SELECT COUNT(*) FROM workflows
WHERE ($1::boolean IS NULL OR enabled = $1);

-- name: UpdateWorkflow :one
UPDATE workflows
SET
    name = COALESCE($2, name),
    description = COALESCE($3, description),
    definition = COALESCE($4, definition),
    tags = COALESCE($5, tags),
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: DeleteWorkflow :exec
DELETE FROM workflows
WHERE id = $1;

-- name: EnableWorkflow :exec
UPDATE workflows
SET enabled = true, updated_at = NOW()
WHERE id = $1;

-- name: DisableWorkflow :exec
UPDATE workflows
SET enabled = false, updated_at = NOW()
WHERE id = $1;

-- name: ListWorkflowVersions :many
SELECT * FROM workflows
WHERE workflow_id = $1
ORDER BY created_at DESC;

-- name: GetEnabledWorkflowsByEvent :many
SELECT * FROM workflows
WHERE enabled = true
AND definition @> jsonb_build_object('trigger', jsonb_build_object('event', $1));
