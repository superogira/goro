package ui

import (
	"testing"

	"github.com/kivutar/goro/db"
	"github.com/kivutar/goro/session"
)

func skillGamepadTestSession() *session.Session {
	return &session.Session{
		Selected: session.Character{ID: 1, Job: db.JobSwordman},
		Skills: session.Skills{
			Points: 9,
			List: []session.Skill{
				{ID: db.SkillSMBash, Level: 4, MaxLevel: 10, Upgradable: true},
				{ID: db.SkillSMMagnum, Level: 0, MaxLevel: 10, Upgradable: true},
				{ID: db.SkillSMProvoke, Level: 1, MaxLevel: 10, Upgradable: true},
			},
		},
	}
}

func TestSkillGamepadNavigateWalksTableRowsAndFooter(t *testing.T) {
	s := skillGamepadTestSession()
	window := &SkillWindow{}
	window.Toggle(Context{Session: s, ScreenW: 1280, ScreenH: 720})
	ctx := Context{Session: s, ScreenW: 1280, ScreenH: 720}

	window.GamepadNavigate(ctx, 0, 1)
	if window.gamepadRow != 0 {
		t.Fatalf("first down: row = %d, want 0", window.gamepadRow)
	}
	window.GamepadNavigate(ctx, 0, 1)
	if window.gamepadRow != 1 {
		t.Fatalf("second down: row = %d, want 1", window.gamepadRow)
	}
	window.GamepadNavigate(ctx, 0, -1)
	if window.gamepadRow != 0 {
		t.Fatalf("up: row = %d, want 0", window.gamepadRow)
	}
	// Down off the last row lands on the footer buttons, up returns.
	skillCount := len(window.activeSkills())
	window.gamepadRow = skillCount - 1
	window.GamepadNavigate(ctx, 0, 1)
	if !window.gamepadOnFooter {
		t.Fatal("down past last row should focus the footer")
	}
	window.GamepadNavigate(ctx, 1, 0)
	if window.gamepadFooter != 1 {
		t.Fatalf("footer right: index = %d, want 1 (Confirm)", window.gamepadFooter)
	}
	window.GamepadNavigate(ctx, 0, -1)
	if window.gamepadOnFooter {
		t.Fatal("up from footer should return to the rows")
	}
}

func TestSkillGamepadActivateStagesAndFooterPresses(t *testing.T) {
	s := skillGamepadTestSession()
	window := &SkillWindow{}
	window.Toggle(Context{Session: s, ScreenW: 1280, ScreenH: 720})
	ctx := Context{Session: s, ScreenW: 1280, ScreenH: 720}

	window.GamepadNavigate(ctx, 0, 1)
	window.GamepadActivate(ctx)
	if got := window.pendingFor(db.SkillSMBash); got != 1 {
		t.Fatalf("staged bash levels = %d, want 1", got)
	}
	// Footer: Reset clears the staged level.
	window.gamepadOnFooter = true
	window.gamepadFooter = 0
	window.GamepadActivate(ctx)
	if got := window.pendingFor(db.SkillSMBash); got != 0 {
		t.Fatalf("reset should clear staged levels, got %d", got)
	}
}

func TestSkillGamepadInfoOpensDetailWindow(t *testing.T) {
	s := skillGamepadTestSession()
	window := &SkillWindow{}
	window.Toggle(Context{Session: s, ScreenW: 1280, ScreenH: 720})
	ctx := Context{Session: s, ScreenW: 1280, ScreenH: 720}

	window.GamepadNavigate(ctx, 0, 1)
	window.GamepadInfo(ctx)
	if !window.detail.IsOpen() {
		t.Fatal("expected the skill detail window to open")
	}
	if window.detail.skill.ID != db.SkillSMBash {
		t.Fatalf("detail skill = %d, want bash", window.detail.skill.ID)
	}
	if !window.CloseTopDetail(ctx) {
		t.Fatal("CloseTopDetail should report the closed window")
	}
	if window.detail.IsOpen() {
		t.Fatal("detail window should be closed")
	}
}

