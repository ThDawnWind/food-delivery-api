package category

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
)

type fakeService struct {
	category   *Category
	categories []Category

	createdName string
	createdSlug string

	updatedCategory *Category

	err error
}

func (f *fakeService) GetByID(ctx context.Context, id int64) (*Category, error) {
	return f.category, f.err
}

func (f *fakeService) List(ctx context.Context) ([]Category, error) {
	return f.categories, f.err
}

func (f *fakeService) Create(ctx context.Context, name, slug string) (*Category, error) {
	f.createdName = name
	f.createdSlug = slug

	return f.category, f.err
}

func (f *fakeService) Update(ctx context.Context, category *Category) error {
	f.updatedCategory = category
	return f.err
}

func (f *fakeService) Delete(ctx context.Context, id int64) error {
	return f.err
}

func assertErrorResponse(t *testing.T, rec *httptest.ResponseRecorder, expectedStatus int, expectedMessage string) {
	t.Helper()

	if rec.Code != expectedStatus {
		t.Fatalf("expected status %d, got %d", expectedStatus, rec.Code)
	}

	if contentType := rec.Header().Get("Content-Type"); contentType != "application/json" {
		t.Fatalf("expected Content-Type %q, got %q", "application/json", contentType)
	}

	var response struct {
		Error string `json:"error"`
	}

	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response.Error != expectedMessage {
		t.Fatalf("expected error %q, got %q", expectedMessage, response.Error)
	}
}

func TestHandler_List(t *testing.T) {
	service := &fakeService{
		categories: []Category{
			{
				ID:   1,
				Name: "Pizza",
				Slug: "pizza",
			},
			{
				ID:   2,
				Name: "Sushi",
				Slug: "sushi",
			},
		},
	}

	handler := NewHandler(
		service,
		newTestLogger(),
	)

	req := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/categories",
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

	var got []Category

	err := json.NewDecoder(rec.Body).Decode(&got)
	if err != nil {
		t.Fatalf("failed to decode response body: %v", err)
	}

	if len(got) != len(service.categories) {
		t.Fatalf(
			"expected %d categories, got %d",
			len(service.categories),
			len(got),
		)
	}

	for i := range service.categories {
		if got[i].ID != service.categories[i].ID {
			t.Errorf(
				"category %d: expected ID %d, got %d",
				i,
				service.categories[i].ID,
				got[i].ID,
			)
		}

		if got[i].Name != service.categories[i].Name {
			t.Errorf(
				"category %d: expected name %q, got %q",
				i,
				service.categories[i].Name,
				got[i].Name,
			)
		}

		if got[i].Slug != service.categories[i].Slug {
			t.Errorf(
				"category %d: expected slug %q, got %q",
				i,
				service.categories[i].Slug,
				got[i].Slug,
			)
		}
	}
}

func TestHandler_List_ServiceError(t *testing.T) {
	service := &fakeService{
		err: errors.New("service failure"),
	}

	handler := NewHandler(
		service,
		newTestLogger(),
	)

	req := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/categories",
		nil,
	)

	rec := httptest.NewRecorder()

	handler.List(rec, req)

	assertErrorResponse(t, rec, http.StatusInternalServerError, "internal server error")
}
func TestHandler_GetByID(t *testing.T) {
	service := &fakeService{
		category: &Category{
			ID:   1,
			Name: "Pizza",
			Slug: "pizza",
		},
	}

	handler := NewHandler(
		service,
		newTestLogger(),
	)

	router := chi.NewRouter()
	router.Get("/api/v1/categories/{id}", handler.GetByID)

	req := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/categories/1",
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

	var got Category

	err := json.NewDecoder(rec.Body).Decode(&got)
	if err != nil {
		t.Fatalf("failed to decode response body: %v", err)
	}

	if got.ID != service.category.ID {
		t.Errorf(
			"expected ID %d, got %d",
			service.category.ID,
			got.ID,
		)
	}

	if got.Name != service.category.Name {
		t.Errorf(
			"expected name %q, got %q",
			service.category.Name,
			got.Name,
		)
	}

	if got.Slug != service.category.Slug {
		t.Errorf(
			"expected slug %q, got %q",
			service.category.Slug,
			got.Slug,
		)
	}
}

