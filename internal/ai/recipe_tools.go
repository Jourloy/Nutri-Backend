package ai

import "strings"

const generateRecipeDraftSystemPrompt = `You are an assistant that generates a structured recipe draft for admin users.

Return ONLY valid JSON with EXACTLY these keys:
- titleRu
- titleEn
- descriptionRu
- descriptionEn
- slug
- prepTime
- cookTime
- totalTime
- servings
- servingsUnit
- difficulty
- calories
- protein
- fat
- carbs
- fiber
- metaDescriptionRu
- metaDescriptionEn
- ingredients
- steps
- categorySuggestion
- tagSuggestions

Rules:
1. The input title, ingredients, and steps are in Russian.
2. Preserve Russian meaning and wording as closely as possible. Do not rewrite the recipe style dramatically.
3. Do not invent unrealistic values.
4. Generate missing bilingual/SEO/nutrition fields.
5. ingredients must be an array of objects:
   - sortOrder (int)
   - nameRu (string)
   - nameEn (string)
   - amount (number or null)
   - unit (string or null)
   - isOptional (boolean)
   - groupName (string or null)
6. steps must be an array of objects:
   - stepNumber (int)
   - instructionRu (string)
   - instructionEn (string)
   - durationMinutes (int or null)
7. difficulty must be one of: easy, medium, hard.
8. categorySuggestion is a short Russian category name suggestion.
9. tagSuggestions is an array of short Russian tags.
10. slug should be URL-friendly latin text when possible.
11. metaDescriptionRu/metaDescriptionEn should be concise and SEO-friendly.

Never add markdown fences or explanations. Return JSON only.`

type GenerateRecipeDraftRequest struct {
	TitleRu       string `json:"titleRu"`
	IngredientsRu string `json:"ingredientsRu"`
	StepsRu       string `json:"stepsRu"`
	ImageURL      string `json:"imageUrl,omitempty"`
	ImageBase64   string `json:"-"`
	Provider      string `json:"provider,omitempty"`
}

type GeneratedRecipeIngredient struct {
	SortOrder  int      `json:"sortOrder"`
	NameRu     string   `json:"nameRu"`
	NameEn     string   `json:"nameEn,omitempty"`
	Amount     *float64 `json:"amount,omitempty"`
	Unit       *string  `json:"unit,omitempty"`
	IsOptional bool     `json:"isOptional"`
	GroupName  *string  `json:"groupName,omitempty"`
}

type GeneratedRecipeStep struct {
	StepNumber      int    `json:"stepNumber"`
	InstructionRu   string `json:"instructionRu"`
	InstructionEn   string `json:"instructionEn,omitempty"`
	DurationMinutes *int   `json:"durationMinutes,omitempty"`
}

type GeneratedRecipeDraft struct {
	TitleRu       string  `json:"titleRu,omitempty"`
	TitleEn       string  `json:"titleEn,omitempty"`
	DescriptionRu string  `json:"descriptionRu,omitempty"`
	DescriptionEn string  `json:"descriptionEn,omitempty"`
	Slug          string  `json:"slug,omitempty"`
	PrepTime      *int    `json:"prepTime,omitempty"`
	CookTime      *int    `json:"cookTime,omitempty"`
	TotalTime     *int    `json:"totalTime,omitempty"`
	Servings      int     `json:"servings,omitempty"`
	ServingsUnit  string  `json:"servingsUnit,omitempty"`
	Difficulty    *string `json:"difficulty,omitempty"`

	Calories *float64 `json:"calories,omitempty"`
	Protein  *float64 `json:"protein,omitempty"`
	Fat      *float64 `json:"fat,omitempty"`
	Carbs    *float64 `json:"carbs,omitempty"`
	Fiber    *float64 `json:"fiber,omitempty"`

	MetaDescriptionRu string `json:"metaDescriptionRu,omitempty"`
	MetaDescriptionEn string `json:"metaDescriptionEn,omitempty"`

	Ingredients []GeneratedRecipeIngredient `json:"ingredients"`
	Steps       []GeneratedRecipeStep       `json:"steps"`

	CategorySuggestion *string  `json:"categorySuggestion,omitempty"`
	TagSuggestions     []string `json:"tagSuggestions,omitempty"`
}

