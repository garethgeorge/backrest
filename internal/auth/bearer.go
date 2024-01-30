package auth

import (
	"fmt"
	"strings"
)

func ParseBearerToken(token string) (string, error) {
	if !strings.HasPrefix(token, "Bearer ") {
		return "", fmt.Errorf("invalid token")
	}
	return token[7:], nil
}
