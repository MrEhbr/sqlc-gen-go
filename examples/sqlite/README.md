# SQLite Example

This example demonstrates sqlc-gen-go with SQLite using modernc.org/sqlite (pure Go driver).

## Query Types Demonstrated

- `:one` - Returns a single row (with RETURNING clause support)
- `:many` - Returns multiple rows
- `:exec` - Executes without returning data
- `:execrows` - Returns number of affected rows
- `:execresult` - Returns sql.Result
- `:execlastid` - Returns last insert ID

## Running Tests

Tests use an in-memory SQLite database and require no external setup:

```bash
go test -v
```

## Generating Code

```bash
sqlc generate
```
