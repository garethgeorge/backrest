package auth

import (
	"encoding/base64"
	"errors"
	"fmt"
	"time"

	v1 "github.com/garethgeorge/backrest/gen/go/v1"
	"github.com/garethgeorge/backrest/internal/config"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

var defaultUsers = []*v1.User{
	{
		Name:     "default",
		Password: &v1.User_PasswordBcrypt{PasswordBcrypt: "JDJhJDEwJDNCdzJoNFlhaWFZQy9TSDN3ZGxSRHVPZHdzV2lsNmtBSHdFSmtIWHk1dS8wYjZuUWJrMGFx"}, // default password is "password"
	},
}

type Authenticator struct {
	config config.ConfigStore
	key    []byte
}

func NewAuthenticator(key []byte, configProvider config.ConfigStore) *Authenticator {
	return &Authenticator{
		config: configProvider,
		key:    key,
	}
}

var ErrUserNotFound = errors.New("user not found")
var ErrInvalidPassword = errors.New("invalid password")

func (a *Authenticator) users() []*v1.User {
	config, err := a.config.Get()
	if err != nil {
		return nil
	}
	if len(config.Auth.GetUsers()) != 0 {
		return config.Auth.GetUsers()
	}
	return defaultUsers
}

func (a *Authenticator) Login(username, password string) (*v1.User, error) {
	for _, user := range a.users() {
		if user.Name != username {
			continue
		}

		if err := checkPassword(user, password); err != nil {
			return nil, err
		}

		return user, nil
	}

	return nil, ErrUserNotFound
}

func (a *Authenticator) VerifyJWT(token string) (*v1.User, error) {
	t, err := jwt.Parse(token, func(t *jwt.Token) (interface{}, error) {
		return a.key, nil
	})

	if err != nil {
		return nil, fmt.Errorf("parse token: %w", err)
	}
	if !t.Valid {
		return nil, fmt.Errorf("invalid token")
	}

	subject, err := t.Claims.GetSubject()
	if err != nil {
		return nil, fmt.Errorf("get subject: %w", err)
	}

	for _, user := range a.users() {
		if user.Name == subject {
			return user, nil
		}
	}

	return nil, ErrUserNotFound
}

func (a *Authenticator) CreateJWT(user *v1.User) (string, error) {
	claims := &jwt.RegisteredClaims{
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(7 * 24 * time.Hour)),
		Subject:   user.Name,
	}

	t := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	s, err := t.SignedString(a.key)
	if err != nil {
		return "", fmt.Errorf("sign token: %w", err)
	}

	return s, nil
}

// checkPassword returns nil if the password is correct, or an error if it is not.
func checkPassword(user *v1.User, password string) error {
	switch pw := user.Password.(type) {
	case *v1.User_PasswordBcrypt:
		pwHash, err := base64.StdEncoding.DecodeString(pw.PasswordBcrypt)
		if err != nil {
			return fmt.Errorf("decode password: %w", err)
		}
		if err := bcrypt.CompareHashAndPassword(pwHash, []byte(password)); err != nil {
			return fmt.Errorf("%w: %w", ErrInvalidPassword, err)
		}
	default:
		return fmt.Errorf("unsupported password type: %T", pw)
	}
	return nil
}

func CreatePassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("generate password: %w", err)
	}
	return base64.StdEncoding.EncodeToString(hash), nil
}
