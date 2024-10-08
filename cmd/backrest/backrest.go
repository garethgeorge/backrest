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
	"path/filepath"
	"runtime"
	"sync"
	"sync/atomic"
	"syscall"

	"github.com/garethgeorge/backrest/gen/go/v1/v1connect"
	"github.com/garethgeorge/backrest/internal/api"
	"github.com/garethgeorge/backrest/internal/auth"
	"github.com/garethgeorge/backrest/internal/config"
	"github.com/garethgeorge/backrest/internal/env"
	"github.com/garethgeorge/backrest/internal/logwriter"
	"github.com/garethgeorge/backrest/internal/metric"
	"github.com/garethgeorge/backrest/internal/oplog"
	"github.com/garethgeorge/backrest/internal/oplog/bboltstore"
	"github.com/garethgeorge/backrest/internal/orchestrator"
	"github.com/garethgeorge/backrest/internal/resticinstaller"
	"github.com/garethgeorge/backrest/webui"
	"github.com/mattn/go-colorable"
	"go.etcd.io/bbolt"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
	"gopkg.in/natefinch/lumberjack.v2"
)

var InstallDepsOnly = flag.Bool("install-deps-only", false, "install dependencies and exit")

func main() {
	flag.Parse()
	installLoggers()

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

	var wg sync.WaitGroup

	// Create / load the operation log
	oplogFile := path.Join(env.DataDir(), "oplog.boltdb")
	opstore, err := bboltstore.NewBboltStore(oplogFile)
	if err != nil {
		if !errors.Is(err, bbolt.ErrTimeout) {
			zap.S().Fatalf("timeout while waiting to open database, is the database open elsewhere?")
		}
		zap.S().Warnf("operation log may be corrupted, if errors recur delete the file %q and restart. Your backups stored in your repos are safe.", oplogFile)
		zap.S().Fatalf("error creating oplog : %v", err)
	}
	defer opstore.Close()

	oplog, err := oplog.NewOpLog(opstore)
	if err != nil {
		zap.S().Fatalf("error creating oplog: %v", err)
	}

	// Create rotating log storage
	logStore, err := logwriter.NewLogManager(path.Join(env.DataDir(), "rotatinglogs"), 14) // 14 days of logs
	if err != nil {
		zap.S().Fatalf("error creating rotating log storage: %v", err)
	}

	// Create orchestrator and start task loop.
	orchestrator, err := orchestrator.NewOrchestrator(resticPath, cfg, oplog, logStore)
	if err != nil {
		zap.S().Fatalf("error creating orchestrator: %v", err)
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

	authenticator := auth.NewAuthenticator(getSecret(), configStore)
	apiAuthenticationHandler := api.NewAuthenticationHandler(authenticator)

	mux := http.NewServeMux()
	mux.Handle(v1connect.NewAuthenticationHandler(apiAuthenticationHandler))
	backrestHandlerPath, backrestHandler := v1connect.NewBackrestHandler(apiBackrestHandler)
	mux.Handle(backrestHandlerPath, auth.RequireAuthentication(backrestHandler, authenticator))
	mux.Handle("/", webui.Handler())
	mux.Handle("/download/", http.StripPrefix("/download", api.NewDownloadHandler(oplog)))
	mux.Handle("/metrics", auth.RequireAuthentication(metric.GetRegistry().Handler(), authenticator))

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

func installLoggers() {
	// Pretty logging for console
	c := zap.NewDevelopmentEncoderConfig()
	c.EncodeLevel = zapcore.CapitalColorLevelEncoder
	c.EncodeTime = zapcore.ISO8601TimeEncoder
	pretty := zapcore.NewCore(
		zapcore.NewConsoleEncoder(c),
		zapcore.AddSync(colorable.NewColorableStdout()),
		zapcore.InfoLevel,
	)

	// JSON logging to log directory
	logsDir := env.LogsPath()
	if err := os.MkdirAll(logsDir, 0755); err != nil {
		zap.ReplaceGlobals(zap.New(pretty))
		zap.S().Errorf("error creating logs directory %q, will only log to console for now: %v", err)
		return
	}

	writer := &lumberjack.Logger{
		Filename:   filepath.Join(logsDir, "backrest.log"),
		MaxSize:    5, // megabytes
		MaxBackups: 3,
		MaxAge:     14,
		Compress:   true,
	}

	ugly := zapcore.NewCore(
		zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()),
		zapcore.AddSync(writer),
		zapcore.DebugLevel,
	)

	zap.ReplaceGlobals(zap.New(zapcore.NewTee(pretty, ugly)))
	zap.S().Infof("writing logs to: %v", logsDir)
}
