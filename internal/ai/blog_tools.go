package ai

import (
	"net/url"
	"regexp"
	"strings"
)

const improveTextSystemPrompt = `You are a text editor. You are given HTML content. Your task is to improve the text formatting and structure.

FORMATTING RULES:
1. Fix punctuation: add commas, periods, colons where necessary
2. Capitalize the first letter of each sentence
3. Add a period at the end of each sentence (except headings)

STRUCTURE RULES - LISTS:
1. Convert comma-separated or semicolon-separated enumerations into <ul><li> lists
2. Convert numbered enumerations (1. 2. 3.) into <ol><li> lists
3. IMPORTANT: When you see a heading ending with colon followed by multiple paragraphs in format "term - description", convert them into a <ul> list where each paragraph becomes a <li>
4. Capitalize the first letter of each list item
5. You may add <strong> for important terms and <em> for emphasis

Example of multi-paragraph enumeration to convert:
Input:
  "Skills to develop:
   communication - ability to express thoughts
   leadership - ability to lead teams"
Output:
  "Skills to develop:
   <ul>
   <li>Communication - ability to express thoughts.</li>
   <li>Leadership - ability to lead teams.</li>
   </ul>"

TAG CLEANUP RULES:
1. If only part of a word is wrapped in <strong> or <em>, extend the tag to cover the entire word
2. If a phrase like "word1/word2" has only one part bolded, bold the entire phrase

FORBIDDEN:
- Do NOT rephrase text or change its meaning
- Do NOT add new sentences or remove existing ones
- Do NOT change the order of paragraphs
- Do NOT remove existing HTML tags (except when replacing with more appropriate ones)

Return ONLY the corrected HTML, without comments or explanations.`

const generateArticleSystemPrompt = `You are an expert nutrition blog writer for the Somivyn product.

Somivyn facts (do not invent anything beyond this list):
- Food tracking with calories, proteins, fats, carbs (KBJU), fiber, and cholesterol.
- Supplement tracking.
- Workout and body metrics tracking.
- Menstrual cycle tracking.
- AI that can estimate product weight and calculate KBJU, fiber, and cholesterol.
- Personalized goals that can adjust automatically based on workouts or user condition.
- Ability to mark diseases and monitor condition.

You MUST return ONLY a valid JSON object with EXACTLY these keys:
- titleRu
- titleEn
- previewTextRu
- previewTextEn
- metaDescriptionRu
- metaDescriptionEn
- contentRu
- contentEn

LANGUAGE RULES (CRITICAL):
- titleRu, previewTextRu, metaDescriptionRu, contentRu must be ONLY in Russian.
- titleEn, previewTextEn, metaDescriptionEn, contentEn must be ONLY in English.
- Never mix languages in the same field.
- Never swap RU and EN fields.

CONTENT RULES:
- contentRu/contentEn must be HTML fragments (no <html>, no <body>)
- Allowed tags only: <p>, <h1>, <h2>, <h3>, <h4>, <ul>, <ol>, <li>, <blockquote>, <strong>, <em>, <a>
- In addition to allowed HTML tags, you may include a plain-text image marker as a standalone line in square brackets.
- Do NOT include numeric citations/references in text: forbidden patterns include [1], [2], [1,2], [2-4].
- Do NOT use tables
- Do NOT include <style> or <script>
- Do NOT use inline styles

IMAGE MARKERS:
- Use this only when an image clearly improves understanding.
- Include at least 1 image marker in contentRu and at least 1 image marker in contentEn.
- Format must be exactly: [english gemini image prompt]
- Text inside square brackets must be English only.
- Keep image marker as a standalone line. Do not mix it with CTA or other text in the same element.

CTA RULES:
- Add 1-2 CTA links in contentRu and 1-2 CTA links in contentEn.
- CTA must be an HTML link with class "somivyn-cta-link".
- CTA link must include target="_blank" and rel="noopener noreferrer".
- Choose CTA URL contextually based on the article topic and user intent.
- Place CTA in natural points (after a practical section and/or near the conclusion).
- Never place CTA links back-to-back.

STRUCTURE:
- Short intro
- 3 to 6 sections with headings
- Actionable tips
- Short conclusion
- Include a short disclaimer paragraph (no medical claims)
- Include natural references to Somivyn capabilities where relevant to the topic
- Mention ONLY real Somivyn capabilities listed above

SAFETY:
- Avoid medical claims and promises
- Do not claim diagnosis or treatment outcomes

LENGTH:
- 5-10 minutes reading time PER language (RU and EN separately)
- Minimum 900 words per language
- Aim 1200-2000 words per language
- Write in-depth: expand each section to 2-4 short paragraphs, add concrete examples, step-by-step guidance, and practical checklists where appropriate.
- Keep sections substantial and specific; avoid short generic paragraphs.
- Do NOT include character/word counts anywhere. No suffixes like "(168 символов)" or "(168 characters)".

SEO:
- metaDescriptionRu/metaDescriptionEn: 140-160 characters
- previewTextRu/previewTextEn: 120-200 characters
- Do NOT append the length in parentheses (e.g. "(168 символов)") or any other meta text.

EXAMPLES:
- Image marker example: [close-up photo of a healthy breakfast bowl with oats, berries, and nuts, natural morning light]
- CTA example RU: <a class="somivyn-cta-link" href="https://somivyn.com/prices?utm_source=blog&utm_medium=cta&utm_campaign=weight-loss-guide" target="_blank" rel="noopener noreferrer">Попробовать Somivyn</a>
- CTA example EN: <a class="somivyn-cta-link" href="https://somivyn.com/prices?utm_source=blog&utm_medium=cta&utm_campaign=meal-planning" target="_blank" rel="noopener noreferrer">Try Somivyn</a>

Return ONLY JSON. No markdown fences. No extra text.`

