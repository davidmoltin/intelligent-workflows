-- name: CreateRule :one
INSERT INTO rules (
    organization_id,
    rule_id,
    name,
    description,
    rule_type,
    definition,
    enabled
) VALUES (
    $1, $2, $3, $4, $5, $6, $7
) RETURNING *;

-- name: GetRule :one
SELECT * FROM rules
WHERE id = $1 AND organization_id = $2;

-- name: GetRuleByRuleID :one
SELECT * FROM rules
WHERE rule_id = $1 AND organization_id = $2;

-- name: ListRules :many
SELECT * FROM rules
WHERE organization_id = $1
AND ($2::boolean IS NULL OR enabled = $2)
AND ($3::text IS NULL OR rule_type = $3)
ORDER BY created_at DESC
LIMIT $4 OFFSET $5;

-- name: CountRules :one
SELECT COUNT(*) FROM rules
WHERE organization_id = $1
AND ($2::boolean IS NULL OR enabled = $2)
AND ($3::text IS NULL OR rule_type = $3);

-- name: UpdateRule :one
UPDATE rules
SET
    name = COALESCE($3, name),
    description = COALESCE($4, description),
    definition = COALESCE($5, definition),
    updated_at = NOW()
WHERE id = $1 AND organization_id = $2
RETURNING *;

-- name: DeleteRule :exec
DELETE FROM rules
WHERE id = $1 AND organization_id = $2;

-- name: EnableRule :exec
UPDATE rules
SET enabled = true, updated_at = NOW()
WHERE id = $1 AND organization_id = $2;

-- name: DisableRule :exec
UPDATE rules
SET enabled = false, updated_at = NOW()
WHERE id = $1 AND organization_id = $2;
