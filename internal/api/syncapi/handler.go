package syncapi

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sync"
	"time"
	"unique"

	"connectrpc.com/connect"
	"github.com/garethgeorge/backrest/gen/go/types"
	v1 "github.com/garethgeorge/backrest/gen/go/v1"
	"github.com/garethgeorge/backrest/gen/go/v1sync"
	"github.com/garethgeorge/backrest/gen/go/v1sync/v1syncconnect"
	"github.com/garethgeorge/backrest/internal/logstore"
	"github.com/garethgeorge/backrest/internal/oplog"
	"github.com/garethgeorge/backrest/internal/protoutil"
	lru "github.com/hashicorp/golang-lru/v2"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/emptypb"
)

type opIdCacheKey struct {
	OriginalInstanceKeyid unique.Handle[string]
	ID                    int64
}

// syncHandler provides server component functionality for the sync API.
type syncHandler struct {
	v1syncconnect.UnimplementedSyncPeerServiceHandler
	oplog    *oplog.OpLog
	logStore *logstore.LogStore

	opCacheMu sync.Mutex
	flowIDLru *lru.Cache[opIdCacheKey, int64]
	opIDLru   *lru.Cache[opIdCacheKey, int64]

	logger *zap.Logger
}

func NewSyncHandler(oplog *oplog.OpLog, logStore *logstore.LogStore) *syncHandler {
	// Both caches want to be reasonably large to avoid db lookups.
	flowIDLru, _ := lru.New[opIdCacheKey, int64](4 * 1024)
	opIDLru, _ := lru.New[opIdCacheKey, int64](16 * 1024)

	return &syncHandler{
		oplog:     oplog,
		logStore:  logStore,
		flowIDLru: flowIDLru,
		opIDLru:   opIDLru,
		logger:    zap.NewNop(),
	}
}

var _ v1syncconnect.SyncPeerServiceHandler = (*syncHandler)(nil)

func (sh *syncHandler) SetLogger(logger *zap.Logger) *syncHandler {
	if logger == nil {
		logger = zap.NewNop()
	}
	sh.logger = logger.Named("synchandler")
	return sh
}

// translateSingleID translates a single ID (either opID or flowID) using the provided cache and query
func (sh *syncHandler) translateSingleID(
	originalInstanceKeyid string,
	originalID int64,
	cache *lru.Cache[opIdCacheKey, int64],
	query oplog.Query,
) (int64, error) {
	if originalID == 0 {
		return 0, nil
	}

	cacheKey := opIdCacheKey{
		OriginalInstanceKeyid: unique.Make(originalInstanceKeyid),
		ID:                    originalID,
	}

	// Check cache first
	if translatedID, ok := cache.Get(cacheKey); ok {
		return translatedID, nil
	}

	// Cache miss - query the database
	op, err := sh.oplog.FindOneMetadata(query)
	if err != nil {
		if errors.Is(err, oplog.ErrNoResults) {
			return 0, nil // No results means the ID is not found
		}
		return 0, err // Other errors should be propagated
	}

	// Cache the result and return
	translatedID := op.FlowID
	cache.Add(cacheKey, translatedID)
	return translatedID, nil
}

func (sh *syncHandler) translateOpIdAndFlowID(originalInstanceKeyid string, originalOpId int64, originalFlowId int64) (int64, int64, error) {
	sh.opCacheMu.Lock()
	defer sh.opCacheMu.Unlock()

	// Translate opID
	opID, err := sh.translateSingleID(
		originalInstanceKeyid,
		originalOpId,
		sh.opIDLru,
		oplog.Query{
			OriginalInstanceKeyid: &originalInstanceKeyid,
			OriginalID:            &originalOpId,
		},
	)
	if err != nil {
		return 0, 0, err
	}

	// Translate flowID
	flowID, err := sh.translateSingleID(
		originalInstanceKeyid,
		originalFlowId,
		sh.flowIDLru,
		oplog.Query{
			OriginalInstanceKeyid: &originalInstanceKeyid,
			OriginalFlowID:        &originalFlowId,
		},
	)
	if err != nil {
		return 0, 0, err
	}

	return opID, flowID, nil
}

func (sh *syncHandler) GetOperationMetadata(ctx context.Context, req *connect.Request[v1.OpSelector]) (*connect.Response[v1sync.GetOperationMetadataResponse], error) {
	peer := PeerFromContext(ctx)
	if peer == nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("no peer found in context"))
	}

	// Check if the peer can read the selector
	if req.Msg.OriginalInstanceKeyid == nil || *req.Msg.OriginalInstanceKeyid != peer.Keyid {
		return nil, connect.NewError(connect.CodePermissionDenied, errors.New("GetOperationMetadata: peer must specify original instance keyid"))
	}

	sel, err := protoutil.OpSelectorToQuery(req.Msg)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	var opIDs []int64
	var modNos []int64

	if err := sh.oplog.QueryMetadata(sel, func(op oplog.OpMetadata) error {
		opIDs = append(opIDs, op.OriginalID)
		modNos = append(modNos, op.Modno)
		return nil
	}); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&v1sync.GetOperationMetadataResponse{
		OpIds:  opIDs,
		Modnos: modNos,
	}), nil
}

