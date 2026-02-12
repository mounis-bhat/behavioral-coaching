package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math/big"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/mounis-bhat/starter/internal/email"
	"github.com/mounis-bhat/starter/internal/storage/db"
)

type BehaviorSchedulerService struct {
	queries *db.Queries
	mailer  email.Mailer
	baseURL string
}

func NewBehaviorSchedulerService(queries *db.Queries, mailer email.Mailer, baseURL string) *BehaviorSchedulerService {
	return &BehaviorSchedulerService{queries: queries, mailer: mailer, baseURL: baseURL}
}

// ExpireOldPlans marks all plans from previous days as expired.
func (s *BehaviorSchedulerService) ExpireOldPlans(ctx context.Context) (int64, error) {
	if s == nil || s.queries == nil {
		return 0, errors.New("behavior scheduler not initialized")
	}
	return s.queries.ExpireOldPlans(ctx)
}

// SendDailyReminders emails users who have today's plan but haven't completed any tasks.
func (s *BehaviorSchedulerService) SendDailyReminders(ctx context.Context) (int, error) {
	if s == nil || s.queries == nil {
		return 0, errors.New("behavior scheduler not initialized")
	}
	if s.mailer == nil {
		log.Printf("behavior scheduler: mailer not configured, skipping reminders")
		return 0, nil
	}

	users, err := s.queries.GetUsersNeedingReminder(ctx)
	if err != nil {
		return 0, fmt.Errorf("get users needing reminder: %w", err)
	}

	var sent int
	for _, u := range users {
		params := email.EmailParams{
			Greeting:   fmt.Sprintf("Hi %s,", u.Name),
			BodyLines:  []string{"You have tasks waiting for you today. Check in and make progress on your goals."},
			ButtonText: "View today's plan",
			ButtonURL:  s.baseURL + "/dashboard",
			FooterText: "Stay consistent — small steps compound over time.",
		}

		textBody := email.RenderText(params)
		htmlBody := email.RenderHTML(params)

		if err := s.mailer.Send(ctx, u.Email, "You have tasks waiting", textBody, htmlBody); err != nil {
			log.Printf("behavior scheduler: failed to send reminder to %s: %v", u.Email, err)
			continue
		}

		if err := s.queries.UpdateProfileNotifiedAt(ctx, u.ID); err != nil {
			log.Printf("behavior scheduler: failed to update notified_at for user %s: %v", u.Email, err)
		}
		sent++
	}

	return sent, nil
}

