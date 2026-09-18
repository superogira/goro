package ui

import (
	"github.com/gogpu/ui/state"
	"github.com/kivutar/goro/session"
)

// Handheld controls for the skill window, driven from game/gamepad.go: the
// d-pad walks the tree grid (or the table rows), A stages a level-up on the
// selected skill, Y opens its detail window, L1/R1 cycle the class tabs, and
// pressing down off the last row moves the cursor onto the footer's
// Reset/Confirm buttons.

// GamepadTab cycles the visible class tabs (dir < 0 = L1, dir > 0 = R1).
func (w *SkillWindow) GamepadTab(ctx Context, dir int) {
	if !w.IsOpen() || dir == 0 || len(w.visibleTabs) == 0 {
		return
	}
	index := 0
	for i, tabID := range w.visibleTabs {
		if tabID == w.tab {
			index = i
			break
		}
	}
	count := len(w.visibleTabs)
	index = ((index+dir)%count + count) % count
	w.switchTab(ctx, w.visibleTabs[index], w.assets, w.actions)
}

// GamepadNavigate moves the handheld cursor: d-pad over the skill cells (or
// table rows), down off the last row onto the footer buttons, up from the
// footer back to the grid.
func (w *SkillWindow) GamepadNavigate(ctx Context, dx, dy int) {
	if !w.IsOpen() || (dx == 0 && dy == 0) {
		return
	}
	if w.gamepadOnFooter {
		switch {
		case dy < 0:
			w.gamepadOnFooter = false
		case dx > 0:
			w.gamepadFooter = minInt(1, w.gamepadFooter+1)
		case dx < 0:
			w.gamepadFooter = maxInt(0, w.gamepadFooter-1)
		default:
			return
		}
		w.refresh(ctx, w.actions)
		return
	}
	if w.gridMode {
		w.gamepadNavigateGrid(ctx, dx, dy)
		return
	}
	w.gamepadNavigateTable(ctx, dy)
}

func (w *SkillWindow) gamepadNavigateGrid(ctx Context, dx, dy int) {
	if w.grid == nil {
		return
	}
	position := w.gamepadPosition
	if position < 0 {
		position = w.grid.firstEntryPosition()
		if position < 0 {
			return
		}
		w.gamepadPosition = position
		w.gamepadScrollGridToRow(ctx, position/skillGridColumns)
		w.refresh(ctx, w.actions)
		return
	}
	col := position % skillGridColumns
	row := position / skillGridColumns
	for {
		col += dx
		row += dy
		if col < 0 || col >= skillGridColumns || row < 0 || row >= w.grid.totalRows() {
			break
		}
		candidate := row*skillGridColumns + col
		if w.grid.hasEntryAt(candidate) {
			w.gamepadPosition = candidate
			w.gamepadScrollGridToRow(ctx, row)
			w.refresh(ctx, w.actions)
			return
		}
	}
	// Nothing further in that direction: down drops to the footer buttons.
	if dy > 0 {
		w.gamepadOnFooter = true
		w.gamepadFooter = 0
		w.refresh(ctx, w.actions)
	}
}

func (w *SkillWindow) gamepadNavigateTable(ctx Context, dy int) {
	if dy == 0 {
		return
	}
	w.ensureSkillView(ctx)
	skills := w.activeSkills()
	if len(skills) == 0 {
		return
	}
	row := w.gamepadRow
	if row < 0 {
		row = 0
	} else {
		row += dy
	}
	if row < 0 {
		row = 0
	}
	if row >= len(skills) {
		if dy > 0 {
			w.gamepadOnFooter = true
			w.gamepadFooter = 0
		}
		w.refresh(ctx, w.actions)
		return
	}
	w.gamepadRow = row
	w.ensureGamepadRowSignal().Set(row)
	w.gamepadScrollTableToRow(ctx, row)
	w.refresh(ctx, w.actions)
}

// GamepadActivate presses whatever the cursor sits on: the footer button, or
// a level-up stage on the selected skill (the same thing clicking the +/cell
// does).
func (w *SkillWindow) GamepadActivate(ctx Context) {
	if !w.IsOpen() {
		return
	}
	if w.gamepadOnFooter {
		if w.gamepadFooter == 0 {
			w.clearPending()
		} else {
			w.confirmPending(ctx)
		}
		w.dirty = true
		return
	}
	skill, ok := w.gamepadSelectedSkill(ctx)
	if !ok || !w.canStageSkill(ctx.Session, skill) {
		return
	}
	w.stageSkill(skill.ID)
	w.dirty = true
}

// GamepadInfo opens the standalone skill detail window next to the skill
// window; pressing it again on another skill swaps the content.
func (w *SkillWindow) GamepadInfo(ctx Context) {
	if !w.IsOpen() {
		return
	}
	skill, ok := w.gamepadSelectedSkill(ctx)
	if !ok || skill.ID == 0 {
		return
	}
	width, _ := w.windowSize()
	icon := w.skillIconImage(ctx, w.assets, skill)
	w.detail.openSkill(ctx, skill, icon, w.x+width+8, w.y)
}

// CloseTopDetail closes the skill detail window; it reports whether one was
// open, so the handheld B stack can close it before the skill window itself.
func (w *SkillWindow) CloseTopDetail(ctx Context) bool {
	return w.detail.closeIfOpen(ctx)
}

func (w *SkillWindow) gamepadSelectedSkill(ctx Context) (session.Skill, bool) {
	if w.gridMode {
		if w.grid == nil || w.gamepadPosition < 0 {
			return session.Skill{}, false
		}
		entry, ok := w.grid.entryAtPosition(w.gamepadPosition)
		if !ok {
			return session.Skill{}, false
		}
		return entry.skill, true
	}
	w.ensureSkillView(ctx)
	skills := w.activeSkills()
	if w.gamepadRow < 0 || w.gamepadRow >= len(skills) {
		return session.Skill{}, false
	}
	return skills[w.gamepadRow], true
}

func (w *SkillWindow) gamepadScrollGridToRow(ctx Context, row int) {
	scroll := w.ensureScrollSignal()
	top := float32(row * skillGridCellH)
	bottom := top + skillGridCellH
	value := scroll.Get()
	switch {
	case top < value:
		scroll.Set(top)
	case bottom > value+skillGridViewH:
		scroll.Set(bottom - skillGridViewH)
	}
}

func (w *SkillWindow) gamepadScrollTableToRow(ctx Context, row int) {
	scroll := w.ensureScrollSignal()
	top := float32(row * skillRowH)
	bottom := top + skillRowH
	value := scroll.Get()
	switch {
	case top < value:
		scroll.Set(top)
	case bottom > value+skillTableBodyH:
		scroll.Set(bottom - skillTableBodyH)
	}
}

func (w *SkillWindow) ensureGamepadRowSignal() state.Signal[int] {
	if w.gamepadRowSig == nil {
		w.gamepadRowSig = state.NewSignal[int](-1)
	}
	return w.gamepadRowSig
}
