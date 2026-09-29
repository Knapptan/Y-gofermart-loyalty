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

	r := chi.NewRouter()
	r.Get("/ping", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	slog.Info("starting server", "addr", cfg.RunAddress)
	if err := http.ListenAndServe(cfg.RunAddress, r); err != nil {
		slog.Error("server failed", "error", err)
		os.Exit(1)
	}
}
