package api

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"io"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/mounis-bhat/starter/internal/config"
	"github.com/mounis-bhat/starter/internal/domain"
	"github.com/mounis-bhat/starter/internal/email"
	"github.com/mounis-bhat/starter/internal/storage"
	"github.com/mounis-bhat/starter/internal/storage/db"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

type contextKey string

const (
	contextKeyUser    contextKey = "authUser"
	contextKeySession contextKey = "authSession"
)

const (
	oauthStateCookieName    = "oauth_state"
	oauthVerifierCookieName = "oauth_verifier"
	oauthCookieMaxAge       = 5 * time.Minute
)

const (
	emailVerificationTokenSize = 32
	emailVerificationTTL       = 24 * time.Hour
)

type AuthHandler struct {
	queries              *db.Queries
	sessions             *domain.SessionService
	cookies              CookieManager
	oauthConfig          *oauth2.Config
	rateLimiter          RateLimiter
	rateLimits           config.RateLimitConfig
	auditLogger          *AuditLogger
	postLoginRedirectURL string
	mailer               email.Mailer
	appBaseURL           string
	trustedProxyHeader   string
}

type RateLimiter interface {
	Allow(ctx context.Context, key string, limit int, window time.Duration) (bool, error)
}

// AuthMeResponse represents the authenticated user
// @Description Authenticated user response
type AuthMeResponse struct {
	ID            string  `json:"id"`
	Email         string  `json:"email"`
	EmailVerified bool    `json:"email_verified"`
	Name          string  `json:"name"`
	Picture       *string `json:"picture,omitempty"`
	Provider      string  `json:"provider"`
}

// LogoutResponse represents a successful logout
// @Description Logout response
type LogoutResponse struct {
	Status string `json:"status" example:"ok"`
}

// RegisterRequest represents registration input
// @Description Registration request
type RegisterRequest struct {
	Email    string `json:"email" example:"user@example.com" validate:"required"`
	Password string `json:"password" example:"verysecurepassword" validate:"required"`
	Name     string `json:"name" example:"Jane Doe" validate:"required"`
}

// LoginRequest represents login input
// @Description Login request
type LoginRequest struct {
	Email    string `json:"email" example:"user@example.com" validate:"required"`
	Password string `json:"password" example:"verysecurepassword" validate:"required"`
}

// ChangePasswordRequest represents password change input
// @Description Password change request
type ChangePasswordRequest struct {
	CurrentPassword string `json:"current_password" validate:"required"`
	NewPassword     string `json:"new_password" validate:"required"`
}

// AuthStatusResponse represents a generic auth response
// @Description Auth status response
type AuthStatusResponse struct {
	Status string `json:"status" example:"ok"`
}

type googleUserInfo struct {
	Sub           string `json:"sub"`
	Email         string `json:"email"`
	EmailVerified bool   `json:"email_verified"`
	Name          string `json:"name"`
	Picture       string `json:"picture"`
}

func NewAuthHandler(store *storage.Store, cfg config.AuthConfig, googleCfg config.GoogleOAuthConfig, emailCfg config.EmailConfig, rateLimitCfg config.RateLimitConfig, limiter RateLimiter, mailer email.Mailer) *AuthHandler {
	var oauthConfig *oauth2.Config
	if googleCfg.ClientID != "" && googleCfg.ClientSecret != "" && googleCfg.RedirectURI != "" {
		oauthConfig = &oauth2.Config{
			ClientID:     googleCfg.ClientID,
			ClientSecret: googleCfg.ClientSecret,
			RedirectURL:  googleCfg.RedirectURI,
			Endpoint:     google.Endpoint,
			Scopes:       []string{"openid", "email", "profile"},
		}
	}

	postLoginRedirect := cfg.PostLoginRedirectURL
	if postLoginRedirect != "" {
		parsed, err := url.Parse(postLoginRedirect)
		if err != nil || (parsed.Scheme != "" && parsed.Scheme != "http" && parsed.Scheme != "https") {
			postLoginRedirect = "/"
		} else if parsed.Host != "" {
			baseURL := strings.TrimRight(emailCfg.AppBaseURL, "/")
			if baseParsed, err := url.Parse(baseURL); err != nil || parsed.Host != baseParsed.Host {
				postLoginRedirect = "/"
			}
		}
	}

	return &AuthHandler{
		queries:              store.Queries,
		sessions:             domain.NewSessionService(store.Queries, cfg.SessionMaxAge, cfg.IdleTimeout),
		cookies:              NewCookieManager(cfg),
		oauthConfig:          oauthConfig,
		rateLimiter:          limiter,
		rateLimits:           rateLimitCfg,
		auditLogger:          NewAuditLogger(store.Queries),
		postLoginRedirectURL: postLoginRedirect,
		mailer:               mailer,
		appBaseURL:           strings.TrimRight(emailCfg.AppBaseURL, "/"),
		trustedProxyHeader:   cfg.TrustedProxyHeader,
	}
}

