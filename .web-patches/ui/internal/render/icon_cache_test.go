package render

import (
	"sync"
	"testing"

	"github.com/gogpu/gg/scene"
	"github.com/gogpu/ui/widget"
)

// --- Construction Tests ---

func TestNewIconCache_DefaultSize(t *testing.T) {
	c := newIconCache(0)
	if c.maxItems != defaultIconCacheMaxEntries {
		t.Errorf("maxItems = %d, want %d", c.maxItems, defaultIconCacheMaxEntries)
	}
}

func TestNewIconCache_CustomSize(t *testing.T) {
	c := newIconCache(128)
	if c.maxItems != 128 {
		t.Errorf("maxItems = %d, want 128", c.maxItems)
	}
}

func TestNewIconCache_NegativeUsesDefault(t *testing.T) {
	c := newIconCache(-1)
	if c.maxItems != defaultIconCacheMaxEntries {
		t.Errorf("maxItems = %d, want %d", c.maxItems, defaultIconCacheMaxEntries)
	}
}

// --- Level 1: Document Cache ---

// minimalSVG is a valid minimal SVG document for testing.
var minimalSVG = []byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24"><path d="M12 2L2 22h20z"/></svg>`)

// anotherSVG is a distinct SVG document for testing separate cache entries.
var anotherSVG = []byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24"><rect width="24" height="24"/></svg>`)

func TestIconCache_GetDoc_ParsesAndCaches(t *testing.T) {
	c := newIconCache(16)

	doc := c.getDoc(minimalSVG)
	if doc == nil {
		t.Fatal("expected non-nil document from valid SVG")
	}

	stats := c.stats()
	if stats.DocEntries != 1 {
		t.Errorf("DocEntries = %d, want 1", stats.DocEntries)
	}
	if stats.DocMisses != 1 {
		t.Errorf("DocMisses = %d, want 1 (first parse)", stats.DocMisses)
	}

	// Second call should hit cache.
	doc2 := c.getDoc(minimalSVG)
	if doc2 != doc {
		t.Error("expected same document pointer on cache hit")
	}

	stats = c.stats()
	if stats.DocHits != 1 {
		t.Errorf("DocHits = %d, want 1", stats.DocHits)
	}
}

func TestIconCache_GetDoc_DifferentSVGs(t *testing.T) {
	c := newIconCache(16)

	doc1 := c.getDoc(minimalSVG)
	doc2 := c.getDoc(anotherSVG)

	if doc1 == nil || doc2 == nil {
		t.Fatal("expected non-nil documents")
	}
	if doc1 == doc2 {
		t.Error("different SVGs should produce different documents")
	}

	stats := c.stats()
	if stats.DocEntries != 2 {
		t.Errorf("DocEntries = %d, want 2", stats.DocEntries)
	}
}

func TestIconCache_GetDoc_InvalidSVG(t *testing.T) {
	c := newIconCache(16)

	doc := c.getDoc([]byte("not valid svg"))
	if doc != nil {
		t.Error("expected nil for invalid SVG")
	}

	stats := c.stats()
	if stats.DocEntries != 0 {
		t.Errorf("DocEntries = %d, want 0 (invalid SVG should not be cached)", stats.DocEntries)
	}
}

func TestIconCache_GetDoc_Empty(t *testing.T) {
	c := newIconCache(16)

	doc := c.getDoc(nil)
	if doc != nil {
		t.Error("expected nil for nil input")
	}

	doc = c.getDoc([]byte{})
	if doc != nil {
		t.Error("expected nil for empty input")
	}
}

// --- Level 2: Image Cache ---

func makeSceneImage(w, h int) *scene.Image {
	img := scene.NewImage(w, h)
	img.Data = make([]byte, w*h*4)
	return img
}

func TestIconCache_PutGetImage_RoundTrip(t *testing.T) {
	c := newIconCache(16)
	img := makeSceneImage(24, 24)
	key := iconImageKey{svgPtr: 0x1000, width: 24, height: 24, color: 0xFF0000FF}

	c.putImage(key, img)

	got := c.getImage(key)
	if got != img {
		t.Error("expected same image pointer on cache hit")
	}

	stats := c.stats()
	if stats.ImageEntries != 1 {
		t.Errorf("ImageEntries = %d, want 1", stats.ImageEntries)
	}
	if stats.Hits != 1 {
		t.Errorf("Hits = %d, want 1", stats.Hits)
	}
}