func TestHandler_GetByID_InvalidID(t *testing.T) {
	service := &fakeService{}
	handler := NewHandler(
		service,
		newTestLogger(),
	)

	router := chi.NewRouter()
	router.Get("/api/v1/categories/{id}", handler.GetByID)

	req := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/categories/abc",
		nil,
	)

	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assertErrorResponse(t, rec, http.StatusBadRequest, "invalid category id")
}
func TestHandler_GetByID_NotFound(t *testing.T) {
	service := &fakeService{
		err: ErrCategoryNotFound,
	}

	handler := NewHandler(
		service,
		newTestLogger(),
	)

	router := chi.NewRouter()
	router.Get("/api/v1/categories/{id}", handler.GetByID)

	req := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/categories/999",
		nil,
	)

	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assertErrorResponse(t, rec, http.StatusNotFound, "category not found")
}
func TestHandler_GetByID_ServiceError(t *testing.T) {
	service := &fakeService{
		err: errors.New("service failure"),
	}

	handler := NewHandler(
		service,
		newTestLogger(),
	)

	router := chi.NewRouter()
	router.Get("/api/v1/categories/{id}", handler.GetByID)

	req := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/categories/1",
		nil,
	)

	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assertErrorResponse(t, rec, http.StatusInternalServerError, "internal server error")
}
func TestHandler_Create(t *testing.T) {
	service := &fakeService{
		category: &Category{
			ID:   1,
			Name: "Pizza",
			Slug: "pizza",
		},
	}

	handler := NewHandler(
		service,
		newTestLogger(),
	)

	body := `{
		"name": "Pizza",
		"slug": "pizza"
	}`

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/categories",
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

	if service.createdName != "Pizza" {
		t.Errorf(
			"expected name %q, got %q",
			"Pizza",
			service.createdName,
		)
	}

	if service.createdSlug != "pizza" {
		t.Errorf(
			"expected slug %q, got %q",
			"pizza",
			service.createdSlug,
		)
	}

	contentType := rec.Header().Get("Content-Type")

	if contentType != "application/json" {
		t.Fatalf(
			"expected Content-Type %q, got %q",
			"application/json",
			contentType,
		)
	}

	var got Category

	err := json.NewDecoder(rec.Body).Decode(&got)
	if err != nil {
		t.Fatalf("failed to decode response body: %v", err)
	}

	if got.ID != service.category.ID {
		t.Errorf(
			"expected ID %d, got %d",
			service.category.ID,
			got.ID,
		)
	}

	if got.Name != service.category.Name {
		t.Errorf(
			"expected name %q, got %q",
			service.category.Name,
			got.Name,
		)
	}

	if got.Slug != service.category.Slug {
		t.Errorf(
			"expected slug %q, got %q",
			service.category.Slug,
			got.Slug,
		)
	}
}

