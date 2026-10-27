package repositories

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/thaind-2785/go-practice/internal/models"
)

var _ EmployeeRepository = (*PostgresEmployeeRepo)(nil)

// employeeColumns must stay in the same order as the Scan call in scanEmployee.
const employeeColumns = "id, name, age, position, department_id, salary, created_at, updated_at"

type PostgresEmployeeRepo struct {
	db          *sql.DB
	getByIDStmt *sql.Stmt
}

// NewPostgresEmployeeRepo prepares the statements the repository reuses on every request.
// Call Close when the repository is no longer needed.
func NewPostgresEmployeeRepo(ctx context.Context, db *sql.DB) (*PostgresEmployeeRepo, error) {
	getByIDStmt, err := db.PrepareContext(ctx,
		`SELECT `+employeeColumns+` FROM employees WHERE id = $1 AND deleted_at IS NULL`)
	if err != nil {
		return nil, fmt.Errorf("prepare get employee: %w", err)
	}

	return &PostgresEmployeeRepo{db: db, getByIDStmt: getByIDStmt}, nil
}

func (r *PostgresEmployeeRepo) Close() error {
	return r.getByIDStmt.Close()
}

func (r *PostgresEmployeeRepo) Create(ctx context.Context, e *models.Employee) error {
	const query = `
		INSERT INTO employees (name, age, position, department_id, salary)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at, updated_at`

	err := r.db.QueryRowContext(ctx, query, e.Name, e.Age, e.Position, e.DepartmentID, e.Salary).
		Scan(&e.ID, &e.CreatedAt, &e.UpdatedAt)
	if err != nil {
		return mapPgError("create employee", err)
	}

	return nil
}

func (r *PostgresEmployeeRepo) GetByID(ctx context.Context, id int64) (*models.Employee, error) {
	e, err := scanEmployee(r.getByIDStmt.QueryRowContext(ctx, id))
	if err != nil {
		return nil, mapPgError("get employee", err)
	}

	return &e, nil
}

func (r *PostgresEmployeeRepo) List(ctx context.Context, f models.ListEmployeesFilter) ([]models.Employee, int, error) {
	// When no department filter is given, $1 is NULL and the condition matches every row.
	const where = `WHERE deleted_at IS NULL AND ($1::bigint IS NULL OR department_id = $1)`

	var total int
	err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM employees `+where, f.DepartmentID).Scan(&total)
	if err != nil {
		return nil, 0, mapPgError("count employees", err)
	}

	employees, err := r.queryEmployees(ctx, "list employees",
		`SELECT `+employeeColumns+` FROM employees `+where+` ORDER BY id LIMIT $2 OFFSET $3`,
		f.DepartmentID, f.Limit, f.Offset)
	if err != nil {
		return nil, 0, err
	}

	return employees, total, nil
}

func (r *PostgresEmployeeRepo) Update(ctx context.Context, e *models.Employee) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return mapPgError("update employee: begin", err)
	}
	// Rollback is a no-op once Commit has succeeded, so it is safe to always defer it.
	defer tx.Rollback()

	// FOR UPDATE locks the row: a concurrent Update or Delete of the same employee
	// waits until this transaction commits or rolls back.
	var id int64
	err = tx.QueryRowContext(ctx,
		`SELECT id FROM employees WHERE id = $1 AND deleted_at IS NULL FOR UPDATE`, e.ID).Scan(&id)
	if err != nil {
		return mapPgError("update employee: lock", err)
	}

	const query = `
		UPDATE employees
		SET name = $2, age = $3, position = $4, department_id = $5, salary = $6, updated_at = now()
		WHERE id = $1
		RETURNING updated_at`

	err = tx.QueryRowContext(ctx, query, e.ID, e.Name, e.Age, e.Position, e.DepartmentID, e.Salary).
		Scan(&e.UpdatedAt)
	if err != nil {
		return mapPgError("update employee", err)
	}

	if err := tx.Commit(); err != nil {
		return mapPgError("update employee: commit", err)
	}

	return nil
}

// Delete is a soft delete: the row stays in the table with deleted_at set.
func (r *PostgresEmployeeRepo) Delete(ctx context.Context, id int64) error {
	res, err := r.db.ExecContext(ctx,
		`UPDATE employees SET deleted_at = now(), updated_at = now() WHERE id = $1 AND deleted_at IS NULL`, id)
	if err != nil {
		return mapPgError("delete employee", err)
	}

	n, err := res.RowsAffected()
	if err != nil {
		return mapPgError("delete employee", err)
	}
	if n == 0 {
		return fmt.Errorf("delete employee: %w", ErrNotFound)
	}

	return nil
}

func (r *PostgresEmployeeRepo) Search(ctx context.Context, keyword string) ([]models.Employee, error) {
	return r.queryEmployees(ctx, "search employees",
		`SELECT `+employeeColumns+` FROM employees
		WHERE deleted_at IS NULL AND (name ILIKE '%' || $1 || '%' OR position ILIKE '%' || $1 || '%')
		ORDER BY id`,
		keyword)
}

// ListByDepartment joins departments so every employee also carries its department name.
func (r *PostgresEmployeeRepo) ListByDepartment(ctx context.Context, deptID int64) ([]models.Employee, error) {
	const op = "list employees by department"
	const query = `
		SELECT e.id, e.name, e.age, e.position, e.department_id, e.salary, e.created_at, e.updated_at, d.name
		FROM employees e
		JOIN departments d ON d.id = e.department_id
		WHERE e.department_id = $1 AND e.deleted_at IS NULL
		ORDER BY e.id`

	rows, err := r.db.QueryContext(ctx, query, deptID)
	if err != nil {
		return nil, mapPgError(op, err)
	}
	defer rows.Close()

	var employees []models.Employee
	for rows.Next() {
		var e models.Employee
		err := rows.Scan(&e.ID, &e.Name, &e.Age, &e.Position, &e.DepartmentID, &e.Salary,
			&e.CreatedAt, &e.UpdatedAt, &e.DepartmentName)
		if err != nil {
			return nil, mapPgError(op, err)
		}
		employees = append(employees, e)
	}
	if err := rows.Err(); err != nil {
		return nil, mapPgError(op, err)
	}

	return employees, nil
}

func (r *PostgresEmployeeRepo) ListAll(ctx context.Context) ([]models.Employee, error) {
	return r.queryEmployees(ctx, "list all employees",
		`SELECT `+employeeColumns+` FROM employees WHERE deleted_at IS NULL ORDER BY id`)
}

// queryEmployees runs a query that returns employeeColumns and scans every row.
func (r *PostgresEmployeeRepo) queryEmployees(ctx context.Context, op, query string, args ...any) ([]models.Employee, error) {
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, mapPgError(op, err)
	}
	defer rows.Close()

	var employees []models.Employee
	for rows.Next() {
		e, err := scanEmployee(rows)
		if err != nil {
			return nil, mapPgError(op, err)
		}
		employees = append(employees, e)
	}
	// rows.Next returns false both at the end and on failure; Err tells them apart.
	if err := rows.Err(); err != nil {
		return nil, mapPgError(op, err)
	}

	return employees, nil
}

// rowScanner is satisfied by both *sql.Row and *sql.Rows.
type rowScanner interface {
	Scan(dest ...any) error
}

func scanEmployee(s rowScanner) (models.Employee, error) {
	var e models.Employee
	err := s.Scan(&e.ID, &e.Name, &e.Age, &e.Position, &e.DepartmentID, &e.Salary, &e.CreatedAt, &e.UpdatedAt)
	return e, err
}
