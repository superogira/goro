package ui

import (
	"image"

	"github.com/gogpu/ui/core/scrollview"
	"github.com/gogpu/ui/primitives"
	"github.com/gogpu/ui/state"
	"github.com/gogpu/ui/widget"
	"github.com/kivutar/goro/session"
	"github.com/kivutar/goro/ui/rotheme"
)

// SkillDetailWindow is a small standalone window that shows one skill's
// tooltip content (level, SP cost, range, description) on handhelds, where
// there is no mouse to hover the tree cells with. Opened with the Y button
// from the skill window's gamepad controls. Long descriptions scroll with the
// d-pad (GamepadScroll).
const (
	skillDetailWindowWidth = 300
	skillDetailWindowPad   = 10
	skillDetailIconSize    = 28
	skillDetailIconRowH    = skillDetailIconSize + 4
	skillDetailMaxHeight   = 340
	skillDetailScrollStep  = itemInfoLineH * 3
)

type SkillDetailWindow struct {
	Window
	skill      session.Skill
	name       string
	lines      []string
	icon       image.Image
	scrollY    state.Signal[float32]
	viewHeight float32
	maxScroll  float32
}

// openSkill (re)builds the detail window for the given skill. Repeated calls
// refresh the content in place.
func (w *SkillDetailWindow) openSkill(ctx Context, skill session.Skill, icon image.Image, x, y int) {
	if skill.ID == 0 {
		return
	}
	w.EnsureWindow(skillDetailWindowWidth, skillDetailMaxHeight)
	w.skill = skill
	w.name = trimRunes(skillDisplayName(ctx.Resources, skill), 34)
	w.lines = skillTooltipLines(ctx, skill)
	w.icon = icon
	if w.scrollY != nil {
		w.scrollY.Set(0)
	}

	w.layout()

	height := int(w.viewHeight) + ROWindowTitleHeight + skillDetailWindowPad*2
	screenW, screenH := ctx.ScreenSize()
	x = clampWindowInt(x, windowScreenMargin, maxInt(windowScreenMargin, screenW-skillDetailWindowWidth-windowScreenMargin))
	y = clampWindowInt(y, windowScreenMargin, maxInt(windowScreenMargin, screenH-height-windowScreenMargin))
	w.SetSize(skillDetailWindowWidth, height)
	w.OpenAt(x, y, w.widgetTree(ctx))
	w.Publish(ctx)
}

// layout computes the viewport height and scroll range from the current
// lines/icon. openSkill calls it; swapping lines and re-calling it reflows
// the window. The content tree gives every child an explicit height and no
// gaps, so the content height — and with it the scroll range — is exact
// rather than estimated: a low estimate left the last lines unreachable
// (the scrollbar claimed room the d-pad scroll refused to use).
func (w *SkillDetailWindow) layout() {
	w.ensureScrollSignal()
	contentHeight := float32(len(w.lines) * itemInfoLineH)
	if w.icon != nil {
		contentHeight += skillDetailIconRowH
	}
	viewHeight := minFloat32(contentHeight, skillDetailMaxHeight-ROWindowTitleHeight-skillDetailWindowPad*2)
	w.viewHeight = viewHeight
	w.maxScroll = maxFloat32(0, contentHeight-viewHeight)
	if w.scrollY != nil && w.scrollY.Get() > w.maxScroll {
		w.scrollY.Set(w.maxScroll)
	}
}

func (w *SkillDetailWindow) widgetTree(ctx Context) widget.Widget {
	// Every child carries an explicit height and the boxes have no gap, so
	// the laid-out content height matches the layout() arithmetic exactly —
	// the d-pad scroll clamp and the scrollview range stay in sync.
	children := make([]widget.Widget, 0, len(w.lines)+1)
	if w.icon != nil {
		children = append(children,
			primitives.Box(newStaticImageWidget(w.icon, skillDetailIconSize, skillDetailIconSize)).
				Height(skillDetailIconRowH).
				Width(skillDetailWindowWidth-ROScrollbarGutter),
		)
	}
	for _, line := range w.lines {
		text := line
		if text == "" {
			text = " "
		}
		children = append(children,
			primitives.Box(
				rotheme.Text(text).
					Color(itemInfoWidgetColor(inventoryTextColor)).
					LineHeight(itemInfoLineH/rotheme.Default.Typography.TextSize),
			).
				Height(itemInfoLineH).
				Width(skillDetailWindowWidth-skillDetailWindowPad*2-ROScrollbarGutter),
		)
	}
	body := primitives.Box(children...).Gap(0)
	if w.maxScroll > 0 {
		body = primitives.Box(
			scrollview.New(
				primitives.Box(body).PaddingRight(ROScrollbarGutter),
				scrollview.DirectionOpt(scrollview.Vertical),
				scrollview.ScrollbarOpt(scrollview.ScrollbarAuto),
				scrollview.ScrollYSignal(w.ensureScrollSignal()),
				scrollview.ScrollStep(itemInfoLineH),
			),
		)
	}
	return Win(
		Title(w.name),
		CloseButton(true),
		OnClose(func() {
			w.Window.Close()
			w.Publish(ctx)
		}),
		Size(skillDetailWindowWidth, float32(w.viewHeight+ROWindowTitleHeight+skillDetailWindowPad*2)),
		Content(
			primitives.Box(body).
				Height(w.viewHeight).
				Padding(skillDetailWindowPad),
		),
	)
}

func (w *SkillDetailWindow) ensureScrollSignal() state.Signal[float32] {
	if w.scrollY == nil {
		w.scrollY = state.NewSignal[float32](0)
	}
	return w.scrollY
}

// GamepadScroll scrolls the description text one step (dir < 0 up, dir > 0
// down). It reports false when the window is closed, the text fits without
// scrolling, or the scroll is already at the edge — the caller lets the d-pad
// fall through to skill navigation in that case.
func (w *SkillDetailWindow) GamepadScroll(dir int) bool {
	if w == nil || !w.IsOpen() || w.scrollY == nil || w.maxScroll <= 0 || dir == 0 {
		return false
	}
	value := w.scrollY.Get() + skillDetailScrollStep*float32(dir)
	if value < 0 {
		value = 0
	}
	if value > w.maxScroll {
		value = w.maxScroll
	}
	if value == w.scrollY.Get() {
		return false
	}
	w.scrollY.Set(value)
	return true
}

// Update pumps window events (the close button) and keeps the window
// published.
func (w *SkillDetailWindow) Update(ctx Context) bool {
	if !w.IsOpen() {
		return false
	}
	consumed := w.Window.Update(ctx)
	if !w.IsOpen() {
		w.Publish(ctx)
		return consumed
	}
	w.Publish(ctx)
	return consumed
}

// closeIfOpen closes the detail window; it reports whether a window was open.
func (w *SkillDetailWindow) closeIfOpen(ctx Context) bool {
	if w == nil || !w.IsOpen() {
		return false
	}
	w.Window.Close()
	w.Publish(ctx)
	return true
}

func minFloat32(a, b float32) float32 {
	if a < b {
		return a
	}
	return b
}

func maxFloat32(a, b float32) float32 {
	if a > b {
		return a
	}
	return b
}
