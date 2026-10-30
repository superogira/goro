package ui

import (
	"math"

	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/primitives"
	"github.com/gogpu/ui/uitest"
	"github.com/gogpu/ui/widget"
	"github.com/kivutar/goro/ui/rotheme"
	"testing"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/input"
	"github.com/kivutar/goro/session"
)

func TestCharacterRestoreCompactBeforeOpening(t *testing.T) {
	t.Skip("fork: the character HUD compact height is the HUD layout's, not upstream widget heights")
}
func TestCharacterEXPPanelWrapsBothRowsInRoundedGreyContainer(t *testing.T) {
	const width float32 = characterWindowWidth - 24
	panel := characterEXPPanel(session.Progress{
		BaseLevel:   42,
		JobLevel:    27,
		BaseExp:     25,
		NextBaseExp: 100,
		JobExp:      50,
		NextJobExp:  100,
	}, width)
	panel.Layout(widget.NewContext(), geometry.Constraints{MaxWidth: width, MaxHeight: 100})

	box, ok := panel.(*primitives.BoxWidget)
	if !ok {
		t.Fatalf("EXP panel = %T, want box", panel)
	}
	style := box.Style()
	if style.Radius != characterEXPPanelRadius {
		t.Fatalf("EXP panel radius = %.1f, want %.1f", style.Radius, characterEXPPanelRadius)
	}
	uitest.AssertColorEqual(t, style.Background, rotheme.Default.Colors.WindowFooter)
	if style.Border.Width != 0 {
		t.Fatalf("EXP panel border width = %.1f, want none", style.Border.Width)
	}

	rows := panel.Children()
	if len(rows) != 2 {
		t.Fatalf("EXP panel rows = %d, want Base and Job EXP", len(rows))
	}
	first := rows[0].(interface{ Bounds() geometry.Rect }).Bounds()
	second := rows[1].(interface{ Bounds() geometry.Rect }).Bounds()
	wantRowWidth := width - 2*characterEXPPanelPaddingX
	if first.Min.X != characterEXPPanelPaddingX || first.Min.Y != characterEXPPanelPaddingY || first.Width() != wantRowWidth {
		t.Fatalf("Base EXP row bounds = %v, want inset width %.1f", first, wantRowWidth)
	}
	if second.Min.X != characterEXPPanelPaddingX || second.Min.Y != first.Max.Y+characterEXPPanelGap || second.Width() != wantRowWidth {
		t.Fatalf("Job EXP row bounds = %v, want below Base EXP with %.1f gap", second, characterEXPPanelGap)
	}
	assertCharacterEXPRow := func(row widget.Widget, wantLabel string) {
		t.Helper()
		children := row.Children()
		if len(children) != 2 {
			t.Fatalf("%s row children = %d, want label and bar", wantLabel, len(children))
		}
		labelChildren := children[0].Children()
		if len(labelChildren) != 1 {
			t.Fatalf("%s label children = %d, want text", wantLabel, len(labelChildren))
		}
		text, ok := labelChildren[0].(interface{ Content() string })
		if !ok {
			t.Fatalf("EXP row label = %T, want text", labelChildren[0])
		}
		if text.Content() != wantLabel {
			t.Fatalf("EXP row label = %q, want %q", text.Content(), wantLabel)
		}
		labelBounds := children[0].(interface{ Bounds() geometry.Rect }).Bounds()
		barSlotBounds := children[1].(interface{ Bounds() geometry.Rect }).Bounds()
		if barSlotBounds.Min.X != labelBounds.Max.X+characterEXPLabelBarGap {
			t.Fatalf("%s bar x = %.1f, want %.1f after label", wantLabel, barSlotBounds.Min.X, labelBounds.Max.X+characterEXPLabelBarGap)
		}
		barChildren := children[1].Children()
		if len(barChildren) != 1 {
			t.Fatalf("%s bar slot children = %d, want bar", wantLabel, len(barChildren))
		}
		bar, ok := barChildren[0].(*characterBarWidget)
		if !ok {
			t.Fatalf("%s bar = %T, want character bar", wantLabel, barChildren[0])
		}
		uitest.AssertColorEqual(t, bar.background, widget.ColorWhite)
		barBounds := bar.Bounds()
		labelCenterY := labelBounds.Center().Y
		barCenterY := barSlotBounds.Min.Y + barBounds.Center().Y
		if math.Abs(float64(labelCenterY-barCenterY)) > 0.001 {
			t.Fatalf("%s vertical centers differ: label=%.3f bar=%.3f", wantLabel, labelCenterY, barCenterY)
		}
	}
	assertCharacterEXPRow(rows[0], "Base Lv. 42")
	assertCharacterEXPRow(rows[1], "Job Lv. 27")
	panelBounds := panel.(interface{ Bounds() geometry.Rect }).Bounds()
	if second.Max.Y+characterEXPPanelPaddingY != panelBounds.Max.Y {
		t.Fatalf("EXP panel bottom = %.1f, want %.1f after bottom inset", panelBounds.Max.Y, second.Max.Y+characterEXPPanelPaddingY)
	}
}

