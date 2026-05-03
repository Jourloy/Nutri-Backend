package ai

import (
	"strings"
	"testing"
)

func TestNormalizeGeneratedArticleLanguages_SwapsPairs(t *testing.T) {
	article := &GeneratedArticle{
		TitleRu:           "Smart tracking with Nutri02",
		TitleEn:           "Как Nutri02 помогает считать КБЖУ",
		PreviewTextRu:     "Track calories, cholesterol and fiber in Nutri02.",
		PreviewTextEn:     "Nutri02 помогает отслеживать КБЖУ, холестерин и клетчатку.",
		MetaDescriptionRu: "Track nutrition and workouts in one app.",
		MetaDescriptionEn: "Как отслеживать питание и тренировки в Nutri02.",
		ContentRu:         "<p>Track supplements and workouts.</p>",
		ContentEn:         "<p>Отслеживайте добавки и тренировки.</p>",
	}

	swapped := NormalizeGeneratedArticleLanguages(article)
	if !swapped {
		t.Fatalf("expected swapped=true")
	}
	if article.TitleRu != "Как Nutri02 помогает считать КБЖУ" {
		t.Fatalf("expected swapped TitleRu, got %q", article.TitleRu)
	}
	if article.TitleEn != "Smart tracking with Nutri02" {
		t.Fatalf("expected swapped TitleEn, got %q", article.TitleEn)
	}
}

func TestNormalizeGeneratedArticleLanguages_LeavesCorrectPairs(t *testing.T) {
	article := &GeneratedArticle{
		TitleRu:           "Как Nutri02 помогает считать КБЖУ",
		TitleEn:           "How Nutri02 helps with macro tracking",
		PreviewTextRu:     "Nutri02 помогает отслеживать добавки и тренировки каждый день.",
		PreviewTextEn:     "Nutri02 helps track supplements and workouts every day.",
		MetaDescriptionRu: "Учёт питания, тренировок и цикла в одном приложении.",
		MetaDescriptionEn: "Track nutrition, workouts, and cycle in one app.",
		ContentRu:         "<p>Отслеживайте КБЖУ, клетчатку и холестерин.</p>",
		ContentEn:         "<p>Track calories, fiber, and cholesterol.</p>",
	}

	swapped := NormalizeGeneratedArticleLanguages(article)
	if swapped {
		t.Fatalf("expected swapped=false")
	}
	if article.TitleRu != "Как Nutri02 помогает считать КБЖУ" {
		t.Fatalf("expected original TitleRu, got %q", article.TitleRu)
	}
	if article.TitleEn != "How Nutri02 helps with macro tracking" {
		t.Fatalf("expected original TitleEn, got %q", article.TitleEn)
	}
}

func TestGenerateArticleSystemPrompt_IncludesImageMarkersAndCtaRules(t *testing.T) {
	requiredSnippets := []string{
		"You MUST return ONLY a valid JSON object with EXACTLY these keys:",
		"LANGUAGE RULES (CRITICAL):",
		"IMAGE MARKERS:",
		"Do NOT include numeric citations/references in text",
		"Include at least 1 image marker in contentRu and at least 1 image marker in contentEn.",
		"[english gemini image prompt]",
		"Text inside square brackets must be English only.",
		"CTA RULES:",
		"Add 1-2 CTA links in contentRu and 1-2 CTA links in contentEn.",
		`class "nutri02-cta-link"`,
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

func TestStripInlineNumericCitations_PreservesGeminiImageMarker(t *testing.T) {
	input := `<p>Protein is important [1] for recovery [2, 3].</p>
[close-up photo of high-protein meal prep containers on a kitchen table, natural light]`

	cleaned := StripInlineNumericCitations(input)
	if strings.Contains(cleaned, "[1]") || strings.Contains(cleaned, "[2, 3]") {
		t.Fatalf("expected numeric citations to be removed, got %q", cleaned)
	}
	if !strings.Contains(cleaned, "[close-up photo of high-protein meal prep containers on a kitchen table, natural light]") {
		t.Fatalf("expected gemini marker to be preserved, got %q", cleaned)
	}
}

func TestEnsureAtLeastOneImageMarker_AppendsFallbackWhenMissing(t *testing.T) {
	input := "<p>Detailed article content without marker.</p>"
	out := EnsureAtLeastOneImageMarker(input, "")

	if !HasGeminiImageMarker(out) {
		t.Fatalf("expected output to contain an image marker, got %q", out)
	}
	if strings.Contains(out, "[1]") {
		t.Fatalf("unexpected numeric citation marker in output: %q", out)
	}
}

func TestNormalizeSourceURLs_ValidatesAndDeduplicates(t *testing.T) {
	in := []string{
		" https://Example.com/path ",
		"https://example.com/path",
		"http://example.com/another",
		"ftp://example.com/not-allowed",
		"example.com/no-scheme",
		"",
	}

	out := NormalizeSourceURLs(in)
	if len(out) != 2 {
		t.Fatalf("expected 2 normalized urls, got %#v", out)
	}
	if out[0] != "https://example.com/path" {
		t.Fatalf("unexpected first url: %q", out[0])
	}
	if out[1] != "http://example.com/another" {
		t.Fatalf("unexpected second url: %q", out[1])
	}
}
