package repositories

import (
	"database/sql"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgconn"
)

// Postgres error codes: https://www.postgresql.org/docs/current/errcodes-appendix.html
const (
	pgUniqueViolation     = "23505"
	pgForeignKeyViolation = "23503"
)

// mapPgError converts database/sql and Postgres errors into this package's sentinel errors.
// op describes the failed operation so logs show where the error came from.
// Errors are wrapped with %w, so errors.Is still finds the original (e.g. context.DeadlineExceeded).
func mapPgError(op string, err error) error {
	if errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("%s: %w", op, ErrNotFound)
	}

	if pgErr, ok := errors.AsType[*pgconn.PgError](err); ok {
		switch pgErr.Code {
		case pgUniqueViolation:
			return fmt.Errorf("%s: %w", op, ErrDuplicate)
		case pgForeignKeyViolation:
			return fmt.Errorf("%s: %w", op, ErrForeignKey)
		}
	}

	return fmt.Errorf("%s: %w", op, err)
}
