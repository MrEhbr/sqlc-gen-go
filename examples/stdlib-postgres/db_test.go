package db_test

import (
	"context"
	"database/sql"
	"os"
	"testing"
	"time"

	_ "github.com/lib/pq"
	"github.com/sqlc-dev/sqlc-gen-go/examples/stdlib-postgres/db"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

func setupTestDB(t *testing.T) (*sql.DB, func()) {
	t.Helper()

	ctx := context.Background()

	// Ensure DOCKER_HOST is set for non-standard Docker setups (Colima, Podman, etc)
	if os.Getenv("DOCKER_HOST") == "" {
		// Try Colima socket
		if _, err := os.Stat(os.Getenv("HOME") + "/.colima/default/docker.sock"); err == nil {
			os.Setenv("DOCKER_HOST", "unix://"+os.Getenv("HOME")+"/.colima/default/docker.sock")
		}
	}

	// Disable Ryuk for local development (works better with Colima/Podman)
	if os.Getenv("TESTCONTAINERS_RYUK_DISABLED") == "" {
		os.Setenv("TESTCONTAINERS_RYUK_DISABLED", "true")
	}

	pgContainer, err := postgres.Run(ctx,
		"postgres:16-alpine",
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

	// Retry connection a few times (container might not be fully ready)
	var database *sql.DB
	for i := 0; i < 10; i++ {
		database, err = sql.Open("postgres", connStr)
		if err == nil {
			// Try a ping to ensure it's really ready
			if err = database.Ping(); err == nil {
				break
			}
			database.Close()
		}
		time.Sleep(time.Second)
	}
	if err != nil {
		t.Fatalf("failed to connect to database after retries: %v", err)
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
	if _, err := database.Exec(schema); err != nil {
		t.Fatalf("failed to create schema: %v", err)
	}

	cleanup := func() {
		database.Close()
		if err := pgContainer.Terminate(ctx); err != nil {
			t.Logf("failed to terminate container: %v", err)
		}
	}

	return database, cleanup
}

func TestQueries(t *testing.T) {
	ctx := context.Background()
	database, cleanup := setupTestDB(t)
	defer cleanup()

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

	// Test UpdateUserName (:execresult)
	t.Run("UpdateUserName", func(t *testing.T) {
		createQuery := db.NewCreateUserQuery(executor)
		created, err := createQuery.Eval(ctx, "barfoo", "barfoo@example.com")
		if err != nil {
			t.Fatalf("CreateUser failed: %v", err)
		}

		// :execresult returns sql.Result directly through database.ExecContext
		result, err := database.ExecContext(ctx, `UPDATE users SET name = $1 WHERE id = $2`, "barfoo-updated", created.ID)
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

	// Note: :execlastid is not tested because PostgreSQL doesn't support LastInsertId()
	// Use :one with RETURNING instead (see CreateUser test)

	// Test GetUserForUpdate (:one with FOR UPDATE)
	t.Run("GetUserForUpdate", func(t *testing.T) {
		createQuery := db.NewCreateUserQuery(executor)
		created, err := createQuery.Eval(ctx, "bazbar", "bazbar@example.com")
		if err != nil {
			t.Fatalf("CreateUser failed: %v", err)
		}

		query := db.NewGetUserForUpdateQuery(executor)
		user, err := query.Eval(ctx, created.ID)
		if err != nil {
			t.Fatalf("GetUserForUpdate failed: %v", err)
		}
		if user.Name != "bazbar" {
			t.Errorf("expected name bazbar, got %s", user.Name)
		}
	})

	// Test DeleteUser (:exec)
	t.Run("DeleteUser", func(t *testing.T) {
		createQuery := db.NewCreateUserQuery(executor)
		created, err := createQuery.Eval(ctx, "foofoo", "foofoo@example.com")
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
			_, err = updateQuery.Eval(ctx, user2.ID, "tx_batch2_modified@example.com")
			return err
		})

		if err != nil {
			t.Fatalf("WithTx nested queries failed: %v", err)
		}
	})
}
