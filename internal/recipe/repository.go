package recipe

import (
	"context"
	"fmt"
	"strings"

	"github.com/jmoiron/sqlx"
	"github.com/jourloy/nutri-backend/internal/database"
)

type Repository interface {
	// Books
	CreateBook(ctx context.Context, b BookCreate) (*Book, error)
	UpdateBook(ctx context.Context, b BookUpdate) (*Book, error)
	DeleteBook(ctx context.Context, id int64, userId string) error
	GetBookById(ctx context.Context, id int64) (*Book, error)
	GetBookByShareToken(ctx context.Context, token string) (*Book, error)
	GetUserBooks(ctx context.Context, userId string) ([]Book, error)
	GetNutriBook(ctx context.Context) (*Book, error)
	SetBookShareToken(ctx context.Context, id int64, token string) error
	ClearBookShareToken(ctx context.Context, id int64) error

	// Categories
	CreateCategory(ctx context.Context, c CategoryCreate) (*Category, error)
	UpdateCategory(ctx context.Context, c CategoryUpdate) (*Category, error)
	DeleteCategory(ctx context.Context, id int64, userId string) error
	GetCategoryById(ctx context.Context, id int64) (*Category, error)
	GetCategoryByName(ctx context.Context, name string, userId *string) (*Category, error)
	GetCategories(ctx context.Context, userId *string) ([]Category, error)

	// Tags
	CreateTag(ctx context.Context, t TagCreate) (*Tag, error)
	UpdateTag(ctx context.Context, t TagUpdate) (*Tag, error)
	DeleteTag(ctx context.Context, id int64, userId string) error
	GetTagById(ctx context.Context, id int64) (*Tag, error)
	GetTags(ctx context.Context, userId *string) ([]Tag, error)
	GetTagsByRecipeId(ctx context.Context, recipeId int64) ([]Tag, error)
	SetRecipeTags(ctx context.Context, recipeId int64, tagIds []int64) error

	// Recipes
	CreateRecipe(ctx context.Context, r RecipeCreate) (*Recipe, error)
	UpdateRecipe(ctx context.Context, r RecipeUpdate) (*Recipe, error)
	DeleteRecipe(ctx context.Context, id int64, userId string) error
	GetRecipeById(ctx context.Context, id int64) (*Recipe, error)
	GetRecipeBySlug(ctx context.Context, slug string) (*Recipe, error)
	GetRecipeByShareToken(ctx context.Context, token string) (*Recipe, error)
	GetRecipes(ctx context.Context, params RecipeListParams) (*RecipeListResponse, error)
	SetRecipeShareToken(ctx context.Context, id int64, token string) error
	ClearRecipeShareToken(ctx context.Context, id int64) error
	IncrementViewCount(ctx context.Context, id int64) error
	IncrementCopyCount(ctx context.Context, id int64) error
	UpdateNutrition(ctx context.Context, id int64, calories, protein, fat, carbs, fiber *float64, byAi bool) error

	// Steps
	CreateSteps(ctx context.Context, recipeId int64, steps []StepCreate) error
	UpdateSteps(ctx context.Context, recipeId int64, steps []StepCreate) error
	GetStepsByRecipeId(ctx context.Context, recipeId int64) ([]Step, error)
	DeleteStepsByRecipeId(ctx context.Context, recipeId int64) error

	// Ingredients
	CreateIngredients(ctx context.Context, recipeId int64, ingredients []IngredientCreate) error
	UpdateIngredients(ctx context.Context, recipeId int64, ingredients []IngredientCreate) error
	GetIngredientsByRecipeId(ctx context.Context, recipeId int64) ([]Ingredient, error)
	DeleteIngredientsByRecipeId(ctx context.Context, recipeId int64) error
	UpdateIngredientNutrition(ctx context.Context, id int64, calories, protein, fat, carbs, fiber float64) error

	// Nutrition
	UpdateRecipeNutrition(ctx context.Context, id int64, calories, protein, fat, carbs, fiber float64) error

	// Images
	CreateImage(ctx context.Context, recipeId int64, img ImageCreate) (*Image, error)
	DeleteImage(ctx context.Context, id int64, recipeId int64) error
	GetImagesByRecipeId(ctx context.Context, recipeId int64) ([]Image, error)
}

type repository struct {
	db *sqlx.DB
}

func NewRepository() Repository {
	return &repository{db: database.Database}
}

// ===== Books =====

