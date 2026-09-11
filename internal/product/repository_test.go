package product

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/ThDawnWind/food-delivery-api/internal/database"
	"github.com/jackc/pgx/v5"
)

func TestRepository_GetByID(t *testing.T) {
	ctx := context.Background()

	dbURL := os.Getenv("TEST_DATABASE_URL")

	dbPool, err := database.New(ctx, dbURL)
	if err != nil {
		t.Fatalf("failed to connect to database: %v", err)
	}

	repo := NewRepository(dbPool)

	var categoryID int64

	suffix := time.Now().UnixNano()

	categoryName := fmt.Sprintf("Test Category %d", suffix)
	categorySlug := fmt.Sprintf("test-category-%d", suffix)

	err = dbPool.QueryRow(
		ctx,
		`
		INSERT INTO categories (name, slug)
		VALUES ($1, $2)
		RETURNING id
		`,
		categoryName,
		categorySlug,
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
			description,
			price,
			weight,
			category_id
		)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id
		`,
		"Pepperoni",
		"Spicy pizza",
		int64(59900),
		450,
		categoryID,
	).Scan(&productID)

	if err != nil {
		t.Fatalf("failed to create product: %v", err)
	}

	t.Cleanup(func() {
		ctx := context.Background()

		_, err := dbPool.Exec(ctx, `DELETE FROM products WHERE id = $1`, productID)
		if err != nil {
			t.Errorf("failed to clean up product: %v", err)
		}

		_, err = dbPool.Exec(ctx, `DELETE FROM categories WHERE id = $1`, categoryID)
		if err != nil {
			t.Errorf("failed to clean up category: %v", err)
		}
	})

	_, err = dbPool.Exec(
		ctx,
		`
		INSERT INTO product_images (
			product_id,
			url,
			sort_order,
			is_primary
		)
		VALUES
			($1, $2, 0, TRUE),
			($1, $3, 1, FALSE)
		`,
		productID,
		"/images/pepperoni-1.webp",
		"/images/pepperoni-2.webp",
	)

	if err != nil {
		t.Fatalf("failed to create product images: %v", err)
	}

	product, err := repo.GetByID(ctx, productID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if product == nil {
		t.Fatal("expected product, got nil")
	}

	if product.ID != productID {
		t.Errorf("expected ID %d, got %d", productID, product.ID)
	}

	if product.Name != "Pepperoni" {
		t.Errorf("expected name %q, got %q", "Pepperoni", product.Name)
	}

	if product.Description == nil {
		t.Fatal("expected description, got nil")
	}

	if *product.Description != "Spicy pizza" {
		t.Errorf(
			"expected description %q, got %q",
			"Spicy pizza",
			*product.Description,
		)
	}

	if product.Price != 59900 {
		t.Errorf("expected price %d, got %d", 59900, product.Price)
	}

	if product.Weight != 450 {
		t.Errorf("expected weight %d, got %d", 450, product.Weight)
	}

	if product.CategoryID != categoryID {
		t.Errorf(
			"expected category ID %d, got %d",
			categoryID,
			product.CategoryID,
		)
	}

	if !product.IsActive {
		t.Error("expected product to be active")
	}

	if len(product.Images) != 2 {
		t.Fatalf(
			"expected %d images, got %d",
			2,
			len(product.Images),
		)
	}

	if product.Images[0].URL != "/images/pepperoni-1.webp" {
		t.Errorf(
			"expected first image URL %q, got %q",
			"/images/pepperoni-1.webp",
			product.Images[0].URL,
		)
	}

	if !product.Images[0].IsPrimary {
		t.Error("expected first image to be primary")
	}

	if product.Images[0].SortOrder != 0 {
		t.Errorf(
			"expected first image sort order %d, got %d",
			0,
			product.Images[0].SortOrder,
		)
	}

	if product.Images[1].URL != "/images/pepperoni-2.webp" {
		t.Errorf(
			"expected second image URL %q, got %q",
			"/images/pepperoni-2.webp",
			product.Images[1].URL,
		)
	}

	if product.Images[1].IsPrimary {
		t.Error("expected second image not to be primary")
	}

	if product.Images[1].SortOrder != 1 {
		t.Errorf(
			"expected second image sort order %d, got %d",
			1,
			product.Images[1].SortOrder,
		)
	}
}

func TestRepository_GetByID_NotFound(t *testing.T) {
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

	product, err := repo.GetByID(ctx, 999999999)

	if product != nil {
		t.Fatalf("expected nil product, got %+v", product)
	}

	if !errors.Is(err, pgx.ErrNoRows) {
		t.Fatalf(
			"expected pgx.ErrNoRows, got %v",
			err,
		)
	}
}
