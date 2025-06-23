package config

import (
	"testing"

	v1 "github.com/garethgeorge/backrest/gen/go/v1"
	"github.com/google/go-cmp/cmp"
	"google.golang.org/protobuf/testing/protocmp"
)

func TestSanitizeForNetwork(t *testing.T) {
	tcs := []struct {
		name      string
		config    *v1.Config
		sanitized *v1.Config
	}{
		{
			name:      "empty config",
			config:    &v1.Config{},
			sanitized: &v1.Config{},
		},
		{
			name: "config with multihost identity",
			config: &v1.Config{
				Multihost: &v1.Multihost{
					Identity: &v1.PrivateKey{
						Keyid:       "test-key-id",
						Ed25519Priv: "test-private-key",
						Ed25519Pub:  "test-public-key",
					},
				},
			},
			sanitized: &v1.Config{
				Multihost: &v1.Multihost{
					Identity: &v1.PrivateKey{
						Keyid:       "test-key-id",
						Ed25519Priv: "",
						Ed25519Pub:  "",
					},
				},
			},
		},
		{
			name: "config with users and passwords",
			config: &v1.Config{
				Auth: &v1.Auth{
					Users: []*v1.User{
						{
							Name: "user1",
							Password: &v1.User_PasswordBcrypt{
								PasswordBcrypt: "hashedpassword123",
							},
						},
						{
							Name: "user2",
							Password: &v1.User_PasswordBcrypt{
								PasswordBcrypt: "hashedpassword456",
							},
						},
					},
				},
			},
			sanitized: &v1.Config{
				Auth: &v1.Auth{
					Users: []*v1.User{
						{
							Name: "user1",
							Password: &v1.User_PasswordBcrypt{
								PasswordBcrypt: "********",
							},
						},
						{
							Name: "user2",
							Password: &v1.User_PasswordBcrypt{
								PasswordBcrypt: "********",
							},
						},
					},
				},
			},
		},
		{
			name: "config with both multihost and users",
			config: &v1.Config{
				Multihost: &v1.Multihost{
					Identity: &v1.PrivateKey{
						Keyid:       "test-key-id",
						Ed25519Priv: "secret-key",
						Ed25519Pub:  "public-key",
					},
				},
				Auth: &v1.Auth{
					Users: []*v1.User{
						{
							Name: "admin",
							Password: &v1.User_PasswordBcrypt{
								PasswordBcrypt: "adminhash",
							},
						},
					},
				},
			},
			sanitized: &v1.Config{
				Multihost: &v1.Multihost{
					Identity: &v1.PrivateKey{
						Keyid:       "test-key-id",
						Ed25519Priv: "",
						Ed25519Pub:  "",
					},
				},
				Auth: &v1.Auth{
					Users: []*v1.User{
						{
							Name: "admin",
							Password: &v1.User_PasswordBcrypt{
								PasswordBcrypt: "********",
							},
						},
					},
				},
			},
		},
		{
			name: "config with nil identity",
			config: &v1.Config{
				Multihost: &v1.Multihost{
					Identity: nil,
				},
			},
			sanitized: &v1.Config{
				Multihost: &v1.Multihost{
					Identity: nil,
				},
			},
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			sanitized := SanitizeForNetwork(tc.config)
			if sanitized == nil {
				t.Fatal("Sanitized config is nil")
			}
			if diff := cmp.Diff(tc.sanitized, sanitized, protocmp.Transform()); diff != "" {
				t.Errorf("Sanitized config mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestRehydrateNetworkSanitizedConfig(t *testing.T) {
	tcs := []struct {
		name      string
		sanitized *v1.Config
		original  *v1.Config
		want      *v1.Config
	}{
		{
			name:      "empty config",
			sanitized: &v1.Config{},
			original:  &v1.Config{},
			want:      &v1.Config{},
		},
		{
			name: "config with multihost identity",
			sanitized: &v1.Config{
				Multihost: &v1.Multihost{
					Identity: &v1.PrivateKey{
						Keyid: "test-key-id",
					},
				},
			},
			original: &v1.Config{
				Multihost: &v1.Multihost{
					Identity: &v1.PrivateKey{
						Keyid:       "test-key-id",
						Ed25519Priv: "secret-key-data",
						Ed25519Pub:  "public-key-data",
					},
				},
			},
			want: &v1.Config{
				Multihost: &v1.Multihost{
					Identity: &v1.PrivateKey{
						Keyid:       "test-key-id",
						Ed25519Priv: "secret-key-data",
						Ed25519Pub:  "public-key-data",
					},
				},
			},
		},
		{
			name: "config with same set of users before and after",
			sanitized: &v1.Config{
				Auth: &v1.Auth{
					Users: []*v1.User{
						{
							Name: "user1",
							Password: &v1.User_PasswordBcrypt{
								PasswordBcrypt: "********",
							},
						},
						{
							Name: "user2",
							Password: &v1.User_PasswordBcrypt{
								PasswordBcrypt: "********",
							},
						},
					},
				},
			},
			original: &v1.Config{
				Auth: &v1.Auth{
					Users: []*v1.User{
						{
							Name: "user1",
							Password: &v1.User_PasswordBcrypt{
								PasswordBcrypt: "actual-hash-1",
							},
						},
						{
							Name: "user2",
							Password: &v1.User_PasswordBcrypt{
								PasswordBcrypt: "actual-hash-2",
							},
						},
					},
				},
			},
			want: &v1.Config{
				Auth: &v1.Auth{
					Users: []*v1.User{
						{
							Name: "user1",
							Password: &v1.User_PasswordBcrypt{
								PasswordBcrypt: "actual-hash-1",
							},
						},
						{
							Name: "user2",
							Password: &v1.User_PasswordBcrypt{
								PasswordBcrypt: "actual-hash-2",
							},
						},
					},
				},
			},
		},
		{
			name: "config with a user with a changed password should be preserved",
			sanitized: &v1.Config{
				Auth: &v1.Auth{
					Users: []*v1.User{
						{
							Name: "user1",
							Password: &v1.User_PasswordBcrypt{
								PasswordBcrypt: "********",
							},
						},
						{
							Name: "user2",
							Password: &v1.User_PasswordBcrypt{
								PasswordBcrypt: "new-password-not-masked",
							},
						},
					},
				},
			},
			original: &v1.Config{
				Auth: &v1.Auth{
					Users: []*v1.User{
						{
							Name: "user1",
							Password: &v1.User_PasswordBcrypt{
								PasswordBcrypt: "original-hash-1",
							},
						},
						{
							Name: "user2",
							Password: &v1.User_PasswordBcrypt{
								PasswordBcrypt: "original-hash-2",
							},
						},
					},
				},
			},
			want: &v1.Config{
				Auth: &v1.Auth{
					Users: []*v1.User{
						{
							Name: "user1",
							Password: &v1.User_PasswordBcrypt{
								PasswordBcrypt: "original-hash-1",
							},
						},
						{
							Name: "user2",
							Password: &v1.User_PasswordBcrypt{
								PasswordBcrypt: "new-password-not-masked",
							},
						},
					},
				},
			},
		},
		{
			name: "config with one user added and another removed",
			sanitized: &v1.Config{
				Auth: &v1.Auth{
					Users: []*v1.User{
						{
							Name: "user1",
							Password: &v1.User_PasswordBcrypt{
								PasswordBcrypt: "********",
							},
						},
						{
							Name: "user3",
							Password: &v1.User_PasswordBcrypt{
								PasswordBcrypt: "new-user-password",
							},
						},
					},
				},
			},
			original: &v1.Config{
				Auth: &v1.Auth{
					Users: []*v1.User{
						{
							Name: "user1",
							Password: &v1.User_PasswordBcrypt{
								PasswordBcrypt: "original-hash-1",
							},
						},
						{
							Name: "user2",
							Password: &v1.User_PasswordBcrypt{
								PasswordBcrypt: "original-hash-2",
							},
						},
					},
				},
			},
			want: &v1.Config{
				Auth: &v1.Auth{
					Users: []*v1.User{
						{
							Name: "user1",
							Password: &v1.User_PasswordBcrypt{
								PasswordBcrypt: "original-hash-1",
							},
						},
						{
							Name: "user3",
							Password: &v1.User_PasswordBcrypt{
								PasswordBcrypt: "new-user-password",
							},
						},
					},
				},
			},
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			rehydrated := RehydrateNetworkSanitizedConfig(tc.sanitized, tc.original)
			if rehydrated == nil {
				t.Fatal("Rehydrated config is nil")
			}
			if diff := cmp.Diff(tc.want, rehydrated, protocmp.Transform()); diff != "" {
				t.Errorf("Rehydrated config mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
