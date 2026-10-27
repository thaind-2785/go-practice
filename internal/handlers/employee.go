package handlers

import (
	"context"
	"net/http"

	"github.com/thaind-2785/go-practice/internal/models"
	"github.com/thaind-2785/go-practice/internal/utils"
)

type EmployeeService interface {
	Create(ctx context.Context, req models.CreateEmployeeRequest) (*models.Employee, error)
	Get(ctx context.Context, id int64) (*models.Employee, error)
	List(ctx context.Context, f models.ListEmployeesFilter) (*models.ListEmployeesResult, error)
	Update(ctx context.Context, id int64, req models.UpdateEmployeeRequest) (*models.Employee, error)
	Delete(ctx context.Context, id int64) error
	Search(ctx context.Context, keyword string) ([]models.Employee, error)
}

type EmployeeHandler struct {
	service EmployeeService
}

func NewEmployeeHandler(s EmployeeService) *EmployeeHandler {
	return &EmployeeHandler{service: s}
}

// RegisterRoutes attaches every employee endpoint to mux.
func (h *EmployeeHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /employees", h.Create)
	mux.HandleFunc("GET /employees", h.List)
	mux.HandleFunc("GET /employees/search", h.Search) // more specific than {id}, so it wins for /employees/search
	mux.HandleFunc("GET /employees/{id}", h.Get)
	mux.HandleFunc("PUT /employees/{id}", h.Update)
	mux.HandleFunc("DELETE /employees/{id}", h.Delete)
}

// GET /employees/{id}
func (h *EmployeeHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r) // 1. parse
	if err != nil {
		utils.WriteError(w, err)
		return
	}

	employee, err := h.service.Get(r.Context(), id) // 2. gọi service
	if err != nil {
		utils.WriteError(w, err)
		return
	}

	utils.WriteJSON(w, http.StatusOK, employee) // 3. ghi response
}

// POST /employees
func (h *EmployeeHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req models.CreateEmployeeRequest
	if err := decodeJSON(w, r, &req); err != nil {
		utils.WriteError(w, err)
		return
	}

	employee, err := h.service.Create(r.Context(), req)
	if err != nil {
		utils.WriteError(w, err)
		return
	}

	utils.WriteJSON(w, http.StatusCreated, employee)
}

// GET /employees?limit=10&offset=0&departmentId=1
func (h *EmployeeHandler) List(w http.ResponseWriter, r *http.Request) {
	limit, err := queryInt(r, "limit")
	if err != nil {
		utils.WriteError(w, err)
		return
	}
	offset, err := queryInt(r, "offset")
	if err != nil {
		utils.WriteError(w, err)
		return
	}
	deptID, err := queryInt(r, "departmentId")
	if err != nil {
		utils.WriteError(w, err)
		return
	}

	// Default and maximum limit are applied by the service.
	filter := models.ListEmployeesFilter{Limit: int(limit), Offset: int(offset)}
	if deptID > 0 {
		filter.DepartmentID = &deptID
	}

	result, err := h.service.List(r.Context(), filter)
	if err != nil {
		utils.WriteError(w, err)
		return
	}

	utils.WriteJSON(w, http.StatusOK, result)
}

// PUT /employees/{id}
func (h *EmployeeHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		utils.WriteError(w, err)
		return
	}

	var req models.UpdateEmployeeRequest
	if err := decodeJSON(w, r, &req); err != nil {
		utils.WriteError(w, err)
		return
	}

	employee, err := h.service.Update(r.Context(), id, req)
	if err != nil {
		utils.WriteError(w, err)
		return
	}

	utils.WriteJSON(w, http.StatusOK, employee)
}

// DELETE /employees/{id}
func (h *EmployeeHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		utils.WriteError(w, err)
		return
	}

	if err := h.service.Delete(r.Context(), id); err != nil {
		utils.WriteError(w, err)
		return
	}

	// 204 No Content: success without a response body.
	w.WriteHeader(http.StatusNoContent)
}

// GET /employees/search?keyword=an
func (h *EmployeeHandler) Search(w http.ResponseWriter, r *http.Request) {
	keyword := r.URL.Query().Get("keyword")

	employees, err := h.service.Search(r.Context(), keyword)
	if err != nil {
		utils.WriteError(w, err)
		return
	}

	utils.WriteJSON(w, http.StatusOK, employees)
}
