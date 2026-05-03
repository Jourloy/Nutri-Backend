package recipe

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/charmbracelet/log"
	"github.com/google/uuid"

	"github.com/jourloy/nutri02/internal/ai"
	"github.com/jourloy/nutri02/internal/fit"
	"github.com/jourloy/nutri02/internal/lib"
	"github.com/jourloy/nutri02/internal/product"
	"github.com/jourloy/nutri02/internal/storage"
)

type Service interface {
	// Books
	CreateBook(ctx context.Context, b BookCreate) (*Book, error)
	UpdateBook(ctx context.Context, b BookUpdate) (*Book, error)
	DeleteBook(ctx context.Context, id int64, userId string) error
	GetBookById(ctx context.Context, id int64) (*Book, error)
	GetUserBooks(ctx context.Context, userId string) ([]Book, error)
	GetNutri02Book(ctx context.Context) (*Book, error)
	ShareBook(ctx context.Context, id int64, userId string) (*ShareResponse, error)
	UnshareBook(ctx context.Context, id int64, userId string) error
	GetSharedBook(ctx context.Context, token string) (*Book, error)
	GetSharedBookRecipes(ctx context.Context, token string, params RecipeListParams) (*RecipeListResponse, error)

	// Categories
	CreateCategory(ctx context.Context, c CategoryCreate) (*Category, error)
	UpdateCategory(ctx context.Context, c CategoryUpdate) (*Category, error)
	DeleteCategory(ctx context.Context, id int64, userId string) error
	GetCategories(ctx context.Context, userId *string) ([]Category, error)

	// Tags
	CreateTag(ctx context.Context, t TagCreate) (*Tag, error)
	UpdateTag(ctx context.Context, t TagUpdate) (*Tag, error)
	DeleteTag(ctx context.Context, id int64, userId string) error
	GetTags(ctx context.Context, userId *string) ([]Tag, error)
	CreateSystemTag(ctx context.Context, t TagCreate) (*Tag, error)
	UpdateSystemTag(ctx context.Context, t TagUpdate) (*Tag, error)
	DeleteSystemTag(ctx context.Context, id int64) error

	// Recipes
	CreateRecipe(ctx context.Context, r RecipeCreate) (*Recipe, error)
	UpdateRecipe(ctx context.Context, r RecipeUpdate) (*Recipe, error)
	DeleteRecipe(ctx context.Context, id int64, userId string) error
	GetRecipeById(ctx context.Context, id int64) (*Recipe, error)
	GetRecipes(ctx context.Context, params RecipeListParams) (*RecipeListResponse, error)
	ShareRecipe(ctx context.Context, id int64, userId string) (*ShareResponse, error)
	UnshareRecipe(ctx context.Context, id int64, userId string) error
	GetSharedRecipe(ctx context.Context, token string) (*Recipe, error)
	CopyRecipe(ctx context.Context, id int64, targetBookId int64, userId string) (*Recipe, error)
	AddToDiary(ctx context.Context, req AddToDiaryRequest, userId string, timezone string) ([]product.Product, error)

	// Nutri02 Recipes (Public)
	GetNutri02Recipes(ctx context.Context, params RecipeListParams) (*RecipeListResponse, error)
	GetNutri02RecipeBySlug(ctx context.Context, slug string) (*Recipe, error)
	GetNutri02RecipeById(ctx context.Context, id int64) (*Recipe, error)

	// Admin
	CreateNutri02Recipe(ctx context.Context, r RecipeCreate) (*Recipe, error)
	UpdateNutri02Recipe(ctx context.Context, r RecipeUpdate) (*Recipe, error)
	DeleteNutri02Recipe(ctx context.Context, id int64) error
	CreateSystemCategory(ctx context.Context, c CategoryCreate) (*Category, error)

	// Image Upload
	UploadImage(ctx context.Context, imageData []byte, filename string) (string, error)

	// AI Nutrition Calculation
	CalculateNutrition(ctx context.Context, recipeId int64, userId string) (*NutritionCalculationResult, error)
}

type service struct {
	repo           Repository
	productService product.Service
	fitService     fit.Service
	aiService      ai.Service
	storage        storage.Service
	logger         *log.Logger
}

