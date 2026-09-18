package auth

import (
	"context"
	"net/http"
	"strings"
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
					writeJSON(
						w,
						http.StatusUnauthorized,
						map[string]string{
							"error": "authorization header is required",
						},
					)
					return
				}

				const prefix = "Bearer "

				if !strings.HasPrefix(authHeader, prefix) {
					writeJSON(
						w,
						http.StatusUnauthorized,
						map[string]string{
							"error": "invalid authorization header",
						},
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
					writeJSON(
						w,
						http.StatusUnauthorized,
						map[string]string{
							"error": "invalid authorization token",
						},
					)
					return
				}

				claims, err := tokens.Parse(
					tokenString,
				)
				if err != nil {
					writeJSON(
						w,
						http.StatusUnauthorized,
						map[string]string{
							"error": "invalid authorization token",
						},
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
		return http.HandlerFunc(func(
			w http.ResponseWriter,
			r *http.Request,
		) {
			role, ok := RoleFromContext(r.Context())
			if !ok {
				http.Error(
					w,
					`{"error":"unauthorized"}`,
					http.StatusUnauthorized,
				)
				return
			}

			if role != requiredRole {
				http.Error(
					w,
					`{"error":"forbidden"}`,
					http.StatusForbidden,
				)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
