package main

import (
	"database/sql"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"

	"github.com/go-chi/chi/v5"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/lib/pq"

	"github.com/knapptan/go-musthave-diploma-tpl/internal/config"
	"github.com/knapptan/go-musthave-diploma-tpl/internal/handler"
	"github.com/knapptan/go-musthave-diploma-tpl/internal/middleware"
	"github.com/knapptan/go-musthave-diploma-tpl/internal/repository/postgres"
	"github.com/knapptan/go-musthave-diploma-tpl/internal/service"
)

func main() {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, nil)))

	cfg := config.ParseFlags()
	slog.Info("config", "run", cfg.RunAddress, "db", cfg.DatabaseURI, "accrual", cfg.AccrualAddress)

	db, err := sql.Open("postgres", cfg.DatabaseURI)
	if err != nil {
		slog.Error("db open failed", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		slog.Error("db ping failed", "error", err)
		os.Exit(1)
	}

	cwd, _ := os.Getwd()
	m, err := migrate.New("file://"+filepath.Join(cwd, "migrations"), cfg.DatabaseURI)
	if err != nil {
		slog.Error("migrate init failed", "error", err)
		os.Exit(1)
	}
	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		slog.Error("migrate up failed", "error", err)
		os.Exit(1)
	}
	slog.Info("migrations applied")

	// --- repositories ---
	userRepo := postgres.NewUserRepo(db)
	orderRepo := postgres.NewOrderRepo(db)

	// --- services ---
	userSvc := service.NewUserService(userRepo, cfg.JWTSecret)
	orderSvc := service.NewOrderService(orderRepo)

	// --- handlers ---
	userH := handler.NewUserHandler(userSvc)
	orderH := handler.NewOrderHandler(orderSvc)

	// --- router ---
	r := chi.NewRouter()

	r.Get("/ping", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	r.Route("/api/user", func(r chi.Router) {
		r.Post("/register", userH.Register)
		r.Post("/login", userH.Login)

		r.Group(func(r chi.Router) {
			r.Use(middleware.Auth(cfg.JWTSecret))
			r.Post("/orders", orderH.Create)
			r.Get("/orders", orderH.List)
		})
	})

	slog.Info("starting server", "addr", cfg.RunAddress)
	if err := http.ListenAndServe(cfg.RunAddress, r); err != nil {
		slog.Error("server failed", "error", err)
		os.Exit(1)
	}
}
