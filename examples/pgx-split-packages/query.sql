-- name: GetAccount :one
SELECT * FROM accounts
WHERE id = $1;

-- name: GetAccountByUsername :one
SELECT * FROM accounts
WHERE username = $1;

-- name: ListAccounts :many
SELECT * FROM accounts
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

-- name: ListAccountsByRole :many
SELECT * FROM accounts
WHERE role = $1
ORDER BY created_at DESC;

-- name: CreateAccount :one
INSERT INTO accounts (username, email, role, status)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: UpdateAccountStatus :execrows
UPDATE accounts
SET status = $2, updated_at = NOW()
WHERE id = $1;

-- name: DeleteAccount :exec
DELETE FROM accounts
WHERE id = $1;

-- name: GetPost :one
SELECT p.*, a.username, a.role
FROM posts p
JOIN accounts a ON p.account_id = a.id
WHERE p.id = $1;

-- name: ListPostsByAccount :many
SELECT * FROM posts
WHERE account_id = $1
ORDER BY created_at DESC;

-- name: CreatePost :one
INSERT INTO posts (account_id, title, content, published)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: PublishPost :execrows
UPDATE posts
SET published = true
WHERE id = $1 AND published = false;
