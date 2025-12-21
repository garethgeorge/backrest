package api

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDownloadToken(t *testing.T) {
	payload := DownloadTokenPayload{
		OpID:     12345,
		Type:     "snapshot",
		FilePath: "/path/to/file",
	}

	token, err := signDownloadToken(payload)
	assert.NoError(t, err)
	assert.NotEmpty(t, token)

	verified, err := verifyDownloadToken(token)
	assert.NoError(t, err)
	assert.Equal(t, payload.OpID, verified.OpID)
	assert.Equal(t, payload.Type, verified.Type)
	assert.Equal(t, payload.FilePath, verified.FilePath)
}

func TestVerifyInvalidToken(t *testing.T) {
	_, err := verifyDownloadToken("invalid.token.here")
	assert.Error(t, err)
}
