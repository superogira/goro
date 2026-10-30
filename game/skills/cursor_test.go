package skills

import (
	"image/color"
	"testing"

	"github.com/kivutar/goro/render"
	"github.com/kivutar/goro/session"
)

func TestPendingSkillCursorLevelTextUsesGRFNumberOnly(t *testing.T) {
	if got := pendingSkillCursorLevelText(session.Skill{ID: 18, Level: 10}); got != "10" {
		t.Fatalf("pending skill cursor level text = %q, want GRF digit text only", got)
	}
}

func TestPendingSkillCursorLevelTextHiddenWithoutPendingSkill(t *testing.T) {
	tests := []session.Skill{
		{ID: 0, Level: 10},
		{ID: 18, Level: 0},
		{ID: 18, Level: -1},
	}
	for _, skill := range tests {
		if got := pendingSkillCursorLevelText(skill); got != "" {
			t.Fatalf("pending skill cursor level text for %+v = %q, want hidden", skill, got)
		}
	}
}

func TestPendingSkillCursorLevelPositionMatchesCursorOverlayOffset(t *testing.T) {
	x, y := pendingSkillCursorLevelPosition(100, 80, 0, 0)
	if x != 120 || y != 68 {
		t.Fatalf("pending skill cursor level position = %.1f,%.1f, want 120.0,68.0", x, y)
	}
}

func TestPendingSkillCursorLevelPositionFollowsSnappedCursor(t *testing.T) {
	x, y := pendingSkillCursorLevelPosition(100, 80, 12.5, -4)
	if x != 107.5 || y != 72 {
		t.Fatalf("pending skill cursor level snapped position = %.1f,%.1f, want 107.5,72.0", x, y)
	}
}

func TestPendingSkillCursorLevelDrawOriginSnapsToScreenScale(t *testing.T) {
	screen := render.NewFrame(800, 600)
	screen.SetScreenScale(2, 2)
	x, y := pendingSkillCursorLevelDrawOrigin(screen, 120.3, 68.3)
	if x != 120.5 || y != 68.5 {
		t.Fatalf("pending skill cursor level draw origin = %.1f,%.1f, want 120.5,68.5", x, y)
	}
}

func TestPendingSkillCursorLevelOutlineUsesDarkBlueGray(t *testing.T) {
	c := pendingSkillCursorLevelOutlineColor
	if !(c.B > c.G && c.G > c.R && c.A == 255) {
		t.Fatalf("pending skill cursor level outline color = %#v, want opaque dark blue-gray", c)
	}
}

func TestPendingSkillCursorLevelGlyphImageDropsBakedBlackPixels(t *testing.T) {
	src := render.NewImage(3, 1)
	src.RGBA().SetRGBA(0, 0, color.RGBA{A: 255})
	src.RGBA().SetRGBA(1, 0, color.RGBA{R: 128, G: 128, B: 128, A: 255})
	src.RGBA().SetRGBA(2, 0, color.RGBA{R: 255, G: 255, B: 255, A: 128})

	glyph := pendingSkillCursorLevelGlyphImage(src)
	if glyph == nil {
		t.Fatal("expected glyph image")
	}
	if got := glyph.RGBA().RGBAAt(0, 0); got.A != 0 {
		t.Fatalf("black baked outline pixel = %#v, want transparent", got)
	}
	if got := glyph.RGBA().RGBAAt(1, 0); got != (color.RGBA{R: 255, G: 255, B: 255, A: 128}) {
		t.Fatalf("gray antialias pixel = %#v, want white alpha mask", got)
	}
	if got := glyph.RGBA().RGBAAt(2, 0); got != (color.RGBA{R: 255, G: 255, B: 255, A: 128}) {
		t.Fatalf("white face pixel = %#v, want preserved alpha mask", got)
	}
}
