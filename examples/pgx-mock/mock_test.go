package db_test

import (
	"context"
	"errors"
	"testing"

	"github.com/sqlc-dev/sqlc-gen-go/examples/pgx-mock/db"
)

// TestSingleQuery shows simple single query mocking
func TestSingleQuery(t *testing.T) {
	ctx := context.Background()

	stub := db.NewStubExecutor(t,
		db.Expect(db.GetUser(1), db.User{ID: 1, Name: "Alice", Email: "alice@test.com"}, nil),
	)

	d := db.New(stub)

	user, err := d.Run(ctx, db.GetUser(1))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if user.Name != "Alice" {
		t.Errorf("expected Alice, got %s", user.Name)
	}

	stub.AssertDone()
}

// TestMultipleQueries shows how to set up multiple query expectations
func TestMultipleQueries(t *testing.T) {
	ctx := context.Background()

	stub := db.NewStubExecutor(t,
		db.Expect(db.GetUser(1), db.User{ID: 1, Name: "Alice", Email: "alice@test.com"}, nil),
		db.Expect(db.ListUsers(), []db.User{
			{ID: 1, Name: "Alice"},
			{ID: 2, Name: "Bob"},
		}, nil),
		db.Expect(db.DeleteUser(1), 1, nil),
	)

	d := db.New(stub)

	// Test GetUser
	user, err := d.Run(ctx, db.GetUser(1))
	if err != nil {
		t.Fatalf("GetUser failed: %v", err)
	}
	if user.Name != "Alice" {
		t.Errorf("expected Alice, got %s", user.Name)
	}

	// Test ListUsers
	users, err := d.Run(ctx, db.ListUsers())
	if err != nil {
		t.Fatalf("ListUsers failed: %v", err)
	}
	if len(users) != 2 {
		t.Errorf("expected 2 users, got %d", len(users))
	}

	// Test DeleteUser
	rows, err := d.Run(ctx, db.DeleteUser(1))
	if err != nil {
		t.Fatalf("DeleteUser failed: %v", err)
	}
	if rows != 1 {
		t.Errorf("expected 1 row, got %d", rows)
	}

	stub.AssertDone()
}

// TestErrorCase shows error handling
func TestErrorCase(t *testing.T) {
	ctx := context.Background()

	stub := db.NewStubExecutor(t,
		db.Expect(db.GetUser(1), db.User{}, errors.New("database error")),
	)

	d := db.New(stub)

	_, err := d.Run(ctx, db.GetUser(1))

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if err.Error() != "database error" {
		t.Errorf("expected 'database error', got %v", err)
	}

	stub.AssertDone()
}

// TestOrderedCalls verifies that queries are executed in the expected order
func TestOrderedCalls(t *testing.T) {
	ctx := context.Background()

	stub := db.NewStubExecutor(t,
		// First call: CreateUser
		db.Expect(db.CreateUser(db.CreateUserParams{Name: "Alice", Email: "alice@test.com"}), db.User{ID: 1, Name: "Alice", Email: "alice@test.com"}, nil),
		// Second call: GetUser
		db.Expect(db.GetUser(1), db.User{ID: 1, Name: "Alice", Email: "alice@test.com"}, nil),
	)

	d := db.New(stub)

	// Create then Get - order matters!
	createdUser, err := d.Run(ctx, db.CreateUser(db.CreateUserParams{Name: "Alice", Email: "alice@test.com"}))
	if err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}
	if createdUser.ID != 1 {
		t.Errorf("expected created user ID 1, got %d", createdUser.ID)
	}

	gotUser, err := d.Run(ctx, db.GetUser(1))
	if err != nil {
		t.Fatalf("GetUser failed: %v", err)
	}
	if gotUser.Name != "Alice" {
		t.Errorf("expected Alice, got %s", gotUser.Name)
	}

	stub.AssertDone()
}