func (r *repository) CreateBook(ctx context.Context, b BookCreate) (*Book, error) {
	const q = `
		INSERT INTO recipe_books (user_id, name, book_type)
		VALUES ($1, $2, 'user')
		RETURNING id, user_id, name, book_type, share_token, is_shared, og_image_url, created_at, updated_at`

	var book Book
	err := r.db.GetContext(ctx, &book, q, b.UserId, b.Name)
	if err != nil {
		return nil, err
	}
	book.RecipeCount = 0
	return &book, nil
}

func (r *repository) UpdateBook(ctx context.Context, b BookUpdate) (*Book, error) {
	const q = `
		UPDATE recipe_books SET
			name = $1,
			updated_at = NOW()
		WHERE id = $2 AND user_id = $3 AND deleted_at IS NULL
		RETURNING id, user_id, name, book_type, share_token, is_shared, og_image_url, created_at, updated_at`

	var book Book
	err := r.db.GetContext(ctx, &book, q, b.Name, b.Id, b.UserId)
	if err != nil {
		return nil, err
	}
	return &book, nil
}

func (r *repository) DeleteBook(ctx context.Context, id int64, userId string) error {
	// Soft delete the book and all its recipes
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Delete book
	_, err = tx.ExecContext(ctx, `
		UPDATE recipe_books SET deleted_at = NOW()
		WHERE id = $1 AND user_id = $2 AND book_type = 'user'`, id, userId)
	if err != nil {
		return err
	}

	// Delete all recipes in the book
	_, err = tx.ExecContext(ctx, `
		UPDATE recipes SET deleted_at = NOW()
		WHERE book_id = $1`, id)
	if err != nil {
		return err
	}

	return tx.Commit()
}

func (r *repository) GetBookById(ctx context.Context, id int64) (*Book, error) {
	const q = `
		SELECT b.id, b.user_id, b.name, b.book_type, b.share_token, b.is_shared, b.og_image_url,
		       b.created_at, b.updated_at,
		       COALESCE((SELECT COUNT(*) FROM recipes WHERE book_id = b.id AND deleted_at IS NULL), 0) as recipe_count
		FROM recipe_books b
		WHERE b.id = $1 AND b.deleted_at IS NULL`

	var book Book
	if err := r.db.GetContext(ctx, &book, q, id); err != nil {
		return nil, err
	}
	return &book, nil
}

func (r *repository) GetBookByShareToken(ctx context.Context, token string) (*Book, error) {
	const q = `
		SELECT b.id, b.user_id, b.name, b.book_type, b.share_token, b.is_shared, b.og_image_url,
		       b.created_at, b.updated_at,
		       COALESCE((SELECT COUNT(*) FROM recipes WHERE book_id = b.id AND deleted_at IS NULL), 0) as recipe_count
		FROM recipe_books b
		WHERE b.share_token = $1 AND b.is_shared = TRUE AND b.deleted_at IS NULL`

	var book Book
	if err := r.db.GetContext(ctx, &book, q, token); err != nil {
		return nil, err
	}
	return &book, nil
}

func (r *repository) GetUserBooks(ctx context.Context, userId string) ([]Book, error) {
	const q = `
		SELECT b.id, b.user_id, b.name, b.book_type, b.share_token, b.is_shared, b.og_image_url,
		       b.created_at, b.updated_at,
		       COALESCE((SELECT COUNT(*) FROM recipes WHERE book_id = b.id AND deleted_at IS NULL), 0) as recipe_count
		FROM recipe_books b
		WHERE b.user_id = $1 AND b.deleted_at IS NULL
		ORDER BY b.created_at DESC`

	var books []Book
	if err := r.db.SelectContext(ctx, &books, q, userId); err != nil {
		return nil, err
	}
	return books, nil
}

func (r *repository) GetNutriBook(ctx context.Context) (*Book, error) {
	const q = `
		SELECT b.id, b.user_id, b.name, b.book_type, b.share_token, b.is_shared, b.og_image_url,
		       b.created_at, b.updated_at,
		       COALESCE((SELECT COUNT(*) FROM recipes WHERE book_id = b.id AND deleted_at IS NULL), 0) as recipe_count
		FROM recipe_books b
		WHERE b.book_type = 'nutri' AND b.deleted_at IS NULL`

	var book Book
	if err := r.db.GetContext(ctx, &book, q); err != nil {
		return nil, err
	}
	return &book, nil
}