func (h *AuthHandler) RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie(h.cookies.name)
		if err != nil || cookie.Value == "" {
			h.cookies.ClearSessionCookie(w)
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
			return
		}

		session, err := h.sessions.ValidateToken(r.Context(), cookie.Value)
		if err != nil {
			if errors.Is(err, domain.ErrSessionNotFound) || errors.Is(err, domain.ErrSessionExpired) {
				h.cookies.ClearSessionCookie(w)
				writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
				return
			}
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
			return
		}

		ctx := context.WithValue(r.Context(), contextKeySession, session)
		ctx = context.WithValue(ctx, contextKeyUser, session.User)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// HandleMe returns the authenticated user
// @Summary      Get current user
// @Description  Returns the authenticated user from the session cookie
// @Tags         auth
// @Produce      json
// @Success      200  {object}  AuthMeResponse
// @Failure      401  {object}  map[string]string
// @Failure      500  {object}  map[string]string
// @Router       /auth/me [get]
func (h *AuthHandler) HandleMe(w http.ResponseWriter, r *http.Request) {
	user, ok := userFromContext(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	writeJSON(w, http.StatusOK, AuthMeResponse{
		ID:            user.ID,
		Email:         user.Email,
		EmailVerified: user.EmailVerified,
		Name:          user.Name,
		Picture:       user.Picture,
		Provider:      user.Provider,
	})
}

// HandleLogout clears the session cookie and revokes the session
// @Summary      Logout
// @Description  Revokes the current session and clears the session cookie
// @Tags         auth
// @Produce      json
// @Success      200  {object}  LogoutResponse
// @Failure      401  {object}  map[string]string
// @Failure      500  {object}  map[string]string
// @Router       /auth/logout [post]
func (h *AuthHandler) HandleLogout(w http.ResponseWriter, r *http.Request) {
	session, ok := sessionFromContext(r.Context())
	if ok {
		if !h.allowRequest(r.Context(), "logout:"+session.TokenHash, r, h.rateLimits.Logout) {
			writeJSON(w, http.StatusTooManyRequests, map[string]string{"error": "too many requests"})
			return
		}
	}
	if ok {
		_ = h.sessions.RevokeByTokenHash(r.Context(), session.TokenHash)
	}

	h.cookies.ClearSessionCookie(w)
	if ok {
		h.auditLogger.Log(r.Context(), "session_revoked", uuidFromString(session.User.ID), h.ipFromRequest(r), r.UserAgent(), map[string]any{
			"reason":             "logout",
			"session_token_hash": session.TokenHash,
		})
		h.auditLogger.Log(r.Context(), "logout", uuidFromString(session.User.ID), h.ipFromRequest(r), r.UserAgent(), map[string]any{
			"session_token_hash": session.TokenHash,
		})
	}
	writeJSON(w, http.StatusOK, LogoutResponse{Status: "ok"})
}

// HandleRegister registers a new user with email/password
// @Summary      Register with credentials
// @Description  Creates a user account with email and password, then starts a session
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request body RegisterRequest true "Registration request"
// @Success      200  {object}  AuthStatusResponse
// @Failure      400  {object}  map[string]string
// @Failure      500  {object}  map[string]string
// @Router       /auth/register [post]
func (h *AuthHandler) HandleRegister(w http.ResponseWriter, r *http.Request) {
	if !h.allowRequest(r.Context(), "register", r, h.rateLimits.Register) {
		writeJSON(w, http.StatusTooManyRequests, map[string]string{"error": "too many requests"})
		return
	}

	var req RegisterRequest
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
		return
	}

	email, err := domain.NormalizeEmail(req.Email)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid email"})
		return
	}

	name := strings.TrimSpace(req.Name)
	if name == "" || len(name) > 255 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid name"})
		return
	}

	if err := domain.ValidatePassword(req.Password); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	if _, err := h.queries.GetUserByEmail(r.Context(), email); err == nil {
		h.auditLogger.Log(r.Context(), "register_duplicate", pgtype.UUID{}, h.ipFromRequest(r), r.UserAgent(), map[string]any{
			"email_hash": hashEmail(email),
		})
		writeJSON(w, http.StatusOK, AuthStatusResponse{Status: "ok"})
		return
	} else if !errors.Is(err, pgx.ErrNoRows) {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		return
	}

	hash, err := domain.HashPassword(req.Password)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		return
	}

	user, err := h.queries.CreateUser(r.Context(), db.CreateUserParams{
		Email:         email,
		EmailVerified: false,
		Name:          name,
		Picture:       pgtype.Text{},
		PasswordHash:  pgtype.Text{String: hash, Valid: true},
		Provider:      "credentials",
		GoogleID:      pgtype.Text{},
	})
	if err != nil {
		if isUniqueViolation(err) {
			h.auditLogger.Log(r.Context(), "register_duplicate", pgtype.UUID{}, h.ipFromRequest(r), r.UserAgent(), map[string]any{
				"email_hash": hashEmail(email),
			})
			writeJSON(w, http.StatusOK, AuthStatusResponse{Status: "ok"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		return
	}

	userAgent := r.UserAgent()
	ipAddress := h.ipFromRequest(r)
	if revoked := h.revokeExistingSession(r); revoked {
		h.auditLogger.Log(r.Context(), "session_revoked", user.ID, ipAddress, userAgent, map[string]any{
			"reason": "rotation",
		})
	}
	token, _, err := h.sessions.CreateSession(r.Context(), user.ID, ipAddress, userAgent)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		return
	}

	h.cookies.SetSessionCookie(w, token)
	h.auditLogger.Log(r.Context(), "register_success", user.ID, ipAddress, userAgent, nil)
	if user.Provider == "credentials" && !user.EmailVerified {
		h.sendVerificationEmail(r.Context(), user, ipAddress, userAgent)
	}
	writeJSON(w, http.StatusOK, AuthStatusResponse{Status: "ok"})
}

// HandleLogin logs in a user with email/password
// @Summary      Login with credentials
// @Description  Verifies credentials, creates a session, and sets a cookie
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request body LoginRequest true "Login request"
// @Success      200  {object}  AuthStatusResponse
// @Failure      400  {object}  map[string]string
// @Failure      401  {object}  map[string]string
// @Failure      500  {object}  map[string]string
// @Router       /auth/login [post]
func (h *AuthHandler) HandleLogin(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
		return
	}

	email, err := domain.NormalizeEmail(req.Email)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid email"})
		return
	}

	if !h.allowRequest(r.Context(), "login:"+email, r, h.rateLimits.Login) {
		writeJSON(w, http.StatusTooManyRequests, map[string]string{"error": "too many requests"})
		return
	}

	if len(req.Password) > 1000 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid password"})
		return
	}

	user, err := h.queries.GetUserByEmail(r.Context(), email)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			domain.FakePasswordHash(req.Password)
			h.auditLogger.Log(r.Context(), "login_failure", pgtype.UUID{}, h.ipFromRequest(r), r.UserAgent(), map[string]any{
				"email_hash": hashEmail(email),
				"reason":     "not_found",
			})
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid email or password"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		return
	}

	now := time.Now()
	if user.LockedUntil.Valid && user.LockedUntil.Time.After(now) {
		h.auditLogger.Log(r.Context(), "login_failure", user.ID, h.ipFromRequest(r), r.UserAgent(), map[string]any{
			"email_hash": hashEmail(email),
			"reason":     "locked",
		})
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid email or password"})
		return
	}
	if user.LockedUntil.Valid && user.LockedUntil.Time.Before(now) {
		if err := h.queries.UnlockUser(r.Context(), user.ID); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
			return
		}
	}

	if user.Provider != "credentials" || !user.PasswordHash.Valid {
		domain.FakePasswordHash(req.Password)
		h.auditLogger.Log(r.Context(), "login_failure", user.ID, h.ipFromRequest(r), r.UserAgent(), map[string]any{
			"email_hash": hashEmail(email),
			"reason":     "invalid_provider",
		})
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid email or password"})
		return
	}

	valid, err := domain.VerifyPassword(req.Password, user.PasswordHash.String)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		return
	}
	if !valid {
		updated, err := h.queries.IncrementFailedLoginAttempts(r.Context(), user.ID)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
			return
		}
		if updated.FailedLoginAttempts >= 10 {
			lockUntil := now.Add(30 * time.Minute)
			if err := h.queries.LockUser(r.Context(), db.LockUserParams{
				ID:          user.ID,
				LockedUntil: pgtype.Timestamptz{Time: lockUntil, Valid: true},
			}); err != nil {
				writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
				return
			}
			h.auditLogger.Log(r.Context(), "account_lockout", user.ID, h.ipFromRequest(r), r.UserAgent(), map[string]any{
				"email_hash": hashEmail(email),
			})
			h.sendLockoutEmail(r.Context(), user, lockUntil, h.ipFromRequest(r), r.UserAgent())
		}
		h.auditLogger.Log(r.Context(), "login_failure", user.ID, h.ipFromRequest(r), r.UserAgent(), map[string]any{
			"email_hash": hashEmail(email),
			"reason":     "invalid_password",
		})
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid email or password"})
		return
	}

	if err := h.queries.ResetFailedLoginAttempts(r.Context(), user.ID); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		return
	}

	userAgent := r.UserAgent()
	ipAddress := h.ipFromRequest(r)
	token, _, err := h.sessions.CreateSession(r.Context(), user.ID, ipAddress, userAgent)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		return
	}

	h.cookies.SetSessionCookie(w, token)
	h.auditLogger.Log(r.Context(), "login_success", user.ID, ipAddress, userAgent, nil)
	writeJSON(w, http.StatusOK, AuthStatusResponse{Status: "ok"})
}

