package ui

import (
	"fmt"
	"image/color"
	"math"
	"strconv"
	"strings"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/db"
	"github.com/kivutar/goro/input"
	"github.com/kivutar/goro/render"
	"github.com/kivutar/goro/session"
)

// The character HUD (name, HP/SP, EXP, weight, zeny) is drawn straight into
// the game frame from cached surfaces and GPU-cached text labels instead of
// a retained ui-widget window: every weight/zeny change (each pickup, each
// regen tick) rebuilt and re-rasterized the whole widget tree, a 60-90ms
// CPU spike that read as a hitch on tablets. The direct-draw version pays
// one cached-image blit per element per frame and never invalidates the ui
// canvas. Dragging and the close button are hit-tested by hand.

const (
	characterWindowX      = windowScreenMargin
	characterWindowY      = windowScreenMargin
	characterWindowWidth  = 324
	characterWindowHeight = 134

	characterHUDWidth      = 324
	characterHUDTitleH     = 18
	characterHUDHeight     = 134
	characterHUDPadX       = 10
	characterHUDPadY       = 8
	characterHUDRowGap     = 6
	characterHUDBarH       = 7
	characterHUDBarGap     = 3
	characterHUDTextH      = 13
	characterHUDExpBarH    = 6
	characterHUDCloseSize  = 17
	characterHUDStatColumn = 146
	characterHUDTextSize  = 11

	hudEdgeTabW = 22
	hudEdgeTabH = 36
	characterHUDExpLabelW  = 64
)

var (
	characterHUDHPColor       = PlayerHPBarColor
	characterHUDSPColor       = PlayerSPBarColor
	characterHUDTextColor     = color.RGBA{R: 235, G: 242, B: 250, A: 255}
	characterHUDMutedColor    = color.RGBA{R: 190, G: 200, B: 214, A: 255}
	characterHUDBarBackColor  = color.RGBA{R: 64, G: 70, B: 82, A: 160}
	characterHUDEXPColor      = color.RGBA{R: 170, G: 182, B: 200, A: 255}
	characterHUDBackground    = color.RGBA{R: 14, G: 18, B: 24, A: 189}
	characterHUDPanelBack     = color.RGBA{R: 255, G: 255, B: 255, A: 18}
	characterHUDBorder        = color.RGBA{R: 180, G: 198, B: 218, A: 94}
	characterHUDRadius        = float32(8)
)

// CharacterWindow is the always-on character HUD overlay. It keeps the
// field surface (position, open state, drag flag) that the basic menu
// follows below it.
type CharacterWindow struct {
	open      bool
	dismissed bool
	x         int
	y         int
	width     int
	height    int
	dragLayer bool
	dragOffX  int
	dragOffY  int
	// dragBottom keeps a manual drag from overlapping the basic menu.
	dragBottom int
}

func (w *CharacterWindow) IsOpen() bool {
	return w != nil && w.open
}

// Close hides the HUD until the next login.
func (w *CharacterWindow) Close() {
	if w == nil {
		return
	}
	w.open = false
	w.dismissed = true
	w.dragLayer = false
}

func clampHUDInt(value, lo, hi int) int {
	if value < lo {
		return lo
	}
	if value > hi {
		return hi
	}
	return value
}

// setPosition moves the HUD (tests and programmatic placement).
func (w *CharacterWindow) setPosition(_ client.Context, x, y int) {
	if w == nil {
		return
	}
	w.x, w.y = x, y
}