func TestSkillGamepadTabCyclesVisibleTabs(t *testing.T) {
	s := skillGamepadTestSession()
	window := &SkillWindow{}
	window.Toggle(Context{Session: s, ScreenW: 1280, ScreenH: 720})
	ctx := Context{Session: s, ScreenW: 1280, ScreenH: 720}
	if len(window.visibleTabs) < 2 {
		t.Fatalf("swordman should expose at least two tabs, got %v", window.visibleTabs)
	}
	first := window.tab
	window.GamepadTab(ctx, 1)
	if window.tab == first {
		t.Fatal("R1 should switch to the next tab")
	}
	if window.gamepadRow != -1 || window.gamepadPosition != -1 || window.gamepadOnFooter {
		t.Fatal("tab switch should reset the gamepad selection")
	}
	window.GamepadTab(ctx, -1)
	if window.tab != first {
		t.Fatalf("L1 should return to the first tab, got %d", window.tab)
	}
}

func TestSkillGamepadNavigateGridWalksTreePositions(t *testing.T) {
	s := skillGamepadTestSession()
	window := &SkillWindow{}
	window.Toggle(Context{Session: s, ScreenW: 1280, ScreenH: 720})
	ctx := Context{Session: s, ScreenW: 1280, ScreenH: 720}
	window.gridMode = true
	window.refresh(ctx, nil)
	if window.grid == nil {
		t.Fatal("grid mode should build the skill grid widget")
	}

	window.GamepadNavigate(ctx, 1, 0)
	if window.gamepadPosition < 0 {
		t.Fatal("first d-pad press should select a tree cell")
	}
	first := window.gamepadPosition
	// Keep stepping right; the selection must always land on an occupied cell.
	window.GamepadNavigate(ctx, 1, 0)
	if window.gamepadPosition != first && !window.grid.hasEntryAt(window.gamepadPosition) {
		t.Fatalf("selection moved to empty position %d", window.gamepadPosition)
	}
	window.GamepadNavigate(ctx, 0, 1)
	if !window.grid.hasEntryAt(window.gamepadPosition) && !window.gamepadOnFooter {
		t.Fatalf("down should land on a cell or the footer, got position %d", window.gamepadPosition)
	}
}

func TestSkillGamepadToggleModeFlipsView(t *testing.T) {
	s := skillGamepadTestSession()
	window := &SkillWindow{}
	window.Toggle(Context{Session: s, ScreenW: 1280, ScreenH: 720})
	ctx := Context{Session: s, ScreenW: 1280, ScreenH: 720}
	if window.gridMode {
		t.Fatal("window should open in table mode")
	}
	window.GamepadNavigate(ctx, 0, 1)
	window.GamepadToggleMode(ctx)
	if !window.gridMode {
		t.Fatal("toggle should switch to the tree view")
	}
	if window.gamepadRow != -1 || window.gamepadPosition != -1 {
		t.Fatal("mode toggle should reset the gamepad selection")
	}
	window.GamepadToggleMode(ctx)
	if window.gridMode {
		t.Fatal("second toggle should return to the table view")
	}
}

func TestSkillDetailScrollsOnlyWhenTextOverflows(t *testing.T) {
	detail := &SkillDetailWindow{}
	// Fits: no scrolling, GamepadScroll reports false.
	detail.openSkill(Context{ScreenW: 1280, ScreenH: 720}, session.Skill{ID: db.SkillSMBash}, nil, 10, 10)
	if detail.maxScroll > 0 {
		t.Fatalf("short detail should not scroll, maxScroll = %v", detail.maxScroll)
	}
	if detail.GamepadScroll(1) {
		t.Fatal("scroll on non-overflowing detail should report false")
	}

	// Long text: scrolls down, stops reporting at the bottom edge.
	lines := make([]string, 80)
	for i := range lines {
		lines[i] = "line"
	}
	detail.lines = lines
	detail.layout()
	if detail.maxScroll <= 0 {
		t.Fatal("long detail should be scrollable")
	}
	if !detail.GamepadScroll(1) {
		t.Fatal("first down should scroll")
	}
	for i := 0; i < 100; i++ {
		detail.GamepadScroll(1)
	}
	if detail.scrollY.Get() != detail.maxScroll {
		t.Fatalf("scroll should clamp at the bottom: %v / %v", detail.scrollY.Get(), detail.maxScroll)
	}
	if detail.GamepadScroll(1) {
		t.Fatal("scroll at the bottom edge should report false so the d-pad falls through")
	}
	if !detail.GamepadScroll(-1) {
		t.Fatal("up from the bottom should scroll")
	}
}
