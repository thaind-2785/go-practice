package services

import (
	"context"
	"errors"

	"github.com/thaind-2785/go-practice/internal/repositories"
	"github.com/thaind-2785/go-practice/internal/utils"
)

// mapRepoError translates storage errors into AppErrors that carry an HTTP code.
// notFoundMsg is used when the repository reports ErrNotFound.
func mapRepoError(err error, notFoundMsg string) error {
	switch {
	case errors.Is(err, repositories.ErrNotFound):
		return utils.NewNotFound(notFoundMsg)
	case errors.Is(err, repositories.ErrDuplicate):
		return utils.NewConflict("resource already exists")
	case errors.Is(err, repositories.ErrForeignKey):
		return utils.NewValidation("related resource does not exist")
	case errors.Is(err, context.DeadlineExceeded):
		return utils.NewServiceUnavailable("database timeout")
	default:
		// Unknown errors are passed through; WriteError logs them and returns a generic 500.
		return err
	}
}
