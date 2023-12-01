package config

import (
	"fmt"
	"os"
	"path"
	"strings"
)

var (
	EnvVarConfigPath  = "RESTICUI_CONFIG_PATH"
	EnvVarDataDir     = "RESTICUI_DATA_DIR"
	EnvVarBindAddress = "RESTICUI_PORT"
	EnvVarBinPath     = "RESTICUI_RESTIC_BIN_PATH"
)

func ConfigFilePath() string {
	if val := os.Getenv(EnvVarConfigPath); val != "" {
		return val
	}
	if val := os.Getenv("XDG_CONFIG_HOME"); val != "" {
		return path.Join(val, "resticui/config.json")
	}
	return path.Join(getHomeDir(), ".config/resticui/config.json")
}

func DataDir() string {
	if val := os.Getenv(EnvVarDataDir); val != "" {
		return val
	}
	if val := os.Getenv("XDG_DATA_HOME"); val != "" {
		return path.Join(val, "resticui")
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
	return "127.0.0.1:9898"
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
