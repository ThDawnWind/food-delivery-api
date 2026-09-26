package address

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

	"github.com/ThDawnWind/food-delivery-api/internal/auth"
)

type fakeAddressService struct {
	createInput   *CreateAddress
	createAddress *Address
	createErr     error

	getByIDAddressID int64
	getByIDUserID    int64
	getByIDAddress   *Address
	getByIDErr       error

	listByUserID        int64
	listByUserAddresses []Address
	listByUserErr       error

	updateAddressID int64
	updateUserID    int64
	updateInput     *UpdateAddress
	updateAddress   *Address
	updateErr       error

	deleteAddressID int64
	deleteUserID    int64
	deleteErr       error
}

func (f *fakeAddressService) Create(
	ctx context.Context,
	input *CreateAddress,
) (*Address, error) {
	f.createInput = input

	if f.createErr != nil {
		return nil, f.createErr
	}

	return f.createAddress, nil
}

func (f *fakeAddressService) GetByID(
	ctx context.Context,
	addressID int64,
	userID int64,
) (*Address, error) {
	f.getByIDAddressID = addressID
	f.getByIDUserID = userID

	if f.getByIDErr != nil {
		return nil, f.getByIDErr
	}

	return f.getByIDAddress, nil
}

func (f *fakeAddressService) ListByUser(
	ctx context.Context,
	userID int64,
) ([]Address, error) {
	f.listByUserID = userID

	if f.listByUserErr != nil {
		return nil, f.listByUserErr
	}

	return f.listByUserAddresses, nil
}

func (f *fakeAddressService) Update(
	ctx context.Context,
	addressID int64,
	userID int64,
	input *UpdateAddress,
) (*Address, error) {
	f.updateAddressID = addressID
	f.updateUserID = userID
	f.updateInput = input

	if f.updateErr != nil {
		return nil, f.updateErr
	}

	return f.updateAddress, nil
}

func (f *fakeAddressService) Delete(
	ctx context.Context,
	addressID int64,
	userID int64,
) error {
	f.deleteAddressID = addressID
	f.deleteUserID = userID

	return f.deleteErr
}

type fakeTokenParser struct {
	claims *auth.Claims
	err    error
}

func (f *fakeTokenParser) Parse(
	token string,
) (*auth.Claims, error) {
	if f.err != nil {
		return nil, f.err
	}

	return f.claims, nil
}

func authenticatedAddressHandler(
	handler http.Handler,
	userID int64,
	role string,
) http.Handler {
	parser := &fakeTokenParser{
		claims: &auth.Claims{
			UserID: userID,
			Role:   role,
		},
	}

	return auth.Middleware(parser)(
		handler,
	)
}

func newTestLogger() *slog.Logger {
	return slog.New(
		slog.NewJSONHandler(
			io.Discard,
			nil,
		),
	)
}

func TestHandler_Create(t *testing.T) {
	service := &fakeAddressService{
		createAddress: &Address{
			ID:          1,
			UserID:      10,
			City:        "Amsterdam",
			Street:      "Test Street",
			HouseNumber: "10",
		},
	}

	handler := NewHandler(
		service,
		newTestLogger(),
	)

	body := []byte(`{
		"label": "Home",
		"city": "Amsterdam",
		"street": "Test Street",
		"house_number": "10",
		"apartment_number": "42"
	}`)

	request := httptest.NewRequest(
		http.MethodPost,
		"/",
		bytes.NewReader(body),
	)

	request.Header.Set(
		"Authorization",
		"Bearer test-token",
	)

	recorder := httptest.NewRecorder()

	authenticatedAddressHandler(
		handler.Routes(),
		10,
		"user",
	).ServeHTTP(
		recorder,
		request,
	)

	if recorder.Code != http.StatusCreated {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusCreated,
			recorder.Code,
		)
	}

	if service.createInput == nil {
		t.Fatal(
			"expected create input",
		)
	}

	if service.createInput.UserID != 10 {
		t.Errorf(
			"expected user ID %d, got %d",
			10,
			service.createInput.UserID,
		)
	}

	if service.createInput.City != "Amsterdam" {
		t.Errorf(
			"expected city %q, got %q",
			"Amsterdam",
			service.createInput.City,
		)
	}

	if service.createInput.Street != "Test Street" {
		t.Errorf(
			"expected street %q, got %q",
			"Test Street",
			service.createInput.Street,
		)
	}

	if service.createInput.HouseNumber != "10" {
		t.Errorf(
			"expected house number %q, got %q",
			"10",
			service.createInput.HouseNumber,
		)
	}

	var response Address

	if err := json.NewDecoder(
		recorder.Body,
	).Decode(&response); err != nil {
		t.Fatalf(
			"failed to decode response: %v",
			err,
		)
	}

	if response.ID != 1 {
		t.Errorf(
			"expected address ID %d, got %d",
			1,
			response.ID,
		)
	}
}

