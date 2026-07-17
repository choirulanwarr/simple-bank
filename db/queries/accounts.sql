-- name: CreateAccount :one
INSERT INTO accounts (customer_id, account_number, currency)
VALUES ($1, $2, $3)
RETURNING *;

-- name: GetAccount :one
SELECT * FROM accounts WHERE id = $1 LIMIT 1;

-- name: GetAccountForUpdate :one
SELECT * FROM accounts
WHERE id = $1
LIMIT 1
FOR NO KEY UPDATE;

-- name: ListAccountsByCustomer :many
SELECT * FROM accounts
WHERE customer_id = $1
ORDER BY id;

-- name: AddAccountBalance :one
UPDATE accounts
SET
  balance = balance + sqlc.arg(amount),
  updated_at = NOW()
WHERE id = sqlc.arg(id)
RETURNING *;

-- name: UpdateAccountStatus :one
UPDATE accounts
SET
  status = $2,
  updated_at = NOW()
WHERE id = $1
RETURNING *;