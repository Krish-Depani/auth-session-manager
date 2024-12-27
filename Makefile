include .env

migrate-create:
	migrate create -ext sql -dir $(MIGRATIONS_DIR) -seq $(name)

migrate-up:
	migrate -path $(MIGRATIONS_DIR) -database $(DATABASE_URL) -verbose up

migrate-down:
	migrate -path $(MIGRATIONS_DIR) -database $(DATABASE_URL) -verbose down 1

migrate-force:
	migrate -path $(MIGRATIONS_DIR) -database $(DATABASE_URL) -verbose force $(version)

migrate-version:
	migrate -path $(MIGRATIONS_DIR) -database $(DATABASE_URL) -verbose version

migrate-fix:
	migrate -path $(MIGRATIONS_DIR) -database $(DATABASE_URL) -verbose fix

migrate-status:
	migrate -path $(MIGRATIONS_DIR) -database $(DATABASE_URL) -verbose status

.PHONY: migrate-create migrate-up migrate-down