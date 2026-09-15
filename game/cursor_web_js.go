//go:build js && wasm

package game

import (
	"bytes"
	"encoding/base64"
	"image"
	"image/png"
	"sync"
	"syscall/js"

	"github.com/kivutar/goro/render"
)

// The DOM cursor mirrors the canvas one: the same cached billboards, the
// same anchor/hotspot math, the same magnet offsets and frame animation.
// A fixed-position element follows the logical (CSS-pixel) mouse position,
// so it stays 1:1 with the real pointer and crisp at any resolution scale.

var cursorWebMutex sync.Mutex
var cursorWebURLs = map[string]string{}

// cursorWebEnabled reports whether the page provides the DOM cursor layer.
func cursorWebEnabled() bool {
	return js.Global().Get("goroCursorSync").Type() == js.TypeFunction
}

// cursorWebSync pushes one cursor frame: the image as a cached data URL
// (keyed by the billboard identity, so each action/motion frame encodes
// exactly once) and the element's top-left position in CSS pixels.
func cursorWebSync(img *render.Image, key string, x, y float64) {
	fn := js.Global().Get("goroCursorSync")
	if fn.Type() != js.TypeFunction || img == nil {
		return
	}
	cursorWebMutex.Lock()
	url, ok := cursorWebURLs[key]
	if !ok {
		url = cursorWebImageURL(img)
		if url == "" {
			cursorWebMutex.Unlock()
			return
		}
		cursorWebURLs[key] = url
	}
	cursorWebMutex.Unlock()
	cursorWebSetOSMode(false)
	fn.Invoke(url, x, y)
}

// cursorWebHide removes the DOM cursor (debug no-cursor mode) and hands
// the pointer back to the OS cursor: without this the pointer-seen CSS
// keeps hiding the OS cursor everywhere and the page is left cursorless.
func cursorWebHide() {
	cursorWebSetOSMode(true)
	if fn := js.Global().Get("goroCursorSync"); fn.Type() == js.TypeFunction {
		fn.Invoke(nil, 0, 0)
	}
}

// cursorWebSetOSMode toggles the page's OS-cursor mode. Idempotent and
// cheap: normal frames re-assert false, no-cursor frames re-assert true.
func cursorWebSetOSMode(os bool) {
	if fn := js.Global().Get("goroCursorOSMode"); fn.Type() == js.TypeFunction {
		fn.Invoke(os)
	}
}

func cursorWebImageURL(img *render.Image) string {
	if img == nil {
		return ""
	}
	src := img.RGBA()
	if src == nil {
		return ""
	}
	// The frame buffers hold straight alpha, but *image.RGBA is
	// premultiplied by convention — png.Encode would un-premultiply the
	// bytes and crush colors (the cursor's white body came out near-black,
	// its blues shifted). A byte-identical NRGBA copy encodes as-is.
	n := image.NewNRGBA(src.Bounds())
	copy(n.Pix, src.Pix)
	var buf bytes.Buffer
	if err := png.Encode(&buf, n); err != nil {
		return ""
	}
	return "data:image/png;base64," + base64.StdEncoding.EncodeToString(buf.Bytes())
}
