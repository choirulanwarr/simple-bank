-- name: CreateCustomer :one
INSERT INTO customers (name, email, password_hash)
VALUES ($1, $2, $3)
RETURNING *;

-- name: GetCustomer :one
SELECT * FROM customers WHERE id = $1 LIMIT 1;

-- name: GetCustomerByEmail :one
SELECT * FROM customers WHERE email = $1 LIMIT 1;

-- name: ListCustomers :many
SELECT * FROM customers
ORDER BY id
LIMIT $1 OFFSET $2;

-- name: UpdateCustomer :one
UPDATE customers
SET
  name = COALESCE($2, name),
  email = COALESCE($3, email),
  is_active = COALESCE($4, is_active),
  updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: DeleteCustomer :exec
DELETE FROM customers WHERE id = $1;