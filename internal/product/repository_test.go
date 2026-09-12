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

func TestRepository_List(t *testing.T) {
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

	var categoryID int64

	err = dbPool.QueryRow(
		ctx,
		`
		INSERT INTO categories (name, slug)
		VALUES ($1, $2)
		RETURNING id
		`,
		fmt.Sprintf("Test Category %d", suffix),
		fmt.Sprintf("test-category-%d", suffix),
	).Scan(&categoryID)

	if err != nil {
		t.Fatalf("failed to create category: %v", err)
	}

	var product1ID int64
	var product2ID int64

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
		fmt.Sprintf("Pizza %d", suffix),
		"Pizza description",
		int64(49900),
		400,
		categoryID,
	).Scan(&product1ID)

	if err != nil {
		t.Fatalf("failed to create first product: %v", err)
	}

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
		fmt.Sprintf("Burger %d", suffix),
		"Burger description",
		int64(39900),
		300,
		categoryID,
	).Scan(&product2ID)

	if err != nil {
		t.Fatalf("failed to create second product: %v", err)
	}

	t.Cleanup(func() {
		ctx := context.Background()

		_, err := dbPool.Exec(
			ctx,
			`DELETE FROM products WHERE id = ANY($1::bigint[])`,
			[]int64{product1ID, product2ID},
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
			($1, '/images/pizza-1.webp', 0, TRUE),
			($1, '/images/pizza-2.webp', 1, FALSE),
			($2, '/images/burger-1.webp', 0, TRUE)
		`,
		product1ID,
		product2ID,
	)

	if err != nil {
		t.Fatalf("failed to create product images: %v", err)
	}

	products, err := repo.List(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(products) < 2 {
		t.Fatalf("expected at least 2 products, got %d", len(products))
	}

	var product1 *Product
	var product2 *Product

	for i := range products {
		switch products[i].ID {
		case product1ID:
			product1 = &products[i]
		case product2ID:
			product2 = &products[i]
		}
	}

	if product1 == nil {
		t.Fatal("expected first product in list")
	}

	if product2 == nil {
		t.Fatal("expected second product in list")
	}

	if len(product1.Images) != 2 {
		t.Fatalf(
			"expected first product to have %d images, got %d",
			2,
			len(product1.Images),
		)
	}

	if len(product2.Images) != 1 {
		t.Fatalf(
			"expected second product to have %d image, got %d",
			1,
			len(product2.Images),
		)
	}

	if product1.Images[0].URL != "/images/pizza-1.webp" {
		t.Errorf(
			"expected first image URL %q, got %q",
			"/images/pizza-1.webp",
			product1.Images[0].URL,
		)
	}

	if product1.Images[0].SortOrder != 0 {
		t.Errorf(
			"expected first image sort order %d, got %d",
			0,
			product1.Images[0].SortOrder,
		)
	}

	if !product1.Images[0].IsPrimary {
		t.Error("expected first image to be primary")
	}

	if product1.Images[1].URL != "/images/pizza-2.webp" {
		t.Errorf(
			"expected second image URL %q, got %q",
			"/images/pizza-2.webp",
			product1.Images[1].URL,
		)
	}

	if product1.Images[1].SortOrder != 1 {
		t.Errorf(
			"expected second image sort order %d, got %d",
			1,
			product1.Images[1].SortOrder,
		)
	}

	if product1.Images[1].IsPrimary {
		t.Error("expected second image not to be primary")
	}

	if product2.Images[0].URL != "/images/burger-1.webp" {
		t.Errorf(
			"expected burger image URL %q, got %q",
			"/images/burger-1.webp",
			product2.Images[0].URL,
		)
	}

	if !product2.Images[0].IsPrimary {
		t.Error("expected burger image to be primary")
	}

}

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

	var categoryID int64

	err = dbPool.QueryRow(
		ctx,
		`
		INSERT INTO categories (name, slug)
		VALUES ($1, $2)
		RETURNING id
		`,
		fmt.Sprintf("Create Category %d", suffix),
		fmt.Sprintf("create-category-%d", suffix),
	).Scan(&categoryID)

	if err != nil {
		t.Fatalf("failed to create category: %v", err)
	}

	description := "Test pizza"

	product := &Product{
		Name:        fmt.Sprintf("Create Pizza %d", suffix),
		Description: &description,
		Price:       59900,
		Weight:      450,
		CategoryID:  categoryID,
		IsActive:    true,
		Images: []ProductImage{
			{
				URL:       "/images/create-pizza-1.webp",
				SortOrder: 0,
				IsPrimary: true,
			},
			{
				URL:       "/images/create-pizza-2.webp",
				SortOrder: 1,
				IsPrimary: false,
			},
		},
	}

	createdProduct, err := repo.Create(ctx, product)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if createdProduct == nil {
		t.Fatal("expected created product, got nil")
	}

	t.Cleanup(func() {
		ctx := context.Background()

		_, err := dbPool.Exec(
			ctx,
			`DELETE FROM products WHERE id = $1`,
			createdProduct.ID,
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
	})

	if createdProduct.ID == 0 {
		t.Error("expected product ID to be set")
	}

	if createdProduct.Name != product.Name {
		t.Errorf(
			"expected name %q, got %q",
			product.Name,
			createdProduct.Name,
		)
	}

	if createdProduct.CreatedAt.IsZero() {
		t.Error("expected created_at to be set")
	}

	if createdProduct.UpdatedAt.IsZero() {
		t.Error("expected updated_at to be set")
	}

	if len(createdProduct.Images) != 2 {
		t.Fatalf(
			"expected %d images, got %d",
			2,
			len(createdProduct.Images),
		)
	}

	for i, image := range createdProduct.Images {
		if image.ID == 0 {
			t.Errorf("expected image %d ID to be set", i)
		}

		if image.ProductID != createdProduct.ID {
			t.Errorf(
				"expected image %d product ID %d, got %d",
				i,
				createdProduct.ID,
				image.ProductID,
			)
		}

		if image.CreatedAt.IsZero() {
			t.Errorf("expected image %d created_at to be set", i)
		}
	}
	savedProduct, err := repo.GetByID(ctx, createdProduct.ID)
	if err != nil {
		t.Fatalf("failed to get created product: %v", err)
	}

	if savedProduct.Name != product.Name {
		t.Errorf(
			"expected saved name %q, got %q",
			product.Name,
			savedProduct.Name,
		)
	}

	if len(savedProduct.Images) != 2 {
		t.Fatalf(
			"expected saved product to have %d images, got %d",
			2,
			len(savedProduct.Images),
		)
	}
}

func TestRepository_Create_RollbackOnImageError(t *testing.T) {
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

	t.Cleanup(func() {
		ctx := context.Background()

		_, err := dbPool.Exec(
			ctx,
			`DELETE FROM categories WHERE id = $1`,
			categoryID,
		)
		if err != nil {
			t.Errorf("failed to clean up category: %v", err)
		}
	})

	description := "Rollback pizza"
	productName := fmt.Sprintf("Rollback Pizza %d", suffix)

	product := &Product{
		Name:        productName,
		Description: &description,
		Price:       59900,
		Weight:      450,
		CategoryID:  categoryID,
		IsActive:    true,
		Images: []ProductImage{
			{
				URL:       "/images/rollback-1.webp",
				SortOrder: 0,
				IsPrimary: true,
			},
			{
				URL:       "/images/rollback-2.webp",
				SortOrder: 1,

				IsPrimary: true,
			},
		},
	}

	createdProduct, err := repo.Create(ctx, product)

	if err == nil {
		t.Fatal("expected create error, got nil")
	}

	if createdProduct != nil {
		t.Fatalf(
			"expected nil product on error, got %+v",
			createdProduct,
		)
	}

	var productCount int

	err = dbPool.QueryRow(
		ctx,
		`
	SELECT COUNT(*)
	FROM products
	WHERE name = $1
	`,
		productName,
	).Scan(&productCount)

	if err != nil {
		t.Fatalf("failed to count products: %v", err)
	}

	if productCount != 0 {
		t.Fatalf(
			"expected product to be rolled back, found %d",
			productCount,
		)
	}
}

func TestRepository_Update(t *testing.T) {
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

	var categoryID int64

	err = dbPool.QueryRow(
		ctx,
		`
		INSERT INTO categories (name, slug)
		VALUES ($1, $2)
		RETURNING id
		`,
		fmt.Sprintf("Update Category %d", suffix),
		fmt.Sprintf("update-category-%d", suffix),
	).Scan(&categoryID)

	if err != nil {
		t.Fatalf("failed to create category: %v", err)
	}

	oldDescription := "Old description"

	product := &Product{
		Name:        fmt.Sprintf("Old Pizza %d", suffix),
		Description: &oldDescription,
		Price:       49900,
		Weight:      400,
		CategoryID:  categoryID,
		IsActive:    true,
		Images: []ProductImage{
			{
				URL:       "/images/old-1.webp",
				SortOrder: 0,
				IsPrimary: true,
			},
			{
				URL:       "/images/old-2.webp",
				SortOrder: 1,
				IsPrimary: false,
			},
		},
	}

	createdProduct, err := repo.Create(ctx, product)
	if err != nil {
		t.Fatalf("failed to create product: %v", err)
	}

	t.Cleanup(func() {
		ctx := context.Background()

		_, err := dbPool.Exec(
			ctx,
			`DELETE FROM products WHERE id = $1`,
			createdProduct.ID,
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
	})

	newDescription := "New description"

	createdProduct.Name = fmt.Sprintf("Updated Pizza %d", suffix)
	createdProduct.Description = &newDescription
	createdProduct.Price = 69900
	createdProduct.Weight = 500
	createdProduct.IsActive = false

	createdProduct.Images = []ProductImage{
		{
			URL:       "/images/new-1.webp",
			SortOrder: 0,
			IsPrimary: true,
		},
		{
			URL:       "/images/new-2.webp",
			SortOrder: 1,
			IsPrimary: false,
		},
	}

	updatedProduct, err := repo.Update(ctx, createdProduct)
	if err != nil {
		t.Fatalf("unexpected update error: %v", err)
	}

	if updatedProduct == nil {
		t.Fatal("expected updated product, got nil")
	}

	savedProduct, err := repo.GetByID(ctx, createdProduct.ID)
	if err != nil {
		t.Fatalf("failed to get updated product: %v", err)
	}

	if savedProduct.Name != createdProduct.Name {
		t.Errorf(
			"expected name %q, got %q",
			createdProduct.Name,
			savedProduct.Name,
		)
	}

	if savedProduct.Price != 69900 {
		t.Errorf(
			"expected price %d, got %d",
			69900,
			savedProduct.Price,
		)
	}

	if savedProduct.Weight != 500 {
		t.Errorf(
			"expected weight %d, got %d",
			500,
			savedProduct.Weight,
		)
	}

	if savedProduct.IsActive {
		t.Error("expected product to be inactive")
	}

	if len(savedProduct.Images) != 2 {
		t.Fatalf(
			"expected %d images, got %d",
			2,
			len(savedProduct.Images),
		)
	}

	if savedProduct.Images[0].URL != "/images/new-1.webp" {
		t.Errorf(
			"expected first image URL %q, got %q",
			"/images/new-1.webp",
			savedProduct.Images[0].URL,
		)
	}

	if savedProduct.Images[1].URL != "/images/new-2.webp" {
		t.Errorf(
			"expected second image URL %q, got %q",
			"/images/new-2.webp",
			savedProduct.Images[1].URL,
		)
	}

}
