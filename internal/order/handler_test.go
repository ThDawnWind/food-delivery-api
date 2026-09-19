package order

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ThDawnWind/food-delivery-api/internal/auth"
)

type fakeOrderService struct {
	createInput *CreateOrder
	createOrder *Order
	createErr   error

	getID    int64
	getOrder *Order
	getErr   error

	listUserID int64
	listLimit  int
	listOffset int
	listOrders []Order
	listErr    error

	updateID     int64
	updateStatus Status
	updateErr    error

	listAllLimit  int
	listAllStatus string
	listAllOffset int
	listAllOrders []*Order
	listAllErr    error
}

func (f *fakeOrderService) Create(ctx context.Context, input *CreateOrder) (*Order, error) {
	f.createInput = input

	if f.createErr != nil {
		return nil, f.createErr
	}

	return f.createOrder, nil
}

func (f *fakeOrderService) GetByID(ctx context.Context, id int64) (*Order, error) {
	f.getID = id

	if f.getErr != nil {
		return nil, f.getErr
	}
	return f.getOrder, nil
}

func (f *fakeOrderService) ListByUser(ctx context.Context, userID int64, limit int, offset int) ([]Order, error) {
	f.listUserID = userID
	f.listLimit = limit
	f.listOffset = offset

	if f.listErr != nil {
		return nil, f.listErr
	}
	return f.listOrders, nil
}

func (f *fakeOrderService) ListAll(ctx context.Context, status string, limit int, offset int) ([]*Order, error) {
	f.listAllStatus = status
	f.listAllLimit = limit
	f.listAllOffset = offset

	if f.listAllErr != nil {
		return nil, f.listAllErr
	}

	return f.listAllOrders, nil
}

func (f *fakeOrderService) UpdateStatus(ctx context.Context, id int64, status Status) error {
	f.updateID = id
	f.updateStatus = status
	return f.updateErr
}

type fakeTokenParser struct {
	claims *auth.Claims
}

func (f *fakeTokenParser) Parse(tokenString string) (*auth.Claims, error) {
	return f.claims, nil
}

func authenticatedHandler(handler http.Handler, userID int64, role string) http.Handler {
	parser := &fakeTokenParser{
		claims: &auth.Claims{
			UserID: userID,
			Role:   role,
		},
	}

	return auth.Middleware(parser)(handler)
}

func TestHandler_Create(t *testing.T) {
	service := &fakeOrderService{
		createOrder: &Order{
			ID:              100,
			UserID:          10,
			Status:          StatusNew,
			TotalPrice:      159700,
			DeliveryAddress: "Test street 1",
			Items: []OrderItem{
				{
					ProductID:     1,
					NameSnapshot:  "Pepperoni",
					PriceSnapshot: 59900,
					Quantity:      2,
				},
			},
		},
	}

	handler := NewHandler(service)

	body := []byte(`
	{
		"delivery_address": "Test street 1",
		"items": [
			{
				"product_id": 1,
				"quantity": 2
			}
		]
	}
	`)

	req := httptest.NewRequest(
		http.MethodPost,
		"/",
		bytes.NewReader(body),
	)

	rec := httptest.NewRecorder()

	req.Header.Set(
		"Authorization",
		"Bearer test-token",
	)
	protected := authenticatedHandler(
		handler.Routes(),
		10,
		"user",
	)

	protected.ServeHTTP(
		rec,
		req,
	)

	if rec.Code != http.StatusCreated {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusCreated,
			rec.Code,
		)
	}

	if service.createInput == nil {
		t.Fatal(
			"expected CreateOrder to be passed to service",
		)
	}

	if service.createInput.UserID != 10 {
		t.Errorf(
			"expected user ID %d, got %d",
			10,
			service.createInput.UserID,
		)
	}

	if service.createInput.DeliveryAddress != "Test street 1" {
		t.Errorf(
			"expected address %q, got %q",
			"Test street 1",
			service.createInput.DeliveryAddress,
		)
	}

	if len(service.createInput.Items) != 1 {
		t.Fatalf(
			"expected %d item, got %d",
			1,
			len(service.createInput.Items),
		)
	}

	if service.createInput.Items[0].ProductID != 1 {
		t.Errorf(
			"expected product ID %d, got %d",
			1,
			service.createInput.Items[0].ProductID,
		)
	}

	if service.createInput.Items[0].Quantity != 2 {
		t.Errorf(
			"expected quantity %d, got %d",
			2,
			service.createInput.Items[0].Quantity,
		)
	}

	var response Order

	err := json.NewDecoder(rec.Body).Decode(&response)
	if err != nil {
		t.Fatalf(
			"failed to decode response: %v",
			err,
		)
	}

	if response.ID != 100 {
		t.Errorf(
			"expected order ID %d, got %d",
			100,
			response.ID,
		)
	}

	if response.Status != StatusNew {
		t.Errorf(
			"expected status %q, got %q",
			StatusNew,
			response.Status,
		)
	}

	if response.TotalPrice != 159700 {
		t.Errorf(
			"expected total price %d, got %d",
			159700,
			response.TotalPrice,
		)
	}
}

