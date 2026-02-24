package ai

import (
	"context"
	"testing"
)

type fakeAIProviderForRecipeDraft struct {
	draftCalls int
	lastDraft  *GenerateRecipeDraftRequest
	outDraft   *GeneratedRecipeDraft
}

func (p *fakeAIProviderForRecipeDraft) AnalyzeImage(ctx context.Context, req ImageAnalysisRequest) (*AnalysisResponse, error) {
	return nil, nil
}

func (p *fakeAIProviderForRecipeDraft) AnalyzeText(ctx context.Context, req TextAnalysisRequest) (*AnalysisResponse, error) {
	return nil, nil
}

func (p *fakeAIProviderForRecipeDraft) ImproveText(ctx context.Context, html string) (string, error) {
	return html, nil
}

func (p *fakeAIProviderForRecipeDraft) GenerateArticle(ctx context.Context, req GenerateArticleRequest) (*GeneratedArticle, error) {
	return &GeneratedArticle{
		TitleRu:   "RU",
		TitleEn:   "EN",
		ContentRu: "<p>RU</p>",
		ContentEn: "<p>EN</p>",
	}, nil
}

func (p *fakeAIProviderForRecipeDraft) GenerateRecipeDraft(ctx context.Context, req GenerateRecipeDraftRequest) (*GeneratedRecipeDraft, error) {
	p.draftCalls++
	p.lastDraft = &req
	return p.outDraft, nil
}

func (p *fakeAIProviderForRecipeDraft) GetModelName() string { return "fake" }

func (p *fakeAIProviderForRecipeDraft) CalculateCost(promptTokens, completionTokens int) float64 {
	return 0
}

func TestGenerateRecipeDraft_SelectsProviderAndNormalizes(t *testing.T) {
	invalidDifficulty := "VERY_HARD"
	primary := &fakeAIProviderForRecipeDraft{
		outDraft: &GeneratedRecipeDraft{
			TitleRu:        "Оладьи",
			TitleEn:        "Pancakes",
			PrepTime:       intPtr(10),
			CookTime:       intPtr(15),
			Servings:       0,
			Difficulty:     &invalidDifficulty,
			TagSuggestions: []string{"Быстрый", "быстрый", " ", "Завтрак"},
			Ingredients: []GeneratedRecipeIngredient{
				{SortOrder: 0, NameRu: "  Мука  ", NameEn: "Flour"},
			},
			Steps: []GeneratedRecipeStep{
				{StepNumber: 0, InstructionRu: "  Смешайте ингредиенты  "},
			},
		},
	}
	fallback := &fakeAIProviderForRecipeDraft{
		outDraft: &GeneratedRecipeDraft{
			TitleRu:  "Запеканка",
			TitleEn:  "Casserole",
			Servings: 3,
			Ingredients: []GeneratedRecipeIngredient{
				{SortOrder: 1, NameRu: "Творог"},
			},
			Steps: []GeneratedRecipeStep{
				{StepNumber: 1, InstructionRu: "Выпекайте"},
			},
		},
	}

	s := &service{
		aiProvider:       primary,
		aiProviderName:   "openai",
		fallbackProvider: fallback,
		fallbackName:     "perplexity",
	}

	draft1, err := s.GenerateRecipeDraft(
		context.Background(),
		"admin",
		"Оладьи",
		"Мука - 100 г",
		"Смешайте",
		nil,
		"https://example.com/image.jpg",
		"openai",
	)
	if err != nil {
		t.Fatalf("GenerateRecipeDraft(openai) returned error: %v", err)
	}

	if primary.draftCalls != 1 || fallback.draftCalls != 0 {
		t.Fatalf("expected primary=1 fallback=0 calls; got primary=%d fallback=%d", primary.draftCalls, fallback.draftCalls)
	}
	if primary.lastDraft == nil || primary.lastDraft.ImageURL == "" {
		t.Fatalf("expected imageURL to be passed to provider")
	}
	if draft1.Servings != 1 {
		t.Fatalf("expected normalized servings=1, got %d", draft1.Servings)
	}
	if draft1.Difficulty != nil {
		t.Fatalf("expected invalid difficulty to be normalized to nil, got %v", *draft1.Difficulty)
	}
	if draft1.TotalTime == nil || *draft1.TotalTime != 25 {
		t.Fatalf("expected totalTime=25, got %+v", draft1.TotalTime)
	}
	if len(draft1.TagSuggestions) != 2 {
		t.Fatalf("expected deduplicated tag suggestions, got %v", draft1.TagSuggestions)
	}

	_, err = s.GenerateRecipeDraft(
		context.Background(),
		"admin",
		"Запеканка",
		"Творог",
		"Выпекайте",
		nil,
		"https://example.com/image2.jpg",
		"perplexity",
	)
	if err != nil {
		t.Fatalf("GenerateRecipeDraft(perplexity) returned error: %v", err)
	}
	if fallback.draftCalls != 1 {
		t.Fatalf("expected fallback provider to be called once, got %d", fallback.draftCalls)
	}
}

func TestGenerateRecipeDraft_ServiceValidation(t *testing.T) {
	s := &service{
		aiProvider:     &fakeAIProviderForRecipeDraft{outDraft: nil},
		aiProviderName: "openai",
	}

	_, err := s.GenerateRecipeDraft(context.Background(), "admin", "", "i", "s", nil, "https://example.com/i.jpg", "auto")
	if err == nil {
		t.Fatalf("expected validation error for empty titleRu")
	}

	_, err = s.GenerateRecipeDraft(context.Background(), "admin", "Title", "i", "s", nil, "", "auto")
	if err == nil {
		t.Fatalf("expected validation error when image source is missing")
	}
}

func intPtr(v int) *int {
	return &v
}
