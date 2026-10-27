package repositories

import (
	"context"
	"database/sql"

	"github.com/thaind-2785/go-practice/internal/models"
)

var _ DepartmentRepository = (*PostgresDepartmentRepo)(nil)

type PostgresDepartmentRepo struct {
	db *sql.DB
}

func NewPostgresDepartmentRepo(db *sql.DB) *PostgresDepartmentRepo {
	return &PostgresDepartmentRepo{db: db}
}

func (r *PostgresDepartmentRepo) Create(ctx context.Context, d *models.Department) error {
	err := r.db.QueryRowContext(ctx,
		`INSERT INTO departments (name) VALUES ($1) RETURNING id, created_at`, d.Name).
		Scan(&d.ID, &d.CreatedAt)
	if err != nil {
		return mapPgError("create department", err)
	}

	return nil
}

func (r *PostgresDepartmentRepo) List(ctx context.Context) ([]models.Department, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id, name, created_at FROM departments ORDER BY id`)
	if err != nil {
		return nil, mapPgError("list departments", err)
	}
	defer rows.Close()

	var departments []models.Department
	for rows.Next() {
		var d models.Department
		if err := rows.Scan(&d.ID, &d.Name, &d.CreatedAt); err != nil {
			return nil, mapPgError("list departments", err)
		}
		departments = append(departments, d)
	}
	if err := rows.Err(); err != nil {
		return nil, mapPgError("list departments", err)
	}

	return departments, nil
}

func (r *PostgresDepartmentRepo) GetByID(ctx context.Context, id int64) (*models.Department, error) {
	var d models.Department
	err := r.db.QueryRowContext(ctx,
		`SELECT id, name, created_at FROM departments WHERE id = $1`, id).
		Scan(&d.ID, &d.Name, &d.CreatedAt)
	if err != nil {
		return nil, mapPgError("get department", err)
	}

	return &d, nil
}