const prepareArticleFromRuSystemPrompt = `You are an expert bilingual nutrition editor for the Somivyn product.

You will receive:
- Russian title
- Russian description
- Russian article HTML (already written by admin)

Task:
1. Keep the Russian meaning and structure aligned with the provided content.
2. Generate an English version of the article HTML.
3. Generate RU/EN SEO fields and EN title.
4. Preserve bracket markers like [english gemini prompt] and [==============CTA==============] when relevant.

Return ONLY JSON with EXACTLY these keys:
- titleRu
- titleEn
- previewTextRu
- previewTextEn
- metaDescriptionRu
- metaDescriptionEn
- contentRu
- contentEn

Rules:
- contentRu/contentEn must be HTML fragments.
- Allowed tags: <p>, <h1>, <h2>, <h3>, <h4>, <ul>, <ol>, <li>, <blockquote>, <strong>, <em>, <a>
- Do not add markdown fences.
- Do not include numeric citations like [1], [2], [1,2], [2-4].
- titleRu/contentRu should stay semantically consistent with provided Russian inputs.
- titleEn/contentEn should be natural English localized to the same meaning.
- metaDescription fields: 140-160 chars.
- previewText fields: 120-200 chars.
`

var (
	reHTMLTags              = regexp.MustCompile(`(?s)<[^>]*>`)
	reTrailingCharCountEn   = regexp.MustCompile(`(?i)\s*\(\s*\d+\s*characters?\s*\)\s*\.?\s*$`)
	reTrailingCharCountRu   = regexp.MustCompile(`(?i)\s*\(\s*\d+\s*символ(?:ов|а)?\s*\)\s*\.?\s*$`)
	reInlineNumericCitation = regexp.MustCompile(`\[(?:\s*\d+\s*(?:[-–]\s*\d+\s*)?(?:,\s*\d+\s*(?:[-–]\s*\d+\s*)?)*)\]`)
	reAnySquareMarker       = regexp.MustCompile(`\[[^\[\]\r\n]{3,500}\]`)
	reLatinLetters          = regexp.MustCompile(`[A-Za-z]`)
	reMultiSpaces           = regexp.MustCompile(`[ \t]{2,}`)
	reSpaceBeforePunct      = regexp.MustCompile(`\s+([,.;:!?])`)
)

