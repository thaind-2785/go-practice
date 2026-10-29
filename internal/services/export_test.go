package services

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/thaind-2785/go-practice/internal/models"
)

func newExportTestService(t *testing.T) (*ExportService, *fakeEmployeeRepo, string) {
	t.Helper()
	empRepo := newFakeEmployeeRepo()
	dir := filepath.Join(t.TempDir(), "exports")
	return NewExportService(empRepo, dir), empRepo, dir
}

func readJSONExport(t *testing.T, path string) []models.Employee {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	var employees []models.Employee
	if err := json.Unmarshal(data, &employees); err != nil {
		t.Fatalf("%s is not valid JSON: %v", path, err)
	}
	if employees == nil {
		t.Fatalf("%s contains null, want an array", path)
	}
	return employees
}

func readCSVExport(t *testing.T, path string) [][]string {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("open %s: %v", path, err)
	}
	defer f.Close()
	rows, err := csv.NewReader(f).ReadAll()
	if err != nil {
		t.Fatalf("%s is not valid CSV: %v", path, err)
	}
	return rows
}

func TestExportService_Export(t *testing.T) {
	svc, empRepo, dir := newExportTestService(t)
	ctx := context.Background()
	for _, e := range []models.Employee{
		{Name: "Nguyen Van A", Age: 28, Position: "Backend Engineer", DepartmentID: 1, Salary: 2000},
		{Name: "Tran, Thi B", Age: 30, Position: "Designer", DepartmentID: 2, Salary: 1500.5},
	} {
		if err := empRepo.Create(ctx, &e); err != nil {
			t.Fatalf("seed employee: %v", err)
		}
	}

	result, err := svc.Export(ctx)
	if err != nil {
		t.Fatalf("Export: %v", err)
	}

	if result.TotalEmployees != 2 {
		t.Errorf("TotalEmployees = %d, want 2", result.TotalEmployees)
	}
	wantFiles := []models.ExportFile{
		{Format: "csv", Path: filepath.Join(dir, "employees.csv")},
		{Format: "json", Path: filepath.Join(dir, "employees.json")},
	}
	if len(result.Files) != len(wantFiles) {
		t.Fatalf("Files = %+v, want %+v", result.Files, wantFiles)
	}
	for i, want := range wantFiles {
		if result.Files[i] != want {
			t.Errorf("Files[%d] = %+v, want %+v", i, result.Files[i], want)
		}
	}

	employees := readJSONExport(t, wantFiles[1].Path)
	if len(employees) != 2 || employees[1].Name != "Tran, Thi B" {
		t.Errorf("JSON export = %+v, want both employees", employees)
	}

	rows := readCSVExport(t, wantFiles[0].Path)
	if len(rows) != 3 {
		t.Fatalf("CSV has %d rows, want header + 2", len(rows))
	}
	if rows[0][0] != "id" || rows[0][1] != "name" {
		t.Errorf("CSV header = %v", rows[0])
	}
	if rows[2][1] != "Tran, Thi B" || rows[2][5] != "1500.50" {
		t.Errorf("CSV row = %v, want name %q and salary %q", rows[2], "Tran, Thi B", "1500.50")
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read export dir: %v", err)
	}
	if len(entries) != 2 {
		t.Errorf("export dir has %d entries, want 2", len(entries))
	}
}

func TestExportService_Export_NoEmployees(t *testing.T) {
	svc, _, dir := newExportTestService(t)

	result, err := svc.Export(context.Background())
	if err != nil {
		t.Fatalf("Export: %v", err)
	}

	if result.TotalEmployees != 0 {
		t.Errorf("TotalEmployees = %d, want 0", result.TotalEmployees)
	}
	if employees := readJSONExport(t, filepath.Join(dir, "employees.json")); len(employees) != 0 {
		t.Errorf("JSON export has %d employees, want 0", len(employees))
	}
	if rows := readCSVExport(t, filepath.Join(dir, "employees.csv")); len(rows) != 1 {
		t.Errorf("CSV has %d rows, want only the header", len(rows))
	}
}

func TestExportService_Export_RepositoryError(t *testing.T) {
	svc, empRepo, dir := newExportTestService(t)
	empRepo.err = context.DeadlineExceeded

	_, err := svc.Export(context.Background())

	assertCode(t, err, http.StatusServiceUnavailable)
	if _, statErr := os.Stat(dir); !errors.Is(statErr, os.ErrNotExist) {
		t.Errorf("export dir should not be created when loading employees fails")
	}
}

func TestExportService_Export_Concurrent(t *testing.T) {
	svc, empRepo, dir := newExportTestService(t)
	if err := empRepo.Create(context.Background(), &models.Employee{Name: "A", Age: 20, DepartmentID: 1, Salary: 100}); err != nil {
		t.Fatalf("seed employee: %v", err)
	}

	var wg sync.WaitGroup
	for range 10 {
		wg.Go(func() {
			if _, err := svc.Export(context.Background()); err != nil {
				t.Errorf("Export: %v", err)
			}
		})
	}
	wg.Wait()

	if employees := readJSONExport(t, filepath.Join(dir, "employees.json")); len(employees) != 1 {
		t.Errorf("JSON export has %d employees, want 1", len(employees))
	}
	if rows := readCSVExport(t, filepath.Join(dir, "employees.csv")); len(rows) != 2 {
		t.Errorf("CSV has %d rows, want header + 1", len(rows))
	}
}
