package auth

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/coreos/go-oidc/v3/oidc"
	v1 "github.com/garethgeorge/backrest/gen/go/v1"
	"github.com/garethgeorge/backrest/internal/config"
	"golang.org/x/oauth2"
	"google.golang.org/protobuf/proto"
)

// DefaultOIDCScopes are requested when the config specifies no scopes.
var DefaultOIDCScopes = []string{oidc.ScopeOpenID, "email", "profile"}

// Identity is the verified identity extracted from an OIDC ID token.
type Identity struct {
	Email         string
	EmailVerified bool
	Name          string
}

// oidcClient bundles a discovered provider with its verifier and oauth2 config.
// RedirectURL on the embedded oauth2.Config is left empty and filled per-request.
type oidcClient struct {
	verifier *oidc.IDTokenVerifier
	oauth2   oauth2.Config
	cfg      *v1.OidcConfig
}

// OIDCManager lazily builds and caches an OIDC client from the current config,
// rebuilding when the oidc settings change. It is safe for concurrent use.
type OIDCManager struct {
	config config.ConfigStore

	mu     sync.Mutex
	hash   string
	cached *oidcClient
}

func NewOIDCManager(cfg config.ConfigStore) *OIDCManager {
	return &OIDCManager{config: cfg}
}

// getClient returns a client for the current oidc config, performing provider
// discovery (a network call) only when the config has changed since last build.
func (m *OIDCManager) getClient(ctx context.Context) (*oidcClient, error) {
	cfg, err := m.config.Get()
	if err != nil {
		return nil, fmt.Errorf("get config: %w", err)
	}
	auth := cfg.GetAuth()
	if config.AuthDriverOf(auth) != config.AuthDriverOIDC {
		return nil, errors.New("oidc auth driver is not enabled")
	}
	oc := auth.GetOidc()
	if oc == nil {
		return nil, errors.New("oidc settings are not configured")
	}

	h := hashOidcConfig(oc)

	m.mu.Lock()
	defer m.mu.Unlock()
	if m.cached != nil && m.hash == h {
		return m.cached, nil
	}

	provider, err := oidc.NewProvider(ctx, strings.TrimSpace(oc.GetIssuerUrl()))
	if err != nil {
		return nil, fmt.Errorf("oidc discovery for %q: %w", oc.GetIssuerUrl(), err)
	}

	scopes := oc.GetScopes()
	if len(scopes) == 0 {
		scopes = DefaultOIDCScopes
	}

	client := &oidcClient{
		verifier: provider.Verifier(&oidc.Config{ClientID: oc.GetClientId()}),
		oauth2: oauth2.Config{
			ClientID:     oc.GetClientId(),
			ClientSecret: oc.GetClientSecret(),
			Endpoint:     provider.Endpoint(),
			Scopes:       scopes,
		},
		cfg: oc,
	}
	m.cached = client
	m.hash = h
	return client, nil
}

// AuthCodeURL returns the provider authorization URL for the given state,
// nonce, and PKCE code verifier. redirectURL must match the value used in Exchange.
func (m *OIDCManager) AuthCodeURL(ctx context.Context, state, nonce, codeVerifier, redirectURL string) (string, error) {
	c, err := m.getClient(ctx)
	if err != nil {
		return "", err
	}
	conf := c.oauth2 // copy so the per-request RedirectURL doesn't race
	conf.RedirectURL = redirectURL
	return conf.AuthCodeURL(state,
		oidc.Nonce(nonce),
		oauth2.S256ChallengeOption(codeVerifier),
	), nil
}

// Exchange exchanges an authorization code for tokens, verifies the ID token
// (signature, audience, and nonce), enforces the allowed email/domain lists,
// and returns the verified identity.
func (m *OIDCManager) Exchange(ctx context.Context, code, nonce, codeVerifier, redirectURL string) (*Identity, error) {
	c, err := m.getClient(ctx)
	if err != nil {
		return nil, err
	}
	conf := c.oauth2
	conf.RedirectURL = redirectURL

	tok, err := conf.Exchange(ctx, code, oauth2.VerifierOption(codeVerifier))
	if err != nil {
		return nil, fmt.Errorf("token exchange: %w", err)
	}

	rawIDToken, ok := tok.Extra("id_token").(string)
	if !ok || rawIDToken == "" {
		return nil, errors.New("no id_token in token response")
	}

	idToken, err := c.verifier.Verify(ctx, rawIDToken)
	if err != nil {
		return nil, fmt.Errorf("verify id token: %w", err)
	}
	if idToken.Nonce != nonce {
		return nil, errors.New("oidc nonce mismatch")
	}

	var claims struct {
		Email         string `json:"email"`
		EmailVerified bool   `json:"email_verified"`
		Name          string `json:"name"`
	}
	if err := idToken.Claims(&claims); err != nil {
		return nil, fmt.Errorf("parse id token claims: %w", err)
	}
	if claims.Email == "" {
		return nil, errors.New("id token has no email claim")
	}
	if err := emailAllowed(claims.Email, c.cfg.GetAllowedEmails(), c.cfg.GetAllowedDomains()); err != nil {
		return nil, err
	}

	return &Identity{
		Email:         claims.Email,
		EmailVerified: claims.EmailVerified,
		Name:          claims.Name,
	}, nil
}

// emailAllowed reports nil if the email passes the allow lists. When neither list
// is configured, any authenticated email is allowed.
func emailAllowed(email string, allowedEmails, allowedDomains []string) error {
	if len(allowedEmails) == 0 && len(allowedDomains) == 0 {
		return nil
	}

	lower := strings.ToLower(strings.TrimSpace(email))
	for _, e := range allowedEmails {
		if strings.ToLower(strings.TrimSpace(e)) == lower {
			return nil
		}
	}

	if at := strings.LastIndex(lower, "@"); at >= 0 {
		domain := lower[at+1:]
		for _, d := range allowedDomains {
			if strings.ToLower(strings.TrimSpace(d)) == domain {
				return nil
			}
		}
	}

	return fmt.Errorf("email %q is not permitted to log in", email)
}

func hashOidcConfig(oc *v1.OidcConfig) string {
	b, _ := proto.MarshalOptions{Deterministic: true}.Marshal(oc)
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}
