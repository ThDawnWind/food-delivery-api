package auth

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ThDawnWind/food-delivery-api/internal/user"
)

type fakeTokenParser struct {
	claims *Claims
	err    error

	token string
}

type fakeUserProvider struct {
	user *user.User
	err  error

	userID int64
}

func (f *fakeTokenParser) Parse(tokenString string) (*Claims, error) {
	f.token = tokenString

	if f.err != nil {
		return nil, f.err
	}

	return f.claims, nil
}

func (f *fakeUserProvider) GetByID(ctx context.Context, id int64) (*user.User, error) {
	f.userID = id

	if f.err != nil {
		return nil, f.err
	}

	return f.user, nil
}

func TestMiddleware_MissingAuthorizationHeader(t *testing.T) {
	parser := &fakeTokenParser{}

	next := http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			t.Fatal("next handler must not be called")
		},
	)

	handler := Middleware(parser)(next)

	req := httptest.NewRequest(
		http.MethodGet,
		"/test",
		nil,
	)

	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusUnauthorized,
			rec.Code,
		)
	}
}

func TestMiddleware_InvalidAuthorizationHeader(t *testing.T) {
	parser := &fakeTokenParser{}

	next := http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			t.Fatal("next handler must not be called")
		},
	)

	handler := Middleware(parser)(next)

	req := httptest.NewRequest(
		http.MethodGet,
		"/test",
		nil,
	)

	req.Header.Set(
		"Authorization",
		"Basic abc123",
	)

	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusUnauthorized,
			rec.Code,
		)
	}
}

func TestMiddleware_InvalidToken(t *testing.T) {
	parser := &fakeTokenParser{
		err: ErrInvalidToken,
	}

	next := http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			t.Fatal("next handler must not be called")
		},
	)

	handler := Middleware(parser)(next)

	req := httptest.NewRequest(
		http.MethodGet,
		"/test",
		nil,
	)

	req.Header.Set(
		"Authorization",
		"Bearer invalid-token",
	)

	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusUnauthorized,
			rec.Code,
		)
	}

	if parser.token != "invalid-token" {
		t.Errorf(
			"expected token %q, got %q",
			"invalid-token",
			parser.token,
		)
	}
}

func TestMiddleware_ValidToken(t *testing.T) {
	parser := &fakeTokenParser{
		claims: &Claims{
			UserID: 10,
			Role:   "user",
		},
	}

	nextCalled := false

	next := http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			nextCalled = true

			userID, ok := UserIDFromContext(
				r.Context(),
			)
			if !ok {
				t.Fatal(
					"expected user ID in context",
				)
			}

			if userID != 10 {
				t.Errorf(
					"expected user ID %d, got %d",
					10,
					userID,
				)
			}

			role, ok := RoleFromContext(
				r.Context(),
			)
			if !ok {
				t.Fatal(
					"expected role in context",
				)
			}

			if role != "user" {
				t.Errorf(
					"expected role %q, got %q",
					"user",
					role,
				)
			}

			w.WriteHeader(http.StatusOK)
		},
	)

	handler := Middleware(parser)(next)

	req := httptest.NewRequest(
		http.MethodGet,
		"/test",
		nil,
	)

	req.Header.Set(
		"Authorization",
		"Bearer test-token",
	)

	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusOK,
			rec.Code,
		)
	}

	if !nextCalled {
		t.Fatal(
			"expected next handler to be called",
		)
	}

	if parser.token != "test-token" {
		t.Errorf(
			"expected token %q, got %q",
			"test-token",
			parser.token,
		)
	}
}

