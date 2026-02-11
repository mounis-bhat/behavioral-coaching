package config

import (
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	Port      string
	Env       string
	Database  DatabaseConfig
	Valkey    ValkeyConfig
	RateLimit RateLimitConfig
	Auth      AuthConfig
	Google    GoogleOAuthConfig
	Audit     AuditConfig
	Email     EmailConfig
	Storage   StorageConfig
}

type DatabaseConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	Database string
	SSLMode  string
}

func (d DatabaseConfig) ConnectionString() string {
	connURL := &url.URL{
		Scheme:   "postgres",
		User:     url.UserPassword(d.User, d.Password),
		Host:     fmt.Sprintf("%s:%s", d.Host, d.Port),
		Path:     d.Database,
		RawQuery: fmt.Sprintf("sslmode=%s", d.SSLMode),
	}
	return connURL.String()
}

type ValkeyConfig struct {
	Host     string
	Port     string
	Password string
}

type RateLimitRule struct {
	Limit  int
	Window time.Duration
}

type RateLimitConfig struct {
	Enabled           bool
	Register          RateLimitRule
	Login             RateLimitRule
	Password          RateLimitRule
	VerifyEmailResend RateLimitRule
	Google            RateLimitRule
	Logout            RateLimitRule
}

type AuthConfig struct {
	CookieName           string
	CookieSecure         bool
	CookieSameSite       http.SameSite
	SessionMaxAge        time.Duration
	IdleTimeout          time.Duration
	PostLoginRedirectURL string
	TrustedProxyHeader   string
}

type GoogleOAuthConfig struct {
	ClientID     string
	ClientSecret string
	RedirectURI  string
}

type AuditConfig struct {
	CleanupCron   string
	RetentionDays int
}

type EmailConfig struct {
	AppBaseURL       string
	ContactEmail     string
	GmailAppPassword string
}

type StorageConfig struct {
	Endpoint           string
	Region             string
	Bucket             string
	AccessKeyID        string
	SecretAccessKey    string
	ForcePathStyle     bool
	PresignUploadTTL   time.Duration
	PresignDownloadTTL time.Duration
	AvatarMaxBytes     int64
}

func (v ValkeyConfig) Addr() string {
	return fmt.Sprintf("%s:%s", v.Host, v.Port)
}

