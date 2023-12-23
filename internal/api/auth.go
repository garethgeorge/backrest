package api

import (
	"encoding/base64"

	v1 "github.com/garethgeorge/backrest/gen/go/v1"
	"golang.org/x/crypto/bcrypt"
)

type User v1.User

func (u *User) CheckPassword(password string) bool {
	switch pw := u.Password.(type) {
	case *v1.User_PasswordBcrypt:
		pwHash, err := base64.StdEncoding.DecodeString(pw.PasswordBcrypt)
		if err != nil {
			return false
		}
		return bcrypt.CompareHashAndPassword(pwHash, []byte(password)) == nil
	default:
		return false
	}
}

type Authenticator struct {
	users map[string]*User
}

func NewAuthenticator(users []*v1.User) *Authenticator {
	auth := &Authenticator{
		users: make(map[string]*User),
	}
	for _, user := range users {
		auth.users[user.Name] = (*User)(user)
	}
	return auth
}

func (a *Authenticator) Authenticate(username, password string) (*User, bool) {
	user, ok := a.users[username]
	if !ok {
		return nil, false
	}
	if !user.CheckPassword(password) {
		return nil, false
	}
	return user, true
}
