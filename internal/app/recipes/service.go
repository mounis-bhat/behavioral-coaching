package recipes

import "context"

// Service orchestrates recipe generation.
type Service struct {
	generator Generator
}

func NewService(generator Generator) *Service {
	return &Service{generator: generator}
}

func (s *Service) Generate(ctx context.Context, req RecipeRequest) (*Recipe, error) {
	return s.generator.Generate(ctx, req)
}