// HandleChangePassword changes the user's password
// @Summary      Change password
// @Description  Updates password for credentials users and rotates sessions
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request body ChangePasswordRequest true "Change password request"
// @Success      200  {object}  AuthStatusResponse
// @Failure      400  {object}  map[string]string
// @Failure      401  {object}  map[string]string
// @Failure      429  {object}  map[string]string
// @Failure      500  {object}  map[string]string
// @Router       /auth/password [post]
func (h *AuthHandler) HandleChangePassword(w http.ResponseWriter, r *http.Request) {
	user, ok := userFromContext(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	if !h.allowRequest(r.Context(), "password:"+user.ID, r, h.rateLimits.Password) {
		writeJSON(w, http.StatusTooManyRequests, map[string]string{"error": "too many requests"})
		return
	}

	var req ChangePasswordRequest
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
		return
	}

	if len(req.NewPassword) > 1000 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid password"})
		return
	}

	userID := uuidFromString(user.ID)
	if !userID.Valid {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	stored, err := h.queries.GetUserByID(r.Context(), userID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		return
	}

	if stored.Provider != "credentials" || !stored.PasswordHash.Valid {
		h.auditLogger.Log(r.Context(), "password_change_failure", stored.ID, h.ipFromRequest(r), r.UserAgent(), map[string]any{
			"reason": "invalid_provider",
		})
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid credentials"})
		return
	}

	valid, err := domain.VerifyPassword(req.CurrentPassword, stored.PasswordHash.String)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		return
	}
	if !valid {
		h.auditLogger.Log(r.Context(), "password_change_failure", stored.ID, h.ipFromRequest(r), r.UserAgent(), map[string]any{
			"reason": "invalid_current_password",
		})
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid credentials"})
		return
	}

	if err := domain.ValidatePassword(req.NewPassword); err != nil {
		h.auditLogger.Log(r.Context(), "password_change_failure", stored.ID, h.ipFromRequest(r), r.UserAgent(), map[string]any{
			"reason": "invalid_new_password",
		})
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	hash, err := domain.HashPassword(req.NewPassword)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		return
	}

	if err := h.queries.UpdateUserPassword(r.Context(), db.UpdateUserPasswordParams{
		ID:           stored.ID,
		PasswordHash: pgtype.Text{String: hash, Valid: true},
	}); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		return
	}

	if err := h.sessions.RevokeUserSessions(r.Context(), stored.ID); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		return
	}

	h.auditLogger.Log(r.Context(), "session_revoked", stored.ID, h.ipFromRequest(r), r.UserAgent(), map[string]any{
		"reason": "password_change",
		"scope":  "all",
	})

	userAgent := r.UserAgent()
	ipAddress := h.ipFromRequest(r)
	token, _, err := h.sessions.CreateSession(r.Context(), stored.ID, ipAddress, userAgent)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		return
	}

	h.cookies.SetSessionCookie(w, token)
	h.auditLogger.Log(r.Context(), "password_change", stored.ID, ipAddress, userAgent, nil)
	writeJSON(w, http.StatusOK, AuthStatusResponse{Status: "ok"})
}

