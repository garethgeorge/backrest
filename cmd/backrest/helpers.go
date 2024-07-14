package main

import (
	"crypto/rand"
	"os"
	"os/signal"
	"path"
	"runtime"
	"sync/atomic"
	"syscall"

	"github.com/garethgeorge/backrest/internal/env"
	"go.uber.org/zap"
)

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
