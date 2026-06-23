package auth

import (
	"testing"

	v1 "github.com/garethgeorge/backrest/gen/go/v1"
	"github.com/garethgeorge/backrest/internal/config"
)

func TestEmailAllowed(t *testing.T) {
	tests := []struct {
		name           string
		email          string
		allowedEmails  []string
		allowedDomains []string
		wantErr        bool
	}{
		{"no restrictions allows any", "anyone@example.com", nil, nil, false},
		{"email in allowed list", "a@example.com", []string{"a@example.com"}, nil, false},
		{"email not in allowed list", "b@example.com", []string{"a@example.com"}, nil, true},
		{"email match is case insensitive", "A@Example.com", []string{"a@example.com"}, nil, false},
		{"domain allowed", "user@corp.com", nil, []string{"corp.com"}, false},
		{"domain not allowed", "user@other.com", nil, []string{"corp.com"}, true},
		{"domain match is case insensitive", "user@CORP.com", nil, []string{"corp.com"}, false},
		{"email list wins over domain", "vip@gmail.com", []string{"vip@gmail.com"}, []string{"corp.com"}, false},
		{"neither list matches", "x@nope.com", []string{"a@example.com"}, []string{"corp.com"}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := emailAllowed(tt.email, tt.allowedEmails, tt.allowedDomains)
			if (err != nil) != tt.wantErr {
				t.Fatalf("emailAllowed(%q) error = %v, wantErr %v", tt.email, err, tt.wantErr)
			}
		})
	}
}

func TestVerifyJWT_OIDCSubject(t *testing.T) {
	store := &config.MemoryStore{
		Config: &v1.Config{
			Auth: &v1.Auth{
				AuthDriver: config.AuthDriverOIDC,
				Oidc: &v1.OidcConfig{
					IssuerUrl: "https://issuer.example.com",
					ClientId:  "client",
				},
			},
		},
	}
	a := NewAuthenticator([]byte("key"), store)

	const email = "alice@example.com"
	token, err := a.CreateJWTForSubject(email)
	if err != nil {
		t.Fatalf("CreateJWTForSubject: %v", err)
	}

	// OIDC sessions are not backed by config users; the signed subject is the identity.
	user, err := a.VerifyJWT(token)
	if err != nil {
		t.Fatalf("VerifyJWT: %v", err)
	}
	if user.GetName() != email {
		t.Fatalf("expected subject %q, got %q", email, user.GetName())
	}
}
