package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ThDawnWind/food-delivery-api/internal/user"
)

type fakeLoginService struct {
	input  *user.LoginUser
	result *LoginResult
	err    error

	registerCalled bool
	registerInput  user.RegisterUser
	registerUser   *user.User
	registerErr    error
}

func (f *fakeLoginService) Register(_ context.Context, input user.RegisterUser) (*user.User, error) {
	f.registerInput = input
	f.registerCalled = true

	if f.registerErr != nil {
		return nil, f.registerErr
	}

	return f.registerUser, nil
}

func (f *fakeLoginService) Login(_ context.Context, input *user.LoginUser) (*LoginResult, error) {
	f.input = input

	if f.err != nil {
		return nil, f.err
	}

	return f.result, nil
}

func TestHandler_Login(t *testing.T) {
	t.Parallel()

	service := &fakeLoginService{
		result: &LoginResult{
			AccessToken: testAccessToken,
			User: &user.User{
				ID:       10,
				Username: testUsername,
				Email:    testEmail,
				Role:     user.RoleUser,
			},
		},
	}

	handler := NewHandler(
		service,
		newTestLogger(),
	)

	body := []byte(`{
		"email": "alex@example.com",
		"password": "password123"
	}`)

	req := httptest.NewRequestWithContext(
		t.Context(),
		http.MethodPost,
		"/login",
		bytes.NewReader(body),
	)

	rec := httptest.NewRecorder()

	handler.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusOK,
			rec.Code,
		)
	}

	if service.input == nil {
		t.Fatal("expected login input to be passed to service")
	}

	if service.input.Email != testEmail {
		t.Errorf(
			"expected email %q, got %q",
			testEmail,
			service.input.Email,
		)
	}

	if service.input.Password != testPassword {
		t.Errorf(
			"expected password %q, got %q",
			testPassword,
			service.input.Password,
		)
	}

	var response LoginResult

	err := json.NewDecoder(rec.Body).Decode(&response)
	if err != nil {
		t.Fatalf(
			"failed to decode response: %v",
			err,
		)
	}

	if response.AccessToken != testAccessToken {
		t.Errorf(
			"expected token %q, got %q",
			testAccessToken,
			response.AccessToken,
		)
	}

	if response.User == nil {
		t.Fatal("expected user, got nil")
	}

	if response.User.ID != 10 {
		t.Errorf(
			"expected user ID %d, got %d",
			10,
			response.User.ID,
		)
	}
}

func TestHandler_Login_InvalidJSON(t *testing.T) {
	t.Parallel()

	service := &fakeLoginService{}

	handler := NewHandler(
		service,
		newTestLogger(),
	)

	req := httptest.NewRequestWithContext(
		t.Context(),
		http.MethodPost,
		"/login",
		bytes.NewBufferString(`{"email":`),
	)

	rec := httptest.NewRecorder()

	handler.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusBadRequest,
			rec.Code,
		)
	}

	if service.input != nil {
		t.Fatal(
			"service must not be called for invalid JSON",
		)
	}
}

func TestHandler_Login_ValidationError(t *testing.T) {
	t.Parallel()

	service := &fakeLoginService{
		err: user.ErrUserValidation,
	}

	handler := NewHandler(
		service,
		newTestLogger(),
	)

	body := []byte(`{
		"email": "",
		"password": ""
	}`)

	req := httptest.NewRequestWithContext(
		t.Context(),
		http.MethodPost,
		"/login",
		bytes.NewReader(body),
	)

	rec := httptest.NewRecorder()

	handler.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusBadRequest,
			rec.Code,
		)
	}
}

func TestHandler_Login_InvalidCredentials(t *testing.T) {
	t.Parallel()

	service := &fakeLoginService{
		err: user.ErrInvalidCredentials,
	}

	handler := NewHandler(
		service,
		newTestLogger(),
	)

	body := []byte(`{
		"email": "alex@example.com",
		"password": "wrong-password"
	}`)

	req := httptest.NewRequestWithContext(
		t.Context(),
		http.MethodPost,
		"/login",
		bytes.NewReader(body),
	)

	rec := httptest.NewRecorder()

	handler.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusUnauthorized,
			rec.Code,
		)
	}
}

func TestHandler_Login_InternalError(t *testing.T) {
	t.Parallel()

	service := &fakeLoginService{
		err: errors.New("database unavailable"),
	}

	var logBuffer bytes.Buffer

	logger := slog.New(
		slog.NewJSONHandler(
			&logBuffer,
			nil,
		),
	)

	handler := NewHandler(
		service,
		logger,
	)

	body := []byte(`{
		"email": "alex@example.com",
		"password": "password123"
	}`)

	req := httptest.NewRequestWithContext(
		t.Context(),
		http.MethodPost,
		"/login",
		bytes.NewReader(body),
	)

	rec := httptest.NewRecorder()

	handler.Routes().ServeHTTP(
		rec,
		req,
	)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusInternalServerError,
			rec.Code,
		)
	}

	var response map[string]string

	err := json.NewDecoder(
		rec.Body,
	).Decode(&response)
	if err != nil {
		t.Fatalf(
			"failed to decode response: %v",
			err,
		)
	}

	if response["error"] != "internal server error" {
		t.Errorf(
			"expected safe client error %q, got %q",
			"internal server error",
			response["error"],
		)
	}

	if bytes.Contains(
		rec.Body.Bytes(),
		[]byte("database unavailable"),
	) {
		t.Fatal(
			"internal error must not be exposed to client",
		)
	}

	var logEntry map[string]any

	err = json.Unmarshal(
		logBuffer.Bytes(),
		&logEntry,
	)
	if err != nil {
		t.Fatalf(
			"failed to decode log entry: %v",
			err,
		)
	}

	if logEntry["level"] != "ERROR" {
		t.Errorf(
			"expected ERROR level, got %v",
			logEntry["level"],
		)
	}

	if logEntry["msg"] != "failed to login user" {
		t.Errorf(
			"unexpected log message: %v",
			logEntry["msg"],
		)
	}

	if logEntry["error"] != "database unavailable" {
		t.Errorf(
			"expected internal error in log, got %v",
			logEntry["error"],
		)
	}
}

