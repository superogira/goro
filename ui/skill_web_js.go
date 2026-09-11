//go:build js && wasm

package ui

import (
	"fmt"
	"strconv"
	"strings"
	"syscall/js"

	"github.com/kivutar/goro/glog"
)

// skillWebEnabled reports whether the page provides the DOM skill window.
func skillWebEnabled() bool {
	return js.Global().Get("goroSkillSync").Type() == js.TypeFunction
}

// webSync pushes the skill window snapshot to the page.
func (w *SkillWindow) webSync(ctx Context) {
	fn := js.Global().Get("goroSkillSync")
	if fn.Type() != js.TypeFunction {
		return
	}
	state := w.buildSkillWebState(ctx)
	obj := js.Global().Get("Object").New()
	obj.Set("open", state.Open)
	obj.Set("points", state.Points)
	obj.Set("pending", state.Pending)
	obj.Set("tab", state.Tab)
	obj.Set("grid", state.Grid)
	tabs := js.Global().Get("Array").New(len(state.Tabs))
	for i, tab := range state.Tabs {
		entry := js.Global().Get("Object").New()
		entry.Set("label", tab.Label)
		entry.Set("id", tab.ID)
		tabs.SetIndex(i, entry)
	}
	obj.Set("tabs", tabs)
	tipArray := func(lines []string) js.Value {
		arr := js.Global().Get("Array").New(len(lines))
		for i, line := range lines {
			arr.SetIndex(i, line)
		}
		return arr
	}
	rows := js.Global().Get("Array").New(len(state.Rows))
	for i, row := range state.Rows {
		entry := js.Global().Get("Object").New()
		entry.Set("id", row.ID)
		entry.Set("name", row.Name)
		entry.Set("icon", w.webSkillIcon(ctx, row.ID))
		entry.Set("type", row.Type)
		entry.Set("level", row.LevelText)
		entry.Set("sp", row.SP)
		entry.Set("range", row.Range)
		entry.Set("canStage", row.CanStage)
		entry.Set("pending", row.Pending)
		entry.Set("selectable", row.Selectable)
		entry.Set("learned", row.Learned)
		entry.Set("tip", tipArray(row.Tip))
		entry.Set("req", reqArray(row.Req))
		rows.SetIndex(i, entry)
	}
	obj.Set("rows", rows)
	cells := js.Global().Get("Array").New(len(state.Cells))
	for i, cell := range state.Cells {
		entry := js.Global().Get("Object").New()
		entry.Set("id", cell.ID)
		entry.Set("pos", cell.Position)
		entry.Set("name", cell.Name)
		entry.Set("icon", w.webSkillIcon(ctx, cell.ID))
		entry.Set("level", cell.LevelText)
		entry.Set("canStage", cell.CanStage)
		entry.Set("selectable", cell.Selectable)
		entry.Set("learned", cell.Learned)
		entry.Set("tip", tipArray(cell.Tip))
		entry.Set("req", reqArray(cell.Req))
		cells.SetIndex(i, entry)
	}
	obj.Set("cells", cells)
	fn.Invoke(obj)
}

// reqArray marshals the prerequisite list for the page's hover highlight.
func reqArray(reqs []skillWebReq) js.Value {
	arr := js.Global().Get("Array").New(len(reqs))
	for i, req := range reqs {
		entry := js.Global().Get("Object").New()
		entry.Set("id", req.ID)
		entry.Set("level", req.Level)
		entry.Set("name", req.Name)
		arr.SetIndex(i, entry)
	}
	return arr
}

// webSkillIcon resolves the shared "skill:<id>" cache key to a data URL.
func (w *SkillWindow) webSkillIcon(ctx Context, skillID uint16) string {
	if w.assets == nil {
		return ""
	}
	skill, _ := shortcutSkillByID(ctx.Session, skillID)
	return hotbarWebIcon(fmt.Sprintf("skill:%d", skillID), w.assets.SkillIconImage(ctx.Resources, skill, 24))
}

// handleSkillWebAction services one drained "skill:" action from the page.
func (w *SkillWindow) handleSkillWebAction(ctx Context, actions GameActions, action string) {
	forceResync := true
	switch {
	case action == "skill:close":
		w.webOpen = false
	case action == "skill:grid":
		w.gridMode = !w.gridMode
	case strings.HasPrefix(action, "skill:tab:"):
		if n, err := strconv.Atoi(strings.TrimPrefix(action, "skill:tab:")); err == nil && n >= 0 && n < skillTabCount {
			w.tab = n
		}
	case action == "skill:reset":
		w.clearPending()
	case action == "skill:confirm":
		w.confirmPending(ctx)
	case strings.HasPrefix(action, "skill:stage:"):
		id, err := strconv.ParseUint(strings.TrimPrefix(action, "skill:stage:"), 10, 16)
		if err != nil {
			return
		}
		skill, ok := shortcutSkillByID(ctx.Session, uint16(id))
		if !ok || !w.canStageSkill(ctx.Session, skill) {
			return
		}
		w.stageSkill(skill.ID)
	case strings.HasPrefix(action, "skill:levelsel:"):
		parts := strings.Split(strings.TrimPrefix(action, "skill:levelsel:"), ":")
		if len(parts) != 2 {
			return
		}
		id, err1 := strconv.ParseUint(parts[0], 10, 16)
		delta, err2 := strconv.Atoi(parts[1])
		if err1 != nil || err2 != nil || delta == 0 {
			return
		}
		skill, ok := shortcutSkillByID(ctx.Session, uint16(id))
		if !ok {
			return
		}
		w.changeSelectedSkillLevel(skill, delta)
	case strings.HasPrefix(action, "skill:use:"):
		id, err := strconv.ParseUint(strings.TrimPrefix(action, "skill:use:"), 10, 16)
		if err != nil {
			return
		}
		w.useSkillByID(ctx, actions, uint16(id))
	default:
		forceResync = false
	}
	if forceResync {
		w.webSyncKey = ""
	}
}

// useSkillByID casts a learned active skill (skill window double-click).
func (w *SkillWindow) useSkillByID(ctx Context, actions GameActions, skillID uint16) {
	skill, ok := shortcutSkillByID(ctx.Session, skillID)
	if !ok {
		return
	}
	if skill.Level <= 0 {
		glog.Debugf("skill use ignored id=%d: not learned", skill.ID)
		return
	}
	if !skillCanUseShortcut(skill) {
		glog.Debugf("skill use ignored id=%d: passive skill", skill.ID)
		return
	}
	if actions == nil {
		glog.Warnf("skill use failed id=%d: no game actions", skill.ID)
		return
	}
	skill.Level = w.selectedSkillLevel(skill)
	if err := actions.UseShortcutSkill(ctx, skill); err != nil {
		glog.Warnf("skill use failed id=%d: %v", skill.ID, err)
	}
}
