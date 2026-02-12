package behavior

import "context"

// PlanGenerator generates daily plan tasks from a behavior profile.
type PlanGenerator interface {
	GeneratePlan(ctx context.Context, req PlanRequest) (*PlanResult, error)
	PromptVersion() string
}

// AdaptationAdvisor recommends plan adaptations based on adherence metrics.
type AdaptationAdvisor interface {
	Advise(ctx context.Context, req AdaptationRequest) (*AdaptationResult, error)
}