func TestHandler_Create_InvalidJSON(t *testing.T) {
	service := &fakeOrderService{}

	handler := NewHandler(service)

	req := httptest.NewRequest(
		http.MethodPost,
		"/",
		bytes.NewBufferString(`{"delivery_address":`),
	)

	rec := httptest.NewRecorder()

	handler.Routes().ServeHTTP(
		rec,
		req,
	)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusBadRequest,
			rec.Code,
		)
	}

	if service.createInput != nil {
		t.Fatal(
			"service must not be called for invalid JSON",
		)
	}
}

func TestHandler_Create_ValidationError(t *testing.T) {
	service := &fakeOrderService{
		createErr: ErrOrderValidation,
	}

	handler := NewHandler(service)

	body := []byte(`
	{
		"delivery_address": "",
		"items": []
	}
	`)

	req := httptest.NewRequest(
		http.MethodPost,
		"/",
		bytes.NewReader(body),
	)

	req.Header.Set(
		"Authorization",
		"Bearer test-token",
	)

	rec := httptest.NewRecorder()

	protected := authenticatedHandler(
		handler.Routes(),
		10,
		"user",
	)

	protected.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusBadRequest,
			rec.Code,
		)
	}
}

func TestHandler_Create_InternalError(t *testing.T) {
	service := &fakeOrderService{
		createErr: errors.New("database unavailable"),
	}

	handler := NewHandler(service)

	body := []byte(`
	{
		"delivery_address": "Test street 1",
		"items": [
			{
				"product_id": 1,
				"quantity": 1
			}
		]
	}
	`)

	req := httptest.NewRequest(
		http.MethodPost,
		"/",
		bytes.NewReader(body),
	)

	req.Header.Set(
		"Authorization",
		"Bearer test-token",
	)

	rec := httptest.NewRecorder()

	protected := authenticatedHandler(
		handler.Routes(),
		10,
		"user",
	)

	protected.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusInternalServerError,
			rec.Code,
		)
	}
}

func TestHandler_GetByID(t *testing.T) {
	service := &fakeOrderService{
		getOrder: &Order{
			ID:              10,
			UserID:          5,
			Status:          StatusNew,
			TotalPrice:      74900,
			DeliveryAddress: "Test street 10",
		},
	}

	handler := NewHandler(service)

	req := httptest.NewRequest(
		http.MethodGet,
		"/10",
		nil,
	)

	req.Header.Set(
		"Authorization",
		"Bearer test-token",
	)

	rec := httptest.NewRecorder()

	protected := authenticatedHandler(
		handler.Routes(),
		5,
		"user",
	)

	protected.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusOK,
			rec.Code,
		)
	}

	if service.getID != 10 {
		t.Errorf(
			"expected order ID %d, got %d",
			10,
			service.getID,
		)
	}
}

func TestHandler_GetByID_OtherUserOrder(t *testing.T) {
	service := &fakeOrderService{
		getOrder: &Order{
			ID:     10,
			UserID: 99,
			Status: StatusNew,
		},
	}

	handler := NewHandler(service)

	req := httptest.NewRequest(
		http.MethodGet,
		"/10",
		nil,
	)

	req.Header.Set(
		"Authorization",
		"Bearer test-token",
	)

	rec := httptest.NewRecorder()

	protected := authenticatedHandler(
		handler.Routes(),
		5,
		"user",
	)

	protected.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusNotFound,
			rec.Code,
		)
	}
}