func NewService(storageService storage.Service) (Service, error) {
	logger := log.NewWithOptions(os.Stderr, log.Options{
		Prefix: "[recipe-svc]",
		Level:  log.DebugLevel,
	})

	repo := NewRepository()
	if storageService == nil {
		return nil, fmt.Errorf("storage service is required")
	}

	// Initialize AI service (optional - may fail if API keys not configured)
	var aiSvc ai.Service
	aiRepo := ai.NewRepository()
	aiSvc, err := ai.NewServiceFromConfig(aiRepo)
	if err != nil {
		logger.Warn("AI service not available for nutrition calculation", "error", err)
	}

	return &service{
		repo:           repo,
		productService: product.NewService(),
		fitService:     fit.NewService(),
		aiService:      aiSvc,
		storage:        storageService,
		logger:         logger,
	}, nil
}

func NewServiceFromConfig() (Service, error) {
	storageService, err := storage.NewS3ServiceFromConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to create storage service: %w", err)
	}
	return NewService(storageService)
}

// ===== Books =====

func (s *service) CreateBook(ctx context.Context, b BookCreate) (*Book, error) {
	// Validate name
	if err := validateBookName(b.Name); err != nil {
		return nil, err
	}
	return s.repo.CreateBook(ctx, b)
}

func (s *service) UpdateBook(ctx context.Context, b BookUpdate) (*Book, error) {
	// Validate name
	if err := validateBookName(b.Name); err != nil {
		return nil, err
	}
	return s.repo.UpdateBook(ctx, b)
}

func (s *service) DeleteBook(ctx context.Context, id int64, userId string) error {
	return s.repo.DeleteBook(ctx, id, userId)
}

func (s *service) GetBookById(ctx context.Context, id int64) (*Book, error) {
	return s.repo.GetBookById(ctx, id)
}

func (s *service) GetUserBooks(ctx context.Context, userId string) ([]Book, error) {
	return s.repo.GetUserBooks(ctx, userId)
}

func (s *service) GetNutri02Book(ctx context.Context) (*Book, error) {
	return s.repo.GetNutri02Book(ctx)
}

func (s *service) ShareBook(ctx context.Context, id int64, userId string) (*ShareResponse, error) {
	book, err := s.repo.GetBookById(ctx, id)
	if err != nil {
		return nil, err
	}

	// Check ownership
	if book.UserId == nil || *book.UserId != userId {
		return nil, fmt.Errorf("access denied")
	}

	// Generate token if not exists
	token := book.ShareToken
	if token == nil {
		newToken := uuid.New().String()
		token = &newToken
		if err := s.repo.SetBookShareToken(ctx, id, newToken); err != nil {
			return nil, err
		}
	}

	return &ShareResponse{
		ShareToken: *token,
		ShareUrl:   fmt.Sprintf("%s/shared/book/%s", lib.Config.FrontURL, *token),
	}, nil
}

func (s *service) UnshareBook(ctx context.Context, id int64, userId string) error {
	book, err := s.repo.GetBookById(ctx, id)
	if err != nil {
		return err
	}

	// Check ownership
	if book.UserId == nil || *book.UserId != userId {
		return fmt.Errorf("access denied")
	}

	return s.repo.ClearBookShareToken(ctx, id)
}

func (s *service) GetSharedBook(ctx context.Context, token string) (*Book, error) {
	return s.repo.GetBookByShareToken(ctx, token)
}

func (s *service) GetSharedBookRecipes(ctx context.Context, token string, params RecipeListParams) (*RecipeListResponse, error) {
	book, err := s.repo.GetBookByShareToken(ctx, token)
	if err != nil {
		return nil, err
	}

	params.BookId = &book.Id
	response, err := s.repo.GetRecipes(ctx, params)
	if err != nil {
		return nil, err
	}

	// Load relations for each recipe
	for i := range response.Recipes {
		s.loadRecipeRelationsInPlace(ctx, &response.Recipes[i])
	}

	return response, nil
}

// ===== Categories =====

func (s *service) CreateCategory(ctx context.Context, c CategoryCreate) (*Category, error) {
	c.CategoryType = "user"
	return s.repo.CreateCategory(ctx, c)
}

func (s *service) UpdateCategory(ctx context.Context, c CategoryUpdate) (*Category, error) {
	return s.repo.UpdateCategory(ctx, c)
}

func (s *service) DeleteCategory(ctx context.Context, id int64, userId string) error {
	return s.repo.DeleteCategory(ctx, id, userId)
}

func (s *service) GetCategories(ctx context.Context, userId *string) ([]Category, error) {
	return s.repo.GetCategories(ctx, userId)
}

