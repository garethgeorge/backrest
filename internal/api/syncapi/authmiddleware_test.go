package syncapi

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	v1 "github.com/garethgeorge/backrest/gen/go/v1"
	"github.com/garethgeorge/backrest/gen/go/v1sync"
	"github.com/garethgeorge/backrest/internal/config"
	"github.com/garethgeorge/backrest/internal/config/migrations"
	"github.com/garethgeorge/backrest/internal/cryptoutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
)

func TestAuthMiddleware(t *testing.T) {
	serverPrivKey, err := cryptoutil.GeneratePrivateKey()
	require.NoError(t, err)

	clientPrivKey, err := cryptoutil.GeneratePrivateKey()
	require.NoError(t, err)

	// Create a mock config manager
	cfgManager := &config.ConfigManager{
		Store: &config.MemoryStore{
			Config: &v1.Config{
				Version:  migrations.CurrentVersion,
				Instance: "test-instance",
				Multihost: &v1.Multihost{
					Identity: serverPrivKey,
					AuthorizedClients: []*v1.Multihost_Peer{
						{
							InstanceId: "client-instance",
							Keyid:      clientPrivKey.Keyid,
						},
					},
				},
			},
		},
	}

	// Create a mock handler
	mockHandler := http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		peer := PeerFromContext(r.Context())
		require.NotNil(t, peer)
		assert.Equal(t, "client-instance", peer.InstanceId)
		rw.WriteHeader(http.StatusOK)
	})

	// Create the auth handler
	authHandler := newAuthHandler(cfgManager, mockHandler)

	// Create a test server
	server := httptest.NewServer(authHandler)
	defer server.Close()

	t.Run("valid auth header", func(t *testing.T) {
		// Create a request with a valid auth header
		req, err := http.NewRequest("GET", server.URL, nil)
		require.NoError(t, err)

		// Create a valid auth header
		clientCfg := &v1.Config{
			Instance: "client-instance",
			Multihost: &v1.Multihost{
				Identity: clientPrivKey,
			},
		}
		authHeader, err := createAuthHeader(clientCfg)
		require.NoError(t, err)
		req.Header.Set(authTokenHeader, authHeader)

		// Make the request
		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		// Check the response
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("missing auth header", func(t *testing.T) {
		req, err := http.NewRequest("GET", server.URL, nil)
		require.NoError(t, err)

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})

	t.Run("invalid auth header", func(t *testing.T) {
		req, err := http.NewRequest("GET", server.URL, nil)
		require.NoError(t, err)
		req.Header.Set(authTokenHeader, "invalid")

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})

	t.Run("unauthorized peer", func(t *testing.T) {
		unauthorizedPrivKey, err := cryptoutil.GeneratePrivateKey()
		require.NoError(t, err)

		req, err := http.NewRequest("GET", server.URL, nil)
		require.NoError(t, err)

		clientCfg := &v1.Config{
			Instance: "unauthorized-instance",
			Multihost: &v1.Multihost{
				Identity: unauthorizedPrivKey,
			},
		}
		authHeader, err := createAuthHeader(clientCfg)
		require.NoError(t, err)
		req.Header.Set(authTokenHeader, authHeader)

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})

	t.Run("instance id mismatch", func(t *testing.T) {
		req, err := http.NewRequest("GET", server.URL, nil)
		require.NoError(t, err)

		clientCfg := &v1.Config{
			Instance: "wrong-instance",
			Multihost: &v1.Multihost{
				Identity: clientPrivKey,
			},
		}
		authHeader, err := createAuthHeader(clientCfg)
		require.NoError(t, err)
		req.Header.Set(authTokenHeader, authHeader)

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})

	t.Run("signature too old", func(t *testing.T) {
		req, err := http.NewRequest("GET", server.URL, nil)
		require.NoError(t, err)

		clientCfg := &v1.Config{
			Instance: "client-instance",
			Multihost: &v1.Multihost{
				Identity: clientPrivKey,
			},
		}

		privKey, err := cryptoutil.NewPrivateKey(clientPrivKey)
		require.NoError(t, err)

		// create a signed message with an old timestamp
		signedMessage, err := createSignedMessage([]byte(clientCfg.Instance), privKey)
		require.NoError(t, err)
		signedMessage.TimestampMillis = time.Now().Add(-2 * maxSignatureAge).UnixMilli()

		// create the auth token
		authToken, err := createAuthToken(signedMessage, privKey)
		require.NoError(t, err)

		req.Header.Set(authTokenHeader, authToken)

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})
}

func createAuthToken(signedMessage *v1.SignedMessage, privKey *cryptoutil.PrivateKey) (string, error) {
	authToken := &v1sync.AuthorizationToken{
		InstanceId: signedMessage,
		PublicKey:  privKey.PublicKeyProto(),
	}

	tokenBytes, err := proto.Marshal(authToken)
	if err != nil {
		return "", fmt.Errorf("marshal auth token: %w", err)
	}

	return base64.StdEncoding.EncodeToString(tokenBytes), nil
}
