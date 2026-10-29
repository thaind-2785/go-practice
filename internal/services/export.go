package services

import (
	"cmp"
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"sync"
	"time"

	"github.com/thaind-2785/go-practice/internal/models"
	"github.com/thaind-2785/go-practice/internal/repositories"
)

type exporter struct {
	format string
	write  func(w io.Writer, employees []models.Employee) error
}

var exporters = []exporter{
	{format: "json", write: writeEmployeesJSON},
	{format: "csv", write: writeEmployeesCSV},
}

type ExportService struct {
	empRepo repositories.EmployeeRepository
	dir     string
}

func NewExportService(e repositories.EmployeeRepository, dir string) *ExportService {
	return &ExportService{empRepo: e, dir: dir}
}

func (s *ExportService) Export(ctx context.Context) (*models.ExportResult, error) {
	start := time.Now()

	employees, err := s.empRepo.ListAll(ctx)
	if err != nil {
		return nil, mapRepoError(err, "")
	}
	if employees == nil {
		employees = []models.Employee{}
	}

	if err := os.MkdirAll(s.dir, 0o755); err != nil {
		return nil, fmt.Errorf("create export dir: %w", err)
	}

	var (
		wg    sync.WaitGroup
		mu    sync.Mutex
		files []models.ExportFile
		errs  []error
	)

	for _, ex := range exporters {
		wg.Go(func() {
			file, err := s.writeFile(ex, employees)

			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				errs = append(errs, err)
				return
			}
			files = append(files, file)
		})
	}

	wg.Wait()

	if err := errors.Join(errs...); err != nil {
		return nil, err
	}

	slices.SortFunc(files, func(a, b models.ExportFile) int {
		return cmp.Compare(a.Format, b.Format)
	})

	return &models.ExportResult{
		TotalEmployees: len(employees),
		Files:          files,
		Duration:       time.Since(start).Round(time.Microsecond).String(),
	}, nil
}

func (s *ExportService) writeFile(ex exporter, employees []models.Employee) (models.ExportFile, error) {
	path := filepath.Join(s.dir, "employees."+ex.format)

	tmp, err := os.CreateTemp(s.dir, "employees-*."+ex.format+".tmp")
	if err != nil {
		return models.ExportFile{}, fmt.Errorf("export %s: %w", ex.format, err)
	}
	defer os.Remove(tmp.Name())

	if err := ex.write(tmp, employees); err != nil {
		tmp.Close()
		return models.ExportFile{}, fmt.Errorf("export %s: %w", ex.format, err)
	}
	if err := tmp.Close(); err != nil {
		return models.ExportFile{}, fmt.Errorf("export %s: %w", ex.format, err)
	}
	if err := os.Rename(tmp.Name(), path); err != nil {
		return models.ExportFile{}, fmt.Errorf("export %s: %w", ex.format, err)
	}

	return models.ExportFile{Format: ex.format, Path: path}, nil
}

func writeEmployeesJSON(w io.Writer, employees []models.Employee) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(employees)
}

func writeEmployeesCSV(w io.Writer, employees []models.Employee) error {
	cw := csv.NewWriter(w)
	if err := cw.Write([]string{"id", "name", "age", "position", "departmentId", "salary", "createdAt", "updatedAt"}); err != nil {
		return err
	}
	for _, e := range employees {
		record := []string{
			strconv.FormatInt(e.ID, 10),
			e.Name,
			strconv.Itoa(e.Age),
			e.Position,
			strconv.FormatInt(e.DepartmentID, 10),
			strconv.FormatFloat(e.Salary, 'f', 2, 64),
			e.CreatedAt.Format(time.RFC3339),
			e.UpdatedAt.Format(time.RFC3339),
		}
		if err := cw.Write(record); err != nil {
			return err
		}
	}
	cw.Flush()
	return cw.Error()
}