func (w *CharacterWindow) Update(ctx client.Context) bool {
	if w == nil || ctx.Input == nil || ctx.Session == nil {
		return false
	}
	if !w.open {
		if !w.dismissed {
			w.open = true
		} else {
			if ctx.Input.MouseJustPressed(input.MouseButtonLeft) && characterEdgeTabHit(ctx.Input.MouseX, ctx.Input.MouseY) {
				w.open = true
				w.dismissed = false
				return true
			}
			return false
		}
	}
	screenW, screenH := ctx.ScreenSize()
	if w.width == 0 {
		w.width = characterHUDWidth
		w.height = characterHUDHeight
		w.x = minInt(characterWindowX, maxInt(0, screenW-characterHUDWidth))
		w.y = characterWindowY
	}
	_, menuHeight := basicMenuSize()
	w.dragBottom = basicMenuFollowGap + menuHeight
	mouseX, mouseY := ctx.Input.MouseX, ctx.Input.MouseY
	inside := pointInRect(mouseX, mouseY, w.x, w.y, w.width, w.height)
	titleRect := [4]int{w.x, w.y, w.width, characterHUDTitleH}
	closeRect := [4]int{w.x + w.width - characterHUDCloseSize - 3, w.y + 1, characterHUDCloseSize, characterHUDCloseSize}

	if w.dragLayer {
		if ctx.Input.MousePressed(input.MouseButtonLeft) {
			w.x = clampHUDInt(mouseX-w.dragOffX, 0, maxInt(0, screenW-w.width))
			maxY := maxInt(0, screenH-w.height-w.dragBottom)
			w.y = clampHUDInt(mouseY-w.dragOffY, 0, maxY)
			return true
		}
		w.dragLayer = false
		return inside
	}
	if !ctx.Input.MouseJustPressed(input.MouseButtonLeft) || !inside {
		return false
	}
	if pointInRect(mouseX, mouseY, closeRect[0], closeRect[1], closeRect[2], closeRect[3]) {
		w.Close()
		return true
	}
	if pointInRect(mouseX, mouseY, titleRect[0], titleRect[1], titleRect[2], titleRect[3]) {
		w.dragLayer = true
		w.dragOffX = mouseX - w.x
		w.dragOffY = mouseY - w.y
		return true
	}
	return true
}

// Draw renders the HUD from cached surfaces and GPU-cached text labels.
// Called every frame from the world's UI overlay pass; the game frame is
// fully redrawn each frame anyway, so this costs a handful of blits.
func (w *CharacterWindow) Draw(screen *render.Frame, ctx client.Context) {
	if w == nil || !w.open || screen == nil || ctx.Session == nil {
		return
	}
	if w.width == 0 {
		return
	}
	if w.dismissed {
		w.drawEdgeTab(screen)
		return
	}
	character, vitals, progress, inventory := characterWindowData(ctx.Session)
	name := strings.TrimSpace(character.Name)
	if name == "" {
		name = "Player"
	}
	jobName := strings.TrimSpace(db.JobDisplayName(int(character.Job)))
	title := trimRunes(name, 20)
	if jobName != "" {
		title = trimRunes(fmt.Sprintf("%s (%s)", name, jobName), 32)
	}

	x, y := w.x, w.y
	// Translucent rounded panel in the DOM chat log's style (same radius,
	// background, and border) so every overlay reads as one family.
	DrawRoundedSurface(screen, x, y, w.width, w.height, characterHUDBackground, characterHUDBorder, characterHUDRadius)
	if titleW := render.MeasureUIText(title, characterHUDTextSize); titleW > 0 {
		render.DrawUITextAtSize(screen, title, float64(x+(w.width-int(titleW))/2), float64(y+3), characterHUDTextColor, characterHUDTextSize)
	} else {
		render.DrawUITextAtSize(screen, title, float64(x+4), float64(y+3), characterHUDTextColor, characterHUDTextSize)
	}
	DrawCloseButton(screen, x+w.width-characterHUDCloseSize-4, y+1, characterHUDCloseSize-2, characterHUDCloseSize-2,
		characterHUDPanelBack, characterHUDMutedColor)

	contentW := w.width - 2*characterHUDPadX
	cx := x + characterHUDPadX
	cy := y + characterHUDTitleH + characterHUDPadY

	// HP / SP columns.
	halfW := (contentW - characterHUDRowGap) / 2
	weightColor := characterHUDTextColor
	if inventory.MaxWeight > 0 && inventory.Weight*100 >= inventory.MaxWeight*50 {
		weightColor = color.RGBA{R: ErrorTextColor.R, G: ErrorTextColor.G, B: ErrorTextColor.B, A: ErrorTextColor.A}
	}
	drawHUDBarsColumn(screen, cx, cy, halfW, "HP", vitals.HP, vitals.MaxHP, characterHUDHPColor)
	drawHUDBarsColumn(screen, cx+halfW+characterHUDRowGap, cy, halfW, "SP", vitals.SP, vitals.MaxSP, characterHUDSPColor)
	cy += characterHUDTextH + characterHUDBarGap + characterHUDBarH + characterHUDRowGap

	// EXP panel.
	panelH := 2*characterHUDTextH + characterHUDRowGap + 2*4
	DrawRoundedSurface(screen, cx-4, cy-4, contentW+8, panelH+8, characterHUDPanelBack, color.RGBA{}, 6)
	drawHUDExpRow(screen, cx, cy, contentW, "Base", progress.BaseLevel, progress.BaseExp, progress.NextBaseExp)
	cy += characterHUDTextH + 4
	drawHUDExpRow(screen, cx, cy, contentW, "Job", progress.JobLevel, progress.JobExp, progress.NextJobExp)
	cy += characterHUDTextH + characterHUDRowGap + 4

	// Weight / Zeny row.
	render.DrawUITextAtSize(screen, fmt.Sprintf("Weight : %d / %d", displayWeight(inventory.Weight), displayWeight(inventory.MaxWeight)),
		float64(cx), float64(cy), weightColor, characterHUDTextSize)
	zeny := fmt.Sprintf("Zeny : %s", formatHUDNumber(inventory.Zeny))
	if zenyW := int(render.MeasureUIText(zeny, characterHUDTextSize)); zenyW > 0 {
		render.DrawUITextAtSize(screen, zeny, float64(cx+contentW-zenyW), float64(cy), characterHUDTextColor, characterHUDTextSize)
	} else {
		render.DrawUITextAtSize(screen, zeny, float64(cx), float64(cy), characterHUDTextColor, characterHUDTextSize)
	}
}

