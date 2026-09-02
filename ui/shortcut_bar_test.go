package ui

import (
	"testing"

	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/widget"
	"github.com/kivutar/goro/db"
	"github.com/kivutar/goro/input"
	"github.com/kivutar/goro/network"
	"github.com/kivutar/goro/session"
)

func TestShortcutSlotHotkeyRoundTrip(t *testing.T) {
	item := shortcutSlotState{kind: shortcutItem, itemID: 601, identified: true}
	itemHotkey := item.hotkey()
	if itemHotkey.Type != network.HotkeyTypeItem || itemHotkey.ID != 601 || itemHotkey.Level != 0 {
		t.Fatalf("item hotkey = %+v", itemHotkey)
	}
	if got := shortcutSlotFromHotkey(session.HotkeySlot{Type: itemHotkey.Type, ID: itemHotkey.ID, Level: itemHotkey.Level}); got.kind != shortcutItem || got.itemID != 601 || !got.identified {
		t.Fatalf("item slot = %+v", got)
	}

	skill := shortcutSlotState{kind: shortcutSkill, skillID: 6, skillLevel: 2}
	skillHotkey := skill.hotkey()
	if skillHotkey.Type != network.HotkeyTypeSkill || skillHotkey.ID != 6 || skillHotkey.Level != 2 {
		t.Fatalf("skill hotkey = %+v", skillHotkey)
	}
	if got := shortcutSlotFromHotkey(session.HotkeySlot{Type: skillHotkey.Type, ID: skillHotkey.ID, Level: skillHotkey.Level}); got != skill {
		t.Fatalf("skill slot = %+v, want %+v", got, skill)
	}
}

func TestShortcutBarSyncsFromSessionHotkeys(t *testing.T) {
	slots := make([]session.HotkeySlot, shortcutTotalSlots)
	slots[0] = session.HotkeySlot{Type: network.HotkeyTypeSkill, ID: 6, Level: 2}
	slots[1] = session.HotkeySlot{Type: network.HotkeyTypeItem, ID: 501}
	slots[shortcutCols+1] = session.HotkeySlot{Type: network.HotkeyTypeSkill, ID: 28, Level: 4}
	slots[2*shortcutCols] = session.HotkeySlot{Type: network.HotkeyTypeSkill, ID: 29, Level: 5}
	ctx := Context{Session: &session.Session{Hotkeys: session.Hotkeys{
		Loaded:  true,
		Version: 3,
		Slots:   slots,
	}}}

	bar := &ShortcutBar{}
	bar.SyncFromSession(ctx)
	if bar.slots[0] != (shortcutSlotState{kind: shortcutSkill, skillID: 6, skillLevel: 2}) {
		t.Fatalf("slot 1 = %+v", bar.slots[0])
	}
	if bar.slots[1].kind != shortcutItem || bar.slots[1].itemID != 501 {
		t.Fatalf("slot 2 = %+v", bar.slots[1])
	}
	if bar.slots[shortcutCols+1] != (shortcutSlotState{kind: shortcutSkill, skillID: 28, skillLevel: 4}) {
		t.Fatalf("second-row slot = %+v", bar.slots[shortcutCols+1])
	}
	if bar.slots[2*shortcutCols] != (shortcutSlotState{kind: shortcutSkill, skillID: 29, skillLevel: 5}) {
		t.Fatalf("third-row slot = %+v", bar.slots[2*shortcutCols])
	}
	if bar.hotkeyVersion != 3 {
		t.Fatalf("hotkey version = %d, want 3", bar.hotkeyVersion)
	}
}

