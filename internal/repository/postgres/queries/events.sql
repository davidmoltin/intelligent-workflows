-- name: CreateEvent :one
INSERT INTO events (
    event_id,
    event_type,
    source,
    payload
) VALUES (
    $1, $2, $3, $4
) RETURNING *;

-- name: GetEvent :one
SELECT * FROM events
WHERE id = $1;

-- name: ListEvents :many
SELECT * FROM events
WHERE ($1::text IS NULL OR event_type = $1)
ORDER BY received_at DESC
LIMIT $2 OFFSET $3;

-- name: CountEvents :one
SELECT COUNT(*) FROM events
WHERE ($1::text IS NULL OR event_type = $1);

-- name: UpdateEventProcessed :exec
UPDATE events
SET
    processed_at = NOW(),
    triggered_workflows = $2
WHERE id = $1;

-- name: CreateApprovalRequest :one
INSERT INTO approval_requests (
    request_id,
    execution_id,
    entity_type,
    entity_id,
    requester_id,
    approver_role,
    status,
    reason,
    expires_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9
) RETURNING *;

-- name: GetApprovalRequest :one
SELECT * FROM approval_requests
WHERE id = $1;

-- name: GetApprovalRequestByRequestID :one
SELECT * FROM approval_requests
WHERE request_id = $1;

-- name: ListApprovalRequests :many
SELECT * FROM approval_requests
WHERE ($1::text IS NULL OR status = $1)
AND ($2::uuid IS NULL OR approver_id = $2)
ORDER BY requested_at DESC
LIMIT $3 OFFSET $4;

-- name: CountApprovalRequests :one
SELECT COUNT(*) FROM approval_requests
WHERE ($1::text IS NULL OR status = $1)
AND ($2::uuid IS NULL OR approver_id = $2);

-- name: UpdateApprovalDecision :exec
UPDATE approval_requests
SET
    status = $2,
    approver_id = $3,
    decision_reason = $4,
    decided_at = NOW()
WHERE id = $1;

-- name: ExpireApprovalRequests :exec
UPDATE approval_requests
SET status = 'expired'
WHERE status = 'pending'
AND expires_at IS NOT NULL
AND expires_at < NOW();
