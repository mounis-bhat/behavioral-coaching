package recipes

import "context"

// Generator defines the AI capability for recipe generation.
type Generator interface {
	Generate(ctx context.Context, req RecipeRequest) (*Recipe, error)
}