func drawHUDBarsColumn(screen *render.Frame, x, y, width int, label string, current, maxValue int, fill color.RGBA) {
	render.DrawUITextAtSize(screen, fmt.Sprintf("%s %d / %d", label, current, maxValue),
		float64(x), float64(y), characterHUDMutedColor, characterHUDTextSize)
	barY := y + characterHUDTextH + characterHUDBarGap
	render.DrawRect(screen, float64(x), float64(barY), float64(width), float64(characterHUDBarH), characterHUDBarBackColor)
	if ratio := ratioInt(current, maxValue); ratio > 0 {
		fillW := int(math.Round(float64(width) * ratio))
		if fillW < 1 {
			fillW = 1
		}
		render.DrawRect(screen, float64(x), float64(barY), float64(fillW), float64(characterHUDBarH), fill)
	}
}

func drawHUDExpRow(screen *render.Frame, x, y, width int, label string, level int, current, next int64) {
	render.DrawUITextAtSize(screen, fmt.Sprintf("%s Lv. %d", label, level), float64(x), float64(y), characterHUDTextColor, characterHUDTextSize)
	barX := x + characterHUDExpLabelW
	barW := width - characterHUDExpLabelW
	if barW <= 0 {
		return
	}
	barY := y + (characterHUDTextH-characterHUDExpBarH)/2
	render.DrawRect(screen, float64(barX), float64(barY), float64(barW), float64(characterHUDExpBarH), characterHUDBarBackColor)
	if ratio := ratioInt64(current, next); ratio > 0 {
		fillW := int(math.Round(float64(barW) * ratio))
		if fillW < 1 {
			fillW = 1
		}
		render.DrawRect(screen, float64(barX), float64(barY), float64(fillW), float64(characterHUDExpBarH), characterHUDEXPColor)
	}
	percent := formatEXPPercent(current, next)
	if percentW := int(render.MeasureUIText(percent, characterHUDTextSize)); percentW > 0 && barW-percentW-4 > 0 {
		render.DrawUITextAtSize(screen, percent, float64(barX+barW-percentW-4), float64(y), characterHUDMutedColor, characterHUDTextSize)
	}
}

// characterEdgeTabRect is the flush-left tab shown while the HUD is
// dismissed; clicking it reopens the window.
func characterEdgeTabRect() (int, int, int, int) {
	return 0, 8, hudEdgeTabW, hudEdgeTabH
}