func TestIconCache_GetImage_Miss(t *testing.T) {
	c := newIconCache(16)
	key := iconImageKey{svgPtr: 0x9999, width: 24, height: 24, color: 0xFF0000FF}

	got := c.getImage(key)
	if got != nil {
		t.Error("expected nil on cache miss")
	}

	stats := c.stats()
	if stats.Misses != 1 {
		t.Errorf("Misses = %d, want 1", stats.Misses)
	}
}

func TestIconCache_PutImage_NilIgnored(t *testing.T) {
	c := newIconCache(16)
	key := iconImageKey{svgPtr: 0x1000, width: 24, height: 24, color: 0xFF0000FF}

	c.putImage(key, nil)
	if c.stats().ImageEntries != 0 {
		t.Error("nil image should not be stored")
	}
}

func TestIconCache_PutImage_ReplacesExisting(t *testing.T) {
	c := newIconCache(16)
	key := iconImageKey{svgPtr: 0x1000, width: 24, height: 24, color: 0xFF0000FF}

	img1 := makeSceneImage(24, 24)
	img2 := makeSceneImage(24, 24)

	c.putImage(key, img1)
	c.putImage(key, img2)

	if c.stats().ImageEntries != 1 {
		t.Errorf("ImageEntries = %d, want 1 (replaced)", c.stats().ImageEntries)
	}

	got := c.getImage(key)
	if got != img2 {
		t.Error("expected replaced image")
	}
}

func TestIconCache_DifferentColors_DifferentEntries(t *testing.T) {
	c := newIconCache(16)
	img1 := makeSceneImage(24, 24)
	img2 := makeSceneImage(24, 24)

	key1 := iconImageKey{svgPtr: 0x1000, width: 24, height: 24, color: 0xFF0000FF}
	key2 := iconImageKey{svgPtr: 0x1000, width: 24, height: 24, color: 0x00FF00FF}

	c.putImage(key1, img1)
	c.putImage(key2, img2)

	if c.stats().ImageEntries != 2 {
		t.Errorf("ImageEntries = %d, want 2", c.stats().ImageEntries)
	}

	if c.getImage(key1) != img1 {
		t.Error("key1 should return img1")
	}
	if c.getImage(key2) != img2 {
		t.Error("key2 should return img2")
	}
}

func TestIconCache_DifferentSizes_DifferentEntries(t *testing.T) {
	c := newIconCache(16)
	img1 := makeSceneImage(16, 16)
	img2 := makeSceneImage(24, 24)

	key1 := iconImageKey{svgPtr: 0x1000, width: 16, height: 16, color: 0xFF0000FF}
	key2 := iconImageKey{svgPtr: 0x1000, width: 24, height: 24, color: 0xFF0000FF}

	c.putImage(key1, img1)
	c.putImage(key2, img2)

	if c.stats().ImageEntries != 2 {
		t.Errorf("ImageEntries = %d, want 2", c.stats().ImageEntries)
	}
}

// --- LRU Eviction Tests ---

func TestIconCache_LRU_Eviction(t *testing.T) {
	c := newIconCache(3) // max 3 entries

	for i := range 5 {
		key := iconImageKey{svgPtr: uintptr(i), width: 24, height: 24, color: 0xFF0000FF}
		c.putImage(key, makeSceneImage(24, 24))
	}

	stats := c.stats()
	if stats.ImageEntries != 3 {
		t.Errorf("ImageEntries = %d, want 3 (capacity)", stats.ImageEntries)
	}
	if stats.Evictions < 2 {
		t.Errorf("Evictions = %d, want >= 2", stats.Evictions)
	}

	// First two should be evicted.
	key0 := iconImageKey{svgPtr: 0, width: 24, height: 24, color: 0xFF0000FF}
	if c.getImage(key0) != nil {
		t.Error("entry 0 should have been evicted")
	}

	key1 := iconImageKey{svgPtr: 1, width: 24, height: 24, color: 0xFF0000FF}
	if c.getImage(key1) != nil {
		t.Error("entry 1 should have been evicted")
	}

	// Last three should still be present.
	for i := 2; i < 5; i++ {
		key := iconImageKey{svgPtr: uintptr(i), width: 24, height: 24, color: 0xFF0000FF}
		if c.getImage(key) == nil {
			t.Errorf("entry %d should still be in cache", i)
		}
	}
}

