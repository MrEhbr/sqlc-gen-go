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

-- name: CountUsers :one
SELECT COUNT(*) FROM users;

-- name: CreatePost :execlastid
INSERT INTO posts (author_id, title, body)
VALUES (?, ?, ?);

-- name: GetPostWithAuthor :one
SELECT sqlc.embed(posts), sqlc.embed(users)
FROM posts
JOIN users ON users.id = posts.author_id
WHERE posts.id = ?;

-- name: ListPostsWithAuthor :many
SELECT sqlc.embed(posts), sqlc.embed(users)
FROM posts
JOIN users ON users.id = posts.author_id
ORDER BY posts.created_at DESC;

-- name: BulkInsertUsers :copyfrom
INSERT INTO users (name, email) VALUES (?, ?);

-- name: ListUsersByIDs :many
SELECT * FROM users WHERE id IN (sqlc.slice('ids')) ORDER BY id;

-- name: CountUsersByIDsExceptName :one
SELECT COUNT(*) FROM users WHERE id IN (sqlc.slice('ids')) AND name <> sqlc.arg('name');

-- name: CountUsersByNameOrEmail :one
SELECT COUNT(*) FROM users WHERE name = sqlc.arg('value') OR email = sqlc.arg('value');
