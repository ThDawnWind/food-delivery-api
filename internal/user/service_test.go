package user

import (
	"context"
	"errors"
	"strings"
	"testing"

	"golang.org/x/crypto/bcrypt"

	"github.com/jackc/pgx/v5"
)

type fakeRepository struct {
	createCalled bool
	createInput  CreateUser
	createUser   *User
	createErr    error

	getByIDUser *User
	getByIDErr  error

	getByEmailUser *User
	getByEmailErr  error

	getByEmailInput string
}

func (f *fakeRepository) Create(_ context.Context, input CreateUser) (*User, error) {
	f.createCalled = true
	f.createInput = input

	if f.createErr != nil {
		return nil, f.createErr
	}

	if f.createUser != nil {
		return f.createUser, nil
	}

	return &User{
		ID:           1,
		Username:     input.Username,
		Email:        input.Email,
		PasswordHash: input.PasswordHash,
		Role:         input.Role,
	}, nil
}

func (f *fakeRepository) GetByID(_ context.Context, _ int64) (*User, error) {
	if f.getByIDErr != nil {
		return nil, f.getByIDErr
	}

	return f.getByIDUser, nil
}

func (f *fakeRepository) GetByEmail(_ context.Context, email string) (*User, error) {
	f.getByEmailInput = email

	if f.getByEmailErr != nil {
		return nil, f.getByEmailErr
	}

	return f.getByEmailUser, nil
}

func TestService_Register(t *testing.T) {
	t.Parallel()

	repository := &fakeRepository{}

	service := NewService(repository)

	input := &RegisterUser{
		Username: "  alex  ",
		Email:    "  alex@example.com  ",
		Password: testPassword,
	}

	user, err := service.Register(
		context.Background(),
		input,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if user == nil {
		t.Fatal("expected user, got nil")
	}

	if !repository.createCalled {
		t.Fatal("expected repository Create to be called")
	}

	if repository.createInput.Username != testUsername {
		t.Errorf(
			"expected username %q, got %q",
			testUsername,
			repository.createInput.Username,
		)
	}

	if repository.createInput.Email != testEmail {
		t.Errorf(
			"expected email %q, got %q",
			testEmail,
			repository.createInput.Email,
		)
	}

	if repository.createInput.Role != RoleUser {
		t.Errorf(
			"expected role %q, got %q",
			RoleUser,
			repository.createInput.Role,
		)
	}

	if repository.createInput.PasswordHash == input.Password {
		t.Fatal(
			"password must not be stored as plain text",
		)
	}

	err = bcrypt.CompareHashAndPassword(
		[]byte(repository.createInput.PasswordHash),
		[]byte(input.Password),
	)
	if err != nil {
		t.Fatalf(
			"expected valid bcrypt hash: %v",
			err,
		)
	}
}

func TestService_Register_Validation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input *RegisterUser
	}{
		{
			name:  "nil input",
			input: nil,
		},
		{
			name: "empty username",
			input: &RegisterUser{
				Username: testBlankValue,
				Email:    testEmail,
				Password: testPassword,
			},
		},
		{
			name: "empty email",
			input: &RegisterUser{
				Username: testUsername,
				Email:    testBlankValue,
				Password: testPassword,
			},
		},
		{
			name: "empty password",
			input: &RegisterUser{
				Username: testUsername,
				Email:    testEmail,
				Password: "",
			},
		},
		{
			name: "short password",
			input: &RegisterUser{
				Username: testUsername,
				Email:    testEmail,
				Password: "1234567",
			},
		},
		{
			name: "password too long",
			input: &RegisterUser{
				Username: testUsername,
				Email:    testEmail,
				Password: strings.Repeat("a", 73),
			},
		},
		{
			name: "username too long",
			input: &RegisterUser{
				Username: strings.Repeat("a", 51),
				Email:    testEmail,
				Password: testPassword,
			},
		},
		{
			name: "email too long",
			input: &RegisterUser{
				Username: testUsername,
				Email: strings.Repeat("a", 89) +
					"@example.com",
				Password: testPassword,
			},
		},
		{
			name: "invalid email",
			input: &RegisterUser{
				Username: testUsername,
				Email:    "not-an-email",
				Password: testPassword,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			repository := &fakeRepository{}

			service := NewService(repository)

			user, err := service.Register(
				context.Background(),
				tt.input,
			)

			if user != nil {
				t.Fatalf(
					"expected nil user, got %+v",
					user,
				)
			}

			if !errors.Is(err, ErrUserValidation) {
				t.Fatalf(
					"expected ErrUserValidation, got %v",
					err,
				)
			}

			if repository.createCalled {
				t.Fatal(
					"repository must not be called on validation error",
				)
			}
		})
	}
}

func TestService_Register_Conflict(t *testing.T) {
	t.Parallel()

	repository := &fakeRepository{
		createErr: ErrUserConflict,
	}

	service := NewService(repository)

	user, err := service.Register(
		context.Background(),
		&RegisterUser{
			Username: testUsername,
			Email:    testEmail,
			Password: testPassword,
		},
	)

	if user != nil {
		t.Fatalf(
			"expected nil user, got %+v",
			user,
		)
	}

	if !errors.Is(err, ErrUserConflict) {
		t.Fatalf(
			"expected ErrUserConflict, got %v",
			err,
		)
	}
}

