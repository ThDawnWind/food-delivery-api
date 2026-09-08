package category

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

func (r *Repository) GetByID(ctx context.Context, id int64) (*Category, error) {
	var category Category

	err := r.db.QueryRow(
		ctx,
		`SELECT id, name, slug, created_at, updated_at FROM categories WHERE id = $1`,
		id,
	).Scan(
		&category.ID,
		&category.Name,
		&category.Slug,
		&category.CreatedAt,
		&category.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	return &category, nil
}

func (r *Repository) List(ctx context.Context) ([]Category, error) {
	rows, err := r.db.Query(
		ctx,
		`SELECT id, name, slug, created_at, updated_at FROM categories ORDER BY id`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var categories []Category

	for rows.Next() {
		var category Category
		err := rows.Scan(
			&category.ID,
			&category.Name,
			&category.Slug,
			&category.CreatedAt,
			&category.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		categories = append(categories, category)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return categories, nil
}

func (r *Repository) Create(ctx context.Context, name, slug string) (*Category, error) {

	category := &Category{
		Name: name,
		Slug: slug,
	}

	err := r.db.QueryRow(
		ctx,
		`INSERT INTO categories(name, slug) 
		VALUES($1, $2)
		RETURNING id, created_at, updated_at`,
		name,
		slug,
	).Scan(
		&category.ID,
		&category.CreatedAt,
		&category.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create category: %w", err)
	}

	return category, nil
}

func (r *Repository) Update(ctx context.Context, category *Category) error {

	err := r.db.QueryRow(
		ctx,
		`UPDATE categories
		SET name = $1, slug = $2, updated_at = NOW()
		WHERE id = $3
		RETURNING updated_at`,
		category.Name,
		category.Slug,
		category.ID,
	).Scan(
		&category.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to update category: %w", err)
	}

	return nil
}
