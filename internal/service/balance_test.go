package service

import (
	"context"
	"testing"
	"time"

	"github.com/knapptan/go-musthave-diploma-tpl/internal/model"
	"github.com/knapptan/go-musthave-diploma-tpl/internal/repository"
)

// fakeBalanceRepoSimple — минимальная реализация BalanceRepository для тестов.
type fakeBalanceRepoSimple struct {
	balance     *model.Balance
	withdrawals []model.Withdrawal
	err         error
}

func (f *fakeBalanceRepoSimple) Get(_ context.Context, _ string) (*model.Balance, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.balance, nil
}

func (f *fakeBalanceRepoSimple) Withdraw(_ context.Context, _, _ string, _ float64) error {
	return f.err
}

func (f *fakeBalanceRepoSimple) ListWithdrawals(_ context.Context, _ string) ([]model.Withdrawal, error) {
	return f.withdrawals, nil
}

// fakeOrderRepoSimple — минимальная реализация OrderRepository для тестов.
type fakeOrderRepoSimple struct {
	err error
}

func (f *fakeOrderRepoSimple) Create(_ context.Context, _, _ string) error {
	return f.err
}

func (f *fakeOrderRepoSimple) GetByNumber(_ context.Context, _ string) (*model.Order, error) {
	return nil, repository.ErrNotFound
}

func (f *fakeOrderRepoSimple) ListByUser(_ context.Context, _ string) ([]model.Order, error) {
	return []model.Order{}, nil
}

func (f *fakeOrderRepoSimple) ListPending(_ context.Context) ([]model.Order, error) {
	return nil, nil
}

func (f *fakeOrderRepoSimple) ApplyResult(_ context.Context, _ string, _ model.OrderStatus, _ float64) error {
	return nil
}

// --- BalanceService ---

func TestNewBalanceService(t *testing.T) {
	repo := &fakeBalanceRepoSimple{}
	svc := NewBalanceService(repo)
	if svc == nil {
		t.Fatal("expected non-nil BalanceService")
	}
}

func TestBalanceService_Get(t *testing.T) {
	svc := NewBalanceService(&fakeBalanceRepoSimple{
		balance: &model.Balance{Current: 100, Withdrawn: 50},
	})
	b, err := svc.Get(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if b.Current != 100 || b.Withdrawn != 50 {
		t.Fatalf("unexpected balance: %+v", b)
	}
}

func TestBalanceService_Withdraw_Success(t *testing.T) {
	svc := NewBalanceService(&fakeBalanceRepoSimple{})
	err := svc.Withdraw(context.Background(), "user-1", "12345678903", 100)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestBalanceService_Withdraw_EmptySum(t *testing.T) {
	svc := NewBalanceService(&fakeBalanceRepoSimple{})
	err := svc.Withdraw(context.Background(), "user-1", "12345678903", 0)
	if err != ErrInvalidWithdrawSum {
		t.Fatalf("want ErrInvalidWithdrawSum, got %v", err)
	}
}

func TestBalanceService_Withdraw_Insufficient(t *testing.T) {
	svc := NewBalanceService(&fakeBalanceRepoSimple{
		err: repository.ErrInsufficientFunds,
	})
	err := svc.Withdraw(context.Background(), "user-1", "12345678903", 100)
	if err != ErrInsufficientFunds {
		t.Fatalf("want ErrInsufficientFunds, got %v", err)
	}
}

func TestBalanceService_ListWithdrawals(t *testing.T) {
	expected := []model.Withdrawal{
		{Order: "111", Sum: 100, ProcessedAt: time.Now()},
		{Order: "222", Sum: 50, ProcessedAt: time.Now().Add(-time.Hour)},
	}
	repo := &fakeBalanceRepoSimple{
		withdrawals: expected,
	}
	svc := NewBalanceService(repo)

	list, err := svc.ListWithdrawals(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("expected 2 withdrawals, got %d", len(list))
	}
	if list[0].Order != "111" || list[0].Sum != 100 {
		t.Errorf("unexpected first withdrawal: %+v", list[0])
	}
}

// --- OrderService ---

func TestNewOrderService(t *testing.T) {
	repo := &fakeOrderRepoSimple{}
	svc := NewOrderService(repo)
	if svc == nil {
		t.Fatal("expected non-nil OrderService")
	}
}

func TestOrderService_CreateOrder(t *testing.T) {
	svc := NewOrderService(&fakeOrderRepoSimple{})
	err := svc.CreateOrder(context.Background(), "user-1", "12345678903")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestOrderService_CreateOrder_InvalidLuhn(t *testing.T) {
	svc := NewOrderService(&fakeOrderRepoSimple{})
	err := svc.CreateOrder(context.Background(), "user-1", "invalid")
	if err != ErrInvalidOrderNumber {
		t.Fatalf("want ErrInvalidOrderNumber, got %v", err)
	}
}

func TestOrderService_CreateOrder_RepoError(t *testing.T) {
	svc := NewOrderService(&fakeOrderRepoSimple{
		err: repository.ErrOrderExistsOther,
	})
	err := svc.CreateOrder(context.Background(), "user-1", "12345678903")
	if err != repository.ErrOrderExistsOther {
		t.Fatalf("want ErrOrderExistsOther, got %v", err)
	}
}

func TestOrderService_ListOrders(t *testing.T) {
	svc := NewOrderService(&fakeOrderRepoSimple{})
	orders, err := svc.ListOrders(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if orders == nil {
		t.Fatal("expected non-nil slice")
	}
}