func TestShortcutDropMarksShortcutOverlayDirty(t *testing.T) {
	app := &shortcutInvalidatingApp{}
	ctx := Context{
		ScreenW:   800,
		ScreenH:   600,
		UIApp:     app,
		UIManager: &shortcutInvalidatingManager{},
		Session:   &session.Session{},
	}
	bar := &ShortcutBar{}
	bar.Publish(ctx, nil, nil)
	if bar.root == nil {
		t.Fatal("shortcut bar root was not published")
	}
	x, y := bar.slotBounds(ctx, 0)

	if !bar.AcceptSkillDrop(ctx, session.Skill{ID: 6, Level: 2, Type: 1}, x+1, y+1) {
		t.Fatal("skill drop was not accepted")
	}
	if app.invalidates != 1 {
		t.Fatalf("shortcut drop invalidates = %d, want 1", app.invalidates)
	}
	if got := bar.slots[0]; got.kind != shortcutSkill || got.skillID != 6 || got.skillLevel != 2 {
		t.Fatalf("slot = %+v", got)
	}
}

func TestShortcutDropRejectsPassiveSkill(t *testing.T) {
	app := &shortcutInvalidatingApp{}
	ctx := Context{
		ScreenW:   800,
		ScreenH:   600,
		UIApp:     app,
		UIManager: &shortcutInvalidatingManager{},
		Session:   &session.Session{},
	}
	bar := &ShortcutBar{}
	bar.Publish(ctx, nil, nil)
	x, y := bar.slotBounds(ctx, 0)
	original := shortcutSlotState{kind: shortcutItem, itemID: 501}
	bar.slots[0] = original

	if !bar.AcceptSkillDrop(ctx, session.Skill{ID: db.SkillALDemonbane, Level: 10, Type: 0}, x+1, y+1) {
		t.Fatal("passive skill drop over shortcut should be consumed")
	}
	if bar.slots[0] != original {
		t.Fatalf("slot = %+v, want unchanged %+v", bar.slots[0], original)
	}
	if app.invalidates != 0 {
		t.Fatalf("shortcut drop invalidates = %d, want 0", app.invalidates)
	}
}

func TestShortcutBarVisibleRowsClamp(t *testing.T) {
	bar := &ShortcutBar{}
	if got := bar.visibleRowCount(); got != shortcutMinRows {
		t.Fatalf("default visible rows = %d, want %d", got, shortcutMinRows)
	}
	bar.setVisibleRows(Context{}, shortcutMaxRows+3)
	if got := bar.visibleRowCount(); got != shortcutMaxRows {
		t.Fatalf("max visible rows = %d, want %d", got, shortcutMaxRows)
	}
	bar.setVisibleRows(Context{}, 0)
	if got := bar.visibleRowCount(); got != shortcutMinRows {
		t.Fatalf("min visible rows = %d, want %d", got, shortcutMinRows)
	}
}

func TestShortcutDropUsesVisibleAdditionalRows(t *testing.T) {
	app := &shortcutInvalidatingApp{}
	ctx := Context{
		ScreenW:   800,
		ScreenH:   600,
		UIApp:     app,
		UIManager: &shortcutInvalidatingManager{},
		Session:   &session.Session{},
	}
	bar := &ShortcutBar{}
	bar.setVisibleRows(ctx, 2)
	x, y := bar.slotBounds(ctx, shortcutCols)

	if !bar.AcceptSkillDrop(ctx, session.Skill{ID: 6, Level: 2, Type: 1}, x+1, y+1) {
		t.Fatal("second-row skill drop was not accepted")
	}
	if got := bar.slots[shortcutCols]; got.kind != shortcutSkill || got.skillID != 6 || got.skillLevel != 2 {
		t.Fatalf("second-row slot = %+v", got)
	}
	if len(ctx.Session.Hotkeys.Slots) <= shortcutCols || ctx.Session.Hotkeys.Slots[shortcutCols].ID != 6 {
		t.Fatalf("session hotkeys = %+v", ctx.Session.Hotkeys.Slots)
	}
}

