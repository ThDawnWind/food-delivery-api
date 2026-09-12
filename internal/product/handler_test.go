package product

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
)

type fakeService struct {
	product       *Product
	products      []Product
	deactivatedID int64
	err           error
}

func (f *fakeService) GetByID(ctx context.Context, id int64) (*Product, error) {
	return f.product, f.err
}

func (f *fakeService) List(ctx context.Context) ([]Product, error) {
	return f.products, f.err
}

func (f *fakeService) Create(ctx context.Context, product *Product) (*Product, error) {
	if f.err != nil {
		return nil, f.err
	}

	f.product = product

	return product, nil
}

func (f *fakeService) Update(ctx context.Context, product *Product) (*Product, error) {
	if f.err != nil {
		return nil, f.err
	}

	f.product = product

	return product, nil
}

func (f *fakeService) Deactivate(ctx context.Context, id int64) error {
	if f.err != nil {
		return f.err
	}

	f.deactivatedID = id

	return nil

}

func TestHandler_GetByID(t *testing.T) {
	service := &fakeService{
		product: &Product{
			ID:         1,
			Name:       "Pepperoni",
			Price:      59900,
			Weight:     450,
			CategoryID: 1,
			IsActive:   true,
			Images:     []ProductImage{},
		},
	}

	handler := NewHandler(service)

	router := chi.NewRouter()
	router.Get("/products/{id}", handler.GetByID)

	req := httptest.NewRequest(
		http.MethodGet,
		"/products/1",
		nil,
	)

	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusOK,
			rec.Code,
		)
	}

	var product Product

	if err := json.NewDecoder(rec.Body).Decode(&product); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if product.ID != 1 {
		t.Errorf(
			"expected product ID %d, got %d",
			1,
			product.ID,
		)
	}

	if product.Name != "Pepperoni" {
		t.Errorf(
			"expected name %q, got %q",
			"Pepperoni",
			product.Name,
		)
	}
}

func TestHandler_GetByID_InvalidID(t *testing.T) {
	service := &fakeService{}
	handler := NewHandler(service)

	router := chi.NewRouter()
	router.Get("/products/{id}", handler.GetByID)

	req := httptest.NewRequest(
		http.MethodGet,
		"/products/abc",
		nil,
	)

	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusBadRequest,
			rec.Code,
		)
	}
}

func TestHandler_GetByID_NotFound(t *testing.T) {
	service := &fakeService{
		err: ErrProductNotFound,
	}

	handler := NewHandler(service)

	router := chi.NewRouter()
	router.Get("/products/{id}", handler.GetByID)

	req := httptest.NewRequest(
		http.MethodGet,
		"/products/999",
		nil,
	)

	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusNotFound,
			rec.Code,
		)
	}
}

func TestHandler_GetByID_ServiceError(t *testing.T) {
	service := &fakeService{
		err: errors.New("service unavailable"),
	}

	handler := NewHandler(service)

	router := chi.NewRouter()
	router.Get("/products/{id}", handler.GetByID)

	req := httptest.NewRequest(
		http.MethodGet,
		"/products/1",
		nil,
	)

	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusInternalServerError,
			rec.Code,
		)
	}
}

func TestHandler_List(t *testing.T) {
	service := &fakeService{
		products: []Product{
			{
				ID:         1,
				Name:       "Pepperoni",
				Price:      59900,
				Weight:     450,
				CategoryID: 1,
				IsActive:   true,
			},
			{
				ID:         2,
				Name:       "Burger",
				Price:      39900,
				Weight:     300,
				CategoryID: 1,
				IsActive:   true,
			},
		},
	}

	handler := NewHandler(service)

	req := httptest.NewRequest(
		http.MethodGet,
		"/products",
		nil,
	)

	rec := httptest.NewRecorder()

	handler.List(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusOK,
			rec.Code,
		)
	}

	var products []Product

	if err := json.NewDecoder(rec.Body).Decode(&products); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(products) != 2 {
		t.Fatalf(
			"expected %d products, got %d",
			2,
			len(products),
		)
	}

	if products[0].Name != "Pepperoni" {
		t.Errorf(
			"expected first product name %q, got %q",
			"Pepperoni",
			products[0].Name,
		)
	}
}