func NormalizeGeneratedRecipeDraft(draft *GeneratedRecipeDraft) *GeneratedRecipeDraft {
	if draft == nil {
		return nil
	}

	draft.TitleRu = strings.TrimSpace(draft.TitleRu)
	draft.TitleEn = strings.TrimSpace(draft.TitleEn)
	draft.DescriptionRu = strings.TrimSpace(draft.DescriptionRu)
	draft.DescriptionEn = strings.TrimSpace(draft.DescriptionEn)
	draft.Slug = strings.TrimSpace(draft.Slug)
	draft.ServingsUnit = strings.TrimSpace(draft.ServingsUnit)
	draft.MetaDescriptionRu = strings.TrimSpace(draft.MetaDescriptionRu)
	draft.MetaDescriptionEn = strings.TrimSpace(draft.MetaDescriptionEn)

	if draft.Servings < 1 {
		draft.Servings = 1
	}

	draft.Difficulty = normalizeRecipeDifficulty(draft.Difficulty)

	if draft.TotalTime == nil {
		total := 0
		if draft.PrepTime != nil && *draft.PrepTime > 0 {
			total += *draft.PrepTime
		}
		if draft.CookTime != nil && *draft.CookTime > 0 {
			total += *draft.CookTime
		}
		if total > 0 {
			draft.TotalTime = &total
		}
	}

	normalizedIngredients := make([]GeneratedRecipeIngredient, 0, len(draft.Ingredients))
	for i, ing := range draft.Ingredients {
		ing.NameRu = strings.TrimSpace(ing.NameRu)
		if ing.NameRu == "" {
			continue
		}
		ing.NameEn = strings.TrimSpace(ing.NameEn)
		if ing.SortOrder <= 0 {
			ing.SortOrder = i + 1
		}
		if ing.Unit != nil {
			trimmed := strings.TrimSpace(*ing.Unit)
			if trimmed == "" {
				ing.Unit = nil
			} else {
				ing.Unit = &trimmed
			}
		}
		if ing.GroupName != nil {
			trimmed := strings.TrimSpace(*ing.GroupName)
			if trimmed == "" {
				ing.GroupName = nil
			} else {
				ing.GroupName = &trimmed
			}
		}
		normalizedIngredients = append(normalizedIngredients, ing)
	}
	draft.Ingredients = normalizedIngredients

	normalizedSteps := make([]GeneratedRecipeStep, 0, len(draft.Steps))
	for i, step := range draft.Steps {
		step.InstructionRu = strings.TrimSpace(step.InstructionRu)
		if step.InstructionRu == "" {
			continue
		}
		step.InstructionEn = strings.TrimSpace(step.InstructionEn)
		if step.StepNumber <= 0 {
			step.StepNumber = i + 1
		}
		normalizedSteps = append(normalizedSteps, step)
	}
	draft.Steps = normalizedSteps

	if draft.CategorySuggestion != nil {
		trimmed := strings.TrimSpace(*draft.CategorySuggestion)
		if trimmed == "" {
			draft.CategorySuggestion = nil
		} else {
			draft.CategorySuggestion = &trimmed
		}
	}
	draft.TagSuggestions = uniqueNonEmptyStrings(draft.TagSuggestions)

	return draft
}

func normalizeRecipeDifficulty(value *string) *string {
	if value == nil {
		return nil
	}
	v := strings.ToLower(strings.TrimSpace(*value))
	switch v {
	case "easy", "medium", "hard":
		return &v
	default:
		return nil
	}
}

func uniqueNonEmptyStrings(values []string) []string {
	if len(values) == 0 {
		return []string{}
	}
	out := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		key := strings.ToLower(trimmed)
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, trimmed)
	}
	return out
}
