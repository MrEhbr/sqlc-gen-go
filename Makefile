.PHONY: build test examples clean-examples bin/sqlc-gen-go bin/sqlc-gen-go.wasm

build:
	go build ./...

test: bin/sqlc-gen-go.wasm
	go test ./...

all: bin/sqlc-gen-go bin/sqlc-gen-go.wasm

bin/sqlc-gen-go: bin go.mod go.sum $(wildcard **/*.go)
	cd plugin && go build -o ../bin/sqlc-gen-go ./main.go

bin/sqlc-gen-go.wasm: bin/sqlc-gen-go
	cd plugin && GOOS=wasip1 GOARCH=wasm go build -o ../bin/sqlc-gen-go.wasm main.go

bin:
	mkdir -p bin

examples: bin/sqlc-gen-go.wasm clean-examples
	cd examples/pgx && sqlc generate
	cd examples/pgx-v4 && sqlc generate
	cd examples/stdlib-postgres && sqlc generate
	cd examples/mysql && sqlc generate
	cd examples/sqlite && sqlc generate
	cd examples/pgx-split-packages && sqlc generate

clean-examples:
	rm -rf examples/pgx/db
	rm -rf examples/pgx-v4/db
	rm -rf examples/stdlib-postgres/db
	rm -rf examples/mysql/db
	rm -rf examples/sqlite/db
	rm -rf examples/pgx-split-packages/db
	rm -rf examples/pgx-split-packages/models
	rm -rf examples/pgx-split-packages/queries