// HandleVerifyEmail verifies a user's email with a token
// @Summary      Verify email
// @Description  Verifies a user's email using a token
// @Tags         auth
// @Produce      json
// @Param        token  query  string  true  "Verification token"
// @Success      200  {object}  AuthStatusResponse
// @Failure      400  {object}  map[string]string
// @Failure      500  {object}  map[string]string
// @Router       /auth/verify-email [get]
func (h *AuthHandler) HandleVerifyEmail(w http.ResponseWriter, r *http.Request) {
	token := strings.TrimSpace(r.URL.Query().Get("token"))
	if token == "" {
		h.writeVerificationResponse(w, r, http.StatusBadRequest, "Invalid verification link", "The verification token is missing or invalid.")
		return
	}

	user, err := h.queries.GetUserByEmailVerificationTokenHash(r.Context(), domain.HashToken(token))
	if err != nil {
		h.writeVerificationResponse(w, r, http.StatusBadRequest, "Invalid verification link", "The verification token is missing or invalid.")
		return
	}

	if user.EmailVerificationExpiresAt.Valid && user.EmailVerificationExpiresAt.Time.Before(time.Now()) {
		h.writeVerificationResponse(w, r, http.StatusBadRequest, "Verification link expired", "Your verification link has expired. Please request a new one.")
		return
	}

	if !user.EmailVerified {
		if _, err := h.queries.VerifyUserEmail(r.Context(), user.ID); err != nil {
			h.writeVerificationResponse(w, r, http.StatusInternalServerError, "Verification failed", "We could not verify your email right now. Please try again.")
			return
		}
		h.auditLogger.Log(r.Context(), "email_verified", user.ID, h.ipFromRequest(r), r.UserAgent(), nil)
	}

	h.writeVerificationResponse(w, r, http.StatusOK, "Email verified", "Your email has been verified successfully.")
}

