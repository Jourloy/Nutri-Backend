package recipe

import "time"

// ===== Recipe Book =====

type Book struct {
	Id         int64      `json:"id" db:"id"`
	UserId     *string    `json:"userId,omitempty" db:"user_id"`
	Name       string     `json:"name" db:"name"`
	BookType   string     `json:"bookType" db:"book_type"`
	ShareToken *string    `json:"shareToken,omitempty" db:"share_token"`
	IsShared   bool       `json:"isShared" db:"is_shared"`
	OgImageUrl *string    `json:"ogImageUrl,omitempty" db:"og_image_url"`
	CreatedAt  time.Time  `json:"createdAt" db:"created_at"`
	UpdatedAt  time.Time  `json:"updatedAt" db:"updated_at"`
	DeletedAt  *time.Time `json:"-" db:"deleted_at"`

	// Computed fields
	RecipeCount int `json:"recipeCount" db:"recipe_count"`
}

type BookCreate struct {
	UserId *string `json:"-"`
	Name   string  `json:"name"`
}

type BookUpdate struct {
	Id     int64  `json:"id"`
	UserId string `json:"-"`
	Name   string `json:"name"`
}

// ===== Recipe Category =====

type Category struct {
	Id           int64      `json:"id" db:"id"`
	UserId       *string    `json:"userId,omitempty" db:"user_id"`
	Slug         *string    `json:"slug,omitempty" db:"slug"`
	NameRu       string     `json:"nameRu" db:"name_ru"`
	NameEn       *string    `json:"nameEn,omitempty" db:"name_en"`
	CategoryType string     `json:"categoryType" db:"category_type"`
	Icon         *string    `json:"icon,omitempty" db:"icon"`
	SortOrder    int        `json:"sortOrder" db:"sort_order"`
	CreatedAt    time.Time  `json:"createdAt" db:"created_at"`
	UpdatedAt    time.Time  `json:"updatedAt" db:"updated_at"`
	DeletedAt    *time.Time `json:"-" db:"deleted_at"`
}

type CategoryCreate struct {
	UserId       *string `json:"-"`
	Slug         *string `json:"slug,omitempty"`
	NameRu       string  `json:"nameRu"`
	NameEn       *string `json:"nameEn,omitempty"`
	CategoryType string  `json:"categoryType,omitempty"`
	Icon         *string `json:"icon,omitempty"`
	SortOrder    int     `json:"sortOrder"`
}

type CategoryUpdate struct {
	Id        int64   `json:"id"`
	UserId    string  `json:"-"`
	Slug      *string `json:"slug,omitempty"`
	NameRu    string  `json:"nameRu"`
	NameEn    *string `json:"nameEn,omitempty"`
	Icon      *string `json:"icon,omitempty"`
	SortOrder int     `json:"sortOrder"`
}

// ===== Recipe Tag =====

type Tag struct {
	Id        int64      `json:"id" db:"id"`
	UserId    *string    `json:"userId,omitempty" db:"user_id"`
	Slug      *string    `json:"slug,omitempty" db:"slug"`
	NameRu    string     `json:"nameRu" db:"name_ru"`
	NameEn    *string    `json:"nameEn,omitempty" db:"name_en"`
	TagType   string     `json:"tagType" db:"tag_type"`
	CreatedAt time.Time  `json:"createdAt" db:"created_at"`
	UpdatedAt time.Time  `json:"updatedAt" db:"updated_at"`
	DeletedAt *time.Time `json:"-" db:"deleted_at"`
}

type TagCreate struct {
	UserId *string `json:"-"`
	Slug   *string `json:"slug,omitempty"`
	NameRu string  `json:"nameRu"`
	NameEn *string `json:"nameEn,omitempty"`
}

type TagUpdate struct {
	Id     int64   `json:"id"`
	UserId string  `json:"-"`
	NameRu string  `json:"nameRu"`
	NameEn *string `json:"nameEn,omitempty"`
}

// ===== Recipe =====

