package recipes

import (
	"context"
	"fmt"

	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/core"
	"github.com/firebase/genkit/go/genkit"
	apprecipes "github.com/mounis-bhat/starter/internal/app/recipes"
)

// GenkitGenerator wraps a Genkit flow for recipe generation.
type GenkitGenerator struct {
	flow *core.Flow[*apprecipes.RecipeRequest, *apprecipes.Recipe, struct{}]
}

func NewGenkitGenerator(g *genkit.Genkit) *GenkitGenerator {
	flow := genkit.DefineFlow(g, "recipeGeneratorFlow", func(ctx context.Context, input *apprecipes.RecipeRequest) (*apprecipes.Recipe, error) {
		dietaryRestrictions := input.DietaryRestrictions
		if dietaryRestrictions == "" {
			dietaryRestrictions = "none"
		}

		prompt := fmt.Sprintf(`Create a recipe with the following requirements:
			Main ingredient: %s
			Dietary restrictions: %s`, input.Ingredient, dietaryRestrictions)

		recipe, _, err := genkit.GenerateData[apprecipes.Recipe](ctx, g, ai.WithPrompt(prompt))
		if err != nil {
			return nil, fmt.Errorf("failed to generate recipe: %w", err)
		}

		return recipe, nil
	})

	return &GenkitGenerator{flow: flow}
}

func (g *GenkitGenerator) Generate(ctx context.Context, req apprecipes.RecipeRequest) (*apprecipes.Recipe, error) {
	return g.flow.Run(ctx, &req)
}
