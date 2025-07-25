package syncapi

import (
	"context"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"time"

	"connectrpc.com/connect"
	v1 "github.com/garethgeorge/backrest/gen/go/v1"
	"github.com/garethgeorge/backrest/gen/go/v1sync"
	"github.com/garethgeorge/backrest/internal/config"
	"github.com/garethgeorge/backrest/internal/cryptoutil"
	"google.golang.org/protobuf/proto"
)

var authTokenHeader = "Authorization"
var maxSignatureAge = 5 * time.Minute // Maximum age of a signature before it is considered invalid

type peerContextKey string

const PeerContextKey peerContextKey = "peer"

func ContextWithPeer(ctx context.Context, peer *v1.Multihost_Peer) context.Context {
	return context.WithValue(ctx, PeerContextKey, peer)
}

func PeerFromContext(ctx context.Context) *v1.Multihost_Peer {
	peer, ok := ctx.Value(PeerContextKey).(*v1.Multihost_Peer)
	if !ok {
		return nil
	}
	return peer
}

func newAuthHandler(config *config.ConfigManager, next http.Handler) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		config, err := config.Get()
		if err != nil {
			http.Error(rw, "internal error", http.StatusInternalServerError)
			return
		}

		authHeaderValue, err := createAuthHeader(config)
		if err != nil {
			http.Error(rw, fmt.Sprintf("internal error: %v", err), http.StatusInternalServerError)
			return
		}
		rw.Header().Set(authTokenHeader, authHeaderValue)

		peer, err := decodeAndVerifyAuthHeader(r, config.Instance, config.GetMultihost().GetAuthorizedClients())
		if err != nil {
			http.Error(rw, fmt.Sprintf("unauthorized: %v", err), http.StatusUnauthorized)
			return
		}

		next.ServeHTTP(rw, r.WithContext(context.WithValue(r.Context(), PeerContextKey, peer)))
	})
}

func createAuthHeader(config *v1.Config) (string, error) {
	if config == nil || config.GetMultihost().GetIdentity() == nil {
		return "", errors.New("config missing multihost.identity")
	}

	privKey, err := cryptoutil.NewPrivateKey(config.GetMultihost().GetIdentity())
	if err != nil {
		return "", fmt.Errorf("load private key: %w", err)
	}

	signedMessage, err := createSignedMessage([]byte(config.Instance), privKey)
	if err != nil {
		return "", fmt.Errorf("create signed message: %w", err)
	}

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

type authHeaderClient struct {
	configManager *config.ConfigManager
	delegate      connect.HTTPClient
	wantPeer      *v1.Multihost_Peer
}

func (c *authHeaderClient) Do(req *http.Request) (*http.Response, error) {
	// create the header
	cfg, err := c.configManager.Get()
	if err != nil {
		return nil, fmt.Errorf("get config: %w", err)
	}
	authHeaderValue, err := createAuthHeader(cfg)
	if err != nil {
		return nil, fmt.Errorf("create auth header: %w", err)
	}
	req.Header.Set(authTokenHeader, authHeaderValue)

	resp, err := c.delegate.Do(req)
	// verify the response header
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return resp, fmt.Errorf("HTTP request failed with status %d: %s", resp.StatusCode, resp.Status)
	}
	peer, err := decodeAndVerifyAuthHeader(req, cfg.Instance, cfg.GetMultihost().GetAuthorizedClients())
	if err != nil {
		return resp, fmt.Errorf("verify auth header: %w", err)
	}

	// Check the peer matches the expected one.
	if c.wantPeer == nil || c.wantPeer.GetInstanceId() != peer.GetInstanceId() {
		return resp, fmt.Errorf("peer instance ID mismatch: expected %s, got %s", c.wantPeer.GetInstanceId(), peer.GetInstanceId())
	}
	if c.wantPeer.GetKeyid() != peer.GetKeyid() {
		return resp, fmt.Errorf("peer key ID mismatch: expected %s, got %s", c.wantPeer.GetKeyid(), peer.GetKeyid())
	}
	return resp, nil
}

func newHTTPClientWithConfig(cfg *config.ConfigManager, delegate connect.HTTPClient) (connect.HTTPClient, error) {
	return &authHeaderClient{
		configManager: cfg,
		delegate:      delegate,
	}, nil
}

