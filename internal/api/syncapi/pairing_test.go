package syncapi

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	v1 "github.com/garethgeorge/backrest/gen/go/v1"
	"github.com/garethgeorge/backrest/gen/go/v1sync"
	"github.com/garethgeorge/backrest/internal/config/migrations"
	"github.com/garethgeorge/backrest/internal/cryptoutil"
	"github.com/garethgeorge/backrest/internal/testutil"
)

func TestValidatePairingSecret(t *testing.T) {
	now := time.Unix(1000, 0)

	tokens := []*v1.Multihost_PairingToken{
		{
			Secret:        "valid-secret",
			Label:         "test-token",
			CreatedAtUnix: 900,
			ExpiresAtUnix: 2000,
			MaxUses:       3,
			Uses:          1,
		},
		{
			Secret:        "unlimited-token",
			Label:         "unlimited",
			CreatedAtUnix: 900,
			ExpiresAtUnix: 0, // no expiry
			MaxUses:       0, // unlimited uses
			Uses:          100,
		},
	}

	tests := []struct {
		name      string
		secret    string
		tokens    []*v1.Multihost_PairingToken
		now       time.Time
		wantLabel string
		wantErr   bool
	}{
		{
			name:      "valid secret",
			secret:    "valid-secret",
			tokens:    tokens,
			now:       now,
			wantLabel: "test-token",
		},
		{
			name:      "unlimited token",
			secret:    "unlimited-token",
			tokens:    tokens,
			now:       now,
			wantLabel: "unlimited",
		},
		{
			name:    "empty secret",
			secret:  "",
			tokens:  tokens,
			now:     now,
			wantErr: true,
		},
		{
			name:    "wrong secret",
			secret:  "wrong-secret",
			tokens:  tokens,
			now:     now,
			wantErr: true,
		},
		{
			name:   "expired token",
			secret: "valid-secret",
			tokens: []*v1.Multihost_PairingToken{
				{
					Secret:        "valid-secret",
					Label:         "expired",
					ExpiresAtUnix: 500,
					MaxUses:       0,
				},
			},
			now:     now,
			wantErr: true,
		},
		{
			name:   "max uses reached",
			secret: "valid-secret",
			tokens: []*v1.Multihost_PairingToken{
				{
					Secret:  "valid-secret",
					Label:   "exhausted",
					MaxUses: 2,
					Uses:    2,
				},
			},
			now:     now,
			wantErr: true,
		},
		{
			name:    "nil tokens",
			secret:  "anything",
			tokens:  nil,
			now:     now,
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			token, err := ValidatePairingSecret(tc.secret, tc.tokens, tc.now)
			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if token.Label != tc.wantLabel {
				t.Errorf("label = %q, want %q", token.Label, tc.wantLabel)
			}
		})
	}
}

