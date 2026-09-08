package category

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

func TestRepository_List(t *testing.T) {
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

	var id1 int64
	var id2 int64

	suffix := time.Now().UnixNano()

	name1 := fmt.Sprintf("Test Category 1 %d", suffix)
	slug1 := fmt.Sprintf("test-category-1-%d", suffix)

	name2 := fmt.Sprintf("Test Category 2 %d", suffix)
	slug2 := fmt.Sprintf("test-category-2-%d", suffix)

	err = dbPool.QueryRow(ctx, `
		INSERT INTO categories (name, slug)
		VALUES ($1, $2)
		RETURNING id
	`,
		name1,
		slug1,
	).Scan(&id1)
	if err != nil {
		t.Fatalf("Failed to inn: %v", err)
	}

	err = dbPool.QueryRow(ctx, `
		INSERT INTO categories (name, slug)
		VALUES ($1, $2)
		RETURNING id
	`, name2,
		slug2,
	).Scan(&id2)
	if err != nil {
		t.Fatalf("Failed to inn: %v", err)
	}

	t.Cleanup(func() {
		_, err := dbPool.Exec(
			context.Background(),
			`DELETE FROM categories WHERE id IN ($1, $2)`,
			id1,
			id2,
		)

		if err != nil {
			t.Errorf("Failed to clean up test categories: %v", err)
		}
	})

	categories, err := repo.List(ctx)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	found1 := false
	found2 := false

	for _, category := range categories {
		if category.ID == id1 {
			found1 = true
		}

		if category.ID == id2 {
			found2 = true
		}
	}

	if !found1 {
		t.Errorf("category with id %d not found", id1)
	}

	if !found2 {
		t.Errorf("category with id %d not found", id2)
	}

}
