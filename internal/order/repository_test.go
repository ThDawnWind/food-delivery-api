package order

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/ThDawnWind/food-delivery-api/internal/database"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

func TestRepository_Create(t *testing.T) {
	ctx := context.Background()

	dbURL := os.Getenv("TEST_DATABASE_URL")

	dbPool, err := database.New(
		ctx,
		database.Config{
			URL: dbURL,
		},
	)
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

	dbPool, err := database.New(
		ctx,
		database.Config{
			URL: dbURL,
		},
	)
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

func TestRepository_GetByID(t *testing.T) {
	ctx := context.Background()

	dbURL := os.Getenv("TEST_DATABASE_URL")

	dbPool, err := database.New(
		ctx,
		database.Config{
			URL: dbURL,
		},
	)
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
		fmt.Sprintf("get-order-user-%d", suffix),
		fmt.Sprintf("get-order-user-%d@example.com", suffix),
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
		fmt.Sprintf("Get Order Category %d", suffix),
		fmt.Sprintf("get-order-category-%d", suffix),
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

		_, _ = dbPool.Exec(
			ctx,
			`DELETE FROM orders WHERE user_id = $1`,
			userID,
		)

		_, _ = dbPool.Exec(
			ctx,
			`DELETE FROM products WHERE id = ANY($1::bigint[])`,
			[]int64{pizzaID, burgerID},
		)

		_, _ = dbPool.Exec(
			ctx,
			`DELETE FROM categories WHERE id = $1`,
			categoryID,
		)

		_, _ = dbPool.Exec(
			ctx,
			`DELETE FROM users WHERE id = $1`,
			userID,
		)
	})

	createdOrder, err := repo.Create(
		ctx,
		&Order{
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
		},
	)

	if err != nil {
		t.Fatalf("failed to create order: %v", err)
	}

	gotOrder, err := repo.GetByID(
		ctx,
		createdOrder.ID,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if gotOrder == nil {
		t.Fatal("expected order, got nil")
	}

	if gotOrder.ID != createdOrder.ID {
		t.Errorf(
			"expected order ID %d, got %d",
			createdOrder.ID,
			gotOrder.ID,
		)
	}

	if gotOrder.UserID != userID {
		t.Errorf(
			"expected user ID %d, got %d",
			userID,
			gotOrder.UserID,
		)
	}

	if gotOrder.Status != StatusNew {
		t.Errorf(
			"expected status %q, got %q",
			StatusNew,
			gotOrder.Status,
		)
	}

	if gotOrder.TotalPrice != 159700 {
		t.Errorf(
			"expected total price %d, got %d",
			159700,
			gotOrder.TotalPrice,
		)
	}

	if gotOrder.DeliveryAddress != "Test street 1" {
		t.Errorf(
			"expected address %q, got %q",
			"Test street 1",
			gotOrder.DeliveryAddress,
		)
	}

	if len(gotOrder.Items) != 2 {
		t.Fatalf(
			"expected %d items, got %d",
			2,
			len(gotOrder.Items),
		)
	}

	firstItem := gotOrder.Items[0]

	if firstItem.OrderID != gotOrder.ID {
		t.Errorf(
			"expected order ID %d, got %d",
			gotOrder.ID,
			firstItem.OrderID,
		)
	}

	if firstItem.ProductID != pizzaID {
		t.Errorf(
			"expected product ID %d, got %d",
			pizzaID,
			firstItem.ProductID,
		)
	}

	if firstItem.NameSnapshot != "Pepperoni" {
		t.Errorf(
			"expected name snapshot %q, got %q",
			"Pepperoni",
			firstItem.NameSnapshot,
		)
	}

	if firstItem.PriceSnapshot != 59900 {
		t.Errorf(
			"expected price snapshot %d, got %d",
			59900,
			firstItem.PriceSnapshot,
		)
	}

	if firstItem.Quantity != 2 {
		t.Errorf(
			"expected quantity %d, got %d",
			2,
			firstItem.Quantity,
		)
	}
}

func TestRepository_GetByID_NotFound(t *testing.T) {
	ctx := context.Background()

	dbURL := os.Getenv("TEST_DATABASE_URL")

	dbPool, err := database.New(
		ctx,
		database.Config{
			URL: dbURL,
		},
	)
	if err != nil {
		t.Fatalf("failed to connect to database: %v", err)
	}

	t.Cleanup(func() {
		dbPool.Close()
	})

	repo := NewRepository(dbPool)

	order, err := repo.GetByID(ctx, 999999999)

	if order != nil {
		t.Fatalf(
			"expected nil order, got %+v",
			order,
		)
	}

	if !errors.Is(err, pgx.ErrNoRows) {
		t.Fatalf(
			"expected pgx.ErrNoRows, got %v",
			err,
		)
	}
}

