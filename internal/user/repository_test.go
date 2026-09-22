package user

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

	input := &CreateUser{
		Username:     fmt.Sprintf("user-%d", suffix),
		Email:        fmt.Sprintf("user-%d@example.com", suffix),
		PasswordHash: "test-password-hash",
		Role:         RoleUser,
	}

	createdUser, err := repo.Create(
		ctx,
		input,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if createdUser == nil {
		t.Fatal("expected user, got nil")
	}

	if createdUser.ID == 0 {
		t.Error("expected user ID to be set")
	}

	if createdUser.Username != input.Username {
		t.Errorf(
			"expected username %q, got %q",
			input.Username,
			createdUser.Username,
		)
	}

	if createdUser.Email != input.Email {
		t.Errorf(
			"expected email %q, got %q",
			input.Email,
			createdUser.Email,
		)
	}

	if createdUser.PasswordHash != input.PasswordHash {
		t.Errorf(
			"expected password hash %q, got %q",
			input.PasswordHash,
			createdUser.PasswordHash,
		)
	}

	if createdUser.Role != RoleUser {
		t.Errorf(
			"expected role %q, got %q",
			RoleUser,
			createdUser.Role,
		)
	}

	if createdUser.CreatedAt.IsZero() {
		t.Error("expected created_at to be set")
	}

	if createdUser.UpdatedAt.IsZero() {
		t.Error("expected updated_at to be set")
	}

	t.Cleanup(func() {
		_, err := dbPool.Exec(
			context.Background(),
			`DELETE FROM users WHERE id = $1`,
			createdUser.ID,
		)
		if err != nil {
			t.Errorf(
				"failed to clean up user: %v",
				err,
			)
		}
	})
}

func TestRepository_Create_DuplicateUsername(t *testing.T) {
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

	firstUser, err := repo.Create(
		ctx,
		&CreateUser{
			Username:     fmt.Sprintf("duplicate-%d", suffix),
			Email:        fmt.Sprintf("first-%d@example.com", suffix),
			PasswordHash: "hash",
			Role:         RoleUser,
		},
	)
	if err != nil {
		t.Fatalf("failed to create first user: %v", err)
	}

	t.Cleanup(func() {
		_, _ = dbPool.Exec(
			context.Background(),
			`DELETE FROM users WHERE id = $1`,
			firstUser.ID,
		)
	})

	secondUser, err := repo.Create(
		ctx,
		&CreateUser{
			Username:     firstUser.Username,
			Email:        fmt.Sprintf("second-%d@example.com", suffix),
			PasswordHash: "hash",
			Role:         RoleUser,
		},
	)

	if secondUser != nil {
		t.Fatalf(
			"expected nil user, got %+v",
			secondUser,
		)
	}

	if !errors.Is(err, ErrUserConflict) {
		t.Fatalf(
			"expected ErrUserConflict, got %v",
			err,
		)
	}
}

func TestRepository_Create_DuplicateEmail(t *testing.T) {
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

	firstUser, err := repo.Create(
		ctx,
		&CreateUser{
			Username:     fmt.Sprintf("first-%d", suffix),
			Email:        fmt.Sprintf("duplicate-%d@example.com", suffix),
			PasswordHash: "hash",
			Role:         RoleUser,
		},
	)
	if err != nil {
		t.Fatalf("failed to create first user: %v", err)
	}

	t.Cleanup(func() {
		_, _ = dbPool.Exec(
			context.Background(),
			`DELETE FROM users WHERE id = $1`,
			firstUser.ID,
		)
	})

	secondUser, err := repo.Create(
		ctx,
		&CreateUser{
			Username:     fmt.Sprintf("second-%d", suffix),
			Email:        firstUser.Email,
			PasswordHash: "hash",
			Role:         RoleUser,
		},
	)

	if secondUser != nil {
		t.Fatalf(
			"expected nil user, got %+v",
			secondUser,
		)
	}

	if !errors.Is(err, ErrUserConflict) {
		t.Fatalf(
			"expected ErrUserConflict, got %v",
			err,
		)
	}
}

func TestRepository_GetByID(t *testing.T) {
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

	createdUser, err := repo.Create(
		ctx,
		&CreateUser{
			Username:     fmt.Sprintf("get-by-id-%d", suffix),
			Email:        fmt.Sprintf("get-by-id-%d@example.com", suffix),
			PasswordHash: "test-hash",
			Role:         RoleUser,
		},
	)
	if err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	t.Cleanup(func() {
		_, _ = dbPool.Exec(
			context.Background(),
			`DELETE FROM users WHERE id = $1`,
			createdUser.ID,
		)
	})

	user, err := repo.GetByID(
		ctx,
		createdUser.ID,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if user == nil {
		t.Fatal("expected user, got nil")
	}

	if user.ID != createdUser.ID {
		t.Errorf(
			"expected user ID %d, got %d",
			createdUser.ID,
			user.ID,
		)
	}

	if user.Username != createdUser.Username {
		t.Errorf(
			"expected username %q, got %q",
			createdUser.Username,
			user.Username,
		)
	}

	if user.Email != createdUser.Email {
		t.Errorf(
			"expected email %q, got %q",
			createdUser.Email,
			user.Email,
		)
	}

	if user.PasswordHash != "test-hash" {
		t.Errorf(
			"expected password hash %q, got %q",
			"test-hash",
			user.PasswordHash,
		)
	}

	if user.Role != RoleUser {
		t.Errorf(
			"expected role %q, got %q",
			RoleUser,
			user.Role,
		)
	}
}

func TestRepository_GetByEmail(t *testing.T) {
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

	createdUser, err := repo.Create(
		ctx,
		&CreateUser{
			Username:     fmt.Sprintf("get-by-email-%d", suffix),
			Email:        fmt.Sprintf("get-by-email-%d@example.com", suffix),
			PasswordHash: "test-hash",
			Role:         RoleUser,
		},
	)
	if err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	t.Cleanup(func() {
		_, _ = dbPool.Exec(
			context.Background(),
			`DELETE FROM users WHERE id = $1`,
			createdUser.ID,
		)
	})

	user, err := repo.GetByEmail(
		ctx,
		createdUser.Email,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if user == nil {
		t.Fatal("expected user, got nil")
	}

	if user.ID != createdUser.ID {
		t.Errorf(
			"expected user ID %d, got %d",
			createdUser.ID,
			user.ID,
		)
	}

	if user.Email != createdUser.Email {
		t.Errorf(
			"expected email %q, got %q",
			createdUser.Email,
			user.Email,
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

	user, err := repo.GetByID(
		ctx,
		999999999,
	)

	if user != nil {
		t.Fatalf(
			"expected nil user, got %+v",
			user,
		)
	}

	if !errors.Is(err, pgx.ErrNoRows) {
		t.Fatalf(
			"expected pgx.ErrNoRows, got %v",
			err,
		)
	}
}

func TestRepository_GetByEmail_NotFound(t *testing.T) {
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

	user, err := repo.GetByEmail(
		ctx,
		"missing-user@example.com",
	)

	if user != nil {
		t.Fatalf(
			"expected nil user, got %+v",
			user,
		)
	}

	if !errors.Is(err, pgx.ErrNoRows) {
		t.Fatalf(
			"expected pgx.ErrNoRows, got %v",
			err,
		)
	}
}