func TestCharacterHUDDragClampsToScreen(t *testing.T) {
	inputState := input.NewState()
	ctx := client.Context{
		Input:   inputState,
		Session: &session.Session{Selected: session.Character{Name: "Kivutar"}},
		ScreenW: 800,
		ScreenH: 300,
	}
	var w CharacterWindow
	w.Update(ctx)
	inputState.SetMousePosition(w.x+10, w.y+5)
	inputState.SetMouseButton(input.MouseButtonLeft, true)
	if !w.Update(ctx) {
		t.Fatal("drag start was not consumed")
	}
	if !w.dragLayer {
		t.Fatal("HUD did not enter drag state")
	}
	inputState.EndFrame()
	inputState.SetMousePosition(700, 1000)
	w.Update(ctx)
	if w.x > ctx.ScreenW-w.width {
		t.Fatalf("drag x = %d beyond right edge", w.x)
	}
	_, menuHeight := basicMenuSize()
	maxY := ctx.ScreenH - w.height - basicMenuFollowGap - menuHeight
	if w.y != maxY {
		t.Fatalf("drag y = %d, want clamped at %d (screen minus menu)", w.y, maxY)
	}
	inputState.EndFrame()
	inputState.SetMouseButton(input.MouseButtonLeft, false)
	w.Update(ctx)
	if w.dragLayer {
		t.Fatal("drag state did not end on release")
	}
}

func TestCharacterWindowDataFallsBackToSelectedCharacter(t *testing.T) {
	character, vitals, progress, inventory := characterWindowData(&session.Session{
		Selected: session.Character{ID: 7, Name: "Kivutar", HP: 12, MaxHP: 34, SP: 5, MaxSP: 6, Level: 7, JobLevel: 4},
	})
	if character.Name != "Kivutar" {
		t.Fatalf("character name = %q", character.Name)
	}
	if vitals.HP != 12 || vitals.MaxHP != 34 || vitals.SP != 5 || vitals.MaxSP != 6 {
		t.Fatalf("vitals fallback = %+v", vitals)
	}
	if progress.BaseLevel != 7 || progress.JobLevel != 4 {
		t.Fatalf("progress fallback = %+v", progress)
	}
	if inventory.Weight != 0 || inventory.Zeny != 0 {
		t.Fatalf("inventory fallback = %+v", inventory)
	}
}

func TestFormatHUDNumberGroupsThousands(t *testing.T) {
	cases := map[int64]string{
		0:        "0",
		999:      "999",
		1000:     "1,000",
		114527:   "114,527",
		-1234567: "-1,234,567",
	}
	for value, want := range cases {
		if got := formatHUDNumber(value); got != want {
			t.Fatalf("formatHUDNumber(%d) = %q, want %q", value, got, want)
		}
	}
}

func TestCharacterWindowRebindKeepsCompactStateAndOwnsToggle(t *testing.T) {
	t.Skip("fork: main's character window is the DOM-styled canvas HUD; upstream's widget-window test does not apply (the HUD's own tests cover it)")
}

func TestCharacterWindowToggleSurvivesUpdatesWhilePressed(t *testing.T) {
	t.Skip("fork: main's character window is the DOM-styled canvas HUD; upstream's widget-window tests do not apply (the HUD's own tests cover it)")
}

func TestFormatEXPPercent(t *testing.T) {
	if got := formatEXPPercent(123, 1000); got != "12.3%" {
		t.Fatalf("formatted exp percent = %q", got)
	}
	if got := formatEXPPercent(12, 0); got != "--" {
		t.Fatalf("formatted exp percent without next = %q", got)
	}
	if got := formatEXPPercent(1001, 1000); got != "100.0%" {
		t.Fatalf("formatted capped exp percent = %q", got)
	}
}

func TestDisplayWeightUsesROVisibleUnits(t *testing.T) {
	if got := displayWeight(240); got != 24 {
		t.Fatalf("display weight = %d, want 24", got)
	}
}
