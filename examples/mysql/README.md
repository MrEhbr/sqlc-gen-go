# MySQL Example

This example demonstrates sqlc-gen-go with MySQL using go-sql-driver/mysql.

## Query Types Demonstrated

- `:one` - Returns a single row
- `:many` - Returns multiple rows
- `:exec` - Executes without returning data
- `:execrows` - Returns number of affected rows
- `:execresult` - Returns sql.Result
- `:execlastid` - Returns last insert ID

## Running Tests

Tests require a running MySQL database. You can use Docker:

```bash
# Start MySQL
docker run --name sqlc-mysql -e MYSQL_ROOT_PASSWORD=password -e MYSQL_DATABASE=testdb -p 3306:3306 -d mysql:8

# Wait for MySQL to be ready
sleep 10

# Run tests
DATABASE_URL="root:password@tcp(localhost:3306)/testdb?parseTime=true" go test -v

# Stop MySQL
docker stop sqlc-mysql && docker rm sqlc-mysql
```

## Generating Code

```bash
sqlc generate
```
