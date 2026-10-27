package services

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/thaind-2785/go-practice/internal/models"
	"github.com/thaind-2785/go-practice/internal/utils"
)

// newTestService returns a service backed by fakes, with departments 1 (Engineering) and 2 (Sales).
func newTestService(t *testing.T) (*EmployeeService, *fakeEmployeeRepo) {
	t.Helper()
	empRepo := newFakeEmployeeRepo()
	deptRepo := newFakeDepartmentRepo()
	for _, name := range []string{"Engineering", "Sales"} {
		if err := deptRepo.Create(context.Background(), &models.Department{Name: name}); err != nil {
			t.Fatalf("seed department: %v", err)
		}
	}
	return NewEmployeeService(empRepo, deptRepo), empRepo
}

func validCreateRequest() models.CreateEmployeeRequest {
	return models.CreateEmployeeRequest{
		Name:         "Nguyen Van A",
		Age:          28,
		Position:     "Backend Engineer",
		DepartmentID: 1,
		Salary:       2000,
	}
}

func mustCreate(t *testing.T, svc *EmployeeService, req models.CreateEmployeeRequest) *models.Employee {
	t.Helper()
	e, err := svc.Create(context.Background(), req)
	if err != nil {
		t.Fatalf("create employee: %v", err)
	}
	return e
}

// assertCode checks that err is nil when wantCode is 0, otherwise an *AppError with that code.
func assertCode(t *testing.T, err error, wantCode int) {
	t.Helper()
	if wantCode == 0 {
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		return
	}
	appErr, ok := errors.AsType[*utils.AppError](err)
	if !ok {
		t.Fatalf("expected *AppError with code %d, got %v", wantCode, err)
	}
	if appErr.Code != wantCode {
		t.Fatalf("code = %d, want %d (message: %q)", appErr.Code, wantCode, appErr.Message)
	}
}

func ptr[T any](v T) *T { return &v }

func TestEmployeeService_Create(t *testing.T) {
	tests := []struct {
		name     string
		modify   func(r *models.CreateEmployeeRequest)
		wantCode int
	}{
		{name: "valid request", modify: func(r *models.CreateEmployeeRequest) {}},
		{name: "empty name", modify: func(r *models.CreateEmployeeRequest) { r.Name = "" }, wantCode: http.StatusUnprocessableEntity},
		{name: "blank name", modify: func(r *models.CreateEmployeeRequest) { r.Name = "   " }, wantCode: http.StatusUnprocessableEntity},
		{name: "age is zero", modify: func(r *models.CreateEmployeeRequest) { r.Age = 0 }, wantCode: http.StatusUnprocessableEntity},
		{name: "negative salary", modify: func(r *models.CreateEmployeeRequest) { r.Salary = -1 }, wantCode: http.StatusUnprocessableEntity},
		{name: "missing department", modify: func(r *models.CreateEmployeeRequest) { r.DepartmentID = 0 }, wantCode: http.StatusUnprocessableEntity},
		{name: "department does not exist", modify: func(r *models.CreateEmployeeRequest) { r.DepartmentID = 99 }, wantCode: http.StatusUnprocessableEntity},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, _ := newTestService(t)
			req := validCreateRequest()
			tt.modify(&req)

			got, err := svc.Create(context.Background(), req)

			assertCode(t, err, tt.wantCode)
			if tt.wantCode == 0 {
				if got.ID == 0 {
					t.Error("expected repository to assign an ID")
				}
				if got.Name != req.Name || got.Salary != req.Salary {
					t.Errorf("got %+v, want fields from %+v", got, req)
				}
			}
		})
	}
}

func TestEmployeeService_Create_RepositoryErrors(t *testing.T) {
	t.Run("timeout becomes 503", func(t *testing.T) {
		svc, empRepo := newTestService(t)
		empRepo.err = context.DeadlineExceeded

		_, err := svc.Create(context.Background(), validCreateRequest())

		assertCode(t, err, http.StatusServiceUnavailable)
	})

	t.Run("unknown error is passed through", func(t *testing.T) {
		svc, empRepo := newTestService(t)
		boom := errors.New("connection reset")
		empRepo.err = boom

		_, err := svc.Create(context.Background(), validCreateRequest())

		if !errors.Is(err, boom) {
			t.Fatalf("err = %v, want %v", err, boom)
		}
	})
}

func TestEmployeeService_Get(t *testing.T) {
	svc, _ := newTestService(t)
	created := mustCreate(t, svc, validCreateRequest())

	t.Run("existing employee", func(t *testing.T) {
		got, err := svc.Get(context.Background(), created.ID)

		assertCode(t, err, 0)
		if got.ID != created.ID {
			t.Errorf("ID = %d, want %d", got.ID, created.ID)
		}
	})

	t.Run("unknown id returns 404", func(t *testing.T) {
		_, err := svc.Get(context.Background(), 999)

		assertCode(t, err, http.StatusNotFound)
	})
}

