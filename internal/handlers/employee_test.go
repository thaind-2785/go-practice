package handlers

import (
	"errors"
	"net/http"
	"testing"

	"github.com/thaind-2785/go-practice/internal/models"
	"github.com/thaind-2785/go-practice/internal/utils"
)

func newEmployeeMux(t *testing.T, svc *fakeEmployeeService) *http.ServeMux {
	t.Helper()
	return newMux(t, svc, &fakeDepartmentService{}, &fakeExportService{})
}

func TestEmployeeHandler_Get(t *testing.T) {
	tests := []struct {
		name        string
		target      string
		serviceErr  error
		wantStatus  int
		wantMessage string
	}{
		{name: "found", target: "/employees/7", wantStatus: http.StatusOK},
		{name: "non-numeric id", target: "/employees/abc", wantStatus: http.StatusBadRequest, wantMessage: "invalid id"},
		{name: "zero id", target: "/employees/0", wantStatus: http.StatusBadRequest, wantMessage: "invalid id"},
		{name: "negative id", target: "/employees/-1", wantStatus: http.StatusBadRequest, wantMessage: "invalid id"},
		{name: "not found", target: "/employees/7", serviceErr: utils.NewNotFound("Employee not found"), wantStatus: http.StatusNotFound, wantMessage: "Employee not found"},
		{name: "database timeout", target: "/employees/7", serviceErr: utils.NewServiceUnavailable("database timeout"), wantStatus: http.StatusServiceUnavailable, wantMessage: "database timeout"},
		{name: "unexpected error is hidden", target: "/employees/7", serviceErr: errors.New("pq: connection reset"), wantStatus: http.StatusInternalServerError, wantMessage: "internal server error"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &fakeEmployeeService{employee: &models.Employee{ID: 7, Name: "Nguyen Van A"}, err: tt.serviceErr}

			rec := serve(newEmployeeMux(t, svc), http.MethodGet, tt.target, "")

			if tt.wantStatus != http.StatusOK {
				assertError(t, rec, tt.wantStatus, tt.wantMessage)
				return
			}
			assertStatus(t, rec, http.StatusOK)
			if got := decode[models.Employee](t, rec); got.ID != 7 || got.Name != "Nguyen Van A" {
				t.Errorf("body = %+v", got)
			}
			if svc.gotID != 7 {
				t.Errorf("service got id %d, want 7", svc.gotID)
			}
		})
	}
}

func TestEmployeeHandler_Create(t *testing.T) {
	valid := `{"name":"Nguyen Van A","age":28,"position":"Backend","departmentId":1,"salary":2000}`

	tests := []struct {
		name        string
		body        string
		serviceErr  error
		wantStatus  int
		wantMessage string
	}{
		{name: "valid", body: valid, wantStatus: http.StatusCreated},
		{name: "malformed JSON", body: `{"name":`, wantStatus: http.StatusBadRequest, wantMessage: "invalid request body"},
		{name: "empty body", body: "", wantStatus: http.StatusBadRequest, wantMessage: "invalid request body"},
		{name: "unknown field", body: `{"name":"A","nickname":"x"}`, wantStatus: http.StatusBadRequest, wantMessage: "unknown field"},
		{name: "wrong type", body: `{"age":"twenty"}`, wantStatus: http.StatusBadRequest, wantMessage: "invalid request body"},
		{name: "validation error", body: valid, serviceErr: utils.NewValidation("age must be greater than 0"), wantStatus: http.StatusUnprocessableEntity, wantMessage: "age must be greater than 0"},
		{name: "department missing", body: valid, serviceErr: utils.NewValidation("department does not exist"), wantStatus: http.StatusUnprocessableEntity, wantMessage: "department does not exist"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &fakeEmployeeService{employee: &models.Employee{ID: 1, Name: "Nguyen Van A"}, err: tt.serviceErr}

			rec := serve(newEmployeeMux(t, svc), http.MethodPost, "/employees", tt.body)

			if tt.wantStatus != http.StatusCreated {
				assertError(t, rec, tt.wantStatus, tt.wantMessage)
				return
			}
			assertStatus(t, rec, http.StatusCreated)
			want := models.CreateEmployeeRequest{Name: "Nguyen Van A", Age: 28, Position: "Backend", DepartmentID: 1, Salary: 2000}
			if svc.gotCreate != want {
				t.Errorf("service got %+v, want %+v", svc.gotCreate, want)
			}
		})
	}
}

