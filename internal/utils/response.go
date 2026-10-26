package utils

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
)

func WriteJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func WriteError(w http.ResponseWriter, err error) {
	if appErr, ok := errors.AsType[*AppError](err); ok {
		WriteJSON(w, appErr.Code, appErr)
		return
	}

	log.Printf("unexpected error: %v", err)
	WriteJSON(w, 500, NewInternal("internal server error"))
}
