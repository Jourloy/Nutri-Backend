package ai

import (
	"context"
	"testing"
)

type fakeAIProviderForGenerateArticle struct {
	calls int
	last  *GenerateArticleRequest
	out   *GeneratedArticle
}

func (p *fakeAIProviderForGenerateArticle) AnalyzeImage(ctx context.Context, req ImageAnalysisRequest) (*AnalysisResponse, error) {
	return nil, nil
}

func (p *fakeAIProviderForGenerateArticle) AnalyzeText(ctx context.Context, req TextAnalysisRequest) (*AnalysisResponse, error) {
	return nil, nil
}

func (p *fakeAIProviderForGenerateArticle) ImproveText(ctx context.Context, html string) (string, error) {
	return html, nil
}

func (p *fakeAIProviderForGenerateArticle) GenerateArticle(ctx context.Context, req GenerateArticleRequest) (*GeneratedArticle, error) {
	p.calls++
	p.last = &req
	return p.out, nil
}

func (p *fakeAIProviderForGenerateArticle) GenerateRecipeDraft(ctx context.Context, req GenerateRecipeDraftRequest) (*GeneratedRecipeDraft, error) {
	return &GeneratedRecipeDraft{
		TitleRu: req.TitleRu,
		TitleEn: "EN title",
		Ingredients: []GeneratedRecipeIngredient{
			{SortOrder: 1, NameRu: "Ингредиент"},
		},
		Steps: []GeneratedRecipeStep{
			{StepNumber: 1, InstructionRu: "Шаг"},
		},
	}, nil
}

func (p *fakeAIProviderForGenerateArticle) GetModelName() string { return "fake" }

func (p *fakeAIProviderForGenerateArticle) CalculateCost(promptTokens, completionTokens int) float64 {
	return 0
}

func TestGenerateArticle_SelectsProvider_AndSanitizesSuffixes(t *testing.T) {
	primary := &fakeAIProviderForGenerateArticle{
		out: &GeneratedArticle{
			TitleRu:           "RU title",
			TitleEn:           "EN title",
			PreviewTextRu:     "Превью (168 символов).",
			PreviewTextEn:     "Preview (168 characters).",
			MetaDescriptionRu: "SEO (168 символов).",
			MetaDescriptionEn: "SEO (168 characters).",
			ContentRu:         "<p>RU content</p>",
			ContentEn:         "<p>EN content</p>",
		},
	}
	fallback := &fakeAIProviderForGenerateArticle{
		out: &GeneratedArticle{
			TitleRu:           "RU title 2",
			TitleEn:           "EN title 2",
			PreviewTextRu:     "Превью (168 символов).",
			PreviewTextEn:     "Preview (168 characters).",
			MetaDescriptionRu: "SEO (168 символов).",
			MetaDescriptionEn: "SEO (168 characters).",
			ContentRu:         "<p>RU content</p>",
			ContentEn:         "<p>EN content</p>",
		},
	}

	s := &service{
		aiProvider:       primary,
		aiProviderName:   "openai",
		fallbackProvider: fallback,
		fallbackName:     "perplexity",
	}

	// Explicit provider selects primary.
	a1, err := s.GenerateArticle(context.Background(), "admin", "topic", "desc", "openai")
	if err != nil {
		t.Fatalf("GenerateArticle(openai) returned error: %v", err)
	}
	if primary.calls != 1 || fallback.calls != 0 {
		t.Fatalf("expected primary=1 fallback=0 calls; got primary=%d fallback=%d", primary.calls, fallback.calls)
	}
	if primary.last == nil || primary.last.Provider != "openai" {
		t.Fatalf("expected request.Provider=openai, got %+v", primary.last)
	}
	if a1.PreviewTextRu == "" || a1.MetaDescriptionRu == "" {
		t.Fatalf("expected non-empty sanitized fields")
	}
	if StripTrailingCharCount("Превью (168 символов).") != a1.PreviewTextRu {
		t.Fatalf("expected PreviewTextRu to be sanitized; got %q", a1.PreviewTextRu)
	}

	// Explicit provider selects fallback.
	a2, err := s.GenerateArticle(context.Background(), "admin", "topic", "desc", "perplexity")
	if err != nil {
		t.Fatalf("GenerateArticle(perplexity) returned error: %v", err)
	}
	if primary.calls != 1 || fallback.calls != 1 {
		t.Fatalf("expected primary=1 fallback=1 calls; got primary=%d fallback=%d", primary.calls, fallback.calls)
	}
	if fallback.last == nil || fallback.last.Provider != "perplexity" {
		t.Fatalf("expected request.Provider=perplexity, got %+v", fallback.last)
	}
	if a2.TitleRu != "RU title 2" {
		t.Fatalf("expected fallback result, got %q", a2.TitleRu)
	}

	// Auto selects primary.
	_, err = s.GenerateArticle(context.Background(), "admin", "topic", "desc", "auto")
	if err != nil {
		t.Fatalf("GenerateArticle(auto) returned error: %v", err)
	}
	if primary.calls != 2 {
		t.Fatalf("expected primary called again for auto; got %d", primary.calls)
	}
}

func TestGenerateArticle_NormalizesSwappedLanguages(t *testing.T) {
	provider := &fakeAIProviderForGenerateArticle{
		out: &GeneratedArticle{
			TitleRu:           "Smart goal tracking with Nutri",
			TitleEn:           "Как Nutri помогает считать КБЖУ и клетчатку",
			PreviewTextRu:     "Nutri helps track workouts and supplements with adaptive goals.",
			PreviewTextEn:     "Nutri помогает отслеживать тренировки, добавки и персональные цели.",
			MetaDescriptionRu: "Track calories, workouts, and cycle in one app with Nutri.",
			MetaDescriptionEn: "Как в Nutri отслеживать КБЖУ, клетчатку и холестерин каждый день.",
			ContentRu:         "<h2>Nutri Features</h2><p>Track calories and workouts daily.</p>",
			ContentEn:         "<h2>Возможности Nutri</h2><p>Отслеживайте КБЖУ и тренировки каждый день.</p>",
		},
	}

	s := &service{
		aiProvider:     provider,
		aiProviderName: "openai",
	}

	article, err := s.GenerateArticle(context.Background(), "admin", "topic", "desc", "openai")
	if err != nil {
		t.Fatalf("GenerateArticle returned error: %v", err)
	}

	if article.TitleRu != "Как Nutri помогает считать КБЖУ и клетчатку" {
		t.Fatalf("expected normalized TitleRu, got %q", article.TitleRu)
	}
	if article.TitleEn != "Smart goal tracking with Nutri" {
		t.Fatalf("expected normalized TitleEn, got %q", article.TitleEn)
	}
	if article.PreviewTextRu != "Nutri помогает отслеживать тренировки, добавки и персональные цели." {
		t.Fatalf("expected normalized PreviewTextRu, got %q", article.PreviewTextRu)
	}
	if article.PreviewTextEn != "Nutri helps track workouts and supplements with adaptive goals." {
		t.Fatalf("expected normalized PreviewTextEn, got %q", article.PreviewTextEn)
	}
}
