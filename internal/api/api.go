package api

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"

	v1 "github.com/garethgeorge/backrest/gen/go/v1"
	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/logging"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func loggingFunc(l *zap.Logger) logging.Logger {
	return logging.LoggerFunc(func(ctx context.Context, lvl logging.Level, msg string, fields ...any) {
		f := make([]zap.Field, 0, len(fields)/2)

		for i := 0; i < len(fields); i += 2 {
			key := fields[i]
			value := fields[i+1]

			switch v := value.(type) {
			case string:
				f = append(f, zap.String(key.(string), v))
			case int:
				f = append(f, zap.Int(key.(string), v))
			case bool:
				f = append(f, zap.Bool(key.(string), v))
			default:
				f = append(f, zap.Any(key.(string), v))
			}
		}

		logger := l.WithOptions(zap.AddCallerSkip(1)).With(f...)

		switch lvl {
		case logging.LevelDebug:
			logger.Debug(msg)
		case logging.LevelInfo:
			logger.Debug(msg)
		case logging.LevelWarn:
			logger.Warn(msg)
		case logging.LevelError:
			logger.Error(msg)
		default:
			panic(fmt.Sprintf("unknown level %v", lvl))
		}
	})
}

func serveGRPC(ctx context.Context, socket string, server *Server) error {
	lis, err := net.Listen("unix", socket)
	if err != nil {
		return fmt.Errorf("failed to listen: %w", err)
	}

	logger := zap.L()
	grpcServer := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			logging.UnaryServerInterceptor(loggingFunc(logger)),
		),
		grpc.ChainStreamInterceptor(
			logging.StreamServerInterceptor(loggingFunc(logger)),
		),
	)
	v1.RegisterBackrestServer(grpcServer, server)
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

func serveHTTPHandlers(ctx context.Context, server *Server, mux *runtime.ServeMux) error {
	tmpDir, err := os.MkdirTemp("", "backrest")
	if err != nil {
		return fmt.Errorf("failed to create temp dir for unix domain socket: %w", err)
	}
	defer func() {
		os.RemoveAll(tmpDir)
	}()

	socket := filepath.Join(tmpDir, "backrest.sock")

	opts := []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}
	err = v1.RegisterBackrestHandlerFromEndpoint(ctx, mux, fmt.Sprintf("unix:%v", socket), opts)
	if err != nil {
		return fmt.Errorf("failed to register gateway: %w", err)
	}

	if err := serveGRPC(ctx, socket, server); err != nil {
		return err
	}

	return nil
}

// Handler returns an http.Handler serving the API, cancel the context to cleanly shut down the server.
func ServeAPI(ctx context.Context, server *Server, mux *http.ServeMux) error {
	apiMux := runtime.NewServeMux()
	mux.Handle("/api/", http.StripPrefix("/api", apiMux))
	return serveHTTPHandlers(ctx, server, apiMux)
}
