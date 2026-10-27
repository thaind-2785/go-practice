package services

import (
	"cmp"
	"context"
	"slices"
	"strings"
	"time"

	"github.com/thaind-2785/go-practice/internal/models"
	"github.com/thaind-2785/go-practice/internal/repositories"
)

// Compile-time checks: the build fails if a fake is missing a method of the interface.
var (
	_ repositories.EmployeeRepository   = (*fakeEmployeeRepo)(nil)
	_ repositories.DepartmentRepository = (*fakeDepartmentRepo)(nil)
)

// fakeEmployeeRepo is an in-memory EmployeeRepository used by service tests.
// Setting err makes every method fail with it, which simulates a broken database.
type fakeEmployeeRepo struct {
	data   map[int64]models.Employee
	nextID int64
	err    error
}

func newFakeEmployeeRepo() *fakeEmployeeRepo {
	return &fakeEmployeeRepo{data: map[int64]models.Employee{}}
}

func (f *fakeEmployeeRepo) Create(ctx context.Context, e *models.Employee) error {
	if f.err != nil {
		return f.err
	}
	f.nextID++
	now := time.Now()
	e.ID = f.nextID
	e.CreatedAt = now
	e.UpdatedAt = now
	f.data[e.ID] = *e
	return nil
}

func (f *fakeEmployeeRepo) GetByID(ctx context.Context, id int64) (*models.Employee, error) {
	if f.err != nil {
		return nil, f.err
	}
	e, ok := f.data[id]
	if !ok || e.DeletedAt != nil {
		return nil, repositories.ErrNotFound
	}
	return &e, nil
}

func (f *fakeEmployeeRepo) List(ctx context.Context, filter models.ListEmployeesFilter) ([]models.Employee, int, error) {
	if f.err != nil {
		return nil, 0, f.err
	}
	var matched []models.Employee
	for _, e := range f.active() {
		if filter.DepartmentID == nil || e.DepartmentID == *filter.DepartmentID {
			matched = append(matched, e)
		}
	}
	total := len(matched)
	if filter.Offset >= total {
		return nil, total, nil
	}
	end := min(filter.Offset+filter.Limit, total)
	return matched[filter.Offset:end], total, nil
}

func (f *fakeEmployeeRepo) Update(ctx context.Context, e *models.Employee) error {
	if f.err != nil {
		return f.err
	}
	old, ok := f.data[e.ID]
	if !ok || old.DeletedAt != nil {
		return repositories.ErrNotFound
	}
	e.UpdatedAt = time.Now()
	f.data[e.ID] = *e
	return nil
}

func (f *fakeEmployeeRepo) Delete(ctx context.Context, id int64) error {
	if f.err != nil {
		return f.err
	}
	e, ok := f.data[id]
	if !ok || e.DeletedAt != nil {
		return repositories.ErrNotFound
	}
	now := time.Now()
	e.DeletedAt = &now
	f.data[id] = e
	return nil
}

func (f *fakeEmployeeRepo) Search(ctx context.Context, keyword string) ([]models.Employee, error) {
	if f.err != nil {
		return nil, f.err
	}
	keyword = strings.ToLower(keyword)
	var result []models.Employee
	for _, e := range f.active() {
		if strings.Contains(strings.ToLower(e.Name), keyword) || strings.Contains(strings.ToLower(e.Position), keyword) {
			result = append(result, e)
		}
	}
	return result, nil
}

func (f *fakeEmployeeRepo) ListByDepartment(ctx context.Context, deptID int64) ([]models.Employee, error) {
	if f.err != nil {
		return nil, f.err
	}
	var result []models.Employee
	for _, e := range f.active() {
		if e.DepartmentID == deptID {
			result = append(result, e)
		}
	}
	return result, nil
}

func (f *fakeEmployeeRepo) ListAll(ctx context.Context) ([]models.Employee, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.active(), nil
}

// active returns non-deleted employees sorted by ID, because map iteration order is random.
func (f *fakeEmployeeRepo) active() []models.Employee {
	var result []models.Employee
	for _, e := range f.data {
		if e.DeletedAt == nil {
			result = append(result, e)
		}
	}
	slices.SortFunc(result, func(a, b models.Employee) int { return cmp.Compare(a.ID, b.ID) })
	return result
}

// fakeDepartmentRepo is an in-memory DepartmentRepository used by service tests.
type fakeDepartmentRepo struct {
	data   map[int64]models.Department
	nextID int64
	err    error
}

func newFakeDepartmentRepo() *fakeDepartmentRepo {
	return &fakeDepartmentRepo{data: map[int64]models.Department{}}
}

func (f *fakeDepartmentRepo) Create(ctx context.Context, d *models.Department) error {
	if f.err != nil {
		return f.err
	}
	for _, existing := range f.data {
		if existing.Name == d.Name {
			return repositories.ErrDuplicate
		}
	}
	f.nextID++
	d.ID = f.nextID
	d.CreatedAt = time.Now()
	f.data[d.ID] = *d
	return nil
}

func (f *fakeDepartmentRepo) List(ctx context.Context) ([]models.Department, error) {
	if f.err != nil {
		return nil, f.err
	}
	var result []models.Department
	for _, d := range f.data {
		result = append(result, d)
	}
	slices.SortFunc(result, func(a, b models.Department) int { return cmp.Compare(a.ID, b.ID) })
	return result, nil
}

func (f *fakeDepartmentRepo) GetByID(ctx context.Context, id int64) (*models.Department, error) {
	if f.err != nil {
		return nil, f.err
	}
	d, ok := f.data[id]
	if !ok {
		return nil, repositories.ErrNotFound
	}
	return &d, nil
}