func (r *repository) SetBookShareToken(ctx context.Context, id int64, token string) error {
	const q = `UPDATE recipe_books SET share_token = $1, is_shared = TRUE, updated_at = NOW() WHERE id = $2`
	_, err := r.db.ExecContext(ctx, q, token, id)
	return err
}

func (r *repository) ClearBookShareToken(ctx context.Context, id int64) error {
	const q = `UPDATE recipe_books SET share_token = NULL, is_shared = FALSE, updated_at = NOW() WHERE id = $1`
	_, err := r.db.ExecContext(ctx, q, id)
	return err
}

// ===== Categories =====

func (r *repository) CreateCategory(ctx context.Context, c CategoryCreate) (*Category, error) {
	catType := "user"
	if c.CategoryType != "" {
		catType = c.CategoryType
	}

	const q = `
		INSERT INTO recipe_categories (user_id, slug, name_ru, name_en, category_type, icon, sort_order)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, user_id, slug, name_ru, name_en, category_type, icon, sort_order, created_at, updated_at`

	var cat Category
	err := r.db.GetContext(ctx, &cat, q, c.UserId, c.Slug, c.NameRu, c.NameEn, catType, c.Icon, c.SortOrder)
	if err != nil {
		return nil, err
	}
	return &cat, nil
}

func (r *repository) UpdateCategory(ctx context.Context, c CategoryUpdate) (*Category, error) {
	const q = `
		UPDATE recipe_categories SET
			slug = $1,
			name_ru = $2,
			name_en = $3,
			icon = $4,
			sort_order = $5,
			updated_at = NOW()
		WHERE id = $6 AND (user_id = $7 OR user_id IS NULL) AND deleted_at IS NULL
		RETURNING id, user_id, slug, name_ru, name_en, category_type, icon, sort_order, created_at, updated_at`

	var cat Category
	err := r.db.GetContext(ctx, &cat, q, c.Slug, c.NameRu, c.NameEn, c.Icon, c.SortOrder, c.Id, c.UserId)
	if err != nil {
		return nil, err
	}
	return &cat, nil
}

func (r *repository) DeleteCategory(ctx context.Context, id int64, userId string) error {
	const q = `UPDATE recipe_categories SET deleted_at = NOW() WHERE id = $1 AND user_id = $2`
	_, err := r.db.ExecContext(ctx, q, id, userId)
	return err
}

func (r *repository) GetCategoryById(ctx context.Context, id int64) (*Category, error) {
	const q = `
		SELECT id, user_id, slug, name_ru, name_en, category_type, icon, sort_order, created_at, updated_at
		FROM recipe_categories
		WHERE id = $1 AND deleted_at IS NULL`

	var cat Category
	if err := r.db.GetContext(ctx, &cat, q, id); err != nil {
		return nil, err
	}
	return &cat, nil
}

func (r *repository) GetCategories(ctx context.Context, userId *string) ([]Category, error) {
	var q string
	var args []interface{}

	if userId != nil {
		q = `
			SELECT id, user_id, slug, name_ru, name_en, category_type, icon, sort_order, created_at, updated_at
			FROM recipe_categories
			WHERE deleted_at IS NULL AND (category_type = 'system' OR user_id = $1)
			ORDER BY sort_order ASC, name_ru ASC`
		args = append(args, *userId)
	} else {
		q = `
			SELECT id, user_id, slug, name_ru, name_en, category_type, icon, sort_order, created_at, updated_at
			FROM recipe_categories
			WHERE deleted_at IS NULL AND category_type = 'system'
			ORDER BY sort_order ASC, name_ru ASC`
	}

	var cats []Category
	if err := r.db.SelectContext(ctx, &cats, q, args...); err != nil {
		return nil, err
	}
	return cats, nil
}

func (r *repository) GetCategoryByName(ctx context.Context, name string, userId *string) (*Category, error) {
	const q = `
		SELECT id, user_id, slug, name_ru, name_en, category_type, icon, sort_order, created_at, updated_at
		FROM recipe_categories
		WHERE deleted_at IS NULL
		  AND name_ru = $1
		  AND (category_type = 'system' OR user_id = $2)
		LIMIT 1`

	var cat Category
	err := r.db.GetContext(ctx, &cat, q, name, userId)
	if err != nil {
		return nil, err
	}
	return &cat, nil
}

// ===== Tags =====

