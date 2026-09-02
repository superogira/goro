package render

import (
	"sync"

	"github.com/gogpu/gg/text"
)

// cachedTextMeasure memoizes text.Measure results. Shaping is pure with
// respect to (font source, size, string), so memoizing cannot change any
// rendered output — it only removes repeated shaping work from layout and
// draw paths.
//
// NOTE: a text run blit cache previously lived here as well. It was
// removed: on canvases with a GPU text accelerator, replacing DrawString
// with a CPU-rasterized blit bypassed the GPU glyph pipeline (slower on
// those platforms) and its baseline metrics differed, which displaced
// text on some browsers.

type textMeasureKey struct {
	source *text.FontSource
	size   float64
	str    string
}

const textMeasureCacheMax = 16384

type textMeasureCacheT struct {
	mu      sync.Mutex
	measure map[textMeasureKey][2]float64
	order   []textMeasureKey
}

var textMeasureCache = &textMeasureCacheT{measure: make(map[textMeasureKey][2]float64)}

func cachedTextMeasure(source *text.FontSource, size float64, s string) (w, h float64) {
	key := textMeasureKey{source: source, size: size, str: s}
	textMeasureCache.mu.Lock()
	if dims, ok := textMeasureCache.measure[key]; ok {
		textMeasureCache.mu.Unlock()
		return dims[0], dims[1]
	}
	textMeasureCache.mu.Unlock()
	face := source.Face(size)
	w, h = text.Measure(s, face)
	textMeasureCache.mu.Lock()
	if len(textMeasureCache.order) >= textMeasureCacheMax {
		oldest := textMeasureCache.order[0]
		textMeasureCache.order = textMeasureCache.order[1:]
		delete(textMeasureCache.measure, oldest)
	}
	textMeasureCache.measure[key] = [2]float64{w, h}
	textMeasureCache.order = append(textMeasureCache.order, key)
	textMeasureCache.mu.Unlock()
	return w, h
}
