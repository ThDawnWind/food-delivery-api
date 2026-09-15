package order

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{
		db: db,
	}
}

func (r *Repository) Create(ctx context.Context, order *Order) (*Order, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf(
			"failed to begin transaction: %w",
			err,
		)
	}

	defer tx.Rollback(ctx)

	err = tx.QueryRow(
		ctx,
		`
			INSERT INTO orders (
				user_id,
				status,
				total_price,
				delivery_address
			)
			VALUES ($1, $2, $3, $4)
			RETURNING
				id,
				created_at,
				updated_at
			`,
		order.UserID,
		order.Status,
		order.TotalPrice,
		order.DeliveryAddress,
	).Scan(
		&order.ID,
		&order.CreatedAt,
		&order.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf(
			"failed to create order: %w",
			err,
		)
	}

	for i := range order.Items {
		order.Items[i].OrderID = order.ID

		err = tx.QueryRow(
			ctx,
			`
					INSERT INTO order_items (
						order_id,
						product_id,
						name_snapshot,
						price_snapshot,
						quantity
					)
					VALUES ($1, $2, $3, $4, $5)
					RETURNING
						id,
						created_at
					`,
			order.ID,
			order.Items[i].ProductID,
			order.Items[i].NameSnapshot,
			order.Items[i].PriceSnapshot,
			order.Items[i].Quantity,
		).Scan(
			&order.Items[i].ID,
			&order.Items[i].CreatedAt,
		)

		if err != nil {
			return nil, fmt.Errorf(
				"failed to create order item: %w",
				err,
			)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf(
			"failed to commit transaction: %w",
			err,
		)
	}

	return order, nil
}
