package main

import (
	"errors"
	"log"
	"net/http"

	"github.com/thaind-2785/go-practice/internal/utils"
)

func main() {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /test-error", func(w http.ResponseWriter, r *http.Request) {
		utils.WriteError(w, utils.NewNotFound("Employee not found"))
	})

	mux.HandleFunc("GET /test-unknown", func(w http.ResponseWriter, r *http.Request) {
		utils.WriteError(w, errors.New("db password wrong"))
	})

	log.Println("server listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", mux)) // giống app.listen(8080)
}
