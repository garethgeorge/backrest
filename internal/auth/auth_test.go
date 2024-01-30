package auth

import (
	"errors"
	"testing"

	v1 "github.com/garethgeorge/backrest/gen/go/v1"
	"github.com/garethgeorge/backrest/internal/config"
)

func TestLogin(t *testing.T) {
	pass := makePass(t, "testPass")
	pass2 := makePass(t, "testPass2")

	config := &config.MemoryStore{
		Config: &v1.Config{
			Auth: &v1.Auth{
				Users: []*v1.User{
					{
						Name: "test",
						Password: &v1.User_PasswordBcrypt{
							PasswordBcrypt: pass,
						},
					},
					{
						Name: "anotheruser",
						Password: &v1.User_PasswordBcrypt{
							PasswordBcrypt: pass2,
						},
					},
				},
			},
		},
	}

	auth := NewAuthenticator("key", config)

	tests := []struct {
		name     string
		username string
		password string
		wantErr  error
	}{
		{"user 1 valid password", "test", "testPass", nil},
		{"user 2 valid password", "anotheruser", "testPass2", nil},
		{"user 1 wrong password", "test", "wrongPass", ErrInvalidPassword},
		{"invalid user", "nonexistent", "testPass", ErrUserNotFound},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			user, err := auth.Login(test.username, test.password)
			if !errors.Is(err, test.wantErr) {
				t.Fatalf("Expected error %v, got %v", test.wantErr, err)
			}
			if err == nil && user.Name != test.username {
				t.Fatalf("Expected user name to be '%s', got '%s'", test.username, user.Name)
			}
		})
	}
}

func makePass(t *testing.T, pass string) string {
	p, err := CreatePassword(pass)
	if err != nil {
		t.Fatalf("Error creating password: %v", err)
	}
	return p
}
