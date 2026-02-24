package recipe

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/jourloy/nutri-backend/internal/auth"
	"github.com/jourloy/nutri-backend/internal/user"
)

type fakeRecipeService struct {
	Service

	getNutriRecipeByIdFn func(ctx context.Context, id int64) (*Recipe, error)
	updateNutriRecipeFn  func(ctx context.Context, r RecipeUpdate) (*Recipe, error)
	deleteNutriRecipeFn  func(ctx context.Context, id int64) error
	createSystemTagFn    func(ctx context.Context, t TagCreate) (*Tag, error)
}

func (f fakeRecipeService) GetNutriRecipeById(ctx context.Context, id int64) (*Recipe, error) {
	if f.getNutriRecipeByIdFn != nil {
		return f.getNutriRecipeByIdFn(ctx, id)
	}
	return &Recipe{Id: id, TitleRu: "Recipe", Servings: 1, BookId: 1}, nil
}

func (f fakeRecipeService) UpdateNutriRecipe(ctx context.Context, r RecipeUpdate) (*Recipe, error) {
	if f.updateNutriRecipeFn != nil {
		return f.updateNutriRecipeFn(ctx, r)
	}
	return &Recipe{Id: r.Id, TitleRu: r.TitleRu, Servings: r.Servings, BookId: 1}, nil
}

func (f fakeRecipeService) DeleteNutriRecipe(ctx context.Context, id int64) error {
	if f.deleteNutriRecipeFn != nil {
		return f.deleteNutriRecipeFn(ctx, id)
	}
	return nil
}

func (f fakeRecipeService) CreateSystemTag(ctx context.Context, t TagCreate) (*Tag, error) {
	if f.createSystemTagFn != nil {
		return f.createSystemTagFn(ctx, t)
	}
	return &Tag{Id: 1, NameRu: t.NameRu, TagType: "system"}, nil
}

func withUser(req *http.Request, isAdmin bool) *http.Request {
	return req.WithContext(auth.ContextWithUser(req.Context(), user.User{
		Id:      "u1",
		IsAdmin: isAdmin,
	}))
}

func TestAdminGetNutriRecipeById_Unauthorized(t *testing.T) {
	r := chi.NewRouter()
	c := &Controller{service: fakeRecipeService{}}
	c.RegisterRoutes(r)

	req := httptest.NewRequest(http.MethodGet, "/recipe/admin/recipes/1", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected %d, got %d", http.StatusUnauthorized, rr.Code)
	}
}

func TestAdminGetNutriRecipeById_ForbiddenForNonAdmin(t *testing.T) {
	r := chi.NewRouter()
	c := &Controller{service: fakeRecipeService{}}
	c.RegisterRoutes(r)

	req := httptest.NewRequest(http.MethodGet, "/recipe/admin/recipes/1", nil)
	req = withUser(req, false)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected %d, got %d", http.StatusForbidden, rr.Code)
	}
}

func TestAdminGetNutriRecipeById_Success(t *testing.T) {
	r := chi.NewRouter()
	c := &Controller{service: fakeRecipeService{
		getNutriRecipeByIdFn: func(ctx context.Context, id int64) (*Recipe, error) {
			return &Recipe{Id: id, TitleRu: "Омлет", Servings: 2, BookId: 10}, nil
		},
	}}
	c.RegisterRoutes(r)

	req := httptest.NewRequest(http.MethodGet, "/recipe/admin/recipes/7", nil)
	req = withUser(req, true)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected %d, got %d: %s", http.StatusOK, rr.Code, rr.Body.String())
	}

	var out Recipe
	if err := json.NewDecoder(rr.Body).Decode(&out); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if out.Id != 7 || out.TitleRu != "Омлет" {
		t.Fatalf("unexpected response: %+v", out)
	}
}

func TestAdminUpdateNutriRecipe_SuccessUpdatesFields(t *testing.T) {
	r := chi.NewRouter()
	c := &Controller{service: fakeRecipeService{
		updateNutriRecipeFn: func(ctx context.Context, ru RecipeUpdate) (*Recipe, error) {
			return &Recipe{
				Id:       ru.Id,
				TitleRu:  ru.TitleRu,
				Servings: ru.Servings,
				BookId:   10,
			}, nil
		},
	}}
	c.RegisterRoutes(r)

	req := httptest.NewRequest(http.MethodPut, "/recipe/admin/recipes/5", strings.NewReader(`{"titleRu":"Новый рецепт","servings":3}`))
	req.Header.Set("Content-Type", "application/json")
	req = withUser(req, true)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected %d, got %d: %s", http.StatusOK, rr.Code, rr.Body.String())
	}

	var out Recipe
	if err := json.NewDecoder(rr.Body).Decode(&out); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if out.Id != 5 || out.TitleRu != "Новый рецепт" || out.Servings != 3 {
		t.Fatalf("unexpected response: %+v", out)
	}
}

func TestAdminDeleteNutriRecipe_SoftDeletePath(t *testing.T) {
	r := chi.NewRouter()
	called := false
	c := &Controller{service: fakeRecipeService{
		deleteNutriRecipeFn: func(ctx context.Context, id int64) error {
			called = true
			if id != 9 {
				t.Fatalf("expected id=9, got %d", id)
			}
			return nil
		},
	}}
	c.RegisterRoutes(r)

	req := httptest.NewRequest(http.MethodDelete, "/recipe/admin/recipes/9", nil)
	req = withUser(req, true)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected %d, got %d: %s", http.StatusOK, rr.Code, rr.Body.String())
	}
	if !called {
		t.Fatalf("expected delete service method to be called")
	}
}

func TestAdminCreateSystemTag_Success(t *testing.T) {
	r := chi.NewRouter()
	c := &Controller{service: fakeRecipeService{
		createSystemTagFn: func(ctx context.Context, t TagCreate) (*Tag, error) {
			return &Tag{Id: 12, NameRu: t.NameRu, TagType: "system"}, nil
		},
	}}
	c.RegisterRoutes(r)

	req := httptest.NewRequest(http.MethodPost, "/recipe/admin/tags", strings.NewReader(`{"nameRu":"Новый тег"}`))
	req.Header.Set("Content-Type", "application/json")
	req = withUser(req, true)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected %d, got %d: %s", http.StatusCreated, rr.Code, rr.Body.String())
	}

	var out Tag
	if err := json.NewDecoder(rr.Body).Decode(&out); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if out.TagType != "system" || out.NameRu != "Новый тег" {
		t.Fatalf("unexpected response: %+v", out)
	}
}
