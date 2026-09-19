package order

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
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

func (r *Repository) GetByID(ctx context.Context, id int64) (*Order, error) {
	order := &Order{}

	err := r.db.QueryRow(
		ctx,
		`
				SELECT
					id,
					user_id,
					status,
					total_price,
					delivery_address,
					created_at,
					updated_at
				FROM orders
				WHERE id = $1
				`,
		id,
	).Scan(
		&order.ID,
		&order.UserID,
		&order.Status,
		&order.TotalPrice,
		&order.DeliveryAddress,
		&order.CreatedAt,
		&order.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"failed to get order: %w",
			err,
		)
	}

	items, err := r.listItemsByOrderID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf(
			"failed to get order items: %w",
			err,
		)
	}

	order.Items = items

	return order, nil
}

func (r *Repository) listItemsByOrderID(ctx context.Context, orderID int64) ([]OrderItem, error) {
	rows, err := r.db.Query(
		ctx,
		`
				SELECT
					id,
					order_id,
					product_id,
					name_snapshot,
					price_snapshot,
					quantity,
					created_at
				FROM order_items
				WHERE order_id = $1
				ORDER BY id
				`,
		orderID,
	)

	if err != nil {
		return nil, fmt.Errorf(
			"failed to query order items: %w",
			err,
		)
	}

	defer rows.Close()

	items := make([]OrderItem, 0)

	for rows.Next() {
		var item OrderItem

		err := rows.Scan(
			&item.ID,
			&item.OrderID,
			&item.ProductID,
			&item.NameSnapshot,
			&item.PriceSnapshot,
			&item.Quantity,
			&item.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf(
				"failed to scan order item: %w",
				err,
			)
		}

		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf(
			"failed to iterate order items: %w",
			err,
		)
	}

	return items, nil
}

func (r *Repository) ListByUser(ctx context.Context, userID int64, limit, offset int) ([]Order, error) {
	rows, err := r.db.Query(
		ctx,
		`
				SELECT
					id,
					user_id,
					status,
					total_price,
					delivery_address,
					created_at,
					updated_at
				FROM orders
				WHERE user_id = $1
				ORDER BY created_at DESC, id DESC
				LIMIT $2
				OFFSET $3
				`,
		userID,
		limit,
		offset,
	)

	if err != nil {
		return nil, fmt.Errorf(
			"failed to query orders: %w",
			err,
		)
	}
	defer rows.Close()

	orders := make([]Order, 0)

	for rows.Next() {
		var order Order

		err := rows.Scan(
			&order.ID,
			&order.UserID,
			&order.Status,
			&order.TotalPrice,
			&order.DeliveryAddress,
			&order.CreatedAt,
			&order.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf(
				"failed to scan order: %w",
				err,
			)
		}

		order.Items = make([]OrderItem, 0)

		orders = append(orders, order)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf(
			"failed to iterate orders: %w",
			err,
		)
	}

	if len(orders) == 0 {
		return orders, nil
	}

	orderIDs := make([]int64, 0, len(orders))

	for _, order := range orders {
		orderIDs = append(orderIDs, order.ID)
	}

	itemsByOrderID, err := r.listItemsByOrderIDs(
		ctx,
		orderIDs,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"failed to get order items: %w",
			err,
		)
	}

	for i := range orders {
		items, ok := itemsByOrderID[orders[i].ID]
		if ok {
			orders[i].Items = items
		}
	}

	return orders, nil
}

func (r *Repository) listItemsByOrderIDs(ctx context.Context, orderIDs []int64) (map[int64][]OrderItem, error) {
	rows, err := r.db.Query(
		ctx,
		`
				SELECT
					id,
					order_id,
					product_id,
					name_snapshot,
					price_snapshot,
					quantity,
					created_at
				FROM order_items
				WHERE order_id = ANY($1::bigint[])
				ORDER BY order_id, id
				`,
		orderIDs,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"failed to query order items: %w",
			err,
		)
	}
	defer rows.Close()

	itemsByOrderID := make(map[int64][]OrderItem)

	for rows.Next() {
		var item OrderItem

		err := rows.Scan(
			&item.ID,
			&item.OrderID,
			&item.ProductID,
			&item.NameSnapshot,
			&item.PriceSnapshot,
			&item.Quantity,
			&item.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf(
				"failed to scan order item: %w",
				err,
			)
		}

		itemsByOrderID[item.OrderID] = append(
			itemsByOrderID[item.OrderID],
			item,
		)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf(
			"failed to iterate order items: %w",
			err,
		)
	}

	return itemsByOrderID, nil
}

func (r *Repository) UpdateStatus(ctx context.Context, id int64, status Status) error {
	result, err := r.db.Exec(
		ctx,
		`
				UPDATE orders
				SET
					status = $2,
					updated_at = NOW()
				WHERE id = $1
				`,
		id,
		status,
	)
	if err != nil {
		return fmt.Errorf(
			"failed to update order status: %w",
			err,
		)
	}

	if result.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (r *Repository) ListAll(ctx context.Context, status string, limit int, offset int) ([]*Order, error) {
	rows, err := r.db.Query(
		ctx,
		`
		SELECT
			id,
			user_id,
			status,
			total_price,
			delivery_address,
			created_at,
			updated_at
		FROM orders
		WHERE ($1 = '' OR status = $1)
		ORDER BY created_at DESC, id DESC
		LIMIT $2
		OFFSET $3
		`,
		status,
		limit,
		offset,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"failed to query orders: %w",
			err,
		)
	}
	defer rows.Close()

	orders := make([]*Order, 0)

	for rows.Next() {
		var order Order

		err := rows.Scan(
			&order.ID,
			&order.UserID,
			&order.Status,
			&order.TotalPrice,
			&order.DeliveryAddress,
			&order.CreatedAt,
			&order.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf(
				"failed to scan order: %w",
				err,
			)
		}

		order.Items = make([]OrderItem, 0)

		orders = append(
			orders,
			&order,
		)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf(
			"failed to iterate orders: %w",
			err,
		)
	}

	if len(orders) == 0 {
		return orders, nil
	}

	orderIDs := make([]int64, 0, len(orders))

	for _, order := range orders {
		orderIDs = append(
			orderIDs,
			order.ID,
		)
	}

	itemsByOrderID, err := r.listItemsByOrderIDs(
		ctx,
		orderIDs,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"failed to get order items: %w",
			err,
		)
	}

	for i := range orders {
		items, ok := itemsByOrderID[orders[i].ID]
		if ok {
			orders[i].Items = items
		}
	}

	return orders, nil
}