func TestHandler_List_ServiceError(t *testing.T) {
	service := &fakeService{
		err: errors.New("service unavailable"),
	}

	handler := NewHandler(service)

	req := httptest.NewRequest(
		http.MethodGet,
		"/products",
		nil,
	)

	rec := httptest.NewRecorder()

	handler.List(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusInternalServerError,
			rec.Code,
		)
	}
}

func TestHandler_Create(t *testing.T) {
	service := &fakeService{}
	handler := NewHandler(service)

	body := `{
		"name": "Pepperoni",
		"description": "Spicy pizza",
		"price": 59900,
		"weight": 450,
		"category_id": 1,
		"images": [
			{
				"url": "/images/pepperoni.webp",
				"sort_order": 0,
				"is_primary": true
			}
		]
	}`

	req := httptest.NewRequest(
		http.MethodPost,
		"/products",
		strings.NewReader(body),
	)

	rec := httptest.NewRecorder()

	handler.Create(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusCreated,
			rec.Code,
		)
	}

	if service.product == nil {
		t.Fatal("expected product to be passed to service")
	}

	if service.product.Name != "Pepperoni" {
		t.Errorf(
			"expected name %q, got %q",
			"Pepperoni",
			service.product.Name,
		)
	}

	if !service.product.IsActive {
		t.Error("expected product to be active by default")
	}

	if len(service.product.Images) != 1 {
		t.Fatalf(
			"expected %d image, got %d",
			1,
			len(service.product.Images),
		)
	}

	if service.product.Images[0].URL != "/images/pepperoni.webp" {
		t.Errorf(
			"expected image URL %q, got %q",
			"/images/pepperoni.webp",
			service.product.Images[0].URL,
		)
	}
}

func TestHandler_Create_InvalidJSON(t *testing.T) {
	service := &fakeService{}
	handler := NewHandler(service)

	req := httptest.NewRequest(
		http.MethodPost,
		"/products",
		strings.NewReader(`{"name":`),
	)

	rec := httptest.NewRecorder()

	handler.Create(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusBadRequest,
			rec.Code,
		)
	}

	if service.product != nil {
		t.Fatal("service must not be called for invalid JSON")
	}
}

func TestHandler_Create_ValidationError(t *testing.T) {
	service := &fakeService{
		err: ErrProductValidation,
	}

	handler := NewHandler(service)

	body := `{
		"name": "",
		"price": 0,
		"weight": 0,
		"category_id": 0
	}`

	req := httptest.NewRequest(
		http.MethodPost,
		"/products",
		strings.NewReader(body),
	)

	rec := httptest.NewRecorder()

	handler.Create(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusBadRequest,
			rec.Code,
		)
	}
}

func TestHandler_Create_ServiceError(t *testing.T) {
	service := &fakeService{
		err: errors.New("service unavailable"),
	}

	handler := NewHandler(service)

	body := `{
		"name": "Pepperoni",
		"price": 59900,
		"weight": 450,
		"category_id": 1
	}`

	req := httptest.NewRequest(
		http.MethodPost,
		"/products",
		strings.NewReader(body),
	)

	rec := httptest.NewRecorder()

	handler.Create(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusInternalServerError,
			rec.Code,
		)
	}
}

func TestHandler_Update(t *testing.T) {
	service := &fakeService{}
	handler := NewHandler(service)

	router := chi.NewRouter()
	router.Put("/products/{id}", handler.Update)

	body := `{
		"name": "Updated Pepperoni",
		"description": "Updated description",
		"price": 69900,
		"weight": 500,
		"category_id": 1,
		"is_active": true,
		"images": [
			{
				"url": "/images/updated.webp",
				"sort_order": 0,
				"is_primary": true
			}
		]
	}`

	req := httptest.NewRequest(
		http.MethodPut,
		"/products/10",
		strings.NewReader(body),
	)

	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusOK,
			rec.Code,
		)
	}

	if service.product == nil {
		t.Fatal("expected product to be passed to service")
	}

	if service.product.ID != 10 {
		t.Errorf(
			"expected product ID %d, got %d",
			10,
			service.product.ID,
		)
	}

	if service.product.Name != "Updated Pepperoni" {
		t.Errorf(
			"expected name %q, got %q",
			"Updated Pepperoni",
			service.product.Name,
		)
	}

	if len(service.product.Images) != 1 {
		t.Fatalf(
			"expected %d image, got %d",
			1,
			len(service.product.Images),
		)
	}
}

