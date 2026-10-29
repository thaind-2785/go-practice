package handlers

import (
	"context"
	"net/http"

	"github.com/thaind-2785/go-practice/internal/models"
	"github.com/thaind-2785/go-practice/internal/utils"
)

type DepartmentService interface {
	Create(ctx context.Context, req models.CreateDepartmentRequest) (*models.Department, error)
	List(ctx context.Context) ([]models.Department, error)
	ListEmployees(ctx context.Context, id int64) ([]models.Employee, error)
}

type DepartmentHandler struct {
	service DepartmentService
}

func NewDepartmentHandler(s DepartmentService) *DepartmentHandler {
	return &DepartmentHandler{service: s}
}

func (h *DepartmentHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /departments", h.Create)
	mux.HandleFunc("GET /departments", h.List)
	mux.HandleFunc("GET /departments/{id}/employees", h.ListEmployees)
}

func (h *DepartmentHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req models.CreateDepartmentRequest
	if err := decodeJSON(w, r, &req); err != nil {
		utils.WriteError(w, err)
		return
	}

	department, err := h.service.Create(r.Context(), req)
	if err != nil {
		utils.WriteError(w, err)
		return
	}

	utils.WriteJSON(w, http.StatusCreated, department)
}

func (h *DepartmentHandler) List(w http.ResponseWriter, r *http.Request) {
	departments, err := h.service.List(r.Context())
	if err != nil {
		utils.WriteError(w, err)
		return
	}

	utils.WriteJSON(w, http.StatusOK, departments)
}

func (h *DepartmentHandler) ListEmployees(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		utils.WriteError(w, err)
		return
	}

	employees, err := h.service.ListEmployees(r.Context(), id)
	if err != nil {
		utils.WriteError(w, err)
		return
	}

	utils.WriteJSON(w, http.StatusOK, employees)
}
