package postgres

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/knapptan/go-musthave-diploma-tpl/internal/model"
	"github.com/knapptan/go-musthave-diploma-tpl/internal/repository"
)

func TestOrderCreate_Success(t *testing.T) {
	truncate(t, "withdrawals", "balances", "orders", "users")
	uid := seedUser(t, "u1")

	if err := NewOrderRepo(testDB).Create(context.Background(), uid, "12345678903"); err != nil {
		t.Fatalf("Create: %v", err)
	}

	o, err := NewOrderRepo(testDB).GetByNumber(context.Background(), "12345678903")
	if err != nil {
		t.Fatalf("GetByNumber: %v", err)
	}
	if o.Status != model.StatusNew {
		t.Fatalf("status=%s, want NEW", o.Status)
	}
}

func TestOrderCreate_DuplicateSameUser(t *testing.T) {
	truncate(t, "withdrawals", "balances", "orders", "users")
	uid := seedUser(t, "u1")

	repo := NewOrderRepo(testDB)
	ctx := context.Background()
	if err := repo.Create(ctx, uid, "111"); err != nil {
		t.Fatalf("first: %v", err)
	}
	err := repo.Create(ctx, uid, "111")
	if !errors.Is(err, repository.ErrOrderExistsSame) {
		t.Fatalf("err=%v, want ErrOrderExistsSame", err)
	}
}

func TestOrderCreate_DuplicateOtherUser(t *testing.T) {
	truncate(t, "withdrawals", "balances", "orders", "users")
	u1 := seedUser(t, "u1")
	u2 := seedUser(t, "u2")

	repo := NewOrderRepo(testDB)
	ctx := context.Background()
	if err := repo.Create(ctx, u1, "222"); err != nil {
		t.Fatalf("first: %v", err)
	}
	err := repo.Create(ctx, u2, "222")
	if !errors.Is(err, repository.ErrOrderExistsOther) {
		t.Fatalf("err=%v, want ErrOrderExistsOther", err)
	}
}

func TestOrderListByUser(t *testing.T) {
	truncate(t, "withdrawals", "balances", "orders", "users")
	uid := seedUser(t, "u1")

	ctx := context.Background()
	// Явно расставляем uploaded_at, чтобы гарантировать порядок.
	for i, num := range []string{"100", "200", "300"} {
		_, err := testDB.ExecContext(ctx,
			`INSERT INTO orders (number, user_id, status, uploaded_at)
			 VALUES ($1, $2, 'NEW', $3)`,
			num, uid, time.Now().Add(time.Duration(i)*time.Second),
		)
		if err != nil {
			t.Fatalf("insert: %v", err)
		}
	}

	orders, err := NewOrderRepo(testDB).ListByUser(ctx, uid)
	if err != nil {
		t.Fatalf("ListByUser: %v", err)
	}
	if len(orders) != 3 {
		t.Fatalf("len=%d, want 3", len(orders))
	}
	if orders[0].Number != "300" || orders[2].Number != "100" {
		t.Fatalf("wrong order: %v %v %v", orders[0].Number, orders[1].Number, orders[2].Number)
	}
}

func TestOrderApplyResult_Processed(t *testing.T) {
	truncate(t, "withdrawals", "balances", "orders", "users")
	uid := seedUser(t, "u1")

	repo := NewOrderRepo(testDB)
	ctx := context.Background()
	if err := repo.Create(ctx, uid, "123"); err != nil {
		t.Fatalf("Create: %v", err)
	}

	if err := repo.ApplyResult(ctx, "123", model.StatusProcessed, 500); err != nil {
		t.Fatalf("ApplyResult: %v", err)
	}

	o, _ := repo.GetByNumber(ctx, "123")
	if o.Status != model.StatusProcessed || o.Accrual != 500 {
		t.Fatalf("got status=%s accrual=%v", o.Status, o.Accrual)
	}

	var current float64
	if err := testDB.QueryRowContext(ctx,
		`SELECT current FROM balances WHERE user_id = $1`, uid,
	).Scan(&current); err != nil {
		t.Fatalf("balance: %v", err)
	}
	if current != 500 {
		t.Fatalf("balance=%v, want 500", current)
	}
}

func TestOrderApplyResult_Idempotent(t *testing.T) {
	truncate(t, "withdrawals", "balances", "orders", "users")
	uid := seedUser(t, "u1")

	repo := NewOrderRepo(testDB)
	ctx := context.Background()
	_ = repo.Create(ctx, uid, "123")
	_ = repo.ApplyResult(ctx, "123", model.StatusProcessed, 500)
	// повторный вызов — не должен начислить ещё раз
	if err := repo.ApplyResult(ctx, "123", model.StatusProcessed, 500); err != nil {
		t.Fatalf("second ApplyResult: %v", err)
	}

	var current float64
	_ = testDB.QueryRowContext(ctx,
		`SELECT current FROM balances WHERE user_id = $1`, uid,
	).Scan(&current)
	if current != 500 {
		t.Fatalf("balance=%v, want 500 (idempotent)", current)
	}
}

func TestOrderApplyResult_Invalid(t *testing.T) {
	truncate(t, "withdrawals", "balances", "orders", "users")
	uid := seedUser(t, "u1")

	repo := NewOrderRepo(testDB)
	ctx := context.Background()
	_ = repo.Create(ctx, uid, "123")
	if err := repo.ApplyResult(ctx, "123", model.StatusInvalid, 0); err != nil {
		t.Fatalf("ApplyResult: %v", err)
	}

	o, _ := repo.GetByNumber(ctx, "123")
	if o.Status != model.StatusInvalid || o.Accrual != 0 {
		t.Fatalf("got status=%s accrual=%v", o.Status, o.Accrual)
	}

	var cnt int
	_ = testDB.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM balances WHERE user_id = $1`, uid,
	).Scan(&cnt)
	if cnt != 0 {
		t.Fatalf("balances row created on INVALID: %d", cnt)
	}
}
