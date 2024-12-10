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

	v1 "github.com/garethgeorge/backrest/gen/go/v1"
	"github.com/garethgeorge/backrest/gen/go/v1/v1connect"
	"github.com/garethgeorge/backrest/internal/api"
	syncapi "github.com/garethgeorge/backrest/internal/api/syncapi"
	"github.com/garethgeorge/backrest/internal/auth"
	"github.com/garethgeorge/backrest/internal/config"
	"github.com/garethgeorge/backrest/internal/env"
	"github.com/garethgeorge/backrest/internal/logstore"
	"github.com/garethgeorge/backrest/internal/metric"
	"github.com/garethgeorge/backrest/internal/oplog"
	"github.com/garethgeorge/backrest/internal/oplog/bboltstore"
	"github.com/garethgeorge/backrest/internal/oplog/sqlitestore"
	"github.com/garethgeorge/backrest/internal/orchestrator"
	"github.com/garethgeorge/backrest/internal/resticinstaller"
	"github.com/garethgeorge/backrest/webui"
	"github.com/mattn/go-colorable"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
	"gopkg.in/natefinch/lumberjack.v2"
)

var InstallDepsOnly = flag.Bool("install-deps-only", false, "install dependencies and exit")
var (
	version = "unknown"
	commit  = "unknown"
)

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
	configMgr := &config.ConfigManager{Store: configStore}

	var wg sync.WaitGroup

	// Create / load the operation log
	oplogFile := path.Join(env.DataDir(), "oplog.sqlite")
	opstore, err := sqlitestore.NewSqliteStore(oplogFile)
	if errors.Is(err, sqlitestore.ErrLocked) {
		zap.S().Fatalf("oplog is locked by another instance of backrest that is using the same data directory %q, kill that instance before starting another one.", env.DataDir())
	} else if err != nil {
		zap.S().Warnf("operation log may be corrupted, if errors recur delete the file %q and restart. Your backups stored in your repos are safe.", oplogFile)
		zap.S().Fatalf("error creating oplog: %v", err)
	}
	defer opstore.Close()

	log, err := oplog.NewOpLog(opstore)
	if err != nil {
		zap.S().Fatalf("error creating oplog: %v", err)
	}
	migrateBboltOplog(opstore)

	// Create rotating log storage
	logStore, err := logstore.NewLogStore(filepath.Join(env.DataDir(), "tasklogs"))
	if err != nil {
		zap.S().Fatalf("error creating task log store: %v", err)
	}
	logstore.MigrateTarLogsInDir(logStore, filepath.Join(env.DataDir(), "rotatinglogs"))
	deleteLogsForOp := func(ops []*v1.Operation, event oplog.OperationEvent) {
		if event != oplog.OPERATION_DELETED {
			return
		}
		for _, op := range ops {
			if err := logStore.DeleteWithParent(op.Id); err != nil {
				zap.S().Warnf("error deleting logs for operation %q: %v", op.Id, err)
			}
		}
	}
	log.Subscribe(oplog.Query{}, &deleteLogsForOp)
	defer func() {
		if err := logStore.Close(); err != nil {
			zap.S().Warnf("error closing log store: %v", err)
		}
		log.Unsubscribe(&deleteLogsForOp)
	}()

	// Create orchestrator and start task loop.
	// TODO: update the orchestrator to accept a configMgr and auto-refresh the config w/o explicit ApplyConfig call.
	orchestrator, err := orchestrator.NewOrchestrator(resticPath, cfg, log, logStore)
	if err != nil {
		zap.S().Fatalf("error creating orchestrator: %v", err)
	}

	wg.Add(1)
	go func() {
		orchestrator.Run(ctx)
		wg.Done()
	}()

	// Create and serve the HTTP gateway
	syncMgr := syncapi.NewSyncManager(configMgr, log, orchestrator, filepath.Join(env.DataDir(), "sync"))
	wg.Add(1)
	go func() {
		syncMgr.RunSync(ctx)
		wg.Done()
	}()

	syncHandler := syncapi.NewBackrestSyncHandler(syncMgr)

	apiBackrestHandler := api.NewBackrestHandler(
		configMgr,
		orchestrator,
		log,
		logStore,
	)
	authenticator := auth.NewAuthenticator(getSecret(), configMgr)
	apiAuthenticationHandler := api.NewAuthenticationHandler(authenticator)

	mux := http.NewServeMux()
	mux.Handle(v1connect.NewAuthenticationHandler(apiAuthenticationHandler))
	mux.Handle(v1connect.NewBackrestSyncServiceHandler(syncHandler))
	backrestHandlerPath, backrestHandler := v1connect.NewBackrestHandler(apiBackrestHandler)
	mux.Handle(backrestHandlerPath, auth.RequireAuthentication(backrestHandler, authenticator))
	mux.Handle("/", webui.Handler())
	mux.Handle("/download/", http.StripPrefix("/download", api.NewDownloadHandler(log)))
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

	debugLevel := zapcore.InfoLevel
	if version == "unknown" { // dev build
		debugLevel = zapcore.DebugLevel
	}
	pretty := zapcore.NewCore(
		zapcore.NewConsoleEncoder(c),
		zapcore.AddSync(colorable.NewColorableStdout()),
		debugLevel,
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
	zap.S().Infof("backrest version %v@%v, using log directory: %v", version, commit, logsDir)
}

// migrateBboltOplog migrates the old bbolt oplog to the new sqlite oplog.
// It is careful to ensure that all migrations are applied before copying
// operations directly to the sqlite logstore.
func migrateBboltOplog(logstore oplog.OpStore) {
	oldBboltOplogFile := path.Join(env.DataDir(), "oplog.boltdb")
	if _, err := os.Stat(oldBboltOplogFile); err != nil {
		return
	}

	zap.S().Warnf("found old bbolt oplog file %q, migrating to sqlite", oldBboltOplogFile)
	oldOpstore, err := bboltstore.NewBboltStore(oldBboltOplogFile)
	if err != nil {
		zap.S().Fatalf("error opening old bolt opstore: %v", oldBboltOplogFile, err)
	}
	oldOplog, err := oplog.NewOpLog(oldOpstore)
	if err != nil {
		zap.S().Fatalf("error opening old bolt oplog: %v", oldBboltOplogFile, err)
	}

	var errs []error
	var count int
	if err := oldOplog.Query(oplog.Query{}, func(op *v1.Operation) error {
		if err := logstore.Add(op); err != nil {
			errs = append(errs, err)
			zap.L().Warn("failed to migrate operation", zap.Error(err), zap.Any("operation", op))
		} else {
			count++
		}
		return nil
	}); err != nil {
		zap.S().Warnf("couldn't migrate all operations from the old bbolt oplog, if this recurs delete the file %q and restart", oldBboltOplogFile)
		zap.S().Fatalf("error migrating old bbolt oplog: %v", err)
	}

	if len(errs) > 0 {
		zap.S().Errorf("encountered %d errors migrating old bbolt oplog, see logs for details.", len(errs), oldBboltOplogFile)
	}
	if err := oldOpstore.Close(); err != nil {
		zap.S().Warnf("error closing old bbolt oplog: %v", err)
	}
	if err := os.Rename(oldBboltOplogFile, oldBboltOplogFile+".deprecated"); err != nil {
		zap.S().Warnf("error removing old bbolt oplog: %v", err)
	}
	zap.S().Infof("migrated %d operations from old bbolt oplog to sqlite", count)
}
