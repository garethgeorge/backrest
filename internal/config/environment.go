package config

import (
	"fmt"
	"os"
	"path"
	"runtime"
	"strings"
)

var (
	EnvVarConfigPath  = "BACKREST_CONFIG"         // path to config file
	EnvVarDataDir     = "BACKREST_DATA"           // path to data directory
	EnvVarBindAddress = "BACKREST_PORT"           // port to bind to (default 9898)
	EnvVarBinPath     = "BACKREST_RESTIC_COMMAND" // path to restic binary (default restic)
)

// ConfigFilePath
// - *nix systems use $XDG_CONFIG_HOME/backrest/config.json
// - windows uses %APPDATA%/backrest/config.json
func ConfigFilePath() string {
	if val := os.Getenv(EnvVarConfigPath); val != "" {
		return val
	}
	return path.Join(getConfigDir(), "backrest/config.json")
}

// DataDir
// - *nix systems use $XDG_DATA_HOME/backrest
// - windows uses %APPDATA%/backrest/data
func DataDir() string {
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
	if val := os.Getenv(EnvVarBindAddress); val != "" {
		if !strings.Contains(val, ":") {
			return ":" + val
		}
		return val
	}
	return ":9898"
}

func ResticBinPath() string {
	if val := os.Getenv(EnvVarBinPath); val != "" {
		return val
	}
	return ""
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