type Recipe struct {
	Id            int64   `json:"id" db:"id"`
	BookId        int64   `json:"bookId" db:"book_id"`
	UserId        *string `json:"userId,omitempty" db:"user_id"`
	Slug          *string `json:"slug,omitempty" db:"slug"`
	TitleRu       string  `json:"titleRu" db:"title_ru"`
	TitleEn       *string `json:"titleEn,omitempty" db:"title_en"`
	DescriptionRu *string `json:"descriptionRu,omitempty" db:"description_ru"`
	DescriptionEn *string `json:"descriptionEn,omitempty" db:"description_en"`
	MainImageUrl  *string `json:"mainImageUrl,omitempty" db:"main_image_url"`
	ExternalUrl   *string `json:"externalUrl,omitempty" db:"external_url"`

	// Timing
	PrepTime  *int `json:"prepTime,omitempty" db:"prep_time"`
	CookTime  *int `json:"cookTime,omitempty" db:"cook_time"`
	TotalTime *int `json:"totalTime,omitempty" db:"total_time"`

	// Servings
	Servings     int     `json:"servings" db:"servings"`
	ServingsUnit *string `json:"servingsUnit,omitempty" db:"servings_unit"`

	// Nutrition
	Calories                *float64 `json:"calories,omitempty" db:"calories"`
	Protein                 *float64 `json:"protein,omitempty" db:"protein"`
	Fat                     *float64 `json:"fat,omitempty" db:"fat"`
	Carbs                   *float64 `json:"carbs,omitempty" db:"carbs"`
	Fiber                   *float64 `json:"fiber,omitempty" db:"fiber"`
	NutritionCalculatedByAi bool     `json:"nutritionCalculatedByAi" db:"nutrition_calculated_by_ai"`

	// Difficulty
	Difficulty *string `json:"difficulty,omitempty" db:"difficulty"`

	// Sharing
	ShareToken *string `json:"shareToken,omitempty" db:"share_token"`
	IsPublic   bool    `json:"isPublic" db:"is_public"`
	OgImageUrl *string `json:"ogImageUrl,omitempty" db:"og_image_url"`

	// SEO
	MetaDescriptionRu *string `json:"metaDescriptionRu,omitempty" db:"meta_description_ru"`
	MetaDescriptionEn *string `json:"metaDescriptionEn,omitempty" db:"meta_description_en"`

	// Stats
	ViewCount    int    `json:"viewCount" db:"view_count"`
	CopyCount    int    `json:"copyCount" db:"copy_count"`
	CopiedFromId *int64 `json:"copiedFromId,omitempty" db:"copied_from_id"`

	// Category
	CategoryId *int64 `json:"categoryId,omitempty" db:"category_id"`

	// Publishing
	PublishedAt *time.Time `json:"publishedAt,omitempty" db:"published_at"`

	// Timestamps
	CreatedAt time.Time  `json:"createdAt" db:"created_at"`
	UpdatedAt time.Time  `json:"updatedAt" db:"updated_at"`
	DeletedAt *time.Time `json:"-" db:"deleted_at"`

	// Joined fields
	Steps       []Step       `json:"steps,omitempty" db:"-"`
	Ingredients []Ingredient `json:"ingredients,omitempty" db:"-"`
	Tags        []Tag        `json:"tags,omitempty" db:"-"`
	Images      []Image      `json:"images,omitempty" db:"-"`
	Category    *Category    `json:"category,omitempty" db:"-"`
	Book        *Book        `json:"book,omitempty" db:"-"`
}

type RecipeCreate struct {
	BookId        int64   `json:"bookId"`
	UserId        *string `json:"-"`
	Slug          *string `json:"slug,omitempty"`
	TitleRu       string  `json:"titleRu"`
	TitleEn       *string `json:"titleEn,omitempty"`
	DescriptionRu *string `json:"descriptionRu,omitempty"`
	DescriptionEn *string `json:"descriptionEn,omitempty"`
	MainImageUrl  *string `json:"mainImageUrl,omitempty"`
	ExternalUrl   *string `json:"externalUrl,omitempty"`

	PrepTime  *int `json:"prepTime,omitempty"`
	CookTime  *int `json:"cookTime,omitempty"`
	TotalTime *int `json:"totalTime,omitempty"`

	Servings     int     `json:"servings"`
	ServingsUnit *string `json:"servingsUnit,omitempty"`

	Calories *float64 `json:"calories,omitempty"`
	Protein  *float64 `json:"protein,omitempty"`
	Fat      *float64 `json:"fat,omitempty"`
	Carbs    *float64 `json:"carbs,omitempty"`
	Fiber    *float64 `json:"fiber,omitempty"`

	Difficulty *string `json:"difficulty,omitempty"`
	CategoryId *int64  `json:"categoryId,omitempty"`
	Category   *string `json:"category,omitempty"` // category name for auto-creation

	MetaDescriptionRu *string `json:"metaDescriptionRu,omitempty"`
	MetaDescriptionEn *string `json:"metaDescriptionEn,omitempty"`

	Steps       []StepCreate       `json:"steps,omitempty"`
	Ingredients []IngredientCreate `json:"ingredients,omitempty"`
	TagIds      []int64            `json:"tagIds,omitempty"`
}

