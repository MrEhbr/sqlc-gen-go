package db_test

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/sqlc-dev/sqlc-gen-go/examples/pgx-v4/db"
)

func setupTestDB(t *testing.T) *pgxpool.Pool {
	t.Helper()

	databaseURL := getEnv("DATABASE_URL", "")
	if databaseURL == "" {
		t.Skip("Skipping test: DATABASE_URL not set")
	}

	ctx := context.Background()
	pool, err := pgxpool.Connect(ctx, databaseURL)
	if err != nil {
		t.Fatalf("failed to connect to database: %v", err)
	}

	// Create schema
	schema := `
DROP TABLE IF EXISTS users;
CREATE TABLE users (
  id BIGSERIAL PRIMARY KEY,
  name TEXT NOT NULL,
  email TEXT NOT NULL UNIQUE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
`
	if _, err := pool.Exec(ctx, schema); err != nil {
		t.Fatalf("failed to create schema: %v", err)
	}

	return pool
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func TestQueries(t *testing.T) {
	ctx := context.Background()
	pool := setupTestDB(t)
	defer pool.Close()

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
}
