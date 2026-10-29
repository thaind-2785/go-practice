package repositories

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/thaind-2785/go-practice/internal/models"
)

func openTestDB(t *testing.T) *sql.DB {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL is not set; skipping Postgres integration tests")
	}

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	if _, err := db.Exec(`TRUNCATE employees, departments RESTART IDENTITY CASCADE`); err != nil {
		t.Fatalf("reset tables: %v", err)
	}
	return db
}

func newTestEmployeeRepo(t *testing.T, db *sql.DB) *PostgresEmployeeRepo {
	t.Helper()
	repo, err := NewPostgresEmployeeRepo(context.Background(), db)
	if err != nil {
		t.Fatalf("new employee repo: %v", err)
	}
	t.Cleanup(func() { repo.Close() })
	return repo
}

func mustCreateDepartment(t *testing.T, repo *PostgresDepartmentRepo, name string) models.Department {
	t.Helper()
	d := models.Department{Name: name}
	if err := repo.Create(context.Background(), &d); err != nil {
		t.Fatalf("create department %q: %v", name, err)
	}
	return d
}

func mustCreateEmployee(t *testing.T, repo *PostgresEmployeeRepo, name, position string, deptID int64) models.Employee {
	t.Helper()
	e := models.Employee{Name: name, Age: 28, Position: position, DepartmentID: deptID, Salary: 2000.5}
	if err := repo.Create(context.Background(), &e); err != nil {
		t.Fatalf("create employee %q: %v", name, err)
	}
	return e
}

func TestPostgresDepartmentRepo(t *testing.T) {
	ctx := context.Background()
	repo := NewPostgresDepartmentRepo(openTestDB(t))

	eng := mustCreateDepartment(t, repo, "Engineering")
	mustCreateDepartment(t, repo, "Sales")
	if eng.ID == 0 || eng.CreatedAt.IsZero() {
		t.Fatalf("Create did not fill ID/CreatedAt: %+v", eng)
	}

	t.Run("duplicate name returns ErrDuplicate", func(t *testing.T) {
		err := repo.Create(ctx, &models.Department{Name: "Engineering"})
		if !errors.Is(err, ErrDuplicate) {
			t.Fatalf("err = %v, want ErrDuplicate", err)
		}
	})

	t.Run("get by id", func(t *testing.T) {
		got, err := repo.GetByID(ctx, eng.ID)
		if err != nil {
			t.Fatalf("GetByID: %v", err)
		}
		if got.Name != "Engineering" {
			t.Errorf("Name = %q, want Engineering", got.Name)
		}
	})

	t.Run("unknown id returns ErrNotFound", func(t *testing.T) {
		if _, err := repo.GetByID(ctx, 999); !errors.Is(err, ErrNotFound) {
			t.Fatalf("err = %v, want ErrNotFound", err)
		}
	})

	t.Run("list is ordered by id", func(t *testing.T) {
		got, err := repo.List(ctx)
		if err != nil {
			t.Fatalf("List: %v", err)
		}
		if len(got) != 2 || got[0].Name != "Engineering" || got[1].Name != "Sales" {
			t.Errorf("List = %+v", got)
		}
	})
}

func TestPostgresEmployeeRepo_CRUD(t *testing.T) {
	ctx := context.Background()
	db := openTestDB(t)
	deptRepo := NewPostgresDepartmentRepo(db)
	repo := newTestEmployeeRepo(t, db)
	eng := mustCreateDepartment(t, deptRepo, "Engineering")
	sales := mustCreateDepartment(t, deptRepo, "Sales")

	created := mustCreateEmployee(t, repo, "Nguyen Van A", "Backend Engineer", eng.ID)
	if created.ID == 0 || created.CreatedAt.IsZero() || created.UpdatedAt.IsZero() {
		t.Fatalf("Create did not fill generated columns: %+v", created)
	}

	t.Run("unknown department returns ErrForeignKey", func(t *testing.T) {
		e := models.Employee{Name: "B", Age: 30, Position: "Dev", DepartmentID: 999, Salary: 1000}
		if err := repo.Create(ctx, &e); !errors.Is(err, ErrForeignKey) {
			t.Fatalf("err = %v, want ErrForeignKey", err)
		}
	})

	t.Run("get by id reads every column", func(t *testing.T) {
		got, err := repo.GetByID(ctx, created.ID)
		if err != nil {
			t.Fatalf("GetByID: %v", err)
		}
		if got.Name != created.Name || got.Age != created.Age || got.Position != created.Position ||
			got.DepartmentID != created.DepartmentID || got.Salary != created.Salary {
			t.Errorf("got %+v, want %+v", got, created)
		}
	})

	t.Run("update saves all fields and bumps updated_at", func(t *testing.T) {
		e := created
		e.Salary = 3500.75
		e.DepartmentID = sales.ID
		if err := repo.Update(ctx, &e); err != nil {
			t.Fatalf("Update: %v", err)
		}
		if !e.UpdatedAt.After(created.UpdatedAt) {
			t.Errorf("UpdatedAt = %v, want after %v", e.UpdatedAt, created.UpdatedAt)
		}

		got, err := repo.GetByID(ctx, created.ID)
		if err != nil {
			t.Fatalf("GetByID: %v", err)
		}
		if got.Salary != 3500.75 || got.DepartmentID != sales.ID {
			t.Errorf("after update got %+v", got)
		}
	})

	t.Run("update with unknown department returns ErrForeignKey", func(t *testing.T) {
		e := created
		e.DepartmentID = 999
		if err := repo.Update(ctx, &e); !errors.Is(err, ErrForeignKey) {
			t.Fatalf("err = %v, want ErrForeignKey", err)
		}
	})

	t.Run("soft delete hides the employee", func(t *testing.T) {
		if err := repo.Delete(ctx, created.ID); err != nil {
			t.Fatalf("Delete: %v", err)
		}
		if _, err := repo.GetByID(ctx, created.ID); !errors.Is(err, ErrNotFound) {
			t.Errorf("GetByID after delete: err = %v, want ErrNotFound", err)
		}
		if err := repo.Delete(ctx, created.ID); !errors.Is(err, ErrNotFound) {
			t.Errorf("second Delete: err = %v, want ErrNotFound", err)
		}
		e := created
		if err := repo.Update(ctx, &e); !errors.Is(err, ErrNotFound) {
			t.Errorf("Update after delete: err = %v, want ErrNotFound", err)
		}

		var deletedAt *time.Time
		if err := db.QueryRow(`SELECT deleted_at FROM employees WHERE id = $1`, created.ID).Scan(&deletedAt); err != nil {
			t.Fatalf("read deleted_at: %v", err)
		}
		if deletedAt == nil {
			t.Error("row should still exist with deleted_at set")
		}
	})
}