func TestIconCache_LRU_AccessPromotes(t *testing.T) {
	c := newIconCache(3)

	key1 := iconImageKey{svgPtr: 1, width: 24, height: 24, color: 0xFF0000FF}
	key2 := iconImageKey{svgPtr: 2, width: 24, height: 24, color: 0xFF0000FF}
	key3 := iconImageKey{svgPtr: 3, width: 24, height: 24, color: 0xFF0000FF}
	key4 := iconImageKey{svgPtr: 4, width: 24, height: 24, color: 0xFF0000FF}

	c.putImage(key1, makeSceneImage(24, 24)) // LRU order: [1]
	c.putImage(key2, makeSceneImage(24, 24)) // LRU order: [2, 1]
	c.putImage(key3, makeSceneImage(24, 24)) // LRU order: [3, 2, 1]

	// Access key1 to promote it to front: [1, 3, 2]
	_ = c.getImage(key1)

	// Insert key4 — should evict key2 (LRU), not key1.
	c.putImage(key4, makeSceneImage(24, 24))

	if c.getImage(key2) != nil {
		t.Error("key2 should have been evicted (LRU)")
	}
	if c.getImage(key1) == nil {
		t.Error("key1 should still be present (recently accessed)")
	}
	if c.getImage(key4) == nil {
		t.Error("key4 should be present (just inserted)")
	}
}

// --- Invalidation Tests ---

func TestIconCache_InvalidateImages(t *testing.T) {
	c := newIconCache(16)

	// Populate both levels.
	_ = c.getDoc(minimalSVG)
	key := iconImageKey{svgPtr: svgSlicePtr(minimalSVG), width: 24, height: 24, color: 0xFF0000FF}
	c.putImage(key, makeSceneImage(24, 24))

	c.invalidateImages()

	stats := c.stats()
	if stats.ImageEntries != 0 {
		t.Errorf("ImageEntries = %d, want 0 after invalidateImages", stats.ImageEntries)
	}
	// Level 1 should be preserved.
	if stats.DocEntries != 1 {
		t.Errorf("DocEntries = %d, want 1 (should be preserved)", stats.DocEntries)
	}
}

func TestIconCache_InvalidateAll(t *testing.T) {
	c := newIconCache(16)

	_ = c.getDoc(minimalSVG)
	key := iconImageKey{svgPtr: svgSlicePtr(minimalSVG), width: 24, height: 24, color: 0xFF0000FF}
	c.putImage(key, makeSceneImage(24, 24))

	c.invalidateAll()

	stats := c.stats()
	if stats.ImageEntries != 0 {
		t.Errorf("ImageEntries = %d, want 0", stats.ImageEntries)
	}
	if stats.DocEntries != 0 {
		t.Errorf("DocEntries = %d, want 0", stats.DocEntries)
	}
}

func TestIconCache_InvalidateImages_EvictionCount(t *testing.T) {
	c := newIconCache(16)

	for i := range 5 {
		key := iconImageKey{svgPtr: uintptr(i), width: 24, height: 24, color: 0xFF0000FF}
		c.putImage(key, makeSceneImage(24, 24))
	}

	c.invalidateImages()

	stats := c.stats()
	if stats.Evictions != 5 {
		t.Errorf("Evictions = %d, want 5", stats.Evictions)
	}
}

// --- packColor Tests ---

func TestPackColor(t *testing.T) {
	tests := []struct {
		name  string
		color widget.Color
		want  uint32
	}{
		{"red", widget.Color{R: 1, G: 0, B: 0, A: 1}, 0xFF0000FF},
		{"green", widget.Color{R: 0, G: 1, B: 0, A: 1}, 0x00FF00FF},
		{"blue", widget.Color{R: 0, G: 0, B: 1, A: 1}, 0x0000FFFF},
		{"white", widget.Color{R: 1, G: 1, B: 1, A: 1}, 0xFFFFFFFF},
		{"black_opaque", widget.Color{R: 0, G: 0, B: 0, A: 1}, 0x000000FF},
		{"transparent", widget.Color{R: 0, G: 0, B: 0, A: 0}, 0x00000000},
		{"half_alpha", widget.Color{R: 1, G: 1, B: 1, A: 0.5}, 0xFFFFFF7F},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := packColor(tt.color)
			if got != tt.want {
				t.Errorf("packColor(%v) = 0x%08X, want 0x%08X", tt.color, got, tt.want)
			}
		})
	}
}

