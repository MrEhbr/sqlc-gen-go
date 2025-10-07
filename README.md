# sqlc-gen-go

> [!NOTE]
> This is a fork of [sqlc-dev/sqlc-gen-go](https://github.com/sqlc-dev/sqlc-gen-go) maintained by [@MrEhbr](https://github.com/MrEhbr) with architectural improvements and additional features.

## What's Different

This fork introduces a **query struct pattern** that replaces the traditional `Querier` interface approach:

- **No Querier interface**: Instead of a single interface with all query methods, each query becomes its own struct type
- **Executor pattern**: Queries implement type-specific interfaces (`QueryOne`, `QueryMany`, `QueryExec`, etc.) and use an executor for database operations
- **Separate models package**: New option to generate models in a separate package from queries
- **Flexible file organization**: Control where query files are generated

### New Configuration Options

This fork adds new options for organizing generated code into separate packages:

- `output_models_package` - Package name for generated models (e.g., `"models"`)
- `models_package_import_path` - Import path for models package (required when `output_models_package` is used)
- `output_queries_package` - Package name for generated queries (e.g., `"queries"`)
- `db_package_import_path` - Import path for db package (used when queries are in separate package)
- `output_query_files_directory` - Subdirectory for query files (e.g., `"queries"`)

See [Building from source](#building-from-source) and [Configuration Examples](#configuration-examples) below.

## Usage

### Basic Configuration

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

## Configuration Examples

### Separate Models Package

Generate models in a separate package from queries:

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

This generates:
- `models/models.go` - models in `package models`
- `db.go` - executor code in `package db`
- `query.sql.go` - queries in `package db` (imports models package)

### Advanced: Split Packages with Custom File Organization

For maximum flexibility, combine multiple options (see `examples/pgx-split-packages`):

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

This generates:
- `models/models.go` - models in `package models`
- `db/db.go` - executor code in `package db`
- `queries/query.sql.gen.go` - queries in `package queries` (imports both models and db packages)

## Building from source

### Requirements

- Go 1.24.0 or later
- Make

### Build

From the project root:

```sh
make all
```

This produces:
- `bin/sqlc-gen-go` - Standalone binary plugin
- `bin/sqlc-gen-go.wasm` - WASM plugin (recommended)

Both plugins are functionally equivalent. WASM is recommended for better portability and easier configuration.

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

## Migrating from Original sqlc-gen-go

> [!WARNING]
> This fork uses a different code generation pattern than both the original sqlc-gen-go and sqlc's built-in codegen. The generated code is **not compatible** - you'll need to update your code to use the new query struct pattern.

### From sqlc's Built-in Codegen

Let's say you're generating Go code today using a sqlc.yaml configuration that looks something like this:

```yaml
version: 2
sql:
- schema: "query.sql"
  queries: "query.sql"
  engine: "postgresql"
  gen:
    go:
      package: "db"
      out: "db"
      emit_json_tags: true
      emit_pointers_for_null_types: true
      query_parameter_limit: 5
      overrides:
      - column: "authors.id"
        go_type: "your/package.SomeType"
      rename:
        foo: "bar"
```

To use this fork's WASM plugin, your config will look like this:

```yaml
version: 2
plugins:
- name: golang
  wasm:
    url: file:///path/to/bin/sqlc-gen-go.wasm
    sha256: ""
sql:
- schema: "query.sql"
  queries: "query.sql"
  engine: "postgresql"
  codegen:
  - plugin: golang
    out: "db"
    options:
      package: "db"
      emit_json_tags: true
      emit_pointers_for_null_types: true
      query_parameter_limit: 5
      overrides:
      - column: "authors.id"
        go_type: "your/package.SomeType"
      rename:
        foo: "bar"
```

The configuration structure is similar, but note:
* Use a `file://` URL pointing to your built WASM plugin
* The `sha256` field is optional (leave empty)
* All the same options work, plus the new options described above
* **Generated code structure is different** - no `Querier` interface, queries are structs instead

### Global overrides and renames

If you have global overrides or renames configured, you’ll need to move those to the new top-level `options` field. Replace the existing `go` field name with the name you gave your plugin in the `plugins` list. We’ve used `"golang"` in this example.

If your existing configuration looks like this:

```yaml
version: "2"
overrides:
  go:
    rename:
      id: "Identifier"
    overrides:
    - db_type: "timestamptz"
      nullable: true
      engine: "postgresql"
      go_type:
        import: "gopkg.in/guregu/null.v4"
        package: "null"
        type: "Time"
...
```

Then your updated configuration would look like this:

```yaml
version: "2"
plugins:
- name: golang
  wasm:
    url: file:///path/to/bin/sqlc-gen-go.wasm
    sha256: ""
options:
  golang:
    rename:
      id: "Identifier"
    overrides:
    - db_type: "timestamptz"
      nullable: true
      engine: "postgresql"
      go_type:
        import: "gopkg.in/guregu/null.v4"
        package: "null"
        type: "Time"
...
```

## Examples

See the [`examples/`](examples/) directory for working examples with different SQL drivers:

- **pgx** - PostgreSQL with pgx/v5
- **pgx-split-packages** - PostgreSQL with separate models package
- **mysql** - MySQL with go-sql-driver/mysql
- **sqlite** - SQLite with modernc.org/sqlite
- **stdlib-postgres** - PostgreSQL with database/sql and lib/pq

Each example includes:
- SQL schema and queries
- sqlc.yaml configuration
- Generated code
- Tests demonstrating usage

## Contributing

This is a fork maintained by [@MrEhbr](https://github.com/MrEhbr). Issues and pull requests are welcome at https://github.com/MrEhbr/sqlc-gen-go.

## License

Same as the original sqlc-gen-go project.
