package ai

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/jourloy/nutri-backend/internal/auth"
	"github.com/jourloy/nutri-backend/internal/user"
)

type fakeService struct {
	improveFn func(ctx context.Context, userId string, html string) (string, error)
	genFn     func(ctx context.Context, userId string, topic string, description string, provider string) (*GeneratedArticle, error)
}

func (f fakeService) AnalyzeFoodImage(ctx context.Context, userId string, imageData []byte, totalWeight *float64, userPrompt string, language string) (*FoodAnalysisResult, error) {
	return nil, nil
}

func (f fakeService) AnalyzeFoodByText(ctx context.Context, userId string, foodName string, foodDescription string, totalWeight float64, language string) (*FoodAnalysisResult, error) {
	return nil, nil
}

func (f fakeService) CheckUserLimit(ctx context.Context, userId, requestType string) (*LimitCheckResult, error) {
	return &LimitCheckResult{Allowed: true}, nil
}

func (f fakeService) GetUserAnalysisHistory(ctx context.Context, userId string, limit int) ([]AnalysisLog, error) {
	return nil, nil
}

func (f fakeService) ImproveText(ctx context.Context, userId string, html string) (string, error) {
	if f.improveFn != nil {
		return f.improveFn(ctx, userId, html)
	}
	return "<p>ok</p>", nil
}

func (f fakeService) GenerateArticle(ctx context.Context, userId string, topic string, description string, provider string) (*GeneratedArticle, error) {
	if f.genFn != nil {
		return f.genFn(ctx, userId, topic, description, provider)
	}
	return &GeneratedArticle{
		TitleRu:           "RU title",
		TitleEn:           "EN title",
		PreviewTextRu:     "RU preview",
		PreviewTextEn:     "EN preview",
		MetaDescriptionRu: "RU meta",
		MetaDescriptionEn: "EN meta",
		ContentRu:         "<p>RU</p>",
		ContentEn:         "<p>EN</p>",
	}, nil
}

func TestImproveText_Unauthorized(t *testing.T) {
	r := chi.NewRouter()
	c := &Controller{service: fakeService{}}
	c.RegisterRoutes(r)

	req := httptest.NewRequest(http.MethodPost, "/ai/improve-text", strings.NewReader(`{"html":"<p>Hi</p>"}`))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected %d, got %d", http.StatusUnauthorized, rr.Code)
	}
}

func TestImproveText_ForbiddenForNonAdmin(t *testing.T) {
	r := chi.NewRouter()
	c := &Controller{service: fakeService{}}
	c.RegisterRoutes(r)

	req := httptest.NewRequest(http.MethodPost, "/ai/improve-text", strings.NewReader(`{"html":"<p>Hi</p>"}`))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(auth.ContextWithUser(req.Context(), user.User{Id: "u1", IsAdmin: false}))

	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected %d, got %d", http.StatusForbidden, rr.Code)
	}
}

func TestImproveText_SuccessForAdmin(t *testing.T) {
	r := chi.NewRouter()
	c := &Controller{service: fakeService{}}
	c.RegisterRoutes(r)

	req := httptest.NewRequest(http.MethodPost, "/ai/improve-text", strings.NewReader(`{"html":"<p>Hi</p>"}`))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(auth.ContextWithUser(req.Context(), user.User{Id: "u1", IsAdmin: true}))

	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected %d, got %d: %s", http.StatusOK, rr.Code, rr.Body.String())
	}

	var out ImproveTextResponse
	if err := json.NewDecoder(rr.Body).Decode(&out); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if out.HTML == "" {
		t.Fatalf("expected non-empty html")
	}
}

func TestGenerateArticle_Unauthorized(t *testing.T) {
	r := chi.NewRouter()
	c := &Controller{service: fakeService{}}
	c.RegisterRoutes(r)

	req := httptest.NewRequest(http.MethodPost, "/ai/generate-article", strings.NewReader(`{"topic":"T","description":"D"}`))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected %d, got %d", http.StatusUnauthorized, rr.Code)
	}
}

func TestGenerateArticle_ForbiddenForNonAdmin(t *testing.T) {
	r := chi.NewRouter()
	c := &Controller{service: fakeService{}}
	c.RegisterRoutes(r)

	req := httptest.NewRequest(http.MethodPost, "/ai/generate-article", strings.NewReader(`{"topic":"T","description":"D"}`))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(auth.ContextWithUser(req.Context(), user.User{Id: "u1", IsAdmin: false}))

	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected %d, got %d", http.StatusForbidden, rr.Code)
	}
}

func TestGenerateArticle_Validation(t *testing.T) {
	r := chi.NewRouter()
	c := &Controller{service: fakeService{}}
	c.RegisterRoutes(r)

	req := httptest.NewRequest(http.MethodPost, "/ai/generate-article", strings.NewReader(`{"topic":"","description":""}`))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(auth.ContextWithUser(req.Context(), user.User{Id: "u1", IsAdmin: true}))

	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected %d, got %d", http.StatusBadRequest, rr.Code)
	}
}

func TestGenerateArticle_SuccessForAdmin(t *testing.T) {
	r := chi.NewRouter()
	c := &Controller{service: fakeService{}}
	c.RegisterRoutes(r)

	req := httptest.NewRequest(http.MethodPost, "/ai/generate-article", strings.NewReader(`{"topic":"T","description":"D","provider":"openai"}`))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(auth.ContextWithUser(req.Context(), user.User{Id: "u1", IsAdmin: true}))

	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected %d, got %d: %s", http.StatusOK, rr.Code, rr.Body.String())
	}

	var out GeneratedArticle
	if err := json.NewDecoder(rr.Body).Decode(&out); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if out.TitleRu == "" || out.TitleEn == "" || out.ContentRu == "" || out.ContentEn == "" {
		t.Fatalf("expected filled article fields")
	}
}

func TestGenerateArticle_InvalidProvider(t *testing.T) {
	r := chi.NewRouter()
	c := &Controller{service: fakeService{}}
	c.RegisterRoutes(r)

	req := httptest.NewRequest(http.MethodPost, "/ai/generate-article", strings.NewReader(`{"topic":"T","description":"D","provider":"nope"}`))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(auth.ContextWithUser(req.Context(), user.User{Id: "u1", IsAdmin: true}))

	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected %d, got %d", http.StatusBadRequest, rr.Code)
	}
}
