package postgres

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/knapptan/go-musthave-diploma-tpl/internal/repository"
)

func TestBalanceGet_CreateIfMissing(t *testing.T) {
	truncate(t, "withdrawals", "balances", "orders", "users")
	uid := seedUser(t, "u1")

	b, err := NewBalanceRepo(testDB).Get(context.Background(), uid)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if b.Current != 0 || b.Withdrawn != 0 {
		t.Fatalf("got %+v, want zero balance", b)
	}
}

func TestBalanceGet_ReturnExisting(t *testing.T) {
	truncate(t, "withdrawals", "balances", "orders", "users")
	uid := seedUser(t, "u1")

	ctx := context.Background()
	if _, err := testDB.ExecContext(ctx,
		`INSERT INTO balances (user_id, current, withdrawn) VALUES ($1, 500, 100)`, uid,
	); err != nil {
		t.Fatalf("insert balance: %v", err)
	}

	b, err := NewBalanceRepo(testDB).Get(ctx, uid)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if b.Current != 500 || b.Withdrawn != 100 {
		t.Fatalf("got %+v", b)
	}
}

func TestBalanceWithdraw_Success(t *testing.T) {
	truncate(t, "withdrawals", "balances", "orders", "users")
	uid := seedUser(t, "u1")

	ctx := context.Background()
	_, _ = testDB.ExecContext(ctx,
		`INSERT INTO balances (user_id, current) VALUES ($1, 500)`, uid)

	if err := NewBalanceRepo(testDB).Withdraw(ctx, uid, "order-1", 200); err != nil {
		t.Fatalf("Withdraw: %v", err)
	}

	b, _ := NewBalanceRepo(testDB).Get(ctx, uid)
	if b.Current != 300 || b.Withdrawn != 200 {
		t.Fatalf("got %+v", b)
	}

	var cnt int
	_ = testDB.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM withdrawals WHERE user_id = $1`, uid).Scan(&cnt)
	if cnt != 1 {
		t.Fatalf("withdrawals cnt=%d, want 1", cnt)
	}
}

func TestBalanceWithdraw_InsufficientFunds(t *testing.T) {
	truncate(t, "withdrawals", "balances", "orders", "users")
	uid := seedUser(t, "u1")

	ctx := context.Background()
	_, _ = testDB.ExecContext(ctx,
		`INSERT INTO balances (user_id, current) VALUES ($1, 10)`, uid)

	err := NewBalanceRepo(testDB).Withdraw(ctx, uid, "order-1", 100)
	if !errors.Is(err, repository.ErrInsufficientFunds) {
		t.Fatalf("err=%v, want ErrInsufficientFunds", err)
	}
}

func TestBalanceWithdraw_AfterCreate(t *testing.T) {
	truncate(t, "withdrawals", "balances", "orders", "users")
	uid := seedUser(t, "u1")

	ctx := context.Background()
	// Строки баланса нет, но сумма 0 — должна создаться и списание пройти (на 0).
	if err := NewBalanceRepo(testDB).Withdraw(ctx, uid, "order-1", 0); err != nil {
		t.Fatalf("Withdraw(0): %v", err)
	}

	var cnt int
	_ = testDB.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM balances WHERE user_id = $1`, uid).Scan(&cnt)
	if cnt != 1 {
		t.Fatalf("balance row not created")
	}
}

func TestBalanceListWithdrawals(t *testing.T) {
	truncate(t, "withdrawals", "balances", "orders", "users")
	uid := seedUser(t, "u1")

	ctx := context.Background()
	for i, sum := range []float64{100, 200} {
		_, err := testDB.ExecContext(ctx,
			`INSERT INTO withdrawals (user_id, order_number, sum, processed_at)
			 VALUES ($1, $2, $3, $4)`,
			uid, "order", sum, time.Now().Add(time.Duration(i)*time.Second),
		)
		if err != nil {
			t.Fatalf("insert: %v", err)
		}
	}

	ws, err := NewBalanceRepo(testDB).ListWithdrawals(ctx, uid)
	if err != nil {
		t.Fatalf("ListWithdrawals: %v", err)
	}
	if len(ws) != 2 {
		t.Fatalf("len=%d, want 2", len(ws))
	}
	if ws[0].Sum != 200 || ws[1].Sum != 100 {
		t.Fatalf("wrong order: %v %v", ws[0].Sum, ws[1].Sum)
	}
}
