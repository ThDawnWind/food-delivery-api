package product

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

func (r *Repository) GetByID(ctx context.Context, id int64) (*Product, error) {
	var product Product

	err := r.db.QueryRow(
		ctx,
		`
		SELECT
			id,
			name,
			description,
			price,
			weight,
			category_id,
			is_active,
			created_at,
			updated_at
		FROM products
		WHERE id = $1
		`,
		id,
	).Scan(
		&product.ID,
		&product.Name,
		&product.Description,
		&product.Price,
		&product.Weight,
		&product.CategoryID,
		&product.IsActive,
		&product.CreatedAt,
		&product.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get product: %w", err)
	}

	images, err := r.listImagesByProductID(ctx, product.ID)
	if err != nil {
		return nil, err
	}

	product.Images = images

	return &product, nil
}

func (r *Repository) listImagesByProductID(
	ctx context.Context,
	productID int64,
) ([]ProductImage, error) {
	rows, err := r.db.Query(
		ctx,
		`
		SELECT
			id,
			product_id,
			url,
			sort_order,
			is_primary,
			created_at
		FROM product_images
		WHERE product_id = $1
		ORDER BY sort_order, id
		`,
		productID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get product images: %w", err)
	}
	defer rows.Close()

	images := make([]ProductImage, 0)

	for rows.Next() {
		var image ProductImage

		err := rows.Scan(
			&image.ID,
			&image.ProductID,
			&image.URL,
			&image.SortOrder,
			&image.IsPrimary,
			&image.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan product image: %w", err)
		}

		images = append(images, image)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate product images: %w", err)
	}

	return images, nil
}