func TestHandler_GetByID_InvalidID(t *testing.T) {
	service := &fakeOrderService{}
	handler := NewHandler(service)

	req := httptest.NewRequest(
		http.MethodGet,
		"/abc",
		nil,
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

	if service.getID != 0 {
		t.Fatal("service must not be called for invalid ID")
	}
}

func TestHandler_GetByID_NotFound(t *testing.T) {
	service := &fakeOrderService{
		getErr: ErrOrderNotFound,
	}

	handler := NewHandler(service)

	req := httptest.NewRequest(
		http.MethodGet,
		"/999",
		nil,
	)

	rec := httptest.NewRecorder()

	req.Header.Set(
		"Authorization",
		"Bearer test-token",
	)

	protected := authenticatedHandler(
		handler.Routes(),
		10,
		"user",
	)

	protected.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusNotFound,
			rec.Code,
		)
	}
}

func TestHandler_GetByID_InternalError(t *testing.T) {
	service := &fakeOrderService{
		getErr: errors.New("database unavailable"),
	}

	handler := NewHandler(service)

	req := httptest.NewRequest(
		http.MethodGet,
		"/10",
		nil,
	)

	rec := httptest.NewRecorder()

	req.Header.Set(
		"Authorization",
		"Bearer test-token",
	)

	protected := authenticatedHandler(
		handler.Routes(),
		10,
		"user",
	)

	protected.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusInternalServerError,
			rec.Code,
		)
	}
}

func TestHandler_ListByUser(t *testing.T) {
	service := &fakeOrderService{
		listOrders: []Order{
			{
				ID:         2,
				UserID:     10,
				Status:     StatusNew,
				TotalPrice: 59900,
			},
			{
				ID:         1,
				UserID:     10,
				Status:     StatusCompleted,
				TotalPrice: 39900,
			},
		},
	}

	handler := NewHandler(service)

	req := httptest.NewRequest(
		http.MethodGet,
		"/?limit=50&offset=5",
		nil,
	)

	req.Header.Set(
		"Authorization",
		"Bearer test-token",
	)

	rec := httptest.NewRecorder()

	protected := authenticatedHandler(
		handler.Routes(),
		10,
		"user",
	)

	protected.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusOK,
			rec.Code,
		)
	}

	if service.listUserID != 10 {
		t.Errorf(
			"expected user ID %d, got %d",
			10,
			service.listUserID,
		)
	}

	if service.listLimit != 50 {
		t.Errorf(
			"expected limit %d, got %d",
			50,
			service.listLimit,
		)
	}

	if service.listOffset != 5 {
		t.Errorf(
			"expected offset %d, got %d",
			5,
			service.listOffset,
		)
	}

	var response []Order

	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf(
			"failed to decode response: %v",
			err,
		)
	}

	if len(response) != 2 {
		t.Fatalf(
			"expected %d orders, got %d",
			2,
			len(response),
		)
	}
}

func TestHandler_ListByUser_Unauthorized(t *testing.T) {
	service := &fakeOrderService{}
	handler := NewHandler(service)

	req := httptest.NewRequest(
		http.MethodGet,
		"/",
		nil,
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

func TestHandler_ListByUser_InvalidLimit(t *testing.T) {
	service := &fakeOrderService{}
	handler := NewHandler(service)

	req := httptest.NewRequest(
		http.MethodGet,
		"/?limit=abc",
		nil,
	)

	req.Header.Set(
		"Authorization",
		"Bearer test-token",
	)

	rec := httptest.NewRecorder()

	protected := authenticatedHandler(
		handler.Routes(),
		10,
		"user",
	)

	protected.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusBadRequest,
			rec.Code,
		)
	}
}

func TestHandler_ListByUser_InvalidOffset(t *testing.T) {
	service := &fakeOrderService{}
	handler := NewHandler(service)

	req := httptest.NewRequest(
		http.MethodGet,
		"/?offset=abc",
		nil,
	)

	req.Header.Set(
		"Authorization",
		"Bearer test-token",
	)

	rec := httptest.NewRecorder()

	protected := authenticatedHandler(
		handler.Routes(),
		10,
		"user",
	)

	protected.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusBadRequest,
			rec.Code,
		)
	}
}

func TestHandler_ListByUser_InternalError(t *testing.T) {
	service := &fakeOrderService{
		listErr: errors.New("database unavailable"),
	}

	handler := NewHandler(service)

	req := httptest.NewRequest(
		http.MethodGet,
		"/",
		nil,
	)

	req.Header.Set(
		"Authorization",
		"Bearer test-token",
	)

	rec := httptest.NewRecorder()

	protected := authenticatedHandler(
		handler.Routes(),
		10,
		"user",
	)

	protected.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusInternalServerError,
			rec.Code,
		)
	}
}