func (r *repository) CreateTag(ctx context.Context, t TagCreate) (*Tag, error) {
	const q = `
		INSERT INTO recipe_tags (user_id, slug, name_ru, name_en, tag_type)
		VALUES ($1, $2, $3, $4, 'user')
		RETURNING id, user_id, slug, name_ru, name_en, tag_type, created_at, updated_at`

	var tag Tag
	err := r.db.GetContext(ctx, &tag, q, t.UserId, t.Slug, t.NameRu, t.NameEn)
	if err != nil {
		return nil, err
	}
	return &tag, nil
}

func (r *repository) UpdateTag(ctx context.Context, t TagUpdate) (*Tag, error) {
	const q = `
		UPDATE recipe_tags SET
			name_ru = $1,
			name_en = $2,
			updated_at = NOW()
		WHERE id = $3 AND user_id = $4 AND deleted_at IS NULL
		RETURNING id, user_id, slug, name_ru, name_en, tag_type, created_at, updated_at`

	var tag Tag
	err := r.db.GetContext(ctx, &tag, q, t.NameRu, t.NameEn, t.Id, t.UserId)
	if err != nil {
		return nil, err
	}
	return &tag, nil
}

func (r *repository) DeleteTag(ctx context.Context, id int64, userId string) error {
	const q = `UPDATE recipe_tags SET deleted_at = NOW() WHERE id = $1 AND user_id = $2`
	_, err := r.db.ExecContext(ctx, q, id, userId)
	return err
}

func (r *repository) GetTagById(ctx context.Context, id int64) (*Tag, error) {
	const q = `
		SELECT id, user_id, slug, name_ru, name_en, tag_type, created_at, updated_at
		FROM recipe_tags
		WHERE id = $1 AND deleted_at IS NULL`

	var tag Tag
	if err := r.db.GetContext(ctx, &tag, q, id); err != nil {
		return nil, err
	}
	return &tag, nil
}

func (r *repository) GetTags(ctx context.Context, userId *string) ([]Tag, error) {
	var q string
	var args []interface{}

	if userId != nil {
		q = `
			SELECT id, user_id, slug, name_ru, name_en, tag_type, created_at, updated_at
			FROM recipe_tags
			WHERE deleted_at IS NULL AND (tag_type = 'system' OR user_id = $1)
			ORDER BY name_ru ASC`
		args = append(args, *userId)
	} else {
		q = `
			SELECT id, user_id, slug, name_ru, name_en, tag_type, created_at, updated_at
			FROM recipe_tags
			WHERE deleted_at IS NULL AND tag_type = 'system'
			ORDER BY name_ru ASC`
	}

	var tags []Tag
	if err := r.db.SelectContext(ctx, &tags, q, args...); err != nil {
		return nil, err
	}
	return tags, nil
}

func (r *repository) GetTagsByRecipeId(ctx context.Context, recipeId int64) ([]Tag, error) {
	const q = `
		SELECT t.id, t.user_id, t.slug, t.name_ru, t.name_en, t.tag_type, t.created_at, t.updated_at
		FROM recipe_tags t
		JOIN recipe_tags_map m ON m.tag_id = t.id
		WHERE m.recipe_id = $1 AND t.deleted_at IS NULL
		ORDER BY t.name_ru ASC`

	var tags []Tag
	if err := r.db.SelectContext(ctx, &tags, q, recipeId); err != nil {
		return nil, err
	}
	return tags, nil
}

func (r *repository) SetRecipeTags(ctx context.Context, recipeId int64, tagIds []int64) error {
	// Delete existing tags
	_, err := r.db.ExecContext(ctx, `DELETE FROM recipe_tags_map WHERE recipe_id = $1`, recipeId)
	if err != nil {
		return err
	}

	if len(tagIds) == 0 {
		return nil
	}

	// Insert new tags
	query := `INSERT INTO recipe_tags_map (recipe_id, tag_id) VALUES `
	values := []string{}
	args := []interface{}{}
	for i, tagId := range tagIds {
		values = append(values, fmt.Sprintf("($%d, $%d)", i*2+1, i*2+2))
		args = append(args, recipeId, tagId)
	}
	query += strings.Join(values, ", ") + " ON CONFLICT DO NOTHING"

	_, err = r.db.ExecContext(ctx, query, args...)
	return err
}

// ===== Recipes =====

