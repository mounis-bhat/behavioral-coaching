package api

import (
	"encoding/json"
	"net/http"

	apprecipes "github.com/mounis-bhat/starter/internal/app/recipes"
)

// RecipeRequest represents the input for recipe generation.
// @Description Recipe generation request
type RecipeRequest struct {
	Ingredient          string `json:"ingredient" jsonschema:"description=Main ingredient or cuisine type" example:"chicken" validate:"required"`
	DietaryRestrictions string `json:"dietaryRestrictions,omitempty" jsonschema:"description=Any dietary restrictions" example:"gluten-free"`
}

// Recipe represents a generated recipe.
// @Description Generated recipe
type Recipe struct {
	Title        string   `json:"title" example:"Grilled Lemon Herb Chicken" validate:"required"`
	Description  string   `json:"description" example:"A delicious and healthy grilled chicken recipe" validate:"required"`
	PrepTime     string   `json:"prepTime" example:"15 minutes" validate:"required"`
	CookTime     string   `json:"cookTime" example:"25 minutes" validate:"required"`
	Servings     int      `json:"servings" example:"4" validate:"required"`
	Ingredients  []string `json:"ingredients" example:"chicken breast,lemon,herbs" validate:"required"`
	Instructions []string `json:"instructions" example:"Marinate chicken,Preheat grill,Grill for 12 minutes" validate:"required"`
	Tips         []string `json:"tips,omitempty" example:"Let rest for 5 minutes before serving"`
}

// makeRecipeHandler creates a handler for recipe generation using Genkit flow
// @Summary      Generate a recipe
// @Description  Uses AI to generate a recipe based on ingredients and dietary restrictions
// @Tags         recipes
// @Accept       json
// @Produce      json
// @Param        request body RecipeRequest true "Recipe generation request"
// @Success      200  {object}  Recipe
// @Failure      400  {object}  map[string]string
// @Failure      500  {object}  map[string]string
// @Router       /recipes/generate [post]
func makeRecipeHandler(service *apprecipes.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req RecipeRequest
		r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
		decoder := json.NewDecoder(r.Body)
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&req); err != nil {
			http.Error(w, "invalid JSON body", http.StatusBadRequest)
			return
		}
		if req.Ingredient == "" {
			http.Error(w, "ingredient is required", http.StatusBadRequest)
			return
		}
		if err := decoder.Decode(&struct{}{}); err == nil {
			http.Error(w, "invalid JSON body", http.StatusBadRequest)
			return
		}

		recipe, err := service.Generate(r.Context(), apprecipes.RecipeRequest{
			Ingredient:          req.Ingredient,
			DietaryRestrictions: req.DietaryRestrictions,
		})
		if err != nil {
			http.Error(w, "failed to generate recipe", http.StatusInternalServerError)
			return
		}

		response := Recipe{
			Title:        recipe.Title,
			Description:  recipe.Description,
			PrepTime:     recipe.PrepTime,
			CookTime:     recipe.CookTime,
			Servings:     recipe.Servings,
			Ingredients:  recipe.Ingredients,
			Instructions: recipe.Instructions,
			Tips:         recipe.Tips,
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(response); err != nil {
			http.Error(w, "failed to write response", http.StatusInternalServerError)
			return
		}
	}
}
