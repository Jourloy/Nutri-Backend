package recipe

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"strconv"

	"github.com/charmbracelet/log"
	"github.com/go-chi/chi/v5"

	"github.com/jourloy/nutri-backend/internal/auth"
)

var (
	logger = log.NewWithOptions(os.Stderr, log.Options{
		Prefix: "[recipe]",
		Level:  log.DebugLevel,
	})
)

type Controller struct {
	service Service
}

func NewController() (*Controller, error) {
	svc, err := NewService()
	if err != nil {
		return nil, err
	}

	return &Controller{
		service: svc,
	}, nil
}

func (c *Controller) RegisterRoutes(router chi.Router) {
	router.Route("/recipe", func(r chi.Router) {
		// Public endpoints (no auth required)
		r.Get("/nutri", c.GetNutriRecipes)
		r.Get("/nutri/{slug}", c.GetNutriRecipeBySlug)
		r.Get("/shared/{token}", c.GetSharedRecipe)
		r.Get("/shared/book/{token}", c.GetSharedBook)
		r.Get("/shared/book/{token}/recipes", c.GetSharedBookRecipes)
		r.Get("/categories", c.GetPublicCategories)
		r.Get("/tags", c.GetPublicTags)

		// User endpoints (auth required)
		r.Group(func(r chi.Router) {
			r.Use(requireAuth)

			// Books
			r.Get("/books", c.GetUserBooks)
			r.Post("/books", c.CreateBook)
			r.Get("/books/{id}", c.GetBookById)
			r.Put("/books/{id}", c.UpdateBook)
			r.Delete("/books/{id}", c.DeleteBook)
			r.Post("/books/{id}/share", c.ShareBook)
			r.Delete("/books/{id}/share", c.UnshareBook)

			// Recipes
			r.Get("/recipes", c.GetRecipes)
			r.Post("/recipes", c.CreateRecipe)
			r.Get("/recipes/{id}", c.GetRecipeById)
			r.Put("/recipes/{id}", c.UpdateRecipe)
			r.Delete("/recipes/{id}", c.DeleteRecipe)
			r.Post("/recipes/{id}/share", c.ShareRecipe)
			r.Delete("/recipes/{id}/share", c.UnshareRecipe)
			r.Post("/recipes/{id}/copy", c.CopyRecipe)
			r.Post("/recipes/{id}/to-diary", c.AddToDiary)
			r.Post("/recipes/{id}/calculate-nutrition", c.CalculateNutrition)

			// Categories (user-specific: system + own categories)
			r.Get("/user/categories", c.GetUserCategories)
			r.Post("/categories", c.CreateCategory)
			r.Put("/categories/{id}", c.UpdateCategory)
			r.Delete("/categories/{id}", c.DeleteCategory)

			// Tags (user-specific: system + own tags)
			r.Get("/user/tags", c.GetUserTags)
			r.Post("/tags", c.CreateTag)
			r.Put("/tags/{id}", c.UpdateTag)
			r.Delete("/tags/{id}", c.DeleteTag)

			// Upload
			r.Post("/upload", c.UploadImage)
		})

		// Admin endpoints
		r.Group(func(r chi.Router) {
			r.Use(requireAdmin)

			r.Get("/admin/recipes/{id}", c.GetNutriRecipeById)
			r.Post("/admin/recipes", c.CreateNutriRecipe)
			r.Put("/admin/recipes/{id}", c.UpdateNutriRecipe)
			r.Delete("/admin/recipes/{id}", c.DeleteNutriRecipe)
			r.Get("/admin/recipes", c.GetAllNutriRecipes)

			r.Post("/admin/categories", c.CreateSystemCategory)
			r.Post("/admin/tags", c.CreateSystemTag)
			r.Put("/admin/tags/{id}", c.UpdateSystemTag)
			r.Delete("/admin/tags/{id}", c.DeleteSystemTag)
		})
	})

	logger.Info("╔═════ Recipe")
	logger.Info("║    GET /recipe/nutri (public: get Nutri recipes)")
	logger.Info("║    GET /recipe/nutri/{slug} (public: get Nutri recipe by slug)")
	logger.Info("║    GET /recipe/shared/{token} (public: get shared recipe)")
	logger.Info("║    GET /recipe/shared/book/{token} (public: get shared book)")
	logger.Info("║    GET /recipe/categories (public: get categories)")
	logger.Info("║    GET /recipe/tags (public: get tags)")
	logger.Info("║    GET /recipe/books (auth: get user books)")
	logger.Info("║   POST /recipe/books (auth: create book)")
	logger.Info("║    PUT /recipe/books/{id} (auth: update book)")
	logger.Info("║ DELETE /recipe/books/{id} (auth: delete book)")
	logger.Info("║   POST /recipe/books/{id}/share (auth: share book)")
	logger.Info("║ DELETE /recipe/books/{id}/share (auth: unshare book)")
	logger.Info("║    GET /recipe/recipes (auth: get recipes)")
	logger.Info("║   POST /recipe/recipes (auth: create recipe)")
	logger.Info("║    PUT /recipe/recipes/{id} (auth: update recipe)")
	logger.Info("║ DELETE /recipe/recipes/{id} (auth: delete recipe)")
	logger.Info("║   POST /recipe/recipes/{id}/share (auth: share recipe)")
	logger.Info("║   POST /recipe/recipes/{id}/copy (auth: copy recipe)")
	logger.Info("║   POST /recipe/recipes/{id}/to-diary (auth: add to diary)")
	logger.Info("║   POST /recipe/recipes/{id}/calculate-nutrition (auth: AI nutrition calc)")
	logger.Info("║    GET /recipe/admin/recipes/{id} (admin: get Nutri recipe by id)")
	logger.Info("║   POST /recipe/admin/recipes (admin: create Nutri recipe)")
	logger.Info("║    PUT /recipe/admin/recipes/{id} (admin: update Nutri recipe)")
	logger.Info("║ DELETE /recipe/admin/recipes/{id} (admin: delete Nutri recipe)")
	logger.Info("║   POST /recipe/admin/tags (admin: create system tag)")
	logger.Info("║    PUT /recipe/admin/tags/{id} (admin: update system tag)")
	logger.Info("║ DELETE /recipe/admin/tags/{id} (admin: delete system tag)")
	logger.Info("╚═════")
}