func (r *repository) CreateRecipe(ctx context.Context, rc RecipeCreate) (*Recipe, error) {
	const q = `
		INSERT INTO recipes (
			book_id, user_id, slug, title_ru, title_en, description_ru, description_en, main_image_url, external_url,
			prep_time, cook_time, total_time, servings, servings_unit,
			calories, protein, fat, carbs, fiber, difficulty, category_id,
			meta_description_ru, meta_description_en
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23
		)
		RETURNING id, book_id, user_id, slug, title_ru, title_en, description_ru, description_en, main_image_url, external_url,
		          prep_time, cook_time, total_time, servings, servings_unit,
		          calories, protein, fat, carbs, fiber, nutrition_calculated_by_ai, difficulty,
		          share_token, is_public, og_image_url, meta_description_ru, meta_description_en,
		          view_count, copy_count, copied_from_id, category_id, published_at, created_at, updated_at`

	servings := rc.Servings
	if servings < 1 {
		servings = 1
	}

	var recipe Recipe
	err := r.db.GetContext(ctx, &recipe, q,
		rc.BookId, rc.UserId, rc.Slug, rc.TitleRu, rc.TitleEn, rc.DescriptionRu, rc.DescriptionEn, rc.MainImageUrl, rc.ExternalUrl,
		rc.PrepTime, rc.CookTime, rc.TotalTime, servings, rc.ServingsUnit,
		rc.Calories, rc.Protein, rc.Fat, rc.Carbs, rc.Fiber, rc.Difficulty, rc.CategoryId,
		rc.MetaDescriptionRu, rc.MetaDescriptionEn)
	if err != nil {
		return nil, err
	}
	return &recipe, nil
}

func (r *repository) UpdateRecipe(ctx context.Context, ru RecipeUpdate) (*Recipe, error) {
	const q = `
		UPDATE recipes SET
			slug = $1,
			title_ru = $2,
			title_en = $3,
			description_ru = $4,
			description_en = $5,
			main_image_url = $6,
			external_url = $7,
			prep_time = $8,
			cook_time = $9,
			total_time = $10,
			servings = $11,
			servings_unit = $12,
			calories = $13,
			protein = $14,
			fat = $15,
			carbs = $16,
			fiber = $17,
			difficulty = $18,
			category_id = $19,
			meta_description_ru = $20,
			meta_description_en = $21,
			updated_at = NOW()
		WHERE id = $22 AND user_id = $23 AND deleted_at IS NULL
		RETURNING id, book_id, user_id, slug, title_ru, title_en, description_ru, description_en, main_image_url, external_url,
		          prep_time, cook_time, total_time, servings, servings_unit,
		          calories, protein, fat, carbs, fiber, nutrition_calculated_by_ai, difficulty,
		          share_token, is_public, og_image_url, meta_description_ru, meta_description_en,
		          view_count, copy_count, copied_from_id, category_id, published_at, created_at, updated_at`

	servings := ru.Servings
	if servings < 1 {
		servings = 1
	}

	var recipe Recipe
	err := r.db.GetContext(ctx, &recipe, q,
		ru.Slug, ru.TitleRu, ru.TitleEn, ru.DescriptionRu, ru.DescriptionEn, ru.MainImageUrl, ru.ExternalUrl,
		ru.PrepTime, ru.CookTime, ru.TotalTime, servings, ru.ServingsUnit,
		ru.Calories, ru.Protein, ru.Fat, ru.Carbs, ru.Fiber, ru.Difficulty, ru.CategoryId,
		ru.MetaDescriptionRu, ru.MetaDescriptionEn, ru.Id, ru.UserId)
	if err != nil {
		return nil, err
	}
	return &recipe, nil
}

func (r *repository) DeleteRecipe(ctx context.Context, id int64, userId string) error {
	const q = `UPDATE recipes SET deleted_at = NOW() WHERE id = $1 AND user_id = $2`
	_, err := r.db.ExecContext(ctx, q, id, userId)
	return err
}

func (r *repository) GetRecipeById(ctx context.Context, id int64) (*Recipe, error) {
	const q = `
		SELECT id, book_id, user_id, slug, title_ru, title_en, description_ru, description_en, main_image_url, external_url,
		       prep_time, cook_time, total_time, servings, servings_unit,
		       calories, protein, fat, carbs, fiber, nutrition_calculated_by_ai, difficulty,
		       share_token, is_public, og_image_url, meta_description_ru, meta_description_en,
		       view_count, copy_count, copied_from_id, category_id, published_at, created_at, updated_at
		FROM recipes
		WHERE id = $1 AND deleted_at IS NULL`

	var recipe Recipe
	if err := r.db.GetContext(ctx, &recipe, q, id); err != nil {
		return nil, err
	}
	return &recipe, nil
}

