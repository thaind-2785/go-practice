package handlers

import (
	"errors"
	"net/http"
	"testing"

	"github.com/thaind-2785/go-practice/internal/models"
	"github.com/thaind-2785/go-practice/internal/utils"
)

func newExportMux(t *testing.T, svc *fakeExportService) *http.ServeMux {
	t.Helper()
	return newMux(t, &fakeEmployeeService{}, &fakeDepartmentService{}, svc)
}

func TestExportHandler_Export(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		svc := &fakeExportService{result: &models.ExportResult{
			TotalEmployees: 2,
			Files: []models.ExportFile{
				{Format: "csv", Path: "exports/employees.csv"},
				{Format: "json", Path: "exports/employees.json"},
			},
			Duration: "1ms",
		}}

		rec := serve(newExportMux(t, svc), http.MethodPost, "/employees/export", "")

		assertStatus(t, rec, http.StatusOK)
		got := decode[models.ExportResult](t, rec)
		if got.TotalEmployees != 2 || len(got.Files) != 2 {
			t.Errorf("body = %+v", got)
		}
	})

	t.Run("database timeout", func(t *testing.T) {
		svc := &fakeExportService{err: utils.NewServiceUnavailable("database timeout")}

		rec := serve(newExportMux(t, svc), http.MethodPost, "/employees/export", "")

		assertError(t, rec, http.StatusServiceUnavailable, "database timeout")
	})

	t.Run("write failure is hidden", func(t *testing.T) {
		svc := &fakeExportService{err: errors.New("export csv: open /exports: permission denied")}

		rec := serve(newExportMux(t, svc), http.MethodPost, "/employees/export", "")

		assertError(t, rec, http.StatusInternalServerError, "internal server error")
	})
}
