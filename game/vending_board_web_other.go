//go:build !js || !wasm

package game

// Native stubs: vending boards keep the canvas path, and the load-test
// hook exists only for the web build.

type vendingBoardWebEntry struct {
	id    uint32
	x     float64
	y     float64
	title string
}

func vendingBoardsWebAvailable() bool          { return false }
func vendingBoardsWebSync(entries []vendingBoardWebEntry) bool { return false }
func vendingBoardsSignature(entries []vendingBoardWebEntry) string {
	return ""
}

func vendingBoardsSimRequested() int { return 0 }
