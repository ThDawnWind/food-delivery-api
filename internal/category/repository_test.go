package category

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/ThDawnWind/food-delivery-api/internal/database"
	"github.com/jackc/pgx/v5"
)

func TestRepository_GetByID(t *testing.T) {
	ctx := context.Background()

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		t.Skip("DATABASE_URL environment variable is not set")
	}

	dbPool, err := database.New(ctx, dbURL)
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}
	t.Cleanup(dbPool.Close)

	repo := NewRepository(dbPool)

	var categoryID int64

	err = dbPool.QueryRow(
		ctx,
		`INSERT INTO categories (name, slug)
		VALUES ($1, $2)
		RETURNING id`,
		"Test Category",
		"test-category",
	).Scan(&categoryID)
	if err != nil {
		t.Fatalf("Failed to insert test category: %v", err)
	}

	t.Cleanup(func() {
		_, err := dbPool.Exec(context.Background(), `DELETE FROM categories WHERE id = $1`, categoryID)
		if err != nil {
			t.Errorf("Failed to clean up test category: %v", err)
		}
	})

	category, err := repo.GetByID(ctx, categoryID)
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}

	if category.Name != "Test Category" || category.Slug != "test-category" {
		t.Errorf("GetByID returned unexpected category: %+v", category)
	}

}

func TestRepository_GetByID_NotFound(t *testing.T) {
	ctx := context.Background()

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		t.Skip("DATABASE_URL environment variable is not set")
	}

	dbPool, err := database.New(ctx, dbURL)
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}
	t.Cleanup(dbPool.Close)

	repo := NewRepository(dbPool)
	checkID := int64(999999)

	_, err = repo.GetByID(ctx, checkID)
	if err == nil {
		t.Fatalf("Expected error for non-existent category ID, got nil")
	}

	if !errors.Is(err, pgx.ErrNoRows) {
		t.Fatalf("Expected pgx.ErrNoRows for non-existent category ID, got: %v", err)
	}
}
