package services

import (
	"context"
	"errors"

	"github.com/thaind-2785/go-practice/internal/repositories"
	"github.com/thaind-2785/go-practice/internal/utils"
)

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
		return err
	}
}