func TestEmployeeHandler_List(t *testing.T) {
	tests := []struct {
		name        string
		target      string
		wantStatus  int
		wantMessage string
		wantFilter  models.ListEmployeesFilter
	}{
		{name: "no query", target: "/employees", wantStatus: http.StatusOK},
		{name: "pagination and department", target: "/employees?limit=5&offset=10&departmentId=2", wantStatus: http.StatusOK, wantFilter: models.ListEmployeesFilter{Limit: 5, Offset: 10, DepartmentID: ptr(int64(2))}},
		{name: "invalid limit", target: "/employees?limit=abc", wantStatus: http.StatusBadRequest, wantMessage: "limit"},
		{name: "invalid offset", target: "/employees?offset=1.5", wantStatus: http.StatusBadRequest, wantMessage: "offset"},
		{name: "invalid departmentId", target: "/employees?departmentId=x", wantStatus: http.StatusBadRequest, wantMessage: "departmentId"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &fakeEmployeeService{list: &models.ListEmployeesResult{TotalCount: 42, Employees: []models.Employee{{ID: 1}}}}

			rec := serve(newEmployeeMux(t, svc), http.MethodGet, tt.target, "")

			if tt.wantStatus != http.StatusOK {
				assertError(t, rec, tt.wantStatus, tt.wantMessage)
				return
			}
			assertStatus(t, rec, http.StatusOK)
			if got := decode[models.ListEmployeesResult](t, rec); got.TotalCount != 42 || len(got.Employees) != 1 {
				t.Errorf("body = %+v, want totalCount 42 and 1 employee", got)
			}
			got := svc.gotFilter
			if got.Limit != tt.wantFilter.Limit || got.Offset != tt.wantFilter.Offset {
				t.Errorf("filter = %+v, want %+v", got, tt.wantFilter)
			}
			if (got.DepartmentID == nil) != (tt.wantFilter.DepartmentID == nil) ||
				(got.DepartmentID != nil && *got.DepartmentID != *tt.wantFilter.DepartmentID) {
				t.Errorf("filter departmentId = %v, want %v", got.DepartmentID, tt.wantFilter.DepartmentID)
			}
		})
	}
}

func TestEmployeeHandler_Update(t *testing.T) {
	t.Run("partial body only sets sent fields", func(t *testing.T) {
		svc := &fakeEmployeeService{employee: &models.Employee{ID: 3, Salary: 3000}}

		rec := serve(newEmployeeMux(t, svc), http.MethodPut, "/employees/3", `{"salary":3000}`)

		assertStatus(t, rec, http.StatusOK)
		if svc.gotID != 3 {
			t.Errorf("service got id %d, want 3", svc.gotID)
		}
		if svc.gotUpdate.Salary == nil || *svc.gotUpdate.Salary != 3000 {
			t.Errorf("salary = %v, want 3000", svc.gotUpdate.Salary)
		}
		if svc.gotUpdate.Name != nil || svc.gotUpdate.Age != nil || svc.gotUpdate.Position != nil || svc.gotUpdate.DepartmentID != nil {
			t.Errorf("fields not in the body must stay nil: %+v", svc.gotUpdate)
		}
	})

	tests := []struct {
		name        string
		target      string
		body        string
		serviceErr  error
		wantStatus  int
		wantMessage string
	}{
		{name: "invalid id", target: "/employees/abc", body: `{"salary":1}`, wantStatus: http.StatusBadRequest, wantMessage: "invalid id"},
		{name: "malformed body", target: "/employees/3", body: `salary=1`, wantStatus: http.StatusBadRequest, wantMessage: "invalid request body"},
		{name: "not found", target: "/employees/3", body: `{"salary":1}`, serviceErr: utils.NewNotFound("Employee not found"), wantStatus: http.StatusNotFound, wantMessage: "Employee not found"},
		{name: "validation error", target: "/employees/3", body: `{"salary":-1}`, serviceErr: utils.NewValidation("salary must be greater than 0"), wantStatus: http.StatusUnprocessableEntity, wantMessage: "salary"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &fakeEmployeeService{err: tt.serviceErr}

			rec := serve(newEmployeeMux(t, svc), http.MethodPut, tt.target, tt.body)

			assertError(t, rec, tt.wantStatus, tt.wantMessage)
		})
	}
}

func TestEmployeeHandler_Delete(t *testing.T) {
	t.Run("success has no body", func(t *testing.T) {
		svc := &fakeEmployeeService{}

		rec := serve(newEmployeeMux(t, svc), http.MethodDelete, "/employees/5", "")

		assertStatus(t, rec, http.StatusNoContent)
		if rec.Body.Len() != 0 {
			t.Errorf("body = %q, want empty", rec.Body.String())
		}
		if svc.gotID != 5 {
			t.Errorf("service got id %d, want 5", svc.gotID)
		}
	})

	t.Run("not found", func(t *testing.T) {
		svc := &fakeEmployeeService{err: utils.NewNotFound("Employee not found")}

		rec := serve(newEmployeeMux(t, svc), http.MethodDelete, "/employees/5", "")

		assertError(t, rec, http.StatusNotFound, "Employee not found")
	})
}

func TestEmployeeHandler_Search(t *testing.T) {
	t.Run("keyword is passed to the service", func(t *testing.T) {
		svc := &fakeEmployeeService{employees: []models.Employee{{ID: 1, Name: "Nguyen Van A"}}}

		rec := serve(newEmployeeMux(t, svc), http.MethodGet, "/employees/search?keyword=nguyen", "")

		assertStatus(t, rec, http.StatusOK)
		if svc.gotKeyword != "nguyen" {
			t.Errorf("keyword = %q, want nguyen", svc.gotKeyword)
		}
		if got := decode[[]models.Employee](t, rec); len(got) != 1 {
			t.Errorf("body = %+v, want 1 employee", got)
		}
	})

	t.Run("missing keyword", func(t *testing.T) {
		svc := &fakeEmployeeService{err: utils.NewBadRequest("keyword is required")}

		rec := serve(newEmployeeMux(t, svc), http.MethodGet, "/employees/search", "")

		assertError(t, rec, http.StatusBadRequest, "keyword is required")
	})
}

func TestEmployeeHandler_MethodNotAllowed(t *testing.T) {
	rec := serve(newEmployeeMux(t, &fakeEmployeeService{}), http.MethodPatch, "/employees/1", `{}`)

	assertStatus(t, rec, http.StatusMethodNotAllowed)
}
