package order

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/ThDawnWind/food-delivery-api/internal/database"
)

func TestRepository_Create(t *testing.T) {
	ctx := context.Background()

	dbURL := os.Getenv("TEST_DATABASE_URL")

	dbPool, err := database.New(ctx, dbURL)
	if err != nil {
		t.Fatalf("failed to connect to database: %v", err)
	}

	t.Cleanup(func() {
		dbPool.Close()
	})

	repo := NewRepository(dbPool)

	suffix := time.Now().UnixNano()

	var userID int64

	err = dbPool.QueryRow(
		ctx,
		`
		INSERT INTO users (
			username,
			email,
			password_hash
		)
		VALUES ($1, $2, $3)
		RETURNING id
		`,
		fmt.Sprintf("order-test-%d", suffix),
		fmt.Sprintf("order-test-%d@example.com", suffix),
		"test-password-hash",
	).Scan(&userID)

	if err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	var categoryID int64

	err = dbPool.QueryRow(
		ctx,
		`
		INSERT INTO categories (name, slug)
		VALUES ($1, $2)
		RETURNING id
		`,
		fmt.Sprintf("Order Category %d", suffix),
		fmt.Sprintf("order-category-%d", suffix),
	).Scan(&categoryID)

	if err != nil {
		t.Fatalf("failed to create category: %v", err)
	}

	var pizzaID int64
	var burgerID int64

	err = dbPool.QueryRow(
		ctx,
		`
		INSERT INTO products (
			name,
			price,
			weight,
			category_id
		)
		VALUES ($1, $2, $3, $4)
		RETURNING id
		`,
		"Pepperoni",
		int64(59900),
		450,
		categoryID,
	).Scan(&pizzaID)

	if err != nil {
		t.Fatalf("failed to create pizza: %v", err)
	}

	err = dbPool.QueryRow(
		ctx,
		`
		INSERT INTO products (
			name,
			price,
			weight,
			category_id
		)
		VALUES ($1, $2, $3, $4)
		RETURNING id
		`,
		"Burger",
		int64(39900),
		300,
		categoryID,
	).Scan(&burgerID)

	if err != nil {
		t.Fatalf("failed to create burger: %v", err)
	}

	t.Cleanup(func() {
		ctx := context.Background()

		_, err := dbPool.Exec(
			ctx,
			`DELETE FROM orders WHERE user_id = $1`,
			userID,
		)
		if err != nil {
			t.Errorf("failed to clean up orders: %v", err)
		}

		_, err = dbPool.Exec(
			ctx,
			`DELETE FROM products WHERE id = ANY($1::bigint[])`,
			[]int64{pizzaID, burgerID},
		)
		if err != nil {
			t.Errorf("failed to clean up products: %v", err)
		}

		_, err = dbPool.Exec(
			ctx,
			`DELETE FROM categories WHERE id = $1`,
			categoryID,
		)
		if err != nil {
			t.Errorf("failed to clean up category: %v", err)
		}

		_, err = dbPool.Exec(
			ctx,
			`DELETE FROM users WHERE id = $1`,
			userID,
		)
		if err != nil {
			t.Errorf("failed to clean up user: %v", err)
		}
	})

	order := &Order{
		UserID:          userID,
		Status:          StatusNew,
		TotalPrice:      159700,
		DeliveryAddress: "Test street 1",
		Items: []OrderItem{
			{
				ProductID:     pizzaID,
				NameSnapshot:  "Pepperoni",
				PriceSnapshot: 59900,
				Quantity:      2,
			},
			{
				ProductID:     burgerID,
				NameSnapshot:  "Burger",
				PriceSnapshot: 39900,
				Quantity:      1,
			},
		},
	}

	createdOrder, err := repo.Create(ctx, order)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if createdOrder == nil {
		t.Fatal("expected created order, got nil")
	}

	if createdOrder.ID == 0 {
		t.Error("expected order ID to be set")
	}

	if createdOrder.CreatedAt.IsZero() {
		t.Error("expected order created_at to be set")
	}

	if createdOrder.UpdatedAt.IsZero() {
		t.Error("expected order updated_at to be set")
	}

	if len(createdOrder.Items) != 2 {
		t.Fatalf(
			"expected %d order items, got %d",
			2,
			len(createdOrder.Items),
		)
	}

	for i, item := range createdOrder.Items {
		if item.ID == 0 {
			t.Errorf(
				"expected item %d ID to be set",
				i,
			)
		}

		if item.OrderID != createdOrder.ID {
			t.Errorf(
				"expected item %d order ID %d, got %d",
				i,
				createdOrder.ID,
				item.OrderID,
			)
		}

		if item.CreatedAt.IsZero() {
			t.Errorf(
				"expected item %d created_at to be set",
				i,
			)
		}
	}

	var savedTotalPrice int64
	var savedStatus Status
	var savedAddress string

	err = dbPool.QueryRow(
		ctx,
		`
		SELECT
			total_price,
			status,
			delivery_address
		FROM orders
		WHERE id = $1
		`,
		createdOrder.ID,
	).Scan(
		&savedTotalPrice,
		&savedStatus,
		&savedAddress,
	)

	if err != nil {
		t.Fatalf("failed to get saved order: %v", err)
	}

	if savedTotalPrice != 159700 {
		t.Errorf(
			"expected total price %d, got %d",
			159700,
			savedTotalPrice,
		)
	}

	if savedStatus != StatusNew {
		t.Errorf(
			"expected status %q, got %q",
			StatusNew,
			savedStatus,
		)
	}

	if savedAddress != "Test street 1" {
		t.Errorf(
			"expected delivery address %q, got %q",
			"Test street 1",
			savedAddress,
		)
	}

	var itemCount int

	err = dbPool.QueryRow(
		ctx,
		`
		SELECT COUNT(*)
		FROM order_items
		WHERE order_id = $1
		`,
		createdOrder.ID,
	).Scan(&itemCount)

	if err != nil {
		t.Fatalf("failed to count order items: %v", err)
	}

	if itemCount != 2 {
		t.Errorf(
			"expected %d order items in database, got %d",
			2,
			itemCount,
		)
	}
	var (
		savedNameSnapshot  string
		savedPriceSnapshot int64
		savedQuantity      int
	)

	err = dbPool.QueryRow(
		ctx,
		`
	SELECT
		name_snapshot,
		price_snapshot,
		quantity
	FROM order_items
	WHERE order_id = $1
	  AND product_id = $2
	`,
		createdOrder.ID,
		pizzaID,
	).Scan(
		&savedNameSnapshot,
		&savedPriceSnapshot,
		&savedQuantity,
	)

	if err != nil {
		t.Fatalf("failed to get saved order item: %v", err)
	}

	if savedNameSnapshot != "Pepperoni" {
		t.Errorf(
			"expected name snapshot %q, got %q",
			"Pepperoni",
			savedNameSnapshot,
		)
	}

	if savedPriceSnapshot != 59900 {
		t.Errorf(
			"expected price snapshot %d, got %d",
			59900,
			savedPriceSnapshot,
		)
	}

	if savedQuantity != 2 {
		t.Errorf(
			"expected quantity %d, got %d",
			2,
			savedQuantity,
		)
	}
}