func (r *repository) GetRecipeBySlug(ctx context.Context, slug string) (*Recipe, error) {
	const q = `
		SELECT id, book_id, user_id, slug, title_ru, title_en, description_ru, description_en, main_image_url, external_url,
		       prep_time, cook_time, total_time, servings, servings_unit,
		       calories, protein, fat, carbs, fiber, nutrition_calculated_by_ai, difficulty,
		       share_token, is_public, og_image_url, meta_description_ru, meta_description_en,
		       view_count, copy_count, copied_from_id, category_id, published_at, created_at, updated_at
		FROM recipes
		WHERE slug = $1 AND deleted_at IS NULL`

	var recipe Recipe
	if err := r.db.GetContext(ctx, &recipe, q, slug); err != nil {
		return nil, err
	}
	return &recipe, nil
}

func (r *repository) GetRecipeByShareToken(ctx context.Context, token string) (*Recipe, error) {
	const q = `
		SELECT id, book_id, user_id, slug, title_ru, title_en, description_ru, description_en, main_image_url, external_url,
		       prep_time, cook_time, total_time, servings, servings_unit,
		       calories, protein, fat, carbs, fiber, nutrition_calculated_by_ai, difficulty,
		       share_token, is_public, og_image_url, meta_description_ru, meta_description_en,
		       view_count, copy_count, copied_from_id, category_id, published_at, created_at, updated_at
		FROM recipes
		WHERE share_token = $1 AND is_public = TRUE AND deleted_at IS NULL`

	var recipe Recipe
	if err := r.db.GetContext(ctx, &recipe, q, token); err != nil {
		return nil, err
	}
	return &recipe, nil
}

func (r *repository) GetRecipes(ctx context.Context, params RecipeListParams) (*RecipeListResponse, error) {
	conditions := []string{"r.deleted_at IS NULL"}
	args := []interface{}{}
	argIndex := 1

	if params.UserId != nil {
		conditions = append(conditions, fmt.Sprintf("r.user_id = $%d", argIndex))
		args = append(args, *params.UserId)
		argIndex++
	}

	if params.BookId != nil {
		conditions = append(conditions, fmt.Sprintf("r.book_id = $%d", argIndex))
		args = append(args, *params.BookId)
		argIndex++
	}

	if params.CategoryId != nil {
		conditions = append(conditions, fmt.Sprintf("r.category_id = $%d", argIndex))
		args = append(args, *params.CategoryId)
		argIndex++
	}

	if params.IsPublic != nil && *params.IsPublic {
		conditions = append(conditions, "r.is_public = TRUE")
	}

	if params.TagId != nil {
		conditions = append(conditions, fmt.Sprintf(`
			EXISTS (
				SELECT 1 FROM recipe_tags_map m
				WHERE m.recipe_id = r.id AND m.tag_id = $%d
			)`, argIndex))
		args = append(args, *params.TagId)
		argIndex++
	}

	if params.Search != nil && *params.Search != "" {
		searchPattern := "%" + strings.ToLower(*params.Search) + "%"
		conditions = append(conditions, fmt.Sprintf(
			"(LOWER(r.title_ru) LIKE $%d OR LOWER(r.title_en) LIKE $%d)",
			argIndex, argIndex))
		args = append(args, searchPattern)
		argIndex++
	}

	whereClause := strings.Join(conditions, " AND ")

	// Count total
	countQuery := fmt.Sprintf(`SELECT COUNT(*) FROM recipes r WHERE %s`, whereClause)

	var total int
	if err := r.db.GetContext(ctx, &total, countQuery, args...); err != nil {
		return nil, err
	}

	// Pagination
	page := params.Page
	if page < 1 {
		page = 1
	}
	perPage := params.PerPage
	if perPage < 1 {
		perPage = 20
	}
	if perPage > 100 {
		perPage = 100
	}
	offset := (page - 1) * perPage

	// Fetch recipes
	selectQuery := fmt.Sprintf(`
		SELECT r.id, r.book_id, r.user_id, r.slug, r.title_ru, r.title_en, r.description_ru, r.description_en, r.main_image_url, r.external_url,
		       r.prep_time, r.cook_time, r.total_time, r.servings, r.servings_unit,
		       r.calories, r.protein, r.fat, r.carbs, r.fiber, r.nutrition_calculated_by_ai, r.difficulty,
		       r.share_token, r.is_public, r.og_image_url, r.meta_description_ru, r.meta_description_en,
		       r.view_count, r.copy_count, r.copied_from_id, r.category_id, r.published_at, r.created_at, r.updated_at
		FROM recipes r
		WHERE %s
		ORDER BY r.created_at DESC
		LIMIT $%d OFFSET $%d`, whereClause, argIndex, argIndex+1)

	args = append(args, perPage, offset)

	var recipes []Recipe
	if err := r.db.SelectContext(ctx, &recipes, selectQuery, args...); err != nil {
		return nil, err
	}

	totalPages := (total + perPage - 1) / perPage

	return &RecipeListResponse{
		Recipes:    recipes,
		Total:      total,
		Page:       page,
		PerPage:    perPage,
		TotalPages: totalPages,
	}, nil
}

