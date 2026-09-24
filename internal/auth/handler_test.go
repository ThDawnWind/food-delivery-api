package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
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

	registerInput *user.RegisterUser
	registerUser  *user.User
	registerErr   error
}

func (f *fakeLoginService) Register(ctx context.Context, input *user.RegisterUser) (*user.User, error) {
	f.registerInput = input

	if f.registerErr != nil {
		return nil, f.registerErr
	}

	return f.registerUser, nil
}

func (f *fakeLoginService) Login(ctx context.Context, input *user.LoginUser) (*LoginResult, error) {
	f.input = input

	if f.err != nil {
		return nil, f.err
	}

	return f.result, nil
}

func TestHandler_Login(t *testing.T) {
	service := &fakeLoginService{
		result: &LoginResult{
			AccessToken: "test-access-token",
			User: &user.User{
				ID:       10,
				Username: "alex",
				Email:    "alex@example.com",
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

	req := httptest.NewRequest(
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

	if service.input.Email != "alex@example.com" {
		t.Errorf(
			"expected email %q, got %q",
			"alex@example.com",
			service.input.Email,
		)
	}

	if service.input.Password != "password123" {
		t.Errorf(
			"expected password %q, got %q",
			"password123",
			service.input.Password,
		)
	}

	var response LoginResult

	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf(
			"failed to decode response: %v",
			err,
		)
	}

	if response.AccessToken != "test-access-token" {
		t.Errorf(
			"expected token %q, got %q",
			"test-access-token",
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
	service := &fakeLoginService{}

	handler := NewHandler(
		service,
		newTestLogger(),
	)

	req := httptest.NewRequest(
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

	req := httptest.NewRequest(
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

	req := httptest.NewRequest(
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

	req := httptest.NewRequest(
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

	if err := json.NewDecoder(
		rec.Body,
	).Decode(&response); err != nil {
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

	if err := json.Unmarshal(
		logBuffer.Bytes(),
		&logEntry,
	); err != nil {
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
	service := &fakeLoginService{
		registerUser: &user.User{
			ID:       10,
			Username: "alex",
			Email:    "alex@example.com",
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

	req := httptest.NewRequest(
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

	if service.registerInput == nil {
		t.Fatal(
			"expected register input to be passed to service",
		)
	}

	if service.registerInput.Username != "alex" {
		t.Errorf(
			"expected username %q, got %q",
			"alex",
			service.registerInput.Username,
		)
	}

	if service.registerInput.Email != "alex@example.com" {
		t.Errorf(
			"expected email %q, got %q",
			"alex@example.com",
			service.registerInput.Email,
		)
	}

	if service.registerInput.Password != "password123" {
		t.Errorf(
			"expected password %q, got %q",
			"password123",
			service.registerInput.Password,
		)
	}
}

func TestHandler_Register_InvalidJSON(t *testing.T) {
	service := &fakeLoginService{}
	handler := NewHandler(
		service,
		newTestLogger(),
	)

	req := httptest.NewRequest(
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

	if service.registerInput != nil {
		t.Fatal(
			"service must not be called for invalid JSON",
		)
	}
}

func TestHandler_Register_ValidationError(t *testing.T) {
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

	req := httptest.NewRequest(
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

	req := httptest.NewRequest(
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

	req := httptest.NewRequest(
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

	if err := json.NewDecoder(
		rec.Body,
	).Decode(&response); err != nil {
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

	if err := json.Unmarshal(
		logBuffer.Bytes(),
		&logEntry,
	); err != nil {
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
	return slog.New(
		slog.NewJSONHandler(
			io.Discard,
			nil,
		),
	)
}
