# sqlc-gen-go Examples

This directory contains working examples demonstrating sqlc-gen-go usage with different SQL drivers.

## Structure

Examples are organized by SQL driver and features:

### Basic Driver Examples

- **pgx/** - PostgreSQL with `pgx/v5` driver
- **stdlib-postgres/** - PostgreSQL with `database/sql` (stdlib)
- **mysql/** - MySQL with `go-sql-driver/mysql`
- **sqlite/** - SQLite with `modernc.org/sqlite` (pure Go)

### Feature Examples

- **pgx-split-package/** - PostgreSQL with models in separate package
  - Demonstrates `output_models_package` and `models_package_import_path` options
- **pgx-query-subdir/** - PostgreSQL with queries in subdirectory
  - Demonstrates `output_query_files_directory` option

## Running Examples

Each example is a standalone Go module with its own `go.mod` file.

### Generate code for all examples

```bash
make examples
```

This regenerates the code in each `db/` directory using the WASM plugin.

### Generate code for a specific example

```bash
cd examples/pgx
sqlc generate
```

### Clean generated code

```bash
make clean-examples
```

### Running Tests

Each example includes tests demonstrating all supported query types.

**SQLite** (no setup required):
```bash
cd examples/sqlite
go test -v
```

**PostgreSQL/MySQL** (requires Docker):
```bash
cd examples/pgx  # or stdlib-postgres or mysql
# See individual example README.md for Docker setup commands
```

See each example's `README.md` for detailed testing instructions.

## Example Contents

Each example includes:

- `schema.sql` - Database schema definition
- `query.sql` - SQL queries with sqlc annotations
- `sqlc.yaml` - Configuration for sqlc code generation
- `go.mod` - Go module definition with dependencies
- `db/` - Generated Go code (committed to git)

## Development

The repository uses Go workspaces to manage the main module and example modules together:

- `go.work` in the repo root includes all modules
- Examples can import and test the generated code
- Example dependencies don't pollute the main generator's `go.mod`

## Adding New Examples

1. Create new directory under `examples/`
2. Add `schema.sql`, `query.sql`, and `sqlc.yaml`
3. Create `go.mod` with appropriate driver dependencies
4. Add the new example to `go.work`
5. Update `Makefile` targets for the new example
6. Run `make examples` to generate code