func TestService_Register_RepositoryError(t *testing.T) {
	t.Parallel()

	repositoryErr := errors.New("database unavailable")

	repository := &fakeRepository{
		createErr: repositoryErr,
	}

	service := NewService(repository)

	user, err := service.Register(
		context.Background(),
		&RegisterUser{
			Username: testUsername,
			Email:    testEmail,
			Password: testPassword,
		},
	)

	if user != nil {
		t.Fatalf(
			"expected nil user, got %+v",
			user,
		)
	}

	if !errors.Is(err, repositoryErr) {
		t.Fatalf(
			"expected repository error, got %v",
			err,
		)
	}
}

func TestService_GetByID(t *testing.T) {
	t.Parallel()

	repository := &fakeRepository{
		getByIDUser: &User{
			ID:       10,
			Username: testUsername,
			Email:    testEmail,
			Role:     RoleUser,
		},
	}

	service := NewService(repository)

	user, err := service.GetByID(
		context.Background(),
		10,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if user == nil {
		t.Fatal("expected user, got nil")
	}

	if user.ID != 10 {
		t.Errorf(
			"expected user ID %d, got %d",
			10,
			user.ID,
		)
	}
}

func TestService_GetByID_InvalidID(t *testing.T) {
	t.Parallel()

	service := NewService(
		&fakeRepository{},
	)

	user, err := service.GetByID(
		context.Background(),
		0,
	)

	if user != nil {
		t.Fatalf(
			"expected nil user, got %+v",
			user,
		)
	}

	if !errors.Is(err, ErrUserValidation) {
		t.Fatalf(
			"expected ErrUserValidation, got %v",
			err,
		)
	}
}

func TestService_GetByID_NotFound(t *testing.T) {
	t.Parallel()

	repository := &fakeRepository{
		getByIDErr: pgx.ErrNoRows,
	}

	service := NewService(repository)

	user, err := service.GetByID(
		context.Background(),
		999,
	)

	if user != nil {
		t.Fatalf(
			"expected nil user, got %+v",
			user,
		)
	}

	if !errors.Is(err, ErrUserNotFound) {
		t.Fatalf(
			"expected ErrUserNotFound, got %v",
			err,
		)
	}
}

func TestService_GetByID_RepositoryError(t *testing.T) {
	t.Parallel()

	repositoryErr := errors.New("database unavailable")

	repository := &fakeRepository{
		getByIDErr: repositoryErr,
	}

	service := NewService(repository)

	user, err := service.GetByID(
		context.Background(),
		1,
	)

	if user != nil {
		t.Fatalf(
			"expected nil user, got %+v",
			user,
		)
	}

	if !errors.Is(err, repositoryErr) {
		t.Fatalf(
			"expected repository error, got %v",
			err,
		)
	}
}

func TestService_Login(t *testing.T) {
	t.Parallel()

	password := testPassword

	hash, err := bcrypt.GenerateFromPassword(
		[]byte(password),
		bcrypt.DefaultCost,
	)
	if err != nil {
		t.Fatalf("failed to hash password: %v", err)
	}

	repository := &fakeRepository{
		getByEmailUser: &User{
			ID:           1,
			Username:     testUsername,
			Email:        testEmail,
			PasswordHash: string(hash),
			Role:         RoleUser,
		},
	}

	service := NewService(repository)

	user, err := service.Login(
		context.Background(),
		&LoginUser{
			Email:    "  alex@example.com  ",
			Password: password,
		},
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if user == nil {
		t.Fatal("expected user, got nil")
	}

	if user.ID != 1 {
		t.Errorf(
			"expected user ID %d, got %d",
			1,
			user.ID,
		)
	}
}

func TestService_Login_InvalidPassword(t *testing.T) {
	t.Parallel()

	hash, err := bcrypt.GenerateFromPassword(
		[]byte("correct-password"),
		bcrypt.DefaultCost,
	)
	if err != nil {
		t.Fatalf("failed to hash password: %v", err)
	}

	repository := &fakeRepository{
		getByEmailUser: &User{
			ID:           1,
			Email:        testEmail,
			PasswordHash: string(hash),
		},
	}

	service := NewService(repository)

	user, err := service.Login(
		context.Background(),
		&LoginUser{
			Email:    testEmail,
			Password: "wrong-password",
		},
	)

	if user != nil {
		t.Fatalf(
			"expected nil user, got %+v",
			user,
		)
	}

	if !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf(
			"expected ErrInvalidCredentials, got %v",
			err,
		)
	}
}

func TestService_Login_UserNotFound(t *testing.T) {
	t.Parallel()

	repository := &fakeRepository{
		getByEmailErr: pgx.ErrNoRows,
	}

	service := NewService(repository)

	user, err := service.Login(
		context.Background(),
		&LoginUser{
			Email:    "missing@example.com",
			Password: testPassword,
		},
	)

	if user != nil {
		t.Fatalf(
			"expected nil user, got %+v",
			user,
		)
	}

	if !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf(
			"expected ErrInvalidCredentials, got %v",
			err,
		)
	}
}

