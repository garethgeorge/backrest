package config

import (
	"fmt"
	"os"
	"path"
	"runtime"
	"strings"
)

var (
	EnvVarConfigPath  = "RESTORA_CONFIG_PATH"
	EnvVarDataDir     = "RESTORA_DATA_DIR"
	EnvVarBindAddress = "RESTORA_PORT"
	EnvVarBinPath     = "RESTORA_RESTIC_BIN_PATH"
)

// ConfigFilePath
// - *nix systems use $XDG_CONFIG_HOME/restora/config.json
// - windows uses %APPDATA%/restora/config.json
func ConfigFilePath() string {
	if val := os.Getenv(EnvVarConfigPath); val != "" {
		return val
	}
	return path.Join(getConfigDir(), "restora/config.json")
}

// DataDir
// - *nix systems use $XDG_DATA_HOME/restora
// - windows uses %APPDATA%/restora/data
func DataDir() string {
	if val := os.Getenv(EnvVarDataDir); val != "" {
		return val
	}
	if val := os.Getenv("XDG_DATA_HOME"); val != "" {
		return path.Join(val, "restora")
	}

	if runtime.GOOS == "windows" {
		return path.Join(getConfigDir(), "restora/data")
	}
	return path.Join(getHomeDir(), ".local/share/restora")
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
	if val := os.Getenv("RESTORA_RESTIC_BIN_PATH"); val != "" {
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
