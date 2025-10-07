package db_test

import (
	"context"
	"database/sql"
	"os"
	"testing"

	_ "github.com/go-sql-driver/mysql"
	"github.com/sqlc-dev/sqlc-gen-go/examples/mysql/db"
)

func setupTestDB(t *testing.T) *sql.DB {
	t.Helper()

	databaseURL := getEnv("DATABASE_URL", "")
	if databaseURL == "" {
		t.Skip("Skipping test: DATABASE_URL not set")
	}

	database, err := sql.Open("mysql", databaseURL)
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}

	// Create schema
	_, err = database.Exec(`DROP TABLE IF EXISTS users`)
	if err != nil {
		t.Fatalf("failed to drop table: %v", err)
	}

	_, err = database.Exec(`
CREATE TABLE users (
  id BIGINT PRIMARY KEY AUTO_INCREMENT,
  name VARCHAR(255) NOT NULL,
  email VARCHAR(255) NOT NULL UNIQUE,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
)`)
	if err != nil {
		t.Fatalf("failed to create table: %v", err)
	}

	return database
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func TestQueries(t *testing.T) {
	ctx := context.Background()
	database := setupTestDB(t)
	defer database.Close()

	executor := db.NewExecutor(database)

	// Test CreateUser (:execresult)
	t.Run("CreateUser", func(t *testing.T) {
		// :execresult returns sql.Result directly through database.ExecContext
		result, err := database.ExecContext(ctx, `INSERT INTO users (name, email) VALUES (?, ?)`, "foobar", "foobar@example.com")
		if err != nil {
			t.Fatalf("CreateUser failed: %v", err)
		}

		id, err := result.LastInsertId()
		if err != nil {
			t.Fatalf("LastInsertId failed: %v", err)
		}
		if id == 0 {
			t.Error("expected non-zero ID")
		}
	})

	// Test GetUser (:one)
	t.Run("GetUser", func(t *testing.T) {
		// First create a user
		result, _ := database.ExecContext(ctx, `INSERT INTO users (name, email) VALUES (?, ?)`, "foobaz", "foobaz@example.com")
		id, _ := result.LastInsertId()

		getQuery := db.NewGetUserQuery(executor)
		user, err := getQuery.Eval(ctx, id)
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

	// Test UpdateUserEmail (:execresult)
	t.Run("UpdateUserEmail", func(t *testing.T) {
		result, _ := database.ExecContext(ctx, `INSERT INTO users (name, email) VALUES (?, ?)`, "barbaz", "barbaz@example.com")
		id, _ := result.LastInsertId()

		// :execresult returns sql.Result directly through database.ExecContext
		updateResult, err := database.ExecContext(ctx, `UPDATE users SET email = ? WHERE id = ?`, "barbaz.updated@example.com", id)
		if err != nil {
			t.Fatalf("UpdateUserEmail failed: %v", err)
		}

		rows, err := updateResult.RowsAffected()
		if err != nil {
			t.Fatalf("RowsAffected failed: %v", err)
		}
		if rows != 1 {
			t.Errorf("expected 1 row updated, got %d", rows)
		}

		getQuery := db.NewGetUserQuery(executor)
		user, err := getQuery.Eval(ctx, id)
		if err != nil {
			t.Fatalf("GetUser failed: %v", err)
		}
		if user.Email != "barbaz.updated@example.com" {
			t.Errorf("expected email barbaz.updated@example.com, got %s", user.Email)
		}
	})

	// Test UpdateUserName (:execrows)
	t.Run("UpdateUserName", func(t *testing.T) {
		result, _ := database.ExecContext(ctx, `INSERT INTO users (name, email) VALUES (?, ?)`, "barfoo", "barfoo@example.com")
		id, _ := result.LastInsertId()

		updateQuery := db.NewUpdateUserNameQuery(executor)
		rows, err := updateQuery.Eval(ctx, "barfoo-updated", id)
		if err != nil {
			t.Fatalf("UpdateUserName failed: %v", err)
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
		result, _ := database.ExecContext(ctx, `INSERT INTO users (name, email) VALUES (?, ?)`, "bazbar", "bazbar@example.com")
		id, _ := result.LastInsertId()

		deleteQuery := db.NewDeleteUserQuery(executor)
		err := deleteQuery.Eval(ctx, id)
		if err != nil {
			t.Fatalf("DeleteUser failed: %v", err)
		}

		getQuery := db.NewGetUserQuery(executor)
		_, err = getQuery.Eval(ctx, id)
		if err != sql.ErrNoRows {
			t.Errorf("expected ErrNoRows, got %v", err)
		}
	})

	// Test WithTx - successful transaction
	t.Run("WithTx_Success", func(t *testing.T) {
		var createdID int64

		err := executor.WithTx(ctx, func(txExecutor db.QueryExecutor) error {
			// Create two users in the same transaction
			createQuery := db.NewCreateUserGetIDQuery(txExecutor)
			id, err := createQuery.Eval(ctx, "tx_user", "tx_user@example.com")
			if err != nil {
				return err
			}
			createdID = id

			// Create another user to verify transaction atomicity
			_, err = createQuery.Eval(ctx, "tx_user2", "tx_user2@example.com")
			return err
		})
		if err != nil {
			t.Fatalf("WithTx failed: %v", err)
		}

		// Verify the transaction was committed - both users should exist
		getQuery := db.NewGetUserQuery(executor)
		user, err := getQuery.Eval(ctx, createdID)
		if err != nil {
			t.Fatalf("GetUser after transaction failed: %v", err)
		}
		if user.Name != "tx_user" {
			t.Errorf("expected name tx_user, got %s", user.Name)
		}
	})

	// Test WithTx - rollback on error
	t.Run("WithTx_Rollback", func(t *testing.T) {
		var createdID int64

		err := executor.WithTx(ctx, func(txExecutor db.QueryExecutor) error {
			// Create a user in transaction
			createQuery := db.NewCreateUserGetIDQuery(txExecutor)
			id, err := createQuery.Eval(ctx, "tx_rollback", "tx_rollback@example.com")
			if err != nil {
				return err
			}
			createdID = id

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
			createQuery := db.NewCreateUserGetIDQuery(txExecutor)

			id1, err := createQuery.Eval(ctx, "tx_batch1", "tx_batch1@example.com")
			if err != nil {
				return err
			}

			id2, err := createQuery.Eval(ctx, "tx_batch2", "tx_batch2@example.com")
			if err != nil {
				return err
			}

			// Verify we can read them within the transaction
			getQuery := db.NewGetUserQuery(txExecutor)
			user1, err := getQuery.Eval(ctx, id1)
			if err != nil {
				return err
			}
			if user1.Name != "tx_batch1" {
				t.Errorf("expected name tx_batch1, got %s", user1.Name)
			}

			// Verify second user as well
			user2, err := getQuery.Eval(ctx, id2)
			if err != nil {
				return err
			}
			if user2.Name != "tx_batch2" {
				t.Errorf("expected name tx_batch2, got %s", user2.Name)
			}

			return nil
		})
		if err != nil {
			t.Fatalf("WithTx nested queries failed: %v", err)
		}
	})
}
