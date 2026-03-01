package main

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/mounis-bhat/starter/internal/ai/behavior"
	"github.com/mounis-bhat/starter/internal/api"
	appbehavior "github.com/mounis-bhat/starter/internal/app/behavior"
	"github.com/mounis-bhat/starter/internal/app/chat"
	"github.com/mounis-bhat/starter/internal/app/journal"
	"github.com/mounis-bhat/starter/internal/config"
	"github.com/mounis-bhat/starter/internal/email"
	"github.com/mounis-bhat/starter/internal/service"
	"github.com/mounis-bhat/starter/internal/storage"
	"github.com/mounis-bhat/starter/internal/storage/blob"

	"github.com/firebase/genkit/go/genkit"
	"github.com/firebase/genkit/go/plugins/googlegenai"
	"github.com/firebase/genkit/go/plugins/server"
	"github.com/robfig/cron/v3"
)

// @title           API
// @version         1.0
// @description     API server

// @BasePath  /api

func main() {
	ctx := context.Background()
	cfg := config.Load()

	// Initialize Genkit with the Google AI plugin
	g := genkit.Init(ctx,
		genkit.WithPlugins(&googlegenai.GoogleAI{}),
		genkit.WithDefaultModel("googleai/gemini-2.5-flash"),
	)

	store, err := storage.New(ctx, cfg.Database)
	if err != nil {
		log.Fatal(err)
	}
	defer store.Close()

	planningAgent := behavior.NewPlanningAgent(g)
	adaptationAdvisor := behavior.NewAdaptationAdvisor(g)
	profilingAgent := behavior.NewProfilingAgent(g)
	coachingAgent := behavior.NewCoachingAgent(g)
	behaviorService := appbehavior.NewService(store.Queries, planningAgent, adaptationAdvisor, profilingAgent)
	journalService := journal.NewService(store.Queries)
	chatService := chat.NewService(store.Queries, coachingAgent)

	blobClient, err := blob.New(ctx, blob.Config{
		Endpoint:           cfg.Storage.Endpoint,
		Region:             cfg.Storage.Region,
		Bucket:             cfg.Storage.Bucket,
		AccessKeyID:        cfg.Storage.AccessKeyID,
		SecretAccessKey:    cfg.Storage.SecretAccessKey,
		ForcePathStyle:     cfg.Storage.ForcePathStyle,
		PresignUploadTTL:   cfg.Storage.PresignUploadTTL,
		PresignDownloadTTL: cfg.Storage.PresignDownloadTTL,
	})
	if err != nil {
		log.Printf("blob storage disabled: %v", err)
		blobClient = nil
	}

	// Initialize mailer (shared by router and scheduler)
	var mailer email.Mailer
	gmailMailer, err := email.NewGmailMailer(cfg.Email.ContactEmail, cfg.Email.GmailAppPassword)
	if err != nil {
		log.Printf("email disabled: %v", err)
	} else {
		mailer = gmailMailer
	}

	// Cron scheduler
	cronScheduler := cron.New()
	hasJobs := false

	// Audit cleanup job
	auditCleanup := service.NewAuditCleanupService(store.Queries)
	if cfg.Audit.CleanupCron != "" && cfg.Audit.RetentionDays > 0 {
		_, err = cronScheduler.AddFunc(cfg.Audit.CleanupCron, func() {
			jobCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
			defer cancel()

			cutoff := time.Now().AddDate(0, 0, -cfg.Audit.RetentionDays)
			deleted, err := auditCleanup.PurgeBefore(jobCtx, cutoff)
			if err != nil {
				log.Printf("audit cleanup failed: %v", err)
				return
			}

			log.Printf("audit cleanup complete: deleted=%d cutoff=%s", deleted, cutoff.Format(time.RFC3339))
		})
		if err != nil {
			log.Printf("invalid audit cleanup cron schedule: %s error=%v", cfg.Audit.CleanupCron, err)
		} else {
			hasJobs = true
		}
	} else {
		log.Printf("audit cleanup job disabled (cron=%q retention_days=%d)", cfg.Audit.CleanupCron, cfg.Audit.RetentionDays)
	}

	// Behavior scheduler jobs
	behaviorScheduler := service.NewBehaviorSchedulerService(store.Queries, mailer, cfg.Email.AppBaseURL)

	if cfg.Behavior.PlanExpiryCron != "" {
		_, err = cronScheduler.AddFunc(cfg.Behavior.PlanExpiryCron, func() {
			jobCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
			defer cancel()

			expired, err := behaviorScheduler.ExpireOldPlans(jobCtx)
			if err != nil {
				log.Printf("plan expiry failed: %v", err)
				return
			}
			if expired > 0 {
				log.Printf("plan expiry complete: expired=%d", expired)
			}
		})
		if err != nil {
			log.Printf("invalid plan expiry cron schedule: %s error=%v", cfg.Behavior.PlanExpiryCron, err)
		} else {
			hasJobs = true
		}
	}

	if cfg.Behavior.ReminderCron != "" {
		_, err = cronScheduler.AddFunc(cfg.Behavior.ReminderCron, func() {
			jobCtx, cancel := context.WithTimeout(ctx, 10*time.Minute)
			defer cancel()

			sent, err := behaviorScheduler.SendDailyReminders(jobCtx)
			if err != nil {
				log.Printf("daily reminders failed: %v", err)
				return
			}
			if sent > 0 {
				log.Printf("daily reminders sent: count=%d", sent)
			}
		})
		if err != nil {
			log.Printf("invalid reminder cron schedule: %s error=%v", cfg.Behavior.ReminderCron, err)
		} else {
			hasJobs = true
		}
	}

	if cfg.Behavior.RelapseCron != "" {
		_, err = cronScheduler.AddFunc(cfg.Behavior.RelapseCron, func() {
			jobCtx, cancel := context.WithTimeout(ctx, 10*time.Minute)
			defer cancel()

			// End-of-day AI adaptation: evaluate all users with today's data
			if err := behaviorService.RunEndOfDayAdaptation(jobCtx); err != nil {
				log.Printf("end-of-day adaptation failed: %v", err)
			}

			handled, err := behaviorScheduler.DetectAndHandleRelapses(jobCtx)
			if err != nil {
				log.Printf("relapse detection failed: %v", err)
				return
			}
			if handled > 0 {
				log.Printf("relapse detection complete: handled=%d", handled)
			}
		})
		if err != nil {
			log.Printf("invalid relapse cron schedule: %s error=%v", cfg.Behavior.RelapseCron, err)
		} else {
			hasJobs = true
		}
	}

	if hasJobs {
		cronScheduler.Start()
		defer cronScheduler.Stop()
	}

	// Setup router
	mux := api.NewRouter(cfg, store, behaviorService, journalService, chatService, blobClient, mailer)
	root := http.NewServeMux()
	root.Handle("/", api.WithSecurityHeaders(cfg, mux))

	log.Printf("Starting server on http://localhost:%s", cfg.Port)
	log.Fatal(server.Start(ctx, "127.0.0.1:"+cfg.Port, root))
}
