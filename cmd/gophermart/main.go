package main

import (
	"context"
	"database/sql"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	_ "github.com/lib/pq"

	"github.com/knapptan/go-musthave-diploma-tpl/internal/accrual"
	"github.com/knapptan/go-musthave-diploma-tpl/internal/config"
	"github.com/knapptan/go-musthave-diploma-tpl/internal/handler"
	"github.com/knapptan/go-musthave-diploma-tpl/internal/middleware"
	"github.com/knapptan/go-musthave-diploma-tpl/internal/repository/postgres"
	"github.com/knapptan/go-musthave-diploma-tpl/internal/service"
	"github.com/knapptan/go-musthave-diploma-tpl/migrations"
)

func main() {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, nil)))

	cfg, err := config.ParseFlags()
	if err != nil {
		slog.Error("config failed", "error", err)
		os.Exit(1)
	}
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

	if err := runMigrations(cfg.DatabaseURI); err != nil {
		slog.Error("migrate failed", "error", err)
		os.Exit(1)
	}
	slog.Info("migrations applied")

	// --- repositories ---
	userRepo := postgres.NewUserRepo(db)
	orderRepo := postgres.NewOrderRepo(db)
	balanceRepo := postgres.NewBalanceRepo(db)

	// --- services ---
	userSvc := service.NewUserService(userRepo, cfg.JWTSecret)
	orderSvc := service.NewOrderService(orderRepo)
	balanceSvc := service.NewBalanceService(balanceRepo)

	// --- handlers ---
	userH := handler.NewUserHandler(userSvc)
	orderH := handler.NewOrderHandler(orderSvc)
	balanceH := handler.NewBalanceHandler(balanceSvc)

	// --- accrual worker ---
	accrualClient := accrual.NewClient(cfg.AccrualAddress)
	accrualWorker := service.NewAccrualWorker(orderRepo, accrualClient, 5*time.Second)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go accrualWorker.Run(ctx)

	// --- router ---
	mux := http.NewServeMux()

	mux.HandleFunc("GET /ping", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	})

	// Публичные маршруты.
	mux.HandleFunc("POST /api/user/register", userH.Register)
	mux.HandleFunc("POST /api/user/login", userH.Login)

	// Защищённые маршруты — оборачиваем хендлеры в middleware.Auth.
	auth := middleware.Auth(cfg.JWTSecret)
	mux.Handle("POST /api/user/orders", auth(http.HandlerFunc(orderH.Create)))
	mux.Handle("GET /api/user/orders", auth(http.HandlerFunc(orderH.List)))
	mux.Handle("GET /api/user/balance", auth(http.HandlerFunc(balanceH.Get)))
	mux.Handle("POST /api/user/balance/withdraw", auth(http.HandlerFunc(balanceH.Withdraw)))
	mux.Handle("GET /api/user/withdrawals", auth(http.HandlerFunc(balanceH.ListWithdrawals)))

	slog.Info("starting server", "addr", cfg.RunAddress)
	if err := http.ListenAndServe(cfg.RunAddress, mux); err != nil {
		slog.Error("server failed", "error", err)
		os.Exit(1)
	}
}

// runMigrations применяет миграции из встроенной ФС.
func runMigrations(dsn string) error {
	src, err := iofs.New(migrations.FS, ".")
	if err != nil {
		return err
	}
	m, err := migrate.NewWithSourceInstance("iofs", src, dsn)
	if err != nil {
		return err
	}
	defer m.Close()

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return err
	}
	return nil
}