// ===== Middlewares =====

func requireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, ok := auth.UserFromContext(r.Context())
		if !ok {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func requireAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u, ok := auth.UserFromContext(r.Context())
		if !ok {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		if !u.IsAdmin {
			http.Error(w, "forbidden: admin access required", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// ===== Helpers =====

func parseRecipeListParams(r *http.Request) RecipeListParams {
	params := RecipeListParams{
		Page:    1,
		PerPage: 20,
	}

	if page := r.URL.Query().Get("page"); page != "" {
		if p, err := strconv.Atoi(page); err == nil && p > 0 {
			params.Page = p
		}
	}

	if perPage := r.URL.Query().Get("perPage"); perPage != "" {
		if pp, err := strconv.Atoi(perPage); err == nil && pp > 0 {
			params.PerPage = pp
		}
	}

	if bookId := r.URL.Query().Get("bookId"); bookId != "" {
		if id, err := strconv.ParseInt(bookId, 10, 64); err == nil {
			params.BookId = &id
		}
	}

	if categoryId := r.URL.Query().Get("categoryId"); categoryId != "" {
		if id, err := strconv.ParseInt(categoryId, 10, 64); err == nil {
			params.CategoryId = &id
		}
	}

	if tagId := r.URL.Query().Get("tagId"); tagId != "" {
		if id, err := strconv.ParseInt(tagId, 10, 64); err == nil {
			params.TagId = &id
		}
	}

	if search := r.URL.Query().Get("search"); search != "" {
		params.Search = &search
	}

	return params
}

// ===== Public Endpoints =====

func (c *Controller) GetNutriRecipes(w http.ResponseWriter, r *http.Request) {
	params := parseRecipeListParams(r)

	response, err := c.service.GetNutriRecipes(context.Background(), params)
	if err != nil {
		logger.Error("failed to get Nutri recipes", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (c *Controller) GetNutriRecipeBySlug(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")

	recipe, err := c.service.GetNutriRecipeBySlug(context.Background(), slug)
	if err != nil {
		logger.Error("failed to get Nutri recipe", "slug", slug, "error", err)
		http.Error(w, "recipe not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(recipe)
}

func (c *Controller) GetSharedRecipe(w http.ResponseWriter, r *http.Request) {
	token := chi.URLParam(r, "token")

	recipe, err := c.service.GetSharedRecipe(context.Background(), token)
	if err != nil {
		logger.Error("failed to get shared recipe", "token", token, "error", err)
		http.Error(w, "recipe not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(recipe)
}

func (c *Controller) GetSharedBook(w http.ResponseWriter, r *http.Request) {
	token := chi.URLParam(r, "token")

	book, err := c.service.GetSharedBook(context.Background(), token)
	if err != nil {
		logger.Error("failed to get shared book", "token", token, "error", err)
		http.Error(w, "book not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(book)
}

func (c *Controller) GetSharedBookRecipes(w http.ResponseWriter, r *http.Request) {
	token := chi.URLParam(r, "token")
	params := parseRecipeListParams(r)

	response, err := c.service.GetSharedBookRecipes(context.Background(), token, params)
	if err != nil {
		logger.Error("failed to get shared book recipes", "token", token, "error", err)
		http.Error(w, "book not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (c *Controller) GetPublicCategories(w http.ResponseWriter, r *http.Request) {
	// Get system categories only (for unauthenticated users)
	categories, err := c.service.GetCategories(context.Background(), nil)
	if err != nil {
		logger.Error("failed to get categories", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(categories)
}

func (c *Controller) GetPublicTags(w http.ResponseWriter, r *http.Request) {
	// Get system tags only (for unauthenticated users)
	tags, err := c.service.GetTags(context.Background(), nil)
	if err != nil {
		logger.Error("failed to get tags", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(tags)
}

// ===== User Categories & Tags (authenticated) =====

func (c *Controller) GetUserCategories(w http.ResponseWriter, r *http.Request) {
	u, _ := auth.UserFromContext(r.Context())

	// Get system categories + user's own categories
	categories, err := c.service.GetCategories(context.Background(), &u.Id)
	if err != nil {
		logger.Error("failed to get user categories", "userId", u.Id, "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(categories)
}

func (c *Controller) GetUserTags(w http.ResponseWriter, r *http.Request) {
	u, _ := auth.UserFromContext(r.Context())

	// Get system tags + user's own tags
	tags, err := c.service.GetTags(context.Background(), &u.Id)
	if err != nil {
		logger.Error("failed to get user tags", "userId", u.Id, "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(tags)
}

// ===== User Book Endpoints =====

func (c *Controller) GetUserBooks(w http.ResponseWriter, r *http.Request) {
	u, _ := auth.UserFromContext(r.Context())

	books, err := c.service.GetUserBooks(context.Background(), u.Id)
	if err != nil {
		logger.Error("failed to get user books", "userId", u.Id, "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(books)
}

func (c *Controller) CreateBook(w http.ResponseWriter, r *http.Request) {
	u, _ := auth.UserFromContext(r.Context())

	var bc BookCreate
	if err := json.NewDecoder(r.Body).Decode(&bc); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	bc.UserId = &u.Id

	book, err := c.service.CreateBook(context.Background(), bc)
	if err != nil {
		logger.Error("failed to create book", "error", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(book)
}

func (c *Controller) GetBookById(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "invalid book id", http.StatusBadRequest)
		return
	}

	book, err := c.service.GetBookById(context.Background(), id)
	if err != nil {
		logger.Error("failed to get book", "id", id, "error", err)
		http.Error(w, "book not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(book)
}

func (c *Controller) UpdateBook(w http.ResponseWriter, r *http.Request) {
	u, _ := auth.UserFromContext(r.Context())

	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "invalid book id", http.StatusBadRequest)
		return
	}

	var bu BookUpdate
	if err := json.NewDecoder(r.Body).Decode(&bu); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	bu.Id = id
	bu.UserId = u.Id

	book, err := c.service.UpdateBook(context.Background(), bu)
	if err != nil {
		logger.Error("failed to update book", "id", id, "error", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(book)
}

func (c *Controller) DeleteBook(w http.ResponseWriter, r *http.Request) {
	u, _ := auth.UserFromContext(r.Context())

	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "invalid book id", http.StatusBadRequest)
		return
	}

	if err := c.service.DeleteBook(context.Background(), id, u.Id); err != nil {
		logger.Error("failed to delete book", "id", id, "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "deleted"})
}

func (c *Controller) ShareBook(w http.ResponseWriter, r *http.Request) {
	u, _ := auth.UserFromContext(r.Context())

	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "invalid book id", http.StatusBadRequest)
		return
	}

	response, err := c.service.ShareBook(context.Background(), id, u.Id)
	if err != nil {
		logger.Error("failed to share book", "id", id, "error", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (c *Controller) UnshareBook(w http.ResponseWriter, r *http.Request) {
	u, _ := auth.UserFromContext(r.Context())

	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "invalid book id", http.StatusBadRequest)
		return
	}

	if err := c.service.UnshareBook(context.Background(), id, u.Id); err != nil {
		logger.Error("failed to unshare book", "id", id, "error", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "unshared"})
}

// ===== User Recipe Endpoints =====

func (c *Controller) GetRecipes(w http.ResponseWriter, r *http.Request) {
	u, _ := auth.UserFromContext(r.Context())
	params := parseRecipeListParams(r)
	params.UserId = &u.Id

	response, err := c.service.GetRecipes(context.Background(), params)
	if err != nil {
		logger.Error("failed to get recipes", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (c *Controller) CreateRecipe(w http.ResponseWriter, r *http.Request) {
	u, _ := auth.UserFromContext(r.Context())

	var rc RecipeCreate
	if err := json.NewDecoder(r.Body).Decode(&rc); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	rc.UserId = &u.Id

	if rc.TitleRu == "" {
		http.Error(w, "titleRu is required", http.StatusBadRequest)
		return
	}

	recipe, err := c.service.CreateRecipe(context.Background(), rc)
	if err != nil {
		logger.Error("failed to create recipe", "error", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(recipe)
}

func (c *Controller) GetRecipeById(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "invalid recipe id", http.StatusBadRequest)
		return
	}

	recipe, err := c.service.GetRecipeById(context.Background(), id)
	if err != nil {
		logger.Error("failed to get recipe", "id", id, "error", err)
		http.Error(w, "recipe not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(recipe)
}

func (c *Controller) UpdateRecipe(w http.ResponseWriter, r *http.Request) {
	u, _ := auth.UserFromContext(r.Context())

	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "invalid recipe id", http.StatusBadRequest)
		return
	}

	var ru RecipeUpdate
	if err := json.NewDecoder(r.Body).Decode(&ru); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	ru.Id = id
	ru.UserId = u.Id

	recipe, err := c.service.UpdateRecipe(context.Background(), ru)
	if err != nil {
		logger.Error("failed to update recipe", "id", id, "error", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(recipe)
}

func (c *Controller) DeleteRecipe(w http.ResponseWriter, r *http.Request) {
	u, _ := auth.UserFromContext(r.Context())

	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "invalid recipe id", http.StatusBadRequest)
		return
	}

	if err := c.service.DeleteRecipe(context.Background(), id, u.Id); err != nil {
		logger.Error("failed to delete recipe", "id", id, "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "deleted"})
}

func (c *Controller) ShareRecipe(w http.ResponseWriter, r *http.Request) {
	u, _ := auth.UserFromContext(r.Context())

	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "invalid recipe id", http.StatusBadRequest)
		return
	}

	response, err := c.service.ShareRecipe(context.Background(), id, u.Id)
	if err != nil {
		logger.Error("failed to share recipe", "id", id, "error", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (c *Controller) UnshareRecipe(w http.ResponseWriter, r *http.Request) {
	u, _ := auth.UserFromContext(r.Context())

	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "invalid recipe id", http.StatusBadRequest)
		return
	}

	if err := c.service.UnshareRecipe(context.Background(), id, u.Id); err != nil {
		logger.Error("failed to unshare recipe", "id", id, "error", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "unshared"})
}

func (c *Controller) CopyRecipe(w http.ResponseWriter, r *http.Request) {
	u, _ := auth.UserFromContext(r.Context())

	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "invalid recipe id", http.StatusBadRequest)
		return
	}

	var req CopyRecipeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	recipe, err := c.service.CopyRecipe(context.Background(), id, req.TargetBookId, u.Id)
	if err != nil {
		logger.Error("failed to copy recipe", "id", id, "error", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(recipe)
}

func (c *Controller) AddToDiary(w http.ResponseWriter, r *http.Request) {
	u, _ := auth.UserFromContext(r.Context())

	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "invalid recipe id", http.StatusBadRequest)
		return
	}

	var req AddToDiaryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	req.RecipeId = id

	if req.ServingsAmount <= 0 {
		req.ServingsAmount = 1
	}

	timezone := ""
	if u.Timezone != nil {
		timezone = *u.Timezone
	}

	products, err := c.service.AddToDiary(context.Background(), req, u.Id, timezone)
	if err != nil {
		logger.Error("failed to add to diary", "recipeId", id, "error", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(products)
}

func (c *Controller) CalculateNutrition(w http.ResponseWriter, r *http.Request) {
	u, _ := auth.UserFromContext(r.Context())

	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "invalid recipe id", http.StatusBadRequest)
		return
	}

	result, err := c.service.CalculateNutrition(context.Background(), id, u.Id)
	if err != nil {
		logger.Error("failed to calculate nutrition", "recipeId", id, "error", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// ===== Category Endpoints =====

func (c *Controller) CreateCategory(w http.ResponseWriter, r *http.Request) {
	u, _ := auth.UserFromContext(r.Context())

	var cc CategoryCreate
	if err := json.NewDecoder(r.Body).Decode(&cc); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	cc.UserId = &u.Id

	if cc.NameRu == "" {
		http.Error(w, "nameRu is required", http.StatusBadRequest)
		return
	}

	category, err := c.service.CreateCategory(context.Background(), cc)
	if err != nil {
		logger.Error("failed to create category", "error", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(category)
}

func (c *Controller) UpdateCategory(w http.ResponseWriter, r *http.Request) {
	u, _ := auth.UserFromContext(r.Context())

	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "invalid category id", http.StatusBadRequest)
		return
	}

	var cu CategoryUpdate
	if err := json.NewDecoder(r.Body).Decode(&cu); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	cu.Id = id
	cu.UserId = u.Id

	category, err := c.service.UpdateCategory(context.Background(), cu)
	if err != nil {
		logger.Error("failed to update category", "id", id, "error", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(category)
}

func (c *Controller) DeleteCategory(w http.ResponseWriter, r *http.Request) {
	u, _ := auth.UserFromContext(r.Context())

	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "invalid category id", http.StatusBadRequest)
		return
	}

	if err := c.service.DeleteCategory(context.Background(), id, u.Id); err != nil {
		logger.Error("failed to delete category", "id", id, "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "deleted"})
}

// ===== Tag Endpoints =====

func (c *Controller) CreateTag(w http.ResponseWriter, r *http.Request) {
	u, _ := auth.UserFromContext(r.Context())

	var tc TagCreate
	if err := json.NewDecoder(r.Body).Decode(&tc); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	tc.UserId = &u.Id

	if tc.NameRu == "" {
		http.Error(w, "nameRu is required", http.StatusBadRequest)
		return
	}

	tag, err := c.service.CreateTag(context.Background(), tc)
	if err != nil {
		logger.Error("failed to create tag", "error", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(tag)
}

func (c *Controller) UpdateTag(w http.ResponseWriter, r *http.Request) {
	u, _ := auth.UserFromContext(r.Context())

	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "invalid tag id", http.StatusBadRequest)
		return
	}

	var tu TagUpdate
	if err := json.NewDecoder(r.Body).Decode(&tu); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	tu.Id = id
	tu.UserId = u.Id

	tag, err := c.service.UpdateTag(context.Background(), tu)
	if err != nil {
		logger.Error("failed to update tag", "id", id, "error", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(tag)
}

func (c *Controller) DeleteTag(w http.ResponseWriter, r *http.Request) {
	u, _ := auth.UserFromContext(r.Context())

	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "invalid tag id", http.StatusBadRequest)
		return
	}

	if err := c.service.DeleteTag(context.Background(), id, u.Id); err != nil {
		logger.Error("failed to delete tag", "id", id, "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "deleted"})
}

// ===== Admin Endpoints =====

func (c *Controller) GetNutriRecipeById(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "invalid recipe id", http.StatusBadRequest)
		return
	}

	recipe, err := c.service.GetNutriRecipeById(context.Background(), id)
	if err != nil {
		logger.Error("failed to get Nutri recipe by id", "id", id, "error", err)
		http.Error(w, "recipe not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(recipe)
}

func (c *Controller) CreateNutriRecipe(w http.ResponseWriter, r *http.Request) {
	var rc RecipeCreate
	if err := json.NewDecoder(r.Body).Decode(&rc); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if rc.TitleRu == "" {
		http.Error(w, "titleRu is required", http.StatusBadRequest)
		return
	}

	recipe, err := c.service.CreateNutriRecipe(context.Background(), rc)
	if err != nil {
		logger.Error("failed to create Nutri recipe", "error", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(recipe)
}

func (c *Controller) UpdateNutriRecipe(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "invalid recipe id", http.StatusBadRequest)
		return
	}

	var ru RecipeUpdate
	if err := json.NewDecoder(r.Body).Decode(&ru); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	ru.Id = id

	recipe, err := c.service.UpdateNutriRecipe(context.Background(), ru)
	if err != nil {
		logger.Error("failed to update Nutri recipe", "id", id, "error", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(recipe)
}

func (c *Controller) DeleteNutriRecipe(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "invalid recipe id", http.StatusBadRequest)
		return
	}

	if err := c.service.DeleteNutriRecipe(context.Background(), id); err != nil {
		logger.Error("failed to delete Nutri recipe", "id", id, "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "deleted"})
}

func (c *Controller) GetAllNutriRecipes(w http.ResponseWriter, r *http.Request) {
	params := parseRecipeListParams(r)

	// Get Nutri book and all recipes (including unpublished)
	book, err := c.service.GetNutriBook(context.Background())
	if err != nil {
		logger.Error("failed to get Nutri book", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	params.BookId = &book.Id

	response, err := c.service.GetRecipes(context.Background(), params)
	if err != nil {
		logger.Error("failed to get Nutri recipes", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (c *Controller) CreateSystemCategory(w http.ResponseWriter, r *http.Request) {
	var cc CategoryCreate
	if err := json.NewDecoder(r.Body).Decode(&cc); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if cc.NameRu == "" || cc.Slug == nil || *cc.Slug == "" {
		http.Error(w, "nameRu and slug are required", http.StatusBadRequest)
		return
	}

	category, err := c.service.CreateSystemCategory(context.Background(), cc)
	if err != nil {
		logger.Error("failed to create system category", "error", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(category)
}

func (c *Controller) CreateSystemTag(w http.ResponseWriter, r *http.Request) {
	var tc TagCreate
	if err := json.NewDecoder(r.Body).Decode(&tc); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if tc.NameRu == "" {
		http.Error(w, "nameRu is required", http.StatusBadRequest)
		return
	}

	tag, err := c.service.CreateSystemTag(context.Background(), tc)
	if err != nil {
		logger.Error("failed to create system tag", "error", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(tag)
}

func (c *Controller) UpdateSystemTag(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "invalid tag id", http.StatusBadRequest)
		return
	}

	var tu TagUpdate
	if err := json.NewDecoder(r.Body).Decode(&tu); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	tu.Id = id

	if tu.NameRu == "" {
		http.Error(w, "nameRu is required", http.StatusBadRequest)
		return
	}

	tag, err := c.service.UpdateSystemTag(context.Background(), tu)
	if err != nil {
		logger.Error("failed to update system tag", "id", id, "error", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(tag)
}

func (c *Controller) DeleteSystemTag(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "invalid tag id", http.StatusBadRequest)
		return
	}

	if err := c.service.DeleteSystemTag(context.Background(), id); err != nil {
		logger.Error("failed to delete system tag", "id", id, "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "deleted"})
}

// ===== Upload =====

func (c *Controller) UploadImage(w http.ResponseWriter, r *http.Request) {
	// Parse multipart form (max 10MB)
	err := r.ParseMultipartForm(10 << 20)
	if err != nil {
		http.Error(w, "failed to parse form", http.StatusBadRequest)
		return
	}

	file, header, err := r.FormFile("image")
	if err != nil {
		http.Error(w, "image file is required", http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Read image data
	imageData, err := io.ReadAll(file)
	if err != nil {
		http.Error(w, "failed to read image", http.StatusBadRequest)
		return
	}

	// Upload to MinIO
	imageUrl, err := c.service.UploadImage(context.Background(), imageData, header.Filename)
	if err != nil {
		logger.Error("failed to upload image", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(ImageUploadResponse{Url: imageUrl})
}
