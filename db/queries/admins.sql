-- name: CreateAdmin :one
INSERT INTO admins (name, email, password_hash, role)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: GetAdmin :one
SELECT * FROM admins WHERE id = $1 LIMIT 1;

-- name: GetAdminByEmail :one
SELECT * FROM admins WHERE email = $1 LIMIT 1;

-- name: ListAdmins :many
SELECT * FROM admins
ORDER BY id
LIMIT $1 OFFSET $2;
