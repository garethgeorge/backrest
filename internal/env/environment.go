package env

import (
	"flag"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"go.uber.org/zap"
)

var (
	EnvVarConfigPath                 = "BACKREST_CONFIG"                       // path to config file
	EnvVarDataDir                    = "BACKREST_DATA"                         // path to data directory
	EnvVarBindAddress                = "BACKREST_PORT"                         // port to bind to (default 9898)
	EnvVarBinPath                    = "BACKREST_RESTIC_COMMAND"               // path to restic binary (default restic)
	EnvVarMultihostHeartbeatInterval = "BACKREST_MULTIHOST_HEARTBEAT_INTERVAL" // interval for multihost heartbeat messages
)

var flagDataDir = flag.String("data-dir", "", "path to data directory, defaults to XDG_DATA_HOME/.local/backrest. Overrides BACKREST_DATA environment variable.")
var flagConfigPath = flag.String("config-file", "", "path to config file, defaults to XDG_CONFIG_HOME/backrest/config.json. Overrides BACKREST_CONFIG environment variable.")
var flagBindAddress = flag.String("bind-address", "", "address to bind to, defaults to 127.0.0.1:9898. Use :9898 to listen on all interfaces. Overrides BACKREST_PORT environment variable.")
var flagResticBinPath = flag.String("restic-cmd", "", "path to restic binary, defaults to a backrest managed version of restic. Overrides BACKREST_RESTIC_COMMAND environment variable.")
var flagMultihostHeartbeatInterval = flag.Duration("multihost-heartbeat-interval", 600*time.Second, "interval in seconds to send heartbeat messages to other hosts in a multihost setup. Defaults to 600 seconds, but can be set lower to keep connections alive with reverse proxies that aggressively timeout idle connections.")

// ConfigFilePath
// - *nix systems use $XDG_CONFIG_HOME/backrest/config.json
// - windows uses %APPDATA%/backrest/config.json
func ConfigFilePath() string {
	if *flagConfigPath != "" {
		return *flagConfigPath
	}
	if val := os.Getenv(EnvVarConfigPath); val != "" {
		return val
	}
	return filepath.Join(getConfigDir(), "backrest", "config.json")
}

func SSHDir() string {
	return filepath.Join(getHomeDir(), ".ssh")
}

// DataDir
// - *nix systems use $XDG_DATA_HOME/backrest
// - windows uses %APPDATA%/backrest/data
func DataDir() string {
	if *flagDataDir != "" {
		return *flagDataDir
	}
	if val := os.Getenv(EnvVarDataDir); val != "" {
		return val
	}
	if val := os.Getenv("XDG_DATA_HOME"); val != "" {
		return path.Join(val, "backrest")
	}

	if runtime.GOOS == "windows" {
		return filepath.Join(getConfigDir(), "backrest", "data")
	}
	return path.Join(getHomeDir(), ".local/share/backrest")
}

func BindAddress() string {
	if *flagBindAddress != "" {
		return formatBindAddress(*flagBindAddress)
	}
	if val := os.Getenv(EnvVarBindAddress); val != "" {
		return formatBindAddress(val)
	}
	return "127.0.0.1:9898"
}

func ResticBinPath() string {
	if *flagResticBinPath != "" {
		return *flagResticBinPath
	}
	if val := os.Getenv(EnvVarBinPath); val != "" {
		return val
	}
	return ""
}

func MultihostHeartbeatInterval() time.Duration {
	if *flagMultihostHeartbeatInterval != 0 {
		return *flagMultihostHeartbeatInterval
	}
	if val := os.Getenv(EnvVarMultihostHeartbeatInterval); val != "" {
		if d, err := time.ParseDuration(val); err == nil {
			return d
		} else {
			zap.S().Warnf("Invalid duration for %s: %s, using default 600 seconds", EnvVarMultihostHeartbeatInterval, val)
		}
	}
	return 600 * time.Second // Default to 10 minutes.
}

func LogsPath() string {
	dataDir := DataDir()
	return filepath.Join(dataDir, "processlogs")
}

func getHomeDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		panic(fmt.Errorf("couldn't determine home directory: %v", err))
	}
	return home
}

func getConfigDir() string {
	if runtime.GOOS == "windows" {
		cfgDir, err := os.UserConfigDir()
		if err != nil {
			panic(fmt.Errorf("couldn't determine config directory: %v", err))
		}
		return cfgDir
	}
	if val := os.Getenv("XDG_CONFIG_HOME"); val != "" {
		return val
	}
	return filepath.Join(getHomeDir(), ".config")
}

func formatBindAddress(addr string) string {
	if !strings.Contains(addr, ":") {
		return ":" + addr
	}
	return addr
}
