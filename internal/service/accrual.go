package service

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/knapptan/go-musthave-diploma-tpl/internal/accrual"
	"github.com/knapptan/go-musthave-diploma-tpl/internal/model"
	"github.com/knapptan/go-musthave-diploma-tpl/internal/repository"
)

const defaultPollInterval = 5 * time.Second

type AccrualWorker struct {
	orders   repository.OrderRepository
	client   *accrual.Client
	interval time.Duration
}

func NewAccrualWorker(orders repository.OrderRepository, client *accrual.Client, interval time.Duration) *AccrualWorker {
	if interval <= 0 {
		interval = defaultPollInterval
	}
	return &AccrualWorker{
		orders:   orders,
		client:   client,
		interval: interval,
	}
}

// Run блокируется и работает до отмены ctx.
func (w *AccrualWorker) Run(ctx context.Context) {
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	slog.Info("accrual worker started", "interval", w.interval)
	for {
		select {
		case <-ctx.Done():
			slog.Info("accrual worker stopped")
			return
		case <-ticker.C:
			w.tick(ctx)
		}
	}
}

func (w *AccrualWorker) tick(ctx context.Context) {
	orders, err := w.orders.ListPending(ctx)
	if err != nil {
		slog.Error("accrual: list pending failed", "error", err)
		return
	}
	if len(orders) == 0 {
		return
	}

	for _, o := range orders {
		select {
		case <-ctx.Done():
			return
		default:
		}

		if err := w.processOrder(ctx, o); err != nil {
			var tooMany *accrual.TooManyRequestsError
			if errors.As(err, &tooMany) {
				slog.Warn("accrual: rate limited", "retry_after", tooMany.RetryAfter)
				return // прерываем весь тик
			}
			slog.Error("accrual: process order failed", "number", o.Number, "error", err)
		}
	}
}

func (w *AccrualWorker) processOrder(ctx context.Context, o model.Order) error {
	resp, err := w.client.GetOrder(ctx, o.Number)
	if errors.Is(err, accrual.ErrNotRegistered) {
		// Заказ пока не попал в систему расчёта — оставляем NEW.
		return nil
	}
	if err != nil {
		return err
	}

	status, accrual := mapAccrualStatus(resp)
	return w.orders.ApplyResult(ctx, o.Number, status, accrual)
}

// mapAccrualStatus переводит внешний статус во внутренний.
func mapAccrualStatus(r *accrual.Response) (model.OrderStatus, float64) {
	switch r.Status {
	case accrual.StatusProcessing:
		return model.StatusProcessing, 0
	case accrual.StatusInvalid:
		return model.StatusInvalid, 0
	case accrual.StatusProcessed:
		return model.StatusProcessed, r.Accrual
	default:
		// REGISTERED и всё неизвестное — оставляем как NEW.
		return model.StatusNew, 0
	}
}
