package envopts

import (
	"fmt"
	"os"
	"path"
	"strings"
)

func ConfigFilePath() string {
	if val := os.Getenv("RESTICUI_CONFIG_PATH"); val != "" {
		return val
	}
	if val := os.Getenv("XDG_CONFIG_HOME"); val != "" {
		return path.Join(val, "resticui/config.json")
	}
	return path.Join(getHomeDir(), ".config/resticui/config.json")
}

func DataDir() string {
	if val := os.Getenv("RESTICUI_DATA_DIR"); val != "" {
		return val
	}
	if val := os.Getenv("XDG_DATA_HOME"); val != "" {
		return path.Join(val, "resticui")
	}
	return path.Join(getHomeDir(), ".local/share/resticui")
}

func BindAddress() string {
	if val := os.Getenv("RESTICUI_PORT"); val != "" {
		if !strings.Contains(val, ":") {
			return ":" + val
		}
		return val
	}
	return ":9898"
}

func getHomeDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		panic(fmt.Errorf("couldn't determine home directory: %v", err))
	}
	return home
}
