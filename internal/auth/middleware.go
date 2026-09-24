package auth

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/ThDawnWind/food-delivery-api/internal/httpx"
	"github.com/ThDawnWind/food-delivery-api/internal/user"
)

type contextKey string

const (
	userIDContextKey contextKey = "user_id"
	roleContextKey   contextKey = "role"
)

type TokenParser interface {
	Parse(tokenString string) (*Claims, error)
}

type UserProvider interface {
	GetByID(ctx context.Context, id int64) (*user.User, error)
}

func Middleware(tokens TokenParser) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				authHeader := strings.TrimSpace(
					r.Header.Get("Authorization"),
				)

				if authHeader == "" {
					httpx.WriteError(
						w,
						http.StatusUnauthorized,
						"authorization header is required",
					)
					return
				}

				const prefix = "Bearer "

				if !strings.HasPrefix(
					authHeader,
					prefix,
				) {
					httpx.WriteError(
						w,
						http.StatusUnauthorized,
						"invalid authorization header",
					)
					return
				}

				tokenString := strings.TrimSpace(
					strings.TrimPrefix(
						authHeader,
						prefix,
					),
				)

				if tokenString == "" {
					httpx.WriteError(
						w,
						http.StatusUnauthorized,
						"invalid authorization token",
					)
					return
				}

				claims, err := tokens.Parse(
					tokenString,
				)
				if err != nil {
					httpx.WriteError(
						w,
						http.StatusUnauthorized,
						"invalid authorization token",
					)
					return
				}

				ctx := context.WithValue(
					r.Context(),
					userIDContextKey,
					claims.UserID,
				)

				ctx = context.WithValue(
					ctx,
					roleContextKey,
					claims.Role,
				)

				next.ServeHTTP(
					w,
					r.WithContext(ctx),
				)
			},
		)
	}
}

func UserIDFromContext(ctx context.Context) (int64, bool) {
	userID, ok := ctx.Value(
		userIDContextKey,
	).(int64)

	return userID, ok
}

func RoleFromContext(ctx context.Context) (string, bool) {
	role, ok := ctx.Value(
		roleContextKey,
	).(string)

	return role, ok
}

func RequireRole(users UserProvider, requiredRole user.Role) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(
			func(
				w http.ResponseWriter,
				r *http.Request,
			) {
				userID, ok := UserIDFromContext(
					r.Context(),
				)
				if !ok {
					httpx.WriteError(
						w,
						http.StatusUnauthorized,
						"unauthorized",
					)
					return
				}

				currentUser, err := users.GetByID(
					r.Context(),
					userID,
				)
				if err != nil {
					if errors.Is(
						err,
						user.ErrUserNotFound,
					) {
						httpx.WriteError(
							w,
							http.StatusUnauthorized,
							"unauthorized",
						)
						return
					}

					httpx.WriteError(
						w,
						http.StatusInternalServerError,
						"internal server error",
					)
					return
				}

				if currentUser.Role != requiredRole {
					httpx.WriteError(
						w,
						http.StatusForbidden,
						"forbidden",
					)
					return
				}

				next.ServeHTTP(w, r)
			},
		)
	}
}
