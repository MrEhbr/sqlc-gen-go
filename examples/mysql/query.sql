-- name: GetUser :one
SELECT * FROM users
WHERE id = ?;

-- name: ListUsers :many
SELECT * FROM users
ORDER BY created_at DESC;

-- name: CreateUser :execresult
INSERT INTO users (name, email)
VALUES (?, ?);

-- name: DeleteUser :exec
DELETE FROM users
WHERE id = ?;

-- name: UpdateUserEmail :execresult
UPDATE users
SET email = ?
WHERE id = ?;

-- name: UpdateUserName :execrows
UPDATE users
SET name = ?
WHERE id = ?;

-- name: CreateUserGetID :execlastid
INSERT INTO users (name, email)
VALUES (?, ?);
