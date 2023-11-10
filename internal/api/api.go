package api

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"

	v1 "github.com/garethgeorge/resticui/gen/go/v1"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func serveGRPC(ctx context.Context, socket string) error {
	lis, err := net.Listen("unix", socket)
	if err != nil {
		return fmt.Errorf("failed to listen: %w", err)
	}
	grpcServer := grpc.NewServer()
	v1.RegisterResticUIServer(grpcServer, &server{})
	go func() {
		<-ctx.Done()
		grpcServer.GracefulStop()
	}()
	err = grpcServer.Serve(lis)
	if err != nil {
		return fmt.Errorf("grpc serving error: %w", err)
	}
	return nil
}

func serveHTTPHandlers(ctx context.Context, mux *runtime.ServeMux) error {
	tmpDir, err := os.MkdirTemp("", "resticui")
	if err != nil {
		return fmt.Errorf("failed to create temp dir for unix domain socket: %w", err)
	}
	defer func() {
		os.RemoveAll(tmpDir)
	}()

	socket := filepath.Join(tmpDir, "resticui.sock")

	opts := []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}
	err = v1.RegisterResticUIHandlerFromEndpoint(ctx, mux, fmt.Sprintf("unix:%v", socket), opts)
	if err != nil {
		return fmt.Errorf("failed to register gateway: %w", err)
	}

	if err := serveGRPC(ctx, socket); err != nil {
		return err
	}

	return nil
}

// Handler returns an http.Handler serving the API, cancel the context to cleanly shut down the server.
func ServeAPI(ctx context.Context, mux *http.ServeMux) error {
	apiMux := runtime.NewServeMux()
	mux.Handle("/api/", http.StripPrefix("/api", apiMux))
	return serveHTTPHandlers(ctx, apiMux)
}