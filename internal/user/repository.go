package user

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgconn"
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

func (r *Repository) Create(ctx context.Context, input *CreateUser) (*User, error) {
	user := &User{}

	err := r.db.QueryRow(
		ctx,
		`
		INSERT INTO users (
			username,
			email,
			password_hash,
			role
		)
		VALUES ($1, $2, $3, $4)
		RETURNING
			id,
			username,
			email,
			password_hash,
			role,
			created_at,
			updated_at
		`,
		input.Username,
		input.Email,
		input.PasswordHash,
		input.Role,
	).Scan(
		&user.ID,
		&user.Username,
		&user.Email,
		&user.PasswordHash,
		&user.Role,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		var pgErr *pgconn.PgError

		if errors.As(err, &pgErr) &&
			pgErr.Code == "23505" {
			return nil, ErrUserConflict
		}

		return nil, fmt.Errorf(
			"failed to create user: %w",
			err,
		)
	}

	return user, nil
}

func (r *Repository) GetByID(ctx context.Context, id int64) (*User, error) {
	user := &User{}

	err := r.db.QueryRow(
		ctx,
		`
				SELECT
					id,
					username,
					email,
					password_hash,
					role,
					created_at,
					updated_at
				FROM users
				WHERE id = $1
				`,
		id,
	).Scan(
		&user.ID,
		&user.Username,
		&user.Email,
		&user.PasswordHash,
		&user.Role,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf(
			"failed to get user by id: %w",
			err,
		)
	}

	return user, nil
}

func (r *Repository) GetByEmail(ctx context.Context, email string) (*User, error) {
	user := &User{}

	err := r.db.QueryRow(
		ctx,
		`
			SELECT
				id,
				username,
				email,
				password_hash,
				role,
				created_at,
				updated_at
			FROM users
			WHERE email = $1
			`,
		email,
	).Scan(
		&user.ID,
		&user.Username,
		&user.Email,
		&user.PasswordHash,
		&user.Role,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf(
			"failed to get user by email: %w",
			err,
		)
	}

	return user, nil
}
