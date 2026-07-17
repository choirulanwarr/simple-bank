-- name: CreateAuditLog :one
INSERT INTO audit_logs (table_name, record_id, operation, old_values, new_values, changed_by)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: ListAuditLogsByRecord :many
SELECT * FROM audit_logs
WHERE table_name = $1 AND record_id = $2
ORDER BY changed_at DESC;