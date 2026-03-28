package repository

import (
	"context"

	"github.com/jackc/pgx/v5"
)

type OrderRepository interface {
	CreateOrderWithEvent(ctx context.Context, item string, eventType string, payload []byte) error
}

type postgresOrderRepo struct {
	db *pgx.Conn
}

func NewPostgresOrderRepo(db *pgx.Conn) OrderRepository {
	return &postgresOrderRepo{db: db}
}

func (r *postgresOrderRepo) CreateOrderWithEvent(ctx context.Context, item string, eventType string, payload []byte) error {
	return pgx.BeginFunc(ctx, r.db, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, "INSERT INTO orders (item) VALUES ($1)", item)
		if err != nil {
			return err
		}

		_, err = tx.Exec(ctx, "INSERT INTO outbox_events (event_type, payload) VALUES ($1, $2)", eventType, payload)
		return err
	})
}
