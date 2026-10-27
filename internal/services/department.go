package services

import (
	"context"
	"errors"
	"strings"

	"github.com/thaind-2785/go-practice/internal/models"
	"github.com/thaind-2785/go-practice/internal/repositories"
	"github.com/thaind-2785/go-practice/internal/utils"
)

const departmentNotFoundMsg = "Department not found"

type DepartmentService struct {
	deptRepo repositories.DepartmentRepository
	empRepo  repositories.EmployeeRepository
}

func NewDepartmentService(d repositories.DepartmentRepository, e repositories.EmployeeRepository) *DepartmentService {
	return &DepartmentService{
		deptRepo: d,
		empRepo:  e,
	}
}

func (s *DepartmentService) Create(ctx context.Context, req models.CreateDepartmentRequest) (*models.Department, error) {
	if err := req.Validate(); err != nil {
		return nil, utils.NewValidation(err.Error())
	}

	// Trim so "Engineering " and "Engineering" count as the same name for the UNIQUE constraint.
	department := models.Department{Name: strings.TrimSpace(req.Name)}

	if err := s.deptRepo.Create(ctx, &department); err != nil {
		if errors.Is(err, repositories.ErrDuplicate) {
			return nil, utils.NewConflict("department name already exists")
		}
		return nil, mapRepoError(err, departmentNotFoundMsg)
	}

	return &department, nil
}

func (s *DepartmentService) List(ctx context.Context) ([]models.Department, error) {
	departments, err := s.deptRepo.List(ctx)
	if err != nil {
		return nil, mapRepoError(err, departmentNotFoundMsg)
	}

	if departments == nil {
		departments = []models.Department{}
	}

	return departments, nil
}

func (s *DepartmentService) ListEmployees(ctx context.Context, id int64) ([]models.Employee, error) {
	// An empty employee list cannot tell "no employees yet" from "no such department",
	// so check that the department exists first.
	if _, err := s.deptRepo.GetByID(ctx, id); err != nil {
		return nil, mapRepoError(err, departmentNotFoundMsg)
	}

	employees, err := s.empRepo.ListByDepartment(ctx, id)
	if err != nil {
		return nil, mapRepoError(err, departmentNotFoundMsg)
	}

	if employees == nil {
		employees = []models.Employee{}
	}

	return employees, nil
}
