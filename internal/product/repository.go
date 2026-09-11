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

func (r *Repository) List(
	ctx context.Context,
) ([]Product, error) {
	rows, err := r.db.Query(
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
		ORDER BY id
		`,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list products: %w", err)
	}
	defer rows.Close()

	products := make([]Product, 0)

	for rows.Next() {
		var product Product

		err := rows.Scan(
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
			return nil, fmt.Errorf("failed to scan product: %w", err)
		}

		products = append(products, product)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate products: %w", err)
	}

	if err := r.loadImages(ctx, products); err != nil {
		return nil, err
	}

	return products, nil
}

func (r *Repository) loadImages(
	ctx context.Context,
	products []Product,
) error {
	if len(products) == 0 {
		return nil
	}

	productIDs := make([]int64, 0, len(products))
	productIndex := make(map[int64]int, len(products))

	for i := range products {
		productIDs = append(productIDs, products[i].ID)
		productIndex[products[i].ID] = i

		products[i].Images = make([]ProductImage, 0)
	}

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
		WHERE product_id = ANY($1::bigint[])
		ORDER BY product_id, sort_order, id
		`,
		productIDs,
	)
	if err != nil {
		return fmt.Errorf("failed to list product images: %w", err)
	}
	defer rows.Close()

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
			return fmt.Errorf("failed to scan product image: %w", err)
		}

		index, ok := productIndex[image.ProductID]
		if !ok {
			continue
		}

		products[index].Images = append(
			products[index].Images,
			image,
		)
	}

	if err := rows.Err(); err != nil {
		return fmt.Errorf("failed to iterate product images: %w", err)
	}

	return nil
}

func (r *Repository) Create(ctx context.Context, product *Product) (*Product, error) {

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}

	defer tx.Rollback(ctx)

	err = tx.QueryRow(
		ctx,
		`INSERT INTO products (
		name,
		description,
		price,
		weight,
		category_id,
		is_active
		)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at, updated_at
		`,
		product.Name,
		product.Description,
		product.Price,
		product.Weight,
		product.CategoryID,
		product.IsActive,
	).Scan(
		&product.ID,
		&product.CreatedAt,
		&product.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create product: %w", err)
	}

	for i := range product.Images {
		product.Images[i].ProductID = product.ID

		err = tx.QueryRow(
			ctx,
			`
			INSERT INTO product_images (
				product_id,
				url,
				sort_order,
				is_primary
			)
			VALUES ($1, $2, $3, $4)
			RETURNING id, created_at
			`,
			product.ID,
			product.Images[i].URL,
			product.Images[i].SortOrder,
			product.Images[i].IsPrimary,
		).Scan(
			&product.Images[i].ID,
			&product.Images[i].CreatedAt,
		)

		if err != nil {
			return nil, fmt.Errorf(
				"failed to create product image: %w",
				err,
			)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return product, nil
}
