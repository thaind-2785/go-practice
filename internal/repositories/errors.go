package repositories

import "errors"

var (
	ErrNotFound   = errors.New("not found")
	ErrDuplicate  = errors.New("duplicate")
	ErrForeignKey = errors.New("foreign key violation")
)
