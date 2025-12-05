-- name: IsBlocked :one
SELECT EXISTS (
    SELECT 1 FROM blocked_ids WHERE id = $1
) AS blocked;

-- name: InsertBlockedID :exec
INSERT INTO blocked_ids(id, updated_at)
VALUES ($1, NOW())
ON CONFLICT (id) DO UPDATE
SET updated_at = NOW();