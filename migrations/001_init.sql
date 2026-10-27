CREATE TABLE IF NOT EXISTS departments (
    id         BIGSERIAL PRIMARY KEY,
    name       TEXT NOT NULL UNIQUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS employees (
    id         BIGSERIAL PRIMARY KEY,
    name       TEXT NOT NULL,
    age        INT NOT NULL CHECK (age > 0),
    position   TEXT NOT NULL,
    department_id BIGINT NOT NULL REFERENCES departments(id),
    salary     NUMERIC(12,2) NOT NULL CHECK (salary > 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_employees_department_id ON employees (department_id);
