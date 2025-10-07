-- name: GetUser :one
SELECT * FROM users
WHERE id = ?;

-- name: ListUsers :many
SELECT * FROM users
ORDER BY created_at DESC;

-- name: CreateUser :one
INSERT INTO users (name, email)
VALUES (?, ?)
RETURNING *;

-- name: DeleteUser :exec
DELETE FROM users
WHERE id = ?;

-- name: UpdateUserEmail :execrows
UPDATE users
SET email = ?
WHERE id = ?;

-- name: UpdateUserName :execresult
UPDATE users
SET name = ?
WHERE id = ?;

-- name: CreateUserGetID :execlastid
INSERT INTO users (name, email)
VALUES (?, ?);
