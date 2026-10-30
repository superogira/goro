package game

import (
	"time"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/config"
	"github.com/kivutar/goro/game/scene"
	"github.com/kivutar/goro/glog"
)

type windowPreferences struct {
	loaded      bool
	last        config.UISettings
	dirty       bool
	pending     config.UISettings
	zoomTarget  float64
	zoomChanged time.Time
}

type windowPositionBinding struct {
	window interface {
		SavedPosition() config.WindowPosition
		RestorePosition(client.Context, config.WindowPosition)
		Dragging() bool
	}
	position *config.WindowPosition
}

func (u *worldUI) windowPositions(s *config.UISettings) []windowPositionBinding {
	return []windowPositionBinding{
		{&u.console, &s.ConsolePosition},
		{&u.shortcutBar, &s.ShortcutPosition},
		{&u.statsWindow, &s.StatsPosition},
		{&u.inventoryBag, &s.InventoryPosition},
		{&u.equipmentWindow, &s.EquipmentPosition},
		{&u.skillWindow, &s.SkillsPosition},
		{&u.questWindow, &s.QuestsPosition},
		{&u.social.FriendsWindow, &s.FriendsPosition},
		{&u.social.GuildWindow, &s.GuildPosition},
		{&u.cartWindow, &s.CartPosition},
		{&u.worldMap, &s.WorldMapPosition},
		{&u.emoteWindow, &s.EmotesPosition},
		{&u.chatShortcuts, &s.ChatShortcutsPosition},
		{&u.settingsWindow, &s.SettingsPosition},
		{&u.pet.Info, &s.PetPosition},
		{&u.companions.HomunculusInfo, &s.HomunculusPosition},
		{&u.companions.HomunculusSkill, &s.HomunculusSkillsPosition},
		{&u.companions.MercenaryInfo, &s.MercenaryPosition},
		{&u.companions.MercenarySkill, &s.MercenarySkillsPosition},
	}
}

func (m *WorldMode) restoreUISettings(ctx client.Context) {
	if ctx.Session == nil {
		return
	}
	p := &m.ui.preferences
	if !p.loaded {
		p.loaded = true
		s, err := ctx.Config.LoadUISettings()
		if err != nil {
			glog.Warnf("load UI settings: %v", err)
		}
		p.pending = s
		if s.CameraZoom > 0 {
			m.camera.zoomTarget = scene.ClampCameraZoom(s.CameraZoom)
			m.camera.zoom = m.camera.zoomTarget
		}
		m.ui.skillWindow.SetTreeMode(ctx, s.SkillTree)
		m.ui.shortcutBar.SetVisibleRows(ctx, s.ShortcutRows)
		for _, binding := range m.ui.windowPositions(&s) {
			binding.window.RestorePosition(ctx, *binding.position)
		}
		if m.ui.minimap.Hidden() != s.MinimapHidden {
			m.ui.minimap.Toggle(ctx)
		}
		if s.InventoryOpen && !m.ui.inventoryBag.IsOpen() {
			m.ui.inventoryBag.Toggle(ctx)
		}
		if s.EquipmentOpen && !m.ui.equipmentWindow.IsOpen() {
			m.ui.equipmentWindow.Toggle(ctx)
		}
		if s.StatsOpen {
			m.ui.statsWindow.OpenWindow(ctx)
		}
		if s.SkillsOpen {
			m.ui.skillWindow.OpenWindow(ctx)
			m.ui.skillWindow.Rebind(ctx, m)
		}
		if s.QuestsOpen {
			m.ui.questWindow.OpenWindow(ctx)
		}
		if s.FriendsOpen {
			m.ui.social.FriendsWindow.OpenWindow(ctx)
		}
		if s.EmotesOpen {
			m.ui.emoteWindow.OpenWindow(ctx, &m.ui.console)
		}
		if s.ChatShortcutsOpen && !m.ui.chatShortcuts.IsOpen() {
			m.ui.chatShortcuts.Toggle(ctx, &m.ui.console, func() {
				m.ui.emoteWindow.OpenWindow(ctx, &m.ui.console)
			})
		}
		if s.SettingsOpen {
			m.ui.settingsWindow.OpenWindow(ctx)
		}
		if s.WorldMapOpen {
			if err := m.ui.worldMap.OpenWindow(ctx); err != nil {
				glog.Warnf("restore world map window: %v", err)
			}
		}
		p.last = m.uiSettings()
		p.zoomTarget = p.last.CameraZoom
	}
	// These windows need information that may arrive after entering the map.
	if p.pending.CartOpen {
		if !m.ui.cartWindow.IsOpen() {
			m.ui.cartWindow.Toggle(ctx)
		}
		p.pending.CartOpen = !m.ui.cartWindow.IsOpen()
	}
	if p.pending.GuildOpen && (ctx.Session.Guild.ID != 0 || ctx.Session.GuildID != 0) {
		if !m.ui.social.GuildWindow.IsOpen() {
			m.social().ToggleGuildWindow(ctx)
		}
		p.pending.GuildOpen = false
	}
	if p.pending.PetOpen && m.petSystem.HasProperty {
		if !m.ui.pet.Info.IsOpen() {
			m.pets().OpenInfo(ctx)
		}
		p.pending.PetOpen = false
	}
	if ctx.Session.Homunculus.Active {
		if p.pending.HomunculusOpen && !m.ui.companions.HomunculusInfo.IsOpen() {
			m.ui.companions.HomunculusInfo.OpenInfo(ctx, ctx.Session.Homunculus)
		}
		if p.pending.HomunculusSkillsOpen && !m.ui.companions.HomunculusSkill.IsOpen() {
			m.ui.companions.HomunculusSkill.Open(ctx, m)
		}
		p.pending.HomunculusOpen, p.pending.HomunculusSkillsOpen = false, false
	}
	if ctx.Session.Mercenary.Active {
		if p.pending.MercenaryOpen && !m.ui.companions.MercenaryInfo.IsOpen() {
			m.ui.companions.MercenaryInfo.OpenInfo(ctx, ctx.Session.Mercenary)
		}
		if p.pending.MercenarySkillsOpen && !m.ui.companions.MercenarySkill.IsOpen() {
			m.ui.companions.MercenarySkill.Open(ctx, m)
		}
		p.pending.MercenaryOpen, p.pending.MercenarySkillsOpen = false, false
	}
}

