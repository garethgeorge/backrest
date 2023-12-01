package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"path"
	"strings"
	"sync"
	"syscall"

	rice "github.com/GeertJohan/go.rice"
	"github.com/garethgeorge/resticui/internal/api"
	"github.com/garethgeorge/resticui/internal/config"
	"github.com/garethgeorge/resticui/internal/oplog"
	"github.com/garethgeorge/resticui/internal/orchestrator"
	"github.com/garethgeorge/resticui/internal/resticinstaller"
	"github.com/mattn/go-colorable"
	"go.etcd.io/bbolt"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func main() {
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	go onterm(cancel)

	configStore := createConfigProvider()

	cfg, err := configStore.Get()
	if err != nil {
		zap.S().Fatalf("Error loading config: %v", err)
	}

	var wg sync.WaitGroup

	// Configure the HTTP mux
	mux := http.NewServeMux()

	if box, err := rice.FindBox("webui/dist"); err == nil {
		mux.Handle("/", http.FileServer(box.HTTPBox()))
	} else {
		zap.S().Warnf("Error loading static assets, not serving UI: %v", err)
	}

	// Create and serve API server
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

	resticPath, err := resticinstaller.FindOrInstallResticBinary()
	if err != nil {
		zap.S().Fatalf("Error finding or installing restic: %v", err)
	}

	orchestrator, err := orchestrator.NewOrchestrator(resticPath, cfg, oplog)
	if err != nil {
		zap.S().Fatalf("Error creating orchestrator: %v", err)
	}

	// Start orchestration loop. Only exits when ctx is cancelled.
	go orchestrator.Run(ctx)

	apiServer := api.NewServer(
		configStore,
		orchestrator, // TODO: eliminate default config
		oplog,
	)

	// Serve the API
	wg.Add(1)
	go func() {
		defer wg.Done()
		err := api.ServeAPI(ctx, apiServer, mux)
		if err != nil {
			zap.S().Fatal("Error serving API", zap.Error(err))
		}
		cancel() // cancel the context when the API server exits (e.g. on fatal error)
	}()

	// Serve the HTTP gateway
	server := &http.Server{
		Addr:    config.BindAddress(),
		Handler: mux,
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		zap.S().Infof("Starting web server %v", server.Addr)
		go func() {
			<-ctx.Done()
			server.Shutdown(context.Background())
		}()
		if err := server.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
			zap.L().Error("Error starting server", zap.Error(err))
		}
		zap.L().Info("HTTP gateway shutdown")
		cancel() // cancel the context when the HTTP server exits (e.g. on fatal error)
	}()

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

func findResticBin() {
	resticBin := config.ResticBinPath()
	if resticBin != "" {

	}
}