func TestHandler_UpdateStatus(t *testing.T) {
	service := &fakeOrderService{}

	handler := NewHandler(service)

	body := []byte(`{
		"status": "confirmed"
	}`)

	req := httptest.NewRequest(
		http.MethodPatch,
		"/10/status",
		bytes.NewReader(body),
	)

	req.Header.Set(
		"Authorization",
		"Bearer test-token",
	)

	rec := httptest.NewRecorder()

	protected := authenticatedHandler(
		handler.Routes(),
		10,
		"admin",
	)

	protected.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusNoContent,
			rec.Code,
		)
	}

	if service.updateID != 10 {
		t.Errorf(
			"expected order ID %d, got %d",
			10,
			service.updateID,
		)
	}

	if service.updateStatus != StatusConfirmed {
		t.Errorf(
			"expected status %q, got %q",
			StatusConfirmed,
			service.updateStatus,
		)
	}
}

func TestHandler_UpdateStatus_InvalidID(t *testing.T) {
	service := &fakeOrderService{}

	handler := NewHandler(service)

	body := []byte(`{
		"status": "confirmed"
	}`)

	req := httptest.NewRequest(
		http.MethodPatch,
		"/abc/status",
		bytes.NewReader(body),
	)

	req.Header.Set(
		"Authorization",
		"Bearer test-token",
	)

	rec := httptest.NewRecorder()

	protected := authenticatedHandler(
		handler.Routes(),
		10,
		"admin",
	)

	protected.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusBadRequest,
			rec.Code,
		)
	}
}

func TestHandler_UpdateStatus_InvalidJSON(t *testing.T) {
	service := &fakeOrderService{}

	handler := NewHandler(service)

	req := httptest.NewRequest(
		http.MethodPatch,
		"/10/status",
		bytes.NewBufferString(`{"status":`),
	)

	req.Header.Set(
		"Authorization",
		"Bearer test-token",
	)

	rec := httptest.NewRecorder()

	protected := authenticatedHandler(
		handler.Routes(),
		10,
		"admin",
	)

	protected.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusBadRequest,
			rec.Code,
		)
	}
}

func TestHandler_UpdateStatus_ValidationError(t *testing.T) {
	service := &fakeOrderService{
		updateErr: ErrOrderValidation,
	}

	handler := NewHandler(service)

	body := []byte(`{
		"status": "completed"
	}`)

	req := httptest.NewRequest(
		http.MethodPatch,
		"/10/status",
		bytes.NewReader(body),
	)

	req.Header.Set(
		"Authorization",
		"Bearer test-token",
	)

	rec := httptest.NewRecorder()

	protected := authenticatedHandler(
		handler.Routes(),
		10,
		"admin",
	)

	protected.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusBadRequest,
			rec.Code,
		)
	}
}

func TestHandler_UpdateStatus_NotFound(t *testing.T) {
	service := &fakeOrderService{
		updateErr: ErrOrderNotFound,
	}

	handler := NewHandler(service)

	body := []byte(`{
		"status": "confirmed"
	}`)

	req := httptest.NewRequest(
		http.MethodPatch,
		"/999/status",
		bytes.NewReader(body),
	)

	req.Header.Set(
		"Authorization",
		"Bearer test-token",
	)

	rec := httptest.NewRecorder()

	protected := authenticatedHandler(
		handler.Routes(),
		10,
		"admin",
	)

	protected.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusNotFound,
			rec.Code,
		)
	}
}

func TestHandler_UpdateStatus_InternalError(t *testing.T) {
	service := &fakeOrderService{
		updateErr: errors.New("database unavailable"),
	}

	handler := NewHandler(service)

	body := []byte(`{
		"status": "confirmed"
	}`)

	req := httptest.NewRequest(
		http.MethodPatch,
		"/10/status",
		bytes.NewReader(body),
	)

	req.Header.Set(
		"Authorization",
		"Bearer test-token",
	)

	rec := httptest.NewRecorder()

	protected := authenticatedHandler(
		handler.Routes(),
		10,
		"admin",
	)

	protected.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusInternalServerError,
			rec.Code,
		)
	}
}

