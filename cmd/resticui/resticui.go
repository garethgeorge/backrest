package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/garethgeorge/resticui/internal/api"
	"github.com/garethgeorge/resticui/static"
	"go.uber.org/zap"

	_ "embed"
)

func main() {
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	go onterm(cancel)
	
	var wg sync.WaitGroup

	// Configure the HTTP mux
	mux := http.NewServeMux()
	mux.Handle("/", http.FileServer(http.FS(static.FS)))

	// Serve the API
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer cancel()
		err := api.ServeAPI(ctx, mux)
		if err != nil {
			zap.S().Fatal("Error serving API", zap.Error(err))
		}
		cancel() // cancel the context when the API server exits (e.g. on fatal error)
	}()

	server := &http.Server{
		Addr:    ":9090",
		Handler: mux,
	}

	// Serve the HTTP gateway
	wg.Add(1)
	go func() {
		defer wg.Done()
		zap.S().Infof("HTTP binding to address %v", server.Addr)
		go func() {
			<-ctx.Done()
			server.Shutdown(context.Background())
		}()
		if err := server.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
			zap.S().Error("Error starting server", zap.Error(err))
		}
		zap.S().Info("HTTP gateway shutdown")
		cancel() // cancel the context when the HTTP server exits (e.g. on fatal error)
	}()

	wg.Wait()
}

func init() {
	zap.ReplaceGlobals(zap.Must(zap.NewProduction()))
	if os.Getenv("DEBUG") != "" {
		zap.ReplaceGlobals(zap.Must(zap.NewDevelopment()))
	}
}

func onterm(callback func()) {
	sigchan := make(chan os.Signal, 1)
	signal.Notify(sigchan, os.Interrupt, syscall.SIGTERM)
	<-sigchan
	callback()
}