// getOrCreateCategory finds existing category by name or creates a new user category
func (s *service) getOrCreateCategory(ctx context.Context, name string, userId *string) (*Category, error) {
	// Try find existing category
	cat, err := s.repo.GetCategoryByName(ctx, name, userId)
	if err == nil && cat != nil {
		return cat, nil
	}

	// Create new user category
	return s.repo.CreateCategory(ctx, CategoryCreate{
		UserId:       userId,
		NameRu:       name,
		CategoryType: "user",
	})
}

// ===== Tags =====

func (s *service) CreateTag(ctx context.Context, t TagCreate) (*Tag, error) {
	return s.repo.CreateTag(ctx, t)
}

func (s *service) UpdateTag(ctx context.Context, t TagUpdate) (*Tag, error) {
	return s.repo.UpdateTag(ctx, t)
}

func (s *service) DeleteTag(ctx context.Context, id int64, userId string) error {
	return s.repo.DeleteTag(ctx, id, userId)
}

func (s *service) GetTags(ctx context.Context, userId *string) ([]Tag, error) {
	return s.repo.GetTags(ctx, userId)
}

func (s *service) CreateSystemTag(ctx context.Context, t TagCreate) (*Tag, error) {
	t.UserId = nil
	return s.repo.CreateSystemTag(ctx, t)
}

func (s *service) UpdateSystemTag(ctx context.Context, t TagUpdate) (*Tag, error) {
	return s.repo.UpdateSystemTag(ctx, t)
}

func (s *service) DeleteSystemTag(ctx context.Context, id int64) error {
	return s.repo.DeleteSystemTag(ctx, id)
}

// ===== Recipes =====

func (s *service) CreateRecipe(ctx context.Context, r RecipeCreate) (*Recipe, error) {
	// Validate book ownership
	book, err := s.repo.GetBookById(ctx, r.BookId)
	if err != nil {
		return nil, err
	}
	if book.UserId == nil || *book.UserId != *r.UserId {
		return nil, fmt.Errorf("access denied")
	}

	// Resolve category by name if provided
	if r.Category != nil && *r.Category != "" && r.CategoryId == nil {
		cat, err := s.getOrCreateCategory(ctx, *r.Category, r.UserId)
		if err != nil {
			s.logger.Error("failed to get or create category", "name", *r.Category, "error", err)
		} else {
			r.CategoryId = &cat.Id
		}
	}

	recipe, err := s.repo.CreateRecipe(ctx, r)
	if err != nil {
		return nil, err
	}

	// Create steps
	if len(r.Steps) > 0 {
		if err := s.repo.CreateSteps(ctx, recipe.Id, r.Steps); err != nil {
			s.logger.Error("failed to create steps", "recipeId", recipe.Id, "error", err)
		}
	}

	// Create ingredients
	if len(r.Ingredients) > 0 {
		if err := s.repo.CreateIngredients(ctx, recipe.Id, r.Ingredients); err != nil {
			s.logger.Error("failed to create ingredients", "recipeId", recipe.Id, "error", err)
		}
	}

	// Set tags
	if len(r.TagIds) > 0 {
		if err := s.repo.SetRecipeTags(ctx, recipe.Id, r.TagIds); err != nil {
			s.logger.Error("failed to set recipe tags", "recipeId", recipe.Id, "error", err)
		}
	}

	return s.loadRecipeWithRelations(ctx, recipe)
}

func (s *service) UpdateRecipe(ctx context.Context, r RecipeUpdate) (*Recipe, error) {
	// Resolve category by name if provided
	if r.Category != nil && *r.Category != "" && r.CategoryId == nil {
		cat, err := s.getOrCreateCategory(ctx, *r.Category, &r.UserId)
		if err != nil {
			s.logger.Error("failed to get or create category", "name", *r.Category, "error", err)
		} else {
			r.CategoryId = &cat.Id
		}
	}

	recipe, err := s.repo.UpdateRecipe(ctx, r)
	if err != nil {
		return nil, err
	}

	// Update steps
	if err := s.repo.UpdateSteps(ctx, recipe.Id, r.Steps); err != nil {
		s.logger.Error("failed to update steps", "recipeId", recipe.Id, "error", err)
	}

	// Update ingredients
	if err := s.repo.UpdateIngredients(ctx, recipe.Id, r.Ingredients); err != nil {
		s.logger.Error("failed to update ingredients", "recipeId", recipe.Id, "error", err)
	}

	// Update tags
	if err := s.repo.SetRecipeTags(ctx, recipe.Id, r.TagIds); err != nil {
		s.logger.Error("failed to update recipe tags", "recipeId", recipe.Id, "error", err)
	}

	return s.loadRecipeWithRelations(ctx, recipe)
}

