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
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"

	"github.com/garethgeorge/backrest/gen/go/v1/v1connect"
	"github.com/garethgeorge/backrest/internal/api"
	"github.com/garethgeorge/backrest/internal/auth"
	"github.com/garethgeorge/backrest/internal/config"
	"github.com/garethgeorge/backrest/internal/env"
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
		zap.S().Fatalf("error finding or installing restic: %v", err)
	}

	if *InstallDepsOnly {
		zap.S().Info("dependencies installed, exiting")
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	go onterm(os.Interrupt, cancel)
	go onterm(os.Interrupt, newForceKillHandler())

	// Load the configuration
	configStore := createConfigProvider()
	cfg, err := configStore.Get()
	if err != nil {
		zap.S().Fatalf("error loading config: %v", err)
	}
	configManager := &config.ConfigManager{ConfigStore: configStore}

	// Create the authenticator
	authenticator := auth.NewAuthenticator(getSecret(), configStore)

	var wg sync.WaitGroup

	// Create / load the operation log
	oplogFile := path.Join(env.DataDir(), "oplog.boltdb")
	oplog, err := oplog.NewOpLog(oplogFile)
	if err != nil {
		if !errors.Is(err, bbolt.ErrTimeout) {
			zap.S().Fatalf("timeout while waiting to open database, is the database open elsewhere?")
		}
		zap.S().Warnf("operation log may be corrupted, if errors recur delete the file %q and restart. Your backups stored in your repos are safe.", oplogFile)
		zap.S().Fatalf("error creating oplog : %v", err)
	}
	defer oplog.Close()

	// Create rotating log storage
	logStore := rotatinglog.NewRotatingLog(path.Join(env.DataDir(), "rotatinglogs"), 14) // 14 days of logs
	if err != nil {
		zap.S().Fatalf("error creating rotating log storage: %v", err)
	}

	// Create orchestrator and start task loop.
	orchestrator, err := orchestrator.NewOrchestrator(resticPath, configManager, oplog, logStore)
	if err != nil {
		zap.S().Fatalf("error creating orchestrator: %v", err)
	}
	defer orchestrator.Shutdown()

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
		env.IsHubServer(),
	)

	apiAuthenticationHandler := api.NewAuthenticationHandler(authenticator)

	mux := http.NewServeMux()
	mux.Handle(v1connect.NewAuthenticationHandler(apiAuthenticationHandler))
	backrestHandlerPath, backrestHandler := v1connect.NewBackrestHandler(apiBackrestHandler)
	mux.Handle(backrestHandlerPath, auth.RequireAuthentication(backrestHandler, authenticator))
	mux.Handle("/", webui.Handler())
	mux.Handle("/download/", http.StripPrefix("/download", api.NewDownloadHandler(oplog)))

	// Serve the HTTP gateway
	server := &http.Server{
		Addr:    env.BindAddress(),
		Handler: h2c.NewHandler(mux, &http2.Server{}), // h2c is HTTP/2 without TLS for grpc-connect support.
	}

	zap.S().Infof("starting web server %v", server.Addr)
	go func() {
		<-ctx.Done()
		server.Shutdown(context.Background())
	}()
	if err := server.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
		zap.L().Error("error starting server", zap.Error(err))
	}
	zap.L().Info("HTTP gateway shutdown")

	wg.Wait()
}

func init() {
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
	} else {
		zap.ReplaceGlobals(zap.New(zapcore.NewCore(
			zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()),
			zapcore.AddSync(os.Stdout),
			zapcore.DebugLevel,
		)))
	}
}

func createConfigProvider() config.ConfigStore {
	return &config.CachingValidatingStore{
		ConfigStore: &config.JsonFileStore{Path: env.ConfigFilePath()},
	}
}

func onterm(s os.Signal, callback func()) {
	sigchan := make(chan os.Signal, 1)
	signal.Notify(sigchan, s, syscall.SIGTERM)
	for {
		<-sigchan
		callback()
	}
}

func getSecret() []byte {
	secretFile := path.Join(env.DataDir(), "jwt-secret")
	data, err := os.ReadFile(secretFile)
	if err == nil {
		zap.L().Debug("loading auth secret from file")
		return data
	}

	zap.L().Info("generating new auth secret")
	secret := make([]byte, 64)
	if n, err := rand.Read(secret); err != nil || n != 64 {
		zap.S().Fatalf("error generating secret: %v", err)
	}
	if err := os.MkdirAll(env.DataDir(), 0700); err != nil {
		zap.S().Fatalf("error creating data directory: %v", err)
	}
	if err := os.WriteFile(secretFile, secret, 0600); err != nil {
		zap.S().Fatalf("error writing secret to file: %v", err)
	}
	return secret
}

func newForceKillHandler() func() {
	var times atomic.Int32
	return func() {
		if times.Load() > 0 {
			buf := make([]byte, 1<<16)
			runtime.Stack(buf, true)
			os.Stderr.Write(buf)
			zap.S().Fatal("dumped all running coroutine stack traces, forcing termination")
		}
		times.Add(1)
		zap.S().Warn("attempting graceful shutdown, to force termination press Ctrl+C again")
	}
}
