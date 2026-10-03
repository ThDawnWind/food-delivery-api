package auth

import (
	"context"
	"errors"
	"testing"

	"github.com/ThDawnWind/food-delivery-api/internal/user"
)

type fakeUserAuthenticator struct {
	user *user.User
	err  error

	registerCalled bool
	registerInput  user.RegisterUser
}

func (f *fakeUserAuthenticator) Register(_ context.Context, input user.RegisterUser) (*user.User, error) {
	f.registerInput = input
	f.registerCalled = true

	if f.err != nil {
		return nil, f.err
	}

	return f.user, nil
}

func (f *fakeUserAuthenticator) Login(_ context.Context, _ user.LoginUser) (*user.User, error) {
	if f.err != nil {
		return nil, f.err
	}

	return f.user, nil
}

type fakeTokenGenerator struct {
	token string
	err   error

	userID int64
	role   string
}

func (f *fakeTokenGenerator) Generate(userID int64, role string) (string, error) {
	f.userID = userID
	f.role = role

	if f.err != nil {
		return "", f.err
	}

	return f.token, nil
}

func TestService_Login(t *testing.T) {
	t.Parallel()

	users := &fakeUserAuthenticator{
		user: &user.User{
			ID:       10,
			Username: testUsername,
			Email:    testEmail,
			Role:     user.RoleUser,
		},
	}

	tokens := &fakeTokenGenerator{
		token: "test-access-token",
	}

	service := NewService(
		users,
		tokens,
	)

	result, err := service.Login(
		context.Background(),
		user.LoginUser{
			Email:    testEmail,
			Password: testPassword,
		},
	)
	if err != nil {
		t.Fatalf(
			"unexpected error: %v",
			err,
		)
	}

	if result == nil {
		t.Fatal("expected login result, got nil")
	}

	if result.AccessToken != testAccessToken {
		t.Errorf(
			"expected access token %q, got %q",
			testAccessToken,
			result.AccessToken,
		)
	}

	if result.User == nil {
		t.Fatal("expected user, got nil")
	}

	if result.User.ID != 10 {
		t.Errorf(
			"expected user ID %d, got %d",
			10,
			result.User.ID,
		)
	}

	if tokens.userID != 10 {
		t.Errorf(
			"expected token user ID %d, got %d",
			10,
			tokens.userID,
		)
	}

	if tokens.role != testRoleUser {
		t.Errorf(
			"expected token role %q, got %q",
			testRoleUser,
			tokens.role,
		)
	}
}

func TestService_Login_UserAuthenticationError(t *testing.T) {
	t.Parallel()

	loginErr := user.ErrInvalidCredentials

	users := &fakeUserAuthenticator{
		err: loginErr,
	}

	tokens := &fakeTokenGenerator{}

	service := NewService(
		users,
		tokens,
	)

	result, err := service.Login(
		context.Background(),
		user.LoginUser{
			Email:    testEmail,
			Password: "wrong-password",
		},
	)

	if result != nil {
		t.Fatalf(
			"expected nil result, got %+v",
			result,
		)
	}

	if !errors.Is(err, loginErr) {
		t.Fatalf(
			"expected login error, got %v",
			err,
		)
	}

	if tokens.userID != 0 {
		t.Fatal(
			"token generator must not be called when authentication fails",
		)
	}
}

func TestService_Login_TokenError(t *testing.T) {
	t.Parallel()

	tokenErr := errors.New("failed to sign token")

	users := &fakeUserAuthenticator{
		user: &user.User{
			ID:       10,
			Username: testUsername,
			Email:    testEmail,
			Role:     user.RoleUser,
		},
	}

	tokens := &fakeTokenGenerator{
		err: tokenErr,
	}

	service := NewService(
		users,
		tokens,
	)

	result, err := service.Login(
		context.Background(),
		user.LoginUser{
			Email:    testEmail,
			Password: testPassword,
		},
	)

	if result != nil {
		t.Fatalf(
			"expected nil result, got %+v",
			result,
		)
	}

	if !errors.Is(err, tokenErr) {
		t.Fatalf(
			"expected token error, got %v",
			err,
		)
	}

	if tokens.userID != 10 {
		t.Errorf(
			"expected token generator user ID %d, got %d",
			10,
			tokens.userID,
		)
	}

	if tokens.role != testRoleUser {
		t.Errorf(
			"expected token role %q, got %q",
			"user",
			tokens.role,
		)
	}
}
