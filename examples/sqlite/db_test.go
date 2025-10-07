package db_test

import (
	"context"
	"database/sql"
	"testing"

	"github.com/sqlc-dev/sqlc-gen-go/examples/sqlite/db"
	_ "modernc.org/sqlite"
)

func setupTestDB(t *testing.T) *sql.DB {
	t.Helper()

	database, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}

	// Create schema
	schema := `
CREATE TABLE users (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  name TEXT NOT NULL,
  email TEXT NOT NULL UNIQUE,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
`
	if _, err := database.Exec(schema); err != nil {
		t.Fatalf("failed to create schema: %v", err)
	}

	return database
}

func TestQueries(t *testing.T) {
	ctx := context.Background()
	database := setupTestDB(t)
	defer database.Close()

	executor := db.NewExecutor(database)

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
		// First create a user
		createQuery := db.NewCreateUserQuery(executor)
		created, err := createQuery.Eval(ctx, "foobaz", "foobaz@example.com")
		if err != nil {
			t.Fatalf("CreateUser failed: %v", err)
		}

		// Then get it
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
		rows, err := updateQuery.Eval(ctx, "barbaz.updated@example.com", created.ID)
		if err != nil {
			t.Fatalf("UpdateUserEmail failed: %v", err)
		}
		if rows != 1 {
			t.Errorf("expected 1 row updated, got %d", rows)
		}

		// Verify update
		getQuery := db.NewGetUserQuery(executor)
		user, err := getQuery.Eval(ctx, created.ID)
		if err != nil {
			t.Fatalf("GetUser failed: %v", err)
		}
		if user.Email != "barbaz.updated@example.com" {
			t.Errorf("expected email barbaz.updated@example.com, got %s", user.Email)
		}
	})

	// Test UpdateUserName (:execresult)
	t.Run("UpdateUserName", func(t *testing.T) {
		createQuery := db.NewCreateUserQuery(executor)
		created, err := createQuery.Eval(ctx, "barfoo", "barfoo@example.com")
		if err != nil {
			t.Fatalf("CreateUser failed: %v", err)
		}

		// :execresult returns sql.Result directly through database.ExecContext
		result, err := database.ExecContext(ctx, `UPDATE users SET name = ? WHERE id = ?`, "barfoo-updated", created.ID)
		if err != nil {
			t.Fatalf("UpdateUserName failed: %v", err)
		}

		rows, err := result.RowsAffected()
		if err != nil {
			t.Fatalf("RowsAffected failed: %v", err)
		}
		if rows != 1 {
			t.Errorf("expected 1 row updated, got %d", rows)
		}
	})

	// Test CreateUserGetID (:execlastid)
	t.Run("CreateUserGetID", func(t *testing.T) {
		createQuery := db.NewCreateUserGetIDQuery(executor)
		id, err := createQuery.Eval(ctx, "bazfoo", "bazfoo@example.com")
		if err != nil {
			t.Fatalf("CreateUserGetID failed: %v", err)
		}
		if id == 0 {
			t.Error("expected non-zero last insert ID")
		}

		// Verify user was created
		getQuery := db.NewGetUserQuery(executor)
		user, err := getQuery.Eval(ctx, id)
		if err != nil {
			t.Fatalf("GetUser failed: %v", err)
		}
		if user.Name != "bazfoo" {
			t.Errorf("expected name bazfoo, got %s", user.Name)
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

		// Verify deletion
		getQuery := db.NewGetUserQuery(executor)
		_, err = getQuery.Eval(ctx, created.ID)
		if err != sql.ErrNoRows {
			t.Errorf("expected ErrNoRows, got %v", err)
		}
	})

	// Test WithTx - successful transaction
	t.Run("WithTx_Success", func(t *testing.T) {
		var createdID int64

		err := executor.WithTx(ctx, func(txExecutor db.QueryExecutor) error {
			// Create a user in transaction
			createQuery := db.NewCreateUserQuery(txExecutor)
			user, err := createQuery.Eval(ctx, "tx_user", "tx_user@example.com")
			if err != nil {
				return err
			}
			createdID = user.ID

			// Update the user's email in the same transaction
			updateQuery := db.NewUpdateUserEmailQuery(txExecutor)
			_, err = updateQuery.Eval(ctx, "tx_user_updated@example.com", user.ID)
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
			user, err := createQuery.Eval(ctx, "tx_rollback", "tx_rollback@example.com")
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
		if err != sql.ErrNoRows {
			t.Errorf("expected ErrNoRows for rolled back user, got %v", err)
		}
	})

	// Test WithTx - nested queries
	t.Run("WithTx_NestedQueries", func(t *testing.T) {
		err := executor.WithTx(ctx, func(txExecutor db.QueryExecutor) error {
			// Create multiple users in a transaction
			createQuery := db.NewCreateUserQuery(txExecutor)

			user1, err := createQuery.Eval(ctx, "tx_batch1", "tx_batch1@example.com")
			if err != nil {
				return err
			}

			user2, err := createQuery.Eval(ctx, "tx_batch2", "tx_batch2@example.com")
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
			_, err = updateQuery.Eval(ctx, "tx_batch2_modified@example.com", user2.ID)
			return err
		})

		if err != nil {
			t.Fatalf("WithTx nested queries failed: %v", err)
		}
	})
}