func (s *service) DeleteRecipe(ctx context.Context, id int64, userId string) error {
	return s.repo.DeleteRecipe(ctx, id, userId)
}

func (s *service) GetRecipeById(ctx context.Context, id int64) (*Recipe, error) {
	recipe, err := s.repo.GetRecipeById(ctx, id)
	if err != nil {
		return nil, err
	}
	return s.loadRecipeWithRelations(ctx, recipe)
}

func (s *service) GetRecipes(ctx context.Context, params RecipeListParams) (*RecipeListResponse, error) {
	response, err := s.repo.GetRecipes(ctx, params)
	if err != nil {
		return nil, err
	}

	// Load relations for each recipe
	for i := range response.Recipes {
		s.loadRecipeRelationsInPlace(ctx, &response.Recipes[i])
	}

	return response, nil
}

func (s *service) ShareRecipe(ctx context.Context, id int64, userId string) (*ShareResponse, error) {
	recipe, err := s.repo.GetRecipeById(ctx, id)
	if err != nil {
		return nil, err
	}

	// Check ownership
	if recipe.UserId == nil || *recipe.UserId != userId {
		return nil, fmt.Errorf("access denied")
	}

	// Generate token if not exists
	token := recipe.ShareToken
	if token == nil {
		newToken := uuid.New().String()
		token = &newToken
		if err := s.repo.SetRecipeShareToken(ctx, id, newToken); err != nil {
			return nil, err
		}
	}

	return &ShareResponse{
		ShareToken: *token,
		ShareUrl:   fmt.Sprintf("%s/shared/recipe/%s", lib.Config.FrontURL, *token),
	}, nil
}

func (s *service) UnshareRecipe(ctx context.Context, id int64, userId string) error {
	recipe, err := s.repo.GetRecipeById(ctx, id)
	if err != nil {
		return err
	}

	// Check ownership
	if recipe.UserId == nil || *recipe.UserId != userId {
		return fmt.Errorf("access denied")
	}

	return s.repo.ClearRecipeShareToken(ctx, id)
}

func (s *service) GetSharedRecipe(ctx context.Context, token string) (*Recipe, error) {
	recipe, err := s.repo.GetRecipeByShareToken(ctx, token)
	if err != nil {
		return nil, err
	}

	// Increment view count
	s.repo.IncrementViewCount(ctx, recipe.Id)

	return s.loadRecipeWithRelations(ctx, recipe)
}

func (s *service) CopyRecipe(ctx context.Context, id int64, targetBookId int64, userId string) (*Recipe, error) {
	// Get original recipe
	original, err := s.repo.GetRecipeById(ctx, id)
	if err != nil {
		return nil, err
	}

	// Check if recipe is public or owned by user
	isOwner := original.UserId != nil && *original.UserId == userId
	if !original.IsPublic && !isOwner {
		return nil, fmt.Errorf("access denied")
	}

	// Validate target book ownership
	targetBook, err := s.repo.GetBookById(ctx, targetBookId)
	if err != nil {
		return nil, err
	}
	if targetBook.UserId == nil || *targetBook.UserId != userId {
		return nil, fmt.Errorf("access denied to target book")
	}

	// Load relations
	original, err = s.loadRecipeWithRelations(ctx, original)
	if err != nil {
		return nil, err
	}

	// Create copy
	rc := RecipeCreate{
		BookId:        targetBookId,
		UserId:        &userId,
		TitleRu:       original.TitleRu,
		TitleEn:       original.TitleEn,
		DescriptionRu: original.DescriptionRu,
		DescriptionEn: original.DescriptionEn,
		MainImageUrl:  original.MainImageUrl,
		PrepTime:      original.PrepTime,
		CookTime:      original.CookTime,
		TotalTime:     original.TotalTime,
		Servings:      original.Servings,
		ServingsUnit:  original.ServingsUnit,
		Calories:      original.Calories,
		Protein:       original.Protein,
		Fat:           original.Fat,
		Carbs:         original.Carbs,
		Fiber:         original.Fiber,
		Difficulty:    original.Difficulty,
		CategoryId:    original.CategoryId,
	}

	// Convert steps
	for _, step := range original.Steps {
		rc.Steps = append(rc.Steps, StepCreate{
			StepNumber:      step.StepNumber,
			InstructionRu:   step.InstructionRu,
			InstructionEn:   step.InstructionEn,
			ImageUrl:        step.ImageUrl,
			DurationMinutes: step.DurationMinutes,
		})
	}

	// Convert ingredients
	for _, ing := range original.Ingredients {
		rc.Ingredients = append(rc.Ingredients, IngredientCreate{
			SortOrder:  ing.SortOrder,
			NameRu:     ing.NameRu,
			NameEn:     ing.NameEn,
			Amount:     ing.Amount,
			Unit:       ing.Unit,
			Calories:   ing.Calories,
			Protein:    ing.Protein,
			Fat:        ing.Fat,
			Carbs:      ing.Carbs,
			Fiber:      ing.Fiber,
			IsOptional: ing.IsOptional,
			GroupName:  ing.GroupName,
		})
	}

	// Convert tag ids
	for _, tag := range original.Tags {
		rc.TagIds = append(rc.TagIds, tag.Id)
	}

	newRecipe, err := s.repo.CreateRecipe(ctx, rc)
	if err != nil {
		return nil, err
	}

	// Create steps and ingredients
	if len(rc.Steps) > 0 {
		s.repo.CreateSteps(ctx, newRecipe.Id, rc.Steps)
	}
	if len(rc.Ingredients) > 0 {
		s.repo.CreateIngredients(ctx, newRecipe.Id, rc.Ingredients)
	}
	if len(rc.TagIds) > 0 {
		s.repo.SetRecipeTags(ctx, newRecipe.Id, rc.TagIds)
	}

	// Increment copy count on original
	s.repo.IncrementCopyCount(ctx, id)

	return s.loadRecipeWithRelations(ctx, newRecipe)
}