func TestHandler_Update_InvalidID(t *testing.T) {
	service := &fakeService{}
	handler := NewHandler(service)

	router := chi.NewRouter()
	router.Put("/products/{id}", handler.Update)

	req := httptest.NewRequest(
		http.MethodPut,
		"/products/abc",
		strings.NewReader(`{}`),
	)

	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusBadRequest,
			rec.Code,
		)
	}

	if service.product != nil {
		t.Fatal("service must not be called for invalid ID")
	}
}

func TestHandler_Update_InvalidJSON(t *testing.T) {
	service := &fakeService{}
	handler := NewHandler(service)

	router := chi.NewRouter()
	router.Put("/products/{id}", handler.Update)

	req := httptest.NewRequest(
		http.MethodPut,
		"/products/1",
		strings.NewReader(`{"name":`),
	)

	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusBadRequest,
			rec.Code,
		)
	}
}

func TestHandler_Update_ValidationError(t *testing.T) {
	service := &fakeService{
		err: ErrProductValidation,
	}

	handler := NewHandler(service)

	router := chi.NewRouter()
	router.Put("/products/{id}", handler.Update)

	body := `{
		"name": "",
		"price": 0,
		"weight": 0,
		"category_id": 0,
		"is_active": true
	}`

	req := httptest.NewRequest(
		http.MethodPut,
		"/products/1",
		strings.NewReader(body),
	)

	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusBadRequest,
			rec.Code,
		)
	}
}

func TestHandler_Update_NotFound(t *testing.T) {
	service := &fakeService{
		err: ErrProductNotFound,
	}

	handler := NewHandler(service)

	router := chi.NewRouter()
	router.Put("/products/{id}", handler.Update)

	body := `{
		"name": "Pepperoni",
		"price": 59900,
		"weight": 450,
		"category_id": 1,
		"is_active": true
	}`

	req := httptest.NewRequest(
		http.MethodPut,
		"/products/999",
		strings.NewReader(body),
	)

	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusNotFound,
			rec.Code,
		)
	}
}

func TestHandler_Update_ServiceError(t *testing.T) {
	service := &fakeService{
		err: errors.New("service unavailable"),
	}

	handler := NewHandler(service)

	router := chi.NewRouter()
	router.Put("/products/{id}", handler.Update)

	body := `{
		"name": "Pepperoni",
		"price": 59900,
		"weight": 450,
		"category_id": 1,
		"is_active": true
	}`

	req := httptest.NewRequest(
		http.MethodPut,
		"/products/1",
		strings.NewReader(body),
	)

	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusInternalServerError,
			rec.Code,
		)
	}
}

func TestHandler_Delete(t *testing.T) {
	service := &fakeService{}
	handler := NewHandler(service)

	router := chi.NewRouter()
	router.Delete("/products/{id}", handler.Delete)

	req := httptest.NewRequest(
		http.MethodDelete,
		"/products/10",
		nil,
	)

	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusNoContent,
			rec.Code,
		)
	}

	if service.deactivatedID != 10 {
		t.Errorf(
			"expected deactivated ID %d, got %d",
			10,
			service.deactivatedID,
		)
	}
}

func TestHandler_Delete_InvalidID(t *testing.T) {
	service := &fakeService{}
	handler := NewHandler(service)

	router := chi.NewRouter()
	router.Delete("/products/{id}", handler.Delete)

	req := httptest.NewRequest(
		http.MethodDelete,
		"/products/abc",
		nil,
	)

	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusBadRequest,
			rec.Code,
		)
	}

	if service.deactivatedID != 0 {
		t.Fatal("service must not be called for invalid ID")
	}
}

func TestHandler_Delete_NotFound(t *testing.T) {
	service := &fakeService{
		err: ErrProductNotFound,
	}

	handler := NewHandler(service)

	router := chi.NewRouter()
	router.Delete("/products/{id}", handler.Delete)

	req := httptest.NewRequest(
		http.MethodDelete,
		"/products/999",
		nil,
	)

	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusNotFound,
			rec.Code,
		)
	}
}

func TestHandler_Delete_ServiceError(t *testing.T) {
	service := &fakeService{
		err: errors.New("service unavailable"),
	}

	handler := NewHandler(service)

	router := chi.NewRouter()
	router.Delete("/products/{id}", handler.Delete)

	req := httptest.NewRequest(
		http.MethodDelete,
		"/products/1",
		nil,
	)

	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusInternalServerError,
			rec.Code,
		)
	}
}
