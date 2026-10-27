package repositories

import (
	"context"

	"github.com/thaind-2785/go-practice/internal/models"
)

type EmployeeRepository interface {
	Create(ctx context.Context, e *models.Employee) error
	GetByID(ctx context.Context, id int64) (*models.Employee, error)
	List(ctx context.Context, f models.ListEmployeesFilter) (employees []models.Employee, total int, err error)
	Update(ctx context.Context, e *models.Employee) error
	Delete(ctx context.Context, id int64) error
	Search(ctx context.Context, keyword string) ([]models.Employee, error)
	ListByDepartment(ctx context.Context, deptID int64) ([]models.Employee, error)
	ListAll(ctx context.Context) ([]models.Employee, error)
}

type DepartmentRepository interface {
	Create(ctx context.Context, d *models.Department) error
	List(ctx context.Context) ([]models.Department, error)
	GetByID(ctx context.Context, id int64) (*models.Department, error)
}
