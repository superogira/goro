//go:build js && wasm

package game

import (
	"fmt"
	"sync/atomic"
	"syscall/js"
)

// DOM twins of the in-world vending boards: the page renders the pills and
// only re-renders what changed, so fifty boards cost one keyed update per
// camera move instead of fifty canvas blits + bitmap-text draws per frame.
// Actions never flow back — clicks keep going through the game's canvas
// hover logic, which reads the bounds captured while drawing.

type vendingBoardWebEntry struct {
	id    uint32
	x     float64
	y     float64
	title string
}

// vendingBoardsWebAvailable reports whether the page provides the board
// layer and installs the debug hooks on first use.
func vendingBoardsWebAvailable() bool {
	if js.Global().Get("goroVendingBoardsSync").Type() != js.TypeFunction {
		return false
	}
	installVendingBoardsHooks()
	return true
}

func vendingBoardsWebSync(entries []vendingBoardWebEntry) bool {
	sync := js.Global().Get("goroVendingBoardsSync")
	if sync.Type() != js.TypeFunction {
		return false
	}
	arr := js.Global().Get("Array").New(len(entries))
	for i, entry := range entries {
		obj := js.Global().Get("Object").New()
		obj.Set("id", int(entry.id))
		obj.Set("x", int(entry.x))
		obj.Set("y", int(entry.y))
		obj.Set("title", entry.title)
		arr.SetIndex(i, obj)
	}
	sync.Invoke(arr)
	return true
}

// vendingBoardsSignature changes only when a board moves a full pixel or a
// title changes, so a still camera costs no syncs at all.
func vendingBoardsSignature(entries []vendingBoardWebEntry) string {
	sig := fmt.Sprintf("n%d", len(entries))
	for _, entry := range entries {
		sig += fmt.Sprintf("|%d:%d:%d:%d", entry.id, int(entry.x), int(entry.y), len(entry.title))
	}
	return sig
}

// vendingBoardsSimCount feeds the load test: goroVendingBoardsSimulate(n)
// from the page console (or automation) spawns n synthetic boards through
// the exact same draw path, canvas or DOM alike.
var vendingBoardsSimCount atomic.Int32

func installVendingBoardsHooks() {
	if js.Global().Get("goroVendingBoardsSimulate").Type() == js.TypeFunction {
		return
	}
	js.Global().Set("goroVendingBoardsSimulate", js.FuncOf(func(this js.Value, args []js.Value) any {
		if len(args) >= 1 {
			n := args[0].Int()
			if n < 0 {
				n = 0
			}
			if n > 500 {
				n = 500
			}
			vendingBoardsSimCount.Store(int32(n))
		}
		return nil
	}))
}

func vendingBoardsSimRequested() int { return int(vendingBoardsSimCount.Load()) }
