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

-- name: CountUsers :one
SELECT COUNT(*) FROM users;

-- name: CreatePost :one
INSERT INTO posts (author_id, title, body)
VALUES ($1, $2, $3)
RETURNING *;

-- name: GetPostWithAuthor :one
SELECT sqlc.embed(posts), sqlc.embed(users)
FROM posts
JOIN users ON users.id = posts.author_id
WHERE posts.id = $1;

-- name: ListPostsWithAuthor :many
SELECT sqlc.embed(posts), sqlc.embed(users)
FROM posts
JOIN users ON users.id = posts.author_id
ORDER BY posts.created_at DESC;
