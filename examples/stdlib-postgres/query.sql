-- name: GetUser :one
SELECT * FROM users
WHERE id = $1;

-- name: ListUsers :many
SELECT * FROM users
ORDER BY created_at DESC;

-- name: CreateUser :one
INSERT INTO users (name, email)
VALUES ($1, $2)
RETURNING *;

-- name: DeleteUser :exec
DELETE FROM users
WHERE id = $1;

-- name: UpdateUserEmail :execrows
UPDATE users
SET email = $2
WHERE id = $1;

-- name: UpdateUserName :execresult
UPDATE users
SET name = $2
WHERE id = $1;

-- Note: :execlastid not used because PostgreSQL doesn't support LastInsertId()
-- Use :one with RETURNING instead

-- name: GetUserForUpdate :one
SELECT * FROM users
WHERE id = $1
FOR UPDATE;

