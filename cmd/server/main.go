package main

import (
	"context"
	"log"
	"net/http"
	"time"

	airecipes "github.com/mounis-bhat/starter/internal/ai/recipes"
	"github.com/mounis-bhat/starter/internal/api"
	apprecipes "github.com/mounis-bhat/starter/internal/app/recipes"
	"github.com/mounis-bhat/starter/internal/config"
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

	recipeGenerator := airecipes.NewGenkitGenerator(g)
	recipeService := apprecipes.NewService(recipeGenerator)

	store, err := storage.New(ctx, cfg.Database)
	if err != nil {
		log.Fatal(err)
	}
	defer store.Close()

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

	auditCleanup := service.NewAuditCleanupService(store.Queries)
	cronScheduler := cron.New()
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
			cronScheduler.Start()
			defer cronScheduler.Stop()
		}
	} else {
		log.Printf("audit cleanup job disabled (cron=%q retention_days=%d)", cfg.Audit.CleanupCron, cfg.Audit.RetentionDays)
	}

	// Setup router
	mux := api.NewRouter(cfg, store, recipeService, blobClient)
	root := http.NewServeMux()
	root.Handle("/", api.WithSecurityHeaders(cfg, mux))

	log.Printf("Starting server on http://localhost:%s", cfg.Port)
	log.Fatal(server.Start(ctx, "127.0.0.1:"+cfg.Port, root))
}