// HandleResendVerification resends the verification email
// @Summary      Resend verification email
// @Description  Resends the verification email for the authenticated user
// @Tags         auth
// @Produce      json
// @Success      200  {object}  AuthStatusResponse
// @Failure      400  {object}  map[string]string
// @Failure      401  {object}  map[string]string
// @Failure      429  {object}  map[string]string
// @Failure      500  {object}  map[string]string
// @Router       /auth/verify-email/resend [post]
func (h *AuthHandler) HandleResendVerification(w http.ResponseWriter, r *http.Request) {
	user, ok := userFromContext(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	if !h.allowRequest(r.Context(), "verify-email-resend:"+user.ID, r, h.rateLimits.VerifyEmailResend) {
		writeJSON(w, http.StatusTooManyRequests, map[string]string{"error": "too many requests"})
		return
	}

	userID := uuidFromString(user.ID)
	if !userID.Valid {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	stored, err := h.queries.GetUserByID(r.Context(), userID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		return
	}

	if stored.Provider != "credentials" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid credentials"})
		return
	}

	if !stored.EmailVerified {
		h.sendVerificationEmail(r.Context(), stored, h.ipFromRequest(r), r.UserAgent())
	}

	writeJSON(w, http.StatusOK, AuthStatusResponse{Status: "ok"})
}

// HandleGoogleLogin redirects to Google OAuth
// @Summary      Login with Google
// @Description  Redirects to Google OAuth authorization URL
// @Tags         auth
// @Produce      json
// @Success      302
// @Failure      429  {object}  map[string]string
// @Failure      500  {object}  map[string]string
// @Router       /auth/google [get]
func (h *AuthHandler) HandleGoogleLogin(w http.ResponseWriter, r *http.Request) {
	if !h.allowRequest(r.Context(), "google", r, h.rateLimits.Google) {
		writeJSON(w, http.StatusTooManyRequests, map[string]string{"error": "too many requests"})
		return
	}

	if h.oauthConfig == nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "google oauth not configured"})
		return
	}

	state, err := generateRandomToken(32)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		return
	}

	verifier, err := generateRandomToken(64)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		return
	}

	challenge := codeChallenge(verifier)

	setOAuthCookie(w, h.cookies, oauthStateCookieName, state)
	setOAuthCookie(w, h.cookies, oauthVerifierCookieName, verifier)

	authURL := h.oauthConfig.AuthCodeURL(
		state,
		oauth2.AccessTypeOnline,
		oauth2.SetAuthURLParam("code_challenge", challenge),
		oauth2.SetAuthURLParam("code_challenge_method", "S256"),
	)

	if wantsJSON(r) {
		writeJSON(w, http.StatusOK, map[string]string{"url": authURL})
		return
	}

	http.Redirect(w, r, authURL, http.StatusFound)
}

