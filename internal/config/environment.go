package config

import (
	"fmt"
	"os"
	"path"
	"runtime"
	"strings"
)

var (
	EnvVarConfigPath  = "RESTICUI_CONFIG_PATH"
	EnvVarDataDir     = "RESTICUI_DATA_DIR"
	EnvVarBindAddress = "RESTICUI_PORT"
	EnvVarBinPath     = "RESTICUI_RESTIC_BIN_PATH"
)

// ConfigFilePath
// - *nix systems use $XDG_CONFIG_HOME/resticui/config.json
// - windows uses %APPDATA%/resticui/config.json
func ConfigFilePath() string {
	if val := os.Getenv(EnvVarConfigPath); val != "" {
		return val
	}
	return path.Join(getConfigDir(), "resticui/config.json")
}

// DataDir
// - *nix systems use $XDG_DATA_HOME/resticui
// - windows uses %APPDATA%/resticui/data
func DataDir() string {
	if val := os.Getenv(EnvVarDataDir); val != "" {
		return val
	}
	if val := os.Getenv("XDG_DATA_HOME"); val != "" {
		return path.Join(val, "resticui")
	}

	if runtime.GOOS == "windows" {
		return path.Join(getConfigDir(), "resticui/data")
	}
	return path.Join(getHomeDir(), ".local/share/resticui")
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
	if val := os.Getenv("RESTICUI_RESTIC_BIN_PATH"); val != "" {
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