func TestPostgresEmployeeRepo_Queries(t *testing.T) {
	ctx := context.Background()
	db := openTestDB(t)
	deptRepo := NewPostgresDepartmentRepo(db)
	repo := newTestEmployeeRepo(t, db)
	eng := mustCreateDepartment(t, deptRepo, "Engineering")
	design := mustCreateDepartment(t, deptRepo, "Design")

	mustCreateEmployee(t, repo, "Nguyen Van An", "Backend Engineer", eng.ID)
	mustCreateEmployee(t, repo, "Le Van Cuong", "Frontend Engineer", eng.ID)
	mustCreateEmployee(t, repo, "Pham Thi Dung", "QA Engineer", eng.ID)
	mustCreateEmployee(t, repo, "Tran Thi Binh", "Designer", design.ID)
	deleted := mustCreateEmployee(t, repo, "Deleted Engineer", "Backend Engineer", eng.ID)
	if err := repo.Delete(ctx, deleted.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	listTests := []struct {
		name      string
		filter    models.ListEmployeesFilter
		wantNames []string
		wantTotal int
	}{
		{
			name:      "first page",
			filter:    models.ListEmployeesFilter{Limit: 2},
			wantNames: []string{"Nguyen Van An", "Le Van Cuong"},
			wantTotal: 4,
		},
		{
			name:      "second page",
			filter:    models.ListEmployeesFilter{Limit: 2, Offset: 2},
			wantNames: []string{"Pham Thi Dung", "Tran Thi Binh"},
			wantTotal: 4,
		},
		{
			name:      "offset past the end",
			filter:    models.ListEmployeesFilter{Limit: 2, Offset: 10},
			wantNames: nil,
			wantTotal: 4,
		},
		{
			name:      "filter by department",
			filter:    models.ListEmployeesFilter{Limit: 10, DepartmentID: &design.ID},
			wantNames: []string{"Tran Thi Binh"},
			wantTotal: 1,
		},
	}
	for _, tt := range listTests {
		t.Run("list/"+tt.name, func(t *testing.T) {
			got, total, err := repo.List(ctx, tt.filter)
			if err != nil {
				t.Fatalf("List: %v", err)
			}
			if total != tt.wantTotal {
				t.Errorf("total = %d, want %d", total, tt.wantTotal)
			}
			assertNames(t, got, tt.wantNames)
		})
	}

	t.Run("search matches name or position, case-insensitive", func(t *testing.T) {
		got, err := repo.Search(ctx, "ENGINEER")
		if err != nil {
			t.Fatalf("Search: %v", err)
		}
		assertNames(t, got, []string{"Nguyen Van An", "Le Van Cuong", "Pham Thi Dung"})

		got, err = repo.Search(ctx, "binh")
		if err != nil {
			t.Fatalf("Search: %v", err)
		}
		assertNames(t, got, []string{"Tran Thi Binh"})
	})

	t.Run("list by department joins the department name", func(t *testing.T) {
		got, err := repo.ListByDepartment(ctx, eng.ID)
		if err != nil {
			t.Fatalf("ListByDepartment: %v", err)
		}
		assertNames(t, got, []string{"Nguyen Van An", "Le Van Cuong", "Pham Thi Dung"})
		for _, e := range got {
			if e.DepartmentName != "Engineering" {
				t.Errorf("%s: DepartmentName = %q, want Engineering", e.Name, e.DepartmentName)
			}
		}
	})

	t.Run("list all skips deleted employees", func(t *testing.T) {
		got, err := repo.ListAll(ctx)
		if err != nil {
			t.Fatalf("ListAll: %v", err)
		}
		assertNames(t, got, []string{"Nguyen Van An", "Le Van Cuong", "Pham Thi Dung", "Tran Thi Binh"})
	})
}

func TestPostgresEmployeeRepo_ContextTimeout(t *testing.T) {
	db := openTestDB(t)
	repo := newTestEmployeeRepo(t, db)

	ctx, cancel := context.WithTimeout(context.Background(), time.Nanosecond)
	defer cancel()
	<-ctx.Done()

	_, err := repo.GetByID(ctx, 1)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("err = %v, want context.DeadlineExceeded", err)
	}
}

func assertNames(t *testing.T, got []models.Employee, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("got %d employees %v, want %v", len(got), names(got), want)
	}
	for i := range want {
		if got[i].Name != want[i] {
			t.Fatalf("got %v, want %v", names(got), want)
		}
	}
}

func names(employees []models.Employee) []string {
	result := make([]string, len(employees))
	for i, e := range employees {
		result[i] = e.Name
	}
	return result
}
