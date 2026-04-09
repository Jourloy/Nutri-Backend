package blog

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/jourloy/somivyn/internal/auth"
	"github.com/jourloy/somivyn/internal/user"
)

type prepareServiceStub struct {
	Service
	out *PrepareArticleResponse
	err error
}

func (s *prepareServiceStub) PrepareArticle(ctx context.Context, req PrepareArticleRequest) (*PrepareArticleResponse, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.out, nil
}

func TestPrepareArticle_Unauthorized(t *testing.T) {
	r := chi.NewRouter()
	c := &Controller{service: &prepareServiceStub{}}
	c.RegisterRoutes(r)

	body := bytes.NewBufferString(`{"titleRu":"t","descriptionRu":"d","contentMarkdownRu":"# x"}`)
	req := httptest.NewRequest(http.MethodPost, "/blog/admin/articles/prepare", body)
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected %d, got %d", http.StatusUnauthorized, rr.Code)
	}
}

func TestPrepareArticle_ForbiddenForNonAdmin(t *testing.T) {
	r := chi.NewRouter()
	c := &Controller{service: &prepareServiceStub{}}
	c.RegisterRoutes(r)

	body := bytes.NewBufferString(`{"titleRu":"t","descriptionRu":"d","contentMarkdownRu":"# x"}`)
	req := httptest.NewRequest(http.MethodPost, "/blog/admin/articles/prepare", body)
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(auth.ContextWithUser(req.Context(), user.User{Id: "u1", IsAdmin: false}))

	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected %d, got %d", http.StatusForbidden, rr.Code)
	}
}

func TestPrepareArticle_Validation(t *testing.T) {
	r := chi.NewRouter()
	c := &Controller{service: &prepareServiceStub{}}
	c.RegisterRoutes(r)

	body := bytes.NewBufferString(`{"titleRu":"","descriptionRu":"","contentMarkdownRu":""}`)
	req := httptest.NewRequest(http.MethodPost, "/blog/admin/articles/prepare", body)
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(auth.ContextWithUser(req.Context(), user.User{Id: "admin", IsAdmin: true}))

	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected %d, got %d", http.StatusBadRequest, rr.Code)
	}
}

func TestPrepareArticle_Success(t *testing.T) {
	r := chi.NewRouter()
	c := &Controller{service: &prepareServiceStub{out: &PrepareArticleResponse{
		Slug:              "my-slug",
		TitleRu:           "Заголовок",
		TitleEn:           "Title",
		ContentRu:         "<p>RU</p>",
		ContentEn:         "<p>EN</p>",
		PreviewTextRu:     "prev ru",
		PreviewTextEn:     "prev en",
		MetaDescriptionRu: "meta ru",
		MetaDescriptionEn: "meta en",
	}}}
	c.RegisterRoutes(r)

	body := bytes.NewBufferString(`{"titleRu":"Заголовок","descriptionRu":"Описание","contentMarkdownRu":"# intro"}`)
	req := httptest.NewRequest(http.MethodPost, "/blog/admin/articles/prepare", body)
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(auth.ContextWithUser(req.Context(), user.User{Id: "admin", IsAdmin: true}))

	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected %d, got %d: %s", http.StatusOK, rr.Code, rr.Body.String())
	}

	var out PrepareArticleResponse
	if err := json.NewDecoder(rr.Body).Decode(&out); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if out.Slug != "my-slug" || out.ContentEn == "" {
		t.Fatalf("unexpected response: %#v", out)
	}
}
