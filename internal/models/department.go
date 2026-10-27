package models

import (
	"errors"
	"strings"
	"time"
)

type Department struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"createdAt"`
}

type CreateDepartmentRequest struct {
	Name string `json:"name"`
}

func (r CreateDepartmentRequest) Validate() error {
	if strings.TrimSpace(r.Name) == "" {
		return errors.New("name cannot be empty")
	}

	return nil
}