func TestShortcutDropUsesThirdVisibleRow(t *testing.T) {
	ctx := Context{
		ScreenW: 800,
		ScreenH: 600,
		Session: &session.Session{},
	}
	bar := &ShortcutBar{}
	bar.setVisibleRows(ctx, 3)
	slot := 2 * shortcutCols
	x, y := bar.slotBounds(ctx, slot)

	if !bar.AcceptSkillDrop(ctx, session.Skill{ID: 6, Level: 2, Type: 1}, x+1, y+1) {
		t.Fatal("third-row skill drop was not accepted")
	}
	if got := bar.slots[slot]; got.kind != shortcutSkill || got.skillID != 6 || got.skillLevel != 2 {
		t.Fatalf("third-row slot = %+v", got)
	}
}

func TestShortcutDropIgnoresHiddenRows(t *testing.T) {
	ctx := Context{
		ScreenW: 800,
		ScreenH: 600,
		Session: &session.Session{},
	}
	bar := &ShortcutBar{}
	x, y := bar.slotBounds(ctx, shortcutCols)

	if bar.AcceptSkillDrop(ctx, session.Skill{ID: 6, Level: 2, Type: 1}, x+1, y+1) {
		t.Fatal("hidden-row skill drop was accepted")
	}
	if got := bar.slots[shortcutCols]; got.kind != shortcutEmpty {
		t.Fatalf("hidden-row slot = %+v", got)
	}
}

func TestShortcutBarClearsCachedSlotsOnCharacterChange(t *testing.T) {
	bar := &ShortcutBar{}
	bar.charID = 1
	bar.hotkeyVersion = 4
	bar.slots[0] = shortcutSlotState{kind: shortcutSkill, skillID: 6, skillLevel: 2}
	bar.slots[shortcutTotalSlots-1] = shortcutSlotState{kind: shortcutSkill, skillID: 28, skillLevel: 4}
	ctx := Context{Session: &session.Session{
		CharID: 2,
		Hotkeys: session.Hotkeys{
			Loaded:  true,
			Version: 5,
			Slots:   make([]session.HotkeySlot, network.HotkeyListSlots2008),
		},
	}}

	bar.SyncFromSession(ctx)
	if got := bar.slots[0]; got.kind != shortcutEmpty {
		t.Fatalf("first-row stale slot = %+v", got)
	}
	if got := bar.slots[shortcutTotalSlots-1]; got.kind != shortcutEmpty {
		t.Fatalf("last-row stale slot = %+v", got)
	}
	if bar.charID != 2 {
		t.Fatalf("bar charID = %d, want 2", bar.charID)
	}
}

func TestShortcutBarActivatesSecondRowNumberKey(t *testing.T) {
	inputState := input.NewState()
	inputState.SetKey(input.Key1, true)
	actions := &skillWindowTestRenderer{}
	bar := &ShortcutBar{}
	bar.slots[shortcutCols] = shortcutSlotState{kind: shortcutSkill, skillID: 6, skillLevel: 2}

	if !bar.Update(shortcutBarActionContext(inputState), actions) {
		t.Fatal("second-row shortcut key was not consumed")
	}
	if actions.used.ID != 6 || actions.used.Level != 2 {
		t.Fatalf("used skill = %+v", actions.used)
	}
}

func TestShortcutBarSkipsKeyActivationWhenKeyboardBlocked(t *testing.T) {
	inputState := input.NewState()
	inputState.SetKey(input.Key1, true)
	actions := &shortcutBlockingActions{blocked: true}
	bar := &ShortcutBar{}
	bar.slots[shortcutCols] = shortcutSlotState{kind: shortcutSkill, skillID: 6, skillLevel: 2}

	if bar.Update(shortcutBarActionContext(inputState), actions) {
		t.Fatal("blocked shortcut key was consumed")
	}
	if actions.used.ID != 0 {
		t.Fatalf("used skill = %+v, want none", actions.used)
	}
}

func TestShortcutBarActivatesThirdRowLetterKey(t *testing.T) {
	inputState := input.NewState()
	inputState.SetKey(input.KeyQ, true)
	actions := &skillWindowTestRenderer{}
	bar := &ShortcutBar{}
	bar.slots[2*shortcutCols] = shortcutSlotState{kind: shortcutSkill, skillID: 28, skillLevel: 4}

	if !bar.Update(shortcutBarActionContext(inputState), actions) {
		t.Fatal("third-row shortcut key was not consumed")
	}
	if actions.used.ID != 28 || actions.used.Level != 4 {
		t.Fatalf("used skill = %+v", actions.used)
	}
}

