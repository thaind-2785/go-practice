package handlers

import (
	"context"
	"net/http"

	"github.com/thaind-2785/go-practice/internal/models"
	"github.com/thaind-2785/go-practice/internal/utils"
)

type ExportService interface {
	Export(ctx context.Context) (*models.ExportResult, error)
}

type ExportHandler struct {
	service ExportService
}

func NewExportHandler(s ExportService) *ExportHandler {
	return &ExportHandler{service: s}
}

func (h *ExportHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /employees/export", h.Export)
}

func (h *ExportHandler) Export(w http.ResponseWriter, r *http.Request) {
	result, err := h.service.Export(r.Context())
	if err != nil {
		utils.WriteError(w, err)
		return
	}

	utils.WriteJSON(w, http.StatusOK, result)
}
