package config

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

// UISettings stores window choices, positions, and camera zoom.
type UISettings struct {
	InventoryOpen            bool
	EquipmentOpen            bool
	StatsOpen                bool
	SkillsOpen               bool
	QuestsOpen               bool
	FriendsOpen              bool
	GuildOpen                bool
	CartOpen                 bool
	WorldMapOpen             bool
	EmotesOpen               bool
	ChatShortcutsOpen        bool
	SettingsOpen             bool
	PetOpen                  bool
	HomunculusOpen           bool
	HomunculusSkillsOpen     bool
	MercenaryOpen            bool
	MercenarySkillsOpen      bool
	MinimapHidden            bool
	BasicInfoCompact         bool
	BasicMenuCollapsed       bool
	SkillTree                bool
	ShortcutRows             int
	ShortcutPosition         WindowPosition
	ConsolePosition          WindowPosition
	StatsPosition            WindowPosition
	InventoryPosition        WindowPosition
	EquipmentPosition        WindowPosition
	SkillsPosition           WindowPosition
	QuestsPosition           WindowPosition
	FriendsPosition          WindowPosition
	GuildPosition            WindowPosition
	CartPosition             WindowPosition
	WorldMapPosition         WindowPosition
	EmotesPosition           WindowPosition
	ChatShortcutsPosition    WindowPosition
	SettingsPosition         WindowPosition
	PetPosition              WindowPosition
	HomunculusPosition       WindowPosition
	HomunculusSkillsPosition WindowPosition
	MercenaryPosition        WindowPosition
	MercenarySkillsPosition  WindowPosition
	BasicInfoPosition        WindowPosition
	MinimapPosition          WindowPosition
	CameraZoom               float64
}

// An unset position keeps the window's automatic placement.
type WindowPosition struct {
	X, Y int
	Set  bool
}

func (p *WindowPosition) parse(value string) error {
	if value == "" {
		*p = WindowPosition{}
		return nil
	}
	x, y, ok := strings.Cut(value, ",")
	xValue, xErr := strconv.Atoi(strings.TrimSpace(x))
	yValue, yErr := strconv.Atoi(strings.TrimSpace(y))
	if !ok || xErr != nil || yErr != nil {
		return fmt.Errorf("window position must be x,y or empty for automatic placement")
	}
	*p = WindowPosition{X: xValue, Y: yValue, Set: true}
	return nil
}

func (p WindowPosition) iniValue() string {
	if !p.Set {
		return ""
	}
	return strconv.Itoa(p.X) + "," + strconv.Itoa(p.Y)
}

func (s *UISettings) boolFields() map[string]*bool {
	return map[string]*bool{
		"inventory_open":         &s.InventoryOpen,
		"equipment_open":         &s.EquipmentOpen,
		"stats_open":             &s.StatsOpen,
		"skills_open":            &s.SkillsOpen,
		"quests_open":            &s.QuestsOpen,
		"friends_open":           &s.FriendsOpen,
		"guild_open":             &s.GuildOpen,
		"cart_open":              &s.CartOpen,
		"world_map_open":         &s.WorldMapOpen,
		"emotes_open":            &s.EmotesOpen,
		"chat_shortcuts_open":    &s.ChatShortcutsOpen,
		"settings_open":          &s.SettingsOpen,
		"pet_open":               &s.PetOpen,
		"homunculus_open":        &s.HomunculusOpen,
		"homunculus_skills_open": &s.HomunculusSkillsOpen,
		"mercenary_open":         &s.MercenaryOpen,
		"mercenary_skills_open":  &s.MercenarySkillsOpen,
		"minimap_hidden":         &s.MinimapHidden,
		"basic_info_compact":     &s.BasicInfoCompact,
		"basic_menu_collapsed":   &s.BasicMenuCollapsed,
		"skill_tree":             &s.SkillTree,
	}
}

func (s *UISettings) positionFields() map[string]*WindowPosition {
	return map[string]*WindowPosition{
		"shortcut_position":          &s.ShortcutPosition,
		"console_position":           &s.ConsolePosition,
		"stats_position":             &s.StatsPosition,
		"inventory_position":         &s.InventoryPosition,
		"equipment_position":         &s.EquipmentPosition,
		"skills_position":            &s.SkillsPosition,
		"quests_position":            &s.QuestsPosition,
		"friends_position":           &s.FriendsPosition,
		"guild_position":             &s.GuildPosition,
		"cart_position":              &s.CartPosition,
		"world_map_position":         &s.WorldMapPosition,
		"emotes_position":            &s.EmotesPosition,
		"chat_shortcuts_position":    &s.ChatShortcutsPosition,
		"settings_position":          &s.SettingsPosition,
		"pet_position":               &s.PetPosition,
		"homunculus_position":        &s.HomunculusPosition,
		"homunculus_skills_position": &s.HomunculusSkillsPosition,
		"mercenary_position":         &s.MercenaryPosition,
		"mercenary_skills_position":  &s.MercenarySkillsPosition,
		"basic_info_position":        &s.BasicInfoPosition,
		"minimap_position":           &s.MinimapPosition,
	}
}

func (s *UISettings) apply(key, value string) error {
	switch key {
	case "camerazoom":
		zoom, err := strconv.ParseFloat(value, 64)
		if err != nil || zoom < 0 || math.IsNaN(zoom) || math.IsInf(zoom, 0) {
			return fmt.Errorf("camera_zoom must be finite and non-negative (0 uses the default)")
		}
		s.CameraZoom = zoom
		return nil
	}
	if key == "shortcutrows" {
		rows, err := strconv.Atoi(value)
		if err != nil || rows < 1 || rows > 3 {
			return fmt.Errorf("shortcut_rows must be between 1 and 3")
		}
		s.ShortcutRows = rows
		return nil
	}
	for name, field := range s.boolFields() {
		if normalizeKey(name) == key {
			return setBool(value, field)
		}
	}
	for name, field := range s.positionFields() {
		if normalizeKey(name) == key {
			return field.parse(value)
		}
	}
	return fmt.Errorf("unknown UI setting %q", key)
}

func (cfg Config) LoadUISettings() (UISettings, error) {
	if cfg.ConfigPath == "" {
		return cfg.UI, nil
	}
	latest := defaultConfig()
	latest.UI = cfg.UI
	if err := applyINIFile(&latest, cfg.ConfigPath, false); err != nil {
		return cfg.UI, err
	}
	return latest.UI, nil
}

func (cfg Config) SaveUISettings(settings UISettings) (string, error) {
	// Zero is the default for configurations constructed without an INI file.
	if settings.ShortcutRows == 0 {
		settings.ShortcutRows = 1
	}
	if settings.ShortcutRows < 1 || settings.ShortcutRows > 3 {
		return "", fmt.Errorf("shortcut_rows must be between 1 and 3")
	}
	if settings.CameraZoom < 0 || math.IsNaN(settings.CameraZoom) || math.IsInf(settings.CameraZoom, 0) {
		return "", fmt.Errorf("camera_zoom must be finite and non-negative")
	}
	values := map[string]string{
		"shortcut_rows": strconv.Itoa(settings.ShortcutRows),
		"camera_zoom":   strconv.FormatFloat(settings.CameraZoom, 'f', -1, 64),
	}
	for key, value := range settings.boolFields() {
		values[key] = formatINIValueBool(*value)
	}
	for key, position := range settings.positionFields() {
		values[key] = position.iniValue()
	}
	return cfg.saveConfigValues(map[string]map[string]string{"ui": values})
}
