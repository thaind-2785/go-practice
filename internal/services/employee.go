package services

import (
	"context"
	"errors"
	"strings"

	"github.com/thaind-2785/go-practice/internal/models"
	"github.com/thaind-2785/go-practice/internal/repositories"
	"github.com/thaind-2785/go-practice/internal/utils"
)

const (
	defaultLimit = 10
	maxLimit     = 100
)

const employeeNotFoundMsg = "Employee not found"

type EmployeeService struct {
	empRepo  repositories.EmployeeRepository
	deptRepo repositories.DepartmentRepository
}

func NewEmployeeService(e repositories.EmployeeRepository, d repositories.DepartmentRepository) *EmployeeService {
	return &EmployeeService{
		empRepo:  e,
		deptRepo: d,
	}
}

func (s *EmployeeService) Create(ctx context.Context, req models.CreateEmployeeRequest) (*models.Employee, error) {
	if err := req.Validate(); err != nil {
		return nil, utils.NewValidation(err.Error())
	}

	if err := s.ensureDepartmentExists(ctx, req.DepartmentID); err != nil {
		return nil, err
	}

	employee := models.Employee{
		Name:         req.Name,
		Age:          req.Age,
		Position:     req.Position,
		DepartmentID: req.DepartmentID,
		Salary:       req.Salary,
	}

	if err := s.empRepo.Create(ctx, &employee); err != nil {
		return nil, mapRepoError(err, employeeNotFoundMsg)
	}

	return &employee, nil
}

func (s *EmployeeService) Get(ctx context.Context, id int64) (*models.Employee, error) {
	employee, err := s.empRepo.GetByID(ctx, id)
	if err != nil {
		return nil, mapRepoError(err, employeeNotFoundMsg)
	}

	return employee, nil
}

func (s *EmployeeService) List(ctx context.Context, f models.ListEmployeesFilter) (*models.ListEmployeesResult, error) {
	if f.Limit <= 0 {
		f.Limit = defaultLimit
	}
	if f.Limit > maxLimit {
		f.Limit = maxLimit
	}
	if f.Offset < 0 {
		f.Offset = 0
	}

	employees, total, err := s.empRepo.List(ctx, f)
	if err != nil {
		return nil, mapRepoError(err, employeeNotFoundMsg)
	}

	if employees == nil {
		employees = []models.Employee{}
	}

	return &models.ListEmployeesResult{
		TotalCount: total,
		Employees:  employees,
	}, nil
}

func (s *EmployeeService) Update(ctx context.Context, id int64, req models.UpdateEmployeeRequest) (*models.Employee, error) {
	if err := req.Validate(); err != nil {
		return nil, utils.NewValidation(err.Error())
	}

	employee, err := s.empRepo.GetByID(ctx, id)
	if err != nil {
		return nil, mapRepoError(err, employeeNotFoundMsg)
	}

	if req.DepartmentID != nil && *req.DepartmentID != employee.DepartmentID {
		if err := s.ensureDepartmentExists(ctx, *req.DepartmentID); err != nil {
			return nil, err
		}
	}

	if req.Name != nil {
		employee.Name = *req.Name
	}
	if req.Age != nil {
		employee.Age = *req.Age
	}
	if req.Position != nil {
		employee.Position = *req.Position
	}
	if req.DepartmentID != nil {
		employee.DepartmentID = *req.DepartmentID
	}
	if req.Salary != nil {
		employee.Salary = *req.Salary
	}

	if err := s.empRepo.Update(ctx, employee); err != nil {
		return nil, mapRepoError(err, employeeNotFoundMsg)
	}

	return employee, nil
}

func (s *EmployeeService) Delete(ctx context.Context, id int64) error {
	if err := s.empRepo.Delete(ctx, id); err != nil {
		return mapRepoError(err, employeeNotFoundMsg)
	}

	return nil
}

func (s *EmployeeService) Search(ctx context.Context, keyword string) ([]models.Employee, error) {
	keyword = strings.TrimSpace(keyword)
	if keyword == "" {
		return nil, utils.NewBadRequest("keyword is required")
	}

	employees, err := s.empRepo.Search(ctx, keyword)
	if err != nil {
		return nil, mapRepoError(err, employeeNotFoundMsg)
	}

	if employees == nil {
		employees = []models.Employee{}
	}

	return employees, nil
}

func (s *EmployeeService) ensureDepartmentExists(ctx context.Context, id int64) error {
	if _, err := s.deptRepo.GetByID(ctx, id); err != nil {
		if errors.Is(err, repositories.ErrNotFound) {
			return utils.NewValidation("department does not exist")
		}
		return mapRepoError(err, "")
	}

	return nil
}
