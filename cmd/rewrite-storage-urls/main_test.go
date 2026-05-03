package main

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"fmt"
	"strings"
	"testing"

	"github.com/jourloy/nutri02/internal/storage"
)

type execCall struct {
	query string
	args  []any
}

type fakeExecutor struct {
	urlRowsByColumn map[string][]stringRow
	blogRows        []blogContentRow
	execCalls       []execCall
}

func (f *fakeExecutor) SelectContext(ctx context.Context, dest any, query string, args ...any) error {
	switch rows := dest.(type) {
	case *[]stringRow:
		key := ""
		switch {
		case strings.Contains(query, "FROM blog_articles") && strings.Contains(query, "preview_image_url AS value"):
			key = "blog_articles.preview_image_url"
		case strings.Contains(query, "FROM blog_articles") && strings.Contains(query, "og_image_url AS value"):
			key = "blog_articles.og_image_url"
		case strings.Contains(query, "FROM recipe_books") && strings.Contains(query, "og_image_url AS value"):
			key = "recipe_books.og_image_url"
		case strings.Contains(query, "FROM recipes") && strings.Contains(query, "main_image_url AS value"):
			key = "recipes.main_image_url"
		case strings.Contains(query, "FROM recipes") && strings.Contains(query, "og_image_url AS value"):
			key = "recipes.og_image_url"
		case strings.Contains(query, "FROM recipe_steps") && strings.Contains(query, "image_url AS value"):
			key = "recipe_steps.image_url"
		case strings.Contains(query, "FROM recipe_images") && strings.Contains(query, "image_url AS value"):
			key = "recipe_images.image_url"
		default:
			*rows = nil
			return nil
		}

		source := f.urlRowsByColumn[key]
		*rows = append([]stringRow(nil), source...)
		return nil
	case *[]blogContentRow:
		*rows = append([]blogContentRow(nil), f.blogRows...)
		return nil
	default:
		return fmt.Errorf("unexpected destination type %T", dest)
	}
}

func (f *fakeExecutor) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	copiedArgs := append([]any(nil), args...)
	f.execCalls = append(f.execCalls, execCall{query: query, args: copiedArgs})
	return driver.RowsAffected(1), nil
}

func mustNewBlogRewriter(t *testing.T) *storage.BlogImageURLCanonicalizer {
	t.Helper()

	rewriter, err := storage.NewBlogImageURLCanonicalizer(storage.Config{
		Endpoint:      "https://internal.example.com",
		PublicBaseURL: "https://cdn.example.com/storage",
		BucketName:    "somivyn-images",
		UseSSL:        true,
	})
	if err != nil {
		t.Fatalf("NewBlogImageURLCanonicalizer() error = %v", err)
	}
	return rewriter
}

func TestRewriteAllBlogScopeOnlyReturnsBlogSummary(t *testing.T) {
	t.Parallel()

	executor := &fakeExecutor{
		urlRowsByColumn: map[string][]stringRow{
			"blog_articles.preview_image_url": {
				{ID: 1, Value: sql.NullString{String: "/nutri02-blog-images/2026/03/preview.png", Valid: true}},
			},
			"blog_articles.og_image_url": {
				{ID: 2, Value: sql.NullString{String: "https://api.nutri02.com/nutri02-blog-images/2026/03/og.png", Valid: true}},
			},
		},
		blogRows: []blogContentRow{
			{ID: 3, ContentRu: `<p><img src="/nutri02-blog-images/2026/03/body.png" /></p>`, ContentEn: `<p>EN</p>`},
		},
	}

	summary, err := rewriteAll(
		context.Background(),
		executor,
		nil,
		mustNewBlogRewriter(t),
		mustNewRecipeRewriter(t),
		"blog",
		true,
	)
	if err != nil {
		t.Fatalf("rewriteAll() error = %v", err)
	}

	if len(summary) != 3 {
		t.Fatalf("expected 3 summary rows, got %#v", summary)
	}
	if summary[0].label != "blog_articles.preview_image_url" {
		t.Fatalf("unexpected first summary row: %#v", summary[0])
	}
	if summary[1].label != "blog_articles.og_image_url" {
		t.Fatalf("unexpected second summary row: %#v", summary[1])
	}
	if summary[2].label != "blog_articles.content_ru/content_en" {
		t.Fatalf("unexpected third summary row: %#v", summary[2])
	}
	if len(executor.execCalls) != 0 {
		t.Fatalf("expected dry-run to skip writes, got %d execs", len(executor.execCalls))
	}
}

func TestRewriteAllBlogScopeAppliesCanonicalBlogUpdates(t *testing.T) {
	t.Parallel()

	executor := &fakeExecutor{
		urlRowsByColumn: map[string][]stringRow{
			"blog_articles.preview_image_url": {
				{ID: 1, Value: sql.NullString{String: "/nutri02-blog-images/2026/03/preview.png", Valid: true}},
			},
			"blog_articles.og_image_url": {
				{ID: 2, Value: sql.NullString{String: "https://minio.jourloy.com/nutri02-blog-images/2026/03/og.png", Valid: true}},
			},
		},
		blogRows: []blogContentRow{
			{
				ID:        3,
				ContentRu: `<p><img src="/nutri02-blog-images/2026/03/body-ru.png" /></p>`,
				ContentEn: `<p><img src="https://api.nutri02.com/nutri02-blog-images/2026/03/body-en.png" /></p>`,
			},
		},
	}

	_, err := rewriteAll(context.Background(), executor, nil, mustNewBlogRewriter(t), mustNewRecipeRewriter(t), "blog", false)
	if err != nil {
		t.Fatalf("rewriteAll() error = %v", err)
	}

	if len(executor.execCalls) != 3 {
		t.Fatalf("expected 3 update statements, got %d", len(executor.execCalls))
	}

	if got := executor.execCalls[0].args[0]; got != "https://cdn.example.com/storage/somivyn-images/blog/2026/03/preview.png" {
		t.Fatalf("preview update arg = %#v", got)
	}
	if got := executor.execCalls[1].args[0]; got != "https://cdn.example.com/storage/somivyn-images/blog/2026/03/og.png" {
		t.Fatalf("og update arg = %#v", got)
	}
	if got := executor.execCalls[2].args[0]; got != `<p><img src="https://cdn.example.com/storage/somivyn-images/blog/2026/03/body-ru.png" /></p>` {
		t.Fatalf("content_ru update arg = %#v", got)
	}
	if got := executor.execCalls[2].args[1]; got != `<p><img src="https://cdn.example.com/storage/somivyn-images/blog/2026/03/body-en.png" /></p>` {
		t.Fatalf("content_en update arg = %#v", got)
	}
}

