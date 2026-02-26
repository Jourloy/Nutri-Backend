package ai

import (
	"strings"
	"testing"
)

func TestNormalizeGeneratedArticleLanguages_SwapsPairs(t *testing.T) {
	article := &GeneratedArticle{
		TitleRu:           "Smart tracking with Nutri",
		TitleEn:           "Как Nutri помогает считать КБЖУ",
		PreviewTextRu:     "Track calories, cholesterol and fiber in Nutri.",
		PreviewTextEn:     "Nutri помогает отслеживать КБЖУ, холестерин и клетчатку.",
		MetaDescriptionRu: "Track nutrition and workouts in one app.",
		MetaDescriptionEn: "Как отслеживать питание и тренировки в Nutri.",
		ContentRu:         "<p>Track supplements and workouts.</p>",
		ContentEn:         "<p>Отслеживайте добавки и тренировки.</p>",
	}

	swapped := NormalizeGeneratedArticleLanguages(article)
	if !swapped {
		t.Fatalf("expected swapped=true")
	}
	if article.TitleRu != "Как Nutri помогает считать КБЖУ" {
		t.Fatalf("expected swapped TitleRu, got %q", article.TitleRu)
	}
	if article.TitleEn != "Smart tracking with Nutri" {
		t.Fatalf("expected swapped TitleEn, got %q", article.TitleEn)
	}
}

func TestNormalizeGeneratedArticleLanguages_LeavesCorrectPairs(t *testing.T) {
	article := &GeneratedArticle{
		TitleRu:           "Как Nutri помогает считать КБЖУ",
		TitleEn:           "How Nutri helps with macro tracking",
		PreviewTextRu:     "Nutri помогает отслеживать добавки и тренировки каждый день.",
		PreviewTextEn:     "Nutri helps track supplements and workouts every day.",
		MetaDescriptionRu: "Учёт питания, тренировок и цикла в одном приложении.",
		MetaDescriptionEn: "Track nutrition, workouts, and cycle in one app.",
		ContentRu:         "<p>Отслеживайте КБЖУ, клетчатку и холестерин.</p>",
		ContentEn:         "<p>Track calories, fiber, and cholesterol.</p>",
	}

	swapped := NormalizeGeneratedArticleLanguages(article)
	if swapped {
		t.Fatalf("expected swapped=false")
	}
	if article.TitleRu != "Как Nutri помогает считать КБЖУ" {
		t.Fatalf("expected original TitleRu, got %q", article.TitleRu)
	}
	if article.TitleEn != "How Nutri helps with macro tracking" {
		t.Fatalf("expected original TitleEn, got %q", article.TitleEn)
	}
}

func TestGenerateArticleSystemPrompt_IncludesImageMarkersAndCtaRules(t *testing.T) {
	requiredSnippets := []string{
		"You MUST return ONLY a valid JSON object with EXACTLY these keys:",
		"LANGUAGE RULES (CRITICAL):",
		"IMAGE MARKERS:",
		"[english gemini image prompt]",
		"Text inside square brackets must be English only.",
		"CTA RULES:",
		"Add 1-2 CTA links in contentRu and 1-2 CTA links in contentEn.",
		`class "nutri-cta-link"`,
		`target="_blank" and rel="noopener noreferrer"`,
		"Image marker example:",
		"CTA example RU:",
		"CTA example EN:",
	}

	for _, snippet := range requiredSnippets {
		if !strings.Contains(generateArticleSystemPrompt, snippet) {
			t.Fatalf("expected prompt to contain %q", snippet)
		}
	}
}
