package handlers

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/thaind-2785/go-practice/internal/models"
)

var (
	_ EmployeeService   = (*fakeEmployeeService)(nil)
	_ DepartmentService = (*fakeDepartmentService)(nil)
	_ ExportService     = (*fakeExportService)(nil)
)

type fakeEmployeeService struct {
	employee  *models.Employee
	list      *models.ListEmployeesResult
	employees []models.Employee
	err       error

	gotID      int64
	gotCreate  models.CreateEmployeeRequest
	gotUpdate  models.UpdateEmployeeRequest
	gotFilter  models.ListEmployeesFilter
	gotKeyword string
}

func (f *fakeEmployeeService) Create(ctx context.Context, req models.CreateEmployeeRequest) (*models.Employee, error) {
	f.gotCreate = req
	return f.employee, f.err
}

func (f *fakeEmployeeService) Get(ctx context.Context, id int64) (*models.Employee, error) {
	f.gotID = id
	return f.employee, f.err
}

func (f *fakeEmployeeService) List(ctx context.Context, filter models.ListEmployeesFilter) (*models.ListEmployeesResult, error) {
	f.gotFilter = filter
	return f.list, f.err
}

func (f *fakeEmployeeService) Update(ctx context.Context, id int64, req models.UpdateEmployeeRequest) (*models.Employee, error) {
	f.gotID = id
	f.gotUpdate = req
	return f.employee, f.err
}

func (f *fakeEmployeeService) Delete(ctx context.Context, id int64) error {
	f.gotID = id
	return f.err
}

func (f *fakeEmployeeService) Search(ctx context.Context, keyword string) ([]models.Employee, error) {
	f.gotKeyword = keyword
	return f.employees, f.err
}

type fakeDepartmentService struct {
	department  *models.Department
	departments []models.Department
	employees   []models.Employee
	err         error

	gotID     int64
	gotCreate models.CreateDepartmentRequest
}

func (f *fakeDepartmentService) Create(ctx context.Context, req models.CreateDepartmentRequest) (*models.Department, error) {
	f.gotCreate = req
	return f.department, f.err
}

func (f *fakeDepartmentService) List(ctx context.Context) ([]models.Department, error) {
	return f.departments, f.err
}

func (f *fakeDepartmentService) ListEmployees(ctx context.Context, id int64) ([]models.Employee, error) {
	f.gotID = id
	return f.employees, f.err
}

type fakeExportService struct {
	result *models.ExportResult
	err    error
}

func (f *fakeExportService) Export(ctx context.Context) (*models.ExportResult, error) {
	return f.result, f.err
}

func newMux(t *testing.T, svc *fakeEmployeeService, deptSvc *fakeDepartmentService, exportSvc *fakeExportService) *http.ServeMux {
	t.Helper()
	original := log.Writer()
	log.SetOutput(io.Discard)
	t.Cleanup(func() { log.SetOutput(original) })

	mux := http.NewServeMux()
	NewEmployeeHandler(svc).RegisterRoutes(mux)
	NewDepartmentHandler(deptSvc).RegisterRoutes(mux)
	NewExportHandler(exportSvc).RegisterRoutes(mux)
	return mux
}

func serve(mux http.Handler, method, target, body string) *httptest.ResponseRecorder {
	var r io.Reader
	if body != "" {
		r = strings.NewReader(body)
	}
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(method, target, r))
	return rec
}

func decode[T any](t *testing.T, rec *httptest.ResponseRecorder) T {
	t.Helper()
	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Fatalf("Content-Type = %q, want application/json", ct)
	}
	var v T
	if err := json.Unmarshal(rec.Body.Bytes(), &v); err != nil {
		t.Fatalf("decode body %q: %v", rec.Body.String(), err)
	}
	return v
}

func assertStatus(t *testing.T, rec *httptest.ResponseRecorder, want int) {
	t.Helper()
	if rec.Code != want {
		t.Fatalf("status = %d, want %d (body: %s)", rec.Code, want, rec.Body.String())
	}
}

func assertError(t *testing.T, rec *httptest.ResponseRecorder, wantCode int, wantMessage string) {
	t.Helper()
	assertStatus(t, rec, wantCode)
	body := decode[struct {
		Error string `json:"error"`
		Code  int    `json:"code"`
	}](t, rec)
	if body.Code != wantCode {
		t.Errorf("body code = %d, want %d", body.Code, wantCode)
	}
	if !strings.Contains(body.Error, wantMessage) {
		t.Errorf("body error = %q, want it to contain %q", body.Error, wantMessage)
	}
}

func ptr[T any](v T) *T { return &v }