func TestShortcutBarF12CyclesVisibleRows(t *testing.T) {
	inputState := input.NewState()
	inputState.SetKey(input.KeyF12, true)
	bar := &ShortcutBar{}

	if !bar.Update(shortcutBarActionContext(inputState), nil) {
		t.Fatal("F12 shortcut was not consumed")
	}
	if got := bar.visibleRowCount(); got != 2 {
		t.Fatalf("visible rows = %d, want 2", got)
	}

	bar.setVisibleRows(Context{}, shortcutMaxRows)
	if !bar.Update(shortcutBarActionContext(inputState), nil) {
		t.Fatal("F12 wrap shortcut was not consumed")
	}
	if got := bar.visibleRowCount(); got != shortcutMinRows {
		t.Fatalf("wrapped visible rows = %d, want %d", got, shortcutMinRows)
	}
}

func TestShortcutLabelsMatchROBrowserDefaults(t *testing.T) {
	tests := map[int]string{
		0:                      "F1",
		shortcutCols - 1:       "F9",
		shortcutCols:           "1",
		2*shortcutCols - 1:     "9",
		2 * shortcutCols:       "Q",
		shortcutTotalSlots - 1: "O",
	}
	for slot, want := range tests {
		if got := shortcutLabelForSlot(slot); got != want {
			t.Fatalf("slot %d label = %q, want %q", slot, got, want)
		}
	}
}

func shortcutBarActionContext(inputState *input.State) Context {
	return Context{
		ScreenW: 800,
		ScreenH: 600,
		Input:   inputState,
		Session: &session.Session{
			Skills: session.Skills{
				List: []session.Skill{
					{ID: 6, Level: 8, Type: 1, Name: "Provoke"},
					{ID: 28, Level: 10, Type: 1, Name: "Cold Bolt"},
				},
			},
		},
	}
}

type shortcutInvalidatingManager struct {
	overlays []widget.Widget
}

func (m *shortcutInvalidatingManager) AddOverlay(root widget.Widget) {
	m.overlays = append(m.overlays, root)
}

func (m *shortcutInvalidatingManager) RemoveOverlay(root widget.Widget) {
	for i, overlay := range m.overlays {
		if overlay == root {
			m.overlays = append(m.overlays[:i], m.overlays[i+1:]...)
			return
		}
	}
}

func (m *shortcutInvalidatingManager) Clear() {
	m.overlays = nil
}

type shortcutInvalidatingApp struct {
	invalidates int
}

type shortcutBlockingActions struct {
	skillWindowTestRenderer
	blocked bool
}

func (a *shortcutBlockingActions) KeyboardShortcutsBlocked(Context) bool {
	return a.blocked
}

func (a *shortcutInvalidatingApp) SetUIRoot(widget.Widget) {}

func (a *shortcutInvalidatingApp) Frame() {}

func (a *shortcutInvalidatingApp) Invalidate() {
	a.invalidates++
}

func (a *shortcutInvalidatingApp) InvalidateRect(geometry.Rect) {
	a.invalidates++
}

func (a *shortcutInvalidatingApp) RequestFullRepaint() {}

func (a *shortcutInvalidatingApp) WidgetContext() widget.Context { return nil }

func (a *shortcutInvalidatingApp) Cursor() widget.CursorType {
	return widget.CursorDefault
}

func (a *shortcutInvalidatingApp) HoveredWidget() widget.Widget {
	return nil
}

