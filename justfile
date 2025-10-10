set shell := ["bash", "-uc"]

# List available recipes
default:
    @just --list
# Build all Go packages
build:
    go build ./...
# Run tests
test: wasm
    go test --count 1 ./...
# Run tests for all examples
test-examples: wasm
    cd examples/pgx && go test --count 1 ./...
    cd examples/pgx-v4 && go test --count 1 ./...
    cd examples/stdlib-postgres && go test --count 1 ./...
    cd examples/mysql && go test --count 1 ./...
    cd examples/sqlite && go test --count 1 ./...
# Build WASM plugin
wasm:
    GOOS=wasip1 GOARCH=wasm go build -o bin/sqlc-gen-go.wasm ./plugin/main.go
# Generate code for all examples
generate: wasm
    #!/usr/bin/env bash
    for mod in $(go list -f '{{{{.Dir}}' -m | grep examples | xargs); do
      cd $mod && sqlc generate
    done
# Clean all build artifacts
clean:
    rm -rf bin/
# Format Go code
fmt:
    go fmt ./...
# Run linter (requires golangci-lint)
lint:
    golangci-lint run --timeout=5m
# Tidy Go modules
tidy:
    go mod tidy
# Run CI checks locally (fmt, lint, test)
ci: fmt lint test
