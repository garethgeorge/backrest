package auth

import (
	"context"
	"net/http"

	"github.com/garethgeorge/backrest/internal/config"
	"go.uber.org/zap"
)

type contextKey string

func (k contextKey) String() string {
	return "auth context value " + string(k)
}

const UserContextKey contextKey = "user"
const APIKeyContextKey contextKey = "api_key"

// SessionCookieName is the httpOnly cookie holding the backrest session JWT.
// Set by the OIDC callback; read here as a fallback to the Authorization header.
const SessionCookieName = "backrest-session"

func RequireAuthentication(h http.Handler, auth *Authenticator) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cfg, err := auth.config.Get()
		if err != nil {
			zap.S().Errorf("auth middleware failed to get config: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		if config.AuthDisabled(cfg.GetAuth()) {
			h.ServeHTTP(w, r)
			return
		}

		// Pass OPTIONS through unauthenticated so CORS preflight succeeds.
		if r.Method == http.MethodOptions {
			h.ServeHTTP(w, r)
			return
		}

		username, password, usesBasicAuth := r.BasicAuth()
		if usesBasicAuth {
			user, err := auth.Login(username, password)
			if err == nil {
				ctx := context.WithValue(r.Context(), UserContextKey, user)
				h.ServeHTTP(w, r.WithContext(ctx))
				return
			}
		}

		// TODO: process the API Key

		token, err := ParseBearerToken(r.Header.Get("Authorization"))
		if err != nil {
			// Fall back to the session cookie set by the OIDC flow.
			if c, cerr := r.Cookie(SessionCookieName); cerr == nil && c.Value != "" {
				token = c.Value
			} else {
				http.Error(w, "Unauthorized (No Authorization Header)", http.StatusUnauthorized)
				return
			}
		}

		user, err := auth.VerifyJWT(token)
		if err != nil {
			zap.S().Warnf("auth middleware blocked bad JWT: %v", err)
			http.Error(w, "Unauthorized (Bad Token)", http.StatusUnauthorized)
			return
		}

		ctx := context.WithValue(r.Context(), UserContextKey, user)
		h.ServeHTTP(w, r.WithContext(ctx))
	})
}
