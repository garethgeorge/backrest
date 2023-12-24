package main

import (
	"context"
	"errors"
	"flag"
	"net/http"
	"os"
	"os/signal"
	"path"
	"strings"
	"sync"
	"syscall"

	rice "github.com/GeertJohan/go.rice"
	"github.com/garethgeorge/backrest/gen/go/v1/v1connect"
	"github.com/garethgeorge/backrest/internal/api"
	"github.com/garethgeorge/backrest/internal/config"
	"github.com/garethgeorge/backrest/internal/oplog"
	"github.com/garethgeorge/backrest/internal/orchestrator"
	"github.com/garethgeorge/backrest/internal/resticinstaller"
	"github.com/mattn/go-colorable"
	"go.etcd.io/bbolt"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
)

func main() {
	flag.Parse()
	ctx, cancel := context.WithCancel(context.Background())
	go onterm(cancel)

	resticPath, err := resticinstaller.FindOrInstallResticBinary()
	if err != nil {
		zap.S().Fatalf("Error finding or installing restic: %v", err)
	}

	// Load the configuration
	configStore := createConfigProvider()
	cfg, err := configStore.Get()
	if err != nil {
		zap.S().Fatalf("Error loading config: %v", err)
	}

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

	// Create orchestrator and start task loop.
	orchestrator, err := orchestrator.NewOrchestrator(resticPath, cfg, oplog)
	if err != nil {
		zap.S().Fatalf("Error creating orchestrator: %v", err)
	}

	wg.Add(1)
	go func() {
		orchestrator.Run(ctx)
		wg.Done()
	}()

	// Create and serve the HTTP gateway
	apiServer := api.NewServer(
		configStore,
		orchestrator,
		oplog,
	)

	mux := http.NewServeMux()

	if box, err := rice.FindBox("webui/dist"); err == nil {
		mux.Handle("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if strings.HasSuffix(r.URL.Path, "/") {
				r.URL.Path += "index.html"
			}
			f, err := box.Open(r.URL.Path + ".gz")
			if err == nil {
				defer f.Close()
				w.Header().Set("Content-Encoding", "gzip")
				http.ServeContent(w, r, r.URL.Path, box.Time(), f)
				return
			}
			f, err = box.Open(r.URL.Path)
			if err == nil {
				defer f.Close()
				http.ServeContent(w, r, r.URL.Path, box.Time(), f)
				return
			}
			http.Error(w, "Not found", http.StatusNotFound)
		}))
	} else {
		zap.S().Warnf("Error loading static assets, not serving UI: %v", err)
	}

	mux.Handle(v1connect.NewBackrestHandler(apiServer))

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
