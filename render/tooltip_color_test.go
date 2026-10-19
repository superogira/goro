package render

import (
	"slices"
	"strings"
	"testing"

	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/uitest"
	"github.com/gogpu/ui/widget"
	"github.com/kivutar/goro/ui/rotheme"
)

func TestTooltipColorsSurviveWrappingAndNewlines(t *testing.T) {
	canvas := &uitest.MockCanvas{}
	style := consoleOverlayTextBoxStyle()
	style.colorCodes = true
	style.wrap = true
	style.maxWidth = 100
	text := "Type:^33cc00 Active^000000\nCatalyst:^ee0000 1 Red Gemstone\nStill red^000000 normal\n^33cc00 " + strings.Repeat("é", 40)
	lines := overlayTextBoxLines(canvas, text, style)
	plainLines := make([]string, len(lines))
	current := style.foreground
	var drawn strings.Builder
	for i, line := range lines {
		plainLines[i] = rotheme.StripColorCodes(line)
		current = drawOverlayTextLine(canvas, line, geometry.NewRect(8, float32(i)*14, 84, 14), style, current)
	}
	plainStyle := style
	plainStyle.colorCodes = false
	if want := overlayTextBoxLines(canvas, rotheme.StripColorCodes(text), plainStyle); !slices.Equal(plainLines, want) {
		t.Fatalf("color codes changed wrapping: %q, want %q", plainLines, want)
	}
	for _, call := range canvas.StyledTexts {
		drawn.WriteString(call.Text)
		want := style.foreground
		switch {
		case strings.Contains(call.Text, "Active"), strings.Contains(call.Text, "é"):
			want = widget.RGBA8(0x33, 0xcc, 0, 255)
		case strings.Contains(call.Text, "Red"), strings.Contains(call.Text, "Gemstone"), strings.Contains(call.Text, "Still"), strings.Contains(call.Text, "red"):
			want = widget.RGBA8(0xee, 0, 0, 255)
		}
		if call.Style.Color != want {
			t.Errorf("%q color = %v, want %v", call.Text, call.Style.Color, want)
		}
	}
	if strings.Contains(drawn.String(), "^") {
		t.Fatalf("visible color codes: %q", drawn.String())
	}
	plainStyle.colorCodes = false
	canvas.StyledTexts = nil
	drawOverlayTextLine(canvas, "Player ^ff0000", geometry.NewRect(0, 0, 200, 14), plainStyle, plainStyle.foreground)
	if len(canvas.StyledTexts) != 1 || canvas.StyledTexts[0].Text != "Player ^ff0000" {
		t.Fatal("plain tooltip interpreted a player's text as markup")
	}
}

func TestTooltipEllipsisKeepsColorCodesIntact(t *testing.T) {
	canvas := &uitest.MockCanvas{}
	style := consoleOverlayTextBoxStyle()
	style.colorCodes = true
	style.maxWidth = 100
	text := "^777777ééé^ee0000 Red Gemstone^000000 and some more text"
	got := ellipsizeOverlayText(canvas, text, style)
	plainStyle := style
	plainStyle.colorCodes = false
	want := ellipsizeOverlayText(canvas, rotheme.StripColorCodes(text), plainStyle)
	if plain := rotheme.StripColorCodes(got); plain != want || strings.Contains(plain, "^") {
		t.Fatalf("truncated colored text = %q, want visible %q", got, want)
	}
}
