package db_test

import (
	"context"
	"database/sql"
	"testing"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/sqlc-dev/sqlc-gen-go/examples/mysql/db"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/mysql"
	"github.com/testcontainers/testcontainers-go/wait"
)

func setupTestDB(t *testing.T) (*sql.DB, func()) {
	t.Helper()

	ctx := context.Background()

	mysqlContainer, err := mysql.Run(ctx,
		"mysql:8",
		mysql.WithDatabase("testdb"),
		mysql.WithUsername("root"),
		mysql.WithPassword("password"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("port: 3306  MySQL Community Server").
				WithOccurrence(1).
				WithStartupTimeout(60*time.Second)),
	)
	if err != nil {
		t.Fatalf("failed to start mysql container: %v", err)
	}

	connStr, err := mysqlContainer.ConnectionString(ctx, "parseTime=true")
	if err != nil {
		t.Fatalf("failed to get connection string: %v", err)
	}

	var database *sql.DB
	for range 10 {
		database, err = sql.Open("mysql", connStr)
		if err == nil {
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
	if _, err = database.Exec(`DROP TABLE IF EXISTS posts`); err != nil {
		t.Fatalf("failed to drop posts: %v", err)
	}
	if _, err = database.Exec(`DROP TABLE IF EXISTS users`); err != nil {
		t.Fatalf("failed to drop users: %v", err)
	}

	_, err = database.Exec(`
CREATE TABLE users (
  id BIGINT PRIMARY KEY AUTO_INCREMENT,
  name VARCHAR(255) NOT NULL,
  email VARCHAR(255) NOT NULL UNIQUE,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
)`)
	if err != nil {
		t.Fatalf("failed to create users: %v", err)
	}

	_, err = database.Exec(`
CREATE TABLE posts (
  id         BIGINT PRIMARY KEY AUTO_INCREMENT,
  author_id  BIGINT NOT NULL,
  title      VARCHAR(255) NOT NULL,
  body       TEXT NOT NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  FOREIGN KEY (author_id) REFERENCES users(id) ON DELETE CASCADE
)`)
	if err != nil {
		t.Fatalf("failed to create posts: %v", err)
	}

	cleanup := func() {
		database.Close()
		if err := mysqlContainer.Terminate(ctx); err != nil {
			t.Logf("failed to terminate container: %v", err)
		}
	}

	return database, cleanup
}

func TestQueries(t *testing.T) {
	ctx := context.Background()
	database, cleanup := setupTestDB(t)
	defer cleanup()

	d := db.New(db.NewExecutor(database))

	// Test CreateUser (:execresult)
	t.Run("CreateUser", func(t *testing.T) {
		result, err := d.Run(ctx, db.CreateUser("foobar", "foobar@example.com"))
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
		result, _ := d.Run(ctx, db.CreateUser("foobaz", "foobaz@example.com"))
		id, _ := result.LastInsertId()

		user, err := d.Run(ctx, db.GetUser(id))
		if err != nil {
			t.Fatalf("GetUser failed: %v", err)
		}
		if user.Name != "foobaz" {
			t.Errorf("expected name foobaz, got %s", user.Name)
		}
	})

	// Test ListUsers (:many)
	t.Run("ListUsers", func(t *testing.T) {
		users, err := d.Run(ctx, db.ListUsers())
		if err != nil {
			t.Fatalf("ListUsers failed: %v", err)
		}
		if len(users) < 2 {
			t.Errorf("expected at least 2 users, got %d", len(users))
		}
	})

	// Test BulkInsertUsers (:copyfrom via LOAD DATA LOCAL INFILE)
	t.Run("BulkInsertUsers", func(t *testing.T) {
		if _, err := database.ExecContext(ctx, "SET GLOBAL local_infile = 1"); err != nil {
			t.Fatalf("enable local_infile failed: %v", err)
		}
		rows := []db.BulkInsertUsersParams{
			{Name: "bulk1", Email: "bulk1@example.com"},
			{Name: "bulk2", Email: "bulk2@example.com"},
			{Name: "bulk3", Email: "bulk3@example.com"},
		}

		count, err := d.Run(ctx, db.BulkInsertUsers(rows))
		if err != nil {
			t.Fatalf("BulkInsertUsers failed: %v", err)
		}
		if count != 3 {
			t.Errorf("expected 3 rows inserted, got %d", count)
		}
	})

	// Test UpdateUserEmail (:execresult)
	t.Run("UpdateUserEmail", func(t *testing.T) {
		result, _ := d.Run(ctx, db.CreateUser("barbaz", "barbaz@example.com"))
		id, _ := result.LastInsertId()

		updateResult, err := d.Run(ctx, db.UpdateUserEmail("barbaz.updated@example.com", id))
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

		user, err := d.Run(ctx, db.GetUser(id))
		if err != nil {
			t.Fatalf("GetUser failed: %v", err)
		}
		if user.Email != "barbaz.updated@example.com" {
			t.Errorf("expected email barbaz.updated@example.com, got %s", user.Email)
		}
	})

	// Test UpdateUserName (:execrows)
	t.Run("UpdateUserName", func(t *testing.T) {
		result, _ := d.Run(ctx, db.CreateUser("barfoo", "barfoo@example.com"))
		id, _ := result.LastInsertId()

		rows, err := d.Run(ctx, db.UpdateUserName("barfoo-updated", id))
		if err != nil {
			t.Fatalf("UpdateUserName failed: %v", err)
		}
		if rows != 1 {
			t.Errorf("expected 1 row updated, got %d", rows)
		}
	})

	// Test CreateUserGetID (:execlastid)
	t.Run("CreateUserGetID", func(t *testing.T) {
		id, err := d.Run(ctx, db.CreateUserGetID("bazfoo", "bazfoo@example.com"))
		if err != nil {
			t.Fatalf("CreateUserGetID failed: %v", err)
		}
		if id == 0 {
			t.Error("expected non-zero last insert ID")
		}

		user, err := d.Run(ctx, db.GetUser(id))
		if err != nil {
			t.Fatalf("GetUser failed: %v", err)
		}
		if user.Name != "bazfoo" {
			t.Errorf("expected name bazfoo, got %s", user.Name)
		}
	})

	// Test DeleteUser (:exec)
	t.Run("DeleteUser", func(t *testing.T) {
		result, _ := d.Run(ctx, db.CreateUser("bazbar", "bazbar@example.com"))
		id, _ := result.LastInsertId()

		_, err := d.Run(ctx, db.DeleteUser(id))
		if err != nil {
			t.Fatalf("DeleteUser failed: %v", err)
		}

		_, err = d.Run(ctx, db.GetUser(id))
		if err != sql.ErrNoRows {
			t.Errorf("expected ErrNoRows, got %v", err)
		}
	})

	// Test ListUsersByIDs, CountUsersByIDsExceptName (sqlc.slice) and CountUsersByNameOrEmail (repeated sqlc.arg)
	t.Run("SqlcSlice", func(t *testing.T) {
		id1, err := d.Run(ctx, db.CreateUserGetID("slice1", "slice1@example.com"))
		if err != nil {
			t.Fatalf("CreateUserGetID failed: %v", err)
		}
		id2, err := d.Run(ctx, db.CreateUserGetID("slice2", "slice2@example.com"))
		if err != nil {
			t.Fatalf("CreateUserGetID failed: %v", err)
		}

		users, err := d.Run(ctx, db.ListUsersByIDs([]int64{id1, id2}))
		if err != nil {
			t.Fatalf("ListUsersByIDs failed: %v", err)
		}
		if len(users) != 2 || users[0].ID != id1 || users[1].ID != id2 {
			t.Errorf("expected users %d and %d, got %+v", id1, id2, users)
		}

		none, err := d.Run(ctx, db.ListUsersByIDs(nil))
		if err != nil {
			t.Fatalf("ListUsersByIDs(nil) failed: %v", err)
		}
		if len(none) != 0 {
			t.Errorf("expected no users for empty slice, got %d", len(none))
		}

		count, err := d.Run(ctx, db.CountUsersByIDsExceptName([]int64{id1, id2}, "slice1"))
		if err != nil {
			t.Fatalf("CountUsersByIDsExceptName failed: %v", err)
		}
		if count != 1 {
			t.Errorf("expected 1 user, got %d", count)
		}

		count, err = d.Run(ctx, db.CountUsersByNameOrEmail("slice2@example.com"))
		if err != nil {
			t.Fatalf("CountUsersByNameOrEmail failed: %v", err)
		}
		if count != 1 {
			t.Errorf("expected 1 user, got %d", count)
		}
	})

	// Test WithTx - successful transaction
	t.Run("WithTx_Success", func(t *testing.T) {
		var createdID int64

		err := d.WithTx(ctx, func(tx db.DB) error {
			// Create two users in the same transaction
			id, err := tx.Run(ctx, db.CreateUserGetID("tx_user", "tx_user@example.com"))
			if err != nil {
				return err
			}
			createdID = id

			// Create another user to verify transaction atomicity
			_, err = tx.Run(ctx, db.CreateUserGetID("tx_user2", "tx_user2@example.com"))
			return err
		})
		if err != nil {
			t.Fatalf("WithTx failed: %v", err)
		}

		// Verify the transaction was committed - both users should exist
		user, err := d.Run(ctx, db.GetUser(createdID))
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

		err := d.WithTx(ctx, func(tx db.DB) error {
			// Create a user in transaction
			id, err := tx.Run(ctx, db.CreateUserGetID("tx_rollback", "tx_rollback@example.com"))
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
		_, err = d.Run(ctx, db.GetUser(createdID))
		if err != sql.ErrNoRows {
			t.Errorf("expected ErrNoRows for rolled back user, got %v", err)
		}
	})

	// Test WithTx - nested queries
	t.Run("WithTx_NestedQueries", func(t *testing.T) {
		err := d.WithTx(ctx, func(tx db.DB) error {
			// Create multiple users in a transaction
			id1, err := tx.Run(ctx, db.CreateUserGetID("tx_batch1", "tx_batch1@example.com"))
			if err != nil {
				return err
			}

			id2, err := tx.Run(ctx, db.CreateUserGetID("tx_batch2", "tx_batch2@example.com"))
			if err != nil {
				return err
			}

			// Verify we can read them within the transaction
			user1, err := tx.Run(ctx, db.GetUser(id1))
			if err != nil {
				return err
			}
			if user1.Name != "tx_batch1" {
				t.Errorf("expected name tx_batch1, got %s", user1.Name)
			}

			// Verify second user as well
			user2, err := tx.Run(ctx, db.GetUser(id2))
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

	t.Run("GetPostWithAuthor", func(t *testing.T) {
		authorID, err := d.Run(ctx, db.CreateUserGetID("embed_author", "embed_author@example.com"))
		if err != nil {
			t.Fatalf("CreateUserGetID failed: %v", err)
		}
		postID, err := d.Run(ctx, db.CreatePost(db.CreatePostParams{AuthorID: authorID, Title: "embed_title", Body: "embed_body"}))
		if err != nil {
			t.Fatalf("CreatePost failed: %v", err)
		}
		row, err := d.Run(ctx, db.GetPostWithAuthor(postID))
		if err != nil {
			t.Fatalf("GetPostWithAuthor failed: %v", err)
		}
		if row.Post.ID != postID || row.Post.Title != "embed_title" {
			t.Errorf("post not hydrated: %+v", row.Post)
		}
		if row.User.ID != authorID || row.User.Name != "embed_author" {
			t.Errorf("embedded user not hydrated: %+v", row.User)
		}
	})

	t.Run("ListPostsWithAuthor", func(t *testing.T) {
		authorID, err := d.Run(ctx, db.CreateUserGetID("embed_lister", "embed_lister@example.com"))
		if err != nil {
			t.Fatalf("CreateUserGetID failed: %v", err)
		}
		for i, title := range []string{"a", "b"} {
			if _, err := d.Run(ctx, db.CreatePost(db.CreatePostParams{AuthorID: authorID, Title: title, Body: "body"})); err != nil {
				t.Fatalf("CreatePost #%d failed: %v", i, err)
			}
		}
		rows, err := d.Run(ctx, db.ListPostsWithAuthor())
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
