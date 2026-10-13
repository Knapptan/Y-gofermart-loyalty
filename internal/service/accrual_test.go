package service

import (
	"context"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"github.com/knapptan/go-musthave-diploma-tpl/internal/accrual"
	"github.com/knapptan/go-musthave-diploma-tpl/internal/model"
	"github.com/knapptan/go-musthave-diploma-tpl/internal/repository"
)

// fakeAccrualRepo — минимальная in-memory реализация OrderRepository
// для тестов воркера.
type fakeAccrualRepo struct {
	mu      sync.Mutex
	orders  map[string]*model.Order
	applied []string
}

func newFakeAccrualRepo() *fakeAccrualRepo {
	return &fakeAccrualRepo{orders: map[string]*model.Order{}}
}

func (f *fakeAccrualRepo) seed(number, userID string, status model.OrderStatus) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.orders[number] = &model.Order{Number: number, UserID: userID, Status: status}
}

func (f *fakeAccrualRepo) Create(context.Context, string, string) error { return nil }

func (f *fakeAccrualRepo) GetByNumber(_ context.Context, number string) (*model.Order, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	o, ok := f.orders[number]
	if !ok {
		return nil, repository.ErrNotFound
	}
	return o, nil
}

func (f *fakeAccrualRepo) ListByUser(context.Context, string) ([]model.Order, error) {
	return nil, nil
}

func (f *fakeAccrualRepo) ListPending(_ context.Context) ([]model.Order, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []model.Order
	for _, o := range f.orders {
		if o.Status == model.StatusNew || o.Status == model.StatusProcessing {
			out = append(out, *o)
		}
	}
	return out, nil
}

func (f *fakeAccrualRepo) ApplyResult(_ context.Context, number string, status model.OrderStatus, amount float64) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	o, ok := f.orders[number]
	if !ok {
		return repository.ErrNotFound
	}
	if o.Status == model.StatusProcessed || o.Status == model.StatusInvalid {
		return nil // идемпотентность
	}
	o.Status = status
	if status == model.StatusProcessed {
		o.Accrual = amount
	}
	f.applied = append(f.applied, number)
	return nil
}

func (f *fakeAccrualRepo) snapshot(number string) (model.OrderStatus, float64) {
	f.mu.Lock()
	defer f.mu.Unlock()
	o := f.orders[number]
	return o.Status, o.Accrual
}

func (f *fakeAccrualRepo) appliedCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.applied)
}

// fakeAccrualClient — управляемый мок accrual-клиента для synctest.
type fakeAccrualClient struct {
	mu        sync.Mutex
	responses map[string]*accrual.Response
	errs      map[string]error
	calls     int
}

func (c *fakeAccrualClient) GetOrder(_ context.Context, number string) (*accrual.Response, error) {
	c.mu.Lock()
	c.calls++
	c.mu.Unlock()

	if err, ok := c.errs[number]; ok {
		return nil, err
	}
	resp, ok := c.responses[number]
	if !ok {
		return nil, accrual.ErrNotRegistered
	}
	return resp, nil
}

func (c *fakeAccrualClient) callCount() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.calls
}

// runWorker запускает воркер и гарантирует остановку до выхода из теста.
func runWorker(t *testing.T, w *AccrualWorker) {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		w.Run(ctx)
	}()
	t.Cleanup(func() {
		cancel()
		wg.Wait()
	})
}

func TestWorker_ProcessedAccrues(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		repo := newFakeAccrualRepo()
		repo.seed("111", "u1", model.StatusNew)

		client := &fakeAccrualClient{
			responses: map[string]*accrual.Response{
				"111": {Order: "111", Status: accrual.StatusProcessed, Accrual: 500},
			},
		}
		w := NewAccrualWorker(repo, client, 50*time.Millisecond)
		runWorker(t, w)

		synctest.Wait() // воркер заблокирован на ticker

		time.Sleep(60 * time.Millisecond) // синтетическое время двигается мгновенно
		synctest.Wait()                   // ждём завершения реакции воркера

		st, acc := repo.snapshot("111")
		if st != model.StatusProcessed || acc != 500 {
			t.Fatalf("order not processed: status=%s accrual=%v", st, acc)
		}
	})
}

func TestWorker_InvalidNoAccrual(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		repo := newFakeAccrualRepo()
		repo.seed("222", "u1", model.StatusNew)

		client := &fakeAccrualClient{
			responses: map[string]*accrual.Response{
				"222": {Order: "222", Status: accrual.StatusInvalid},
			},
		}
		w := NewAccrualWorker(repo, client, 50*time.Millisecond)
		runWorker(t, w)

		synctest.Wait()
		time.Sleep(60 * time.Millisecond)
		synctest.Wait()

		st, acc := repo.snapshot("222")
		if st != model.StatusInvalid {
			t.Fatalf("want INVALID, got %s", st)
		}
		if acc != 0 {
			t.Fatalf("invalid order must not accrue, got %v", acc)
		}
	})
}

func TestWorker_IdempotentAfterFinal(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		repo := newFakeAccrualRepo()
		repo.seed("333", "u1", model.StatusNew)

		client := &fakeAccrualClient{
			responses: map[string]*accrual.Response{
				"333": {Order: "333", Status: accrual.StatusProcessed, Accrual: 100},
			},
		}
		w := NewAccrualWorker(repo, client, 30*time.Millisecond)
		runWorker(t, w)

		synctest.Wait()
		time.Sleep(200 * time.Millisecond) // несколько тиков
		synctest.Wait()

		if got := repo.appliedCount(); got != 1 {
			t.Fatalf("expected exactly 1 ApplyResult, got %d", got)
		}
	})
}

func TestWorker_NotRegisteredKeepsNew(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		repo := newFakeAccrualRepo()
		repo.seed("444", "u1", model.StatusNew)

		client := &fakeAccrualClient{} // все заказы → ErrNotRegistered
		w := NewAccrualWorker(repo, client, 30*time.Millisecond)
		runWorker(t, w)

		synctest.Wait()
		time.Sleep(150 * time.Millisecond)
		synctest.Wait()

		st, _ := repo.snapshot("444")
		if st != model.StatusNew {
			t.Fatalf("status must stay NEW, got %s", st)
		}
		if got := repo.appliedCount(); got != 0 {
			t.Fatalf("ApplyResult must not be called, got %d", got)
		}
	})
}

// TestWorker_RateLimitWaits — ключевой тест на паузу по Retry-After.
func TestWorker_RateLimitWaits(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		repo := newFakeAccrualRepo()
		repo.seed("429", "u1", model.StatusNew)

		client := &fakeAccrualClient{
			errs: map[string]error{
				"429": &accrual.TooManyRequestsError{RetryAfter: 500 * time.Millisecond},
			},
		}
		w := NewAccrualWorker(repo, client, 50*time.Millisecond)
		runWorker(t, w)

		synctest.Wait()
		time.Sleep(60 * time.Millisecond) // первый тик на 50 мс
		synctest.Wait()

		if got := client.callCount(); got != 1 {
			t.Fatalf("after first tick expected 1 call, got %d", got)
		}

		// Двигаем время на 200 мс — воркер должен ещё спать (Retry-After=500мс).
		time.Sleep(200 * time.Millisecond)
		synctest.Wait()

		if got := client.callCount(); got != 1 {
			t.Fatalf("worker ignored Retry-After: got %d calls", got)
		}
	})
}