func (m *WorldMode) uiSettings() config.UISettings {
	u := &m.ui
	p := u.preferences.pending
	settings := config.UISettings{
		InventoryOpen:        u.inventoryBag.IsOpen(),
		EquipmentOpen:        u.equipmentWindow.IsOpen(),
		StatsOpen:            u.statsWindow.IsOpen(),
		SkillsOpen:           u.skillWindow.IsOpen(),
		QuestsOpen:           u.questWindow.IsOpen(),
		FriendsOpen:          u.social.FriendsWindow.IsOpen(),
		GuildOpen:            u.social.GuildWindow.IsOpen() || p.GuildOpen,
		CartOpen:             u.cartWindow.IsOpen() || p.CartOpen,
		WorldMapOpen:         u.worldMap.IsOpen(),
		EmotesOpen:           u.emoteWindow.IsOpen(),
		ChatShortcutsOpen:    u.chatShortcuts.IsOpen(),
		SettingsOpen:         u.settingsWindow.IsOpen(),
		PetOpen:              u.pet.Info.IsOpen() || p.PetOpen,
		HomunculusOpen:       u.companions.HomunculusInfo.IsOpen() || p.HomunculusOpen,
		HomunculusSkillsOpen: u.companions.HomunculusSkill.IsOpen() || p.HomunculusSkillsOpen,
		MercenaryOpen:        u.companions.MercenaryInfo.IsOpen() || p.MercenaryOpen,
		MercenarySkillsOpen:  u.companions.MercenarySkill.IsOpen() || p.MercenarySkillsOpen,
		MinimapHidden:        u.minimap.Hidden(),
		SkillTree:            u.skillWindow.TreeMode(),
		ShortcutRows:         u.shortcutBar.VisibleRows(),
		CameraZoom:           m.camera.targetZoom(),
	}
	for _, binding := range u.windowPositions(&settings) {
		*binding.position = binding.window.SavedPosition()
	}
	return settings
}

func (m *WorldMode) SaveUISettings(ctx client.Context) {
	m.saveUISettings(ctx, false)
}

func (m *WorldMode) FlushUISettings(ctx client.Context) {
	m.saveUISettings(ctx, true)
}

func (m *WorldMode) saveUISettings(ctx client.Context, retry bool) {
	p := &m.ui.preferences
	if !p.loaded || ctx.Config.Headless || ctx.Config.ConfigPath == "" {
		return
	}
	if !retry {
		for _, binding := range m.ui.windowPositions(&p.last) {
			if binding.window.Dragging() {
				return
			}
		}
	}
	next := m.uiSettings()
	if next.CameraZoom != p.zoomTarget {
		p.zoomTarget = next.CameraZoom
		p.zoomChanged = time.Now()
	}
	if !retry && time.Since(p.zoomChanged) < 250*time.Millisecond {
		return
	}
	if next == p.last && !(retry && p.dirty) {
		return
	}
	// Remember the attempt to avoid retrying every frame; keep failures dirty
	// until a later change or a lifecycle flush successfully writes them.
	p.last = next
	_, err := ctx.Config.SaveUISettings(next)
	p.dirty = err != nil
	if err != nil {
		glog.Warnf("save UI settings: %v", err)
	}
}