func (s *service) AddToDiary(ctx context.Context, req AddToDiaryRequest, userId string, timezone string) ([]product.Product, error) {
	recipe, err := s.repo.GetRecipeById(ctx, req.RecipeId)
	if err != nil {
		return nil, err
	}

	// Check access - must be owner or public
	isOwner := recipe.UserId != nil && *recipe.UserId == userId
	if !recipe.IsPublic && !isOwner {
		return nil, fmt.Errorf("access denied")
	}

	// Load ingredients
	ingredients, err := s.repo.GetIngredientsByRecipeId(ctx, req.RecipeId)
	if err != nil {
		return nil, err
	}

	if len(ingredients) == 0 {
		return nil, fmt.Errorf("recipe has no ingredients")
	}

	// Calculate multiplier based on servings
	servingsMultiplier := req.ServingsAmount / float64(recipe.Servings)
	if servingsMultiplier <= 0 {
		servingsMultiplier = 1
	}

	// Parse date
	var loggedAt *time.Time
	if req.Date != nil && *req.Date != "" {
		parsed, err := time.Parse("2006-01-02", *req.Date)
		if err == nil {
			loggedAt = &parsed
		}
	}

	// Get today for product service
	today := time.Now().Format("2006-01-02")
	if timezone != "" {
		loc, err := time.LoadLocation(timezone)
		if err == nil {
			today = time.Now().In(loc).Format("2006-01-02")
		}
	}

	var products []product.Product

	for _, ing := range ingredients {
		// Skip optional ingredients without nutrition info
		if ing.IsOptional && ing.Calories == nil {
			continue
		}

		// Calculate amounts based on multiplier
		amount := int64(100)
		if ing.Amount != nil {
			amount = int64(*ing.Amount * servingsMultiplier)
		}

		calories := 0.0
		if ing.Calories != nil {
			calories = *ing.Calories * servingsMultiplier
		}

		protein := 0.0
		if ing.Protein != nil {
			protein = *ing.Protein * servingsMultiplier
		}

		fat := 0.0
		if ing.Fat != nil {
			fat = *ing.Fat * servingsMultiplier
		}

		carbs := 0.0
		if ing.Carbs != nil {
			carbs = *ing.Carbs * servingsMultiplier
		}

		unit := "g"
		if ing.Unit != nil {
			unit = *ing.Unit
		}

		pc := product.ProductCreate{
			Name:          ing.NameRu,
			Amount:        amount,
			Unit:          unit,
			Calories:      calories,
			Protein:       protein,
			Fat:           fat,
			Carbs:         carbs,
			Fiber:         ing.Fiber,
			BasicCalories: calories / servingsMultiplier,
			BasicProtein:  protein / servingsMultiplier,
			BasicFat:      fat / servingsMultiplier,
			BasicCarbs:    carbs / servingsMultiplier,
			BasicFiber:    ing.Fiber,
			MealType:      req.MealType,
			LoggedAt:      loggedAt,
			UserId:        userId,
		}

		created, err := s.productService.CreateProduct(ctx, pc, today)
		if err != nil {
			s.logger.Error("failed to create product", "ingredient", ing.NameRu, "error", err)
			continue
		}

		if len(created) > 0 {
			products = append(products, created...)
		}
	}

	return products, nil
}