func TestRequireRole_UsesCurrentRoleFromDatabase(t *testing.T) {
	handler := http.HandlerFunc(func(
		w http.ResponseWriter,
		r *http.Request,
	) {
		w.WriteHeader(http.StatusOK)
	})

	parser := &fakeTokenParser{
		claims: &Claims{
			UserID: 10,
			Role:   "user",
		},
	}

	users := &fakeUserProvider{
		user: &user.User{
			ID:   10,
			Role: user.RoleAdmin,
		},
	}

	protected := Middleware(parser)(
		RequireRole(
			users,
			user.RoleAdmin,
		)(handler),
	)

	request := httptest.NewRequest(
		http.MethodGet,
		"/",
		nil,
	)

	request.Header.Set(
		"Authorization",
		"Bearer test-token",
	)

	recorder := httptest.NewRecorder()

	protected.ServeHTTP(
		recorder,
		request,
	)

	if recorder.Code != http.StatusOK {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusOK,
			recorder.Code,
		)
	}

	if users.userID != 10 {
		t.Errorf(
			"expected user ID %d, got %d",
			10,
			users.userID,
		)
	}
}

func TestRequireRole_RejectsStaleAdminRole(t *testing.T) {
	handler := http.HandlerFunc(func(
		w http.ResponseWriter,
		r *http.Request,
	) {
		t.Fatal("next handler must not be called")
	})

	parser := &fakeTokenParser{
		claims: &Claims{
			UserID: 10,
			Role:   "admin",
		},
	}

	users := &fakeUserProvider{
		user: &user.User{
			ID:   10,
			Role: user.RoleUser,
		},
	}

	protected := Middleware(parser)(
		RequireRole(
			users,
			user.RoleAdmin,
		)(handler),
	)

	request := httptest.NewRequest(
		http.MethodGet,
		"/",
		nil,
	)

	request.Header.Set(
		"Authorization",
		"Bearer test-token",
	)

	recorder := httptest.NewRecorder()

	protected.ServeHTTP(
		recorder,
		request,
	)

	if recorder.Code != http.StatusForbidden {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusForbidden,
			recorder.Code,
		)
	}
}

func TestRequireRole_Unauthorized(t *testing.T) {
	handler := http.HandlerFunc(func(
		w http.ResponseWriter,
		r *http.Request,
	) {
		t.Fatal("next handler must not be called")
	})

	users := &fakeUserProvider{}

	request := httptest.NewRequest(
		http.MethodGet,
		"/",
		nil,
	)

	recorder := httptest.NewRecorder()

	RequireRole(
		users,
		user.RoleAdmin,
	)(handler).ServeHTTP(
		recorder,
		request,
	)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusUnauthorized,
			recorder.Code,
		)
	}
}

func TestRequireRole_UserNotFound(t *testing.T) {
	handler := http.HandlerFunc(func(
		w http.ResponseWriter,
		r *http.Request,
	) {
		t.Fatal("next handler must not be called")
	})

	parser := &fakeTokenParser{
		claims: &Claims{
			UserID: 10,
			Role:   "admin",
		},
	}

	users := &fakeUserProvider{
		err: user.ErrUserNotFound,
	}

	protected := Middleware(parser)(
		RequireRole(
			users,
			user.RoleAdmin,
		)(handler),
	)

	request := httptest.NewRequest(
		http.MethodGet,
		"/",
		nil,
	)

	request.Header.Set(
		"Authorization",
		"Bearer test-token",
	)

	recorder := httptest.NewRecorder()

	protected.ServeHTTP(
		recorder,
		request,
	)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusUnauthorized,
			recorder.Code,
		)
	}
}

func TestRequireRole_UserProviderError(t *testing.T) {
	handler := http.HandlerFunc(func(
		w http.ResponseWriter,
		r *http.Request,
	) {
		t.Fatal("next handler must not be called")
	})

	parser := &fakeTokenParser{
		claims: &Claims{
			UserID: 10,
			Role:   "admin",
		},
	}

	users := &fakeUserProvider{
		err: errors.New("database error"),
	}

	protected := Middleware(parser)(
		RequireRole(
			users,
			user.RoleAdmin,
		)(handler),
	)

	request := httptest.NewRequest(
		http.MethodGet,
		"/",
		nil,
	)

	request.Header.Set(
		"Authorization",
		"Bearer test-token",
	)

	recorder := httptest.NewRecorder()

	protected.ServeHTTP(
		recorder,
		request,
	)

	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusInternalServerError,
			recorder.Code,
		)
	}
}