func TestHandler_UpdateStatus_ForbiddenForUser(t *testing.T) {
	service := &fakeOrderService{}

	handler := NewHandler(service)

	body := []byte(`{
		"status": "confirmed"
	}`)

	req := httptest.NewRequest(
		http.MethodPatch,
		"/10/status",
		bytes.NewReader(body),
	)

	req.Header.Set(
		"Authorization",
		"Bearer test-token",
	)

	rec := httptest.NewRecorder()

	protected := authenticatedHandler(
		handler.Routes(),
		10,
		"user",
	)

	protected.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusForbidden,
			rec.Code,
		)
	}

	if service.updateID != 0 {
		t.Fatal(
			"service must not be called for non-admin user",
		)
	}
}

func TestHandler_UpdateStatus_Unauthorized(t *testing.T) {
	service := &fakeOrderService{}
	handler := NewHandler(service)

	body := []byte(`{
		"status": "confirmed"
	}`)

	req := httptest.NewRequest(
		http.MethodPatch,
		"/10/status",
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

func TestHandler_ListAll_DefaultPagination(t *testing.T) {
	service := &fakeOrderService{
		listAllOrders: []*Order{
			{
				ID:     1,
				UserID: 10,
				Status: "new",
			},
			{
				ID:     2,
				UserID: 20,
				Status: "confirmed",
			},
		},
	}

	handler := NewHandler(service)

	request := httptest.NewRequest(
		http.MethodGet,
		"/admin/orders",
		nil,
	)

	recorder := httptest.NewRecorder()

	handler.ListAll(
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

	if service.listAllLimit != 20 {
		t.Errorf(
			"expected limit %d, got %d",
			20,
			service.listAllLimit,
		)
	}

	if service.listAllOffset != 0 {
		t.Errorf(
			"expected offset %d, got %d",
			0,
			service.listAllOffset,
		)
	}

	var response []*Order

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
			"expected %d orders, got %d",
			2,
			len(response),
		)
	}
}

func TestHandler_ListAll_CustomPagination(t *testing.T) {
	service := &fakeOrderService{
		listAllOrders: []*Order{},
	}

	handler := NewHandler(service)

	request := httptest.NewRequest(
		http.MethodGet,
		"/admin/orders?limit=50&offset=25",
		nil,
	)

	recorder := httptest.NewRecorder()

	handler.ListAll(
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

	if service.listAllLimit != 50 {
		t.Errorf(
			"expected limit %d, got %d",
			50,
			service.listAllLimit,
		)
	}

	if service.listAllOffset != 25 {
		t.Errorf(
			"expected offset %d, got %d",
			25,
			service.listAllOffset,
		)
	}
}

func TestHandler_ListAll_InvalidPagination(t *testing.T) {
	tests := []struct {
		name string
		url  string
	}{
		{
			name: "invalid limit",
			url:  "/admin/orders?limit=abc",
		},
		{
			name: "invalid offset",
			url:  "/admin/orders?offset=abc",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := &fakeOrderService{}
			handler := NewHandler(service)

			request := httptest.NewRequest(
				http.MethodGet,
				tt.url,
				nil,
			)

			recorder := httptest.NewRecorder()

			handler.ListAll(
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
		})
	}
}
func TestHandler_ListAll_InternalError(t *testing.T) {
	service := &fakeOrderService{
		listAllErr: errors.New(
			"database unavailable",
		),
	}

	handler := NewHandler(service)

	request := httptest.NewRequest(
		http.MethodGet,
		"/admin/orders",
		nil,
	)

	recorder := httptest.NewRecorder()

	handler.ListAll(
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

func TestHandler_ListAll_ValidationError(t *testing.T) {
	service := &fakeOrderService{
		listAllErr: ErrOrderValidation,
	}

	handler := NewHandler(service)

	request := httptest.NewRequest(
		http.MethodGet,
		"/admin/orders?limit=101",
		nil,
	)

	recorder := httptest.NewRecorder()

	handler.ListAll(
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

func TestHandler_ListAll_ServiceValidationError(t *testing.T) {
	tests := []struct {
		name string
		url  string
	}{
		{
			name: "limit too large",
			url:  "/admin/orders?limit=101",
		},
		{
			name: "negative offset",
			url:  "/admin/orders?offset=-1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := &fakeOrderService{
				listAllErr: ErrOrderValidation,
			}

			handler := NewHandler(service)

			request := httptest.NewRequest(
				http.MethodGet,
				tt.url,
				nil,
			)

			recorder := httptest.NewRecorder()

			handler.ListAll(
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
		})
	}
}