func TestShortcutPublishDoesNotKeepOverlayDirty(t *testing.T) {
	ctx := Context{
		ScreenW:   800,
		ScreenH:   600,
		UIManager: NewManager(),
		Session:   &session.Session{},
	}
	bar := &ShortcutBar{}
	bar.Publish(ctx, nil, nil)
	if clear, ok := bar.root.(interface {
		ClearRedraw()
		ClearSceneDirty()
	}); ok {
		clear.ClearRedraw()
		clear.ClearSceneDirty()
	} else {
		t.Fatal("shortcut overlay cannot simulate clean boundary")
	}

	bar.Publish(ctx, nil, nil)
	if redraw, ok := bar.root.(interface{ NeedsRedraw() bool }); !ok || redraw.NeedsRedraw() {
		t.Fatal("shortcut publish dirtied an unchanged overlay")
	}
}

func TestInventoryItemForShortcutFallsBackToItemID(t *testing.T) {
	s := &session.Session{
		Inventory: session.Inventory{
			Items: []session.InventoryItem{
				{Index: 8, ItemID: 501, Amount: 2},
				{Index: 14, ItemID: 601, Amount: 3},
			},
		},
	}

	item, ok := inventoryItemForShortcut(s, 99, 601)
	if !ok {
		t.Fatal("item not found")
	}
	if item.Index != 14 || item.ItemID != 601 {
		t.Fatalf("item = %+v", item)
	}
}

func TestInventoryItemForShortcutRejectsReusedIndexWithDifferentItem(t *testing.T) {
	s := &session.Session{
		Inventory: session.Inventory{
			Items: []session.InventoryItem{
				{Index: 12, ItemID: 602, Amount: 1},
			},
		},
	}

	item, ok := inventoryItemForShortcut(s, 12, 501)
	if ok {
		t.Fatalf("shortcut resolved reused index to wrong item: %+v", item)
	}
}

func TestShortcutBarClearsTotallyConsumedItem(t *testing.T) {
	app := &shortcutInvalidatingApp{}
	s := &session.Session{Hotkeys: session.Hotkeys{
		Loaded:  true,
		Version: 7,
		Slots: []session.HotkeySlot{
			{},
			{},
			{Type: network.HotkeyTypeItem, ID: 501},
			{Type: network.HotkeyTypeItem, ID: 602},
			{Type: network.HotkeyTypeItem, ID: 501},
		},
	}}
	ctx := Context{Session: s, UIApp: app}
	bar := &ShortcutBar{}
	bar.slots[2] = shortcutSlotState{kind: shortcutItem, itemIndex: 12, itemID: 501}
	bar.slots[3] = shortcutSlotState{kind: shortcutItem, itemIndex: 12, itemID: 602}
	bar.slots[4] = shortcutSlotState{kind: shortcutItem, itemIndex: 14, itemID: 501}

	if !bar.ClearDepletedItem(ctx, 12, 501) {
		t.Fatal("totally consumed item shortcuts were not cleared")
	}
	if bar.slots[2].kind != shortcutEmpty || bar.slots[4].kind != shortcutEmpty {
		t.Fatalf("consumed item shortcuts remain: slot3=%+v slot5=%+v", bar.slots[2], bar.slots[4])
	}
	if bar.slots[3].kind != shortcutItem || bar.slots[3].itemID != 602 {
		t.Fatalf("unrelated shortcut changed: slot4=%+v", bar.slots[3])
	}
	if s.Hotkeys.Slots[2].ID != 0 || s.Hotkeys.Slots[4].ID != 0 || s.Hotkeys.Slots[3].ID != 602 {
		t.Fatalf("session hotkeys = %+v", s.Hotkeys.Slots)
	}
	if app.invalidates != 1 {
		t.Fatalf("shortcut invalidates = %d, want 1", app.invalidates)
	}
}