func TestHandler_Create_Unauthorized(
	t *testing.T,
) {
	service := &fakeAddressService{}

	handler := NewHandler(
		service,
		newTestLogger(),
	)

	body := []byte(`{
		"city": "Amsterdam",
		"street": "Test Street",
		"house_number": "10"
	}`)

	request := httptest.NewRequest(
		http.MethodPost,
		"/",
		bytes.NewReader(body),
	)

	recorder := httptest.NewRecorder()

	handler.Routes().ServeHTTP(
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

	if service.createInput != nil {
		t.Fatal(
			"service must not be called for unauthorized request",
		)
	}
}

func TestHandler_Create_InvalidJSON(
	t *testing.T,
) {
	service := &fakeAddressService{}

	handler := NewHandler(
		service,
		newTestLogger(),
	)

	request := httptest.NewRequest(
		http.MethodPost,
		"/",
		bytes.NewBufferString(
			`{invalid json`,
		),
	)

	request.Header.Set(
		"Authorization",
		"Bearer test-token",
	)

	recorder := httptest.NewRecorder()

	authenticatedAddressHandler(
		handler.Routes(),
		10,
		"user",
	).ServeHTTP(
		recorder,
		request,
	)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusBadRequest,
			recorder.Code,
		)
	}

	if service.createInput != nil {
		t.Fatal(
			"service must not be called for invalid JSON",
		)
	}
}

func TestHandler_Create_ValidationError(
	t *testing.T,
) {
	service := &fakeAddressService{
		createErr: ErrAddressValidation,
	}

	handler := NewHandler(
		service,
		newTestLogger(),
	)

	body := []byte(`{
		"city": "",
		"street": "Test Street",
		"house_number": "10"
	}`)

	request := httptest.NewRequest(
		http.MethodPost,
		"/",
		bytes.NewReader(body),
	)

	request.Header.Set(
		"Authorization",
		"Bearer test-token",
	)

	recorder := httptest.NewRecorder()

	authenticatedAddressHandler(
		handler.Routes(),
		10,
		"user",
	).ServeHTTP(
		recorder,
		request,
	)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusBadRequest,
			recorder.Code,
		)
	}
}