// drawEdgeTab renders a small left-edge tab with a mini HP/SP bar glyph so
// the reopen affordance matches what the window shows.
func (w *CharacterWindow) drawEdgeTab(screen *render.Frame) {
	tx, ty, tw, th := characterEdgeTabRect()
	DrawRoundedSurface(screen, tx, ty, tw, th, characterHUDBackground, characterHUDBorder, characterHUDRadius)
	glyphX := tx + tw/2 - 5
	render.DrawRect(screen, float64(glyphX), float64(ty+9), 10, 4, characterHUDHPColor)
	render.DrawRect(screen, float64(glyphX), float64(ty+16), 7, 4, characterHUDSPColor)
	render.DrawRect(screen, float64(glyphX), float64(ty+23), 9, 3, characterHUDEXPColor)
}

func characterEdgeTabHit(mouseX, mouseY int) bool {
	tx, ty, tw, th := characterEdgeTabRect()
	return pointInRect(mouseX, mouseY, tx, ty, tw, th)
}

func characterWindowData(s *session.Session) (session.Character, session.Vitals, session.Progress, session.Inventory) {
	character := selectedCharacter(s)
	if s == nil {
		return character, session.Vitals{}, session.Progress{}, session.Inventory{}
	}
	vitals := s.Vitals
	if vitals.HP == 0 && vitals.MaxHP == 0 && vitals.SP == 0 && vitals.MaxSP == 0 {
		vitals = sessionVitalsFromCharacter(character)
	}
	progress := s.Progress
	if progress.BaseLevel == 0 {
		progress = sessionProgressFromCharacter(character)
		progress.BaseExp = s.Progress.BaseExp
		progress.NextBaseExp = s.Progress.NextBaseExp
		progress.JobExp = s.Progress.JobExp
		progress.NextJobExp = s.Progress.NextJobExp
	}
	if progress.JobLevel == 0 && character.JobLevel > 0 {
		progress.JobLevel = int(character.JobLevel)
	}
	return character, vitals, progress, s.Inventory
}

func displayWeight(raw int) int {
	return raw / 10
}

func DisplayWeight(raw int) int {
	return displayWeight(raw)
}

func ratioInt(current, maxValue int) float64 {
	if maxValue <= 0 {
		return 0
	}
	return clampUnit(float64(current) / float64(maxValue))
}

func ratioInt64(current, maxValue int64) float64 {
	if maxValue <= 0 {
		return 0
	}
	return clampUnit(float64(current) / float64(maxValue))
}

func sessionVitalsFromCharacter(character session.Character) session.Vitals {
	return session.Vitals{
		HP:    int(character.HP),
		MaxHP: int(character.MaxHP),
		SP:    int(character.SP),
		MaxSP: int(character.MaxSP),
	}
}

func sessionProgressFromCharacter(character session.Character) session.Progress {
	return session.Progress{
		BaseLevel: int(character.Level),
		JobLevel:  int(character.JobLevel),
	}
}

func formatEXPPercent(current, next int64) string {
	if next <= 0 {
		return "--"
	}
	percent := 100 * float64(current) / float64(next)
	if percent < 0 {
		percent = 0
	}
	if percent > 100 {
		percent = 100
	}
	return fmt.Sprintf("%.1f%%", math.Floor(percent*10)/10)
}

func FormatEXPPercent(current, next int64) string {
	return formatEXPPercent(current, next)
}

func formatHUDNumber(value int64) string {
	sign := ""
	if value < 0 {
		sign = "-"
		value = -value
	}
	text := strconv.FormatInt(value, 10)
	if len(text) <= 3 {
		return sign + text
	}
	var b strings.Builder
	prefix := len(text) % 3
	if prefix == 0 {
		prefix = 3
	}
	b.WriteString(text[:prefix])
	for i := prefix; i < len(text); i += 3 {
		b.WriteByte(',')
		b.WriteString(text[i : i+3])
	}
	return sign + b.String()
}

func FormatHUDNumber(value int64) string {
	return formatHUDNumber(value)
}
