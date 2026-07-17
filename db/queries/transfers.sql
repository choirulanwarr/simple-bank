-- name: CreateTransfer :one
INSERT INTO transfers (
  from_account_id, to_account_id, amount, fee,
  status, reference, description
) VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING *;

-- name: GetTransfer :one
SELECT * FROM transfers WHERE id = $1 LIMIT 1;

-- name: UpdateTransferStatus :one
UPDATE transfers
SET
  status = $2,
  completed_at = CASE WHEN $2 = 'completed' THEN NOW() ELSE completed_at END
WHERE id = $1
RETURNING *;

-- name: ListTransfers :many
SELECT * FROM transfers
WHERE from_account_id = $1 OR to_account_id = $2
ORDER BY created_at DESC
LIMIT $3 OFFSET $4;