func Load() *Config {
	bootEnv := os.Getenv("ENV")
	if bootEnv == "" {
		bootEnv = "development"
	}

	if bootEnv == "production" {
		_ = godotenv.Load(".env.production", ".env")
	} else {
		_ = godotenv.Load(".env.development", ".env")
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "3400"
	}

	env := os.Getenv("ENV")
	if env == "" {
		env = "development"
	}

	appBaseURL := strings.TrimRight(os.Getenv("APP_BASE_URL"), "/")
	if appBaseURL == "" {
		appBaseURL = fmt.Sprintf("http://localhost:%s", port)
	}

	authConfig := AuthConfig{
		CookieName:           "session",
		CookieSecure:         false,
		CookieSameSite:       http.SameSiteLaxMode,
		SessionMaxAge:        7 * 24 * time.Hour,
		IdleTimeout:          30 * time.Minute,
		PostLoginRedirectURL: os.Getenv("AUTH_POST_LOGIN_REDIRECT_URL"),
		TrustedProxyHeader:   os.Getenv("TRUSTED_PROXY_HEADER"),
	}

	rateLimitEnabled := true
	if value, ok := getEnvBool("RATE_LIMIT_ENABLED"); ok {
		rateLimitEnabled = value
	}

	rateLimitConfig := RateLimitConfig{
		Enabled: rateLimitEnabled,
		Register: RateLimitRule{
			Limit:  getEnvIntOrDefault("RATE_LIMIT_REGISTER_LIMIT", 3),
			Window: time.Duration(getEnvIntOrDefault("RATE_LIMIT_REGISTER_WINDOW_SECONDS", 3600)) * time.Second,
		},
		Login: RateLimitRule{
			Limit:  getEnvIntOrDefault("RATE_LIMIT_LOGIN_LIMIT", 5),
			Window: time.Duration(getEnvIntOrDefault("RATE_LIMIT_LOGIN_WINDOW_SECONDS", 900)) * time.Second,
		},
		Password: RateLimitRule{
			Limit:  getEnvIntOrDefault("RATE_LIMIT_PASSWORD_LIMIT", 5),
			Window: time.Duration(getEnvIntOrDefault("RATE_LIMIT_PASSWORD_WINDOW_SECONDS", 900)) * time.Second,
		},
		VerifyEmailResend: RateLimitRule{
			Limit:  getEnvIntOrDefault("RATE_LIMIT_VERIFY_EMAIL_LIMIT", 3),
			Window: time.Duration(getEnvIntOrDefault("RATE_LIMIT_VERIFY_EMAIL_WINDOW_SECONDS", 3600)) * time.Second,
		},
		Google: RateLimitRule{
			Limit:  getEnvIntOrDefault("RATE_LIMIT_GOOGLE_LIMIT", 10),
			Window: time.Duration(getEnvIntOrDefault("RATE_LIMIT_GOOGLE_WINDOW_SECONDS", 900)) * time.Second,
		},
		Logout: RateLimitRule{
			Limit:  getEnvIntOrDefault("RATE_LIMIT_LOGOUT_LIMIT", 10),
			Window: time.Duration(getEnvIntOrDefault("RATE_LIMIT_LOGOUT_WINDOW_SECONDS", 60)) * time.Second,
		},
	}

	if env == "production" {
		authConfig.CookieName = "__Host-session"
		authConfig.CookieSecure = true
		authConfig.CookieSameSite = http.SameSiteStrictMode
	}

	if value, ok := getEnvBool("AUTH_COOKIE_SECURE"); ok {
		authConfig.CookieSecure = value
		if !value && authConfig.CookieName == "__Host-session" {
			authConfig.CookieName = "session"
		}
	}

	return &Config{
		Port: port,
		Env:  env,
		Database: DatabaseConfig{
			Host:     getEnvOrDefault("POSTGRES_HOST", "localhost"),
			Port:     getEnvOrDefault("POSTGRES_PORT", "5432"),
			User:     getEnvOrDefault("POSTGRES_USER", "app"),
			Password: os.Getenv("POSTGRES_PASSWORD"),
			Database: getEnvOrDefault("POSTGRES_DB", "app"),
			SSLMode:  getEnvOrDefault("POSTGRES_SSLMODE", "disable"),
		},
		Valkey: ValkeyConfig{
			Host:     getEnvOrDefault("VALKEY_HOST", "localhost"),
			Port:     getEnvOrDefault("VALKEY_PORT", "6379"),
			Password: os.Getenv("VALKEY_PASSWORD"),
		},
		RateLimit: rateLimitConfig,
		Auth:      authConfig,
		Google: GoogleOAuthConfig{
			ClientID:     os.Getenv("GOOGLE_CLIENT_ID"),
			ClientSecret: os.Getenv("GOOGLE_CLIENT_SECRET"),
			RedirectURI:  os.Getenv("GOOGLE_REDIRECT_URI"),
		},
		Audit: AuditConfig{
			CleanupCron:   getEnvOrDefault("AUDIT_CLEANUP_CRON", "0 3 * * *"),
			RetentionDays: getEnvIntOrDefault("AUDIT_RETENTION_DAYS", 90),
		},
		Email: EmailConfig{
			AppBaseURL:       appBaseURL,
			ContactEmail:     os.Getenv("CONTACT_EMAIL"),
			GmailAppPassword: os.Getenv("GMAIL_APP_PASSWORD"),
		},
		Storage: StorageConfig{
			Endpoint:           strings.TrimRight(os.Getenv("S3_ENDPOINT"), "/"),
			Region:             getEnvOrDefault("S3_REGION", "us-east-1"),
			Bucket:             os.Getenv("S3_BUCKET"),
			AccessKeyID:        os.Getenv("S3_ACCESS_KEY_ID"),
			SecretAccessKey:    os.Getenv("S3_SECRET_ACCESS_KEY"),
			ForcePathStyle:     getEnvBoolOrDefault("S3_FORCE_PATH_STYLE", true),
			PresignUploadTTL:   time.Duration(getEnvIntOrDefault("S3_PRESIGN_UPLOAD_TTL_SECONDS", 900)) * time.Second,
			PresignDownloadTTL: time.Duration(getEnvIntOrDefault("S3_PRESIGN_DOWNLOAD_TTL_SECONDS", 600)) * time.Second,
			AvatarMaxBytes:     int64(getEnvIntOrDefault("S3_AVATAR_MAX_BYTES", 5*1024*1024)),
		},
	}
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvBool(key string) (bool, bool) {
	value := os.Getenv(key)
	if value == "" {
		return false, false
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return false, false
	}
	return parsed, true
}

func getEnvBoolOrDefault(key string, defaultValue bool) bool {
	value, ok := getEnvBool(key)
	if !ok {
		return defaultValue
	}
	return value
}

func getEnvIntOrDefault(key string, defaultValue int) int {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return defaultValue
	}
	return parsed
}
