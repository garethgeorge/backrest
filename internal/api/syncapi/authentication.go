package syncapi

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strings"

	v1 "github.com/garethgeorge/backrest/gen/go/v1"
	"github.com/garethgeorge/backrest/internal/config"
	"github.com/garethgeorge/backrest/internal/cryptoutil"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
)

type peerContextKey struct{}
type peerPublicKeyContextKey struct{}

func PeerFromContext(ctx context.Context) *v1.Multihost_Peer {
	return ctx.Value(peerContextKey{}).(*v1.Multihost_Peer)
}

func PeerPublicKeyFromContext(ctx context.Context) *cryptoutil.PublicKey {
	return ctx.Value(peerPublicKeyContextKey{}).(*cryptoutil.PublicKey)
}

func ContextWithPeer(ctx context.Context, peer *v1.Multihost_Peer, publicKey *cryptoutil.PublicKey) context.Context {
	if peer == nil {
		return ctx
	}
	ctx = context.WithValue(ctx, peerContextKey{}, peer)
	ctx = context.WithValue(ctx, peerPublicKeyContextKey{}, publicKey)
	return ctx
}

// HTTP decorator for authentication middleware.
func AuthenticationMiddleware(configManager *config.ConfigManager, handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, "Unauthorized: missing authentication header", http.StatusUnauthorized)
			return
		}

		config, err := configManager.Get()
		if err != nil {
			zap.S().Errorf("failed to get authorized clients from config: %v", err)
			http.Error(w, "Internal error", http.StatusInternalServerError)
			return
		}
		authorizedClientPeers := config.GetMultihost().GetAuthorizedClients()

		peerKey, instanceID, err := verifyAuthenticationHeader(authHeader)
		if err != nil {
			http.Error(w, fmt.Sprintf("Unauthorized: %v", err), http.StatusUnauthorized)
			return
		}

		authorizedPeerIdx := slices.IndexFunc(authorizedClientPeers, func(peer *v1.Multihost_Peer) bool {
			return peer.Keyid == peerKey.KeyID()
		})
		if authorizedPeerIdx == -1 {
			http.Error(w, fmt.Sprintf("Unauthorized: peer key %q is not listed in authorized clients", peerKey.KeyID()), http.StatusUnauthorized)
			return
		}
		authorizedPeer := authorizedClientPeers[authorizedPeerIdx]
		if authorizedPeer.InstanceId != instanceID {
			http.Error(w, fmt.Sprintf("Unauthorized: instance ID mismatch for peer key %q, expected %q, got %q", peerKey.KeyID(), authorizedPeer.InstanceId, instanceID), http.StatusUnauthorized)
			return
		}
		ctx := ContextWithPeer(r.Context(), authorizedPeer, peerKey)
		handler.ServeHTTP(w, r.WithContext(ctx))
	})
}

func createHandshakePacket(instanceID string, identity *cryptoutil.PrivateKey) (*v1.SyncStreamItem, error) {
	signedMessage, err := createSignedMessage([]byte(instanceID), identity)
	if err != nil {
		return nil, fmt.Errorf("signing instance ID: %w", err)
	}

	return &v1.SyncStreamItem{
		Action: &v1.SyncStreamItem_Handshake{
			Handshake: &v1.SyncStreamItem_SyncActionHandshake{
				ProtocolVersion: SyncProtocolVersion,
				InstanceId:      signedMessage,
				PublicKey:       identity.PublicKeyProto(),
			},
		},
	}, nil
}

// send authentication in a header rather than in the stream.
func createAuthenticationHeader(instanceID string, identity *cryptoutil.PrivateKey) (string, error) {
	signedMessage, err := createSignedMessage([]byte(instanceID), identity)
	if err != nil {
		return "", fmt.Errorf("signing instance ID for authentication header: %w", err)
	}

	handshakePacket := &v1.SyncStreamItem_SyncActionHandshake{
		ProtocolVersion: SyncProtocolVersion,
		InstanceId:      signedMessage,
		PublicKey:       identity.PublicKeyProto(),
	}

	encodedHandshake, err := proto.Marshal(handshakePacket)
	if err != nil {
		return "", fmt.Errorf("marshalling handshake packet: %w", err)
	}

	if len(encodedHandshake) > 512 {
		return "", fmt.Errorf("authorization header is too large, max size is 512 bytes, got %d bytes", len(encodedHandshake))
	}

	base64Handshake := base64.StdEncoding.EncodeToString(encodedHandshake)
	return fmt.Sprintf("Backrest-Sync-Auth %s", base64Handshake), nil
}

// verifyHandshakePacket verifies that
//   - the signature on the instance ID is valid against the public key provided in the handshake
//   - that the public key's ID is as attested in the handshake packet e.g. matches handshake.PublicKey.Keyid
//
// To authenticate, the caller must then check that the public key is trusted by checking the key ID against a local list.
func verifyHandshakePacket(item *v1.SyncStreamItem) (*cryptoutil.PublicKey, error) {
	handshake := item.GetHandshake()
	if handshake == nil {
		return nil, fmt.Errorf("empty or nil handshake, handshake packet must be sent first")
	}

	if handshake.ProtocolVersion != SyncProtocolVersion {
		return nil, fmt.Errorf("protocol version mismatch: expected %d, got %d", SyncProtocolVersion, handshake.ProtocolVersion)
	}

	if len(handshake.InstanceId.GetPayload()) == 0 || len(handshake.InstanceId.GetSignature()) == 0 {
		return nil, errors.New("instance ID payload and signature must not be empty")
	}

	if len(handshake.PublicKey.Keyid) == 0 {
		return nil, errors.New("public key ID must not be empty")
	}

	peerKey, err := cryptoutil.NewPublicKey(handshake.PublicKey)
	if err != nil {
		return nil, fmt.Errorf("loading peer public key: %w", err)
	}

	if err := verifySignedMessage(handshake.InstanceId, peerKey); err != nil {
		return nil, fmt.Errorf("verifying instance ID signature: %w", err)
	}

	return peerKey, nil
}

func verifyAuthenticationHeader(header string) (*cryptoutil.PublicKey, string, error) {
	if len(header) == 0 {
		return nil, "", errors.New("authentication header must not be empty")
	}

	// The header is expected to be in the format "Backrest-Sync-Auth <base64-encoded-handshake-packet>"
	if !strings.HasPrefix(header, "Backrest-Sync-Auth ") {
		return nil, "", fmt.Errorf("invalid authentication header format, expected 'Backrest-Sync-Auth <base64-encoded-handshake-packet>', got %s", header)
	}

	// Extract the base64-encoded handshake packet
	header = header[len("Backrest-Sync-Auth "):]
	decoded, err := base64.StdEncoding.DecodeString(header)
	if err != nil {
		return nil, "", fmt.Errorf("decoding authentication header: %w", err)
	}

	// Unmarshal the decoded header into a handshake packet
	var handshakePacket v1.SyncStreamItem_SyncActionHandshake
	if err := proto.Unmarshal(decoded, &handshakePacket); err != nil {
		return nil, "", fmt.Errorf("unmarshalling handshake packet: %w", err)
	}

	// Verify the handshake packet
	peerKey, err := verifyHandshakePacket(&v1.SyncStreamItem{
		Action: &v1.SyncStreamItem_Handshake{
			Handshake: &handshakePacket,
		},
	})
	if err != nil {
		return nil, "", fmt.Errorf("verifying handshake packet: %w", err)
	}

		return peerKey, string(handshakePacket.GetInstanceId().GetPayload()), nil
}
