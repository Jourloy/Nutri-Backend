package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"mime/multipart"
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
	draftFn   func(ctx context.Context, userId string, titleRu, ingredientsRu, stepsRu string, imageData []byte, imageURL, provider string) (*GeneratedRecipeDraft, error)
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
		Sources:           []string{"https://example.com/source"},
	}, nil
}

func (f fakeService) GenerateRecipeDraft(
	ctx context.Context,
	userId string,
	titleRu, ingredientsRu, stepsRu string,
	imageData []byte,
	imageURL, provider string,
) (*GeneratedRecipeDraft, error) {
	if f.draftFn != nil {
		return f.draftFn(ctx, userId, titleRu, ingredientsRu, stepsRu, imageData, imageURL, provider)
	}
	return &GeneratedRecipeDraft{
		TitleRu:       titleRu,
		TitleEn:       "Oatmeal Pancakes",
		DescriptionRu: "Короткое описание",
		DescriptionEn: "Short description",
		Servings:      2,
		Ingredients: []GeneratedRecipeIngredient{
			{SortOrder: 1, NameRu: "Овсянка"},
		},
		Steps: []GeneratedRecipeStep{
			{StepNumber: 1, InstructionRu: "Смешайте ингредиенты"},
		},
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
	if len(out.Sources) != 1 || out.Sources[0] != "https://example.com/source" {
		t.Fatalf("expected sources in response, got %#v", out.Sources)
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

func buildRecipeDraftMultipart(t *testing.T, withImage bool) (*bytes.Buffer, string) {
	t.Helper()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	_ = writer.WriteField("titleRu", "Овсяные панкейки")
	_ = writer.WriteField("ingredientsRu", "Овсянка - 100 г\nЯйцо - 1 шт")
	_ = writer.WriteField("stepsRu", "Смешайте ингредиенты\nОбжарьте на сковороде")
	_ = writer.WriteField("provider", "openai")

	if withImage {
		part, err := writer.CreateFormFile("image", "test.jpg")
		if err != nil {
			t.Fatalf("failed to create form file: %v", err)
		}
		if _, err := part.Write([]byte("fake-image")); err != nil {
			t.Fatalf("failed to write image data: %v", err)
		}
	}

	if err := writer.Close(); err != nil {
		t.Fatalf("failed to close multipart writer: %v", err)
	}

	return body, writer.FormDataContentType()
}

func TestGenerateRecipeDraft_Unauthorized(t *testing.T) {
	r := chi.NewRouter()
	c := &Controller{service: fakeService{}}
	c.RegisterRoutes(r)

	body, contentType := buildRecipeDraftMultipart(t, true)
	req := httptest.NewRequest(http.MethodPost, "/ai/generate-recipe-draft", body)
	req.Header.Set("Content-Type", contentType)

	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected %d, got %d", http.StatusUnauthorized, rr.Code)
	}
}

func TestGenerateRecipeDraft_ForbiddenForNonAdmin(t *testing.T) {
	r := chi.NewRouter()
	c := &Controller{service: fakeService{}}
	c.RegisterRoutes(r)

	body, contentType := buildRecipeDraftMultipart(t, true)
	req := httptest.NewRequest(http.MethodPost, "/ai/generate-recipe-draft", body)
	req.Header.Set("Content-Type", contentType)
	req = req.WithContext(auth.ContextWithUser(req.Context(), user.User{Id: "u1", IsAdmin: false}))

	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected %d, got %d", http.StatusForbidden, rr.Code)
	}
}

func TestGenerateRecipeDraft_Validation(t *testing.T) {
	r := chi.NewRouter()
	c := &Controller{service: fakeService{}}
	c.RegisterRoutes(r)

	body, contentType := buildRecipeDraftMultipart(t, false)
	req := httptest.NewRequest(http.MethodPost, "/ai/generate-recipe-draft", body)
	req.Header.Set("Content-Type", contentType)
	req = req.WithContext(auth.ContextWithUser(req.Context(), user.User{Id: "u1", IsAdmin: true}))

	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected %d, got %d", http.StatusBadRequest, rr.Code)
	}
}

func TestGenerateRecipeDraft_SuccessForAdmin(t *testing.T) {
	r := chi.NewRouter()
	c := &Controller{service: fakeService{
		draftFn: func(ctx context.Context, userId string, titleRu, ingredientsRu, stepsRu string, imageData []byte, imageURL, provider string) (*GeneratedRecipeDraft, error) {
			if len(imageData) == 0 {
				t.Fatalf("expected image bytes to be forwarded to service")
			}
			if provider != "openai" {
				t.Fatalf("expected provider=openai, got %q", provider)
			}
			return &GeneratedRecipeDraft{
				TitleRu: titleRu,
				TitleEn: "Oatmeal Pancakes",
				Ingredients: []GeneratedRecipeIngredient{
					{SortOrder: 1, NameRu: "Овсянка"},
				},
				Steps: []GeneratedRecipeStep{
					{StepNumber: 1, InstructionRu: "Смешайте ингредиенты"},
				},
			}, nil
		},
	}}
	c.RegisterRoutes(r)

	body, contentType := buildRecipeDraftMultipart(t, true)
	req := httptest.NewRequest(http.MethodPost, "/ai/generate-recipe-draft", body)
	req.Header.Set("Content-Type", contentType)
	req = req.WithContext(auth.ContextWithUser(req.Context(), user.User{Id: "u1", IsAdmin: true}))

	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected %d, got %d: %s", http.StatusOK, rr.Code, rr.Body.String())
	}

	var out GeneratedRecipeDraft
	if err := json.NewDecoder(rr.Body).Decode(&out); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if out.TitleEn == "" || len(out.Ingredients) == 0 || len(out.Steps) == 0 {
		t.Fatalf("expected filled recipe draft fields")
	}
}
