package auth

import (
	"context"
	"net/http"

	"go.uber.org/zap"
)

type contextKey string

func (k contextKey) String() string {
	return "auth context value " + string(k)
}

const UserContextKey contextKey = "user"

func RequireAuthentication(h http.Handler, auth *Authenticator) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		username, password, usesBasicAuth := r.BasicAuth()
		if usesBasicAuth {
			user, err := auth.Login(username, password)
			if err == nil {
				ctx := context.WithValue(r.Context(), UserContextKey, user)
				h.ServeHTTP(w, r.WithContext(ctx))
				return
			}
		}

		token, err := ParseBearerToken(r.Header.Get("Authorization"))
		if err != nil {
			http.Error(w, "Unauthorized (No Authorization Header)", http.StatusUnauthorized)
			return
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