func decodeAndVerifyAuthHeader(r *http.Request, localInstanceID string, peers []*v1.Multihost_Peer) (*v1.Multihost_Peer, error) {
	authHeader := r.Header.Get(authTokenHeader)
	if len(authHeader) == 0 {
		return nil, errors.New("missing authorization header")
	}

	// Decode the auth token from the header
	tokenBytes, err := base64.StdEncoding.DecodeString(authHeader)
	if err != nil {
		return nil, errors.New("invalid authorization header format")
	}

	var token v1sync.AuthorizationToken
	if err := proto.Unmarshal(tokenBytes, &token); err != nil {
		return nil, fmt.Errorf("unmarshal authorization token: %w", err)
	}

	// Load the public key from the token
	publicKey, err := cryptoutil.NewPublicKey(token.GetPublicKey())
	if err != nil {
		return nil, fmt.Errorf("load public key: %w", err)
	}
	if publicKey.KeyID() != token.InstanceId.GetKeyid() {
		return nil, fmt.Errorf("instance ID must be signed with public key in token: expected %s, got %s", token.InstanceId.GetKeyid(), publicKey.KeyID())
	}

	// Verify the signed message
	if err := verifySignedMessage(token.GetInstanceId(), publicKey); err != nil {
		return nil, fmt.Errorf("verify signed message: %w", err)
	}

	// Now that we've validated that the peer was able to sign the message, we can look it up in the config
	peerIdx := slices.IndexFunc(peers, func(peer *v1.Multihost_Peer) bool {
		return peer.Keyid == publicKey.KeyID()
	})
	if peerIdx == -1 {
		return nil, fmt.Errorf("peer with key ID %s not found in authorized clients", publicKey.KeyID())
	}

	// Finally check that the instance ID in the token matches the one in the config
	peer := peers[peerIdx]
	tokenInstanceID := string(token.GetInstanceId().GetPayload())
	if peer.InstanceId != tokenInstanceID {
		return nil, fmt.Errorf("instance ID mismatch: expected %s, got %s", peer.InstanceId, tokenInstanceID)
	}

	return peer, nil
}

func createSignedMessage(payload []byte, identity *cryptoutil.PrivateKey) (*v1.SignedMessage, error) {
	if len(payload) == 0 {
		return nil, errors.New("payload must not be empty")
	}

	timestampMillis := time.Now().UnixMilli()

	payloadWithTimestamp := make([]byte, 0, len(payload)+8)
	binary.BigEndian.AppendUint64(payloadWithTimestamp, uint64(timestampMillis))
	payloadWithTimestamp = append(payloadWithTimestamp, payload...)

	signature, err := identity.Sign(payloadWithTimestamp)
	if err != nil {
		return nil, fmt.Errorf("signing payload: %w", err)
	}

	return &v1.SignedMessage{
		Payload:         payload,
		Signature:       signature,
		Keyid:           identity.KeyID(),
		TimestampMillis: timestampMillis,
	}, nil
}

func verifySignedMessage(msg *v1.SignedMessage, publicKey *cryptoutil.PublicKey) error {
	if msg == nil {
		return errors.New("signed message must not be nil")
	}
	if len(msg.GetPayload()) == 0 {
		return errors.New("signed message payload must not be empty")
	}
	if len(msg.GetSignature()) == 0 {
		return errors.New("signed message signature must not be empty")
	}
	if len(msg.GetKeyid()) == 0 {
		return errors.New("signed message key ID must not be empty")
	}

	if publicKey.KeyID() != msg.GetKeyid() {
		return fmt.Errorf("public key ID mismatch: expected %s, got %s", publicKey.KeyID(), msg.GetKeyid())
	}

	payloadWithTimestamp := make([]byte, 0, len(msg.GetPayload())+8)
	binary.BigEndian.AppendUint64(payloadWithTimestamp, uint64(msg.GetTimestampMillis()))
	payloadWithTimestamp = append(payloadWithTimestamp, msg.GetPayload()...)

	if err := publicKey.Verify(payloadWithTimestamp, msg.GetSignature()); err != nil {
		return fmt.Errorf("verifying signed message: %w", err)
	}

	if time.Since(time.UnixMilli(msg.GetTimestampMillis())) > maxSignatureAge {
		return fmt.Errorf("signature is too old, max age is %s. Is the clock out of sync?", maxSignatureAge)
	}

	return nil
}
