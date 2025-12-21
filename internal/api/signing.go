package api

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var (
	secret = make([]byte, 32)
)

func init() {
	n, err := rand.Read(secret)
	if n != 32 || err != nil {
		panic("failed to generate secret key; is /dev/urandom available?")
	}
}

type DownloadTokenPayload struct {
	OpID     int64  `json:"op_id"`
	Type     string `json:"type"` // "snapshot" or "restore"
	FilePath string `json:"file_path"`
}

func signDownloadToken(payload DownloadTokenPayload) (string, error) {
	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("failed to marshal payload: %w", err)
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.RegisteredClaims{
		Subject:   string(jsonPayload),
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
	})

	return token.SignedString(secret)
}

func verifyDownloadToken(tokenString string) (*DownloadTokenPayload, error) {
	token, err := jwt.ParseWithClaims(tokenString, &jwt.RegisteredClaims{}, func(token *jwt.Token) (interface{}, error) {
		return secret, nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	claims, ok := token.Claims.(*jwt.RegisteredClaims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token claims")
	}

	var payload DownloadTokenPayload
	if err := json.Unmarshal([]byte(claims.Subject), &payload); err != nil {
		return nil, fmt.Errorf("failed to unmarshal payload from subject: %w", err)
	}

	return &payload, nil
}
