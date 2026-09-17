package ui

import (
	"fmt"
	"image"
	"sort"
	"strings"
	"time"

	"github.com/gogpu/ui/core/scrollview"
	"github.com/gogpu/ui/event"
	"github.com/gogpu/ui/primitives"
	"github.com/gogpu/ui/state"
	"github.com/gogpu/ui/widget"
	"github.com/kivutar/goro/res"
	"github.com/kivutar/goro/session"
	"github.com/kivutar/goro/ui/rotheme"
)

const (
	questWindowW = 350
	questWindowH = 310
	questTabW    = 34
	questBodyH   = questWindowH - ROWindowTitleHeight - ROWindowFooterHeight
	questDetailW = 360
	questDetailH = 390
)

// QuestWindow owns the legacy journal and its separate detail window.
// It displays session state; toggles only request a change from the server.
type QuestWindow struct {
	Window
	detail        Window
	tab           int
	selectedID    uint32
	detailID      uint32
	rows          []session.Quest
	listVersion   uint64
	detailVersion uint64
	scrollY       state.Signal[float32]
	detailScrollY state.Signal[float32]
	selectedRow   state.Signal[int]
	deadlineText  state.Signal[string]
	deadline      uint32
	lastSecond    int64
	images        map[string]image.Image
	onSetActive   func(uint32, bool)
}

func (w *QuestWindow) Toggle(ctx Context) {
	if w.IsOpen() {
		w.Close()
		return
	}
	w.OpenWindow(ctx)
}

func (w *QuestWindow) OpenWindow(ctx Context) {
	w.EnsureWindow(questWindowW, questWindowH)
	if w.scrollY == nil {
		w.scrollY = state.NewSignal[float32](0)
		w.selectedRow = state.NewSignal(-1)
	}
	w.refreshRows(ctx)
	w.Window.Open(ctx, w.widgetTree(ctx))
	w.Publish(ctx)
}

func (w *QuestWindow) Rebind(ctx Context, onSetActive func(uint32, bool)) {
	w.onSetActive = onSetActive
	if w.IsOpen() {
		w.refreshRows(ctx)
		w.RebindContent(ctx, w.widgetTree(ctx))
	}
	if w.detail.IsOpen() {
		if quest, ok := ctx.Session.Quests.Entries[w.detailID]; ok {
			w.detail.RebindContent(ctx, w.detailTree(ctx, quest))
		} else {
			w.detail.Close()
		}
	}
}

// Called before pointer/modal dispatch, so hovering a different window cannot
// hold up quest updates. Idle and closed journals do no list/asset work.
func (w *QuestWindow) UpdatePresentation(ctx Context, now time.Time) {
	if (!w.IsOpen() && !w.detail.IsOpen()) || ctx.Session == nil {
		return
	}
	if w.IsOpen() && w.listVersion != ctx.Session.Quests.Version {
		w.refreshRows(ctx)
		w.SetContent(w.widgetTree(ctx))
		w.Publish(ctx)
	}
	if w.detail.IsOpen() && w.detailVersion != ctx.Session.Quests.Version {
		if quest, ok := ctx.Session.Quests.Entries[w.detailID]; ok {
			w.detail.SetContent(w.detailTree(ctx, quest))
			w.detail.Publish(ctx)
		} else {
			w.detail.Close()
		}
	}
	if w.detail.IsOpen() && w.deadline != 0 && now.Unix() != w.lastSecond {
		w.lastSecond = now.Unix()
		w.deadlineText.Set(questTimeRemaining(w.deadline, now))
	}
}

func (w *QuestWindow) Update(ctx Context) bool {
	if w.detail.IsOpen() {
		consumed := w.detail.Update(ctx)
		w.detail.Publish(ctx)
		if consumed {
			return true
		}
	}
	if !w.IsOpen() {
		return false
	}
	consumed := w.Window.Update(ctx)
	w.Publish(ctx)
	return consumed
}

func (w *QuestWindow) refreshRows(ctx Context) {
	w.rows = w.rows[:0]
	if ctx.Session != nil {
		w.listVersion = ctx.Session.Quests.Version
		for _, quest := range ctx.Session.Quests.Entries {
			if w.tab == 2 || quest.Active == (w.tab == 0) {
				w.rows = append(w.rows, quest)
			}
		}
	}
	sort.Slice(w.rows, func(i, j int) bool { return w.rows[i].ID < w.rows[j].ID })
	selected := -1
	for i, quest := range w.rows {
		if quest.ID == w.selectedID {
			selected = i
			break
		}
	}
	if selected < 0 && len(w.rows) > 0 {
		selected = 0
	}
	w.selectRow(selected)
}

