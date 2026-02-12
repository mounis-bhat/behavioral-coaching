package api

import (
	"net/http"

	"github.com/mounis-bhat/starter/internal/app/behavior"
	"github.com/mounis-bhat/starter/internal/config"
	"github.com/mounis-bhat/starter/internal/email"
	"github.com/mounis-bhat/starter/internal/ratelimit"
	"github.com/mounis-bhat/starter/internal/storage"
	"github.com/mounis-bhat/starter/internal/storage/blob"
)

func NewRouter(cfg *config.Config, store *storage.Store, behaviorService *behavior.Service, blobClient *blob.Client) *http.ServeMux {
	mux := http.NewServeMux()

	var limiter RateLimiter
	if cfg.RateLimit.Enabled {
		limiter = ratelimit.NewValkeyLimiter(cfg.Valkey.Addr(), cfg.Valkey.Password)
	}
	mailer, err := email.NewGmailMailer(cfg.Email.ContactEmail, cfg.Email.GmailAppPassword)
	if err != nil {
		mailer = nil
	}
	authHandler := NewAuthHandler(store, cfg.Auth, cfg.Google, cfg.Email, cfg.RateLimit, limiter, mailer)
	avatarHandler := NewAvatarHandler(store, blobClient, cfg.Storage)
	behaviorHandler := NewBehaviorHandler(behaviorService)

	// API routes
	mux.HandleFunc("GET /api/health", handleHealth)

	// Auth routes
	mux.HandleFunc("POST /api/auth/register", authHandler.HandleRegister)
	mux.HandleFunc("POST /api/auth/login", authHandler.HandleLogin)
	mux.HandleFunc("GET /api/auth/google", authHandler.HandleGoogleLogin)
	mux.HandleFunc("GET /api/auth/google/callback", authHandler.HandleGoogleCallback)
	mux.HandleFunc("GET /api/auth/verify-email", authHandler.HandleVerifyEmail)
	mux.Handle("GET /api/auth/me", authHandler.RequireAuth(http.HandlerFunc(authHandler.HandleMe)))
	mux.Handle("GET /api/auth/avatar-url", authHandler.RequireAuth(http.HandlerFunc(avatarHandler.HandleAvatarURL)))
	mux.Handle("POST /api/auth/avatar/upload-url", authHandler.RequireAuth(http.HandlerFunc(avatarHandler.HandleAvatarUploadURL)))
	mux.Handle("POST /api/auth/avatar/confirm", authHandler.RequireAuth(http.HandlerFunc(avatarHandler.HandleAvatarConfirm)))
	mux.Handle("POST /api/auth/logout", authHandler.RequireAuth(http.HandlerFunc(authHandler.HandleLogout)))
	mux.Handle("POST /api/auth/password", authHandler.RequireAuth(http.HandlerFunc(authHandler.HandleChangePassword)))
	mux.Handle("POST /api/auth/verify-email/resend", authHandler.RequireAuth(http.HandlerFunc(authHandler.HandleResendVerification)))

	// Behavior routes
	mux.Handle("POST /api/behavior/profile", authHandler.RequireAuth(http.HandlerFunc(behaviorHandler.HandleCreateProfile)))
	mux.Handle("GET /api/behavior/profile", authHandler.RequireAuth(http.HandlerFunc(behaviorHandler.HandleGetProfile)))
	mux.Handle("PUT /api/behavior/profile", authHandler.RequireAuth(http.HandlerFunc(behaviorHandler.HandleUpdateProfile)))
	mux.Handle("POST /api/behavior/plan/generate", authHandler.RequireAuth(http.HandlerFunc(behaviorHandler.HandleGeneratePlan)))
	mux.Handle("GET /api/behavior/plan/today", authHandler.RequireAuth(http.HandlerFunc(behaviorHandler.HandleGetTodaysPlan)))
	mux.Handle("POST /api/behavior/tasks/{taskID}/log", authHandler.RequireAuth(http.HandlerFunc(behaviorHandler.HandleLogExecution)))
	mux.Handle("GET /api/behavior/adherence", authHandler.RequireAuth(http.HandlerFunc(behaviorHandler.HandleGetAdherence)))

	// Documentation routes (dev only)
	if cfg.Env == "development" {
		mux.HandleFunc("GET /api/openapi.json", handleOpenAPISpec)
		mux.HandleFunc("GET /api/docs", handleScalarDocs)
		mux.HandleFunc("GET /api/docs/scalar.js", handleScalarScript)
	}

	// Static files (SPA) - served last as catch-all
	mux.Handle("/", staticHandler(cfg))

	return mux
}
