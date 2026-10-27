package services

import (
	"context"
	"net/http"
	"testing"

	"github.com/thaind-2785/go-practice/internal/models"
)

func newDepartmentTestService() (*DepartmentService, *fakeDepartmentRepo, *fakeEmployeeRepo) {
	deptRepo := newFakeDepartmentRepo()
	empRepo := newFakeEmployeeRepo()
	return NewDepartmentService(deptRepo, empRepo), deptRepo, empRepo
}

func TestDepartmentService_Create(t *testing.T) {
	t.Run("valid name is trimmed and saved", func(t *testing.T) {
		svc, _, _ := newDepartmentTestService()

		got, err := svc.Create(context.Background(), models.CreateDepartmentRequest{Name: "  Engineering "})

		assertCode(t, err, 0)
		if got.ID == 0 {
			t.Error("expected repository to assign an ID")
		}
		if got.Name != "Engineering" {
			t.Errorf("Name = %q, want %q", got.Name, "Engineering")
		}
	})

	t.Run("blank name returns 422", func(t *testing.T) {
		svc, _, _ := newDepartmentTestService()

		_, err := svc.Create(context.Background(), models.CreateDepartmentRequest{Name: "   "})

		assertCode(t, err, http.StatusUnprocessableEntity)
	})

	t.Run("duplicate name returns 409", func(t *testing.T) {
		svc, _, _ := newDepartmentTestService()
		if _, err := svc.Create(context.Background(), models.CreateDepartmentRequest{Name: "Sales"}); err != nil {
			t.Fatalf("first create: %v", err)
		}

		_, err := svc.Create(context.Background(), models.CreateDepartmentRequest{Name: " Sales "})

		assertCode(t, err, http.StatusConflict)
	})
}

func TestDepartmentService_List(t *testing.T) {
	svc, _, _ := newDepartmentTestService()

	got, err := svc.List(context.Background())
	assertCode(t, err, 0)
	if got == nil || len(got) != 0 {
		t.Fatalf("empty list = %#v, want an empty non-nil slice", got)
	}

	for _, name := range []string{"Engineering", "Sales"} {
		if _, err := svc.Create(context.Background(), models.CreateDepartmentRequest{Name: name}); err != nil {
			t.Fatalf("create %q: %v", name, err)
		}
	}

	got, err = svc.List(context.Background())
	assertCode(t, err, 0)
	if len(got) != 2 || got[0].Name != "Engineering" || got[1].Name != "Sales" {
		t.Errorf("List = %+v", got)
	}
}

func TestDepartmentService_ListEmployees(t *testing.T) {
	ctx := context.Background()
	svc, _, empRepo := newDepartmentTestService()
	eng, err := svc.Create(ctx, models.CreateDepartmentRequest{Name: "Engineering"})
	if err != nil {
		t.Fatalf("create department: %v", err)
	}
	empty, err := svc.Create(ctx, models.CreateDepartmentRequest{Name: "Empty"})
	if err != nil {
		t.Fatalf("create department: %v", err)
	}

	for _, e := range []models.Employee{
		{Name: "A", Age: 20, Position: "Dev", DepartmentID: eng.ID, Salary: 1},
		{Name: "B", Age: 20, Position: "Dev", DepartmentID: eng.ID, Salary: 1},
		{Name: "Other", Age: 20, Position: "Dev", DepartmentID: 999, Salary: 1},
	} {
		if err := empRepo.Create(ctx, &e); err != nil {
			t.Fatalf("seed employee: %v", err)
		}
	}

	t.Run("returns only employees of that department", func(t *testing.T) {
		got, err := svc.ListEmployees(ctx, eng.ID)

		assertCode(t, err, 0)
		if len(got) != 2 || got[0].Name != "A" || got[1].Name != "B" {
			t.Errorf("got %+v", got)
		}
	})

	t.Run("department without employees returns empty list", func(t *testing.T) {
		got, err := svc.ListEmployees(ctx, empty.ID)

		assertCode(t, err, 0)
		if got == nil || len(got) != 0 {
			t.Errorf("got %#v, want an empty non-nil slice", got)
		}
	})

	t.Run("unknown department returns 404", func(t *testing.T) {
		_, err := svc.ListEmployees(ctx, 999)

		assertCode(t, err, http.StatusNotFound)
	})
}
