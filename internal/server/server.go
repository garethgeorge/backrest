package server

import (
	"github.com/garethgeorge/backrest/internal/config"
	"github.com/garethgeorge/backrest/internal/oplog"
	"github.com/garethgeorge/backrest/internal/orchestrator"
	"github.com/garethgeorge/backrest/internal/rotatinglog"
)

type Server struct {
	isDaemon bool
	isHub    bool

	config       *config.ConfigManager
	oplog        *oplog.OpLog
	orchestrator *orchestrator.Orchestrator
	logstore     *rotatinglog.RotatingLog
}

func NewHubServer(config *config.ConfigManager, oplog *oplog.OpLog, orchestrator *orchestrator.Orchestrator, logstore *rotatinglog.RotatingLog) *Server {
	return &Server{
		isHub:        true,
		config:       config,
		oplog:        oplog,
		orchestrator: orchestrator,
		logstore:     logstore,
	}
}

func NewDaemonServer(config *config.ConfigManager, oplog *oplog.OpLog, orchestrator *orchestrator.Orchestrator, logstore *rotatinglog.RotatingLog) *Server {
	return &Server{
		isDaemon:     true,
		config:       config,
		oplog:        oplog,
		orchestrator: orchestrator,
		logstore:     logstore,
	}
}
