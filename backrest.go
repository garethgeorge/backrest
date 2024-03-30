package main

import (
	"context"
	"crypto/rand"
	"errors"
	"flag"
	"net/http"
	"os"
	"os/signal"
	"path"
	"strings"
	"sync"
	"syscall"

	"github.com/garethgeorge/backrest/gen/go/v1/v1connect"
	"github.com/garethgeorge/backrest/internal/api"
	"github.com/garethgeorge/backrest/internal/auth"
	"github.com/garethgeorge/backrest/internal/config"
	"github.com/garethgeorge/backrest/internal/oplog"
	"github.com/garethgeorge/backrest/internal/orchestrator"
	"github.com/garethgeorge/backrest/internal/resticinstaller"
	"github.com/garethgeorge/backrest/internal/rotatinglog"
	"github.com/garethgeorge/backrest/webui"
	"github.com/mattn/go-colorable"
	"go.etcd.io/bbolt"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
)

var InstallDepsOnly = flag.Bool("install-deps-only", false, "install dependencies and exit")

func main() {
	flag.Parse()

	resticPath, err := resticinstaller.FindOrInstallResticBinary()
	if err != nil {
		zap.S().Fatalf("Error finding or installing restic: %v", err)
	}

	if *InstallDepsOnly {
		zap.S().Info("Dependencies installed, exiting")
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	go onterm(cancel)

	// Load the configuration
	configStore := createConfigProvider()
	cfg, err := configStore.Get()
	if err != nil {
		zap.S().Fatalf("Error loading config: %v", err)
	}

	// Create the authenticator
	authenticator := auth.NewAuthenticator(getSecret(), configStore)

	var wg sync.WaitGroup

	// Create / load the operation log
	oplogFile := path.Join(config.DataDir(), "oplog.boltdb")
	oplog, err := oplog.NewOpLog(oplogFile)
	if err != nil {
		if !errors.Is(err, bbolt.ErrTimeout) {
			zap.S().Fatalf("Timeout while waiting to open database, is the database open elsewhere?")
		}
		zap.S().Warnf("Operation log may be corrupted, if errors recur delete the file %q and restart. Your backups stored in your repos are safe.", oplogFile)
		zap.S().Fatalf("Error creating oplog : %v", err)
	}
	defer oplog.Close()

	// Create rotating log storage
	logStore := rotatinglog.NewRotatingLog(path.Join(config.DataDir(), "rotatinglogs"), 30) // 30 days of logs
	if err != nil {
		zap.S().Fatalf("Error creating rotating log storage: %v", err)
	}

	// Create orchestrator and start task loop.
	orchestrator, err := orchestrator.NewOrchestrator(resticPath, cfg, oplog, logStore)
	if err != nil {
		zap.S().Fatalf("Error creating orchestrator: %v", err)
	}

	wg.Add(1)
	go func() {
		orchestrator.Run(ctx)
		wg.Done()
	}()

	// Create and serve the HTTP gateway
	apiBackrestHandler := api.NewBackrestHandler(
		configStore,
		orchestrator,
		oplog,
		logStore,
	)

	apiAuthenticationHandler := api.NewAuthenticationHandler(authenticator)

	mux := http.NewServeMux()
	mux.Handle(v1connect.NewAuthenticationHandler(apiAuthenticationHandler))
	backrestHandlerPath, backrestHandler := v1connect.NewBackrestHandler(apiBackrestHandler)
	mux.Handle(backrestHandlerPath, auth.RequireAuthentication(backrestHandler, authenticator))
	mux.Handle("/", webui.Handler())

	// Serve the HTTP gateway
	server := &http.Server{
		Addr:    config.BindAddress(),
		Handler: h2c.NewHandler(mux, &http2.Server{}), // h2c is HTTP/2 without TLS for grpc-connect support.
	}

	zap.S().Infof("Starting web server %v", server.Addr)
	go func() {
		<-ctx.Done()
		server.Shutdown(context.Background())
	}()
	if err := server.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
		zap.L().Error("Error starting server", zap.Error(err))
	}
	zap.L().Info("HTTP gateway shutdown")

	wg.Wait()
}

func init() {
	zap.ReplaceGlobals(zap.Must(zap.NewProduction()))
	if !strings.HasPrefix(os.Getenv("ENV"), "prod") {
		c := zap.NewDevelopmentEncoderConfig()
		c.EncodeLevel = zapcore.CapitalColorLevelEncoder
		c.EncodeTime = zapcore.ISO8601TimeEncoder
		l := zap.New(zapcore.NewCore(
			zapcore.NewConsoleEncoder(c),
			zapcore.AddSync(colorable.NewColorableStdout()),
			zapcore.DebugLevel,
		))
		zap.ReplaceGlobals(l)
	}
}

func createConfigProvider() config.ConfigStore {
	return &config.CachingValidatingStore{
		ConfigStore: &config.JsonFileStore{Path: config.ConfigFilePath()},
	}
}

func onterm(callback func()) {
	sigchan := make(chan os.Signal, 1)
	signal.Notify(sigchan, os.Interrupt, syscall.SIGTERM)
	<-sigchan
	callback()
}

func getSecret() []byte {
	secretFile := path.Join(config.DataDir(), "jwt-secret")
	data, err := os.ReadFile(secretFile)
	if err == nil {
		zap.L().Debug("Loaded auth secret from file")
		return data
	}

	zap.L().Info("Generating new auth secret")
	secret := make([]byte, 64)
	if n, err := rand.Read(secret); err != nil || n != 64 {
		zap.S().Fatalf("Error generating secret: %v", err)
	}
	if err := os.MkdirAll(config.DataDir(), 0700); err != nil {
		zap.S().Fatalf("Error creating data directory: %v", err)
	}
	if err := os.WriteFile(secretFile, secret, 0600); err != nil {
		zap.S().Fatalf("Error writing secret to file: %v", err)
	}
	return secret
}