func (r *repository) SetRecipeShareToken(ctx context.Context, id int64, token string) error {
	const q = `UPDATE recipes SET share_token = $1, is_public = TRUE, updated_at = NOW() WHERE id = $2`
	_, err := r.db.ExecContext(ctx, q, token, id)
	return err
}

func (r *repository) ClearRecipeShareToken(ctx context.Context, id int64) error {
	const q = `UPDATE recipes SET share_token = NULL, is_public = FALSE, updated_at = NOW() WHERE id = $1`
	_, err := r.db.ExecContext(ctx, q, id)
	return err
}

func (r *repository) IncrementViewCount(ctx context.Context, id int64) error {
	const q = `UPDATE recipes SET view_count = view_count + 1 WHERE id = $1`
	_, err := r.db.ExecContext(ctx, q, id)
	return err
}

func (r *repository) IncrementCopyCount(ctx context.Context, id int64) error {
	const q = `UPDATE recipes SET copy_count = copy_count + 1 WHERE id = $1`
	_, err := r.db.ExecContext(ctx, q, id)
	return err
}

func (r *repository) UpdateNutrition(ctx context.Context, id int64, calories, protein, fat, carbs, fiber *float64, byAi bool) error {
	const q = `
		UPDATE recipes SET
			calories = $1, protein = $2, fat = $3, carbs = $4, fiber = $5,
			nutrition_calculated_by_ai = $6, updated_at = NOW()
		WHERE id = $7`
	_, err := r.db.ExecContext(ctx, q, calories, protein, fat, carbs, fiber, byAi, id)
	return err
}

// ===== Steps =====

func (r *repository) CreateSteps(ctx context.Context, recipeId int64, steps []StepCreate) error {
	if len(steps) == 0 {
		return nil
	}

	query := `INSERT INTO recipe_steps (recipe_id, step_number, instruction_ru, instruction_en, image_url, duration_minutes) VALUES `
	values := []string{}
	args := []interface{}{}
	argIdx := 1

	for _, s := range steps {
		values = append(values, fmt.Sprintf("($%d, $%d, $%d, $%d, $%d, $%d)", argIdx, argIdx+1, argIdx+2, argIdx+3, argIdx+4, argIdx+5))
		args = append(args, recipeId, s.StepNumber, s.InstructionRu, s.InstructionEn, s.ImageUrl, s.DurationMinutes)
		argIdx += 6
	}

	query += strings.Join(values, ", ")
	_, err := r.db.ExecContext(ctx, query, args...)
	return err
}

func (r *repository) UpdateSteps(ctx context.Context, recipeId int64, steps []StepCreate) error {
	// Delete existing steps and recreate
	if err := r.DeleteStepsByRecipeId(ctx, recipeId); err != nil {
		return err
	}
	return r.CreateSteps(ctx, recipeId, steps)
}

func (r *repository) GetStepsByRecipeId(ctx context.Context, recipeId int64) ([]Step, error) {
	const q = `
		SELECT id, recipe_id, step_number, instruction_ru, instruction_en, image_url, duration_minutes, created_at, updated_at
		FROM recipe_steps
		WHERE recipe_id = $1
		ORDER BY step_number ASC`

	var steps []Step
	if err := r.db.SelectContext(ctx, &steps, q, recipeId); err != nil {
		return nil, err
	}
	return steps, nil
}

func (r *repository) DeleteStepsByRecipeId(ctx context.Context, recipeId int64) error {
	const q = `DELETE FROM recipe_steps WHERE recipe_id = $1`
	_, err := r.db.ExecContext(ctx, q, recipeId)
	return err
}

// ===== Ingredients =====

