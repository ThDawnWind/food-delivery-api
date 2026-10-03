package address

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

func newTestPool(t *testing.T) *pgxpool.Pool {
	t.Helper()

	databaseURL := os.Getenv("TEST_DATABASE_URL")

	if databaseURL == "" {
		env, err := godotenv.Read("../../.env.test")
		if err != nil {
			t.Fatalf(
				"failed to read .env.test: %v",
				err,
			)
		}

		databaseURL = env["TEST_DATABASE_URL"]
	}

	if databaseURL == "" {
		t.Fatal("TEST_DATABASE_URL is not set")
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

func cleanupExec(
	t *testing.T,
	db *pgxpool.Pool,
	query string,
	args ...any,
) {
	t.Helper()

	_, err := db.Exec(
		context.Background(),
		query,
		args...,
	)
	if err != nil {
		t.Errorf(
			"failed to clean up test data: %v",
			err,
		)
	}
}

func TestRepository_Create(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	db := newTestPool(t)
	repository := NewRepository(db)

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
		"address_test_user",
		"address_test@example.com",
		"test-password-hash",
		"user",
	).Scan(&userID)
	if err != nil {
		t.Fatalf(
			"failed to create test user: %v",
			err,
		)
	}

	t.Cleanup(func() {
		cleanupExec(
			t,
			db,
			`DELETE FROM users WHERE id = $1`,
			userID,
		)
	})

	label := "Home"
	apartment := "42"

	input := CreateAddress{
		UserID:          userID,
		Label:           &label,
		City:            testCityAmsterdam,
		Street:          testStreetTest,
		HouseNumber:     "10",
		ApartmentNumber: &apartment,
	}

	address, err := repository.Create(
		ctx,
		input,
	)
	if err != nil {
		t.Fatalf(
			"unexpected error: %v",
			err,
		)
	}

	if address == nil {
		t.Fatal("expected address, got nil")
	}

	if address.ID <= 0 {
		t.Errorf(
			"expected positive address ID, got %d",
			address.ID,
		)
	}

	if address.UserID != userID {
		t.Errorf(
			"expected user ID %d, got %d",
			userID,
			address.UserID,
		)
	}

	if address.City != testCityAmsterdam {
		t.Errorf(
			"expected city %q, got %q",
			testCityAmsterdam,
			address.City,
		)
	}

	if address.Street != testStreetTest {
		t.Errorf(
			"expected street %q, got %q",
			testStreetTest,
			address.Street,
		)
	}

	if address.HouseNumber != "10" {
		t.Errorf(
			"expected house number %q, got %q",
			"10",
			address.HouseNumber,
		)
	}

	if address.Label == nil || *address.Label != "Home" {
		t.Errorf(
			"expected label %q, got %v",
			"Home",
			address.Label,
		)
	}

	if address.ApartmentNumber == nil ||
		*address.ApartmentNumber != "42" {
		t.Errorf(
			"expected apartment number %q, got %v",
			"42",
			address.ApartmentNumber,
		)
	}

	if address.CreatedAt.IsZero() {
		t.Error("expected created_at to be set")
	}

	if address.UpdatedAt.IsZero() {
		t.Error("expected updated_at to be set")
	}
}

func TestRepository_Create_NullableFields(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	db := newTestPool(t)
	repository := NewRepository(db)

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
		"address_null_test_user",
		"address_null_test@example.com",
		"test-password-hash",
		"user",
	).Scan(&userID)
	if err != nil {
		t.Fatalf(
			"failed to create test user: %v",
			err,
		)
	}

	t.Cleanup(func() {
		cleanupExec(
			t,
			db,
			`DELETE FROM addresses WHERE user_id = $1`,
			userID,
		)

		cleanupExec(
			t,
			db,
			`DELETE FROM users WHERE id = $1`,
			userID,
		)
	})

	input := CreateAddress{
		UserID:      userID,
		City:        testCityAmsterdam,
		Street:      testStreetTest,
		HouseNumber: "20",
	}

	address, err := repository.Create(
		ctx,
		input,
	)
	if err != nil {
		t.Fatalf(
			"unexpected error: %v",
			err,
		)
	}

	if address.Label != nil {
		t.Errorf(
			"expected nil label, got %v",
			address.Label,
		)
	}

	if address.ApartmentNumber != nil {
		t.Errorf(
			"expected nil apartment number, got %v",
			address.ApartmentNumber,
		)
	}

	if address.Entrance != nil {
		t.Errorf(
			"expected nil entrance, got %v",
			address.Entrance,
		)
	}

	if address.Floor != nil {
		t.Errorf(
			"expected nil floor, got %v",
			address.Floor,
		)
	}

	if address.Comment != nil {
		t.Errorf(
			"expected nil comment, got %v",
			address.Comment,
		)
	}
}

