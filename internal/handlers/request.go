package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/thaind-2785/go-practice/internal/utils"
)

func parseID(r *http.Request) (int64, error) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		return 0, utils.NewBadRequest("invalid id")
	}
	return id, nil
}

func queryInt(r *http.Request, key string) (int64, error) {
	raw := r.URL.Query().Get(key)
	if raw == "" {
		return 0, nil
	}
	n, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return 0, utils.NewBadRequest("invalid query parameter: " + key)
	}
	return n, nil
}

func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) error {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		return utils.NewBadRequest("invalid request body: " + err.Error())
	}
	return nil
}
