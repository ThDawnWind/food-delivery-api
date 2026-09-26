package auth

import (
	"errors"
	"strconv"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func TestTokenManager_GenerateAndParse(t *testing.T) {
	manager, err := NewTokenManager(
		"this-is-a-test-secret-key-with-32-chars",
		time.Hour,
	)

	if err != nil {
		t.Fatalf(
			"failed to create token manager: %v",
			err,
		)
	}

	tokenString, err := manager.Generate(
		10,
		"user",
	)
	if err != nil {
		t.Fatalf(
			"failed to generate token: %v",
			err,
		)
	}

	if tokenString == "" {
		t.Fatal("expected token, got empty string")
	}

	claims, err := manager.Parse(tokenString)
	if err != nil {
		t.Fatalf(
			"failed to parse token: %v",
			err,
		)
	}

	if claims.UserID != 10 {
		t.Errorf(
			"expected user ID %d, got %d",
			10,
			claims.UserID,
		)
	}

	if claims.Role != "user" {
		t.Errorf(
			"expected role %q, got %q",
			"user",
			claims.Role,
		)
	}

	if claims.Subject != "10" {
		t.Errorf(
			"expected subject %q, got %q",
			"10",
			claims.Subject,
		)
	}
	if claims.ExpiresAt == nil {
		t.Fatal("expected expires_at claim")
	}

	if claims.IssuedAt == nil {
		t.Fatal("expected issued_at claim")
	}
}

func TestTokenManager_Parse_WrongSecret(t *testing.T) {
	managerA, err := NewTokenManager(
		"first-test-secret-key-with-32-characters",
		time.Hour,
	)
	if err != nil {
		t.Fatalf("failed to create manager A: %v", err)
	}

	managerB, err := NewTokenManager(
		"second-test-secret-key-with-32-characters",
		time.Hour,
	)
	if err != nil {
		t.Fatalf("failed to create manager B: %v", err)
	}

	tokenString, err := managerA.Generate(
		10,
		"user",
	)
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	claims, err := managerB.Parse(tokenString)

	if claims != nil {
		t.Fatalf(
			"expected nil claims, got %+v",
			claims,
		)
	}

	if !errors.Is(err, ErrInvalidToken) {
		t.Fatalf(
			"expected ErrInvalidToken, got %v",
			err,
		)
	}
}

func TestTokenManager_Parse_InvalidToken(t *testing.T) {
	manager, err := NewTokenManager(
		"this-is-a-test-secret-key-with-32-chars",
		time.Hour,
	)
	if err != nil {
		t.Fatalf("failed to create token manager: %v", err)
	}

	claims, err := manager.Parse(
		"this-is-not-a-jwt",
	)

	if claims != nil {
		t.Fatalf(
			"expected nil claims, got %+v",
			claims,
		)
	}

	if !errors.Is(err, ErrInvalidToken) {
		t.Fatalf(
			"expected ErrInvalidToken, got %v",
			err,
		)
	}
}

func TestTokenManager_Parse_ExpiredToken(t *testing.T) {
	manager, err := NewTokenManager(
		"this-is-a-test-secret-key-with-32-chars",
		time.Hour,
	)
	if err != nil {
		t.Fatalf("failed to create token manager: %v", err)
	}

	now := time.Now()

	claims := Claims{
		UserID: 10,
		Role:   "user",

		Subject: strconv.FormatInt(
			10,
			10,
		),

		IssuedAt: jwt.NewNumericDate(
			now.Add(-2 * time.Hour),
		),

		ExpiresAt: jwt.NewNumericDate(
			now.Add(-time.Hour),
		),
	}

	token := jwt.NewWithClaims(
		jwt.SigningMethodHS256,
		claims,
	)

	tokenString, err := token.SignedString(
		manager.secret,
	)
	if err != nil {
		t.Fatalf(
			"failed to sign token: %v",
			err,
		)
	}

	parsedClaims, err := manager.Parse(tokenString)

	if parsedClaims != nil {
		t.Fatalf(
			"expected nil claims, got %+v",
			parsedClaims,
		)
	}

	if !errors.Is(err, ErrInvalidToken) {
		t.Fatalf(
			"expected ErrInvalidToken, got %v",
			err,
		)
	}
}

func TestTokenManager_Generate_InvalidUserID(t *testing.T) {
	manager, err := NewTokenManager(
		"this-is-a-test-secret-key-with-32-chars",
		time.Hour,
	)
	if err != nil {
		t.Fatalf("failed to create token manager: %v", err)
	}

	token, err := manager.Generate(
		0,
		"user",
	)

	if token != "" {
		t.Fatalf(
			"expected empty token, got %q",
			token,
		)
	}

	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestTokenManager_Generate_EmptyRole(t *testing.T) {
	manager, err := NewTokenManager(
		"this-is-a-test-secret-key-with-32-chars",
		time.Hour,
	)
	if err != nil {
		t.Fatalf("failed to create token manager: %v", err)
	}

	token, err := manager.Generate(
		10,
		"   ",
	)

	if token != "" {
		t.Fatalf(
			"expected empty token, got %q",
			token,
		)
	}

	if err == nil {
		t.Fatal("expected error, got nil")
	}
}