func TestService_Login_Validation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input *LoginUser
	}{
		{
			name:  "nil input",
			input: nil,
		},
		{
			name: "empty email",
			input: &LoginUser{
				Email:    "   ",
				Password: testPassword,
			},
		},
		{
			name: "empty password",
			input: &LoginUser{
				Email:    testEmail,
				Password: "",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			service := NewService(
				&fakeRepository{},
			)

			user, err := service.Login(
				context.Background(),
				tt.input,
			)

			if user != nil {
				t.Fatalf(
					"expected nil user, got %+v",
					user,
				)
			}

			if !errors.Is(err, ErrUserValidation) {
				t.Fatalf(
					"expected ErrUserValidation, got %v",
					err,
				)
			}
		})
	}
}

func TestService_Login_RepositoryError(t *testing.T) {
	t.Parallel()

	repositoryErr := errors.New("database unavailable")

	repository := &fakeRepository{
		getByEmailErr: repositoryErr,
	}

	service := NewService(repository)

	user, err := service.Login(
		context.Background(),
		&LoginUser{
			Email:    testEmail,
			Password: testPassword,
		},
	)

	if user != nil {
		t.Fatalf(
			"expected nil user, got %+v",
			user,
		)
	}

	if !errors.Is(err, repositoryErr) {
		t.Fatalf(
			"expected repository error, got %v",
			err,
		)
	}
}

func TestService_Register_MaxPasswordLength(t *testing.T) {
	t.Parallel()

	repository := &fakeRepository{}
	service := NewService(repository)

	user, err := service.Register(
		context.Background(),
		&RegisterUser{
			Username: testUsername,
			Email:    testEmail,
			Password: strings.Repeat("a", 72),
		},
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if user == nil {
		t.Fatal("expected user, got nil")
	}
}

func TestService_Login_InvalidCredentialsFormat(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input *LoginUser
	}{
		{
			name: "invalid email",
			input: &LoginUser{
				Email:    "not-an-email",
				Password: testPassword,
			},
		},
		{
			name: "email too long",
			input: &LoginUser{
				Email: strings.Repeat("a", 89) +
					"@example.com",
				Password: testPassword,
			},
		},
		{
			name: "password too long",
			input: &LoginUser{
				Email:    testEmail,
				Password: strings.Repeat("a", 73),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			repository := &fakeRepository{}

			service := NewService(repository)

			userData, err := service.Login(
				context.Background(),
				tt.input,
			)

			if userData != nil {
				t.Fatalf(
					"expected nil user, got %+v",
					userData,
				)
			}

			if !errors.Is(
				err,
				ErrInvalidCredentials,
			) {
				t.Fatalf(
					"expected ErrInvalidCredentials, got %v",
					err,
				)
			}

			if repository.getByEmailInput != "" {
				t.Fatal(
					"repository must not be called on invalid credentials",
				)
			}
		})
	}
}

func TestDummyPasswordHashUsesDefaultCost(t *testing.T) {
	t.Parallel()

	cost, err := bcrypt.Cost(
		dummyPasswordHash,
	)
	if err != nil {
		t.Fatalf(
			"invalid dummy password hash: %v",
			err,
		)
	}

	if cost != bcrypt.DefaultCost {
		t.Fatalf(
			"expected dummy hash cost %d, got %d",
			bcrypt.DefaultCost,
			cost,
		)
	}
}

// func BenchmarkServiceLoginExistingUserWrongPassword(b *testing.B) {
// 	password := "correct-password"

// 	hash, err := bcrypt.GenerateFromPassword(
// 		[]byte(password),
// 		bcrypt.DefaultCost,
// 	)
// 	if err != nil {
// 		b.Fatal(err)
// 	}

// 	repository := &fakeRepository{
// 		getByEmailUser: &User{
// 			ID:           1,
// 			Email:        "alex@example.com",
// 			PasswordHash: string(hash),
// 			Role:         RoleUser,
// 		},
// 	}

// 	service := NewService(repository)

// 	input := &LoginUser{
// 		Email:    "alex@example.com",
// 		Password: "wrong-password",
// 	}

// 	b.ResetTimer()

// 	for b.Loop() {
// 		_, _ = service.Login(
// 			context.Background(),
// 			input,
// 		)
// 	}
// }

// func BenchmarkServiceLoginMissingUser(b *testing.B) {
// 	repository := &fakeRepository{
// 		getByEmailErr: pgx.ErrNoRows,
// 	}

// 	service := NewService(repository)

// 	input := &LoginUser{
// 		Email:    "missing@example.com",
// 		Password: "wrong-password",
// 	}

// 	b.ResetTimer()

// 	for b.Loop() {
// 		_, _ = service.Login(
// 			context.Background(),
// 			input,
// 		)
// 	}
// }
