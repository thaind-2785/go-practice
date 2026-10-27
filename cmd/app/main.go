package main

import (
	"context"
	"database/sql"
	"errors"
	"log"
	"net/http"
	"os"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/thaind-2785/go-practice/internal/utils"
)

func main() {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		log.Fatal("DATABASE_URL is not set")
	}

	db, err := openDB(dsn)
	if err != nil {
		log.Fatalf("connect database: %v", err)
	}
	defer db.Close()
	log.Println("database connected")

	_ = db // tạm thời: bước 5 sẽ truyền db vào repository

	mux := http.NewServeMux()

	mux.HandleFunc("GET /test-error", func(w http.ResponseWriter, r *http.Request) {
		utils.WriteError(w, utils.NewNotFound("Employee not found"))
	})

	mux.HandleFunc("GET /test-unknown", func(w http.ResponseWriter, r *http.Request) {
		utils.WriteError(w, errors.New("db password wrong"))
	})

	log.Println("server listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", mux))
}

func openDB(dsn string) (*sql.DB, error) {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, err
	}

	// Connection pool: giới hạn số kết nối tới DB dùng cùng lúc
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(25)
	db.SetConnMaxLifetime(5 * time.Minute)

	// Thử kết nối thật, chờ tối đa 5 giây
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, err
	}

	return db, nil
}