type RecipeUpdate struct {
	Id            int64   `json:"id"`
	UserId        string  `json:"-"`
	Slug          *string `json:"slug,omitempty"`
	TitleRu       string  `json:"titleRu"`
	TitleEn       *string `json:"titleEn,omitempty"`
	DescriptionRu *string `json:"descriptionRu,omitempty"`
	DescriptionEn *string `json:"descriptionEn,omitempty"`
	MainImageUrl  *string `json:"mainImageUrl,omitempty"`
	ExternalUrl   *string `json:"externalUrl,omitempty"`

	PrepTime  *int `json:"prepTime,omitempty"`
	CookTime  *int `json:"cookTime,omitempty"`
	TotalTime *int `json:"totalTime,omitempty"`

	Servings     int     `json:"servings"`
	ServingsUnit *string `json:"servingsUnit,omitempty"`

	Calories *float64 `json:"calories,omitempty"`
	Protein  *float64 `json:"protein,omitempty"`
	Fat      *float64 `json:"fat,omitempty"`
	Carbs    *float64 `json:"carbs,omitempty"`
	Fiber    *float64 `json:"fiber,omitempty"`

	Difficulty *string `json:"difficulty,omitempty"`
	CategoryId *int64  `json:"categoryId,omitempty"`
	Category   *string `json:"category,omitempty"` // category name for auto-creation

	MetaDescriptionRu *string `json:"metaDescriptionRu,omitempty"`
	MetaDescriptionEn *string `json:"metaDescriptionEn,omitempty"`

	Steps       []StepCreate       `json:"steps,omitempty"`
	Ingredients []IngredientCreate `json:"ingredients,omitempty"`
	TagIds      []int64            `json:"tagIds,omitempty"`
}

// ===== Recipe Step =====

type Step struct {
	Id              int64     `json:"id" db:"id"`
	RecipeId        int64     `json:"recipeId" db:"recipe_id"`
	StepNumber      int       `json:"stepNumber" db:"step_number"`
	InstructionRu   string    `json:"instructionRu" db:"instruction_ru"`
	InstructionEn   *string   `json:"instructionEn,omitempty" db:"instruction_en"`
	ImageUrl        *string   `json:"imageUrl,omitempty" db:"image_url"`
	DurationMinutes *int      `json:"durationMinutes,omitempty" db:"duration_minutes"`
	CreatedAt       time.Time `json:"createdAt" db:"created_at"`
	UpdatedAt       time.Time `json:"updatedAt" db:"updated_at"`
}

type StepCreate struct {
	StepNumber      int     `json:"stepNumber"`
	InstructionRu   string  `json:"instructionRu"`
	InstructionEn   *string `json:"instructionEn,omitempty"`
	ImageUrl        *string `json:"imageUrl,omitempty"`
	DurationMinutes *int    `json:"durationMinutes,omitempty"`
}

// ===== Recipe Ingredient =====

type Ingredient struct {
	Id         int64     `json:"id" db:"id"`
	RecipeId   int64     `json:"recipeId" db:"recipe_id"`
	SortOrder  int       `json:"sortOrder" db:"sort_order"`
	NameRu     string    `json:"nameRu" db:"name_ru"`
	NameEn     *string   `json:"nameEn,omitempty" db:"name_en"`
	Amount     *float64  `json:"amount,omitempty" db:"amount"`
	Unit       *string   `json:"unit,omitempty" db:"unit"`
	Calories   *float64  `json:"calories,omitempty" db:"calories"`
	Protein    *float64  `json:"protein,omitempty" db:"protein"`
	Fat        *float64  `json:"fat,omitempty" db:"fat"`
	Carbs      *float64  `json:"carbs,omitempty" db:"carbs"`
	Fiber      *float64  `json:"fiber,omitempty" db:"fiber"`
	IsOptional bool      `json:"isOptional" db:"is_optional"`
	GroupName  *string   `json:"groupName,omitempty" db:"group_name"`
	CreatedAt  time.Time `json:"createdAt" db:"created_at"`
	UpdatedAt  time.Time `json:"updatedAt" db:"updated_at"`
}