func TestPackColor_Deterministic(t *testing.T) {
	c := widget.Color{R: 0.5, G: 0.25, B: 0.75, A: 1.0}
	p1 := packColor(c)
	p2 := packColor(c)
	if p1 != p2 {
		t.Errorf("packColor not deterministic: 0x%08X != 0x%08X", p1, p2)
	}
}

// --- Pointer helper tests ---

func TestSvgSlicePtr_StableForSameSlice(t *testing.T) {
	data := []byte("test svg data")
	p1 := svgSlicePtr(data)
	p2 := svgSlicePtr(data)
	if p1 != p2 {
		t.Errorf("same slice should produce same pointer: %v != %v", p1, p2)
	}
	if p1 == 0 {
		t.Error("expected non-zero pointer for non-empty slice")
	}
}

func TestSvgSlicePtr_NilAndEmpty(t *testing.T) {
	if svgSlicePtr(nil) != 0 {
		t.Error("nil slice should return 0")
	}
	if svgSlicePtr([]byte{}) != 0 {
		t.Error("empty slice should return 0")
	}
}

func TestSvgStringPtr_StableForSameString(t *testing.T) {
	s := "M12 2L2 22h20z"
	p1 := svgStringPtr(s)
	p2 := svgStringPtr(s)
	if p1 != p2 {
		t.Errorf("same string should produce same pointer: %v != %v", p1, p2)
	}
	if p1 == 0 {
		t.Error("expected non-zero pointer for non-empty string")
	}
}

func TestSvgStringPtr_Empty(t *testing.T) {
	if svgStringPtr("") != 0 {
		t.Error("empty string should return 0")
	}
}

// --- Statistics Tests ---

func TestIconCache_Stats(t *testing.T) {
	c := newIconCache(16)

	_ = c.getDoc(minimalSVG)
	_ = c.getDoc(minimalSVG) // doc hit

	key := iconImageKey{svgPtr: 1, width: 24, height: 24, color: 0xFF0000FF}
	c.putImage(key, makeSceneImage(24, 24))
	_ = c.getImage(key)                                                             // hit
	_ = c.getImage(iconImageKey{svgPtr: 9, width: 24, height: 24, color: 0xFF00FF}) // miss

	stats := c.stats()
	if stats.DocEntries != 1 {
		t.Errorf("DocEntries = %d, want 1", stats.DocEntries)
	}
	if stats.DocHits != 1 {
		t.Errorf("DocHits = %d, want 1", stats.DocHits)
	}
	if stats.DocMisses != 1 {
		t.Errorf("DocMisses = %d, want 1", stats.DocMisses)
	}
	if stats.ImageEntries != 1 {
		t.Errorf("ImageEntries = %d, want 1", stats.ImageEntries)
	}
	if stats.Hits != 1 {
		t.Errorf("Hits = %d, want 1", stats.Hits)
	}
	if stats.Misses != 1 {
		t.Errorf("Misses = %d, want 1", stats.Misses)
	}
	if stats.HitRate < 0.49 || stats.HitRate > 0.51 {
		t.Errorf("HitRate = %f, want ~0.5", stats.HitRate)
	}
	if stats.MaxImageEntries != 16 {
		t.Errorf("MaxImageEntries = %d, want 16", stats.MaxImageEntries)
	}
}

func TestIconCache_Stats_Empty(t *testing.T) {
	c := newIconCache(16)
	stats := c.stats()

	if stats.HitRate != 0 {
		t.Errorf("HitRate = %f, want 0", stats.HitRate)
	}
	if stats.DocEntries != 0 || stats.ImageEntries != 0 {
		t.Error("empty cache should have 0 entries")
	}
}

