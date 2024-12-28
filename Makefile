include .env

start-dev:
	go run main.go

start-prod:
	go build -o bin/main main.go
	GIN_MODE=release ./bin/main

migrate-create:
	migrate create -ext sql -dir $(MIGRATIONS_DIR) -seq $(name)

migrate-up:
	migrate -path $(MIGRATIONS_DIR) -database $(DATABASE_URL) -verbose up

migrate-down:
	migrate -path $(MIGRATIONS_DIR) -database $(DATABASE_URL) -verbose down $(n)

migrate-down-to:
	migrate -path $(MIGRATIONS_DIR) -database $(DATABASE_URL) -verbose goto $(version)

migrate-force:
	migrate -path $(MIGRATIONS_DIR) -database $(DATABASE_URL) -verbose force $(version)

migrate-status:
	migrate -path $(MIGRATIONS_DIR) -database $(DATABASE_URL) -verbose version

.PHONY: migrate-create migrate-up migrate-down migrate-down-to migrate-force migrate-status