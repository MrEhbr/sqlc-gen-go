# sqlc-gen-go

[![Go Version](https://img.shields.io/github/go-mod/go-version/MrEhbr/sqlc-gen-go)](https://go.dev/)
[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)
[![Release](https://img.shields.io/github/v/release/MrEhbr/sqlc-gen-go)](https://github.com/MrEhbr/sqlc-gen-go/releases)

A Go code generator plugin for [sqlc](https://sqlc.dev/) that generates type-safe, composable query structs from SQL.

> [!NOTE]
> This is a fork of [sqlc-dev/sqlc-gen-go](https://github.com/sqlc-dev/sqlc-gen-go) maintained by [@MrEhbr](https://github.com/MrEhbr) with an alternative query struct pattern and package organization features.

> [!WARNING]
> **Breaking Changes**: This fork uses a query struct pattern instead of the traditional `Querier` interface. The generated code is **not compatible** with standard sqlc.

## What's Different

This fork introduces a **query struct pattern** that replaces the traditional `Querier` interface approach:

- **No Querier interface**: Instead of a single interface with all query methods, each query becomes its own struct type
- **Executor pattern**: Queries implement type-specific interfaces (`QueryOne`, `QueryMany`, `QueryExec`, etc.) and use an executor for database operations
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

**This fork (query struct pattern):**

```go
// Usage - more flexible
executor := NewExecutor(db)
query := NewGetUserQuery(executor)
user, err := query.Eval(ctx, 1)
```

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
└── query.sql.go    # Query structs and methods
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
query.sql.go        # Query structs (package db, imports models)
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
└── query.sql.gen.go         # Query structs (package queries, imports models + db)
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
