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
