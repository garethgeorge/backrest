package auth

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"go.uber.org/zap"
	"golang.org/x/oauth2"
)

const (
	oidcLoginPath    = "/auth/oidc/login"
	oidcCallbackPath = "/auth/oidc/callback"
	oidcLogoutPath   = "/auth/oidc/logout"

	// Short-lived cookies carrying the per-login CSRF/replay protections. Scoped
	// to the oidc paths and cleared on callback.
	oidcStateCookie    = "backrest-oidc-state"
	oidcNonceCookie    = "backrest-oidc-nonce"
	oidcVerifierCookie = "backrest-oidc-verifier"
	oidcCookiePath     = "/auth/oidc"

	oidcTempCookieTTL = 10 * time.Minute
	sessionTTL        = 7 * 24 * time.Hour
)

// OIDCHandler serves the OIDC authorization-code redirect flow over plain HTTP
// (not connectrpc) because the browser navigates these routes directly.
type OIDCHandler struct {
	manager       *OIDCManager
	authenticator *Authenticator
}

func NewOIDCHandler(manager *OIDCManager, authenticator *Authenticator) *OIDCHandler {
	return &OIDCHandler{manager: manager, authenticator: authenticator}
}

// Register mounts the OIDC routes on the given mux. These routes are
// unauthenticated by design.
func (h *OIDCHandler) Register(mux *http.ServeMux) {
	mux.HandleFunc(oidcLoginPath, h.handleLogin)
	mux.HandleFunc(oidcCallbackPath, h.handleCallback)
	mux.HandleFunc(oidcLogoutPath, h.handleLogout)
}

func (h *OIDCHandler) handleLogin(w http.ResponseWriter, r *http.Request) {
	state, err := randToken()
	if err != nil {
		h.fail(w, r, fmt.Errorf("generate state: %w", err))
		return
	}
	nonce, err := randToken()
	if err != nil {
		h.fail(w, r, fmt.Errorf("generate nonce: %w", err))
		return
	}
	verifier := oauth2.GenerateVerifier()

	redirectURL := h.redirectURL(r)
	authURL, err := h.manager.AuthCodeURL(r.Context(), state, nonce, verifier, redirectURL)
	if err != nil {
		h.fail(w, r, fmt.Errorf("build authorization url: %w", err))
		return
	}

	secure := isSecureRequest(r)
	setTempCookie(w, oidcStateCookie, state, secure)
	setTempCookie(w, oidcNonceCookie, nonce, secure)
	setTempCookie(w, oidcVerifierCookie, verifier, secure)

	http.Redirect(w, r, authURL, http.StatusFound)
}

func (h *OIDCHandler) handleCallback(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	if errMsg := q.Get("error"); errMsg != "" {
		desc := q.Get("error_description")
		h.fail(w, r, fmt.Errorf("provider returned error %q: %s", errMsg, desc))
		return
	}

	stateCookie, err := r.Cookie(oidcStateCookie)
	if err != nil || stateCookie.Value == "" || stateCookie.Value != q.Get("state") {
		h.fail(w, r, errors.New("invalid or missing oidc state"))
		return
	}
	nonceCookie, err := r.Cookie(oidcNonceCookie)
	if err != nil || nonceCookie.Value == "" {
		h.fail(w, r, errors.New("missing oidc nonce"))
		return
	}
	verifierCookie, err := r.Cookie(oidcVerifierCookie)
	if err != nil || verifierCookie.Value == "" {
		h.fail(w, r, errors.New("missing oidc pkce verifier"))
		return
	}

	identity, err := h.manager.Exchange(r.Context(), q.Get("code"), nonceCookie.Value, verifierCookie.Value, h.redirectURL(r))
	if err != nil {
		h.fail(w, r, fmt.Errorf("oidc exchange: %w", err))
		return
	}

	token, err := h.authenticator.CreateJWTForSubject(identity.Email)
	if err != nil {
		h.fail(w, r, fmt.Errorf("mint session token: %w", err))
		return
	}

	secure := isSecureRequest(r)
	clearCookie(w, oidcStateCookie, secure)
	clearCookie(w, oidcNonceCookie, secure)
	clearCookie(w, oidcVerifierCookie, secure)

	http.SetCookie(w, &http.Cookie{
		Name:     SessionCookieName,
		Value:    token,
		Path:     "/",
		MaxAge:   int(sessionTTL.Seconds()),
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
	})

	zap.S().Infof("oidc login for %q", identity.Email)
	http.Redirect(w, r, "/", http.StatusFound)
}

func (h *OIDCHandler) handleLogout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     SessionCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   isSecureRequest(r),
		SameSite: http.SameSiteLaxMode,
	})
	http.Redirect(w, r, "/", http.StatusFound)
}

// redirectURL returns the configured redirect_url, or one derived from the
// incoming request when unset. Must be identical between login and callback.
func (h *OIDCHandler) redirectURL(r *http.Request) string {
	if cfg, err := h.manager.config.Get(); err == nil {
		if configured := strings.TrimSpace(cfg.GetAuth().GetOidc().GetRedirectUrl()); configured != "" {
			return configured
		}
	}
	scheme := "http"
	if isSecureRequest(r) {
		scheme = "https"
	}
	return fmt.Sprintf("%s://%s%s", scheme, r.Host, oidcCallbackPath)
}

func (h *OIDCHandler) fail(w http.ResponseWriter, r *http.Request, err error) {
	zap.S().Warnf("oidc auth failed: %v", err)
	// Clear any partial temp cookies so a retry starts clean.
	secure := isSecureRequest(r)
	clearCookie(w, oidcStateCookie, secure)
	clearCookie(w, oidcNonceCookie, secure)
	clearCookie(w, oidcVerifierCookie, secure)
	// Redirect to the UI with an error flag; the UI shows it instead of re-redirecting.
	http.Redirect(w, r, "/?authError="+url.QueryEscape("OIDC login failed"), http.StatusFound)
}

func isSecureRequest(r *http.Request) bool {
	if r.TLS != nil {
		return true
	}
	return strings.EqualFold(r.Header.Get("X-Forwarded-Proto"), "https")
}

func setTempCookie(w http.ResponseWriter, name, value string, secure bool) {
	http.SetCookie(w, &http.Cookie{
		Name:     name,
		Value:    value,
		Path:     oidcCookiePath,
		MaxAge:   int(oidcTempCookieTTL.Seconds()),
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
	})
}

func clearCookie(w http.ResponseWriter, name string, secure bool) {
	http.SetCookie(w, &http.Cookie{
		Name:     name,
		Value:    "",
		Path:     oidcCookiePath,
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
	})
}

func randToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}
