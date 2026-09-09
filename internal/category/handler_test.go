package category

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

type fakeService struct {
	categories []Category
	err        error
}

func (f *fakeService) GetByID(ctx context.Context, id int64) (*Category, error) {
	return nil, nil
}

func (f *fakeService) List(ctx context.Context) ([]Category, error) {
	return f.categories, f.err
}

func (f *fakeService) Create(ctx context.Context, name, slug string) (*Category, error) {
	return nil, nil
}

func (f *fakeService) Update(ctx context.Context, category *Category) error {
	return nil
}

func (f *fakeService) Delete(ctx context.Context, id int64) error {
	return nil
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

	handler := NewHandler(service)

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
	expectedErr := errors.New("service failure")
	expectedBody := "internal server error\n"

	service := &fakeService{
		err: expectedErr,
	}

	handler := NewHandler(service)

	req := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/categories",
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

	if rec.Body.String() != expectedBody {
		t.Errorf(
			"expected body %q, got %q",
			expectedBody,
			rec.Body.String(),
		)
	}
}