// HandleGoogleCallback handles Google OAuth callback
// @Summary      Google OAuth callback
// @Description  Handles Google OAuth callback and creates a session
// @Tags         auth
// @Produce      json
// @Success      302
// @Failure      400  {object}  map[string]string
// @Failure      500  {object}  map[string]string
// @Router       /auth/google/callback [get]
func (h *AuthHandler) HandleGoogleCallback(w http.ResponseWriter, r *http.Request) {
	if h.oauthConfig == nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "google oauth not configured"})
		return
	}

	state := r.URL.Query().Get("state")
	code := r.URL.Query().Get("code")
	if state == "" || code == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
		return
	}

	stateCookie, err := r.Cookie(oauthStateCookieName)
	if err != nil || stateCookie.Value == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid state"})
		return
	}

	verifierCookie, err := r.Cookie(oauthVerifierCookieName)
	if err != nil || verifierCookie.Value == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid state"})
		return
	}

	clearOAuthCookie(w, h.cookies, oauthStateCookieName)
	clearOAuthCookie(w, h.cookies, oauthVerifierCookieName)

	if subtle.ConstantTimeCompare([]byte(state), []byte(stateCookie.Value)) != 1 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid state"})
		return
	}

	token, err := h.oauthConfig.Exchange(r.Context(), code, oauth2.SetAuthURLParam("code_verifier", verifierCookie.Value))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid oauth code"})
		return
	}

	client := h.oauthConfig.Client(r.Context(), token)
	resp, err := client.Get("https://openidconnect.googleapis.com/v1/userinfo")
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid oauth response"})
		return
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		return
	}

	var info googleUserInfo
	if err := json.Unmarshal(body, &info); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		return
	}

	if info.Sub == "" || info.Email == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid oauth response"})
		return
	}

	email, err := domain.NormalizeEmail(info.Email)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid oauth response"})
		return
	}

	if existing, err := h.queries.GetUserByEmail(r.Context(), email); err == nil {
		if existing.Provider != "google" || !existing.GoogleID.Valid || existing.GoogleID.String != info.Sub {
			h.auditLogger.Log(r.Context(), "oauth_login_failure", pgtype.UUID{}, h.ipFromRequest(r), r.UserAgent(), map[string]any{
				"email_hash": hashEmail(email),
				"reason":     "email_conflict",
			})
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "unable to authenticate"})
			return
		}
	} else if !errors.Is(err, pgx.ErrNoRows) {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		return
	}

	name := strings.TrimSpace(info.Name)
	if name == "" {
		name = email
	}

	user, err := h.queries.UpsertUserByGoogleID(r.Context(), db.UpsertUserByGoogleIDParams{
		Email:         email,
		EmailVerified: info.EmailVerified,
		Name:          name,
		Picture:       pgtype.Text{String: info.Picture, Valid: info.Picture != ""},
		GoogleID:      pgtype.Text{String: info.Sub, Valid: info.Sub != ""},
	})
	if err != nil {
		if isUniqueViolation(err) {
			h.auditLogger.Log(r.Context(), "oauth_login_failure", pgtype.UUID{}, h.ipFromRequest(r), r.UserAgent(), map[string]any{
				"email_hash": hashEmail(email),
				"reason":     "email_conflict",
			})
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "unable to authenticate"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		return
	}

	userAgent := r.UserAgent()
	ipAddress := h.ipFromRequest(r)
	if revoked := h.revokeExistingSession(r); revoked {
		h.auditLogger.Log(r.Context(), "session_revoked", user.ID, ipAddress, userAgent, map[string]any{
			"reason": "rotation",
		})
	}
	rawToken, _, err := h.sessions.CreateSession(r.Context(), user.ID, ipAddress, userAgent)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		return
	}

	h.cookies.SetSessionCookie(w, rawToken)
	h.auditLogger.Log(r.Context(), "oauth_login", user.ID, ipAddress, userAgent, map[string]any{
		"provider": "google",
	})
	redirectTarget := h.postLoginRedirectURL
	if redirectTarget == "" {
		redirectTarget = "/"
	}
	http.Redirect(w, r, redirectTarget, http.StatusFound)
}

