package env

import (
	"crypto/sha256"
	"encoding/hex"
	"flag"
	"fmt"
	"os"
	"path"
	"runtime"
	"strings"

	"go.uber.org/zap"
)

var (
	EnvVarConfigPath  = "BACKREST_CONFIG"         // path to config file
	EnvVarDataDir     = "BACKREST_DATA"           // path to data directory
	EnvVarBindAddress = "BACKREST_PORT"           // port to bind to (default 9898)
	EnvVarBinPath     = "BACKREST_RESTIC_COMMAND" // path to restic binary (default restic)
)

var flagDataDir = flag.String("data-dir", "", "path to data directory, defaults to XDG_DATA_HOME/.local/backrest. Overrides BACKREST_DATA environment variable.")
var flagConfigPath = flag.String("config-file", "", "path to config file, defaults to XDG_CONFIG_HOME/backrest/config.json. Overrides BACKREST_CONFIG environment variable.")
var flagBindAddress = flag.String("bind-address", "", "address to bind to, defaults to 127.0.0.1:9898. Use :9898 to listen on all interfaces. Overrides BACKREST_PORT environment variable.")
var flagResticBinPath = flag.String("restic-cmd", "", "path to restic binary, defaults to a backrest managed version of restic. Overrides BACKREST_RESTIC_COMMAND environment variable.")
var hubConnectAddr = flag.String("connect-to-hub", "", "hub address to connect to")
var hubServeAddr = flag.String("serve-hub-address", "", "address to serve the hub on")

func ValidateEnvironment() {
	if IsHubServer() && IsHubClient() {
		zap.L().Error("hub server and client cannot be enabled at the same time")
		os.Exit(1)
	}

	if IsHubClient() && (*flagConfigPath != "" || os.Getenv(EnvVarConfigPath) != "") {
		zap.L().Error("cannot customize config path when in daemon mode, config is cached in data directory.")
		os.Exit(1)
	}
}

// ConfigFilePath
// - *nix systems use $XDG_CONFIG_HOME/backrest/config.json
// - windows uses %APPDATA%/backrest/config.json
// - hub clients use a path in the data directory
func ConfigFilePath() string {
	if IsHubClient() {
		sum := sha256.Sum256([]byte(HubConnectAddr()))
		return path.Join(DataDir(), "daemon-config-cache", fmt.Sprintf("%v-config.json", hex.EncodeToString(sum[:])))
	}
	if *flagConfigPath != "" {
		return *flagConfigPath
	}
	if val := os.Getenv(EnvVarConfigPath); val != "" {
		return val
	}
	return path.Join(getConfigDir(), "backrest/config.json")
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
		return path.Join(getConfigDir(), "backrest/data")
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

func IsHubServer() bool {
	return *hubServeAddr != ""
}

func HubServeAddr() string {
	return *hubServeAddr
}

func IsHubClient() bool {
	return *hubConnectAddr != ""
}

func HubConnectAddr() string {
	return *hubConnectAddr
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
	return path.Join(getHomeDir(), ".config")
}

func formatBindAddress(addr string) string {
	if !strings.Contains(addr, ":") {
		return ":" + addr
	}
	return addr
}