func TestRepository_GetByID(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	db := newTestPool(t)
	repository := NewRepository(db)

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
		"address_get_user",
		"address_get@example.com",
		"test-password-hash",
		"user",
	).Scan(&userID)
	if err != nil {
		t.Fatalf(
			"failed to create test user: %v",
			err,
		)
	}

	t.Cleanup(func() {
		cleanupExec(
			t,
			db,
			`DELETE FROM addresses WHERE user_id = $1`,
			userID,
		)

		cleanupExec(
			t,
			db,
			`DELETE FROM users WHERE id = $1`,
			userID,
		)
	})

	created, err := repository.Create(
		ctx,
		CreateAddress{
			UserID:      userID,
			City:        testCityAmsterdam,
			Street:      testStreetTest,
			HouseNumber: "30",
		},
	)
	if err != nil {
		t.Fatalf(
			"failed to create address: %v",
			err,
		)
	}

	address, err := repository.GetByID(
		ctx,
		created.ID,
		userID,
	)
	if err != nil {
		t.Fatalf(
			"unexpected error: %v",
			err,
		)
	}

	if address.ID != created.ID {
		t.Errorf(
			"expected address ID %d, got %d",
			created.ID,
			address.ID,
		)
	}

	if address.UserID != userID {
		t.Errorf(
			"expected user ID %d, got %d",
			userID,
			address.UserID,
		)
	}
}

func TestRepository_GetByID_OtherUser(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	db := newTestPool(t)
	repository := NewRepository(db)

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
		"address_owner_user",
		"address_owner@example.com",
		"test-password-hash",
		"user",
	).Scan(&userID)
	if err != nil {
		t.Fatalf(
			"failed to create test user: %v",
			err,
		)
	}

	t.Cleanup(func() {
		cleanupExec(
			t,
			db,
			`DELETE FROM addresses WHERE user_id = $1`,
			userID,
		)

		cleanupExec(
			t,
			db,
			`DELETE FROM users WHERE id = $1`,
			userID,
		)
	})

	created, err := repository.Create(
		ctx,
		CreateAddress{
			UserID:      userID,
			City:        testCityAmsterdam,
			Street:      testStreetTest,
			HouseNumber: "40",
		},
	)
	if err != nil {
		t.Fatalf(
			"failed to create address: %v",
			err,
		)
	}

	_, err = repository.GetByID(
		ctx,
		created.ID,
		userID+999,
	)

	if !errors.Is(err, pgx.ErrNoRows) {
		t.Fatalf(
			"expected pgx.ErrNoRows, got %v",
			err,
		)
	}
}

func TestRepository_ListByUser(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	db := newTestPool(t)
	repository := NewRepository(db)

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
		"address_list_user",
		"address_list@example.com",
		"test-password-hash",
		"user",
	).Scan(&userID)
	if err != nil {
		t.Fatalf(
			"failed to create test user: %v",
			err,
		)
	}

	t.Cleanup(func() {
		cleanupExec(
			t,
			db,
			`DELETE FROM addresses WHERE user_id = $1`,
			userID,
		)

		cleanupExec(
			t,
			db,
			`DELETE FROM users WHERE id = $1`,
			userID,
		)
	})

	_, err = repository.Create(
		ctx,
		CreateAddress{
			UserID:      userID,
			City:        testCityAmsterdam,
			Street:      testStreetFirst,
			HouseNumber: "1",
		},
	)
	if err != nil {
		t.Fatalf(
			"failed to create first address: %v",
			err,
		)
	}

	_, err = repository.Create(
		ctx,
		CreateAddress{
			UserID:      userID,
			City:        testCityAmsterdam,
			Street:      testStreetSecond,
			HouseNumber: "2",
		},
	)
	if err != nil {
		t.Fatalf(
			"failed to create second address: %v",
			err,
		)
	}

	addresses, err := repository.ListByUser(
		ctx,
		userID,
	)
	if err != nil {
		t.Fatalf(
			"unexpected error: %v",
			err,
		)
	}

	if len(addresses) != 2 {
		t.Fatalf(
			"expected %d addresses, got %d",
			2,
			len(addresses),
		)
	}

	for _, address := range addresses {
		if address.UserID != userID {
			t.Errorf(
				"expected user ID %d, got %d",
				userID,
				address.UserID,
			)
		}
	}
}

func TestRepository_ListByUser_Empty(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	db := newTestPool(t)
	repository := NewRepository(db)

	addresses, err := repository.ListByUser(
		ctx,
		999999,
	)
	if err != nil {
		t.Fatalf(
			"unexpected error: %v",
			err,
		)
	}

	if len(addresses) != 0 {
		t.Fatalf(
			"expected empty list, got %d addresses",
			len(addresses),
		)
	}
}