// TestRepeatedQueries shows how to expect the same query multiple times
func TestRepeatedQueries(t *testing.T) {
	ctx := context.Background()

	// Expect GetUser to be called twice with different IDs
	stub := db.NewStubExecutor(t,
		db.Expect(db.GetUser(1), db.User{ID: 1, Name: "Alice"}, nil),
		db.Expect(db.GetUser(2), db.User{ID: 2, Name: "Bob"}, nil),
		db.Expect(db.GetUser(1), db.User{ID: 1, Name: "Alice"}, nil), // Called again!
	)
	d := db.New(stub)

	user1, _ := d.Run(ctx, db.GetUser(1))
	if user1.Name != "Alice" {
		t.Errorf("expected Alice, got %s", user1.Name)
	}

	user2, _ := d.Run(ctx, db.GetUser(2))
	if user2.Name != "Bob" {
		t.Errorf("expected Bob, got %s", user2.Name)
	}

	user1Again, _ := d.Run(ctx, db.GetUser(1))
	if user1Again.Name != "Alice" {
		t.Errorf("expected Alice, got %s", user1Again.Name)
	}

	stub.AssertDone()
}

// TestUnexpectedQuery shows that unexpected queries fail the test
func TestUnexpectedQuery(t *testing.T) {
	// Create a test helper to capture errors without failing the outer test
	fakeT := &fakeT{}

	ctx := context.Background()

	stub := db.NewStubExecutor(fakeT,
		db.Expect(db.GetUser(1), db.User{ID: 1, Name: "Alice"}, nil),
	)

	d := db.New(stub)

	// Execute the expected query
	d.Run(ctx, db.GetUser(1))

	// Try to execute an unexpected query
	d.Run(ctx, db.GetUser(2))

	if !fakeT.failed {
		t.Error("expected test to fail on unexpected query")
	}
}

// TestIncompleteSteps shows that AssertDone fails when not all steps executed
func TestIncompleteSteps(t *testing.T) {
	fakeT := &fakeT{}

	stub := db.NewStubExecutor(fakeT,
		db.Expect(db.GetUser(1), db.User{ID: 1, Name: "Alice"}, nil),
		db.Expect(db.GetUser(2), db.User{ID: 2, Name: "Bob"}, nil),
	)

	d := db.New(stub)

	// Only execute one query
	ctx := context.Background()
	d.Run(ctx, db.GetUser(1))

	// Call AssertDone - should fail because second step wasn't executed
	stub.AssertDone()

	if !fakeT.failed {
		t.Error("expected AssertDone to fail when steps remain")
	}
}

// userEmails is a hand-written query; any db.Query[T] runs through DB and StubExecutor.
type userEmails struct {
	ids []int64
}

func (q userEmails) SQL() string { return "SELECT id, email FROM users WHERE id = ANY($1)" }
func (q userEmails) Args() []any { return []any{q.ids} }

func (q userEmails) Do(ctx context.Context, conn db.DBTX) (map[int64]string, error) {
	rows, err := conn.Query(ctx, q.SQL(), q.Args()...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[int64]string{}
	for rows.Next() {
		var id int64
		var email string
		if err := rows.Scan(&id, &email); err != nil {
			return nil, err
		}
		out[id] = email
	}
	return out, rows.Err()
}

// TestCustomQuery shows mocking a user-defined query alongside generated ones
func TestCustomQuery(t *testing.T) {
	ctx := context.Background()

	stub := db.NewStubExecutor(t,
		db.Expect(userEmails{ids: []int64{1, 2}}, map[int64]string{1: "alice@test.com", 2: "bob@test.com"}, nil),
		db.Expect(db.NewStatement("SELECT COUNT(*) FROM users").One(db.ScanValue[int64]), 2, nil),
	)
	d := db.New(stub)

	emails, err := d.Run(ctx, userEmails{ids: []int64{1, 2}})
	if err != nil {
		t.Fatalf("userEmails failed: %v", err)
	}
	if emails[2] != "bob@test.com" {
		t.Errorf("expected bob@test.com, got %q", emails[2])
	}

	count, err := d.Run(ctx, db.NewStatement("SELECT COUNT(*) FROM users").One(db.ScanValue[int64]))
	if err != nil {
		t.Fatalf("count failed: %v", err)
	}
	if count != 2 {
		t.Errorf("expected 2, got %d", count)
	}

	stub.AssertDone()
}

// fakeT implements the testing interface to capture failures
type fakeT struct {
	failed bool
}

func (f *fakeT) Helper() {}
func (f *fakeT) Errorf(format string, args ...any) {
	f.failed = true
}
