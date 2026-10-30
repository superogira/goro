package skills

import (
	"image"
	"image/color"
	"math"
	"strconv"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/game/scene"
	"github.com/kivutar/goro/render"
	"github.com/kivutar/goro/session"
)

const (
	pendingSkillCursorLevelOffsetX = 20
	pendingSkillCursorLevelOffsetY = -12
	pendingSkillCursorLevelScale   = 1.1
)

var (
	pendingSkillCursorLevelFallbackColor = color.RGBA{R: 255, G: 255, B: 255, A: 255}
	pendingSkillCursorLevelOutlineColor  = color.RGBA{R: 82, G: 96, B: 178, A: 255}
)

func (m *System) DrawPendingSkillCursorLevel(screen *render.Frame, ctx client.Context, skill session.Skill, magnetX, magnetY float64) {
	if screen == nil || ctx.Input == nil {
		return
	}
	text := pendingSkillCursorLevelText(skill)
	if text == "" {
		return
	}
	x, y := pendingSkillCursorLevelPosition(ctx.Input.MouseX, ctx.Input.MouseY, magnetX, magnetY)
	if billboard, ok := m.pendingSkillCursorLevelBillboard(ctx, text); ok {
		drawPendingSkillCursorLevelBillboard(screen, billboard, x, y)
		return
	}
	x, y = pendingSkillCursorLevelDrawOrigin(screen, x, y)
	render.DrawOutlinedTextAt(screen, text, int(math.Round(x)), int(math.Round(y)), pendingSkillCursorLevelFallbackColor, pendingSkillCursorLevelOutlineColor)
}

func pendingSkillCursorLevelText(skill session.Skill) string {
	if skill.ID == 0 || skill.Level <= 0 {
		return ""
	}
	return strconv.Itoa(skill.Level)
}

func pendingSkillCursorLevelPosition(mouseX, mouseY int, magnetX, magnetY float64) (float64, float64) {
	return float64(mouseX) + pendingSkillCursorLevelOffsetX - magnetX, float64(mouseY) + pendingSkillCursorLevelOffsetY - magnetY
}

func (m *System) pendingSkillCursorLevelBillboard(ctx client.Context, text string) (*scene.Billboard, bool) {
	if text == "" {
		return nil, false
	}
	if m.CursorLevelNums == nil {
		m.CursorLevelNums = make(map[string]*scene.Billboard)
	}
	if billboard, ok := m.CursorLevelNums[text]; ok {
		return billboard, true
	}
	raw, ok := m.bindings.NumberBillboard(ctx, text)
	if !ok {
		return nil, false
	}
	image := pendingSkillCursorLevelGlyphImage(raw.Image)
	if image == nil {
		return nil, false
	}
	billboard := &scene.Billboard{
		Image:   image,
		AnchorX: raw.AnchorX,
		AnchorY: raw.AnchorY,
	}
	m.CursorLevelNums[text] = billboard
	return billboard, true
}

func pendingSkillCursorLevelGlyphImage(src *render.Image) *render.Image {
	if src == nil || src.RGBA() == nil {
		return nil
	}
	bounds := src.Bounds()
	if bounds.Empty() {
		return nil
	}
	dst := image.NewRGBA(image.Rect(0, 0, bounds.Dx(), bounds.Dy()))
	pixels := src.RGBA()
	for y := 0; y < bounds.Dy(); y++ {
		for x := 0; x < bounds.Dx(); x++ {
			offset := pixels.PixOffset(x+bounds.Min.X, y+bounds.Min.Y)
			alpha := pixels.Pix[offset+3]
			if alpha == 0 {
				continue
			}
			brightness := maxRGBByte(pixels.Pix[offset], pixels.Pix[offset+1], pixels.Pix[offset+2])
			faceAlpha := uint8((uint16(alpha)*uint16(brightness) + 127) / 255)
			if faceAlpha == 0 {
				continue
			}
			dst.SetRGBA(x, y, color.RGBA{R: 255, G: 255, B: 255, A: faceAlpha})
		}
	}
	return render.NewImageFromImage(dst)
}

func drawPendingSkillCursorLevelBillboard(screen *render.Frame, billboard *scene.Billboard, x, y float64) {
	if screen == nil || billboard == nil || billboard.Image == nil {
		return
	}
	x, y = pendingSkillCursorLevelDrawOrigin(screen, x, y)
	outlineOffsets := [...]struct{ x, y float64 }{
		{-1, -1}, {0, -1}, {1, -1},
		{-1, 0}, {1, 0},
		{-1, 1}, {0, 1}, {1, 1},
	}
	outlineR := float32(pendingSkillCursorLevelOutlineColor.R) / 255
	outlineG := float32(pendingSkillCursorLevelOutlineColor.G) / 255
	outlineB := float32(pendingSkillCursorLevelOutlineColor.B) / 255
	outlineA := float32(pendingSkillCursorLevelOutlineColor.A) / 255
	for _, offset := range outlineOffsets {
		var outline render.DrawImageOptions
		outline.GeoM.Scale(pendingSkillCursorLevelScale, pendingSkillCursorLevelScale)
		outline.GeoM.Translate(x+offset.x, y+offset.y)
		outline.Filter = render.FilterLinear
		outline.ColorScale.Scale(outlineR, outlineG, outlineB, outlineA)
		screen.DrawImage(billboard.Image, &outline)
	}

	var opts render.DrawImageOptions
	opts.GeoM.Scale(pendingSkillCursorLevelScale, pendingSkillCursorLevelScale)
	opts.GeoM.Translate(x, y)
	opts.Filter = render.FilterLinear
	screen.DrawImage(billboard.Image, &opts)
}

func pendingSkillCursorLevelDrawOrigin(screen *render.Frame, x, y float64) (float64, float64) {
	return render.SnapScreenPoint(screen, x, y)
}

func maxRGBByte(a, b, c byte) byte {
	if b > a {
		a = b
	}
	if c > a {
		a = c
	}
	return a
}
