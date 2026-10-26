package models

import "time"

type Department struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"createdAt"`
}

type CreateDepartmentRequest struct {
	Name string `json:"name"`
}
