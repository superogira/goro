package render

import (
	"testing"

	"github.com/gogpu/gpucontext"
	"github.com/kivutar/goro/input"
)

func TestWireInputAltTabWithoutKeyRelease(t *testing.T) {
	for _, name := range []string{"AltLeft", "AltRight"} {
		t.Run(name, func(t *testing.T) {
			events := &fanoutEventSource{}
			state := input.NewState()
			wireInput(events, state)
			alt, ok := input.KeyCodeFromName(name)
			if !ok {
				t.Fatalf("unknown key code %q", name)
			}
			for _, fn := range events.keyPress {
				fn(alt, gpucontext.ModAlt)
				fn(gpucontext.KeyTab, gpucontext.ModAlt)
			}
			state.EndFrame()

			// Alt is released in the other application, so Goro receives no
			// key release between losing and regaining keyboard focus.
			for _, fn := range events.focus {
				fn(false)
			}
			if state.Pressed(input.KeyAlt) || state.KeyCodeDown(alt) || state.Pressed(input.KeyTab) {
				t.Fatal("keys remained held after focus loss")
			}
			for _, fn := range events.focus {
				fn(true)
			}
			for _, fn := range events.mousePress {
				fn(gpucontext.MouseButtonLeft, 400, 300)
			}
			if state.Pressed(input.KeyAlt) || !state.MouseJustPressed(input.MouseButtonLeft) {
				t.Fatal("first click after Alt+Tab was not an ordinary left click")
			}

			// Genuine Alt shortcuts must still work after returning.
			for _, fn := range events.keyPress {
				fn(alt, gpucontext.ModAlt)
			}
			if !state.JustPressed(input.KeyAlt) || !state.KeyCodeJustPressed(alt) {
				t.Fatal("fresh Alt press was not recognized")
			}
			for _, fn := range events.keyRelease {
				fn(alt, 0)
			}
			if state.Pressed(input.KeyAlt) || !state.KeyCodeJustReleased(alt) {
				t.Fatal("fresh Alt release was not recognized")
			}
		})
	}
}
