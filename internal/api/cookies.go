package api

import (
	"net/http"
	"time"

	"github.com/mounis-bhat/starter/internal/config"
)

type CookieManager struct {
	name     string
	secure   bool
	sameSite http.SameSite
	maxAge   time.Duration
}

func NewCookieManager(cfg config.AuthConfig) CookieManager {
	return CookieManager{
		name:     cfg.CookieName,
		secure:   cfg.CookieSecure,
		sameSite: cfg.CookieSameSite,
		maxAge:   cfg.SessionMaxAge,
	}
}

func (c CookieManager) SetSessionCookie(w http.ResponseWriter, token string) {
	http.SetCookie(w, &http.Cookie{
		Name:     c.name,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   c.secure,
		SameSite: c.sameSite,
		MaxAge:   int(c.maxAge.Seconds()),
	})
}

func (c CookieManager) ClearSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     c.name,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   c.secure,
		SameSite: c.sameSite,
		MaxAge:   -1,
	})
}
