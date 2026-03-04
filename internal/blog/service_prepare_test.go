package blog

import (
	"context"
	"strings"
	"testing"

	"github.com/jourloy/nutri-backend/internal/ai"
)

type prepareRepoStub struct {
	Repository
	exists map[string]bool
}

func (s *prepareRepoStub) ArticleSlugExists(ctx context.Context, slug string) (bool, error) {
	return s.exists[slug], nil
}

type prepareAIStub struct {
	ai.AIProvider
	article *ai.GeneratedArticle
	err     error
}

func (s *prepareAIStub) GenerateArticle(ctx context.Context, req ai.GenerateArticleRequest) (*ai.GeneratedArticle, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.article, nil
}

func TestPrepareArticle_RequiresMarkdown(t *testing.T) {
	svc := &service{
		repo: &prepareRepoStub{exists: map[string]bool{}},
		aiProvider: &prepareAIStub{article: &ai.GeneratedArticle{
			TitleEn:   "How to track calories",
			ContentEn: "<p>EN content</p>",
		}},
	}

	_, err := svc.PrepareArticle(context.Background(), PrepareArticleRequest{
		TitleRu:       "Как считать калории",
		DescriptionRu: "Краткое описание",
	})
	if err == nil || !strings.Contains(err.Error(), "contentMarkdownRu is required") {
		t.Fatalf("expected markdown validation error, got %v", err)
	}
}

func TestPrepareArticle_GeneratesUniqueSlugAndConvertsMarkdown(t *testing.T) {
	repo := &prepareRepoStub{exists: map[string]bool{
		"kak-schitat-kalorii":   true,
		"kak-schitat-kalorii-2": true,
	}}
	provider := &prepareAIStub{article: &ai.GeneratedArticle{
		TitleEn:           "How to count calories",
		PreviewTextRu:     "",
		PreviewTextEn:     "",
		MetaDescriptionRu: "",
		MetaDescriptionEn: "",
		ContentEn:         "<p>English article body</p>",
		Sources:           []string{"https://Example.com/source"},
	}}

	svc := &service{
		repo:           repo,
		aiProvider:     provider,
		aiProviderName: "openai",
	}

	prepared, err := svc.PrepareArticle(context.Background(), PrepareArticleRequest{
		TitleRu:         "Как считать калории",
		DescriptionRu:   "Описание статьи",
		ContentMarkdown: "# Введение\n\nТекст статьи",
	})
	if err != nil {
		t.Fatalf("PrepareArticle returned error: %v", err)
	}

	if prepared.Slug != "kak-schitat-kalorii-3" {
		t.Fatalf("expected unique slug with suffix, got %q", prepared.Slug)
	}
	if !strings.Contains(prepared.ContentRu, "<h1>Введение</h1>") {
		t.Fatalf("expected markdown header converted to HTML, got %q", prepared.ContentRu)
	}
	if !strings.Contains(prepared.ContentEn, "[") {
		t.Fatalf("expected english content to include image marker fallback, got %q", prepared.ContentEn)
	}
	if prepared.PreviewTextRu != "Описание статьи" {
		t.Fatalf("expected preview RU fallback from description, got %q", prepared.PreviewTextRu)
	}
	if prepared.MetaDescriptionRu != "Описание статьи" {
		t.Fatalf("expected meta RU fallback, got %q", prepared.MetaDescriptionRu)
	}
	if len(prepared.Sources) != 1 || prepared.Sources[0] != "https://example.com/source" {
		t.Fatalf("expected normalized sources, got %#v", prepared.Sources)
	}
}