func TestShortcutBarKeepsItemWhenAnotherStackRemains(t *testing.T) {
	s := &session.Session{Inventory: session.Inventory{Items: []session.InventoryItem{
		{Index: 14, ItemID: 501, Amount: 2},
	}}}
	bar := &ShortcutBar{}
	bar.slots[2] = shortcutSlotState{kind: shortcutItem, itemIndex: 12, itemID: 501}

	if bar.ClearDepletedItem(Context{Session: s}, 12, 501) {
		t.Fatal("item shortcut was cleared while another stack remained")
	}
	if bar.slots[2].kind != shortcutItem {
		t.Fatalf("item shortcut changed: %+v", bar.slots[2])
	}
}

func TestSkillForShortcutUsesSelectedLevel(t *testing.T) {
	s := &session.Session{
		Skills: session.Skills{
			List: []session.Skill{{ID: 6, Level: 8, Range: 9}},
		},
	}

	skill, ok := skillForShortcut(s, shortcutSlotState{kind: shortcutSkill, skillID: 6, skillLevel: 2})
	if !ok {
		t.Fatal("skill not found")
	}
	if skill.Level != 2 {
		t.Fatalf("shortcut skill level = %d, want selected level 2", skill.Level)
	}
}

func TestSkillForShortcutFallsBackAndClampsLevel(t *testing.T) {
	s := &session.Session{
		Skills: session.Skills{
			List: []session.Skill{{ID: 6, Level: 4, Range: 9}},
		},
	}

	skill, ok := skillForShortcut(s, shortcutSlotState{kind: shortcutSkill, skillID: 6})
	if !ok {
		t.Fatal("legacy skill shortcut not found")
	}
	if skill.Level != 4 {
		t.Fatalf("legacy shortcut level = %d, want learned level 4", skill.Level)
	}

	skill, ok = skillForShortcut(s, shortcutSlotState{kind: shortcutSkill, skillID: 6, skillLevel: 9})
	if !ok {
		t.Fatal("clamped skill shortcut not found")
	}
	if skill.Level != 4 {
		t.Fatalf("clamped shortcut level = %d, want learned level 4", skill.Level)
	}
}

func TestFixedLevelShortcutUsesLearnedLevel(t *testing.T) {
	s := &session.Session{Skills: session.Skills{List: []session.Skill{{
		ID: db.SkillCRShieldboomerang, Level: 5, Type: 1,
	}}}}

	skill, ok := skillForShortcut(s, shortcutSlotState{kind: shortcutSkill, skillID: db.SkillCRShieldboomerang, skillLevel: 2})
	if !ok {
		t.Fatal("Shield Boomerang shortcut skill not found")
	}
	if skill.Level != 5 {
		t.Fatalf("fixed-level Shield Boomerang shortcut level = %d, want learned level 5", skill.Level)
	}
}

func TestSkillForShortcutResolvesHomunculusSkills(t *testing.T) {
	s := &session.Session{
		Homunculus: session.Companion{
			Active: true,
			Skills: session.Skills{
				List: []session.Skill{{ID: db.SkillHvanCaprice, Level: 4, Type: 1, Name: "Caprice"}},
			},
		},
	}

	skill, ok := skillForShortcut(s, shortcutSlotState{kind: shortcutSkill, skillID: db.SkillHvanCaprice, skillLevel: 2})
	if !ok {
		t.Fatal("homunculus shortcut skill not found")
	}
	if skill.ID != db.SkillHvanCaprice || skill.Level != 2 || skill.Name != "Caprice" {
		t.Fatalf("homunculus shortcut skill = %+v", skill)
	}
}

func TestSkillForShortcutResolvesAndActivatesGuildSkills(t *testing.T) {
	skill := session.Skill{ID: db.SkillGdBattleorder, Level: 1, Type: 4, Name: "Battle Command"}
	s := &session.Session{Guild: session.Guild{Skills: []session.Skill{skill}}}
	entry := shortcutSlotState{kind: shortcutSkill, skillID: skill.ID, skillLevel: 1}

	resolved, ok := skillForShortcut(s, entry)
	if !ok {
		t.Fatal("guild shortcut skill not found")
	}
	if resolved.ID != skill.ID || resolved.Level != skill.Level || resolved.Name != skill.Name {
		t.Fatalf("guild shortcut skill = %+v, want %+v", resolved, skill)
	}

	actions := &skillWindowTestRenderer{}
	bar := &ShortcutBar{}
	bar.slots[0] = entry
	bar.activate(Context{Session: s}, actions, 0)
	if actions.used.ID != skill.ID || actions.used.Level != skill.Level {
		t.Fatalf("activated guild skill = %+v, want %+v", actions.used, skill)
	}
}

