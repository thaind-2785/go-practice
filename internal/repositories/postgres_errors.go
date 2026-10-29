package repositories

import (
	"database/sql"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgconn"
)

const (
	pgUniqueViolation     = "23505"
	pgForeignKeyViolation = "23503"
)

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