func (h *AuthHandler) sendVerificationEmail(ctx context.Context, user db.User, ip *netip.Addr, userAgent string) {
	if h.mailer == nil {
		return
	}

	token, err := generateRandomToken(emailVerificationTokenSize)
	if err != nil {
		h.auditLogger.Log(ctx, "email_verification_token_failed", user.ID, ip, userAgent, map[string]any{
			"error": err.Error(),
		})
		return
	}

	expiresAt := pgtype.Timestamptz{Time: time.Now().Add(emailVerificationTTL), Valid: true}
	if err := h.queries.SetEmailVerificationToken(ctx, db.SetEmailVerificationTokenParams{
		ID:                         user.ID,
		EmailVerificationTokenHash: domain.HashToken(token),
		EmailVerificationExpiresAt: expiresAt,
	}); err != nil {
		h.auditLogger.Log(ctx, "email_verification_token_failed", user.ID, ip, userAgent, map[string]any{
			"error": err.Error(),
		})
		return
	}

	verificationURL := h.verificationURL(token)
	name := strings.TrimSpace(user.Name)
	if name == "" {
		name = user.Email
	}

	subject := "Verify your email"
	params := email.EmailParams{
		Greeting:   fmt.Sprintf("Hi %s,", name),
		BodyLines:  []string{"Please verify your email address to get started."},
		ButtonText: "Verify Email",
		ButtonURL:  verificationURL,
		FooterText: "If you did not create an account, you can safely ignore this email.",
	}
	textBody := email.RenderText(params)
	htmlBody := email.RenderHTML(params)

	if err := h.mailer.Send(ctx, user.Email, subject, textBody, htmlBody); err != nil {
		h.auditLogger.Log(ctx, "email_send_failed", user.ID, ip, userAgent, map[string]any{
			"type":  "verification",
			"error": err.Error(),
		})
		return
	}

	h.auditLogger.Log(ctx, "email_verification_sent", user.ID, ip, userAgent, nil)
}

func (h *AuthHandler) sendLockoutEmail(ctx context.Context, user db.User, lockedUntil time.Time, ip *netip.Addr, userAgent string) {
	if h.mailer == nil {
		return
	}

	ipValue := "unknown"
	if ip != nil {
		ipValue = ip.String()
	}

	name := strings.TrimSpace(user.Name)
	if name == "" {
		name = user.Email
	}

	until := lockedUntil.UTC().Format(time.RFC1123)
	subject := "Your account has been locked"
	params := email.EmailParams{
		Greeting: fmt.Sprintf("Hi %s,", name),
		BodyLines: []string{
			"We locked your account after too many failed login attempts.",
			fmt.Sprintf("Lockout ends: %s", until),
			fmt.Sprintf("IP address: %s", ipValue),
		},
		FooterText: "If this wasn't you, please reset your password immediately.",
	}
	textBody := email.RenderText(params)
	htmlBody := email.RenderHTML(params)

	if err := h.mailer.Send(ctx, user.Email, subject, textBody, htmlBody); err != nil {
		h.auditLogger.Log(ctx, "email_send_failed", user.ID, ip, userAgent, map[string]any{
			"type":  "lockout",
			"error": err.Error(),
		})
	}
}

func (h *AuthHandler) verificationURL(token string) string {
	if h.appBaseURL == "" {
		return "/api/auth/verify-email?token=" + url.QueryEscape(token)
	}
	return h.appBaseURL + "/api/auth/verify-email?token=" + url.QueryEscape(token)
}

