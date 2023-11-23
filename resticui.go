package main

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"os/signal"
	"path"
	"sync"
	"syscall"

	"github.com/garethgeorge/resticui/internal/api"
	"github.com/garethgeorge/resticui/internal/config"
	"github.com/garethgeorge/resticui/internal/database/oplog"
	"github.com/garethgeorge/resticui/internal/orchestrator"
	static "github.com/garethgeorge/resticui/webui"
	"github.com/mattn/go-colorable"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	_ "embed"
)

func main() {
	port := os.Getenv("RESTICUI_PORT")
	if port == "" {
		port = "9898"
	}

	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	go onterm(cancel)
	
	if _, err := config.Default.Get(); err != nil {
		zap.S().Fatalf("Error loading config: %v", err)
	}
	
	var wg sync.WaitGroup

	// Configure the HTTP mux
	mux := http.NewServeMux()
	mux.Handle("/", http.FileServer(http.FS(&SubdirFilesystem{FS: static.FS, subdir: "dist"})))


	// Create and serve API server
	oplogFile := path.Join(dataPath(), "oplog.boltdb")
	oplog, err := oplog.NewOpLog(oplogFile)
	if err != nil {
		zap.S().Warnf("Operation log may be corrupted, if errors recur delete the file %q and restart. Your backups stored in your repos are safe.", oplogFile)
		zap.S().Fatalf("Error creating oplog : %v", err)
	}
	defer oplog.Close()

	orchestrator, err := orchestrator.NewOrchestrator(config.Default, oplog)
	if err != nil {
		zap.S().Fatalf("Error creating orchestrator: %v", err)
	}

	// Start orchestration loop.
	go func() {
		err := orchestrator.Run(ctx)
		if err != nil && !errors.Is(err, context.Canceled) {
			zap.S().Fatal("Orchestrator loop exited with error: ", zap.Error(err))
			cancel() // cancel the context when the orchestrator exits (e.g. on fatal error)
		}
	}()

	apiServer := api.NewServer(
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
		Addr:    fmt.Sprintf(":%s", port),
		Handler: mux,
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		zap.S().Infof("HTTP binding to address %v", server.Addr)
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
	if os.Getenv("DEBUG") != "" {
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

func onterm(callback func()) {
	sigchan := make(chan os.Signal, 1)
	signal.Notify(sigchan, os.Interrupt, syscall.SIGTERM)
	<-sigchan
	callback()
}

type SubdirFilesystem struct {
	fs.FS
	subdir string
}

var _ fs.FS = &SubdirFilesystem{}
var _ fs.ReadDirFS = &SubdirFilesystem{}

func (s *SubdirFilesystem) Open(name string) (fs.File, error) {
	return s.FS.Open(path.Join(s.subdir, name))
}

func (s *SubdirFilesystem) 	ReadDir(name string) ([]fs.DirEntry, error) {
	readDirFS := s.FS.(fs.ReadDirFS)
	if readDirFS == nil {
		return nil, &fs.PathError{Op: "readdir", Path: name, Err: errors.New("not implemented")}
	}
	return readDirFS.ReadDir(path.Join(s.subdir, name))
}

func dataPath() string {
	datahome := os.Getenv("XDG_DATA_HOME")
	if datahome == "" {
		datahome = path.Join(os.Getenv("HOME") + "/.local/share")
	}
	return path.Join(datahome, "resticui")
}