func TestEmployeeService_List(t *testing.T) {
	svc, _ := newTestService(t)
	// 12 employees in department 1 and 3 in department 2.
	for i := range 15 {
		req := validCreateRequest()
		if i >= 12 {
			req.DepartmentID = 2
		}
		mustCreate(t, svc, req)
	}

	tests := []struct {
		name      string
		filter    models.ListEmployeesFilter
		wantCount int
		wantTotal int
		wantFirst int64
	}{
		{name: "default limit", filter: models.ListEmployeesFilter{}, wantCount: 10, wantTotal: 15, wantFirst: 1},
		{name: "second page", filter: models.ListEmployeesFilter{Limit: 10, Offset: 10}, wantCount: 5, wantTotal: 15, wantFirst: 11},
		{name: "limit is capped", filter: models.ListEmployeesFilter{Limit: 1000}, wantCount: 15, wantTotal: 15, wantFirst: 1},
		{name: "negative offset starts at 0", filter: models.ListEmployeesFilter{Limit: 5, Offset: -3}, wantCount: 5, wantTotal: 15, wantFirst: 1},
		{name: "offset past the end", filter: models.ListEmployeesFilter{Limit: 10, Offset: 100}, wantCount: 0, wantTotal: 15},
		{name: "filter by department", filter: models.ListEmployeesFilter{DepartmentID: ptr(int64(2))}, wantCount: 3, wantTotal: 3, wantFirst: 13},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := svc.List(context.Background(), tt.filter)

			assertCode(t, err, 0)
			if got.Employees == nil {
				t.Fatal("Employees is nil; it must be an empty slice so JSON renders []")
			}
			if len(got.Employees) != tt.wantCount {
				t.Errorf("len(Employees) = %d, want %d", len(got.Employees), tt.wantCount)
			}
			if got.TotalCount != tt.wantTotal {
				t.Errorf("TotalCount = %d, want %d", got.TotalCount, tt.wantTotal)
			}
			if tt.wantCount > 0 && got.Employees[0].ID != tt.wantFirst {
				t.Errorf("first ID = %d, want %d", got.Employees[0].ID, tt.wantFirst)
			}
		})
	}
}

func TestEmployeeService_Update(t *testing.T) {
	t.Run("partial update keeps fields that were not sent", func(t *testing.T) {
		svc, _ := newTestService(t)
		created := mustCreate(t, svc, validCreateRequest())

		got, err := svc.Update(context.Background(), created.ID, models.UpdateEmployeeRequest{Salary: ptr(3000.0)})

		assertCode(t, err, 0)
		if got.Salary != 3000 {
			t.Errorf("Salary = %v, want 3000", got.Salary)
		}
		if got.Name != created.Name || got.Age != created.Age || got.Position != created.Position {
			t.Errorf("unchanged fields were modified: got %+v, created %+v", got, created)
		}
	})

	t.Run("move to another existing department", func(t *testing.T) {
		svc, _ := newTestService(t)
		created := mustCreate(t, svc, validCreateRequest())

		got, err := svc.Update(context.Background(), created.ID, models.UpdateEmployeeRequest{DepartmentID: ptr(int64(2))})

		assertCode(t, err, 0)
		if got.DepartmentID != 2 {
			t.Errorf("DepartmentID = %d, want 2", got.DepartmentID)
		}
	})

	tests := []struct {
		name     string
		id       int64
		req      models.UpdateEmployeeRequest
		wantCode int
	}{
		{name: "salary set to zero", id: 1, req: models.UpdateEmployeeRequest{Salary: ptr(0.0)}, wantCode: http.StatusUnprocessableEntity},
		{name: "blank name", id: 1, req: models.UpdateEmployeeRequest{Name: ptr("  ")}, wantCode: http.StatusUnprocessableEntity},
		{name: "department does not exist", id: 1, req: models.UpdateEmployeeRequest{DepartmentID: ptr(int64(99))}, wantCode: http.StatusUnprocessableEntity},
		{name: "employee does not exist", id: 999, req: models.UpdateEmployeeRequest{Age: ptr(30)}, wantCode: http.StatusNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, _ := newTestService(t)
			mustCreate(t, svc, validCreateRequest())

			_, err := svc.Update(context.Background(), tt.id, tt.req)

			assertCode(t, err, tt.wantCode)
		})
	}
}

func TestEmployeeService_Delete(t *testing.T) {
	svc, _ := newTestService(t)
	created := mustCreate(t, svc, validCreateRequest())

	assertCode(t, svc.Delete(context.Background(), created.ID), 0)

	t.Run("deleted employee is no longer found", func(t *testing.T) {
		_, err := svc.Get(context.Background(), created.ID)
		assertCode(t, err, http.StatusNotFound)
	})

	t.Run("deleting twice returns 404", func(t *testing.T) {
		assertCode(t, svc.Delete(context.Background(), created.ID), http.StatusNotFound)
	})
}

func TestEmployeeService_Search(t *testing.T) {
	svc, _ := newTestService(t)
	for _, e := range []struct{ name, position string }{
		{"Nguyen Van An", "Backend Engineer"},
		{"Tran Thi Binh", "Designer"},
		{"Le Van Cuong", "Frontend Engineer"},
	} {
		req := validCreateRequest()
		req.Name, req.Position = e.name, e.position
		mustCreate(t, svc, req)
	}

	tests := []struct {
		name      string
		keyword   string
		wantCount int
		wantCode  int
	}{
		{name: "matches name", keyword: "binh", wantCount: 1},
		{name: "matches position", keyword: "engineer", wantCount: 2},
		{name: "case insensitive and trimmed", keyword: "  NGUYEN ", wantCount: 1},
		{name: "no match returns empty list", keyword: "xyz", wantCount: 0},
		{name: "empty keyword", keyword: "   ", wantCode: http.StatusBadRequest},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := svc.Search(context.Background(), tt.keyword)

			assertCode(t, err, tt.wantCode)
			if tt.wantCode != 0 {
				return
			}
			if got == nil {
				t.Fatal("result is nil; it must be an empty slice so JSON renders []")
			}
			if len(got) != tt.wantCount {
				t.Errorf("len = %d, want %d", len(got), tt.wantCount)
			}
		})
	}
}
