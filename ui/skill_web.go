package ui

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/kivutar/goro/db"
	"github.com/kivutar/goro/session"
)

// Skill window DOM state. The page renders tabs, a table or tree grid view,
// the staged level-up footer, and hover tooltips from this snapshot; the
// icon is a cache key resolved to a data URL by the wasm-side marshaler.

type skillWebTab struct {
	Label string
	ID    int
	Icon  string // tab number as text, kept for markup symmetry
}

type skillWebRow struct {
	ID          uint16
	Name        string
	Icon        string
	Type        string // "P" passive / "A" active
	LevelText   string
	SP          int
	Range       int
	CanStage    bool
	Pending     int
	Selectable  bool
	Learned     bool
	Tip         []string
}

type skillWebCell struct {
	ID         uint16
	Position   int
	Name       string
	Icon       string
	LevelText  string
	CanStage   bool
	Selectable bool
	Learned    bool
	Tip        []string
}

type skillWebState struct {
	Open    bool
	Points  int
	Pending int
	Tab     int
	Grid    bool
	Tabs    []skillWebTab
	Rows    []skillWebRow
	Cells   []skillWebCell
}

// skillWebKey summarizes everything the DOM skill window shows.
func (w *SkillWindow) skillWebKey(ctx Context) string {
	return fmt.Sprintf("%t|%d|%t|%s", w.webOpen, w.tab, w.gridMode, w.skillSnapshot(ctx.Session))
}

// buildSkillWebState assembles the DOM snapshot for the active tab. Pure
// data: icons are cache keys ("skill:<id>"), tips are pre-wrapped lines.
func (w *SkillWindow) buildSkillWebState(ctx Context) skillWebState {
	w.ensureSkillView(ctx)
	state := skillWebState{
		Open:    w.webOpen,
		Points:  maxInt(0, sessionSkillPoints(ctx.Session)-w.pendingCount()),
		Pending: w.pendingCount(),
		Tab:     w.tab,
		Grid:    w.gridMode,
	}
	for _, tabID := range w.visibleTabs {
		state.Tabs = append(state.Tabs, skillWebTab{
			Label: skillTabs[tabID].label,
			ID:    tabID,
			Icon:  strconv.Itoa(tabID),
		})
	}
	if w.gridMode {
		state.Cells = w.buildSkillWebCells(ctx)
	} else {
		state.Rows = w.buildSkillWebRows(ctx)
	}
	return state
}

func (w *SkillWindow) buildSkillWebRows(ctx Context) []skillWebRow {
	skills := w.activeSkills()
	rows := make([]skillWebRow, 0, len(skills))
	for _, skill := range skills {
		display := w.skillWithPending(skill)
		selectable, known := db.SkillLevelSelectable(skill.ID)
		isSelectable := known && selectable && skill.Level > 0
		rows = append(rows, skillWebRow{
			ID:         skill.ID,
			Name:       trimRunes(skillDisplayName(ctx.Resources, display), 18),
			Icon:       fmt.Sprintf("skill:%d", skill.ID),
			Type:       skillTypeLabel(display),
			LevelText:  skillLevelText(display, w.selectedSkillLevel(skill), isSelectable),
			SP:         display.SPCost,
			Range:      display.Range,
			CanStage:   w.canStageSkill(ctx.Session, skill),
			Pending:    w.pendingFor(skill.ID),
			Selectable: isSelectable,
			Learned:    display.Level > 0,
			Tip:        skillWebTipLines(ctx, skill),
		})
	}
	return rows
}

func (w *SkillWindow) buildSkillWebCells(ctx Context) []skillWebCell {
	skills := w.gridSkills(ctx)
	positions := skillGridPositions(ctx.Resources, selectedJob(ctx.Session), w.tab, skills)
	cells := make([]skillWebCell, 0, len(skills))
	for _, skill := range skills {
		display := w.skillWithPending(skill)
		selectable, known := db.SkillLevelSelectable(skill.ID)
		isSelectable := known && selectable && skill.Level > 0
		cells = append(cells, skillWebCell{
			ID:         skill.ID,
			Position:   positions[skill.ID],
			Name:       trimRunes(skillDisplayName(ctx.Resources, display), 10),
			Icon:       fmt.Sprintf("skill:%d", skill.ID),
			LevelText:  skillLevelText(display, w.selectedSkillLevel(skill), isSelectable),
			CanStage:   w.canStageSkill(ctx.Session, skill),
			Selectable: isSelectable,
			Learned:    display.Level > 0,
			Tip:        skillWebTipLines(ctx, skill),
		})
	}
	return cells
}

func skillLevelText(display session.Skill, selected int, selectable bool) string {
	if selectable {
		return fmt.Sprintf("%d/%d", selected, display.Level)
	}
	return strconv.Itoa(display.Level)
}

func skillWebTipLines(ctx Context, skill session.Skill) []string {
	return strings.Split(skillTooltipText(ctx, skill), "\n")
}