func (r *repository) CreateIngredients(ctx context.Context, recipeId int64, ingredients []IngredientCreate) error {
	if len(ingredients) == 0 {
		return nil
	}

	query := `INSERT INTO recipe_ingredients (recipe_id, sort_order, name_ru, name_en, amount, unit, calories, protein, fat, carbs, fiber, is_optional, group_name) VALUES `
	values := []string{}
	args := []interface{}{}
	argIdx := 1

	for _, ing := range ingredients {
		values = append(values, fmt.Sprintf("($%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d)",
			argIdx, argIdx+1, argIdx+2, argIdx+3, argIdx+4, argIdx+5, argIdx+6, argIdx+7, argIdx+8, argIdx+9, argIdx+10, argIdx+11, argIdx+12))
		args = append(args, recipeId, ing.SortOrder, ing.NameRu, ing.NameEn, ing.Amount, ing.Unit, ing.Calories, ing.Protein, ing.Fat, ing.Carbs, ing.Fiber, ing.IsOptional, ing.GroupName)
		argIdx += 13
	}

	query += strings.Join(values, ", ")
	_, err := r.db.ExecContext(ctx, query, args...)
	return err
}

func (r *repository) UpdateIngredients(ctx context.Context, recipeId int64, ingredients []IngredientCreate) error {
	// Delete existing ingredients and recreate
	if err := r.DeleteIngredientsByRecipeId(ctx, recipeId); err != nil {
		return err
	}
	return r.CreateIngredients(ctx, recipeId, ingredients)
}

func (r *repository) GetIngredientsByRecipeId(ctx context.Context, recipeId int64) ([]Ingredient, error) {
	const q = `
		SELECT id, recipe_id, sort_order, name_ru, name_en, amount, unit, calories, protein, fat, carbs, fiber, is_optional, group_name, created_at, updated_at
		FROM recipe_ingredients
		WHERE recipe_id = $1
		ORDER BY sort_order ASC`

	var ingredients []Ingredient
	if err := r.db.SelectContext(ctx, &ingredients, q, recipeId); err != nil {
		return nil, err
	}
	return ingredients, nil
}

func (r *repository) DeleteIngredientsByRecipeId(ctx context.Context, recipeId int64) error {
	const q = `DELETE FROM recipe_ingredients WHERE recipe_id = $1`
	_, err := r.db.ExecContext(ctx, q, recipeId)
	return err
}

func (r *repository) UpdateIngredientNutrition(ctx context.Context, id int64, calories, protein, fat, carbs, fiber float64) error {
	const q = `
		UPDATE recipe_ingredients SET
			calories = $1, protein = $2, fat = $3, carbs = $4, fiber = $5, updated_at = NOW()
		WHERE id = $6`
	_, err := r.db.ExecContext(ctx, q, calories, protein, fat, carbs, fiber, id)
	return err
}

// ===== Nutrition =====

func (r *repository) UpdateRecipeNutrition(ctx context.Context, id int64, calories, protein, fat, carbs, fiber float64) error {
	const q = `
		UPDATE recipes SET
			calories = $1, protein = $2, fat = $3, carbs = $4, fiber = $5,
			nutrition_calculated_by_ai = TRUE, updated_at = NOW()
		WHERE id = $6`
	_, err := r.db.ExecContext(ctx, q, calories, protein, fat, carbs, fiber, id)
	return err
}

// ===== Images =====

func (r *repository) CreateImage(ctx context.Context, recipeId int64, img ImageCreate) (*Image, error) {
	const q = `
		INSERT INTO recipe_images (recipe_id, image_url, caption_ru, caption_en, sort_order)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, recipe_id, image_url, caption_ru, caption_en, sort_order, created_at`

	var image Image
	err := r.db.GetContext(ctx, &image, q, recipeId, img.ImageUrl, img.CaptionRu, img.CaptionEn, img.SortOrder)
	if err != nil {
		return nil, err
	}
	return &image, nil
}

func (r *repository) DeleteImage(ctx context.Context, id int64, recipeId int64) error {
	const q = `DELETE FROM recipe_images WHERE id = $1 AND recipe_id = $2`
	_, err := r.db.ExecContext(ctx, q, id, recipeId)
	return err
}

func (r *repository) GetImagesByRecipeId(ctx context.Context, recipeId int64) ([]Image, error) {
	const q = `
		SELECT id, recipe_id, image_url, caption_ru, caption_en, sort_order, created_at
		FROM recipe_images
		WHERE recipe_id = $1
		ORDER BY sort_order ASC`

	var images []Image
	if err := r.db.SelectContext(ctx, &images, q, recipeId); err != nil {
		return nil, err
	}
	return images, nil
}
