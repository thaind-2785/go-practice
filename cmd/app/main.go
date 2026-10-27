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

// requestTimeout is how long a single request may take before its database calls are cancelled.
const requestTimeout = 3 * time.Second

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

	mux := http.NewServeMux()
	handlers.NewEmployeeHandler(empService).RegisterRoutes(mux)
	handlers.NewDepartmentHandler(deptService).RegisterRoutes(mux)

	users, err := parseUsers(os.Getenv("AUTH_USERS"))
	if err != nil {
		log.Fatal(err)
	}

	// Order matters: a request goes through these from top to bottom before reaching mux.
	handler := middleware.Chain(mux,
		middleware.RequestID,        // first, so every later log line has the request ID
		middleware.Recover,          // catches panics from everything below
		middleware.BasicAuth(users), // rejects unauthenticated requests early
		middleware.Logging,          // inside BasicAuth, so it can log the user
		middleware.Timeout(requestTimeout),
	)

	log.Println("server listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", handler))
}

// parseUsers reads AUTH_USERS in the form "alice:password1,bob:password2".
func parseUsers(s string) (map[string]string, error) {
	if s == "" {
		return nil, errors.New("AUTH_USERS is not set")
	}

	users := map[string]string{}
	for i, pair := range strings.Split(s, ",") {
		user, pass, ok := strings.Cut(strings.TrimSpace(pair), ":")
		if !ok || user == "" || pass == "" {
			// Report the position only, never the value: it contains a password.
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
