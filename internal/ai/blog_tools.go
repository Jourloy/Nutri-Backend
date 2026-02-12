package ai

import (
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

const generateArticleSystemPrompt = `You are an expert nutrition blog writer.

You MUST return ONLY a valid JSON object with EXACTLY these keys:
- titleRu
- titleEn
- previewTextRu
- previewTextEn
- metaDescriptionRu
- metaDescriptionEn
- contentRu
- contentEn

CONTENT RULES:
- contentRu/contentEn must be HTML fragments (no <html>, no <body>)
- Allowed tags only: <p>, <h1>, <h2>, <h3>, <h4>, <ul>, <ol>, <li>, <blockquote>, <strong>, <em>, <a>
- Do NOT use tables
- Do NOT include <style> or <script>
- Do NOT use inline styles

STRUCTURE:
- Short intro
- 3 to 6 sections with headings
- Actionable tips
- Short conclusion
- Include a short disclaimer paragraph (no medical claims)

SAFETY:
- Avoid medical claims and promises

LENGTH:
- 5-10 minutes reading time PER language (RU and EN separately)
- Minimum 900 words per language
- Aim 1200-2000 words per language
- Write in-depth: expand each section to 2-4 short paragraphs, add concrete examples, step-by-step guidance, and practical checklists where appropriate.
- Do NOT include character/word counts anywhere. No suffixes like "(168 символов)" or "(168 characters)".

SEO:
- metaDescriptionRu/metaDescriptionEn: 140-160 characters
- previewTextRu/previewTextEn: 120-200 characters
- Do NOT append the length in parentheses (e.g. "(168 символов)") or any other meta text.

Return ONLY JSON. No markdown fences. No extra text.`

var (
	reHTMLTags            = regexp.MustCompile(`(?s)<[^>]*>`)
	reTrailingCharCountEn = regexp.MustCompile(`(?i)\s*\(\s*\d+\s*characters?\s*\)\s*\.?\s*$`)
	reTrailingCharCountRu = regexp.MustCompile(`(?i)\s*\(\s*\d+\s*символ(?:ов|а)?\s*\)\s*\.?\s*$`)
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

type ImproveTextRequest struct {
	HTML string `json:"html"`
}

type ImproveTextResponse struct {
	HTML string `json:"html"`
}

type GenerateArticleRequest struct {
	Topic       string `json:"topic"`
	Description string `json:"description"`
	Provider    string `json:"provider,omitempty"`
}

type GeneratedArticle struct {
	TitleRu           string `json:"titleRu"`
	TitleEn           string `json:"titleEn"`
	PreviewTextRu     string `json:"previewTextRu"`
	PreviewTextEn     string `json:"previewTextEn"`
	MetaDescriptionRu string `json:"metaDescriptionRu"`
	MetaDescriptionEn string `json:"metaDescriptionEn"`
	ContentRu         string `json:"contentRu"`
	ContentEn         string `json:"contentEn"`
}
