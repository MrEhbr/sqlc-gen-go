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
		db.ExpectGetUser(1, db.User{ID: 1, Name: "Alice", Email: "alice@test.com"}, nil),
	)

	query := db.NewGetUserQuery(stub)
	user, err := query.Eval(ctx, 1)
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
		db.ExpectGetUser(1, db.User{ID: 1, Name: "Alice", Email: "alice@test.com"}, nil),
		db.ExpectListUsers([]db.User{
			{ID: 1, Name: "Alice"},
			{ID: 2, Name: "Bob"},
		}, nil),
		db.ExpectDeleteUser(1, 1, nil),
	)

	// Test GetUser
	getQuery := db.NewGetUserQuery(stub)
	user, err := getQuery.Eval(ctx, 1)
	if err != nil {
		t.Fatalf("GetUser failed: %v", err)
	}
	if user.Name != "Alice" {
		t.Errorf("expected Alice, got %s", user.Name)
	}

	// Test ListUsers
	listQuery := db.NewListUsersQuery(stub)
	users, err := listQuery.Eval(ctx)
	if err != nil {
		t.Fatalf("ListUsers failed: %v", err)
	}
	if len(users) != 2 {
		t.Errorf("expected 2 users, got %d", len(users))
	}

	// Test DeleteUser
	deleteQuery := db.NewDeleteUserQuery(stub)
	rows, err := deleteQuery.Eval(ctx, 1)
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
		db.ExpectGetUser(1, db.User{}, errors.New("database error")),
	)

	query := db.NewGetUserQuery(stub)
	_, err := query.Eval(ctx, 1)

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
		db.ExpectCreateUser(
			db.CreateUserParams{Name: "Alice", Email: "alice@test.com"},
			db.User{ID: 1, Name: "Alice", Email: "alice@test.com"},
			nil,
		),
		// Second call: GetUser
		db.ExpectGetUser(1, db.User{ID: 1, Name: "Alice", Email: "alice@test.com"}, nil),
	)

	// Create then Get - order matters!
	createQuery := db.NewCreateUserQuery(stub)
	createdUser, err := createQuery.Eval(ctx, db.CreateUserParams{Name: "Alice", Email: "alice@test.com"})
	if err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}
	if createdUser.ID != 1 {
		t.Errorf("expected created user ID 1, got %d", createdUser.ID)
	}

	getQuery := db.NewGetUserQuery(stub)
	gotUser, err := getQuery.Eval(ctx, 1)
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
		db.ExpectGetUser(1, db.User{ID: 1, Name: "Alice"}, nil),
		db.ExpectGetUser(2, db.User{ID: 2, Name: "Bob"}, nil),
		db.ExpectGetUser(1, db.User{ID: 1, Name: "Alice"}, nil), // Called again!
	)

	query := db.NewGetUserQuery(stub)

	user1, _ := query.Eval(ctx, 1)
	if user1.Name != "Alice" {
		t.Errorf("expected Alice, got %s", user1.Name)
	}

	user2, _ := query.Eval(ctx, 2)
	if user2.Name != "Bob" {
		t.Errorf("expected Bob, got %s", user2.Name)
	}

	user1Again, _ := query.Eval(ctx, 1)
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
		db.ExpectGetUser(1, db.User{ID: 1, Name: "Alice"}, nil),
	)

	// Execute the expected query
	query := db.NewGetUserQuery(stub)
	query.Eval(ctx, 1)

	// Try to execute an unexpected query
	query.Eval(ctx, 2)

	if !fakeT.failed {
		t.Error("expected test to fail on unexpected query")
	}
}

// TestIncompleteSteps shows that AssertDone fails when not all steps executed
func TestIncompleteSteps(t *testing.T) {
	fakeT := &fakeT{}

	stub := db.NewStubExecutor(fakeT,
		db.ExpectGetUser(1, db.User{ID: 1, Name: "Alice"}, nil),
		db.ExpectGetUser(2, db.User{ID: 2, Name: "Bob"}, nil),
	)

	// Only execute one query
	ctx := context.Background()
	query := db.NewGetUserQuery(stub)
	query.Eval(ctx, 1)

	// Call AssertDone - should fail because second step wasn't executed
	stub.AssertDone()

	if !fakeT.failed {
		t.Error("expected AssertDone to fail when steps remain")
	}
}

// fakeT implements the testing interface to capture failures
type fakeT struct {
	failed bool
}

func (f *fakeT) Helper() {}
func (f *fakeT) Errorf(format string, args ...any) {
	f.failed = true
}
