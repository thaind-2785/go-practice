package main

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"os"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/thaind-2785/go-practice/internal/handlers"
	"github.com/thaind-2785/go-practice/internal/repositories"
	"github.com/thaind-2785/go-practice/internal/services"
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

	ctx := context.Background()

	empRepo, err := repositories.NewPostgresEmployeeRepo(ctx, db)
	if err != nil {
		log.Fatalf("init employee repository: %v", err)
	}
	defer empRepo.Close()
	deptRepo := repositories.NewPostgresDepartmentRepo(db)

	empService := services.NewEmployeeService(empRepo, deptRepo)

	mux := http.NewServeMux()
	handlers.NewEmployeeHandler(empService).RegisterRoutes(mux)

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