// DetectAndHandleRelapses finds users with declining adherence or inactivity,
// reduces their difficulty, creates adaptation logs, and sends recovery emails.
func (s *BehaviorSchedulerService) DetectAndHandleRelapses(ctx context.Context) (int, error) {
	if s == nil || s.queries == nil {
		return 0, errors.New("behavior scheduler not initialized")
	}

	// Collect relapsing users (low completion rate)
	relapsing, err := s.queries.GetRelapsingUsers(ctx)
	if err != nil {
		return 0, fmt.Errorf("get relapsing users: %w", err)
	}

	// Collect inactive users (no plans in 3+ days)
	inactive, err := s.queries.GetInactiveUsers(ctx)
	if err != nil {
		return 0, fmt.Errorf("get inactive users: %w", err)
	}

	// Deduplicate by user ID
	seen := make(map[[16]byte]bool)
	type relapseTarget struct {
		id         pgtype.UUID
		email      string
		name       string
		reason     string
		difficulty float64
	}
	var targets []relapseTarget

	for _, u := range relapsing {
		if seen[u.ID.Bytes] {
			continue
		}
		seen[u.ID.Bytes] = true
		rate := numericToFloat(u.CompletionRate)
		targets = append(targets, relapseTarget{
			id:         u.ID,
			email:      u.Email,
			name:       u.Name,
			reason:     fmt.Sprintf("Automatic relapse recovery: completion rate dropped to %.0f%%", rate*100),
			difficulty: numericToFloat(u.DifficultyLevel),
		})
	}

	for _, u := range inactive {
		if seen[u.ID.Bytes] {
			continue
		}
		seen[u.ID.Bytes] = true
		// Need to get their current difficulty
		profile, err := s.queries.GetBehaviorProfileByUserID(ctx, u.ID)
		if err != nil {
			log.Printf("behavior scheduler: failed to get profile for inactive user %s: %v", u.Email, err)
			continue
		}
		targets = append(targets, relapseTarget{
			id:         u.ID,
			email:      u.Email,
			name:       u.Name,
			reason:     "Automatic relapse recovery: no activity for 3+ days",
			difficulty: numericToFloat(profile.DifficultyLevel),
		})
	}

	var handled int
	for _, t := range targets {
		newDifficulty := t.difficulty - 1.0
		if newDifficulty < 1.0 {
			newDifficulty = 1.0
		}
		if newDifficulty == t.difficulty {
			// Already at minimum, just notify
			s.sendRelapseEmail(ctx, t.email, t.name)
			if err := s.queries.UpdateProfileNotifiedAt(ctx, t.id); err != nil {
				log.Printf("behavior scheduler: failed to update notified_at: %v", err)
			}
			handled++
			continue
		}

		difficultyChange := newDifficulty - t.difficulty

		// Update profile difficulty
		_, err := s.queries.UpdateBehaviorProfile(ctx, db.UpdateBehaviorProfileParams{
			UserID:          t.id,
			DifficultyLevel: numericFromFloat(newDifficulty),
		})
		if err != nil {
			log.Printf("behavior scheduler: failed to update difficulty for %s: %v", t.email, err)
			continue
		}

		// Create adaptation log
		triggerMetrics, _ := json.Marshal(map[string]any{
			"source":              "relapse_detection",
			"previous_difficulty": t.difficulty,
			"new_difficulty":      newDifficulty,
		})
		_, err = s.queries.CreateAdaptationLog(ctx, db.CreateAdaptationLogParams{
			UserID:             t.id,
			AdaptationReason:   t.reason,
			DifficultyChange:   numericFromFloat(difficultyChange),
			PreviousDifficulty: numericFromFloat(t.difficulty),
			NewDifficulty:      numericFromFloat(newDifficulty),
			TriggerMetrics:     triggerMetrics,
		})
		if err != nil {
			log.Printf("behavior scheduler: failed to create adaptation log for %s: %v", t.email, err)
		}

		s.sendRelapseEmail(ctx, t.email, t.name)

		if err := s.queries.UpdateProfileNotifiedAt(ctx, t.id); err != nil {
			log.Printf("behavior scheduler: failed to update notified_at: %v", err)
		}

		handled++
	}

	return handled, nil
}

func (s *BehaviorSchedulerService) sendRelapseEmail(ctx context.Context, toEmail, name string) {
	if s.mailer == nil {
		return
	}
	params := email.EmailParams{
		Greeting:   fmt.Sprintf("Hi %s,", name),
		BodyLines: []string{
			"We noticed you've been having a tough stretch.",
			"We've lowered your plan difficulty to help you get back on track. Focus on building momentum — even one completed task is a win.",
		},
		ButtonText: "View your plan",
		ButtonURL:  s.baseURL + "/dashboard",
		FooterText: "Every day is a fresh start.",
	}

	textBody := email.RenderText(params)
	htmlBody := email.RenderHTML(params)

	if err := s.mailer.Send(ctx, toEmail, "Let's get back on track", textBody, htmlBody); err != nil {
		log.Printf("behavior scheduler: failed to send relapse email to %s: %v", toEmail, err)
	}
}

// numericFromFloat converts a float64 to pgtype.Numeric (1 decimal place).
func numericFromFloat(f float64) pgtype.Numeric {
	bf := new(big.Float).SetFloat64(f)
	bf.Mul(bf, new(big.Float).SetInt64(10))
	bi, _ := bf.Int(nil)
	return pgtype.Numeric{Int: bi, Exp: -1, Valid: true}
}

// numericToFloat converts a pgtype.Numeric to float64.
func numericToFloat(n pgtype.Numeric) float64 {
	if !n.Valid || n.Int == nil {
		return 0
	}
	f, _ := new(big.Float).SetInt(n.Int).Float64()
	for i := int32(0); i < -n.Exp; i++ {
		f /= 10
	}
	for i := int32(0); i < n.Exp; i++ {
		f *= 10
	}
	return f
}
