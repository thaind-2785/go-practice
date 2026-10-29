package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/thaind-2785/go-practice/internal/handlers"
	"github.com/thaind-2785/go-practice/internal/middleware"
	"github.com/thaind-2785/go-practice/internal/repositories"
	"github.com/thaind-2785/go-practice/internal/services"
)

const (
	requestTimeout = 3 * time.Second
	exportDir      = "exports"
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
	deptService := services.NewDepartmentService(deptRepo, empRepo)
	exportService := services.NewExportService(empRepo, exportDir)

	mux := http.NewServeMux()
	handlers.NewEmployeeHandler(empService).RegisterRoutes(mux)
	handlers.NewDepartmentHandler(deptService).RegisterRoutes(mux)
	handlers.NewExportHandler(exportService).RegisterRoutes(mux)

	users, err := parseUsers(os.Getenv("AUTH_USERS"))
	if err != nil {
		log.Fatal(err)
	}

	handler := middleware.Chain(mux,
		middleware.RequestID,
		middleware.Recover,
		middleware.BasicAuth(users),
		middleware.Logging,
		middleware.Timeout(requestTimeout),
	)

	log.Println("server listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", handler))
}

func parseUsers(s string) (map[string]string, error) {
	if s == "" {
		return nil, errors.New("AUTH_USERS is not set")
	}

	users := map[string]string{}
	for i, pair := range strings.Split(s, ",") {
		user, pass, ok := strings.Cut(strings.TrimSpace(pair), ":")
		if !ok || user == "" || pass == "" {
			return nil, fmt.Errorf("AUTH_USERS entry #%d must look like user:password", i+1)
		}
		users[user] = pass
	}

	return users, nil
}

func openDB(dsn string) (*sql.DB, error) {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, err
	}

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(25)
	db.SetConnMaxLifetime(5 * time.Minute)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, err
	}

	return db, nil
}
