-- name: CreateRule :one
INSERT INTO rules (
    rule_id,
    name,
    description,
    rule_type,
    definition,
    enabled
) VALUES (
    $1, $2, $3, $4, $5, $6
) RETURNING *;

-- name: GetRule :one
SELECT * FROM rules
WHERE id = $1;

-- name: GetRuleByRuleID :one
SELECT * FROM rules
WHERE rule_id = $1;

-- name: ListRules :many
SELECT * FROM rules
WHERE ($1::boolean IS NULL OR enabled = $1)
AND ($2::text IS NULL OR rule_type = $2)
ORDER BY created_at DESC
LIMIT $3 OFFSET $4;

-- name: CountRules :one
SELECT COUNT(*) FROM rules
WHERE ($1::boolean IS NULL OR enabled = $1)
AND ($2::text IS NULL OR rule_type = $2);

-- name: UpdateRule :one
UPDATE rules
SET
    name = COALESCE($2, name),
    description = COALESCE($3, description),
    definition = COALESCE($4, definition),
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: DeleteRule :exec
DELETE FROM rules
WHERE id = $1;

-- name: EnableRule :exec
UPDATE rules
SET enabled = true, updated_at = NOW()
WHERE id = $1;

-- name: DisableRule :exec
UPDATE rules
SET enabled = false, updated_at = NOW()
WHERE id = $1;
