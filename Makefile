-include .env
export

DB_CONTAINER := emp-pg
PSQL := docker exec -i -e PGOPTIONS=--client-min-messages=warning $(DB_CONTAINER) psql -U app -v ON_ERROR_STOP=1 -q

.PHONY: run test db-up db-down migrate

run:
	go run ./cmd/app

test:
	go test ./... -race

db-up:
	docker start $(DB_CONTAINER) 2>/dev/null || docker run --name $(DB_CONTAINER) \
		-e POSTGRES_USER=app -e POSTGRES_PASSWORD=secret -e POSTGRES_DB=employees \
		-v emp-pg-data:/var/lib/postgresql/data -p 5434:5432 -d postgres:17
	until docker exec $(DB_CONTAINER) pg_isready -h 127.0.0.1 -U app -d employees >/dev/null 2>&1; do sleep 1; done

db-down:
	docker stop $(DB_CONTAINER)

migrate:
	$(PSQL) -d employees -tAc "SELECT 1 FROM pg_database WHERE datname = 'employees_test'" | grep -q 1 \
		|| $(PSQL) -d employees -c "CREATE DATABASE employees_test"
	$(PSQL) -d employees < migrations/001_init.sql
	$(PSQL) -d employees_test < migrations/001_init.sql