func TestHandler_Create_InternalError(
	t *testing.T,
) {
	service := &fakeAddressService{
		createErr: errors.New(
			"database unavailable",
		),
	}

	handler := NewHandler(
		service,
		newTestLogger(),
	)

	body := []byte(`{
		"city": "Amsterdam",
		"street": "Test Street",
		"house_number": "10"
	}`)

	request := httptest.NewRequest(
		http.MethodPost,
		"/",
		bytes.NewReader(body),
	)

	request.Header.Set(
		"Authorization",
		"Bearer test-token",
	)

	recorder := httptest.NewRecorder()

	authenticatedAddressHandler(
		handler.Routes(),
		10,
		"user",
	).ServeHTTP(
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

func TestHandler_ListByUser(t *testing.T) {
	service := &fakeAddressService{
		listByUserAddresses: []Address{
			{
				ID:          1,
				UserID:      10,
				City:        "Amsterdam",
				Street:      "First Street",
				HouseNumber: "1",
			},
			{
				ID:          2,
				UserID:      10,
				City:        "Amsterdam",
				Street:      "Second Street",
				HouseNumber: "2",
			},
		},
	}

	handler := NewHandler(
		service,
		newTestLogger(),
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

	authenticatedAddressHandler(
		handler.Routes(),
		10,
		"user",
	).ServeHTTP(
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

	if service.listByUserID != 10 {
		t.Errorf(
			"expected user ID %d, got %d",
			10,
			service.listByUserID,
		)
	}

	var response []Address

	if err := json.NewDecoder(
		recorder.Body,
	).Decode(&response); err != nil {
		t.Fatalf(
			"failed to decode response: %v",
			err,
		)
	}

	if len(response) != 2 {
		t.Fatalf(
			"expected %d addresses, got %d",
			2,
			len(response),
		)
	}
}

func TestHandler_ListByUser_Unauthorized(
	t *testing.T,
) {
	service := &fakeAddressService{}

	handler := NewHandler(
		service,
		newTestLogger(),
	)

	request := httptest.NewRequest(
		http.MethodGet,
		"/",
		nil,
	)

	recorder := httptest.NewRecorder()

	handler.Routes().ServeHTTP(
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

	if service.listByUserID != 0 {
		t.Fatal(
			"service must not be called for unauthorized request",
		)
	}
}

func TestHandler_ListByUser_InternalError(
	t *testing.T,
) {
	service := &fakeAddressService{
		listByUserErr: errors.New(
			"database unavailable",
		),
	}

	handler := NewHandler(
		service,
		newTestLogger(),
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

	authenticatedAddressHandler(
		handler.Routes(),
		10,
		"user",
	).ServeHTTP(
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

func TestHandler_GetByID(t *testing.T) {
	service := &fakeAddressService{
		getByIDAddress: &Address{
			ID:          5,
			UserID:      10,
			City:        "Amsterdam",
			Street:      "Test Street",
			HouseNumber: "10",
		},
	}

	handler := NewHandler(
		service,
		newTestLogger(),
	)

	request := httptest.NewRequest(
		http.MethodGet,
		"/5",
		nil,
	)

	request.Header.Set(
		"Authorization",
		"Bearer test-token",
	)

	recorder := httptest.NewRecorder()

	authenticatedAddressHandler(
		handler.Routes(),
		10,
		"user",
	).ServeHTTP(
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

	if service.getByIDAddressID != 5 {
		t.Errorf(
			"expected address ID %d, got %d",
			5,
			service.getByIDAddressID,
		)
	}

	if service.getByIDUserID != 10 {
		t.Errorf(
			"expected user ID %d, got %d",
			10,
			service.getByIDUserID,
		)
	}

	var response Address

	if err := json.NewDecoder(
		recorder.Body,
	).Decode(&response); err != nil {
		t.Fatalf(
			"failed to decode response: %v",
			err,
		)
	}

	if response.ID != 5 {
		t.Errorf(
			"expected address ID %d, got %d",
			5,
			response.ID,
		)
	}
}

func TestHandler_GetByID_InvalidID(
	t *testing.T,
) {
	service := &fakeAddressService{}

	handler := NewHandler(
		service,
		newTestLogger(),
	)

	request := httptest.NewRequest(
		http.MethodGet,
		"/abc",
		nil,
	)

	recorder := httptest.NewRecorder()

	handler.Routes().ServeHTTP(
		recorder,
		request,
	)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusBadRequest,
			recorder.Code,
		)
	}
}

func TestHandler_GetByID_NotFound(
	t *testing.T,
) {
	service := &fakeAddressService{
		getByIDErr: ErrAddressNotFound,
	}

	handler := NewHandler(
		service,
		newTestLogger(),
	)

	request := httptest.NewRequest(
		http.MethodGet,
		"/999",
		nil,
	)

	request.Header.Set(
		"Authorization",
		"Bearer test-token",
	)

	recorder := httptest.NewRecorder()

	authenticatedAddressHandler(
		handler.Routes(),
		10,
		"user",
	).ServeHTTP(
		recorder,
		request,
	)

	if recorder.Code != http.StatusNotFound {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusNotFound,
			recorder.Code,
		)
	}
}

func TestHandler_GetByID_Unauthorized(
	t *testing.T,
) {
	service := &fakeAddressService{}

	handler := NewHandler(
		service,
		newTestLogger(),
	)

	request := httptest.NewRequest(
		http.MethodGet,
		"/5",
		nil,
	)

	recorder := httptest.NewRecorder()

	handler.Routes().ServeHTTP(
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

func TestHandler_Update(t *testing.T) {
	service := &fakeAddressService{
		updateAddress: &Address{
			ID:          5,
			UserID:      10,
			City:        "Rotterdam",
			Street:      "New Street",
			HouseNumber: "20",
		},
	}

	handler := NewHandler(
		service,
		newTestLogger(),
	)

	body := []byte(`{
		"city": "Rotterdam",
		"street": "New Street"
	}`)

	request := httptest.NewRequest(
		http.MethodPatch,
		"/5",
		bytes.NewReader(body),
	)

	request.Header.Set(
		"Authorization",
		"Bearer test-token",
	)

	recorder := httptest.NewRecorder()

	authenticatedAddressHandler(
		handler.Routes(),
		10,
		"user",
	).ServeHTTP(
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

	if service.updateAddressID != 5 {
		t.Errorf(
			"expected address ID %d, got %d",
			5,
			service.updateAddressID,
		)
	}

	if service.updateUserID != 10 {
		t.Errorf(
			"expected user ID %d, got %d",
			10,
			service.updateUserID,
		)
	}

	if service.updateInput == nil {
		t.Fatal(
			"expected update input",
		)
	}

	if service.updateInput.City == nil {
		t.Fatal(
			"expected city",
		)
	}

	if *service.updateInput.City != "Rotterdam" {
		t.Errorf(
			"expected city %q, got %q",
			"Rotterdam",
			*service.updateInput.City,
		)
	}
}

func TestHandler_Update_NotFound(
	t *testing.T,
) {
	service := &fakeAddressService{
		updateErr: ErrAddressNotFound,
	}

	handler := NewHandler(
		service,
		newTestLogger(),
	)

	request := httptest.NewRequest(
		http.MethodPatch,
		"/999",
		bytes.NewBufferString(
			`{"city":"Rotterdam"}`,
		),
	)

	request.Header.Set(
		"Authorization",
		"Bearer test-token",
	)

	recorder := httptest.NewRecorder()

	authenticatedAddressHandler(
		handler.Routes(),
		10,
		"user",
	).ServeHTTP(
		recorder,
		request,
	)

	if recorder.Code != http.StatusNotFound {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusNotFound,
			recorder.Code,
		)
	}
}

func TestHandler_Update_InvalidJSON(
	t *testing.T,
) {
	service := &fakeAddressService{}

	handler := NewHandler(
		service,
		newTestLogger(),
	)

	request := httptest.NewRequest(
		http.MethodPatch,
		"/5",
		bytes.NewBufferString(
			`{invalid`,
		),
	)

	request.Header.Set(
		"Authorization",
		"Bearer test-token",
	)

	recorder := httptest.NewRecorder()

	authenticatedAddressHandler(
		handler.Routes(),
		10,
		"user",
	).ServeHTTP(
		recorder,
		request,
	)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusBadRequest,
			recorder.Code,
		)
	}
}

func TestHandler_Delete(t *testing.T) {
	service := &fakeAddressService{}

	handler := NewHandler(
		service,
		newTestLogger(),
	)

	request := httptest.NewRequest(
		http.MethodDelete,
		"/5",
		nil,
	)

	request.Header.Set(
		"Authorization",
		"Bearer test-token",
	)

	recorder := httptest.NewRecorder()

	authenticatedAddressHandler(
		handler.Routes(),
		10,
		"user",
	).ServeHTTP(
		recorder,
		request,
	)

	if recorder.Code != http.StatusNoContent {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusNoContent,
			recorder.Code,
		)
	}

	if service.deleteAddressID != 5 {
		t.Errorf(
			"expected address ID %d, got %d",
			5,
			service.deleteAddressID,
		)
	}

	if service.deleteUserID != 10 {
		t.Errorf(
			"expected user ID %d, got %d",
			10,
			service.deleteUserID,
		)
	}
}

func TestHandler_Delete_NotFound(
	t *testing.T,
) {
	service := &fakeAddressService{
		deleteErr: ErrAddressNotFound,
	}

	handler := NewHandler(
		service,
		newTestLogger(),
	)

	request := httptest.NewRequest(
		http.MethodDelete,
		"/999",
		nil,
	)

	request.Header.Set(
		"Authorization",
		"Bearer test-token",
	)

	recorder := httptest.NewRecorder()

	authenticatedAddressHandler(
		handler.Routes(),
		10,
		"user",
	).ServeHTTP(
		recorder,
		request,
	)

	if recorder.Code != http.StatusNotFound {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusNotFound,
			recorder.Code,
		)
	}
}

func TestHandler_Delete_Unauthorized(
	t *testing.T,
) {
	service := &fakeAddressService{}

	handler := NewHandler(
		service,
		newTestLogger(),
	)

	request := httptest.NewRequest(
		http.MethodDelete,
		"/5",
		nil,
	)

	recorder := httptest.NewRecorder()

	handler.Routes().ServeHTTP(
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

	if service.deleteAddressID != 0 {
		t.Fatal(
			"service must not be called for unauthorized request",
		)
	}
}
