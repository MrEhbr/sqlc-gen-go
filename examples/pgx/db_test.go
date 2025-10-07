package db_test

import (
	"context"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sqlc-dev/sqlc-gen-go/examples/pgx/db"
)

func setupTestDB(t *testing.T) *pgxpool.Pool {
	t.Helper()

	databaseURL := getEnv("DATABASE_URL", "")
	if databaseURL == "" {
		t.Skip("Skipping test: DATABASE_URL not set")
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatalf("failed to connect to database: %v", err)
	}

	// Create schema
	schema := `
DROP TYPE IF EXISTS user_status CASCADE;
CREATE TYPE user_status AS ENUM ('active', 'inactive', 'banned');

DROP TABLE IF EXISTS users;
CREATE TABLE users (
  id BIGSERIAL PRIMARY KEY,
  name TEXT NOT NULL,
  email TEXT NOT NULL UNIQUE,
  status user_status NOT NULL DEFAULT 'active',
  description TEXT DEFAULT NULL,
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
		user, err := query.Eval(ctx, db.CreateUserParams{
			Name:   "foobar",
			Email:  "foobar@example.com",
			Status: db.UserStatusActive,
		})
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
		created, err := createQuery.Eval(ctx, db.CreateUserParams{
			Name:   "foobaz",
			Email:  "foobaz@example.com",
		Status: db.UserStatusActive,
		})
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
		created, err := createQuery.Eval(ctx, db.CreateUserParams{
			Name:   "barbaz",
			Email:  "barbaz@example.com",
		Status: db.UserStatusActive,
		})
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
		created, err := createQuery.Eval(ctx, db.CreateUserParams{
			Name:   "barfoo",
			Email:  "barfoo@example.com",
		Status: db.UserStatusActive,
		})
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
		created, err := createQuery.Eval(ctx, db.CreateUserParams{
			Name:   "bazbar",
			Email:  "bazbar@example.com",
		Status: db.UserStatusActive,
		})
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

	// Test BatchInsertUsers (:batchexec)
	t.Run("BatchInsertUsers", func(t *testing.T) {
		batchQuery := db.NewBatchInsertUsersQuery(executor)
		params := []db.BatchInsertUsersParams{
			{Name:   "batch1", Email: "batch1@example.com"},
			{Name:   "batch2", Email: "batch2@example.com"},
			{Name:   "batch3", Email: "batch3@example.com"},
		}

		results, err := batchQuery.Eval(ctx, params)
		if err != nil {
			t.Fatalf("BatchInsertUsers failed: %v", err)
		}
		defer results.Close()

		count := 0
		results.Exec(func(i int, err error) {
			if err != nil {
				t.Errorf("batch item %d failed: %v", i, err)
			}
			count++
		})
		if count != 3 {
			t.Errorf("expected 3 batch executions, got %d", count)
		}
	})

	// Test BatchGetUsers (:batchone)
	t.Run("BatchGetUsers", func(t *testing.T) {
		// First create some users to get
		createQuery := db.NewCreateUserQuery(executor)
		user1, _ := createQuery.Eval(ctx, db.CreateUserParams{Name:   "batchget1", Email:  "batchget1@example.com",
			Status: db.UserStatusActive,
		})
		user2, _ := createQuery.Eval(ctx, db.CreateUserParams{Name:   "batchget2", Email:  "batchget2@example.com",
			Status: db.UserStatusActive,
		})

		batchQuery := db.NewBatchGetUsersQuery(executor)
		ids := []int64{user1.ID, user2.ID}

		results, err := batchQuery.Eval(ctx, ids)
		if err != nil {
			t.Fatalf("BatchGetUsers failed: %v", err)
		}
		defer results.Close()

		count := 0
		results.QueryRow(func(i int, user db.User, err error) {
			if err != nil {
				t.Errorf("batch item %d failed: %v", i, err)
			}
			count++
		})
		if count != 2 {
			t.Errorf("expected 2 batch results, got %d", count)
		}
	})

	// Test BatchListUsersByEmail (:batchmany)
	t.Run("BatchListUsersByEmail", func(t *testing.T) {
		batchQuery := db.NewBatchListUsersByEmailQuery(executor)
		emails := []string{"batch%", "foobar%"}

		results, err := batchQuery.Eval(ctx, emails)
		if err != nil {
			t.Fatalf("BatchListUsersByEmail failed: %v", err)
		}
		defer results.Close()

		count := 0
		results.Query(func(i int, users []db.User, err error) {
			if err != nil {
				t.Errorf("batch item %d failed: %v", i, err)
			}
			count++
		})
		if count != 2 {
			t.Errorf("expected 2 batch results, got %d", count)
		}
	})

	// Test BulkInsertUsers (:copyfrom)
	t.Run("BulkInsertUsers", func(t *testing.T) {
		bulkQuery := db.NewBulkInsertUsersQuery(executor)
		rows := []db.BulkInsertUsersParams{
			{Name: "bulk1", Email: "bulk1@example.com"},
			{Name: "bulk2", Email: "bulk2@example.com"},
			{Name: "bulk3", Email: "bulk3@example.com"},
		}

		count, err := bulkQuery.Eval(ctx, rows)
		if err != nil {
			t.Fatalf("BulkInsertUsers failed: %v", err)
		}
		if count != 3 {
			t.Errorf("expected 3 rows inserted, got %d", count)
		}
	})

	// Test WithTx - successful transaction
	t.Run("WithTx_Success", func(t *testing.T) {
		var createdID int64

		err := executor.WithTx(ctx, func(txExecutor db.QueryExecutor) error {
			// Create a user in transaction
			createQuery := db.NewCreateUserQuery(txExecutor)
			user, err := createQuery.Eval(ctx, db.CreateUserParams{
				Name:   "tx_user",
				Email:  "tx_user@example.com",
			Status: db.UserStatusActive,
		})
			if err != nil {
				return err
			}
			createdID = user.ID

			// Update the user's email in the same transaction
			updateQuery := db.NewUpdateUserEmailQuery(txExecutor)
			_, err = updateQuery.Eval(ctx, user.ID, "tx_user_updated@example.com")
			return err
		})

		if err != nil {
			t.Fatalf("WithTx failed: %v", err)
		}

		// Verify the transaction was committed
		getQuery := db.NewGetUserQuery(executor)
		user, err := getQuery.Eval(ctx, createdID)
		if err != nil {
			t.Fatalf("GetUser after transaction failed: %v", err)
		}
		if user.Email != "tx_user_updated@example.com" {
			t.Errorf("expected email tx_user_updated@example.com, got %s", user.Email)
		}
	})

	// Test WithTx - rollback on error
	t.Run("WithTx_Rollback", func(t *testing.T) {
		var createdID int64

		err := executor.WithTx(ctx, func(txExecutor db.QueryExecutor) error {
			// Create a user in transaction
			createQuery := db.NewCreateUserQuery(txExecutor)
			user, err := createQuery.Eval(ctx, db.CreateUserParams{
				Name:   "tx_rollback",
				Email:  "tx_rollback@example.com",
			Status: db.UserStatusActive,
		})
			if err != nil {
				return err
			}
			createdID = user.ID

			// Return an error to trigger rollback
			return context.Canceled
		})

		if err == nil {
			t.Fatal("expected error from WithTx, got nil")
		}

		// Verify the transaction was rolled back - user should not exist
		getQuery := db.NewGetUserQuery(executor)
		_, err = getQuery.Eval(ctx, createdID)
		if err == nil {
			t.Error("expected error for rolled back user creation, got nil")
		}
	})

	// Test WithTx - nested queries
	t.Run("WithTx_NestedQueries", func(t *testing.T) {
		err := executor.WithTx(ctx, func(txExecutor db.QueryExecutor) error {
			// Create multiple users in a transaction
			createQuery := db.NewCreateUserQuery(txExecutor)

			user1, err := createQuery.Eval(ctx, db.CreateUserParams{
				Name:   "tx_batch1",
				Email:  "tx_batch1@example.com",
			Status: db.UserStatusActive,
		})
			if err != nil {
				return err
			}

			user2, err := createQuery.Eval(ctx, db.CreateUserParams{
				Name:   "tx_batch2",
				Email:  "tx_batch2@example.com",
			Status: db.UserStatusActive,
		})
			if err != nil {
				return err
			}

			// Verify we can read them within the transaction
			getQuery := db.NewGetUserQuery(txExecutor)
			retrieved, err := getQuery.Eval(ctx, user1.ID)
			if err != nil {
				return err
			}
			if retrieved.Name != "tx_batch1" {
				t.Errorf("expected name tx_batch1, got %s", retrieved.Name)
			}

			// Update one of them
			updateQuery := db.NewUpdateUserEmailQuery(txExecutor)
			_, err = updateQuery.Eval(ctx, user2.ID, "tx_batch2_modified@example.com")
			return err
		})

		if err != nil {
			t.Fatalf("WithTx nested queries failed: %v", err)
		}
	})
}
