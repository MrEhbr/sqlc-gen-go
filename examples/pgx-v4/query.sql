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

-- name: GetUserForUpdate :one
SELECT * FROM users
WHERE id = $1
FOR UPDATE;

-- name: BatchInsertUsers :batchexec
INSERT INTO users (name, email)
VALUES ($1, $2);

-- name: BatchUpdateEmails :batchexec
UPDATE users
SET email = $2
WHERE id = $1;

-- name: BatchGetUsers :batchone
SELECT * FROM users
WHERE id = $1;

-- name: BatchListUsersByEmail :batchmany
SELECT * FROM users
WHERE email LIKE $1;

-- name: BulkInsertUsers :copyfrom
INSERT INTO users (name, email) VALUES ($1, $2);