func TestPairingTokenFlow(t *testing.T) {
	testutil.InstallZapLogger(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	peerHostAddr := testutil.AllocOpenBindAddr(t)
	peerClientAddr := testutil.AllocOpenBindAddr(t)

	pairingSecret, err := cryptoutil.GeneratePairingSecret()
	if err != nil {
		t.Fatalf("failed to generate pairing secret: %v", err)
	}

	// Host has a pairing token but NO authorized clients yet.
	peerHostConfig := &v1.Config{
		Version:  migrations.CurrentVersion,
		Instance: defaultHostID,
		Repos:    []*v1.Repo{},
		Multihost: &v1.Multihost{
			Identity:          identity1,
			AuthorizedClients: []*v1.Multihost_Peer{}, // empty — client not pre-authorized
			PairingTokens: []*v1.Multihost_PairingToken{
				{
					Secret:        pairingSecret,
					Label:         "test-pairing",
					CreatedAtUnix: time.Now().Unix(),
					ExpiresAtUnix: time.Now().Add(1 * time.Hour).Unix(),
					MaxUses:       1,
					Uses:          0,
				},
			},
		},
	}

	// Client knows about the host and has the pairing secret.
	peerClientConfig := &v1.Config{
		Version:  migrations.CurrentVersion,
		Instance: defaultClientID,
		Repos:    []*v1.Repo{},
		Multihost: &v1.Multihost{
			Identity: identity2,
			KnownHosts: []*v1.Multihost_Peer{
				{
					Keyid:                identity1.Keyid,
					InstanceId:           defaultHostID,
					InstanceUrl:          fmt.Sprintf("http://%s", peerHostAddr),
					InitialPairingSecret: pairingSecret,
				},
			},
		},
	}

	peerHost := newPeerUnderTest(t, peerHostConfig)
	peerClient := newPeerUnderTest(t, peerClientConfig)

	startRunningSyncAPI(t, peerHost, peerHostAddr)
	startRunningSyncAPI(t, peerClient, peerClientAddr)

	// The client should successfully connect via the pairing token.
	tryConnect(t, ctx, peerClient, peerClientConfig.Multihost.KnownHosts[0])

	// Verify the host now has the client in its authorized_clients.
	hostConfig, err := peerHost.configMgr.Get()
	if err != nil {
		t.Fatalf("failed to get host config: %v", err)
	}
	if len(hostConfig.Multihost.AuthorizedClients) != 1 {
		t.Fatalf("expected 1 authorized client, got %d", len(hostConfig.Multihost.AuthorizedClients))
	}
	ac := hostConfig.Multihost.AuthorizedClients[0]
	if ac.Keyid != identity2.Keyid {
		t.Errorf("authorized client keyid = %q, want %q", ac.Keyid, identity2.Keyid)
	}
	if ac.InstanceId != defaultClientID {
		t.Errorf("authorized client instance id = %q, want %q", ac.InstanceId, defaultClientID)
	}

	// Verify the pairing token was consumed (max_uses=1, so it should be removed).
	if len(hostConfig.Multihost.PairingTokens) != 0 {
		t.Errorf("expected 0 pairing tokens after consumption, got %d", len(hostConfig.Multihost.PairingTokens))
	}
}

func TestPairingTokenExpiredRejected(t *testing.T) {
	testutil.InstallZapLogger(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	peerHostAddr := testutil.AllocOpenBindAddr(t)
	peerClientAddr := testutil.AllocOpenBindAddr(t)

	pairingSecret, _ := cryptoutil.GeneratePairingSecret()

	// Host has an expired pairing token.
	peerHostConfig := &v1.Config{
		Version:  migrations.CurrentVersion,
		Instance: defaultHostID,
		Repos:    []*v1.Repo{},
		Multihost: &v1.Multihost{
			Identity:          identity1,
			AuthorizedClients: []*v1.Multihost_Peer{},
			PairingTokens: []*v1.Multihost_PairingToken{
				{
					Secret:        pairingSecret,
					Label:         "expired-token",
					CreatedAtUnix: time.Now().Add(-2 * time.Hour).Unix(),
					ExpiresAtUnix: time.Now().Add(-1 * time.Hour).Unix(), // expired 1 hour ago
					MaxUses:       0,
				},
			},
		},
	}

	peerClientConfig := &v1.Config{
		Version:  migrations.CurrentVersion,
		Instance: defaultClientID,
		Repos:    []*v1.Repo{},
		Multihost: &v1.Multihost{
			Identity: identity2,
			KnownHosts: []*v1.Multihost_Peer{
				{
					Keyid:                identity1.Keyid,
					InstanceId:           defaultHostID,
					InstanceUrl:          fmt.Sprintf("http://%s", peerHostAddr),
					InitialPairingSecret: pairingSecret,
				},
			},
		},
	}

	peerHost := newPeerUnderTest(t, peerHostConfig)
	peerClient := newPeerUnderTest(t, peerClientConfig)

	startRunningSyncAPI(t, peerHost, peerHostAddr)
	startRunningSyncAPI(t, peerClient, peerClientAddr)

	// Connection should fail with auth error.
	waitForConnectionState(t, ctx, peerClient, peerClientConfig.Multihost.KnownHosts[0], v1sync.ConnectionState_CONNECTION_STATE_ERROR_AUTH)

	// Host should still have no authorized clients.
	hostConfig, _ := peerHost.configMgr.Get()
	if len(hostConfig.Multihost.AuthorizedClients) != 0 {
		t.Errorf("expected 0 authorized clients, got %d", len(hostConfig.Multihost.AuthorizedClients))
	}
}

func TestPairingTokenMaxUsesEnforced(t *testing.T) {
	testutil.InstallZapLogger(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	peerHostAddr := testutil.AllocOpenBindAddr(t)

	pairingSecret, _ := cryptoutil.GeneratePairingSecret()

	identity3, _ := cryptoutil.GeneratePrivateKey()

	// Host has a pairing token with max_uses=1.
	peerHostConfig := &v1.Config{
		Version:  migrations.CurrentVersion,
		Instance: defaultHostID,
		Repos:    []*v1.Repo{},
		Multihost: &v1.Multihost{
			Identity:          identity1,
			AuthorizedClients: []*v1.Multihost_Peer{},
			PairingTokens: []*v1.Multihost_PairingToken{
				{
					Secret:        pairingSecret,
					Label:         "single-use",
					CreatedAtUnix: time.Now().Unix(),
					ExpiresAtUnix: time.Now().Add(1 * time.Hour).Unix(),
					MaxUses:       1,
					Uses:          0,
				},
			},
		},
	}

	// First client pairs successfully.
	peerClient1Config := &v1.Config{
		Version:  migrations.CurrentVersion,
		Instance: "client-1",
		Repos:    []*v1.Repo{},
		Multihost: &v1.Multihost{
			Identity: identity2,
			KnownHosts: []*v1.Multihost_Peer{
				{
					Keyid:                identity1.Keyid,
					InstanceId:           defaultHostID,
					InstanceUrl:          fmt.Sprintf("http://%s", peerHostAddr),
					InitialPairingSecret: pairingSecret,
				},
			},
		},
	}

	peerHost := newPeerUnderTest(t, peerHostConfig)
	peerClient1 := newPeerUnderTest(t, peerClient1Config)

	var wg sync.WaitGroup
	syncCtx, cancelSync := context.WithCancel(ctx)

	wg.Add(2)
	go func() { defer wg.Done(); runSyncAPIWithCtx(syncCtx, peerHost, peerHostAddr) }()
	go func() {
		defer wg.Done()
		peerClient1Addr := testutil.AllocOpenBindAddr(t)
		runSyncAPIWithCtx(syncCtx, peerClient1, peerClient1Addr)
	}()

	tryConnect(t, ctx, peerClient1, peerClient1Config.Multihost.KnownHosts[0])

	// Stop first client, start second client with same pairing secret.
	cancelSync()
	wg.Wait()

	peerClient2Config := &v1.Config{
		Version:  migrations.CurrentVersion,
		Instance: "client-2",
		Repos:    []*v1.Repo{},
		Multihost: &v1.Multihost{
			Identity: identity3,
			KnownHosts: []*v1.Multihost_Peer{
				{
					Keyid:                identity1.Keyid,
					InstanceId:           defaultHostID,
					InstanceUrl:          fmt.Sprintf("http://%s", peerHostAddr),
					InitialPairingSecret: pairingSecret,
				},
			},
		},
	}

	peerClient2 := newPeerUnderTest(t, peerClient2Config)

	startRunningSyncAPI(t, peerHost, peerHostAddr)
	startRunningSyncAPI(t, peerClient2, testutil.AllocOpenBindAddr(t))

	// Second client should fail — token is consumed.
	waitForConnectionState(t, ctx, peerClient2, peerClient2Config.Multihost.KnownHosts[0], v1sync.ConnectionState_CONNECTION_STATE_ERROR_AUTH)
}

func TestNoPairingSecretRejected(t *testing.T) {
	testutil.InstallZapLogger(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	peerHostAddr := testutil.AllocOpenBindAddr(t)
	peerClientAddr := testutil.AllocOpenBindAddr(t)

	// Host has NO pairing tokens and NO authorized clients.
	peerHostConfig := &v1.Config{
		Version:  migrations.CurrentVersion,
		Instance: defaultHostID,
		Repos:    []*v1.Repo{},
		Multihost: &v1.Multihost{
			Identity:          identity1,
			AuthorizedClients: []*v1.Multihost_Peer{},
		},
	}

	// Client tries to connect without any pairing secret.
	peerClientConfig := &v1.Config{
		Version:  migrations.CurrentVersion,
		Instance: defaultClientID,
		Repos:    []*v1.Repo{},
		Multihost: &v1.Multihost{
			Identity: identity2,
			KnownHosts: []*v1.Multihost_Peer{
				{
					Keyid:       identity1.Keyid,
					InstanceId:  defaultHostID,
					InstanceUrl: fmt.Sprintf("http://%s", peerHostAddr),
				},
			},
		},
	}

	peerHost := newPeerUnderTest(t, peerHostConfig)
	peerClient := newPeerUnderTest(t, peerClientConfig)

	startRunningSyncAPI(t, peerHost, peerHostAddr)
	startRunningSyncAPI(t, peerClient, peerClientAddr)

	waitForConnectionState(t, ctx, peerClient, peerClientConfig.Multihost.KnownHosts[0], v1sync.ConnectionState_CONNECTION_STATE_ERROR_AUTH)
}
