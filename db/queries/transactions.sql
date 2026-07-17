-- name: CreateTransaction :one
INSERT INTO transactions (
  account_id, type, amount, balance_before, balance_after,
  reference, description
) VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING *;

-- name: ListTransactionsByAccount :many
SELECT * FROM transactions
WHERE account_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;