func mustNewRecipeRewriter(t *testing.T) *storage.RecipeImageURLCanonicalizer {
	t.Helper()

	rewriter, err := storage.NewRecipeImageURLCanonicalizer(storage.Config{
		Endpoint:      "https://internal.example.com",
		PublicBaseURL: "https://cdn.example.com/storage",
		BucketName:    "somivyn-images",
		UseSSL:        true,
	})
	if err != nil {
		t.Fatalf("NewRecipeImageURLCanonicalizer() error = %v", err)
	}
	return rewriter
}

func TestRewriteAllRecipeScopeOnlyReturnsRecipeSummary(t *testing.T) {
	t.Parallel()

	executor := &fakeExecutor{
		urlRowsByColumn: map[string][]stringRow{
			"recipe_books.og_image_url": {
				{ID: 1, Value: sql.NullString{String: "/nutri02-recipe-images/2026/03/book.webp", Valid: true}},
			},
			"recipes.main_image_url": {
				{ID: 2, Value: sql.NullString{String: "https://api.nutri02.com/nutri02-recipe-images/2026/03/main.webp", Valid: true}},
			},
			"recipes.og_image_url": {
				{ID: 3, Value: sql.NullString{String: "https://minio.jourloy.com/nutri02-recipe-images/2026/03/og.webp", Valid: true}},
			},
			"recipe_steps.image_url": {
				{ID: 4, Value: sql.NullString{String: "https://s3.nutri02.com/cd83329f-b1dd-42b6-afac-9af67c6c8cc1/recipe/2026/03/step.webp", Valid: true}},
			},
			"recipe_images.image_url": {
				{ID: 5, Value: sql.NullString{String: "https://cdn.example.com/storage/somivyn-images/recipe/2026/03/extra.webp", Valid: true}},
			},
		},
	}

	summary, err := rewriteAll(context.Background(), executor, nil, mustNewBlogRewriter(t), mustNewRecipeRewriter(t), "recipe", true)
	if err != nil {
		t.Fatalf("rewriteAll() error = %v", err)
	}

	if len(summary) != 5 {
		t.Fatalf("expected 5 summary rows, got %#v", summary)
	}
	if summary[0].label != "recipe_books.og_image_url" {
		t.Fatalf("unexpected first summary row: %#v", summary[0])
	}
	if summary[4].label != "recipe_images.image_url" {
		t.Fatalf("unexpected last summary row: %#v", summary[4])
	}
	if len(executor.execCalls) != 0 {
		t.Fatalf("expected dry-run to skip writes, got %d execs", len(executor.execCalls))
	}
}

func TestRewriteAllRecipeScopeAppliesCanonicalRecipeUpdates(t *testing.T) {
	t.Parallel()

	executor := &fakeExecutor{
		urlRowsByColumn: map[string][]stringRow{
			"recipe_books.og_image_url": {
				{ID: 1, Value: sql.NullString{String: "/nutri02-recipe-images/2026/03/book.webp", Valid: true}},
			},
			"recipes.main_image_url": {
				{ID: 2, Value: sql.NullString{String: "https://api.nutri02.com/nutri02-recipe-images/2026/03/main.webp", Valid: true}},
			},
			"recipes.og_image_url": {
				{ID: 3, Value: sql.NullString{String: "https://minio.jourloy.com/nutri02-recipe-images/2026/03/og.webp", Valid: true}},
			},
			"recipe_steps.image_url": {
				{ID: 4, Value: sql.NullString{String: "https://s3.nutri02.com/cd83329f-b1dd-42b6-afac-9af67c6c8cc1/recipe/2026/03/step.webp", Valid: true}},
			},
			"recipe_images.image_url": {
				{ID: 5, Value: sql.NullString{String: "https://cdn.example.com/storage/somivyn-images/recipe/2026/03/extra.webp", Valid: true}},
			},
		},
	}

	_, err := rewriteAll(context.Background(), executor, nil, mustNewBlogRewriter(t), mustNewRecipeRewriter(t), "recipe", false)
	if err != nil {
		t.Fatalf("rewriteAll() error = %v", err)
	}

	if len(executor.execCalls) != 4 {
		t.Fatalf("expected 4 update statements, got %d", len(executor.execCalls))
	}
	if got := executor.execCalls[0].args[0]; got != "https://cdn.example.com/storage/somivyn-images/recipe/2026/03/book.webp" {
		t.Fatalf("recipe book update arg = %#v", got)
	}
	if got := executor.execCalls[3].args[0]; got != "https://cdn.example.com/storage/somivyn-images/recipe/2026/03/step.webp" {
		t.Fatalf("recipe step update arg = %#v", got)
	}
}