func (w *QuestWindow) selectRow(row int) {
	w.selectedID = 0
	if row >= 0 && row < len(w.rows) {
		w.selectedID = w.rows[row].ID
	}
	w.selectedRow.Set(row)
}

func (w *QuestWindow) widgetTree(ctx Context) widget.Widget {
	tabs := make([]widget.Widget, 0, 3)
	for tab, label := range []string{"Active", "Inactive", "All"} {
		tabs = append(tabs, newTabWidget(tabWidgetConfig{
			label: label, labelRotation: rotheme.TextRotationCounterClockwise,
			active: tab == w.tab, width: questTabW, height: 70,
			onClick: func() {
				if w.tab == tab {
					return
				}
				w.tab = tab
				w.scrollY.Set(0)
				w.refreshRows(ctx)
				w.SetContent(w.widgetTree(ctx))
				w.Publish(ctx)
			},
		}))
	}
	return Win(Title("Quest"), CloseButton(true), OnClose(w.Close), Size(questWindowW, questWindowH),
		Content(verticalTabFrame(
			primitives.Box(tabs...).Width(questTabW).Height(questBodyH).Gap(-1),
			primitives.Box(w.questTable(ctx)).Width(questWindowW-questTabW-verticalTabDividerW).Height(questBodyH),
		)),
		Footer(primitives.Expanded(primitives.Box()),
			rotheme.ButtonDisabledFn("View", func() bool { return w.selectedRow.Get() < 0 }, func() { w.openDetail(ctx, w.selectedID) }),
			rotheme.Button("Close", w.Close),
		),
	)
}

func (w *QuestWindow) questTable(ctx Context) widget.Widget {
	// Capture immutable display rows, not the live session, for widget callbacks.
	rows := append([]session.Quest(nil), w.rows...)
	titles := make([]string, len(rows))
	icons := make([]image.Image, len(rows))
	for i, quest := range rows {
		meta := questMetadata(ctx, quest.ID)
		titles[i] = meta.Title
		icons[i] = w.questImage(ctx, meta.Icon)
	}
	return rotheme.TableView(
		rotheme.TableViewColumns([]rotheme.TableViewColumn{{Key: "quest", Flex: 1}}),
		rotheme.TableViewShowHeader(false), rotheme.TableViewRowCount(len(rows)), rotheme.TableViewRowHeight(32),
		rotheme.TableViewEmptyText("No quests"), rotheme.TableViewScrollYSignal(w.scrollY), rotheme.TableViewSelectedRow(w.selectedRow),
		rotheme.TableViewDispatchHoverToCells(false),
		rotheme.TableViewBuildSimpleCell(func(cell rotheme.TableViewCellContext) rotheme.TableViewSimpleCell {
			return rotheme.TableViewSimpleCell{Icon: icons[cell.Row], Text: titles[cell.Row]}
		}),
		rotheme.TableViewOnRowClick(w.selectRow),
		rotheme.TableViewOnRowEvent(func(row int, e event.Event) bool {
			mouse, ok := e.(*event.MouseEvent)
			if !ok {
				return false
			}
			if mouse.MouseType == event.MousePress && mouse.Button == event.ButtonRight {
				quest := rows[row]
				if w.onSetActive != nil {
					w.onSetActive(quest.ID, !quest.Active)
				}
				return true
			}
			if mouse.MouseType == event.MouseDoubleClick && mouse.Button == event.ButtonLeft {
				w.openDetail(ctx, rows[row].ID)
				return true
			}
			return false
		}),
	)
}

func (w *QuestWindow) openDetail(ctx Context, id uint32) {
	if ctx.Session == nil {
		return
	}
	quest, ok := ctx.Session.Quests.Entries[id]
	if !ok {
		return
	}
	w.detail.EnsureWindow(questDetailW, questDetailH)
	if !w.detail.positioned {
		w.detail.placeNew(ctx, w.x+w.width+8, w.y)
	}
	if w.detailScrollY == nil {
		w.detailScrollY = state.NewSignal[float32](0)
	}
	if w.detailID != id {
		w.detailScrollY.Set(0)
	}
	w.detailID = id
	w.detail.Open(ctx, w.detailTree(ctx, quest))
	w.detail.Publish(ctx)
}

