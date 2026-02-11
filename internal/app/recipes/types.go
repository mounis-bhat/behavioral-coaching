package recipes

// RecipeRequest represents the input for recipe generation.
type RecipeRequest struct {
	Ingredient          string `json:"ingredient" jsonschema:"description=Main ingredient or cuisine type" example:"chicken" validate:"required"`
	DietaryRestrictions string `json:"dietaryRestrictions,omitempty" jsonschema:"description=Any dietary restrictions" example:"gluten-free"`
}

// Recipe represents a generated recipe.
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