type IngredientCreate struct {
	SortOrder  int      `json:"sortOrder"`
	NameRu     string   `json:"nameRu"`
	NameEn     *string  `json:"nameEn,omitempty"`
	Amount     *float64 `json:"amount,omitempty"`
	Unit       *string  `json:"unit,omitempty"`
	Calories   *float64 `json:"calories,omitempty"`
	Protein    *float64 `json:"protein,omitempty"`
	Fat        *float64 `json:"fat,omitempty"`
	Carbs      *float64 `json:"carbs,omitempty"`
	Fiber      *float64 `json:"fiber,omitempty"`
	IsOptional bool     `json:"isOptional"`
	GroupName  *string  `json:"groupName,omitempty"`
}

// ===== Recipe Image =====

type Image struct {
	Id        int64     `json:"id" db:"id"`
	RecipeId  int64     `json:"recipeId" db:"recipe_id"`
	ImageUrl  string    `json:"imageUrl" db:"image_url"`
	CaptionRu *string   `json:"captionRu,omitempty" db:"caption_ru"`
	CaptionEn *string   `json:"captionEn,omitempty" db:"caption_en"`
	SortOrder int       `json:"sortOrder" db:"sort_order"`
	CreatedAt time.Time `json:"createdAt" db:"created_at"`
}

type ImageCreate struct {
	ImageUrl  string  `json:"imageUrl"`
	CaptionRu *string `json:"captionRu,omitempty"`
	CaptionEn *string `json:"captionEn,omitempty"`
	SortOrder int     `json:"sortOrder"`
}

// ===== List Params & Responses =====

type RecipeListParams struct {
	Page       int
	PerPage    int
	BookId     *int64
	CategoryId *int64
	TagId      *int64
	Search     *string
	UserId     *string
	IsPublic   *bool
}

type RecipeListResponse struct {
	Recipes    []Recipe `json:"recipes"`
	Total      int      `json:"total"`
	Page       int      `json:"page"`
	PerPage    int      `json:"perPage"`
	TotalPages int      `json:"totalPages"`
}

type BookListResponse struct {
	Books      []Book `json:"books"`
	Total      int    `json:"total"`
	Page       int    `json:"page"`
	PerPage    int    `json:"perPage"`
	TotalPages int    `json:"totalPages"`
}

// ===== Add to Diary =====

type AddToDiaryRequest struct {
	RecipeId       int64   `json:"recipeId"`
	Date           *string `json:"date,omitempty"`
	MealType       *string `json:"mealType,omitempty"`
	ServingsAmount float64 `json:"servingsAmount"`
}

// ===== Copy Recipe =====

type CopyRecipeRequest struct {
	TargetBookId int64 `json:"targetBookId"`
}

// ===== Share =====

type ShareResponse struct {
	ShareToken string `json:"shareToken"`
	ShareUrl   string `json:"shareUrl"`
}

// ===== Nutrition Calculation =====

type NutritionCalculationResult struct {
	Calories          *float64              `json:"calories"`
	Protein           *float64              `json:"protein"`
	Fat               *float64              `json:"fat"`
	Carbs             *float64              `json:"carbs"`
	Fiber             *float64              `json:"fiber"`
	IngredientsDetail []IngredientNutrition `json:"ingredientsDetail,omitempty"`
}

type IngredientNutrition struct {
	Name     string   `json:"name"`
	Calories *float64 `json:"calories,omitempty"`
	Protein  *float64 `json:"protein,omitempty"`
	Fat      *float64 `json:"fat,omitempty"`
	Carbs    *float64 `json:"carbs,omitempty"`
	Fiber    *float64 `json:"fiber,omitempty"`
}

// ===== Image Upload =====

type ImageUploadResponse struct {
	Url string `json:"url"`
}
