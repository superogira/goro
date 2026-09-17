package config

import (
	"fmt"
	"strconv"
	"strings"
)

// ChatShortcuts are client-side Alt+1 through Alt+0 commands, independent of
// the server's item and skill hotkey slots.
type ChatShortcuts [10]string

func defaultChatShortcuts() ChatShortcuts {
	return ChatShortcuts{"/!", "/?", "/ho", "/lv", "/swt", "/ic", "/an", "/ag", "/$", "/..."}
}

// LoadChatShortcuts refreshes saved bindings when entering the game again;
// the startup Config may predate edits made before character selection.
func (cfg Config) LoadChatShortcuts() (ChatShortcuts, error) {
	if cfg.ConfigPath == "" {
		return cfg.ChatShortcuts, nil
	}
	latest := defaultConfig()
	latest.ChatShortcuts = cfg.ChatShortcuts
	if err := applyINIFile(&latest, cfg.ConfigPath, false); err != nil {
		return cfg.ChatShortcuts, err
	}
	return latest.ChatShortcuts, nil
}

func (cfg Config) SaveChatShortcuts(commands ChatShortcuts) (string, error) {
	values := make(map[string]string, len(commands))
	for slot, command := range commands {
		if strings.ContainsAny(command, "\r\n\x00") {
			return "", fmt.Errorf("shortcut must be a single line without NUL characters")
		}
		values[strconv.Itoa((slot+1)%10)] = `"` + command + `"`
	}
	return cfg.saveConfigValues(map[string]map[string]string{"chatshortcuts": values})
}
