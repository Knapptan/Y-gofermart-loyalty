package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/knapptan/go-musthave-diploma-tpl/internal/auth"
	"github.com/knapptan/go-musthave-diploma-tpl/internal/middleware"
	"github.com/knapptan/go-musthave-diploma-tpl/internal/model"
	"github.com/knapptan/go-musthave-diploma-tpl/internal/repository"
	"github.com/knapptan/go-musthave-diploma-tpl/internal/service"
)

type fakeBalanceRepo struct {
	balances    map[string]*model.Balance
	withdrawals map[string][]model.Withdrawal
}

func newFakeBalanceRepo() *fakeBalanceRepo {
	return &fakeBalanceRepo{
		balances:    map[string]*model.Balance{},
		withdrawals: map[string][]model.Withdrawal{},
	}
}

func (f *fakeBalanceRepo) Get(_ context.Context, userID string) (*model.Balance, error) {
	b, ok := f.balances[userID]
	if !ok {
		return &model.Balance{}, nil
	}
	return b, nil
}

func (f *fakeBalanceRepo) Withdraw(_ context.Context, userID, order string, sum float64) error {
	b, ok := f.balances[userID]
	if !ok {
		b = &model.Balance{}
		f.balances[userID] = b
	}
	if b.Current < sum {
		return repository.ErrInsufficientFunds
	}
	b.Current -= sum
	b.Withdrawn += sum
	f.withdrawals[userID] = append(f.withdrawals[userID], model.Withdrawal{
		Order: order, Sum: sum, ProcessedAt: time.Now(),
	})
	return nil
}

func (f *fakeBalanceRepo) ListWithdrawals(_ context.Context, userID string) ([]model.Withdrawal, error) {
	return f.withdrawals[userID], nil
}

func newBalanceRouter(repo repository.BalanceRepository) http.Handler {
	svc := service.NewBalanceService(repo)
	h := NewBalanceHandler(svc)
	mw := middleware.Auth(testSecret)
	mux := http.NewServeMux()
	mux.Handle("/balance", mw(http.HandlerFunc(h.Get)))
	mux.Handle("/balance/withdraw", mw(http.HandlerFunc(h.Withdraw)))
	mux.Handle("/withdrawals", mw(http.HandlerFunc(h.ListWithdrawals)))
	return mux
}

func TestBalance_GetZero(t *testing.T) {
	h := newBalanceRouter(newFakeBalanceRepo())
	req := httptest.NewRequest(http.MethodGet, "/balance", nil)
	req.Header.Set("Authorization", "Bearer "+tokenFor(t, "u1"))
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", rr.Code)
	}
	var b model.Balance
	if err := json.Unmarshal(rr.Body.Bytes(), &b); err != nil {
		t.Fatal(err)
	}
	if b.Current != 0 || b.Withdrawn != 0 {
		t.Fatalf("unexpected balance: %+v", b)
	}
}

func TestBalance_Withdraw_Success(t *testing.T) {
	repo := newFakeBalanceRepo()
	repo.balances["u1"] = &model.Balance{Current: 500}
	h := newBalanceRouter(repo)

	body, _ := json.Marshal(map[string]any{"order": "2377225624", "sum": 100})
	req := httptest.NewRequest(http.MethodPost, "/balance/withdraw", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+tokenFor(t, "u1"))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("want 200, got %d (%s)", rr.Code, rr.Body.String())
	}
	if repo.balances["u1"].Current != 400 {
		t.Fatalf("expected 400, got %v", repo.balances["u1"].Current)
	}
}

func TestBalance_Withdraw_Insufficient(t *testing.T) {
	repo := newFakeBalanceRepo()
	repo.balances["u1"] = &model.Balance{Current: 10}
	h := newBalanceRouter(repo)

	body, _ := json.Marshal(map[string]any{"order": "2377225624", "sum": 100})
	req := httptest.NewRequest(http.MethodPost, "/balance/withdraw", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+tokenFor(t, "u1"))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusPaymentRequired {
		t.Fatalf("want 402, got %d", rr.Code)
	}
}

func TestBalance_Withdraw_BadLuhn(t *testing.T) {
	h := newBalanceRouter(newFakeBalanceRepo())
	body, _ := json.Marshal(map[string]any{"order": "123", "sum": 10})
	req := httptest.NewRequest(http.MethodPost, "/balance/withdraw", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+tokenFor(t, "u1"))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnprocessableEntity {
		t.Fatalf("want 422, got %d", rr.Code)
	}
}

func TestBalance_Withdrawals_Empty(t *testing.T) {
	h := newBalanceRouter(newFakeBalanceRepo())
	req := httptest.NewRequest(http.MethodGet, "/withdrawals", nil)
	req.Header.Set("Authorization", "Bearer "+tokenFor(t, "u1"))
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Fatalf("want 204, got %d", rr.Code)
	}
}

func TestBalance_Withdrawals_WithData(t *testing.T) {
	repo := newFakeBalanceRepo()
	repo.balances["u1"] = &model.Balance{Current: 500}
	repo.withdrawals["u1"] = []model.Withdrawal{
		{Order: "2377225624", Sum: 100, ProcessedAt: time.Now()},
	}
	h := newBalanceRouter(repo)

	req := httptest.NewRequest(http.MethodGet, "/withdrawals", nil)
	req.Header.Set("Authorization", "Bearer "+tokenFor(t, "u1"))
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", rr.Code)
	}
	var list []model.Withdrawal
	if err := json.Unmarshal(rr.Body.Bytes(), &list); err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 || list[0].Order != "2377225624" {
		t.Fatalf("unexpected list: %+v", list)
	}
}

// защита от неиспользуемых импортов
var _ = auth.NewToken