func (h *AuthHandler) writeVerificationResponse(w http.ResponseWriter, r *http.Request, status int, title, message string) {
	if wantsJSON(r) {
		if status >= 400 {
			writeJSON(w, status, map[string]string{"error": message})
			return
		}
		writeJSON(w, status, AuthStatusResponse{Status: "ok"})
		return
	}

	link := h.appBaseURL
	if link == "" {
		link = "/"
	}

	w.Header().Set("Content-Type", "text/html; charset=UTF-8")
	w.WriteHeader(status)
	_, _ = fmt.Fprintf(
		w,
		"<!doctype html><html><head><meta charset=\"utf-8\"><title>%s</title></head><body><main style=\"font-family:Arial, sans-serif; max-width:640px; margin:48px auto; padding:0 24px;\"><h1>%s</h1><p>%s</p><p><a href=\"%s\">Continue</a></p></main></body></html>",
		html.EscapeString(title),
		html.EscapeString(title),
		html.EscapeString(message),
		html.EscapeString(link),
	)
}

func userFromContext(ctx context.Context) (domain.SessionUser, bool) {
	value := ctx.Value(contextKeyUser)
	user, ok := value.(domain.SessionUser)
	return user, ok
}

func sessionFromContext(ctx context.Context) (*domain.SessionInfo, bool) {
	value := ctx.Value(contextKeySession)
	session, ok := value.(*domain.SessionInfo)
	return session, ok
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func wantsJSON(r *http.Request) bool {
	accept := r.Header.Get("Accept")
	if strings.Contains(accept, "application/json") {
		return true
	}
	if r.Header.Get("Sec-Fetch-Mode") == "cors" {
		return true
	}
	if r.Header.Get("X-Requested-With") != "" {
		return true
	}
	return false
}

func (h *AuthHandler) allowRequest(ctx context.Context, key string, r *http.Request, rule config.RateLimitRule) bool {
	if !h.rateLimits.Enabled {
		return true
	}

	if rule.Limit <= 0 || rule.Window <= 0 {
		return true
	}

	if h.rateLimiter == nil {
		return true
	}

	ip := h.ipFromRequest(r)
	ipKey := "unknown"
	if ip != nil {
		ipKey = ip.String()
	}

	allowed, err := h.rateLimiter.Allow(ctx, key+":"+ipKey, rule.Limit, rule.Window)
	if err != nil {
		return false
	}
	return allowed
}

func (h *AuthHandler) revokeExistingSession(r *http.Request) bool {
	cookie, err := r.Cookie(h.cookies.name)
	if err != nil || cookie.Value == "" {
		return false
	}
	_ = h.sessions.RevokeByTokenHash(r.Context(), domain.HashToken(cookie.Value))
	return true
}

func generateRandomToken(size int) (string, error) {
	buf := make([]byte, size)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

func codeChallenge(verifier string) string {
	sum := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

func setOAuthCookie(w http.ResponseWriter, cookies CookieManager, name, value string) {
	http.SetCookie(w, &http.Cookie{
		Name:     name,
		Value:    value,
		Path:     "/api/auth/google/callback",
		HttpOnly: true,
		Secure:   cookies.secure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int(oauthCookieMaxAge.Seconds()),
	})
}

func clearOAuthCookie(w http.ResponseWriter, cookies CookieManager, name string) {
	http.SetCookie(w, &http.Cookie{
		Name:     name,
		Value:    "",
		Path:     "/api/auth/google/callback",
		HttpOnly: true,
		Secure:   cookies.secure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	})
}

func (h *AuthHandler) ipFromRequest(r *http.Request) *netip.Addr {
	if h.trustedProxyHeader != "" {
		if value := r.Header.Get(h.trustedProxyHeader); value != "" {
			raw := strings.TrimSpace(strings.SplitN(value, ",", 2)[0])
			if addr, err := netip.ParseAddr(raw); err == nil {
				return &addr
			}
		}
	}

	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		if addr, parseErr := netip.ParseAddr(r.RemoteAddr); parseErr == nil {
			return &addr
		}
		return nil
	}

	addr, err := netip.ParseAddr(host)
	if err != nil {
		return nil
	}
	return &addr
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == "23505"
	}
	return false
}