// ===== Nutri02 Recipes (Public) =====

func (s *service) GetNutri02Recipes(ctx context.Context, params RecipeListParams) (*RecipeListResponse, error) {
	// Get Nutri02 book
	book, err := s.repo.GetNutri02Book(ctx)
	if err != nil {
		if err == sql.ErrNoRows {
			return &RecipeListResponse{
				Recipes:    []Recipe{},
				Total:      0,
				Page:       1,
				PerPage:    20,
				TotalPages: 0,
			}, nil
		}
		return nil, err
	}

	params.BookId = &book.Id
	isPublic := true
	params.IsPublic = &isPublic

	response, err := s.repo.GetRecipes(ctx, params)
	if err != nil {
		return nil, err
	}

	// Load relations for each recipe
	for i := range response.Recipes {
		s.loadRecipeRelationsInPlace(ctx, &response.Recipes[i])
	}

	return response, nil
}

func (s *service) GetNutri02RecipeBySlug(ctx context.Context, slug string) (*Recipe, error) {
	recipe, err := s.repo.GetRecipeBySlug(ctx, slug)
	if err != nil {
		return nil, err
	}

	// Verify it's a Nutri02 recipe
	book, err := s.repo.GetNutri02Book(ctx)
	if err != nil {
		return nil, err
	}

	if recipe.BookId != book.Id {
		return nil, fmt.Errorf("recipe not found")
	}

	// Increment view count
	s.repo.IncrementViewCount(ctx, recipe.Id)

	return s.loadRecipeWithRelations(ctx, recipe)
}

func (s *service) GetNutri02RecipeById(ctx context.Context, id int64) (*Recipe, error) {
	recipe, err := s.repo.GetRecipeById(ctx, id)
	if err != nil {
		return nil, err
	}

	book, err := s.repo.GetNutri02Book(ctx)
	if err != nil {
		return nil, err
	}

	if recipe.BookId != book.Id {
		return nil, fmt.Errorf("recipe not found")
	}

	return s.loadRecipeWithRelations(ctx, recipe)
}

// ===== Admin =====

func (s *service) CreateNutri02Recipe(ctx context.Context, r RecipeCreate) (*Recipe, error) {
	// Get Nutri02 book
	book, err := s.repo.GetNutri02Book(ctx)
	if err != nil {
		return nil, err
	}

	r.BookId = book.Id
	r.UserId = nil // Nutri02 recipes have no user

	recipe, err := s.repo.CreateRecipe(ctx, r)
	if err != nil {
		return nil, err
	}

	// Create steps
	if len(r.Steps) > 0 {
		if err := s.repo.CreateSteps(ctx, recipe.Id, r.Steps); err != nil {
			s.logger.Error("failed to create steps", "recipeId", recipe.Id, "error", err)
		}
	}

	// Create ingredients
	if len(r.Ingredients) > 0 {
		if err := s.repo.CreateIngredients(ctx, recipe.Id, r.Ingredients); err != nil {
			s.logger.Error("failed to create ingredients", "recipeId", recipe.Id, "error", err)
		}
	}

	// Set tags
	if len(r.TagIds) > 0 {
		if err := s.repo.SetRecipeTags(ctx, recipe.Id, r.TagIds); err != nil {
			s.logger.Error("failed to set recipe tags", "recipeId", recipe.Id, "error", err)
		}
	}

	// Make public
	newToken := uuid.New().String()
	s.repo.SetRecipeShareToken(ctx, recipe.Id, newToken)

	return s.loadRecipeWithRelations(ctx, recipe)
}

