package syncapi

import (
	"encoding/binary"
	"testing"
	"time"

	v1 "github.com/garethgeorge/backrest/gen/go/v1"
	"github.com/garethgeorge/backrest/internal/cryptoutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateSignedMessage(t *testing.T) {
	protoKey, err := cryptoutil.GeneratePrivateKey()
	require.NoError(t, err)
	identity, err := cryptoutil.NewPrivateKey(protoKey)
	require.NoError(t, err)

	testCases := []struct {
		name        string
		payload     []byte
		identity    *cryptoutil.PrivateKey
		wantErr     bool
		expectedErr string
	}{
		{
			name:     "valid payload and identity",
			payload:  []byte("test payload"),
			identity: identity,
			wantErr:  false,
		},
		{
			name:        "empty payload",
			payload:     []byte{},
			identity:    identity,
			wantErr:     true,
			expectedErr: "payload must not be empty",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			signedMsg, err := createSignedMessage(tc.payload, tc.identity)

			if tc.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tc.expectedErr)
				assert.Nil(t, signedMsg)
			} else {
				assert.NoError(t, err)
				require.NotNil(t, signedMsg)
				assert.Equal(t, tc.payload, signedMsg.Payload)
				assert.Equal(t, tc.identity.KeyID(), signedMsg.Keyid)
				assert.NotEmpty(t, signedMsg.Signature)
				assert.WithinDuration(t, time.Now(), time.UnixMilli(signedMsg.TimestampMillis), 1*time.Second)
			}
		})
	}
}

func TestVerifySignedMessage(t *testing.T) {
	protoKey1, err := cryptoutil.GeneratePrivateKey()
	require.NoError(t, err)
	identity, err := cryptoutil.NewPrivateKey(protoKey1)
	require.NoError(t, err)
	publicKey := identity.PublicKey

	protoKey2, err := cryptoutil.GeneratePrivateKey()
	require.NoError(t, err)
	otherIdentity, err := cryptoutil.NewPrivateKey(protoKey2)
	require.NoError(t, err)
	otherPublicKey := otherIdentity.PublicKey

	validPayload := []byte("test payload")
	validMsg, err := createSignedMessage(validPayload, identity)
	require.NoError(t, err)

	// Create a message with an old timestamp
	oldTimestamp := time.Now().Add(-(maxSignatureAge + 1*time.Minute)).UnixMilli()
	payloadWithTimestamp := make([]byte, 0, len(validPayload)+8)
	binary.BigEndian.AppendUint64(payloadWithTimestamp, uint64(oldTimestamp))
	payloadWithTimestamp = append(payloadWithTimestamp, validPayload...)
	signature, err := identity.Sign(payloadWithTimestamp)
	require.NoError(t, err)
	expiredMsg := &v1.SignedMessage{
		Payload:         validPayload,
		Signature:       signature,
		Keyid:           identity.KeyID(),
		TimestampMillis: oldTimestamp,
	}

	// Create a message with a bad signature
	badSigMsg := &v1.SignedMessage{
		Payload:         validMsg.Payload,
		Signature:       []byte("bad signature"),
		Keyid:           identity.KeyID(),
		TimestampMillis: validMsg.TimestampMillis,
	}

	testCases := []struct {
		name        string
		msg         *v1.SignedMessage
		publicKey   *cryptoutil.PublicKey
		wantErr     bool
		expectedErr string
	}{
		{
			name:      "valid message",
			msg:       validMsg,
			publicKey: publicKey,
			wantErr:   false,
		},
		{
			name:        "nil message",
			msg:         nil,
			publicKey:   publicKey,
			wantErr:     true,
			expectedErr: "signed message must not be nil",
		},
		{
			name: "empty payload",
			msg: &v1.SignedMessage{
				Signature:       validMsg.Signature,
				Keyid:           identity.KeyID(),
				TimestampMillis: validMsg.TimestampMillis,
			},
			publicKey:   publicKey,
			wantErr:     true,
			expectedErr: "signed message payload must not be empty",
		},
		{
			name: "empty signature",
			msg: &v1.SignedMessage{
				Payload:         validMsg.Payload,
				Keyid:           identity.KeyID(),
				TimestampMillis: validMsg.TimestampMillis,
			},
			publicKey:   publicKey,
			wantErr:     true,
			expectedErr: "signed message signature must not be empty",
		},
		{
			name: "empty key id",
			msg: &v1.SignedMessage{
				Payload:         validMsg.Payload,
				Signature:       validMsg.Signature,
				TimestampMillis: validMsg.TimestampMillis,
			},
			publicKey:   publicKey,
			wantErr:     true,
			expectedErr: "signed message key ID must not be empty",
		},
		{
			name:        "key id mismatch",
			msg:         validMsg,
			publicKey:   otherPublicKey,
			wantErr:     true,
			expectedErr: "public key ID mismatch",
		},
		{
			name:        "invalid signature",
			msg:         badSigMsg,
			publicKey:   publicKey,
			wantErr:     true,
			expectedErr: "verifying signed message",
		},
		{
			name:        "expired signature",
			msg:         expiredMsg,
			publicKey:   publicKey,
			wantErr:     true,
			expectedErr: "signature is too old",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := verifySignedMessage(tc.msg, tc.publicKey)
			if tc.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tc.expectedErr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