func TestIconCache_ResetStats(t *testing.T) {
	c := newIconCache(16)

	key := iconImageKey{svgPtr: 1, width: 24, height: 24, color: 0xFF0000FF}
	c.putImage(key, makeSceneImage(24, 24))
	_ = c.getImage(key) // hit
	_ = c.getImage(iconImageKey{svgPtr: 9, width: 24, height: 24, color: 0xFF00FF})

	c.resetStats()

	stats := c.stats()
	if stats.Hits != 0 || stats.Misses != 0 || stats.Evictions != 0 {
		t.Errorf("after resetStats: Hits=%d Misses=%d Evictions=%d, want all 0",
			stats.Hits, stats.Misses, stats.Evictions)
	}
	if stats.DocHits != 0 || stats.DocMisses != 0 {
		t.Errorf("after resetStats: DocHits=%d DocMisses=%d, want all 0",
			stats.DocHits, stats.DocMisses)
	}
	// Entries should still be present.
	if stats.ImageEntries != 1 {
		t.Errorf("ImageEntries = %d, want 1 (resetStats should not clear entries)", stats.ImageEntries)
	}
}

// --- Thread Safety Tests ---

func TestIconCache_ConcurrentPutGet(t *testing.T) {
	c := newIconCache(64)
	const numGoroutines = 16
	const opsPerGoroutine = 50

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for g := range numGoroutines {
		go func(id int) {
			defer wg.Done()
			for i := range opsPerGoroutine {
				key := iconImageKey{
					svgPtr: uintptr(id*opsPerGoroutine + i),
					width:  24,
					height: 24,
					color:  uint32(id),
				}
				c.putImage(key, makeSceneImage(24, 24))
				_ = c.getImage(key)
			}
		}(g)
	}

	wg.Wait()

	stats := c.stats()
	if stats.ImageEntries > 64 {
		t.Errorf("ImageEntries = %d, should be <= maxItems (64)", stats.ImageEntries)
	}
}

func TestIconCache_ConcurrentDocAccess(t *testing.T) {
	c := newIconCache(16)
	const numGoroutines = 8

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for range numGoroutines {
		go func() {
			defer wg.Done()
			doc := c.getDoc(minimalSVG)
			if doc == nil {
				t.Error("expected non-nil document")
			}
		}()
	}

	wg.Wait()

	stats := c.stats()
	if stats.DocEntries != 1 {
		t.Errorf("DocEntries = %d, want 1 (same SVG)", stats.DocEntries)
	}
}

func TestIconCache_ConcurrentPutAndInvalidate(t *testing.T) {
	c := newIconCache(32)
	const numGoroutines = 8
	const opsPerGoroutine = 50

	var wg sync.WaitGroup
	wg.Add(numGoroutines * 2)

	for g := range numGoroutines {
		go func(id int) {
			defer wg.Done()
			for i := range opsPerGoroutine {
				key := iconImageKey{
					svgPtr: uintptr(id*opsPerGoroutine + i),
					width:  24,
					height: 24,
					color:  0xFF0000FF,
				}
				c.putImage(key, makeSceneImage(24, 24))
			}
		}(g)
		go func() {
			defer wg.Done()
			for range opsPerGoroutine {
				c.invalidateImages()
			}
		}()
	}

	wg.Wait()

	// Must not panic. Entries may or may not be present.
	stats := c.stats()
	if stats.ImageEntries < 0 {
		t.Errorf("ImageEntries = %d, should not be negative", stats.ImageEntries)
	}
}

// --- Public API Tests ---

func TestInvalidateIconImages(t *testing.T) {
	// Ensure the global function does not panic.
	InvalidateIconImages()
}

func TestInvalidateIconCache(t *testing.T) {
	// Ensure the global function does not panic.
	InvalidateIconCache()
}

func TestIconCacheStatsSnapshot(t *testing.T) {
	stats := IconCacheStatsSnapshot()
	if stats.MaxImageEntries != defaultIconCacheMaxEntries {
		t.Errorf("MaxImageEntries = %d, want %d", stats.MaxImageEntries, defaultIconCacheMaxEntries)
	}
}

// --- Global cache singleton test ---

func TestGlobalIconCache_Exists(t *testing.T) {
	if globalIconCache == nil {
		t.Fatal("globalIconCache should not be nil")
	}
	if globalIconCache.maxItems != defaultIconCacheMaxEntries {
		t.Errorf("maxItems = %d, want %d", globalIconCache.maxItems, defaultIconCacheMaxEntries)
	}
}