func TestHandler_Register(t *testing.T) {
	t.Parallel()

	service := &fakeLoginService{
		registerUser: &user.User{
			ID:       10,
			Username: testUsername,
			Email:    testEmail,
			Role:     user.RoleUser,
		},
	}

	handler := NewHandler(
		service,
		newTestLogger(),
	)

	body := []byte(`{
		"username": "alex",
		"email": "alex@example.com",
		"password": "password123"
	}`)

	req := httptest.NewRequestWithContext(
		t.Context(),
		http.MethodPost,
		"/register",
		bytes.NewReader(body),
	)

	rec := httptest.NewRecorder()

	handler.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusCreated,
			rec.Code,
		)
	}

	if !service.registerCalled {
		t.Fatal(
			"expected service Register to be called",
		)
	}

	if service.registerInput.Username != testUsername {
		t.Errorf(
			"expected username %q, got %q",
			testUsername,
			service.registerInput.Username,
		)
	}

	if service.registerInput.Email != testEmail {
		t.Errorf(
			"expected email %q, got %q",
			testEmail,
			service.registerInput.Email,
		)
	}

	if service.registerInput.Password != testPassword {
		t.Errorf(
			"expected password %q, got %q",
			testPassword,
			service.registerInput.Password,
		)
	}
}

func TestHandler_Register_InvalidJSON(t *testing.T) {
	t.Parallel()

	service := &fakeLoginService{}
	handler := NewHandler(
		service,
		newTestLogger(),
	)

	req := httptest.NewRequestWithContext(
		t.Context(),
		http.MethodPost,
		"/register",
		bytes.NewBufferString(`{"username":`),
	)

	rec := httptest.NewRecorder()

	handler.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusBadRequest,
			rec.Code,
		)
	}

	if service.registerCalled {
		t.Fatal(
			"service must not be called for invalid JSON",
		)
	}
}

func TestHandler_Register_ValidationError(t *testing.T) {
	t.Parallel()

	service := &fakeLoginService{
		registerErr: user.ErrUserValidation,
	}

	handler := NewHandler(
		service,
		newTestLogger(),
	)

	body := []byte(`{
		"username": "",
		"email": "",
		"password": ""
	}`)

	req := httptest.NewRequestWithContext(
		t.Context(),
		http.MethodPost,
		"/register",
		bytes.NewReader(body),
	)

	rec := httptest.NewRecorder()

	handler.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusBadRequest,
			rec.Code,
		)
	}
}

func TestHandler_Register_Conflict(t *testing.T) {
	t.Parallel()

	service := &fakeLoginService{
		registerErr: user.ErrUserConflict,
	}

	handler := NewHandler(
		service,
		newTestLogger(),
	)

	body := []byte(`{
		"username": "alex",
		"email": "alex@example.com",
		"password": "password123"
	}`)

	req := httptest.NewRequestWithContext(
		t.Context(),
		http.MethodPost,
		"/register",
		bytes.NewReader(body),
	)

	rec := httptest.NewRecorder()

	handler.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusConflict {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusConflict,
			rec.Code,
		)
	}
}

func TestHandler_Register_InternalError(t *testing.T) {
	t.Parallel()

	service := &fakeLoginService{
		registerErr: errors.New("database unavailable"),
	}

	var logBuffer bytes.Buffer

	logger := slog.New(
		slog.NewJSONHandler(
			&logBuffer,
			nil,
		),
	)

	handler := NewHandler(
		service,
		logger,
	)

	body := []byte(`{
		"username": "alex",
		"email": "alex@example.com",
		"password": "password123"
	}`)

	req := httptest.NewRequestWithContext(
		t.Context(),
		http.MethodPost,
		"/register",
		bytes.NewReader(body),
	)

	rec := httptest.NewRecorder()

	handler.Routes().ServeHTTP(
		rec,
		req,
	)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusInternalServerError,
			rec.Code,
		)
	}

	var response map[string]string

	err := json.NewDecoder(
		rec.Body,
	).Decode(&response)
	if err != nil {
		t.Fatalf(
			"failed to decode response: %v",
			err,
		)
	}

	if response["error"] != "internal server error" {
		t.Errorf(
			"expected safe client error %q, got %q",
			"internal server error",
			response["error"],
		)
	}

	if bytes.Contains(
		rec.Body.Bytes(),
		[]byte("database unavailable"),
	) {
		t.Fatal(
			"internal error must not be exposed to client",
		)
	}

	var logEntry map[string]any

	err = json.Unmarshal(
		logBuffer.Bytes(),
		&logEntry,
	)
	if err != nil {
		t.Fatalf(
			"failed to decode log entry: %v",
			err,
		)
	}

	if logEntry["level"] != "ERROR" {
		t.Errorf(
			"expected ERROR level, got %v",
			logEntry["level"],
		)
	}

	if logEntry["msg"] != "failed to register user" {
		t.Errorf(
			"unexpected log message: %v",
			logEntry["msg"],
		)
	}

	if logEntry["error"] != "database unavailable" {
		t.Errorf(
			"expected internal error in log, got %v",
			logEntry["error"],
		)
	}
}

func newTestLogger() *slog.Logger {
	return slog.New(slog.DiscardHandler)
}
