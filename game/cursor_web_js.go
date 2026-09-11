//go:build js && wasm

package game

import (
	"bytes"
	"encoding/base64"
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
	fn.Invoke(url, x, y)
}

// cursorWebHide removes the DOM cursor (debug no-cursor mode).
func cursorWebHide() {
	if fn := js.Global().Get("goroCursorSync"); fn.Type() == js.TypeFunction {
		fn.Invoke(nil, 0, 0)
	}
}

func cursorWebImageURL(img *render.Image) string {
	if img == nil {
		return ""
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img.RGBA()); err != nil {
		return ""
	}
	return "data:image/png;base64," + base64.StdEncoding.EncodeToString(buf.Bytes())
}
