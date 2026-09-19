package ui

import (
	"fmt"
	"image/color"
	"math"

	"github.com/kivutar/goro/render"
)

// MiniVitalsHUD is the small always-on corner readout for the handheld: HP,
// SP, Base/Job EXP, and Weight, each with a thin bar. Drawn straight into
// the game frame from GPU-cached text like the old character HUD, so it never
// rebuilds a widget tree and never invalidates the UI canvas.
const (
	miniHUDWidth = 168
	// Height = pad + 4 bar rows + the weight text line + pad. The old 100
	// left the weight line outside the panel.
	miniHUDHeight = 8 + 4*(15+3+4+4) + 15 + 8
	miniHUDPad    = 8
	miniHUDTextH  = 15
	miniHUDTextSz = 12
	miniHUDBarH   = 4
	miniHUDBarGap = 3
	miniHUDRowH   = miniHUDTextH + miniHUDBarGap + miniHUDBarH + 4
)

var (
	miniHUDBarBack    = color.RGBA{R: 40, G: 44, B: 54, A: 190}
	miniHUDBackground = color.RGBA{R: 14, G: 18, B: 24, A: 200}
	miniHUDBorder     = color.RGBA{R: 160, G: 175, B: 195, A: 90}
	miniHUDText       = color.RGBA{R: 244, G: 248, B: 252, A: 255}
	miniHUDOutline    = color.RGBA{R: 10, G: 12, B: 16, A: 210}
	miniHUDWeightOver = color.RGBA{R: 245, G: 120, B: 120, A: 255}
	miniHUDHPColor    = PlayerHPBarColor
	miniHUDSPColor    = PlayerSPBarColor
	miniHUDExpColor   = color.RGBA{R: 120, G: 170, B: 235, A: 255}
	miniHUDJobColor   = color.RGBA{R: 235, G: 190, B: 110, A: 255}
)

// MiniVitalsHUD draws the readout; it is stateless.
type MiniVitalsHUD struct{}

// Draw renders the readout at the top-left corner.
func (h *MiniVitalsHUD) Draw(screen *render.Frame, hp, maxHP, sp, maxSP int, baseExp, nextBaseExp, jobExp, nextJobExp int64, weight, maxWeight int) {
	if screen == nil {
		return
	}
	x, y := windowScreenMargin, windowScreenMargin
	DrawRoundedSurface(screen, x, y, miniHUDWidth, miniHUDHeight, miniHUDBackground, miniHUDBorder, 6)
	cx := x + miniHUDPad
	cy := y + miniHUDPad
	w := miniHUDWidth - 2*miniHUDPad

	weightColor := miniHUDText
	if maxWeight > 0 && weight*100 >= maxWeight*50 {
		weightColor = miniHUDWeightOver
	}

	h.row(screen, cx, cy, w, fmt.Sprintf("HP %d/%d", hp, maxHP), hp, maxHP, miniHUDHPColor)
	cy += miniHUDRowH
	h.row(screen, cx, cy, w, fmt.Sprintf("SP %d/%d", sp, maxSP), sp, maxSP, miniHUDSPColor)
	cy += miniHUDRowH
	h.row(screen, cx, cy, w, fmt.Sprintf("Base %.2f%%", expPercent(baseExp, nextBaseExp)), int(clampInt64(baseExp, 0, 1<<31-1)), int(clampInt64(nextBaseExp, 1, 1<<31-1)), miniHUDExpColor)
	cy += miniHUDRowH
	h.row(screen, cx, cy, w, fmt.Sprintf("Job %.2f%%", expPercent(jobExp, nextJobExp)), int(clampInt64(jobExp, 0, 1<<31-1)), int(clampInt64(nextJobExp, 1, 1<<31-1)), miniHUDJobColor)
	cy += miniHUDRowH
	render.DrawUIOutlinedTextAt(screen, fmt.Sprintf("Weight %d/%d", displayWeight(weight), displayWeight(maxWeight)),
		float64(cx), float64(cy), weightColor, miniHUDOutline)
}

func (h *MiniVitalsHUD) row(screen *render.Frame, x, y, w int, label string, current, maxValue int, fill color.RGBA) {
	render.DrawUIOutlinedTextAt(screen, label, float64(x), float64(y), miniHUDText, miniHUDOutline)
	barY := y + miniHUDTextH + miniHUDBarGap
	render.DrawRect(screen, float64(x), float64(barY), float64(w), float64(miniHUDBarH), miniHUDBarBack)
	if ratio := ratioInt(current, maxValue); ratio > 0 {
		fillW := int(math.Round(float64(w) * ratio))
		if fillW < 1 {
			fillW = 1
		}
		render.DrawRect(screen, float64(x), float64(barY), float64(fillW), float64(miniHUDBarH), fill)
	}
}

func expPercent(current, next int64) float64 {
	if next <= 0 {
		return 0
	}
	ratio := float64(current) / float64(next)
	if ratio < 0 {
		ratio = 0
	}
	if ratio > 1 {
		ratio = 1
	}
	return ratio * 100
}

func clampInt64(v, lo, hi int64) int64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
