package postgres

import (
	"context"
	"database/sql"
	"errors"

	"github.com/knapptan/go-musthave-diploma-tpl/internal/model"
	"github.com/knapptan/go-musthave-diploma-tpl/internal/repository"
)

type OrderRepo struct {
	db *sql.DB
}

func NewOrderRepo(db *sql.DB) *OrderRepo { return &OrderRepo{db: db} }

// Create вставляет заказ. Если номер уже есть:
//   - принадлежит этому же пользователю → ErrOrderExistsSame
//   - принадлежит другому → ErrOrderExistsOther
func (r *OrderRepo) Create(ctx context.Context, userID, number string) error {
	var inserted string
	err := r.db.QueryRowContext(ctx,
		`INSERT INTO orders (number, user_id, status)
		 VALUES ($1, $2, 'NEW')
		 ON CONFLICT (number) DO NOTHING
		 RETURNING number`,
		number, userID,
	).Scan(&inserted)

	if err == nil {
		return nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return err
	}

	// Конфликт — узнаём владельца.
	var owner string
	if err := r.db.QueryRowContext(ctx,
		`SELECT user_id FROM orders WHERE number = $1`, number,
	).Scan(&owner); err != nil {
		return err
	}
	if owner == userID {
		return repository.ErrOrderExistsSame
	}
	return repository.ErrOrderExistsOther
}

func (r *OrderRepo) GetByNumber(ctx context.Context, number string) (*model.Order, error) {
	var o model.Order
	err := r.db.QueryRowContext(ctx,
		`SELECT number, user_id, status, accrual, uploaded_at
		 FROM orders WHERE number = $1`, number,
	).Scan(&o.Number, &o.UserID, &o.Status, &o.Accrual, &o.UploadedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, repository.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &o, nil
}

func (r *OrderRepo) ListByUser(ctx context.Context, userID string) ([]model.Order, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT number, user_id, status, accrual, uploaded_at
		 FROM orders WHERE user_id = $1
		 ORDER BY uploaded_at DESC`, userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []model.Order
	for rows.Next() {
		var o model.Order
		if err := rows.Scan(&o.Number, &o.UserID, &o.Status, &o.Accrual, &o.UploadedAt); err != nil {
			return nil, err
		}
		out = append(out, o)
	}
	return out, rows.Err()
}

// ListPending возвращает заказы в статусах NEW/PROCESSING для воркера.
func (r *OrderRepo) ListPending(ctx context.Context) ([]model.Order, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT number, user_id, status, accrual, uploaded_at
		 FROM orders WHERE status IN ('NEW', 'PROCESSING')
		 ORDER BY uploaded_at ASC
		 LIMIT 100`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []model.Order
	for rows.Next() {
		var o model.Order
		if err := rows.Scan(&o.Number, &o.UserID, &o.Status, &o.Accrual, &o.UploadedAt); err != nil {
			return nil, err
		}
		out = append(out, o)
	}
	return out, rows.Err()
}

func (r *OrderRepo) UpdateStatus(ctx context.Context, number string, status model.OrderStatus, accrual float64) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE orders SET status = $1, accrual = $2 WHERE number = $3`,
		status, accrual, number,
	)
	return err
}

// ApplyResult атомарно обновляет статус заказа и начисляет баллы при PROCESSED.
// Идемпотентно: повторный вызов для уже финализированного заказа ничего не делает.
func (r *OrderRepo) ApplyResult(ctx context.Context, number string, status model.OrderStatus, accrual float64) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var userID string
	var prevStatus model.OrderStatus
	err = tx.QueryRowContext(ctx,
		`SELECT user_id, status FROM orders WHERE number = $1 FOR UPDATE`, number,
	).Scan(&userID, &prevStatus)
	if errors.Is(err, sql.ErrNoRows) {
		return repository.ErrNotFound
	}
	if err != nil {
		return err
	}

	// Финальные статусы не перезаписываем.
	if prevStatus == model.StatusProcessed || prevStatus == model.StatusInvalid {
		return nil
	}

	if _, err := tx.ExecContext(ctx,
		`UPDATE orders SET status = $1, accrual = $2 WHERE number = $3`,
		status, accrual, number,
	); err != nil {
		return err
	}

	// Начисляем баллы только при PROCESSED и положительном accrual.
	if status == model.StatusProcessed && accrual > 0 {
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO balances (user_id) VALUES ($1) ON CONFLICT DO NOTHING`, userID,
		); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx,
			`UPDATE balances SET current = current + $1 WHERE user_id = $2`,
			accrual, userID,
		); err != nil {
			return err
		}
	}

	return tx.Commit()
}
