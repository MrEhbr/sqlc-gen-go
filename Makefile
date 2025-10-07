.PHONY: build test examples test-examples clean-examples bin/sqlc-gen-go bin/sqlc-gen-go.wasm

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

test-examples:
	@docker run -d --name sqlc-test-postgres -e POSTGRES_PASSWORD=password -e POSTGRES_DB=testdb -p 5432:5432 postgres:16-alpine || true
	@docker run -d --name sqlc-test-mysql -e MYSQL_ROOT_PASSWORD=password -e MYSQL_DATABASE=testdb -p 3306:3306 mysql:8-oracle || true
	@sleep 15
	-cd examples/pgx && DATABASE_URL="postgres://postgres:password@localhost:5432/testdb?sslmode=disable" go test -v --count 1
	-cd examples/pgx-v4 && env DATABASE_URL="postgres://postgres:password@localhost:5432/testdb?sslmode=disable" go test -v --count 1
	-cd examples/stdlib-postgres && DATABASE_URL="postgres://postgres:password@localhost:5432/testdb?sslmode=disable" go test -v --count 1
	-cd examples/mysql && DATABASE_URL="root:password@tcp(localhost:3306)/testdb?parseTime=true" go test -v --count 1
	-cd examples/sqlite && go test -v --count 1
	@docker stop sqlc-test-postgres sqlc-test-mysql || true
	@docker rm sqlc-test-postgres sqlc-test-mysql || true

clean-examples:
	rm -rf examples/pgx/db
	rm -rf examples/pgx-v4/db
	rm -rf examples/stdlib-postgres/db
	rm -rf examples/mysql/db
	rm -rf examples/sqlite/db
	rm -rf examples/pgx-split-packages/db
	rm -rf examples/pgx-split-packages/models
	rm -rf examples/pgx-split-packages/queries
