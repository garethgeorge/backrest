package syncapi

import (
	v1 "github.com/garethgeorge/backrest/gen/go/v1"
	"github.com/garethgeorge/backrest/gen/go/v1sync/v1syncconnect"
)

type syncClient struct {
	client *v1syncconnect.SyncPeerServiceClient
	peer   *v1.Multihost_Peer
}

func newSyncClient(client *v1syncconnect.SyncPeerServiceClient, peer *v1.Multihost_Peer) *syncClient {
	return &syncClient{
		client: client,
		peer:   peer,
	}
}