// StripMarkdownCodeFences extracts content from the first fenced code block if present.
// Otherwise it returns the trimmed input.
func StripMarkdownCodeFences(s string) string {
	s = strings.TrimSpace(s)
	if !strings.Contains(s, "```") {
		return s
	}

	start := strings.Index(s, "```")
	if start == -1 {
		return s
	}

	// Skip the opening fence and optional language spec up to the first newline.
	after := strings.Index(s[start+3:], "\n")
	if after == -1 {
		return s
	}
	after = start + 3 + after + 1

	end := strings.Index(s[after:], "```")
	if end == -1 {
		return s
	}
	end = after + end

	return strings.TrimSpace(s[after:end])
}

// StripTrailingCharCount removes common "(N символов)" / "(N characters)" suffixes from SEO/preview fields.
func StripTrailingCharCount(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return s
	}
	s = reTrailingCharCountRu.ReplaceAllString(s, "")
	s = reTrailingCharCountEn.ReplaceAllString(s, "")
	return strings.TrimSpace(s)
}

// ApproxWordCountFromHTML approximates word count by stripping tags and splitting by whitespace.
func ApproxWordCountFromHTML(html string) int {
	html = strings.TrimSpace(html)
	if html == "" {
		return 0
	}

	plain := reHTMLTags.ReplaceAllString(html, " ")
	plain = strings.ReplaceAll(plain, "&nbsp;", " ")
	plain = strings.ReplaceAll(plain, "\u00a0", " ")
	plain = strings.TrimSpace(plain)
	if plain == "" {
		return 0
	}
	return len(strings.Fields(plain))
}

// StripInlineNumericCitations removes numeric references like [1], [2], [1,2], [3-4].
// It preserves non-numeric square-bracket markers such as Gemini image prompts.
func StripInlineNumericCitations(content string) string {
	if strings.TrimSpace(content) == "" {
		return content
	}

	cleaned := reInlineNumericCitation.ReplaceAllString(content, "")
	cleaned = reSpaceBeforePunct.ReplaceAllString(cleaned, "$1")
	cleaned = reMultiSpaces.ReplaceAllString(cleaned, " ")
	cleaned = strings.ReplaceAll(cleaned, " \n", "\n")
	cleaned = strings.ReplaceAll(cleaned, "\n ", "\n")
	return strings.TrimSpace(cleaned)
}

// HasGeminiImageMarker returns true when content contains at least one non-numeric
// square-bracket marker with Latin letters (expected for English Gemini prompts).
func HasGeminiImageMarker(content string) bool {
	for _, marker := range reAnySquareMarker.FindAllString(content, -1) {
		if reInlineNumericCitation.MatchString(marker) {
			continue
		}
		if reLatinLetters.MatchString(marker) {
			return true
		}
	}
	return false
}

const DefaultFallbackImageMarkerPrompt = "editorial photo of healthy meal planning setup with fresh ingredients, natural light, realistic details"

// EnsureAtLeastOneImageMarker appends a fallback marker when none is present.
func EnsureAtLeastOneImageMarker(content string, fallbackPrompt string) string {
	content = strings.TrimSpace(content)
	if HasGeminiImageMarker(content) {
		return content
	}

	prompt := strings.TrimSpace(fallbackPrompt)
	if prompt == "" {
		prompt = DefaultFallbackImageMarkerPrompt
	}
	marker := "[" + prompt + "]"
	if content == "" {
		return marker
	}
	return strings.TrimSpace(content + "\n\n" + marker)
}

