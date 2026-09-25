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

CREATE TABLE posts (
  id         INTEGER PRIMARY KEY AUTOINCREMENT,
  author_id  INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  title      TEXT NOT NULL,
  body       TEXT NOT NULL,
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

	d := db.New(db.NewExecutor(database))

	// Test CreateUser (:one with RETURNING)
	t.Run("CreateUser", func(t *testing.T) {
		user, err := d.Run(ctx, db.CreateUser("foobar", "foobar@example.com"))
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
		created, err := d.Run(ctx, db.CreateUser("foobaz", "foobaz@example.com"))
		if err != nil {
			t.Fatalf("CreateUser failed: %v", err)
		}

		// Then get it
		user, err := d.Run(ctx, db.GetUser(created.ID))
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

	// Test UpdateUserEmail (:execrows)
	t.Run("UpdateUserEmail", func(t *testing.T) {
		created, err := d.Run(ctx, db.CreateUser("barbaz", "barbaz@example.com"))
		if err != nil {
			t.Fatalf("CreateUser failed: %v", err)
		}

		rows, err := d.Run(ctx, db.UpdateUserEmail("barbaz.updated@example.com", created.ID))
		if err != nil {
			t.Fatalf("UpdateUserEmail failed: %v", err)
		}
		if rows != 1 {
			t.Errorf("expected 1 row updated, got %d", rows)
		}

		// Verify update
		user, err := d.Run(ctx, db.GetUser(created.ID))
		if err != nil {
			t.Fatalf("GetUser failed: %v", err)
		}
		if user.Email != "barbaz.updated@example.com" {
			t.Errorf("expected email barbaz.updated@example.com, got %s", user.Email)
		}
	})

	// Test UpdateUserName (:execresult)
	t.Run("UpdateUserName", func(t *testing.T) {
		created, err := d.Run(ctx, db.CreateUser("barfoo", "barfoo@example.com"))
		if err != nil {
			t.Fatalf("CreateUser failed: %v", err)
		}

		result, err := d.Run(ctx, db.UpdateUserName("barfoo-updated", created.ID))
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
		id, err := d.Run(ctx, db.CreateUserGetID("bazfoo", "bazfoo@example.com"))
		if err != nil {
			t.Fatalf("CreateUserGetID failed: %v", err)
		}
		if id == 0 {
			t.Error("expected non-zero last insert ID")
		}

		// Verify user was created
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
		created, err := d.Run(ctx, db.CreateUser("bazbar", "bazbar@example.com"))
		if err != nil {
			t.Fatalf("CreateUser failed: %v", err)
		}

		_, err = d.Run(ctx, db.DeleteUser(created.ID))
		if err != nil {
			t.Fatalf("DeleteUser failed: %v", err)
		}

		// Verify deletion
		_, err = d.Run(ctx, db.GetUser(created.ID))
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

		count, err := d.Run(ctx, db.CountUsersByIDsExceptName("slice1", []int64{id1, id2}))
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
			// Create a user in transaction
			user, err := tx.Run(ctx, db.CreateUser("tx_user", "tx_user@example.com"))
			if err != nil {
				return err
			}
			createdID = user.ID

			// Update the user's email in the same transaction
			_, err = tx.Run(ctx, db.UpdateUserEmail("tx_user_updated@example.com", user.ID))
			return err
		})
		if err != nil {
			t.Fatalf("WithTx failed: %v", err)
		}

		// Verify the transaction was committed
		user, err := d.Run(ctx, db.GetUser(createdID))
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

		err := d.WithTx(ctx, func(tx db.DB) error {
			// Create a user in transaction
			user, err := tx.Run(ctx, db.CreateUser("tx_rollback", "tx_rollback@example.com"))
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
		_, err = d.Run(ctx, db.GetUser(createdID))
		if err != sql.ErrNoRows {
			t.Errorf("expected ErrNoRows for rolled back user, got %v", err)
		}
	})

	// Test WithTx - nested queries
	t.Run("WithTx_NestedQueries", func(t *testing.T) {
		err := d.WithTx(ctx, func(tx db.DB) error {
			// Create multiple users in a transaction
			user1, err := tx.Run(ctx, db.CreateUser("tx_batch1", "tx_batch1@example.com"))
			if err != nil {
				return err
			}

			user2, err := tx.Run(ctx, db.CreateUser("tx_batch2", "tx_batch2@example.com"))
			if err != nil {
				return err
			}

			// Verify we can read them within the transaction
			retrieved, err := tx.Run(ctx, db.GetUser(user1.ID))
			if err != nil {
				return err
			}
			if retrieved.Name != "tx_batch1" {
				t.Errorf("expected name tx_batch1, got %s", retrieved.Name)
			}

			// Update one of them
			_, err = tx.Run(ctx, db.UpdateUserEmail("tx_batch2_modified@example.com", user2.ID))
			return err
		})
		if err != nil {
			t.Fatalf("WithTx nested queries failed: %v", err)
		}
	})

	t.Run("GetPostWithAuthor", func(t *testing.T) {
		author, err := d.Run(ctx, db.CreateUser("embed_author", "embed_author@example.com"))
		if err != nil {
			t.Fatalf("CreateUser failed: %v", err)
		}
		post, err := d.Run(ctx, db.CreatePost(db.CreatePostParams{AuthorID: author.ID, Title: "embed_title", Body: "embed_body"}))
		if err != nil {
			t.Fatalf("CreatePost failed: %v", err)
		}
		row, err := d.Run(ctx, db.GetPostWithAuthor(post.ID))
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
		author, err := d.Run(ctx, db.CreateUser("embed_lister", "embed_lister@example.com"))
		if err != nil {
			t.Fatalf("CreateUser failed: %v", err)
		}
		for i, title := range []string{"a", "b"} {
			if _, err := d.Run(ctx, db.CreatePost(db.CreatePostParams{AuthorID: author.ID, Title: title, Body: "body"})); err != nil {
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
