package auth

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var ErrInvalidToken = errors.New("invalid token")

type Claims struct {
	UserID int64  `json:"user_id"`
	Role   string `json:"role"`

	jwt.RegisteredClaims
}

type TokenManager struct {
	secret []byte
	ttl    time.Duration
}

func NewTokenManager(secret string, ttl time.Duration) (*TokenManager, error) {
	secret = strings.TrimSpace(secret)

	if len(secret) < 32 {
		return nil, errors.New(
			"jwt secret must be at least 32 characters",
		)
	}

	if ttl <= 0 {
		return nil, errors.New(
			"jwt ttl must be greater than zero",
		)
	}

	return &TokenManager{
		secret: []byte(secret),
		ttl:    ttl,
	}, nil
}

func (m *TokenManager) Generate(userID int64, role string) (string, error) {
	if userID <= 0 {
		return "", errors.New(
			"invalid user id",
		)
	}

	role = strings.TrimSpace(role)

	if role == "" {
		return "", errors.New(
			"role is required",
		)
	}

	now := time.Now()

	claims := Claims{
		UserID: userID,
		Role:   role,

		Subject: strconv.FormatInt(
			userID,
			10,
		),

		IssuedAt: jwt.NewNumericDate(
			now,
		),

		ExpiresAt: jwt.NewNumericDate(
			now.Add(m.ttl),
		),
	}

	token := jwt.NewWithClaims(
		jwt.SigningMethodHS256,
		claims,
	)

	tokenString, err := token.SignedString(
		m.secret,
	)
	if err != nil {
		return "", fmt.Errorf(
			"failed to sign token: %w",
			err,
		)
	}

	return tokenString, nil
}

func (m *TokenManager) Parse(tokenString string) (*Claims, error) {
	tokenString = strings.TrimSpace(tokenString)

	if tokenString == "" {
		return nil, ErrInvalidToken
	}

	claims := &Claims{}

	token, err := jwt.ParseWithClaims(
		tokenString,
		claims,
		func(t *jwt.Token) (any, error) {
			return m.secret, nil
		},
		jwt.WithValidMethods(
			[]string{
				jwt.SigningMethodHS256.Alg(),
			},
		),
		jwt.WithExpirationRequired(),
	)
	if err != nil {
		return nil, fmt.Errorf(
			"%w: %v",
			ErrInvalidToken,
			err,
		)
	}

	if !token.Valid {
		return nil, ErrInvalidToken
	}

	if claims.UserID <= 0 {
		return nil, ErrInvalidToken
	}

	if strings.TrimSpace(claims.Role) == "" {
		return nil, ErrInvalidToken
	}

	return claims, nil
}
