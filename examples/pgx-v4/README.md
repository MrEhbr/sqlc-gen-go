# PostgreSQL with pgx/v5 Example

This example demonstrates sqlc-gen-go with PostgreSQL using the pgx/v5 driver.

## Query Types Demonstrated

- `:one` - Returns a single row
- `:many` - Returns multiple rows
- `:exec` - Executes without returning data
- `:execrows` - Returns number of affected rows
- `:batchexec` - Batch execution (pgx-specific)
- `:batchone` - Batch query returning single rows (pgx-specific)
- `:batchmany` - Batch query returning multiple rows (pgx-specific)
- `:copyfrom` - Bulk copy operation (pgx-specific)

## Running Tests

Tests require a running PostgreSQL database. You can use Docker:

```bash
# Start PostgreSQL
docker run --name sqlc-postgres -e POSTGRES_PASSWORD=password -e POSTGRES_DB=testdb -p 5432:5432 -d postgres:16

# Run tests
DATABASE_URL="postgres://postgres:password@localhost:5432/testdb?sslmode=disable" go test -v

# Stop PostgreSQL
docker stop sqlc-postgres && docker rm sqlc-postgres
```

## Generating Code

```bash
sqlc generate
```
