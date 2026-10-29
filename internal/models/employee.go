package models

import (
	"errors"
	"strings"
	"time"
)

type Employee struct {
	ID             int64      `json:"id"`
	Name           string     `json:"name"`
	Age            int        `json:"age"`
	Position       string     `json:"position"`
	DepartmentID   int64      `json:"departmentId"`
	DepartmentName string     `json:"departmentName,omitempty"`
	Salary         float64    `json:"salary"`
	CreatedAt      time.Time  `json:"createdAt"`
	UpdatedAt      time.Time  `json:"updatedAt"`
	DeletedAt      *time.Time `json:"-"`
}

type CreateEmployeeRequest struct {
	Name         string  `json:"name"`
	Age          int     `json:"age"`
	Position     string  `json:"position"`
	DepartmentID int64   `json:"departmentId"`
	Salary       float64 `json:"salary"`
}

func (r CreateEmployeeRequest) Validate() error {
	if strings.TrimSpace(r.Name) == "" {
		return errors.New("name cannot be empty")
	}

	if r.Age <= 0 {
		return errors.New("age must be greater than 0")
	}

	if r.Salary <= 0 {
		return errors.New("salary must be greater than 0")
	}

	if r.DepartmentID <= 0 {
		return errors.New("departmentId must be greater than 0")
	}

	return nil
}

type UpdateEmployeeRequest struct {
	Name         *string  `json:"name"`
	Age          *int     `json:"age"`
	Position     *string  `json:"position"`
	DepartmentID *int64   `json:"departmentId"`
	Salary       *float64 `json:"salary"`
}

func (r UpdateEmployeeRequest) Validate() error {
	if r.Name != nil && strings.TrimSpace(*r.Name) == "" {
		return errors.New("name cannot be empty")
	}

	if r.Age != nil && *r.Age <= 0 {
		return errors.New("age must be greater than 0")
	}

	if r.Salary != nil && *r.Salary <= 0 {
		return errors.New("salary must be greater than 0")
	}

	if r.DepartmentID != nil && *r.DepartmentID <= 0 {
		return errors.New("departmentId must be greater than 0")
	}

	return nil
}

type ListEmployeesFilter struct {
	Limit        int
	Offset       int
	DepartmentID *int64
}
type ListEmployeesResult struct {
	TotalCount int        `json:"totalCount"`
	Employees  []Employee `json:"employees"`
}
