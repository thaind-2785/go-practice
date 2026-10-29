# Employee Management System

A REST API for managing employees and departments, written in pure Go: `net/http`, `encoding/json`, `database/sql`, `context`, `sync` and `testing` from the standard library. The only external module is the PostgreSQL driver [pgx](https://github.com/jackc/pgx).

## Requirements

- Go 1.27+
- Docker (runs PostgreSQL 17 on port 5434)

## Getting started

```bash
cp .env.example .env   # then set your own AUTH_USERS password
make db-up             # start PostgreSQL in Docker
make migrate           # create tables in the dev and test databases
make run               # API on http://localhost:8080
```

Stop the server with `Ctrl+C`. It finishes in-flight requests before exiting. `make db-down` stops the database; the data stays in the `emp-pg-data` volume.

### Environment variables

| Name | Description |
|---|---|
| `DATABASE_URL` | PostgreSQL connection string used by the API |
| `TEST_DATABASE_URL` | Database for repository integration tests. They are skipped when it is not set |
| `AUTH_USERS` | Basic Auth users, `user:password` pairs separated by commas |

## API

Every endpoint requires Basic Auth, for example `curl -u admin:change-me ...`.

| Method | Path | Description |
|---|---|---|
| `POST` | `/employees` | Create an employee |
| `GET` | `/employees?limit=10&offset=0&departmentId=1` | List employees with pagination and `totalCount` |
| `GET` | `/employees/{id}` | Get one employee |
| `PUT` | `/employees/{id}` | Partial update: only the fields sent are changed |
| `DELETE` | `/employees/{id}` | Soft delete |
| `GET` | `/employees/search?keyword=an` | Search by name or position |
| `POST` | `/employees/export` | Write `exports/employees.json` and `exports/employees.csv` concurrently |
| `POST` | `/departments` | Create a department |
| `GET` | `/departments` | List departments |
| `GET` | `/departments/{id}/employees` | Employees of a department, with the department name |

### Example

```bash
curl -u admin:change-me -X POST localhost:8080/departments -d '{"name":"Engineering"}'

curl -u admin:change-me -X POST localhost:8080/employees \
  -d '{"name":"Nguyen Van A","age":28,"position":"Backend Engineer","departmentId":1,"salary":2000}'

curl -u admin:change-me "localhost:8080/employees?limit=10&offset=0"
curl -u admin:change-me -X PUT localhost:8080/employees/1 -d '{"salary":2500}'
curl -u admin:change-me -X POST localhost:8080/employees/export
```

### Errors

Errors always have the same JSON shape:

```json
{"error": "Employee not found", "code": 404}
```

| Code | When |
|---|---|
| 400 | Invalid ID, malformed JSON, unknown field, invalid query parameter |
| 401 | Missing or wrong credentials |
| 404 | Employee or department not found |
| 409 | Department name already exists |
| 422 | Validation failed, or the department of an employee does not exist |
| 500 | Unexpected error (details are logged, not returned) |
| 503 | Database did not answer within the request timeout (3s) |

## Project structure

```
cmd/app/                 entry point: config, wiring, middleware chain, graceful shutdown
internal/handlers/       HTTP layer: parse request, call service, write JSON
internal/services/       business rules and validation
internal/repositories/   repository interfaces and the PostgreSQL implementation
internal/middleware/     request ID, recover, basic auth, logging, timeout
internal/models/         request, response and domain types
internal/utils/          AppError and JSON response helpers
migrations/              SQL schema
```

Requests flow `handler → service → repository`. Services depend on repository interfaces, so they are unit tested with in-memory fakes; handlers depend on service interfaces and are tested with `httptest`.

Notable details:

- **PostgreSQL**: `$n` placeholders everywhere, a prepared statement for `GetByID`, and a transaction with `SELECT ... FOR UPDATE` for updates.
- **Context**: each request gets a 3-second deadline that cancels its database queries; the authenticated user and request ID travel in the request context.
- **Concurrent export**: one goroutine per format, a `sync.WaitGroup` to wait for them and a `sync.Mutex` around the shared results. Files are written to a temp file and renamed, so concurrent exports never produce a half-written file.

## Tests

```bash
make test   # go test ./... -race
```

Repository tests run against `TEST_DATABASE_URL` and truncate its tables; all other tests need no database.