func TestRepository_Delete(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	db := newTestPool(t)
	repository := NewRepository(db)

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
		"address_delete_user",
		"address_delete@example.com",
		"test-password-hash",
		"user",
	).Scan(&userID)
	if err != nil {
		t.Fatalf(
			"failed to create test user: %v",
			err,
		)
	}

	t.Cleanup(func() {
		cleanupExec(
			t,
			db,
			`DELETE FROM addresses WHERE user_id = $1`,
			userID,
		)

		cleanupExec(
			t,
			db,
			`DELETE FROM users WHERE id = $1`,
			userID,
		)
	})

	created, err := repository.Create(
		ctx,
		CreateAddress{
			UserID:      userID,
			City:        testCityAmsterdam,
			Street:      "Delete Street",
			HouseNumber: "10",
		},
	)
	if err != nil {
		t.Fatalf(
			"failed to create address: %v",
			err,
		)
	}

	err = repository.Delete(
		ctx,
		created.ID,
		userID,
	)
	if err != nil {
		t.Fatalf(
			"unexpected delete error: %v",
			err,
		)
	}

	_, err = repository.GetByID(
		ctx,
		created.ID,
		userID,
	)

	if !errors.Is(err, pgx.ErrNoRows) {
		t.Fatalf(
			"expected pgx.ErrNoRows, got %v",
			err,
		)
	}
}

func TestRepository_Delete_OtherUser(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	db := newTestPool(t)
	repository := NewRepository(db)

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
		"address_delete_owner",
		"address_delete_owner@example.com",
		"test-password-hash",
		"user",
	).Scan(&userID)
	if err != nil {
		t.Fatalf(
			"failed to create test user: %v",
			err,
		)
	}

	t.Cleanup(func() {
		cleanupExec(
			t,
			db,
			`DELETE FROM addresses WHERE user_id = $1`,
			userID,
		)

		cleanupExec(
			t,
			db,
			`DELETE FROM users WHERE id = $1`,
			userID,
		)
	})

	created, err := repository.Create(
		ctx,
		CreateAddress{
			UserID:      userID,
			City:        "Amsterdam",
			Street:      "Owner Street",
			HouseNumber: "20",
		},
	)
	if err != nil {
		t.Fatalf(
			"failed to create address: %v",
			err,
		)
	}

	err = repository.Delete(
		ctx,
		created.ID,
		userID+999,
	)

	if !errors.Is(err, ErrAddressNotFound) {
		t.Fatalf(
			"expected ErrAddressNotFound, got %v",
			err,
		)
	}
}

func TestRepository_Update(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	db := newTestPool(t)
	repository := NewRepository(db)

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
		"address_update_user",
		"address_update@example.com",
		"test-password-hash",
		"user",
	).Scan(&userID)
	if err != nil {
		t.Fatalf(
			"failed to create test user: %v",
			err,
		)
	}

	t.Cleanup(func() {
		cleanupExec(
			t,
			db,
			`DELETE FROM addresses WHERE user_id = $1`,
			userID,
		)

		cleanupExec(
			t,
			db,
			`DELETE FROM users WHERE id = $1`,
			userID,
		)
	})

	created, err := repository.Create(
		ctx,
		CreateAddress{
			UserID:      userID,
			City:        testCityAmsterdam,
			Street:      "Old Street",
			HouseNumber: "10",
		},
	)
	if err != nil {
		t.Fatalf(
			"failed to create address: %v",
			err,
		)
	}

	newCity := testCityRotterdam
	newStreet := testStreetNew

	updated, err := repository.Update(
		ctx,
		created.ID,
		userID,
		&UpdateAddress{
			City:   &newCity,
			Street: &newStreet,
		},
	)
	if err != nil {
		t.Fatalf(
			"unexpected update error: %v",
			err,
		)
	}

	if updated.City != "Rotterdam" {
		t.Errorf(
			"expected city %q, got %q",
			"Rotterdam",
			updated.City,
		)
	}

	if updated.Street != "New Street" {
		t.Errorf(
			"expected street %q, got %q",
			"New Street",
			updated.Street,
		)
	}

	if updated.HouseNumber != "10" {
		t.Errorf(
			"expected house number %q, got %q",
			"10",
			updated.HouseNumber,
		)
	}
}

func TestRepository_Update_OtherUser(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	db := newTestPool(t)
	repository := NewRepository(db)

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
		"address_update_owner",
		"address_update_owner@example.com",
		"test-password-hash",
		"user",
	).Scan(&userID)
	if err != nil {
		t.Fatalf(
			"failed to create test user: %v",
			err,
		)
	}

	t.Cleanup(func() {
		cleanupExec(
			t,
			db,
			`DELETE FROM addresses WHERE user_id = $1`,
			userID,
		)

		cleanupExec(
			t,
			db,
			`DELETE FROM users WHERE id = $1`,
			userID,
		)
	})

	created, err := repository.Create(
		ctx,
		CreateAddress{
			UserID:      userID,
			City:        "Amsterdam",
			Street:      "Owner Street",
			HouseNumber: "10",
		},
	)
	if err != nil {
		t.Fatalf(
			"failed to create address: %v",
			err,
		)
	}

	newCity := "Rotterdam"

	_, err = repository.Update(
		ctx,
		created.ID,
		userID+999,
		&UpdateAddress{
			City: &newCity,
		},
	)

	if !errors.Is(err, pgx.ErrNoRows) {
		t.Fatalf(
			"expected pgx.ErrNoRows, got %v",
			err,
		)
	}
}
