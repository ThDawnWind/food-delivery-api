package auth

import (
	"context"
	"net/http"
	"strings"

	"github.com/ThDawnWind/food-delivery-api/internal/httpx"
)

type contextKey string

const (
	userIDContextKey contextKey = "user_id"
	roleContextKey   contextKey = "role"
)

type TokenParser interface {
	Parse(tokenString string) (*Claims, error)
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

func RequireRole(requiredRole string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(
			func(
				w http.ResponseWriter,
				r *http.Request,
			) {
				role, ok := RoleFromContext(
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

				if role != requiredRole {
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
