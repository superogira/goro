package ui

import (
	"strings"
	"testing"

	"github.com/gogpu/ui/uitest"
)

func TestPlayerContextGMEntriesUseTheSelectedPlayer(t *testing.T) {
	ctx, _, app := newWindowInstanceTest()
	var menu PlayerContextMenu
	menu.Open(ctx, 100, 100, 42, "Alice Smith", PlayerContextOptions{})
	app.Frame()
	canvas := &uitest.MockCanvas{}
	app.Window().DrawTo(canvas)
	for _, text := range canvas.StyledTexts {
		if strings.HasPrefix(text.Text, "GM:") {
			t.Fatal("ordinary player menu contains GM controls")
		}
	}
	for _, test := range []struct{ name, check string }{
		{"Alice Smith", `/check "Alice Smith"`},
		{`"Alice"`, `/check ""Alice""`},
		{`"Alice`, `/check ""Alice"`},
	} {
		for i, want := range []string{test.check, "/kill 42", "/mute 42 60", "/unmute 42"} {
			menu.Open(ctx, 100, 100, 42, test.name, PlayerContextOptions{ShowGM: true})
			app.Frame()
			app.Window().DrawTo(&uitest.MockCanvas{})
			x := float32(menu.x + 30)
			y := float32(menu.y + contextMenuPadding + (2+i)*contextMenuRowH + contextMenuRowH/2)
			app.Window().HandleEvent(uitest.Click(x, y))
			app.Window().HandleEvent(uitest.Release(x, y))
			action := menu.PopAction()
			if action.Kind != PlayerContextActionGMCommand || action.ActorID != 42 || action.Name != test.name || action.Command != want || menu.IsOpen() {
				t.Fatalf("GM action=%+v open=%t, want %q", action, menu.IsOpen(), want)
			}
		}
	}
	menu.Open(ctx, 100, 100, 99, "Bob", PlayerContextOptions{ShowGM: true})
	app.Frame()
	app.Window().DrawTo(&uitest.MockCanvas{})
	x := float32(menu.x + 30)
	y := float32(menu.y + contextMenuPadding + 3*contextMenuRowH + contextMenuRowH/2)
	app.Window().HandleEvent(uitest.Click(x, y))
	app.Window().HandleEvent(uitest.Release(x, y))
	if action := menu.PopAction(); action.Command != "/kill 99" {
		t.Fatalf("reopened menu retained the previous target: %+v", action)
	}
}
