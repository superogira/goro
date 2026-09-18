package ui

import (
	"image"

	"github.com/gogpu/ui/primitives"
	"github.com/gogpu/ui/widget"
	"github.com/kivutar/goro/session"
	"github.com/kivutar/goro/ui/rotheme"
)

// SkillDetailWindow is a small standalone window that shows one skill's
// tooltip content (level, SP cost, range, description) on handhelds, where
// there is no mouse to hover the tree cells with. Opened with the Y button
// from the skill window's gamepad controls.
const (
	skillDetailWindowWidth = 300
	skillDetailWindowPad   = 10
	skillDetailIconSize    = 28
)

type SkillDetailWindow struct {
	Window
	skill session.Skill
	name  string
	lines []string
	icon  image.Image
}

// openSkill (re)builds the detail window for the given skill. Repeated calls
// refresh the content in place.
func (w *SkillDetailWindow) openSkill(ctx Context, skill session.Skill, icon image.Image, x, y int) {
	if skill.ID == 0 {
		return
	}
	w.EnsureWindow(skillDetailWindowWidth, itemInfoWindowMaxHeight)
	w.skill = skill
	w.name = trimRunes(skillDisplayName(ctx.Resources, skill), 34)
	w.lines = skillTooltipLines(ctx, skill)
	w.icon = icon

	height := skillDetailWindowHeight(len(w.lines), icon != nil)
	screenW, screenH := ctx.ScreenSize()
	x = clampWindowInt(x, windowScreenMargin, maxInt(windowScreenMargin, screenW-skillDetailWindowWidth-windowScreenMargin))
	y = clampWindowInt(y, windowScreenMargin, maxInt(windowScreenMargin, screenH-height-windowScreenMargin))
	w.SetSize(skillDetailWindowWidth, height)
	w.OpenAt(x, y, w.widgetTree(ctx))
	w.Publish(ctx)
}

func skillDetailWindowHeight(lineCount int, hasIcon bool) int {
	height := ROWindowTitleHeight + skillDetailWindowPad*2 + maxInt(1, lineCount)*itemInfoLineH
	if hasIcon {
		height += skillDetailIconSize + skillDetailWindowPad
	}
	return minInt(height, itemInfoWindowMaxHeight)
}

func (w *SkillDetailWindow) widgetTree(ctx Context) widget.Widget {
	children := make([]widget.Widget, 0, len(w.lines)+1)
	if w.icon != nil {
		children = append(children,
			primitives.Box(newStaticImageWidget(w.icon, skillDetailIconSize, skillDetailIconSize)).
				Height(skillDetailIconSize).
				Width(skillDetailIconSize),
		)
	}
	for _, line := range w.lines {
		if line == "" {
			children = append(children, rotheme.Text(" ").
				Color(itemInfoWidgetColor(inventoryTextColor)).
				LineHeight(itemInfoLineH / rotheme.Default.Typography.TextSize))
			continue
		}
		children = append(children, rotheme.Text(line).
			Color(itemInfoWidgetColor(inventoryTextColor)).
			LineHeight(itemInfoLineH / rotheme.Default.Typography.TextSize))
	}
	return Win(
		Title(w.name),
		CloseButton(true),
		OnClose(func() {
			w.Window.Close()
			w.Publish(ctx)
		}),
		Size(skillDetailWindowWidth, float32(skillDetailWindowHeight(len(w.lines), w.icon != nil))),
		Content(
			primitives.Box(children...).
				Padding(skillDetailWindowPad).
				Gap(2),
		),
	)
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
