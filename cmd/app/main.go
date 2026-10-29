package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/thaind-2785/go-practice/internal/handlers"
	"github.com/thaind-2785/go-practice/internal/middleware"
	"github.com/thaind-2785/go-practice/internal/repositories"
	"github.com/thaind-2785/go-practice/internal/services"
)

const (
	addr            = ":8080"
	requestTimeout  = 3 * time.Second
	shutdownTimeout = 10 * time.Second
	exportDir       = "exports"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		return errors.New("DATABASE_URL is not set")
	}

	users, err := parseUsers(os.Getenv("AUTH_USERS"))
	if err != nil {
		return err
	}

	db, err := openDB(dsn)
	if err != nil {
		return fmt.Errorf("connect database: %w", err)
	}
	defer db.Close()
	log.Println("database connected")

	empRepo, err := repositories.NewPostgresEmployeeRepo(context.Background(), db)
	if err != nil {
		return fmt.Errorf("init employee repository: %w", err)
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

	handler := middleware.Chain(mux,
		middleware.RequestID,
		middleware.Recover,
		middleware.BasicAuth(users),
		middleware.Logging,
		middleware.Timeout(requestTimeout),
	)

	srv := &http.Server{
		Addr:              addr,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	serverErr := make(chan error, 1)
	go func() {
		log.Printf("server listening on %s", srv.Addr)
		serverErr <- srv.ListenAndServe()
	}()

	select {
	case err := <-serverErr:
		return fmt.Errorf("listen: %w", err)
	case <-ctx.Done():
	}
	stop()

	log.Println("shutting down, waiting for in-flight requests")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("shutdown: %w", err)
	}

	log.Println("server stopped")
	return nil
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