func (sh *syncHandler) SendOperations(ctx context.Context, stream *connect.ClientStream[v1.Operation]) (*connect.Response[emptypb.Empty], error) {
	peer := PeerFromContext(ctx)
	if peer == nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("no peer found in context"))
	}

	for stream.Receive() {
		op := stream.Msg()
		if op == nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("received nil operation"))
		}

		id, flowID, err := sh.translateOpIdAndFlowID(peer.Keyid, op.Id, op.FlowId)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}

		// Update the operation with the translated IDs
		opCopy := proto.Clone(op).(*v1.Operation)
		opCopy.Id = id
		opCopy.FlowId = flowID

		// Set the operation in the oplog
		if err := sh.oplog.Set(opCopy); err != nil {
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("set operation: %w", err))
		}
	}

	return nil, connect.NewError(connect.CodeUnimplemented, errors.New("v1sync.SyncPeerService.SendOperations is not implemented"))
}

func (sh *syncHandler) GetLog(ctx context.Context, req *connect.Request[types.StringValue], stream *connect.ServerStream[v1sync.LogDataEntry]) error {
	peer := PeerFromContext(ctx)
	if peer == nil {
		return connect.NewError(connect.CodeUnauthenticated, errors.New("no peer found in context"))
	}
	logID := req.Msg.Value

	metadata, err := sh.logStore.GetMetadata(logID)
	if err != nil {
		if errors.Is(err, logstore.ErrLogNotFound) {
			return connect.NewError(connect.CodeNotFound, fmt.Errorf("log with ID %s not found", logID))
		}
		return connect.NewError(connect.CodeInternal, fmt.Errorf("get log metadata: %w", err))
	}

	log, err := sh.logStore.Open(logID)
	if err != nil {
		return connect.NewError(connect.CodeInternal, fmt.Errorf("get log: %w", err))
	} else if log == nil {
		return connect.NewError(connect.CodeNotFound, fmt.Errorf("log with ID %s not found", logID))
	}
	defer log.Close()

	// Send first entry with log ID and owner operation ID
	entry := &v1sync.LogDataEntry{
		LogId:     logID,
		OwnerOpid: metadata.OwnerOpID,
	}
	if metadata.ExpirationTime != (time.Time{}) {
		entry.ExpirationTsUnix = metadata.ExpirationTime.Unix()
	}
	if err := stream.Send(entry); err != nil {
		if errors.Is(err, io.EOF) {
			return nil // Client closed the stream
		}
		return connect.NewError(connect.CodeInternal, fmt.Errorf("send log entry: %w", err))
	}

	// Read the log in chunks and send each chunk as a LogDataEntry
	buf := make([]byte, 0, 32*1024)
	for {
		n, err := log.Read(buf)
		if err != nil {
			if errors.Is(err, io.EOF) {
				break // End of log
			}
			return connect.NewError(connect.CodeInternal, fmt.Errorf("read log: %w", err))
		}
		if n == 0 {
			break
		}
		bytes := buf[:n]
		entry := &v1sync.LogDataEntry{
			Chunk: bytes,
		}
		if err := stream.Send(entry); err != nil {
			if errors.Is(err, io.EOF) {
				break // Client closed the stream
			}
			return connect.NewError(connect.CodeInternal, fmt.Errorf("send log entry: %w", err))
		}
	}
	if err := log.Close(); err != nil {
		return connect.NewError(connect.CodeInternal, fmt.Errorf("close log: %w", err))
	}
	return nil
}

func (sh *syncHandler) SetAvailableResources(ctx context.Context, req *connect.Request[v1sync.SetAvailableResourcesRequest]) (*connect.Response[emptypb.Empty], error) {
	peer := PeerFromContext(ctx)
	if peer == nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("no peer found in context"))
	}
	return nil, connect.NewError(connect.CodeUnimplemented, errors.New("v1sync.SyncPeerService.SetAvailableResources is not implemented"))
}

func (sh *syncHandler) SetConfig(context.Context, *connect.Request[v1sync.SetConfigRequest]) (*connect.Response[emptypb.Empty], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, errors.New("v1sync.SyncPeerService.SetConfig is not implemented"))
}

func (sh *syncHandler) GetConfig(context.Context, *connect.Request[emptypb.Empty]) (*connect.Response[v1sync.RemoteConfig], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, errors.New("v1sync.SyncPeerService.GetConfig is not implemented"))
}

type syncHandlerClient struct {
}
