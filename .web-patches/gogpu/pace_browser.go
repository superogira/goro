//go:build js && wasm

package gogpu

import (
	"sync"
	"syscall/js"
	"time"
)

// Browser frame pacing: wait for the compositor's requestAnimationFrame
// callback instead of sleeping a fixed interval. A fixed sleep desynchronizes
// the loop from the display refresh and beats against vsync, which shows up
// as micro-stutter; rAF hands control back exactly once per compositor frame.
//
// Go on wasm cannot block on a JS callback directly (a goroutine parked on a
// channel never lets the browser deliver events), so the wait polls a signal
// with short sleeps. Each sleep returns control to the JS event loop, letting
// the rAF callback run — the same pattern the fetch and WebSocket bridges
// use. A fallback timeout keeps the loop alive when rAF stops firing
// (browsers suspend it in hidden tabs) so the game keeps ticking at a
// reduced rate instead of freezing.

const (
	pacePollInterval = 2 * time.Millisecond
	paceRafTimeout   = 100 * time.Millisecond
)

var (
	rafOnce   sync.Once
	rafSignal = make(chan struct{}, 1)
	rafFunc   js.Func
)

func rafCallback() js.Func {
	rafOnce.Do(func() {
		rafFunc = js.FuncOf(func(_ js.Value, _ []js.Value) any {
			select {
			case rafSignal <- struct{}{}:
			default:
			}
			return nil
		})
	})
	return rafFunc
}

// paceBrowserFrame blocks until the next requestAnimationFrame callback or
// the fallback timeout, yielding to the browser event loop while waiting.
func paceBrowserFrame() {
	// Drain a stale signal — e.g. a late rAF from a previous timed-out wait
	// that fired during frame work — so this frame really waits for the
	// next compositor tick.
	select {
	case <-rafSignal:
	default:
	}
	js.Global().Call("requestAnimationFrame", rafCallback())
	deadline := time.Now().Add(paceRafTimeout)
	for {
		select {
		case <-rafSignal:
			return
		default:
		}
		if time.Now().After(deadline) {
			return
		}
		time.Sleep(pacePollInterval)
	}
}
