//go:build !js || !wasm

package config

import (
	"os"
	"path/filepath"
)

// applyStoredUserINI is a no-op on native builds: the user ini is already
// applied from goro.ini in LoadConfig.
func applyStoredUserINI(cfg *Config) {}

// writeUserSettings persists the settings ini to the user config file on
// native builds.
func writeUserSettings(values map[string]map[string]string) (string, error) {
	path, err := UserConfigPath()
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", err
	}
	existing, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return "", err
	}
	data := upsertINIValues(string(existing), values)
	if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
		return "", err
	}
	return path, nil
}