func TestRepository_Create_RollbackOnItemError(t *testing.T) {
	ctx := context.Background()

	dbURL := os.Getenv("TEST_DATABASE_URL")

	dbPool, err := database.New(ctx, dbURL)
	if err != nil {
		t.Fatalf("failed to connect to database: %v", err)
	}

	t.Cleanup(func() {
		dbPool.Close()
	})

	repo := NewRepository(dbPool)

	suffix := time.Now().UnixNano()

	var userID int64

	err = dbPool.QueryRow(
		ctx,
		`
		INSERT INTO users (
			username,
			email,
			password_hash
		)
		VALUES ($1, $2, $3)
		RETURNING id
		`,
		fmt.Sprintf("rollback-user-%d", suffix),
		fmt.Sprintf("rollback-user-%d@example.com", suffix),
		"test-password-hash",
	).Scan(&userID)

	if err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	var categoryID int64

	err = dbPool.QueryRow(
		ctx,
		`
		INSERT INTO categories (name, slug)
		VALUES ($1, $2)
		RETURNING id
		`,
		fmt.Sprintf("Rollback Category %d", suffix),
		fmt.Sprintf("rollback-category-%d", suffix),
	).Scan(&categoryID)

	if err != nil {
		t.Fatalf("failed to create category: %v", err)
	}

	var productID int64

	err = dbPool.QueryRow(
		ctx,
		`
		INSERT INTO products (
			name,
			price,
			weight,
			category_id
		)
		VALUES ($1, $2, $3, $4)
		RETURNING id
		`,
		fmt.Sprintf("Rollback Product %d", suffix),
		int64(59900),
		450,
		categoryID,
	).Scan(&productID)

	if err != nil {
		t.Fatalf("failed to create product: %v", err)
	}

	t.Cleanup(func() {
		ctx := context.Background()

		_, err := dbPool.Exec(
			ctx,
			`DELETE FROM orders WHERE user_id = $1`,
			userID,
		)
		if err != nil {
			t.Errorf("failed to clean up orders: %v", err)
		}

		_, err = dbPool.Exec(
			ctx,
			`DELETE FROM products WHERE id = $1`,
			productID,
		)
		if err != nil {
			t.Errorf("failed to clean up product: %v", err)
		}

		_, err = dbPool.Exec(
			ctx,
			`DELETE FROM categories WHERE id = $1`,
			categoryID,
		)
		if err != nil {
			t.Errorf("failed to clean up category: %v", err)
		}

		_, err = dbPool.Exec(
			ctx,
			`DELETE FROM users WHERE id = $1`,
			userID,
		)
		if err != nil {
			t.Errorf("failed to clean up user: %v", err)
		}
	})

	order := &Order{
		UserID:          userID,
		Status:          StatusNew,
		TotalPrice:      119800,
		DeliveryAddress: "Rollback street 1",
		Items: []OrderItem{
			{
				ProductID:     productID,
				NameSnapshot:  "Rollback Product",
				PriceSnapshot: 59900,
				Quantity:      1,
			},
			{
				ProductID:     productID,
				NameSnapshot:  "Rollback Product",
				PriceSnapshot: 59900,
				Quantity:      1,
			},
		},
	}

	createdOrder, err := repo.Create(ctx, order)

	if err == nil {
		t.Fatal("expected create error, got nil")
	}

	if createdOrder != nil {
		t.Fatalf(
			"expected nil order, got %+v",
			createdOrder,
		)
	}

	var orderCount int

	err = dbPool.QueryRow(
		ctx,
		`
		SELECT COUNT(*)
		FROM orders
		WHERE user_id = $1
		  AND delivery_address = $2
		`,
		userID,
		"Rollback street 1",
	).Scan(&orderCount)

	if err != nil {
		t.Fatalf("failed to count orders: %v", err)
	}

	if orderCount != 0 {
		t.Fatalf(
			"expected order rollback, found %d orders",
			orderCount,
		)
	}

	var itemCount int

	err = dbPool.QueryRow(
		ctx,
		`
		SELECT COUNT(*)
		FROM order_items oi
		JOIN orders o ON o.id = oi.order_id
		WHERE o.user_id = $1
		  AND o.delivery_address = $2
		`,
		userID,
		"Rollback street 1",
	).Scan(&itemCount)

	if err != nil {
		t.Fatalf("failed to count order items: %v", err)
	}

	if itemCount != 0 {
		t.Fatalf(
			"expected order items rollback, found %d items",
			itemCount,
		)
	}
}