// NormalizeSourceURLs keeps only valid absolute http/https URLs and deduplicates them.
func NormalizeSourceURLs(sources []string) []string {
	if len(sources) == 0 {
		return nil
	}

	normalized := make([]string, 0, len(sources))
	seen := make(map[string]struct{}, len(sources))

	for _, raw := range sources {
		raw = strings.TrimSpace(raw)
		if raw == "" {
			continue
		}

		parsed, err := url.Parse(raw)
		if err != nil || !parsed.IsAbs() || parsed.Host == "" {
			continue
		}

		scheme := strings.ToLower(parsed.Scheme)
		if scheme != "http" && scheme != "https" {
			continue
		}
		parsed.Scheme = scheme
		parsed.Host = strings.ToLower(parsed.Host)

		cleaned := strings.TrimSpace(parsed.String())
		if cleaned == "" {
			continue
		}

		key := strings.ToLower(cleaned)
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		normalized = append(normalized, cleaned)
	}

	if len(normalized) == 0 {
		return nil
	}
	return normalized
}

func normalizeLanguageSample(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return s
	}
	plain := reHTMLTags.ReplaceAllString(s, " ")
	plain = strings.ReplaceAll(plain, "&nbsp;", " ")
	plain = strings.ReplaceAll(plain, "\u00a0", " ")
	return strings.TrimSpace(plain)
}

func countLatinAndCyrillic(s string) (latin int, cyrillic int) {
	for _, r := range s {
		switch {
		case (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z'):
			latin++
		case (r >= 'А' && r <= 'я') || r == 'Ё' || r == 'ё':
			cyrillic++
		}
	}
	return latin, cyrillic
}

func isLikelyRussian(s string) bool {
	latin, cyrillic := countLatinAndCyrillic(normalizeLanguageSample(s))
	return cyrillic >= 4 && cyrillic > latin
}

func isLikelyEnglish(s string) bool {
	latin, cyrillic := countLatinAndCyrillic(normalizeLanguageSample(s))
	return latin >= 4 && latin > cyrillic
}

// NormalizeGeneratedArticleLanguages detects clearly swapped RU/EN pairs and swaps all pairs back.
// Returns true when a swap was applied.
func NormalizeGeneratedArticleLanguages(article *GeneratedArticle) bool {
	if article == nil {
		return false
	}

	pairs := []struct {
		ru *string
		en *string
	}{
		{ru: &article.TitleRu, en: &article.TitleEn},
		{ru: &article.PreviewTextRu, en: &article.PreviewTextEn},
		{ru: &article.MetaDescriptionRu, en: &article.MetaDescriptionEn},
		{ru: &article.ContentRu, en: &article.ContentEn},
	}

	swapSignals := 0
	keepSignals := 0

	for _, pair := range pairs {
		ru := strings.TrimSpace(*pair.ru)
		en := strings.TrimSpace(*pair.en)
		if ru == "" || en == "" {
			continue
		}

		if isLikelyEnglish(ru) && isLikelyRussian(en) {
			swapSignals++
			continue
		}
		if isLikelyRussian(ru) && isLikelyEnglish(en) {
			keepSignals++
		}
	}

	if swapSignals < 2 || swapSignals <= keepSignals {
		return false
	}

	for _, pair := range pairs {
		*pair.ru, *pair.en = *pair.en, *pair.ru
	}
	return true
}

type ImproveTextRequest struct {
	HTML string `json:"html"`
}

type ImproveTextResponse struct {
	HTML string `json:"html"`
}

type GenerateArticleRequest struct {
	Topic       string `json:"topic"`
	Description string `json:"description"`
	TitleRu     string `json:"titleRu,omitempty"`
	ContentRu   string `json:"contentRu,omitempty"`
	Provider    string `json:"provider,omitempty"`
}

type GeneratedArticle struct {
	TitleRu           string   `json:"titleRu"`
	TitleEn           string   `json:"titleEn"`
	PreviewTextRu     string   `json:"previewTextRu"`
	PreviewTextEn     string   `json:"previewTextEn"`
	MetaDescriptionRu string   `json:"metaDescriptionRu"`
	MetaDescriptionEn string   `json:"metaDescriptionEn"`
	ContentRu         string   `json:"contentRu"`
	ContentEn         string   `json:"contentEn"`
	Sources           []string `json:"sources,omitempty"`
}
