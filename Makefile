## Filename Makefile

include .envrc

.PHONY: fmt
fmt: 
	go fmt ./...

.PHONY: vet
vet: fmt
	go vet ./...

.PHONY: run
run: vet
	go run ./cmd/web -addr=":4000" -dsn=${TRACKER_DB_DSN}

.PHONY: db/psql
db/psql:
	psql ${TRACKER_DB_DSN}

## db/migrations/up: apply all up database migrations
.PHONY: db/migrations/up
db/migrations/up:
	@echo 'Running up migrations...'
	migrate -path ./migrations -database ${TRACKER_DB_DSN} up

.PHONY: db/migrations/down
db/migrations/down:
	@echo 'Running down migrations...'
	migrate -path ./migrations -database ${TRACKER_DB_DSN} down