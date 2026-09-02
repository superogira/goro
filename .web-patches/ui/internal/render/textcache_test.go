package render

import (
	"os"
	"testing"

	"github.com/gogpu/gg/text"
)

// cachedTextMeasure must return values identical to a direct text.Measure
// and stay stable across repeated calls.
func TestCachedTextMeasureMatchesDirect(t *testing.T) {
	data, err := os.ReadFile("testdata/Sarabun-Regular.ttf")
	if err != nil {
		t.Skipf("test font missing: %v", err)
	}
	source, err := text.NewFontSource(data)
	if err != nil {
		t.Fatalf("font source: %v", err)
	}
	const size = 14.0
	const str = "measure me"
	w1, h1 := cachedTextMeasure(source, size, str)
	w2, h2 := cachedTextMeasure(source, size, str)
	if w1 != w2 || h1 != h2 {
		t.Fatalf("cache unstable: (%v,%v) vs (%v,%v)", w1, h1, w2, h2)
	}
	dw, dh := text.Measure(str, source.Face(size))
	if w1 != dw || h1 != dh {
		t.Fatalf("cached (%v,%v) != direct (%v,%v)", w1, h1, dw, dh)
	}
}