func TestHandler_Create_InvalidJSON(t *testing.T) {
	service := &fakeService{}
	handler := NewHandler(
		service,
		newTestLogger(),
	)

	body := `{
		"name": "Pizza",
		"slug":
	}`

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/categories",
		strings.NewReader(body),
	)

	rec := httptest.NewRecorder()

	handler.Create(rec, req)

	assertErrorResponse(t, rec, http.StatusBadRequest, "invalid request body")
}
func TestHandler_Create_ValidationError(t *testing.T) {
	service := &fakeService{
		err: ErrCategoryValidation,
	}

	handler := NewHandler(
		service,
		newTestLogger(),
	)

	body := `{
		"name": "",
		"slug": "pizza"
	}`

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/categories",
		strings.NewReader(body),
	)

	rec := httptest.NewRecorder()

	handler.Create(rec, req)

	assertErrorResponse(t, rec, http.StatusBadRequest, "invalid category data")
}
func TestHandler_Create_ServiceError(t *testing.T) {
	service := &fakeService{
		err: errors.New("service failure"),
	}

	handler := NewHandler(
		service,
		newTestLogger(),
	)

	body := `{
		"name": "Pizza",
		"slug": "pizza"
	}`

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/categories",
		strings.NewReader(body),
	)

	rec := httptest.NewRecorder()

	handler.Create(rec, req)

	assertErrorResponse(t, rec, http.StatusInternalServerError, "internal server error")
}
func TestHandler_Update(t *testing.T) {
	service := &fakeService{}
	handler := NewHandler(
		service,
		newTestLogger(),
	)

	router := chi.NewRouter()
	router.Put("/api/v1/categories/{id}", handler.Update)

	body := `{
		"name": "Italian Pizza",
		"slug": "italian-pizza"
	}`

	req := httptest.NewRequest(
		http.MethodPut,
		"/api/v1/categories/1",
		strings.NewReader(body),
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

	if service.updatedCategory == nil {
		t.Fatal("expected Service.Update to be called")
	}

	if service.updatedCategory.ID != 1 {
		t.Errorf(
			"expected ID %d, got %d",
			1,
			service.updatedCategory.ID,
		)
	}

	if service.updatedCategory.Name != "Italian Pizza" {
		t.Errorf(
			"expected name %q, got %q",
			"Italian Pizza",
			service.updatedCategory.Name,
		)
	}

	if service.updatedCategory.Slug != "italian-pizza" {
		t.Errorf(
			"expected slug %q, got %q",
			"italian-pizza",
			service.updatedCategory.Slug,
		)
	}
}

func TestHandler_Update_InvalidID(t *testing.T) {
	service := &fakeService{}
	handler := NewHandler(
		service,
		newTestLogger(),
	)

	router := chi.NewRouter()
	router.Put("/api/v1/categories/{id}", handler.Update)

	body := `{
		"name": "Italian Pizza",
		"slug": "italian-pizza"
	}`

	req := httptest.NewRequest(
		http.MethodPut,
		"/api/v1/categories/abc",
		strings.NewReader(body),
	)

	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assertErrorResponse(t, rec, http.StatusBadRequest, "invalid category id")
}
func TestHandler_Update_InvalidJSON(t *testing.T) {
	service := &fakeService{}
	handler := NewHandler(
		service,
		newTestLogger(),
	)

	router := chi.NewRouter()
	router.Put("/api/v1/categories/{id}", handler.Update)

	body := `{
		"name": "Italian Pizza",
		"slug":
	}`

	req := httptest.NewRequest(
		http.MethodPut,
		"/api/v1/categories/1",
		strings.NewReader(body),
	)

	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assertErrorResponse(t, rec, http.StatusBadRequest, "invalid request body")
}
func TestHandler_Update_NotFound(t *testing.T) {
	service := &fakeService{
		err: ErrCategoryNotFound,
	}

	handler := NewHandler(
		service,
		newTestLogger(),
	)

	router := chi.NewRouter()
	router.Put("/api/v1/categories/{id}", handler.Update)

	body := `{
		"name": "Italian Pizza",
		"slug": "italian-pizza"
	}`

	req := httptest.NewRequest(
		http.MethodPut,
		"/api/v1/categories/999",
		strings.NewReader(body),
	)

	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assertErrorResponse(t, rec, http.StatusNotFound, "category not found")
}
func TestHandler_Update_ServiceError(t *testing.T) {
	service := &fakeService{
		err: errors.New("service failure"),
	}

	handler := NewHandler(
		service,
		newTestLogger(),
	)

	router := chi.NewRouter()
	router.Put("/api/v1/categories/{id}", handler.Update)

	body := `{
		"name": "Italian Pizza",
		"slug": "italian-pizza"
	}`

	req := httptest.NewRequest(
		http.MethodPut,
		"/api/v1/categories/1",
		strings.NewReader(body),
	)

	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assertErrorResponse(t, rec, http.StatusInternalServerError, "internal server error")
}
func TestHandler_Delete(t *testing.T) {
	service := &fakeService{}
	handler := NewHandler(
		service,
		newTestLogger(),
	)

	router := chi.NewRouter()
	router.Delete("/api/v1/categories/{id}", handler.Delete)

	req := httptest.NewRequest(
		http.MethodDelete,
		"/api/v1/categories/1",
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
}

func TestHandler_Delete_InvalidID(t *testing.T) {
	service := &fakeService{}
	handler := NewHandler(
		service,
		newTestLogger(),
	)

	router := chi.NewRouter()
	router.Delete("/api/v1/categories/{id}", handler.Delete)

	req := httptest.NewRequest(
		http.MethodDelete,
		"/api/v1/categories/abc",
		nil,
	)

	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assertErrorResponse(t, rec, http.StatusBadRequest, "invalid category id")
}
func TestHandler_Delete_NotFound(t *testing.T) {
	service := &fakeService{
		err: ErrCategoryNotFound,
	}

	handler := NewHandler(
		service,
		newTestLogger(),
	)

	router := chi.NewRouter()
	router.Delete("/api/v1/categories/{id}", handler.Delete)

	req := httptest.NewRequest(
		http.MethodDelete,
		"/api/v1/categories/999",
		nil,
	)

	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assertErrorResponse(t, rec, http.StatusNotFound, "category not found")
}
func TestHandler_Delete_ServiceError(t *testing.T) {
	service := &fakeService{
		err: errors.New("service failure"),
	}

	handler := NewHandler(
		service,
		newTestLogger(),
	)

	router := chi.NewRouter()
	router.Delete("/api/v1/categories/{id}", handler.Delete)

	req := httptest.NewRequest(
		http.MethodDelete,
		"/api/v1/categories/1",
		nil,
	)

	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assertErrorResponse(t, rec, http.StatusInternalServerError, "internal server error")
}

func newTestLogger() *slog.Logger {
	return slog.New(
		slog.NewJSONHandler(
			io.Discard,
			nil,
		),
	)
}
