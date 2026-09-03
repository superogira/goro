//go:build js && wasm

package audio

import (
	"sync"
	"syscall/js"

	"github.com/ebitengine/oto/v3"
)

var gestureResumeOnce sync.Once

// registerGestureAudioResume starts the suspended AudioContext on the first
// pointer press. oto installs its own gesture listeners for keyup/mouseup/
// touchend, but the game preventDefaults pointerdown, which suppresses the
// compatibility mouseup on desktop browsers — clicks alone never woke the
// audio, only keyboard presses did.
func registerGestureAudioResume(ctx *oto.Context) {
	gestureResumeOnce.Do(func() {
		cb := js.FuncOf(func(js.Value, []js.Value) any {
			_ = ctx.Resume()
			return nil
		})
		opts := js.ValueOf(map[string]any{"once": true, "capture": true})
		js.Global().Get("document").Call("addEventListener", "pointerdown", cb, opts)
	})
}
