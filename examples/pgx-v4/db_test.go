package db_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/sqlc-dev/sqlc-gen-go/examples/pgx-v4/db"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

func setupTestDB(t *testing.T) (*pgxpool.Pool, func()) {
	t.Helper()

	ctx := context.Background()

	pgContainer, err := postgres.Run(ctx,
		"postgres:18-alpine",
		postgres.WithDatabase("testdb"),
		postgres.WithUsername("postgres"),
		postgres.WithPassword("password"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(60*time.Second)),
	)
	if err != nil {
		t.Fatalf("failed to start postgres container: %v", err)
	}

	connStr, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("failed to get connection string: %v", err)
	}

	var pool *pgxpool.Pool
	for range 10 {
		pool, err = pgxpool.Connect(ctx, connStr)
		if err == nil {
			if err = pool.Ping(ctx); err == nil {
				break
			}
			pool.Close()
		}
		time.Sleep(time.Second)
	}
	if err != nil {
		t.Fatalf("failed to connect to database after retries: %v", err)
	}

	// Create schema
	schema := `
DROP TABLE IF EXISTS posts;
DROP TABLE IF EXISTS users;
CREATE TABLE users (
  id BIGSERIAL PRIMARY KEY,
  name TEXT NOT NULL,
  email TEXT NOT NULL UNIQUE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE posts (
  id         BIGSERIAL PRIMARY KEY,
  author_id  BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  title      TEXT NOT NULL,
  body       TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
`
	if _, err := pool.Exec(ctx, schema); err != nil {
		t.Fatalf("failed to create schema: %v", err)
	}

	cleanup := func() {
		pool.Close()
		if err := pgContainer.Terminate(ctx); err != nil {
			t.Logf("failed to terminate container: %v", err)
		}
	}

	return pool, cleanup
}

