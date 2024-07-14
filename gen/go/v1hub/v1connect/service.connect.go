// Code generated by protoc-gen-connect-go. DO NOT EDIT.
//
// Source: v1hub/service.proto

package v1connect

import (
	connect "connectrpc.com/connect"
	context "context"
	errors "errors"
	v1 "github.com/garethgeorge/backrest/gen/go/v1"
	http "net/http"
	strings "strings"
)

// This is a compile-time assertion to ensure that this generated file and the connect package are
// compatible. If you get a compiler error that this constant is not defined, this code was
// generated with a version of connect newer than the one compiled into your binary. You can fix the
// problem by either regenerating this code with an older version of connect or updating the connect
// version compiled into your binary.
const _ = connect.IsAtLeastVersion1_13_0

const (
	// HubName is the fully-qualified name of the Hub service.
	HubName = "v1hub.Hub"
)

// These constants are the fully-qualified names of the RPCs defined in this package. They're
// exposed at runtime as Spec.Procedure and as the final two segments of the HTTP route.
//
// Note that these are different from the fully-qualified method names used by
// google.golang.org/protobuf/reflect/protoreflect. To convert from these constants to
// reflection-formatted method names, remove the leading slash and convert the remaining slash to a
// period.
const (
	// HubSyncOperationsProcedure is the fully-qualified name of the Hub's SyncOperations RPC.
	HubSyncOperationsProcedure = "/v1hub.Hub/SyncOperations"
	// HubGetConfigProcedure is the fully-qualified name of the Hub's GetConfig RPC.
	HubGetConfigProcedure = "/v1hub.Hub/GetConfig"
)

// These variables are the protoreflect.Descriptor objects for the RPCs defined in this package.
var (
	hubServiceDescriptor              = v1.File_v1hub_service_proto.Services().ByName("Hub")
	hubSyncOperationsMethodDescriptor = hubServiceDescriptor.Methods().ByName("SyncOperations")
	hubGetConfigMethodDescriptor      = hubServiceDescriptor.Methods().ByName("GetConfig")
)

// HubClient is a client for the v1hub.Hub service.
type HubClient interface {
	// SyncOperations is a bidirectional stream of operations.
	// The client pushes id, modno to the server and may optionally push the operation update itself.
	// The server responds with the id, modno of the latest operation data it has or -1 if it has no data.
	// The client pushes operations it detects to be missing or out of date on the server.
	SyncOperations(context.Context) *connect.BidiStreamForClient[v1.OpSyncMetadata, v1.OpSyncMetadata]
	// GetConfig returns a stream of config updates related to the instance_id.
	GetConfig(context.Context, *connect.Request[v1.GetConfigRequest]) (*connect.ServerStreamForClient[v1.Config], error)
}

// NewHubClient constructs a client for the v1hub.Hub service. By default, it uses the Connect
// protocol with the binary Protobuf Codec, asks for gzipped responses, and sends uncompressed
// requests. To use the gRPC or gRPC-Web protocols, supply the connect.WithGRPC() or
// connect.WithGRPCWeb() options.
//
// The URL supplied here should be the base URL for the Connect or gRPC server (for example,
// http://api.acme.com or https://acme.com/grpc).
func NewHubClient(httpClient connect.HTTPClient, baseURL string, opts ...connect.ClientOption) HubClient {
	baseURL = strings.TrimRight(baseURL, "/")
	return &hubClient{
		syncOperations: connect.NewClient[v1.OpSyncMetadata, v1.OpSyncMetadata](
			httpClient,
			baseURL+HubSyncOperationsProcedure,
			connect.WithSchema(hubSyncOperationsMethodDescriptor),
			connect.WithClientOptions(opts...),
		),
		getConfig: connect.NewClient[v1.GetConfigRequest, v1.Config](
			httpClient,
			baseURL+HubGetConfigProcedure,
			connect.WithSchema(hubGetConfigMethodDescriptor),
			connect.WithClientOptions(opts...),
		),
	}
}

// hubClient implements HubClient.
type hubClient struct {
	syncOperations *connect.Client[v1.OpSyncMetadata, v1.OpSyncMetadata]
	getConfig      *connect.Client[v1.GetConfigRequest, v1.Config]
}

// SyncOperations calls v1hub.Hub.SyncOperations.
func (c *hubClient) SyncOperations(ctx context.Context) *connect.BidiStreamForClient[v1.OpSyncMetadata, v1.OpSyncMetadata] {
	return c.syncOperations.CallBidiStream(ctx)
}

// GetConfig calls v1hub.Hub.GetConfig.
func (c *hubClient) GetConfig(ctx context.Context, req *connect.Request[v1.GetConfigRequest]) (*connect.ServerStreamForClient[v1.Config], error) {
	return c.getConfig.CallServerStream(ctx, req)
}

// HubHandler is an implementation of the v1hub.Hub service.
type HubHandler interface {
	// SyncOperations is a bidirectional stream of operations.
	// The client pushes id, modno to the server and may optionally push the operation update itself.
	// The server responds with the id, modno of the latest operation data it has or -1 if it has no data.
	// The client pushes operations it detects to be missing or out of date on the server.
	SyncOperations(context.Context, *connect.BidiStream[v1.OpSyncMetadata, v1.OpSyncMetadata]) error
	// GetConfig returns a stream of config updates related to the instance_id.
	GetConfig(context.Context, *connect.Request[v1.GetConfigRequest], *connect.ServerStream[v1.Config]) error
}

// NewHubHandler builds an HTTP handler from the service implementation. It returns the path on
// which to mount the handler and the handler itself.
//
// By default, handlers support the Connect, gRPC, and gRPC-Web protocols with the binary Protobuf
// and JSON codecs. They also support gzip compression.
func NewHubHandler(svc HubHandler, opts ...connect.HandlerOption) (string, http.Handler) {
	hubSyncOperationsHandler := connect.NewBidiStreamHandler(
		HubSyncOperationsProcedure,
		svc.SyncOperations,
		connect.WithSchema(hubSyncOperationsMethodDescriptor),
		connect.WithHandlerOptions(opts...),
	)
	hubGetConfigHandler := connect.NewServerStreamHandler(
		HubGetConfigProcedure,
		svc.GetConfig,
		connect.WithSchema(hubGetConfigMethodDescriptor),
		connect.WithHandlerOptions(opts...),
	)
	return "/v1hub.Hub/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case HubSyncOperationsProcedure:
			hubSyncOperationsHandler.ServeHTTP(w, r)
		case HubGetConfigProcedure:
			hubGetConfigHandler.ServeHTTP(w, r)
		default:
			http.NotFound(w, r)
		}
	})
}

// UnimplementedHubHandler returns CodeUnimplemented from all methods.
type UnimplementedHubHandler struct{}

func (UnimplementedHubHandler) SyncOperations(context.Context, *connect.BidiStream[v1.OpSyncMetadata, v1.OpSyncMetadata]) error {
	return connect.NewError(connect.CodeUnimplemented, errors.New("v1hub.Hub.SyncOperations is not implemented"))
}

func (UnimplementedHubHandler) GetConfig(context.Context, *connect.Request[v1.GetConfigRequest], *connect.ServerStream[v1.Config]) error {
	return connect.NewError(connect.CodeUnimplemented, errors.New("v1hub.Hub.GetConfig is not implemented"))
}