func (s *service) UpdateNutri02Recipe(ctx context.Context, r RecipeUpdate) (*Recipe, error) {
	// Verify it's a Nutri02 recipe
	existing, err := s.repo.GetRecipeById(ctx, r.Id)
	if err != nil {
		return nil, err
	}

	book, err := s.repo.GetNutri02Book(ctx)
	if err != nil {
		return nil, err
	}

	if existing.BookId != book.Id {
		return nil, fmt.Errorf("not a Nutri02 recipe")
	}

	// Resolve system category by name if provided.
	if r.Category != nil && *r.Category != "" && r.CategoryId == nil {
		cat, err := s.repo.GetCategoryByName(ctx, *r.Category, nil)
		if err != nil {
			s.logger.Warn("failed to resolve system category", "name", *r.Category, "error", err)
		} else {
			r.CategoryId = &cat.Id
		}
	}

	recipe, err := s.repo.UpdateNutri02Recipe(ctx, r)
	if err != nil {
		return nil, err
	}

	if err := s.repo.UpdateSteps(ctx, recipe.Id, r.Steps); err != nil {
		s.logger.Error("failed to update steps", "recipeId", recipe.Id, "error", err)
	}

	if err := s.repo.UpdateIngredients(ctx, recipe.Id, r.Ingredients); err != nil {
		s.logger.Error("failed to update ingredients", "recipeId", recipe.Id, "error", err)
	}

	if err := s.repo.SetRecipeTags(ctx, recipe.Id, r.TagIds); err != nil {
		s.logger.Error("failed to update recipe tags", "recipeId", recipe.Id, "error", err)
	}

	if recipe.ShareToken == nil {
		newToken := uuid.New().String()
		if err := s.repo.SetRecipeShareToken(ctx, recipe.Id, newToken); err != nil {
			s.logger.Warn("failed to set share token for Nutri02 recipe", "recipeId", recipe.Id, "error", err)
		}
	}

	return s.loadRecipeWithRelations(ctx, recipe)
}

func (s *service) DeleteNutri02Recipe(ctx context.Context, id int64) error {
	// Verify it's a Nutri02 recipe
	recipe, err := s.repo.GetRecipeById(ctx, id)
	if err != nil {
		return err
	}

	book, err := s.repo.GetNutri02Book(ctx)
	if err != nil {
		return err
	}

	if recipe.BookId != book.Id {
		return fmt.Errorf("not a Nutri02 recipe")
	}

	return s.repo.DeleteNutri02Recipe(ctx, id)
}

func (s *service) CreateSystemCategory(ctx context.Context, c CategoryCreate) (*Category, error) {
	c.CategoryType = "system"
	c.UserId = nil
	return s.repo.CreateCategory(ctx, c)
}

// ===== Image Upload =====

func (s *service) UploadImage(ctx context.Context, imageData []byte, filename string) (string, error) {
	// Generate unique filename
	ext := ""
	if idx := strings.LastIndex(filename, "."); idx != -1 {
		ext = filename[idx:]
	}
	if ext == "" {
		ext = ".jpg"
	}

	objectName := fmt.Sprintf("%s/%s%s", time.Now().Format("2006/01"), uuid.New().String(), ext)

	// Determine content type
	contentType := "image/jpeg"
	switch strings.ToLower(ext) {
	case ".png":
		contentType = "image/png"
	case ".gif":
		contentType = "image/gif"
	case ".webp":
		contentType = "image/webp"
	}

	imageURL, err := s.storage.Upload(ctx, storage.FolderRecipe, objectName, imageData, contentType)
	if err != nil {
		return "", fmt.Errorf("failed to upload image: %w", err)
	}

	return imageURL, nil
}

// ===== AI Nutrition Calculation =====