func TestRepository_ListByUser(t *testing.T) {
	ctx := context.Background()

	dbURL := os.Getenv("TEST_DATABASE_URL")

	dbPool, err := database.New(
		ctx,
		database.Config{
			URL: dbURL,
		},
	)
	if err != nil {
		t.Fatalf("failed to connect to database: %v", err)
	}

	t.Cleanup(func() {
		dbPool.Close()
	})

	repo := NewRepository(dbPool)

	suffix := time.Now().UnixNano()

	var userID int64
	var otherUserID int64

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
		fmt.Sprintf("list-user-%d", suffix),
		fmt.Sprintf("list-user-%d@example.com", suffix),
		"test-password-hash",
	).Scan(&userID)
	if err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

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
		fmt.Sprintf("other-user-%d", suffix),
		fmt.Sprintf("other-user-%d@example.com", suffix),
		"test-password-hash",
	).Scan(&otherUserID)
	if err != nil {
		t.Fatalf("failed to create other user: %v", err)
	}

	var categoryID int64

	err = dbPool.QueryRow(
		ctx,
		`
		INSERT INTO categories (name, slug)
		VALUES ($1, $2)
		RETURNING id
		`,
		fmt.Sprintf("List Order Category %d", suffix),
		fmt.Sprintf("list-order-category-%d", suffix),
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
		"Pepperoni",
		int64(59900),
		450,
		categoryID,
	).Scan(&productID)
	if err != nil {
		t.Fatalf("failed to create product: %v", err)
	}

	t.Cleanup(func() {
		ctx := context.Background()

		_, _ = dbPool.Exec(
			ctx,
			`DELETE FROM orders WHERE user_id = ANY($1::bigint[])`,
			[]int64{userID, otherUserID},
		)

		_, _ = dbPool.Exec(
			ctx,
			`DELETE FROM products WHERE id = $1`,
			productID,
		)

		_, _ = dbPool.Exec(
			ctx,
			`DELETE FROM categories WHERE id = $1`,
			categoryID,
		)

		_, _ = dbPool.Exec(
			ctx,
			`DELETE FROM users WHERE id = ANY($1::bigint[])`,
			[]int64{userID, otherUserID},
		)
	})

	firstOrder, err := repo.Create(
		ctx,
		&Order{
			UserID:          userID,
			Status:          StatusNew,
			TotalPrice:      59900,
			DeliveryAddress: "First street",
			Items: []OrderItem{
				{
					ProductID:     productID,
					NameSnapshot:  "Pepperoni",
					PriceSnapshot: 59900,
					Quantity:      1,
				},
			},
		},
	)
	if err != nil {
		t.Fatalf("failed to create first order: %v", err)
	}

	secondOrder, err := repo.Create(
		ctx,
		&Order{
			UserID:          userID,
			Status:          StatusNew,
			TotalPrice:      119800,
			DeliveryAddress: "Second street",
			Items: []OrderItem{
				{
					ProductID:     productID,
					NameSnapshot:  "Pepperoni",
					PriceSnapshot: 59900,
					Quantity:      2,
				},
			},
		},
	)
	if err != nil {
		t.Fatalf("failed to create second order: %v", err)
	}

	_, err = repo.Create(
		ctx,
		&Order{
			UserID:          otherUserID,
			Status:          StatusNew,
			TotalPrice:      59900,
			DeliveryAddress: "Other street",
			Items: []OrderItem{
				{
					ProductID:     productID,
					NameSnapshot:  "Pepperoni",
					PriceSnapshot: 59900,
					Quantity:      1,
				},
			},
		},
	)
	if err != nil {
		t.Fatalf("failed to create other user order: %v", err)
	}

	orders, err := repo.ListByUser(
		ctx,
		userID,
		20,
		0,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(orders) != 2 {
		t.Fatalf(
			"expected %d orders, got %d",
			2,
			len(orders),
		)
	}

	if orders[0].ID != secondOrder.ID {
		t.Errorf(
			"expected first order ID %d, got %d",
			secondOrder.ID,
			orders[0].ID,
		)
	}

	if orders[1].ID != firstOrder.ID {
		t.Errorf(
			"expected second order ID %d, got %d",
			firstOrder.ID,
			orders[1].ID,
		)
	}

	for _, order := range orders {
		if order.UserID != userID {
			t.Errorf(
				"expected user ID %d, got %d",
				userID,
				order.UserID,
			)
		}
	}

	if len(orders[0].Items) != 1 {
		t.Fatalf(
			"expected %d item in first order, got %d",
			1,
			len(orders[0].Items),
		)
	}

	if orders[0].Items[0].OrderID != secondOrder.ID {
		t.Errorf(
			"expected item order ID %d, got %d",
			secondOrder.ID,
			orders[0].Items[0].OrderID,
		)
	}

	if orders[0].Items[0].Quantity != 2 {
		t.Errorf(
			"expected quantity %d, got %d",
			2,
			orders[0].Items[0].Quantity,
		)
	}

	firstPage, err := repo.ListByUser(
		ctx,
		userID,
		1,
		0,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(firstPage) != 1 {
		t.Fatalf(
			"expected %d order, got %d",
			1,
			len(firstPage),
		)
	}

	if firstPage[0].ID != secondOrder.ID {
		t.Errorf(
			"expected order ID %d, got %d",
			secondOrder.ID,
			firstPage[0].ID,
		)
	}

	secondPage, err := repo.ListByUser(
		ctx,
		userID,
		1,
		1,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(secondPage) != 1 {
		t.Fatalf(
			"expected %d order, got %d",
			1,
			len(secondPage),
		)
	}

	if secondPage[0].ID != firstOrder.ID {
		t.Errorf(
			"expected order ID %d, got %d",
			firstOrder.ID,
			secondPage[0].ID,
		)
	}
}

func TestRepository_ListByUser_Empty(t *testing.T) {
	ctx := context.Background()

	dbURL := os.Getenv("TEST_DATABASE_URL")

	dbPool, err := database.New(
		ctx,
		database.Config{
			URL: dbURL,
		},
	)
	if err != nil {
		t.Fatalf(
			"failed to connect to database: %v",
			err,
		)
	}

	t.Cleanup(func() {
		dbPool.Close()
	})

	repo := NewRepository(dbPool)

	orders, err := repo.ListByUser(
		ctx,
		999999999,
		20,
		0,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if orders == nil {
		t.Fatal(
			"expected empty slice, got nil",
		)
	}

	if len(orders) != 0 {
		t.Fatalf(
			"expected empty orders, got %d",
			len(orders),
		)
	}
}

func TestRepository_UpdateStatus_NotFound(t *testing.T) {
	ctx := context.Background()

	dbURL := os.Getenv("TEST_DATABASE_URL")

	dbPool, err := database.New(
		ctx,
		database.Config{
			URL: dbURL,
		},
	)
	if err != nil {
		t.Fatalf(
			"failed to connect to database: %v",
			err,
		)
	}

	t.Cleanup(func() {
		dbPool.Close()
	})

	repo := NewRepository(dbPool)

	err = repo.UpdateStatus(
		ctx,
		999999999,
		StatusNew,
		StatusConfirmed,
	)

	if !errors.Is(err, pgx.ErrNoRows) {
		t.Fatalf(
			"expected pgx.ErrNoRows, got %v",
			err,
		)
	}
}

func TestRepository_UpdateStatus(t *testing.T) {
	ctx := context.Background()

	dbURL := os.Getenv("TEST_DATABASE_URL")

	dbPool, err := database.New(
		ctx,
		database.Config{
			URL: dbURL,
		},
	)
	if err != nil {
		t.Fatalf(
			"failed to connect to database: %v",
			err,
		)
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
		fmt.Sprintf("status-user-%d", suffix),
		fmt.Sprintf("status-user-%d@example.com", suffix),
		"test-password-hash",
	).Scan(&userID)
	if err != nil {
		t.Fatalf(
			"failed to create user: %v",
			err,
		)
	}

	var orderID int64
	var createdAt time.Time

	err = dbPool.QueryRow(
		ctx,
		`
		INSERT INTO orders (
			user_id,
			status,
			total_price,
			delivery_address
		)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at
		`,
		userID,
		StatusNew,
		int64(59900),
		"Test street 1",
	).Scan(
		&orderID,
		&createdAt,
	)
	if err != nil {
		t.Fatalf(
			"failed to create order: %v",
			err,
		)
	}

	t.Cleanup(func() {
		ctx := context.Background()

		_, _ = dbPool.Exec(
			ctx,
			`DELETE FROM orders WHERE id = $1`,
			orderID,
		)

		_, _ = dbPool.Exec(
			ctx,
			`DELETE FROM users WHERE id = $1`,
			userID,
		)
	})

	err = repo.UpdateStatus(
		ctx,
		orderID,
		StatusNew,
		StatusConfirmed,
	)
	if err != nil {
		t.Fatalf(
			"unexpected update status error: %v",
			err,
		)
	}

	var (
		status    Status
		updatedAt time.Time
	)

	err = dbPool.QueryRow(
		ctx,
		`
		SELECT
			status,
			updated_at
		FROM orders
		WHERE id = $1
		`,
		orderID,
	).Scan(
		&status,
		&updatedAt,
	)
	if err != nil {
		t.Fatalf(
			"failed to get updated order: %v",
			err,
		)
	}

	if status != StatusConfirmed {
		t.Errorf(
			"expected status %q, got %q",
			StatusConfirmed,
			status,
		)
	}

	if updatedAt.Before(createdAt) {
		t.Errorf(
			"expected updated_at not to be before created_at",
		)
	}
}

func TestRepository_UpdateStatus_StatusChanged(t *testing.T) {
	pool := newTestPool(t)

	ctx := context.Background()

	var userID int64

	err := pool.QueryRow(
		ctx,
		`
			INSERT INTO users (
				username,
				email,
				password_hash,
				role
			)
			VALUES ($1, $2, $3, $4)
			RETURNING id
		`,
		"status_conflict_user",
		"status_conflict@example.com",
		"test_password_hash",
		"user",
	).Scan(&userID)
	if err != nil {
		t.Fatalf(
			"failed to insert test user: %v",
			err,
		)
	}

	var orderID int64

	err = pool.QueryRow(
		ctx,
		`
			INSERT INTO orders (
				user_id,
				status,
				total_price,
				delivery_address
			)
			VALUES ($1, $2, $3, $4)
			RETURNING id
		`,
		userID,
		StatusCancelled,
		1000,
		"Test street 1",
	).Scan(&orderID)
	if err != nil {
		t.Fatalf(
			"failed to insert test order: %v",
			err,
		)
	}

	t.Cleanup(func() {
		_, _ = pool.Exec(
			context.Background(),
			`DELETE FROM orders WHERE id = $1`,
			orderID,
		)

		_, _ = pool.Exec(
			context.Background(),
			`DELETE FROM users WHERE id = $1`,
			userID,
		)
	})

	repository := NewRepository(pool)

	err = repository.UpdateStatus(
		ctx,
		orderID,
		StatusConfirmed,
		StatusCooking,
	)

	if !errors.Is(err, pgx.ErrNoRows) {
		t.Fatalf(
			"expected pgx.ErrNoRows, got %v",
			err,
		)
	}

	var status Status

	err = pool.QueryRow(
		ctx,
		`
			SELECT status
			FROM orders
			WHERE id = $1
		`,
		orderID,
	).Scan(&status)
	if err != nil {
		t.Fatalf(
			"failed to get order status: %v",
			err,
		)
	}

	if status != StatusCancelled {
		t.Fatalf(
			"expected status to remain %q, got %q",
			StatusCancelled,
			status,
		)
	}
}

func TestRepository_ListAll(t *testing.T) {
	ctx := context.Background()

	db := newTestPool(t)
	repository := NewRepository(db)

	suffix := time.Now().UnixNano()

	var firstUserID int64

	err := db.QueryRow(
		ctx,
		`
		INSERT INTO users (
			username,
			email,
			password_hash,
			role
		)
		VALUES ($1, $2, $3, $4)
		RETURNING id
		`,
		fmt.Sprintf("list_all_user_1_%d", suffix),
		fmt.Sprintf("list_all_user_1_%d@example.com", suffix),
		"test-password-hash",
		"user",
	).Scan(&firstUserID)
	if err != nil {
		t.Fatalf(
			"failed to create first user: %v",
			err,
		)
	}

	var secondUserID int64

	err = db.QueryRow(
		ctx,
		`
		INSERT INTO users (
			username,
			email,
			password_hash,
			role
		)
		VALUES ($1, $2, $3, $4)
		RETURNING id
		`,
		fmt.Sprintf("list_all_user_2_%d", suffix),
		fmt.Sprintf("list_all_user_2_%d@example.com", suffix),
		"test-password-hash",
		"user",
	).Scan(&secondUserID)
	if err != nil {
		t.Fatalf(
			"failed to create second user: %v",
			err,
		)
	}

	var firstOrderID int64

	err = db.QueryRow(
		ctx,
		`
		INSERT INTO orders (
			user_id,
			status,
			total_price,
			delivery_address
		)
		VALUES ($1, $2, $3, $4)
		RETURNING id
		`,
		firstUserID,
		"new",
		10000,
		"First test address",
	).Scan(&firstOrderID)
	if err != nil {
		t.Fatalf(
			"failed to create first order: %v",
			err,
		)
	}

	var secondOrderID int64

	err = db.QueryRow(
		ctx,
		`
		INSERT INTO orders (
			user_id,
			status,
			total_price,
			delivery_address
		)
		VALUES ($1, $2, $3, $4)
		RETURNING id
		`,
		secondUserID,
		"new",
		20000,
		"Second test address",
	).Scan(&secondOrderID)
	if err != nil {
		t.Fatalf(
			"failed to create second order: %v",
			err,
		)
	}

	t.Cleanup(func() {
		_, _ = db.Exec(
			context.Background(),
			`
			DELETE FROM orders
			WHERE id = $1 OR id = $2
			`,
			firstOrderID,
			secondOrderID,
		)

		_, _ = db.Exec(
			context.Background(),
			`
			DELETE FROM users
			WHERE id = $1 OR id = $2
			`,
			firstUserID,
			secondUserID,
		)
	})

	orders, err := repository.ListAll(
		ctx,
		ListOrdersFilter{
			Limit:  100,
			Offset: 0,
		},
	)
	if err != nil {
		t.Fatalf(
			"unexpected error: %v",
			err,
		)
	}

	var firstFound bool
	var secondFound bool

	for _, order := range orders {
		switch order.ID {
		case firstOrderID:
			firstFound = true

			if order.UserID != firstUserID {
				t.Errorf(
					"expected first user ID %d, got %d",
					firstUserID,
					order.UserID,
				)
			}

		case secondOrderID:
			secondFound = true

			if order.UserID != secondUserID {
				t.Errorf(
					"expected second user ID %d, got %d",
					secondUserID,
					order.UserID,
				)
			}
		}
	}

	if !firstFound {
		t.Errorf(
			"expected order %d to be returned",
			firstOrderID,
		)
	}

	if !secondFound {
		t.Errorf(
			"expected order %d to be returned",
			secondOrderID,
		)
	}
}

func newTestPool(t *testing.T) *pgxpool.Pool {
	t.Helper()

	if os.Getenv("TEST_DATABASE_URL") == "" {
		_ = godotenv.Load("../../.env.test")
	}

	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Fatal(
			"TEST_DATABASE_URL is not set",
		)
	}

	pool, err := pgxpool.New(
		context.Background(),
		databaseURL,
	)
	if err != nil {
		t.Fatalf(
			"failed to create test database pool: %v",
			err,
		)
	}

	t.Cleanup(func() {
		pool.Close()
	})

	return pool
}

func TestRepository_ListAll_FilterByStatus(t *testing.T) {
	ctx := context.Background()

	db := newTestPool(t)
	repository := NewRepository(db)

	suffix := time.Now().UnixNano()

	var userID int64

	err := db.QueryRow(
		ctx,
		`
		INSERT INTO users (
			username,
			email,
			password_hash,
			role
		)
		VALUES ($1, $2, $3, $4)
		RETURNING id
		`,
		fmt.Sprintf("status_filter_user_%d", suffix),
		fmt.Sprintf("status_filter_user_%d@example.com", suffix),
		"test-password-hash",
		"user",
	).Scan(&userID)
	if err != nil {
		t.Fatalf(
			"failed to create user: %v",
			err,
		)
	}

	var newOrderID int64

	err = db.QueryRow(
		ctx,
		`
		INSERT INTO orders (
			user_id,
			status,
			total_price,
			delivery_address
		)
		VALUES ($1, $2, $3, $4)
		RETURNING id
		`,
		userID,
		"new",
		10000,
		"New order address",
	).Scan(&newOrderID)
	if err != nil {
		t.Fatalf(
			"failed to create new order: %v",
			err,
		)
	}

	var confirmedOrderID int64

	err = db.QueryRow(
		ctx,
		`
		INSERT INTO orders (
			user_id,
			status,
			total_price,
			delivery_address
		)
		VALUES ($1, $2, $3, $4)
		RETURNING id
		`,
		userID,
		"confirmed",
		20000,
		"Confirmed order address",
	).Scan(&confirmedOrderID)
	if err != nil {
		t.Fatalf(
			"failed to create confirmed order: %v",
			err,
		)
	}

	t.Cleanup(func() {
		_, _ = db.Exec(
			context.Background(),
			`
			DELETE FROM orders
			WHERE id = $1 OR id = $2
			`,
			newOrderID,
			confirmedOrderID,
		)

		_, _ = db.Exec(
			context.Background(),
			`
			DELETE FROM users
			WHERE id = $1
			`,
			userID,
		)
	})

	orders, err := repository.ListAll(
		ctx,
		ListOrdersFilter{
			Status: "confirmed",
			Limit:  100,
			Offset: 0,
		},
	)
	if err != nil {
		t.Fatalf(
			"unexpected error: %v",
			err,
		)
	}

	var confirmedFound bool

	for _, order := range orders {
		if order.Status != "confirmed" {
			t.Fatalf(
				"expected only confirmed orders, got status %q",
				order.Status,
			)
		}

		if order.ID == confirmedOrderID {
			confirmedFound = true
		}

		if order.ID == newOrderID {
			t.Fatalf(
				"new order %d must not be returned",
				newOrderID,
			)
		}
	}

	if !confirmedFound {
		t.Fatalf(
			"expected confirmed order %d to be returned",
			confirmedOrderID,
		)
	}
}

func TestRepository_ListAll_FilterByUserID(t *testing.T) {
	pool := newTestPool(t)
	repository := NewRepository(pool)

	ctx := context.Background()
	suffix := time.Now().UnixNano()

	var userID1 int64

	err := pool.QueryRow(
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
		fmt.Sprintf("user1-%d", suffix),
		fmt.Sprintf("user1-%d@example.com", suffix),
		"hash",
	).Scan(&userID1)
	if err != nil {
		t.Fatalf("failed to create first user: %v", err)
	}

	var userID2 int64

	err = pool.QueryRow(
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
		fmt.Sprintf("user2-%d", suffix),
		fmt.Sprintf("user2-%d@example.com", suffix),
		"hash",
	).Scan(&userID2)
	if err != nil {
		t.Fatalf("failed to create second user: %v", err)
	}

	t.Cleanup(func() {
		_, _ = pool.Exec(
			context.Background(),
			`DELETE FROM orders WHERE user_id IN ($1, $2)`,
			userID1,
			userID2,
		)

		_, _ = pool.Exec(
			context.Background(),
			`DELETE FROM users WHERE id IN ($1, $2)`,
			userID1,
			userID2,
		)
	})

	var orderID1 int64

	err = pool.QueryRow(
		ctx,
		`
		INSERT INTO orders (
			user_id,
			status,
			total_price,
			delivery_address
		)
		VALUES ($1, $2, $3, $4)
		RETURNING id
		`,
		userID1,
		StatusNew,
		59900,
		"Address 1",
	).Scan(&orderID1)
	if err != nil {
		t.Fatalf("failed to create first order: %v", err)
	}

	var orderID2 int64

	err = pool.QueryRow(
		ctx,
		`
		INSERT INTO orders (
			user_id,
			status,
			total_price,
			delivery_address
		)
		VALUES ($1, $2, $3, $4)
		RETURNING id
		`,
		userID2,
		StatusNew,
		59900,
		"Address 2",
	).Scan(&orderID2)
	if err != nil {
		t.Fatalf("failed to create second order: %v", err)
	}

	orders, err := repository.ListAll(
		ctx,
		ListOrdersFilter{
			UserID: &userID1,
			Limit:  100,
			Offset: 0,
		},
	)
	if err != nil {
		t.Fatalf("ListAll returned error: %v", err)
	}

	found := false

	for _, order := range orders {
		if order.UserID != userID1 {
			t.Fatalf(
				"expected only user_id %d, got order with user_id %d",
				userID1,
				order.UserID,
			)
		}

		if order.ID == orderID1 {
			found = true
		}
	}

	if !found {
		t.Fatalf(
			"expected order %d to be returned",
			orderID1,
		)
	}
}
func TestRepository_ListAll_FilterByCreatedAt(t *testing.T) {
	pool := newTestPool(t)
	repository := NewRepository(pool)

	ctx := context.Background()

	from := time.Now().Add(-24 * time.Hour)
	to := time.Now().Add(24 * time.Hour)

	orders, err := repository.ListAll(
		ctx,
		ListOrdersFilter{
			CreatedFrom: &from,
			CreatedTo:   &to,
			Limit:       100,
			Offset:      0,
		},
	)
	if err != nil {
		t.Fatalf(
			"ListAll returned error: %v",
			err,
		)
	}

	for _, order := range orders {
		if order.CreatedAt.Before(from) {
			t.Errorf(
				"order %d created_at %v is before %v",
				order.ID,
				order.CreatedAt,
				from,
			)
		}

		if order.CreatedAt.After(to) {
			t.Errorf(
				"order %d created_at %v is after %v",
				order.ID,
				order.CreatedAt,
				to,
			)
		}
	}
}

func TestRepository_CountAll(t *testing.T) {
	pool := newTestPool(t)
	repository := NewRepository(pool)

	ctx := context.Background()

	total, err := repository.CountAll(
		ctx,
		ListOrdersFilter{},
	)
	if err != nil {
		t.Fatalf(
			"CountAll returned error: %v",
			err,
		)
	}

	if total < 0 {
		t.Fatalf(
			"expected non-negative total, got %d",
			total,
		)
	}
}

func TestRepository_CountAll_FilterByStatus(t *testing.T) {
	pool := newTestPool(t)
	repository := NewRepository(pool)

	ctx := context.Background()

	total, err := repository.CountAll(
		ctx,
		ListOrdersFilter{
			Status: "cooking",
		},
	)
	if err != nil {
		t.Fatalf(
			"CountAll returned error: %v",
			err,
		)
	}

	orders, err := repository.ListAll(
		ctx,
		ListOrdersFilter{
			Status: "cooking",
			Limit:  100,
			Offset: 0,
		},
	)
	if err != nil {
		t.Fatalf(
			"ListAll returned error: %v",
			err,
		)
	}

	if total != int64(len(orders)) {
		t.Errorf(
			"expected total %d, got %d",
			len(orders),
			total,
		)
	}
}

func TestRepository_CancelByUser(t *testing.T) {
	pool := newTestPool(t)

	ctx := context.Background()

	var userID int64

	err := pool.QueryRow(
		ctx,
		`
		INSERT INTO users (
			username,
			email,
			password_hash,
			role
		)
		VALUES ($1, $2, $3, $4)
		RETURNING id
	`,
		"cancel_test_user",
		"cancel_test@example.com",
		"test_password_hash",
		"user",
	).Scan(&userID)
	if err != nil {
		t.Fatalf(
			"failed to insert test user: %v",
			err,
		)
	}

	var orderID int64

	err = pool.QueryRow(
		ctx,
		`
			INSERT INTO orders (
				user_id,
				status,
				total_price,
				delivery_address
			)
			VALUES ($1, $2, $3, $4)
			RETURNING id
		`,
		userID,
		StatusNew,
		1000,
		"Test street 1",
	).Scan(&orderID)
	if err != nil {
		t.Fatalf(
			"failed to insert test order: %v",
			err,
		)
	}

	t.Cleanup(func() {
		_, _ = pool.Exec(
			context.Background(),
			`DELETE FROM orders WHERE id = $1`,
			orderID,
		)

		_, _ = pool.Exec(
			context.Background(),
			`DELETE FROM users WHERE id = $1`,
			userID,
		)
	})
	repository := NewRepository(pool)

	err = repository.CancelByUser(
		ctx,
		orderID,
		userID,
	)
	if err != nil {
		t.Fatalf(
			"unexpected error: %v",
			err,
		)
	}

	var status Status

	err = pool.QueryRow(
		ctx,
		`
			SELECT status
			FROM orders
			WHERE id = $1
		`,
		orderID,
	).Scan(&status)
	if err != nil {
		t.Fatalf(
			"failed to get order status: %v",
			err,
		)
	}

	if status != StatusCancelled {
		t.Fatalf(
			"expected status %q, got %q",
			StatusCancelled,
			status,
		)
	}
}

func TestRepository_CancelByUser_Confirmed(t *testing.T) {
	pool := newTestPool(t)

	ctx := context.Background()

	var userID int64

	err := pool.QueryRow(
		ctx,
		`
		INSERT INTO users (
			username,
			email,
			password_hash,
			role
		)
		VALUES ($1, $2, $3, $4)
		RETURNING id
	`,
		"cancel_test_user",
		"cancel_test@example.com",
		"test_password_hash",
		"user",
	).Scan(&userID)
	if err != nil {
		t.Fatalf(
			"failed to insert test user: %v",
			err,
		)
	}

	var orderID int64

	err = pool.QueryRow(
		ctx,
		`
			INSERT INTO orders (
				user_id,
				status,
				total_price,
				delivery_address
			)
			VALUES ($1, $2, $3, $4)
			RETURNING id
		`,
		userID,
		StatusNew,
		1000,
		"Test street 1",
	).Scan(&orderID)
	if err != nil {
		t.Fatalf(
			"failed to insert test order: %v",
			err,
		)
	}

	t.Cleanup(func() {
		_, _ = pool.Exec(
			context.Background(),
			`DELETE FROM orders WHERE id = $1`,
			orderID,
		)

		_, _ = pool.Exec(
			context.Background(),
			`DELETE FROM users WHERE id = $1`,
			userID,
		)
	})

	repository := NewRepository(pool)

	err = repository.CancelByUser(
		ctx,
		orderID,
		userID,
	)
	if err != nil {
		t.Fatalf(
			"unexpected error: %v",
			err,
		)
	}
}

func TestRepository_CancelByUser_InvalidStatus(t *testing.T) {
	pool := newTestPool(t)

	ctx := context.Background()

	var userID int64

	err := pool.QueryRow(
		ctx,
		`
		INSERT INTO users (
			username,
			email,
			password_hash,
			role
		)
		VALUES ($1, $2, $3, $4)
		RETURNING id
	`,
		"cancel_test_user",
		"cancel_test@example.com",
		"test_password_hash",
		"user",
	).Scan(&userID)
	if err != nil {
		t.Fatalf(
			"failed to insert test user: %v",
			err,
		)
	}

	var orderID int64

	err = pool.QueryRow(
		ctx,
		`
			INSERT INTO orders (
				user_id,
				status,
				total_price,
				delivery_address
			)
			VALUES ($1, $2, $3, $4)
			RETURNING id
		`,
		userID,
		StatusCooking,
		1000,
		"Test street 1",
	).Scan(&orderID)
	if err != nil {
		t.Fatalf(
			"failed to insert test order: %v",
			err,
		)
	}

	t.Cleanup(func() {
		_, _ = pool.Exec(
			context.Background(),
			`DELETE FROM orders WHERE id = $1`,
			orderID,
		)

		_, _ = pool.Exec(
			context.Background(),
			`DELETE FROM users WHERE id = $1`,
			userID,
		)
	})

	repository := NewRepository(pool)

	err = repository.CancelByUser(
		ctx,
		orderID,
		userID,
	)

	if !errors.Is(
		err,
		ErrOrderCannotCancel,
	) {
		t.Fatalf(
			"expected ErrOrderCannotCancel, got %v",
			err,
		)
	}

	var status Status

	err = pool.QueryRow(
		ctx,
		`
			SELECT status
			FROM orders
			WHERE id = $1
		`,
		orderID,
	).Scan(&status)
	if err != nil {
		t.Fatalf(
			"failed to get order status: %v",
			err,
		)
	}

	if status != StatusCooking {
		t.Fatalf(
			"expected status to remain %q, got %q",
			StatusCooking,
			status,
		)
	}
}

func TestRepository_CancelByUser_OtherUser(t *testing.T) {
	pool := newTestPool(t)

	ctx := context.Background()

	var userID int64

	err := pool.QueryRow(
		ctx,
		`
		INSERT INTO users (
			username,
			email,
			password_hash,
			role
		)
		VALUES ($1, $2, $3, $4)
		RETURNING id
	`,
		"cancel_test_user",
		"cancel_test@example.com",
		"test_password_hash",
		"user",
	).Scan(&userID)
	if err != nil {
		t.Fatalf(
			"failed to insert test user: %v",
			err,
		)
	}

	var orderID int64

	err = pool.QueryRow(
		ctx,
		`
			INSERT INTO orders (
				user_id,
				status,
				total_price,
				delivery_address
			)
			VALUES ($1, $2, $3, $4)
			RETURNING id
		`,
		userID,
		StatusNew,
		1000,
		"Test street 1",
	).Scan(&orderID)
	if err != nil {
		t.Fatalf(
			"failed to insert test order: %v",
			err,
		)
	}

	t.Cleanup(func() {
		_, _ = pool.Exec(
			context.Background(),
			`DELETE FROM orders WHERE id = $1`,
			orderID,
		)

		_, _ = pool.Exec(
			context.Background(),
			`DELETE FROM users WHERE id = $1`,
			userID,
		)
	})
	repository := NewRepository(pool)

	err = repository.CancelByUser(
		ctx,
		orderID,
		userID+999,
	)

	if !errors.Is(err, pgx.ErrNoRows) {
		t.Fatalf(
			"expected pgx.ErrNoRows, got %v",
			err,
		)
	}
}
