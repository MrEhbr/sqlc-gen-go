# sqlc-gen-go

[![Go Version](https://img.shields.io/github/go-mod/go-version/MrEhbr/sqlc-gen-go)](https://go.dev/)
[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)
[![Release](https://img.shields.io/github/v/release/MrEhbr/sqlc-gen-go)](https://github.com/MrEhbr/sqlc-gen-go/releases)

A Go code generator plugin for [sqlc](https://sqlc.dev/) that generates type-safe, composable query values from SQL.

> [!NOTE]
> This is a fork of [sqlc-dev/sqlc-gen-go](https://github.com/sqlc-dev/sqlc-gen-go) maintained by [@MrEhbr](https://github.com/MrEhbr) with typed query values and package organization features.

> [!WARNING]
> **Breaking Changes**: This fork generates typed query values instead of the traditional `Querier` interface. The generated code is **not compatible** with standard sqlc and requires **Go 1.27+**.

## What's Different

This fork replaces the traditional `Querier` interface with **typed query values**:

- **No Querier interface**: Instead of a single interface with all query methods, each query is a constructor returning a `Query[T]` value
- **Executor pattern**: Queries run through a swappable `QueryExecutor` (real database, transaction, or test stub)
- **Typed results**: `DB.Run` returns `T`, and user-defined queries plug into the same executor
- **Separate models package**: New option to generate models in a separate package from queries
- **Flexible file organization**: Control where query files are generated

### Code Comparison

Given this SQL:

```sql
-- name: GetUser :one
SELECT * FROM users WHERE id = $1;

-- name: ListUsers :many
SELECT * FROM users ORDER BY created_at DESC;
```

**Standard sqlc (interface-based):**

```go
// All queries in one interface
type Querier interface {
    GetUser(ctx context.Context, id int64) (User, error)
    ListUsers(ctx context.Context) ([]User, error)
}

// Usage
q := New(db)
user, err := q.GetUser(ctx, 1)
```

**This fork (typed query values, requires Go 1.27+):**

```go
d := New(NewExecutor(pool))
user, err := d.Run(ctx, GetUser(1))     // Query[User]
users, err := d.Run(ctx, ListUsers())   // Query[[]User]
```

### Query Values

With every driver (`pgx/v5`, `pgx/v4`, `database/sql` for PostgreSQL, MySQL and SQLite), every query compiles to a constructor returning a `Query[T]` value. `DB.Run` executes it on a `QueryExecutor` and returns `T`. The generated code uses generic methods and requires **Go 1.27+**.

| Command | Generated type |
|---------|----------------|
| `:one` | `Query[Row]` |
| `:many` | `Query[[]Row]` |
| `:exec`, `:execrows` | `Query[int64]` (rows affected) |
| `:execresult` | `Query[pgconn.CommandTag]` (pgx), `Query[sql.Result]` (database/sql) |
| `:execlastid` | `Query[int64]` (last insert ID, database/sql) |
| `:copyfrom` | `Query[int64]` (rows copied; pgx, or MySQL with `sql_driver: github.com/go-sql-driver/mysql`) |
| `:batchexec`, `:batchone`, `:batchmany` | `Query[*<name>BatchResults]` (pgx) |

Transactions:

```go
err := d.WithTx(ctx, func(tx DB) error {
    _, err := tx.Run(ctx, UpdateUserEmail(id, email))
    return err
})
```

Custom queries implement the same interface and run through `DB`, transactions and `StubExecutor`:

```go
type Query[T any] interface {
    SQL() string
    Args() []any
    Do(ctx context.Context, db DBTX) (T, error)
}

// Or build one from the helpers generated code uses:
active := NewStatement("SELECT COUNT(*) FROM users WHERE status = $1", "active").One(ScanValue[int64])
n, err := d.Run(ctx, active)
```

Mocking with `emit_mock_executor: true`:

```go
stub := NewStubExecutor(t,
    Expect(GetUser(1), User{ID: 1, Name: "Alice"}, nil),
    Expect(ListUsers(), []User{{ID: 1}}, nil),
)
d := New(stub)
```

With `emit_exported_queries: true`, query constants get an `SQL` suffix (`GetUserSQL`), since `GetUser` is the constructor. Query names that collide with the generated runtime (`Query`, `DB`, `New`, `Expect`, …) are rejected at generation time.

`sqlc.slice` (MySQL, SQLite) is expanded when the query value is constructed, so `Expect` matches the expanded SQL and flattened arguments. With SQLite, place `sqlc.slice` after other parameters in the query: sqlc emits numbered placeholders (`?1`, `?2`) there, and a slice expanded before them shifts their positions.

### New Configuration Options

This fork adds new options for organizing generated code into separate packages:

- `output_models_package` - Package name for generated models (e.g., `"models"`)
- `models_package_import_path` - Import path for models package (required when `output_models_package` is used)
- `output_queries_package` - Package name for generated queries (e.g., `"queries"`)
- `db_package_import_path` - Import path for db package (used when queries are in separate package)
- `output_query_files_directory` - Subdirectory for query files (e.g., `"queries"`)

See [Building from source](#building-from-source) and [Configuration Examples](#configuration-examples) below.

## Usage

### Installing the Plugin

You can use this plugin either from a GitHub release or by building from source.

#### Using GitHub Releases

Download the latest WASM plugin from the [releases page](https://github.com/MrEhbr/sqlc-gen-go/releases):

```yaml
version: '2'
plugins:
- name: golang
  wasm:
    url: https://github.com/MrEhbr/sqlc-gen-go/releases/download/<version>/sqlc-gen-go.wasm
    sha256: ""  # Get from checksums.txt in the release assets
sql:
- schema: schema.sql
  queries: query.sql
  engine: postgresql
  codegen:
  - plugin: golang
    out: db
    options:
      package: db
      sql_package: pgx/v5
```

**Finding the SHA256 checksum:**

1. Go to the [releases page](https://github.com/MrEhbr/sqlc-gen-go/releases)
2. Download the `checksums.txt` file from the release assets
3. Copy the SHA256 hash and paste it into your `sqlc.yaml`

The `sha256` field is optional but recommended for better performance (sqlc caches plugins with verified checksums).

#### Using Local Build

If you've built the plugin from source, use a `file://` URL:

```yaml
version: '2'
plugins:
- name: golang
  wasm:
    url: file:///path/to/bin/sqlc-gen-go.wasm
    sha256: ""  # Optional for local builds
sql:
- schema: schema.sql
  queries: query.sql
  engine: postgresql
  codegen:
  - plugin: golang
    out: db
    options:
      package: db
      sql_package: pgx/v5
```

## Configuration Examples

### Basic Setup (Single Package)

```yaml
version: '2'
plugins:
- name: golang
  wasm:
    url: file:///path/to/bin/sqlc-gen-go.wasm
    sha256: ""
sql:
- schema: schema.sql
  queries: query.sql
  engine: postgresql
  codegen:
  - plugin: golang
    out: db
    options:
      package: db
      sql_package: pgx/v5
```

**Generated structure:**

```
db/
├── db.go           # Executor and database code
├── models.go       # Table models
└── query.sql.go    # Query constructors and row scanners
```

### Separate Models Package

```yaml
version: '2'
plugins:
- name: golang
  wasm:
    url: file:///path/to/bin/sqlc-gen-go.wasm
    sha256: ""
sql:
- schema: schema.sql
  queries: query.sql
  engine: postgresql
  codegen:
  - plugin: golang
    out: .
    options:
      package: db
      sql_package: pgx/v5
      output_models_package: models
      models_package_import_path: github.com/yourorg/yourproject/models
      output_models_file_name: models/models.go
```

**Generated structure:**

```
models/
└── models.go       # Table models (package models)
db.go               # Executor code (package db)
query.sql.go        # Query constructors (package db, imports models)
```

### Split Packages with Custom File Organization

```yaml
version: '2'
plugins:
- name: golang
  wasm:
    url: file:///path/to/bin/sqlc-gen-go.wasm
    sha256: ""
sql:
- schema: schema.sql
  queries: query.sql
  engine: postgresql
  codegen:
  - plugin: golang
    out: .
    options:
      package: db
      sql_package: pgx/v5
      # Models in separate package
      output_models_package: models
      models_package_import_path: github.com/yourorg/yourproject/models
      output_models_file_name: models/models.go
      # Queries in separate package
      output_queries_package: queries
      output_query_files_directory: queries
      db_package_import_path: github.com/yourorg/yourproject/db
      # DB code in db/ subdirectory
      output_db_file_name: db/db.go
      # Add .gen suffix to generated files
      output_files_suffix: .gen
```

**Generated structure:**

```
models/
└── models.go                # Table models (package models)
db/
└── db.go                    # Executor code (package db)
queries/
└── query.sql.gen.go         # Query constructors (package queries, imports models + db)
```

## Building from source

### Requirements

- Go 1.24 or later
- [just](https://github.com/casey/just)

### Build

From the project root:

```sh
just wasm
```

This produces `bin/sqlc-gen-go.wasm` - the WASM plugin for sqlc.

**Available commands:**

```sh
just --list        # Show all available recipes
just build         # Build Go packages
just test          # Run tests
just test-examples # Run example tests
just generate      # Generate code for all examples
just fmt           # Format code
just lint          # Run linter
just clean         # Clean build artifacts
```

To use a local WASM build with sqlc, just update your configuration with a `file://`
URL pointing at the WASM blob in your `bin` directory:

```yaml
plugins:
- name: golang
  wasm:
    url: file:///path/to/bin/sqlc-gen-go.wasm
    sha256: ""
```

As-of sqlc v1.24.0 the `sha256` is optional, but without it sqlc won't cache your
module internally which will impact performance.

## Examples

See the [`examples/`](examples/) directory for working examples with different SQL drivers.

## Contributing

This fork is maintained by [@MrEhbr](https://github.com/MrEhbr). Contributions are welcome!

### Development Workflow

1. **Fork and clone**:

   ```bash
   git clone https://github.com/YOUR-USERNAME/sqlc-gen-go
   cd sqlc-gen-go
   ```

2. **Install dependencies**:

   ```bash
   go mod download
   ```

3. **Make changes** and test:

   ```bash
   just build         # Build Go packages
   just wasm          # Build WASM plugin
   just test          # Run tests
   just test-examples # Test all examples
   ```

4. **Format and lint**:

   ```bash
   just fmt  # Format code
   just lint # Run linter
   ```

5. **Submit a PR** with:
   - Clear description of the change
   - Tests covering new functionality
   - Updated documentation if needed
   - Passing CI checks

### Reporting Issues

Please include:

- sqlc version (`sqlc version`)
- Plugin version or commit hash
- Minimal reproduction (SQL schema, queries, config)
- Expected vs actual behavior
- Error messages and stack traces

## License

MIT License - see [LICENSE](LICENSE) file for details.