func (s *service) CalculateNutrition(ctx context.Context, recipeId int64, userId string) (*NutritionCalculationResult, error) {
	if s.aiService == nil {
		return nil, fmt.Errorf("AI service is not available")
	}

	// Get recipe
	recipe, err := s.repo.GetRecipeById(ctx, recipeId)
	if err != nil {
		return nil, err
	}

	// Check access
	isOwner := recipe.UserId != nil && *recipe.UserId == userId
	isNutri02Recipe := recipe.UserId == nil // Nutri02 recipes have no userId
	if !isOwner && !isNutri02Recipe {
		return nil, fmt.Errorf("access denied")
	}

	// Get ingredients
	ingredients, err := s.repo.GetIngredientsByRecipeId(ctx, recipeId)
	if err != nil {
		return nil, err
	}

	if len(ingredients) == 0 {
		return nil, fmt.Errorf("recipe has no ingredients")
	}

	result := &NutritionCalculationResult{
		IngredientsDetail: make([]IngredientNutrition, 0, len(ingredients)),
	}

	var totalCalories, totalProtein, totalFat, totalCarbs, totalFiber float64

	for _, ing := range ingredients {
		// Build description for AI
		description := ""
		if ing.Amount != nil {
			description = fmt.Sprintf("%.1f", *ing.Amount)
			if ing.Unit != nil {
				description += " " + *ing.Unit
			}
		}

		weight := 100.0
		if ing.Amount != nil {
			weight = *ing.Amount
		}

		// Call AI to analyze this ingredient
		analysisResult, err := s.aiService.AnalyzeFoodByText(ctx, userId, ing.NameRu, description, weight, "ru")
		if err != nil {
			s.logger.Warn("failed to analyze ingredient", "ingredient", ing.NameRu, "error", err)
			// Add ingredient without nutrition data
			result.IngredientsDetail = append(result.IngredientsDetail, IngredientNutrition{
				Name: ing.NameRu,
			})
			continue
		}

		// Update ingredient with calculated nutrition
		if analysisResult.Calories > 0 {
			calories := analysisResult.Calories
			protein := analysisResult.Protein
			fat := analysisResult.Fat
			carbs := analysisResult.Carbs
			fiber := 0.0
			if analysisResult.Fiber != nil {
				fiber = *analysisResult.Fiber
			}

			// Update ingredient in database
			s.repo.UpdateIngredientNutrition(ctx, ing.Id, calories, protein, fat, carbs, fiber)

			totalCalories += calories
			totalProtein += protein
			totalFat += fat
			totalCarbs += carbs
			totalFiber += fiber

			result.IngredientsDetail = append(result.IngredientsDetail, IngredientNutrition{
				Name:     ing.NameRu,
				Calories: &calories,
				Protein:  &protein,
				Fat:      &fat,
				Carbs:    &carbs,
				Fiber:    &fiber,
			})
		} else {
			result.IngredientsDetail = append(result.IngredientsDetail, IngredientNutrition{
				Name: ing.NameRu,
			})
		}
	}

	// Calculate per-serving nutrition
	servings := float64(recipe.Servings)
	if servings <= 0 {
		servings = 1
	}

	perServingCalories := totalCalories / servings
	perServingProtein := totalProtein / servings
	perServingFat := totalFat / servings
	perServingCarbs := totalCarbs / servings
	perServingFiber := totalFiber / servings

	result.Calories = &perServingCalories
	result.Protein = &perServingProtein
	result.Fat = &perServingFat
	result.Carbs = &perServingCarbs
	result.Fiber = &perServingFiber

	// Update recipe nutrition in database
	s.repo.UpdateRecipeNutrition(ctx, recipeId, perServingCalories, perServingProtein, perServingFat, perServingCarbs, perServingFiber)

	return result, nil
}

// ===== Helpers =====

func (s *service) loadRecipeWithRelations(ctx context.Context, recipe *Recipe) (*Recipe, error) {
	s.loadRecipeRelationsInPlace(ctx, recipe)
	return recipe, nil
}

func (s *service) loadRecipeRelationsInPlace(ctx context.Context, recipe *Recipe) {
	// Load steps
	steps, err := s.repo.GetStepsByRecipeId(ctx, recipe.Id)
	if err == nil {
		recipe.Steps = steps
	} else {
		recipe.Steps = []Step{}
	}

	// Load ingredients
	ingredients, err := s.repo.GetIngredientsByRecipeId(ctx, recipe.Id)
	if err == nil {
		recipe.Ingredients = ingredients
	} else {
		recipe.Ingredients = []Ingredient{}
	}

	// Load tags
	tags, err := s.repo.GetTagsByRecipeId(ctx, recipe.Id)
	if err == nil {
		recipe.Tags = tags
	} else {
		recipe.Tags = []Tag{}
	}

	// Load images
	images, err := s.repo.GetImagesByRecipeId(ctx, recipe.Id)
	if err == nil {
		recipe.Images = images
	} else {
		recipe.Images = []Image{}
	}

	// Load category
	if recipe.CategoryId != nil {
		cat, err := s.repo.GetCategoryById(ctx, *recipe.CategoryId)
		if err == nil {
			recipe.Category = cat
		}
	}

	// Load book
	book, err := s.repo.GetBookById(ctx, recipe.BookId)
	if err == nil {
		recipe.Book = book
	}
}

func validateBookName(name string) error {
	if len(name) == 0 {
		return fmt.Errorf("book name is required")
	}
	if len(name) > 256 {
		return fmt.Errorf("book name must be at most 256 characters")
	}

	// Allow letters (any language), numbers, spaces, and hyphens
	re := regexp.MustCompile(`^[\p{L}\p{N}\s\-]+$`)
	if !re.MatchString(name) {
		return fmt.Errorf("book name can only contain letters, numbers, spaces, and hyphens")
	}

	return nil
}
