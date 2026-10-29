package handlers

import (
	"errors"
	"net/http"
	"testing"

	"github.com/thaind-2785/go-practice/internal/models"
	"github.com/thaind-2785/go-practice/internal/utils"
)

func newDepartmentMux(t *testing.T, svc *fakeDepartmentService) *http.ServeMux {
	t.Helper()
	return newMux(t, &fakeEmployeeService{}, svc, &fakeExportService{})
}

func TestDepartmentHandler_Create(t *testing.T) {
	tests := []struct {
		name        string
		body        string
		serviceErr  error
		wantStatus  int
		wantMessage string
	}{
		{name: "valid", body: `{"name":"Engineering"}`, wantStatus: http.StatusCreated},
		{name: "malformed JSON", body: `{"name"`, wantStatus: http.StatusBadRequest, wantMessage: "invalid request body"},
		{name: "blank name", body: `{"name":" "}`, serviceErr: utils.NewValidation("name is required"), wantStatus: http.StatusUnprocessableEntity, wantMessage: "name is required"},
		{name: "duplicate name", body: `{"name":"Engineering"}`, serviceErr: utils.NewConflict("department name already exists"), wantStatus: http.StatusConflict, wantMessage: "already exists"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &fakeDepartmentService{department: &models.Department{ID: 1, Name: "Engineering"}, err: tt.serviceErr}

			rec := serve(newDepartmentMux(t, svc), http.MethodPost, "/departments", tt.body)

			if tt.wantStatus != http.StatusCreated {
				assertError(t, rec, tt.wantStatus, tt.wantMessage)
				return
			}
			assertStatus(t, rec, http.StatusCreated)
			if svc.gotCreate.Name != "Engineering" {
				t.Errorf("service got %+v", svc.gotCreate)
			}
			if got := decode[models.Department](t, rec); got.ID != 1 {
				t.Errorf("body = %+v", got)
			}
		})
	}
}

func TestDepartmentHandler_List(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		svc := &fakeDepartmentService{departments: []models.Department{{ID: 1, Name: "Engineering"}, {ID: 2, Name: "Sales"}}}

		rec := serve(newDepartmentMux(t, svc), http.MethodGet, "/departments", "")

		assertStatus(t, rec, http.StatusOK)
		if got := decode[[]models.Department](t, rec); len(got) != 2 {
			t.Errorf("body = %+v, want 2 departments", got)
		}
	})

	t.Run("unexpected error", func(t *testing.T) {
		svc := &fakeDepartmentService{err: errors.New("connection refused")}

		rec := serve(newDepartmentMux(t, svc), http.MethodGet, "/departments", "")

		assertError(t, rec, http.StatusInternalServerError, "internal server error")
	})
}

func TestDepartmentHandler_ListEmployees(t *testing.T) {
	tests := []struct {
		name        string
		target      string
		serviceErr  error
		wantStatus  int
		wantMessage string
	}{
		{name: "success", target: "/departments/2/employees", wantStatus: http.StatusOK},
		{name: "invalid id", target: "/departments/abc/employees", wantStatus: http.StatusBadRequest, wantMessage: "invalid id"},
		{name: "department not found", target: "/departments/9/employees", serviceErr: utils.NewNotFound("Department not found"), wantStatus: http.StatusNotFound, wantMessage: "Department not found"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &fakeDepartmentService{
				employees: []models.Employee{{ID: 1, DepartmentID: 2, DepartmentName: "Sales"}},
				err:       tt.serviceErr,
			}

			rec := serve(newDepartmentMux(t, svc), http.MethodGet, tt.target, "")

			if tt.wantStatus != http.StatusOK {
				assertError(t, rec, tt.wantStatus, tt.wantMessage)
				return
			}
			assertStatus(t, rec, http.StatusOK)
			if svc.gotID != 2 {
				t.Errorf("service got id %d, want 2", svc.gotID)
			}
			if got := decode[[]models.Employee](t, rec); len(got) != 1 || got[0].DepartmentName != "Sales" {
				t.Errorf("body = %+v", got)
			}
		})
	}
}