func TestSkillForShortcutPrefersMercenaryThenHomunculusBeforePlayer(t *testing.T) {
	s := &session.Session{
		Skills: session.Skills{
			List: []session.Skill{{ID: db.SkillMsBash, Level: 10, Type: 1, Name: "Player"}},
		},
		Homunculus: session.Companion{
			Active: true,
			Skills: session.Skills{
				List: []session.Skill{{ID: db.SkillMsBash, Level: 4, Type: 1, Name: "Homunculus"}},
			},
		},
		Mercenary: session.Companion{
			Active: true,
			Skills: session.Skills{
				List: []session.Skill{{ID: db.SkillMsBash, Level: 2, Type: 1, Name: "Mercenary"}},
			},
		},
	}

	skill, ok := skillForShortcut(s, shortcutSlotState{kind: shortcutSkill, skillID: db.SkillMsBash})
	if !ok {
		t.Fatal("shortcut skill not found")
	}
	if skill.Name != "Mercenary" || skill.Level != 2 {
		t.Fatalf("shortcut skill = %+v, want mercenary first", skill)
	}

	s.Mercenary.Active = false
	skill, ok = skillForShortcut(s, shortcutSlotState{kind: shortcutSkill, skillID: db.SkillMsBash})
	if !ok {
		t.Fatal("shortcut skill not found after mercenary deactivated")
	}
	if skill.Name != "Homunculus" || skill.Level != 4 {
		t.Fatalf("shortcut skill = %+v, want homunculus second", skill)
	}
}

func TestShortcutSkillTooltipUsesHotkeyAndName(t *testing.T) {
	bar := &ShortcutBar{}
	bar.slots[1] = shortcutSlotState{kind: shortcutSkill, skillID: 6, skillLevel: 2}
	bar.ctx = Context{
		ScreenW:   800,
		ScreenH:   600,
		UIManager: NewManager(),
		Session: &session.Session{
			Skills: session.Skills{
				List: []session.Skill{{ID: 6, Level: 8, Name: "Provoke"}},
			},
		},
	}

	bar.showTooltip(1)
	if !bar.tooltip.Open() {
		t.Fatal("tooltip should be open")
	}
	if got := bar.tooltipText(1); got != "[ F2 ] Provoke" {
		t.Fatalf("tooltip text = %q", got)
	}
	if got := bar.tooltip.Text(); got != "[ F2 ] Provoke" {
		t.Fatalf("published tooltip text = %q", got)
	}
}

func TestShortcutTooltipShowsHotkeyForEmptySlot(t *testing.T) {
	bar := &ShortcutBar{}
	bar.slots[0] = shortcutSlotState{kind: shortcutSkill, skillID: 6, skillLevel: 2}
	bar.ctx = Context{
		ScreenW:   800,
		ScreenH:   600,
		UIManager: NewManager(),
		Session: &session.Session{
			Skills: session.Skills{
				List: []session.Skill{{ID: 6, Level: 2, Name: "Provoke"}},
			},
		},
	}

	bar.showTooltip(0)
	if !bar.tooltip.Open() {
		t.Fatal("tooltip did not open")
	}
	bar.showTooltip(1)
	if !bar.tooltip.Open() {
		t.Fatal("tooltip closed after hovering empty slot")
	}
	if got := bar.tooltipText(1); got != "F2" {
		t.Fatalf("empty tooltip text = %q, want F2", got)
	}
	if got := bar.tooltip.Text(); got != "F2" {
		t.Fatalf("published empty tooltip text = %q, want F2", got)
	}
}