func TestQueries(t *testing.T) {
	ctx := context.Background()
	pool, cleanup := setupTestDB(t)
	defer cleanup()

	executor := db.NewExecutor(pool)

	// Test CreateUser (:one with RETURNING)
	t.Run("CreateUser", func(t *testing.T) {
		query := db.NewCreateUserQuery(executor)
		user, err := query.Eval(ctx, "foobar", "foobar@example.com")
		if err != nil {
			t.Fatalf("CreateUser failed: %v", err)
		}
		if user.Name != "foobar" {
			t.Errorf("expected name foobar, got %s", user.Name)
		}
		if user.Email != "foobar@example.com" {
			t.Errorf("expected email foobar@example.com, got %s", user.Email)
		}
		if user.ID == 0 {
			t.Error("expected non-zero ID")
		}
	})

	// Test GetUser (:one)
	t.Run("GetUser", func(t *testing.T) {
		createQuery := db.NewCreateUserQuery(executor)
		created, err := createQuery.Eval(ctx, "foobaz", "foobaz@example.com")
		if err != nil {
			t.Fatalf("CreateUser failed: %v", err)
		}

		getQuery := db.NewGetUserQuery(executor)
		user, err := getQuery.Eval(ctx, created.ID)
		if err != nil {
			t.Fatalf("GetUser failed: %v", err)
		}
		if user.Name != "foobaz" {
			t.Errorf("expected name foobaz, got %s", user.Name)
		}
	})

	// Test ListUsers (:many)
	t.Run("ListUsers", func(t *testing.T) {
		query := db.NewListUsersQuery(executor)
		users, err := query.Eval(ctx)
		if err != nil {
			t.Fatalf("ListUsers failed: %v", err)
		}
		if len(users) < 2 {
			t.Errorf("expected at least 2 users, got %d", len(users))
		}
	})

	// Test UpdateUserEmail (:execrows)
	t.Run("UpdateUserEmail", func(t *testing.T) {
		createQuery := db.NewCreateUserQuery(executor)
		created, err := createQuery.Eval(ctx, "barbaz", "barbaz@example.com")
		if err != nil {
			t.Fatalf("CreateUser failed: %v", err)
		}

		updateQuery := db.NewUpdateUserEmailQuery(executor)
		rows, err := updateQuery.Eval(ctx, created.ID, "barbaz.updated@example.com")
		if err != nil {
			t.Fatalf("UpdateUserEmail failed: %v", err)
		}
		if rows != 1 {
			t.Errorf("expected 1 row updated, got %d", rows)
		}

		getQuery := db.NewGetUserQuery(executor)
		user, err := getQuery.Eval(ctx, created.ID)
		if err != nil {
			t.Fatalf("GetUser failed: %v", err)
		}
		if user.Email != "barbaz.updated@example.com" {
			t.Errorf("expected email barbaz.updated@example.com, got %s", user.Email)
		}
	})

	// Test GetUserForUpdate (:one with FOR UPDATE)
	t.Run("GetUserForUpdate", func(t *testing.T) {
		createQuery := db.NewCreateUserQuery(executor)
		created, err := createQuery.Eval(ctx, "barfoo", "barfoo@example.com")
		if err != nil {
			t.Fatalf("CreateUser failed: %v", err)
		}

		query := db.NewGetUserForUpdateQuery(executor)
		user, err := query.Eval(ctx, created.ID)
		if err != nil {
			t.Fatalf("GetUserForUpdate failed: %v", err)
		}
		if user.Name != "barfoo" {
			t.Errorf("expected name barfoo, got %s", user.Name)
		}
	})

	// Test DeleteUser (:exec)
	t.Run("DeleteUser", func(t *testing.T) {
		createQuery := db.NewCreateUserQuery(executor)
		created, err := createQuery.Eval(ctx, "bazbar", "bazbar@example.com")
		if err != nil {
			t.Fatalf("CreateUser failed: %v", err)
		}

		deleteQuery := db.NewDeleteUserQuery(executor)
		err = deleteQuery.Eval(ctx, created.ID)
		if err != nil {
			t.Fatalf("DeleteUser failed: %v", err)
		}

		getQuery := db.NewGetUserQuery(executor)
		_, err = getQuery.Eval(ctx, created.ID)
		if err == nil {
			t.Error("expected error for deleted user, got nil")
		}
	})

	// Test WithTx_Success
	t.Run("WithTx_Success", func(t *testing.T) {
		err := executor.WithTx(ctx, func(tx db.QueryExecutor) error {
			createQuery := db.NewCreateUserQuery(tx)
			_, err := createQuery.Eval(ctx, "txuser", "txuser@example.com")
			return err
		})
		if err != nil {
			t.Fatalf("WithTx failed: %v", err)
		}
	})

	// Test WithTx_Rollback
	t.Run("WithTx_Rollback", func(t *testing.T) {
		initialCount := 0
		listQuery := db.NewListUsersQuery(executor)
		users, _ := listQuery.Eval(ctx)
		initialCount = len(users)

		err := executor.WithTx(ctx, func(tx db.QueryExecutor) error {
			createQuery := db.NewCreateUserQuery(tx)
			_, err := createQuery.Eval(ctx, "rollbackuser", "rollbackuser@example.com")
			if err != nil {
				return err
			}
			return fmt.Errorf("intentional rollback")
		})
		if err == nil {
			t.Fatal("expected error for rollback, got nil")
		}

		users, _ = listQuery.Eval(ctx)
		if len(users) != initialCount {
			t.Errorf("expected user count to remain %d after rollback, got %d", initialCount, len(users))
		}
	})

	t.Run("GetPostWithAuthor", func(t *testing.T) {
		author, err := db.NewCreateUserQuery(executor).Eval(ctx, "embed_author", "embed_author@example.com")
		if err != nil {
			t.Fatalf("CreateUser failed: %v", err)
		}
		post, err := db.NewCreatePostQuery(executor).Eval(ctx, db.CreatePostParams{AuthorID: author.ID, Title: "embed_title", Body: "embed_body"})
		if err != nil {
			t.Fatalf("CreatePost failed: %v", err)
		}
		row, err := db.NewGetPostWithAuthorQuery(executor).Eval(ctx, post.ID)
		if err != nil {
			t.Fatalf("GetPostWithAuthor failed: %v", err)
		}
		if row.Post.ID != post.ID || row.Post.Title != "embed_title" {
			t.Errorf("post not hydrated: %+v", row.Post)
		}
		if row.User.ID != author.ID || row.User.Name != "embed_author" {
			t.Errorf("embedded user not hydrated: %+v", row.User)
		}
	})

	t.Run("ListPostsWithAuthor", func(t *testing.T) {
		author, err := db.NewCreateUserQuery(executor).Eval(ctx, "embed_lister", "embed_lister@example.com")
		if err != nil {
			t.Fatalf("CreateUser failed: %v", err)
		}
		for i, title := range []string{"a", "b"} {
			if _, err := db.NewCreatePostQuery(executor).Eval(ctx, db.CreatePostParams{AuthorID: author.ID, Title: title, Body: "body"}); err != nil {
				t.Fatalf("CreatePost #%d failed: %v", i, err)
			}
		}
		rows, err := db.NewListPostsWithAuthorQuery(executor).Eval(ctx)
		if err != nil {
			t.Fatalf("ListPostsWithAuthor failed: %v", err)
		}
		if len(rows) < 2 {
			t.Fatalf("expected at least 2 rows, got %d", len(rows))
		}
		for _, row := range rows {
			if row.Post.ID == 0 || row.User.ID == 0 {
				t.Errorf("row not hydrated: %+v", row)
			}
			if row.Post.AuthorID != row.User.ID {
				t.Errorf("author_id %d does not match embedded user.id %d", row.Post.AuthorID, row.User.ID)
			}
		}
	})
}
