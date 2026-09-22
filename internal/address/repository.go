package address

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

func (r *Repository) Create(ctx context.Context, input *CreateAddress) (*Address, error) {
	const query = `
		INSERT INTO addresses (
			user_id,
			label,
			city,
			street,
			house_number,
			apartment_number,
			entrance,
			floor,
			comment
		)
		VALUES (
			$1, $2, $3, $4, $5,
			$6, $7, $8, $9
		)
		RETURNING
			id,
			user_id,
			label,
			city,
			street,
			house_number,
			apartment_number,
			entrance,
			floor,
			comment,
			created_at,
			updated_at
	`

	var address Address

	err := r.db.QueryRow(
		ctx,
		query,
		input.UserID,
		input.Label,
		input.City,
		input.Street,
		input.HouseNumber,
		input.ApartmentNumber,
		input.Entrance,
		input.Floor,
		input.Comment,
	).Scan(
		&address.ID,
		&address.UserID,
		&address.Label,
		&address.City,
		&address.Street,
		&address.HouseNumber,
		&address.ApartmentNumber,
		&address.Entrance,
		&address.Floor,
		&address.Comment,
		&address.CreatedAt,
		&address.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"failed to create address: %w",
			err,
		)
	}

	return &address, nil
}

func (r *Repository) GetByID(ctx context.Context, addressID int64, userID int64) (*Address, error) {
	const query = `
		SELECT
			id,
			user_id,
			label,
			city,
			street,
			house_number,
			apartment_number,
			entrance,
			floor,
			comment,
			created_at,
			updated_at
		FROM addresses
		WHERE id = $1
		  AND user_id = $2
	`

	var address Address

	err := r.db.QueryRow(
		ctx,
		query,
		addressID,
		userID,
	).Scan(
		&address.ID,
		&address.UserID,
		&address.Label,
		&address.City,
		&address.Street,
		&address.HouseNumber,
		&address.ApartmentNumber,
		&address.Entrance,
		&address.Floor,
		&address.Comment,
		&address.CreatedAt,
		&address.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"failed to get address: %w",
			err,
		)
	}

	return &address, nil
}

func (r *Repository) ListByUser(ctx context.Context, userID int64) ([]Address, error) {
	const query = `
		SELECT
			id,
			user_id,
			label,
			city,
			street,
			house_number,
			apartment_number,
			entrance,
			floor,
			comment,
			created_at,
			updated_at
		FROM addresses
		WHERE user_id = $1
		ORDER BY id DESC
	`

	rows, err := r.db.Query(
		ctx,
		query,
		userID,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"failed to list addresses: %w",
			err,
		)
	}
	defer rows.Close()

	addresses := make([]Address, 0)

	for rows.Next() {
		var address Address

		err := rows.Scan(
			&address.ID,
			&address.UserID,
			&address.Label,
			&address.City,
			&address.Street,
			&address.HouseNumber,
			&address.ApartmentNumber,
			&address.Entrance,
			&address.Floor,
			&address.Comment,
			&address.CreatedAt,
			&address.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf(
				"failed to scan address: %w",
				err,
			)
		}

		addresses = append(
			addresses,
			address,
		)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf(
			"failed to iterate addresses: %w",
			err,
		)
	}

	return addresses, nil
}

func (r *Repository) Delete(ctx context.Context, addressID int64, userID int64) error {
	const query = `
		DELETE FROM addresses
		WHERE id = $1
		  AND user_id = $2
	`

	result, err := r.db.Exec(
		ctx,
		query,
		addressID,
		userID,
	)
	if err != nil {
		return fmt.Errorf(
			"failed to delete address: %w",
			err,
		)
	}

	if result.RowsAffected() == 0 {
		return ErrAddressNotFound
	}

	return nil
}

func (r *Repository) Update(ctx context.Context, addressID int64, userID int64, input *UpdateAddress) (*Address, error) {
	const query = `
		UPDATE addresses
		SET
			label = COALESCE($3, label),
			city = COALESCE($4, city),
			street = COALESCE($5, street),
			house_number = COALESCE($6, house_number),
			apartment_number = COALESCE($7, apartment_number),
			entrance = COALESCE($8, entrance),
			floor = COALESCE($9, floor),
			comment = COALESCE($10, comment),
			updated_at = NOW()
		WHERE id = $1
		  AND user_id = $2
		RETURNING
			id,
			user_id,
			label,
			city,
			street,
			house_number,
			apartment_number,
			entrance,
			floor,
			comment,
			created_at,
			updated_at
	`

	var address Address

	err := r.db.QueryRow(
		ctx,
		query,
		addressID,
		userID,
		input.Label,
		input.City,
		input.Street,
		input.HouseNumber,
		input.ApartmentNumber,
		input.Entrance,
		input.Floor,
		input.Comment,
	).Scan(
		&address.ID,
		&address.UserID,
		&address.Label,
		&address.City,
		&address.Street,
		&address.HouseNumber,
		&address.ApartmentNumber,
		&address.Entrance,
		&address.Floor,
		&address.Comment,
		&address.CreatedAt,
		&address.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"failed to update address: %w",
			err,
		)
	}

	return &address, nil
}
