package settings

import (
	"os"
	"path/filepath"
)

// ConfigPath resolves the settings file: $MHTODO_CONFIG_PATH first, then
// $XDG_CONFIG_HOME/mhtodo/config.yml (default ~/.config/mhtodo/config.yml).
func ConfigPath() string {
	if p := os.Getenv("MHTODO_CONFIG_PATH"); p != "" {
		return p
	}
	base := os.Getenv("XDG_CONFIG_HOME")
	if base == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			base = ".config"
		} else {
			base = filepath.Join(home, ".config")
		}
	}
	return filepath.Join(base, "mhtodo", "config.yml")
}
