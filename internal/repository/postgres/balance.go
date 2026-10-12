package postgres

import (
	"context"
	"database/sql"
	"errors"

	"github.com/knapptan/go-musthave-diploma-tpl/internal/model"
	"github.com/knapptan/go-musthave-diploma-tpl/internal/repository"
)

// BalanceRepo — PostgreSQL-реализация BalanceRepository.
type BalanceRepo struct {
	db *sql.DB
}

// NewBalanceRepo создаёт BalanceRepo.
func NewBalanceRepo(db *sql.DB) *BalanceRepo { return &BalanceRepo{db: db} }

// Get возвращает баланс; если записи нет — создаёт с нулями.
func (r *BalanceRepo) Get(ctx context.Context, userID string) (*model.Balance, error) {
	var b model.Balance
	err := r.db.QueryRowContext(ctx,
		`SELECT current, withdrawn FROM balances WHERE user_id = $1`, userID,
	).Scan(&b.Current, &b.Withdrawn)

	if errors.Is(err, sql.ErrNoRows) {
		// Пользователь есть, но записи баланса ещё нет — создаём.
		if _, err := r.db.ExecContext(ctx,
			`INSERT INTO balances (user_id) VALUES ($1) ON CONFLICT DO NOTHING`, userID,
		); err != nil {
			return nil, err
		}
		return &model.Balance{}, nil
	}
	if err != nil {
		return nil, err
	}
	return &b, nil
}

// Withdraw транзакционное списание: SELECT FOR UPDATE, проверка средств,
// обновление current/withdrawn, запись в withdrawals.
func (r *BalanceRepo) Withdraw(ctx context.Context, userID, order string, sum float64) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Убедимся, что строка баланса существует.
	if _, err := tx.ExecContext(ctx,
		`INSERT INTO balances (user_id) VALUES ($1) ON CONFLICT DO NOTHING`, userID,
	); err != nil {
		return err
	}

	var current float64
	err = tx.QueryRowContext(ctx,
		`SELECT current FROM balances WHERE user_id = $1 FOR UPDATE`, userID,
	).Scan(&current)
	if err != nil {
		return err
	}

	if current < sum {
		return repository.ErrInsufficientFunds
	}

	if _, err := tx.ExecContext(ctx,
		`UPDATE balances
		 SET current = current - $1, withdrawn = withdrawn + $1
		 WHERE user_id = $2`,
		sum, userID,
	); err != nil {
		return err
	}

	if _, err := tx.ExecContext(ctx,
		`INSERT INTO withdrawals (user_id, order_number, sum) VALUES ($1, $2, $3)`,
		userID, order, sum,
	); err != nil {
		return err
	}

	return tx.Commit()
}

// ListWithdrawals возвращает списания пользователя (DESC по processed_at).
func (r *BalanceRepo) ListWithdrawals(ctx context.Context, userID string) ([]model.Withdrawal, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT order_number, sum, processed_at
		 FROM withdrawals WHERE user_id = $1
		 ORDER BY processed_at DESC`, userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []model.Withdrawal
	for rows.Next() {
		var w model.Withdrawal
		if err := rows.Scan(&w.Order, &w.Sum, &w.ProcessedAt); err != nil {
			return nil, err
		}
		out = append(out, w)
	}
	return out, rows.Err()
}