func (w *QuestWindow) detailTree(ctx Context, quest session.Quest) widget.Widget {
	w.detailVersion = ctx.Session.Quests.Version
	meta := questMetadata(ctx, quest.ID)
	content := []widget.Widget{rotheme.Label(meta.Title).MaxLines(1)}
	if img := w.questImage(ctx, meta.Image); img != nil {
		content = append(content, newStaticImageWidget(img, img.Bounds().Dx(), img.Bounds().Dy()))
	}
	if meta.Summary != "" {
		content = append(content, rotheme.Label("Objective"), questDescription(meta.Summary))
	}
	content = append(content, questDescription(meta.Description))
	if len(quest.Objectives) > 0 {
		content = append(content, rotheme.Label("Hunting"))
		for _, objective := range quest.Objectives {
			content = append(content, rotheme.Text(questObjectiveText(objective)).MaxLines(1))
		}
	}
	w.deadline = quest.ExpiresAt
	if w.deadlineText == nil {
		w.deadlineText = state.NewSignal("")
	}
	w.deadlineText.Set(questTimeRemaining(w.deadline, time.Now()))
	if w.deadline != 0 {
		content = append(content, rotheme.Text("").ContentSignal(w.deadlineText))
	}
	return Win(Title("Quest Details"), CloseButton(true), OnClose(w.detail.Close), Size(questDetailW, questDetailH),
		Content(primitives.Box(scrollview.New(
			primitives.Box(content...).Gap(8).PaddingRight(ROScrollbarGutter),
			scrollview.DirectionOpt(scrollview.Vertical), scrollview.ScrollbarOpt(scrollview.ScrollbarAuto),
			scrollview.ScrollYSignal(w.detailScrollY),
		)).Padding(10)),
		Footer(primitives.Expanded(primitives.Box()), rotheme.Button("Close", w.detail.Close)),
	)
}

func questMetadata(ctx Context, id uint32) res.QuestMetadata {
	meta, _ := ctx.Resources.QuestMetadata(id)
	meta.Title = stripItemInfoColorCodes(meta.Title)
	if meta.Title == "" {
		meta.Title = fmt.Sprintf("Quest #%d", id)
	}
	if meta.Description == "" {
		meta.Description = "No description available."
	}
	if meta.Icon == "" {
		meta.Icon = "SG_FEEL"
	}
	if meta.Image == "" {
		meta.Image = "QUE_NOIMAGE"
	}
	return meta
}

func (w *QuestWindow) questImage(ctx Context, name string) image.Image {
	if ctx.Resources == nil {
		return nil
	}
	if img, ok := w.images[name]; ok {
		return img
	}
	if w.images == nil {
		w.images = make(map[string]image.Image)
	}
	img, _, _ := res.LoadImage(ctx.Resources, res.ItemIconTextureCandidates(name))
	w.images[name] = img // Cache missing optional artwork too.
	return img
}

func questDescription(text string) widget.Widget {
	var lines []itemInfoTextLine
	for _, line := range strings.Split(text, "\n") {
		lines = append(lines, parseItemInfoTextLine(line))
	}
	var children []widget.Widget
	for _, line := range wrapItemInfoTextLines(lines, questDescriptionRunes) {
		children = append(children, itemInfoTextLineWidget(line))
	}
	return primitives.Box(children...)
}

const questDescriptionRunes = 40

func questObjectiveText(objective session.QuestObjective) string {
	name := objective.Name
	if name == "" {
		name = fmt.Sprintf("Monster #%d", objective.MonsterID)
	}
	if objective.Required == 0 {
		return fmt.Sprintf("%s: %d", name, objective.Current)
	}
	return fmt.Sprintf("%s: %d / %d", name, objective.Current, objective.Required)
}

func questTimeRemaining(deadline uint32, now time.Time) string {
	seconds := int64(deadline) - now.Unix()
	if seconds <= 0 {
		return "Time limit expired"
	}
	return fmt.Sprintf("Time left: %02d:%02d:%02d", seconds/3600, seconds/60%60, seconds%60)
}
