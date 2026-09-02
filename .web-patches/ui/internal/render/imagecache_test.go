package render

import (
	"container/list"
	"image"
	"sync"
	"testing"
)

// newLRUList creates a new container/list.List for test cache construction.
func newLRUList() *list.List {
	return list.New()
}

// makeImage creates a minimal *image.RGBA of the given dimensions.
func makeImage(w, h int) *image.RGBA {
	return image.NewRGBA(image.Rect(0, 0, w, h))
}

// --- Construction Tests ---

func TestNewImageCache_DefaultSize(t *testing.T) {
	c := DefaultImageCache()
	if c.MaxSize() != int64(DefaultImageCacheMaxSizeMB)*imageCacheBytesPerMB {
		t.Errorf("MaxSize = %d, want %d", c.MaxSize(), int64(DefaultImageCacheMaxSizeMB)*imageCacheBytesPerMB)
	}
	if c.EntryCount() != 0 {
		t.Errorf("EntryCount = %d, want 0", c.EntryCount())
	}
	if c.Size() != 0 {
		t.Errorf("Size = %d, want 0", c.Size())
	}
}

func TestNewImageCache_CustomSize(t *testing.T) {
	c := NewImageCache(32)
	want := int64(32) * imageCacheBytesPerMB
	if c.MaxSize() != want {
		t.Errorf("MaxSize = %d, want %d", c.MaxSize(), want)
	}
}

func TestNewImageCache_ZeroOrNegativeUsesDefault(t *testing.T) {
	for _, mb := range []int{0, -1, -100} {
		c := NewImageCache(mb)
		want := int64(DefaultImageCacheMaxSizeMB) * imageCacheBytesPerMB
		if c.MaxSize() != want {
			t.Errorf("NewImageCache(%d): MaxSize = %d, want %d", mb, c.MaxSize(), want)
		}
	}
}

// --- Put/Get Round Trip ---

func TestImageCache_PutGet_RoundTrip(t *testing.T) {
	c := NewImageCache(1)
	img := makeImage(10, 10)

	c.Put(1, img, 42)

	got, ok := c.Get(1)
	if !ok {
		t.Fatal("expected cache hit")
	}
	if got != img {
		t.Error("returned image does not match stored image")
	}
}

func TestImageCache_Get_Miss(t *testing.T) {
	c := NewImageCache(1)

	got, ok := c.Get(999)
	if ok {
		t.Error("expected cache miss for nonexistent key")
	}
	if got != nil {
		t.Error("expected nil image on miss")
	}

	stats := c.Stats()
	if stats.Misses != 1 {
		t.Errorf("Misses = %d, want 1", stats.Misses)
	}
}

func TestImageCache_Put_NilImage(t *testing.T) {
	c := NewImageCache(1)
	c.Put(1, nil, 0)

	if c.EntryCount() != 0 {
		t.Error("nil image should not be stored")
	}
}

func TestImageCache_Put_ReplacesExistingKey(t *testing.T) {
	c := NewImageCache(1)
	img1 := makeImage(10, 10)
	img2 := makeImage(20, 20)

	c.Put(1, img1, 1)
	c.Put(1, img2, 2)

	if c.EntryCount() != 1 {
		t.Errorf("EntryCount = %d, want 1 (should replace, not add)", c.EntryCount())
	}

	got, ok := c.Get(1)
	if !ok {
		t.Fatal("expected cache hit after replace")
	}
	if got != img2 {
		t.Error("expected replaced image")
	}

	ver, ok := c.GetVersion(1)
	if !ok || ver != 2 {
		t.Errorf("GetVersion = (%d, %v), want (2, true)", ver, ok)
	}
}

func TestImageCache_Put_ZeroSizeImage(t *testing.T) {
	c := NewImageCache(1)
	img := makeImage(0, 0)
	c.Put(1, img, 0)

	if c.EntryCount() != 0 {
		t.Error("zero-size image should not be stored")
	}
}

func TestImageCache_Put_ExceedsBudget(t *testing.T) {
	// Create cache with 1MB budget.
	c := NewImageCache(1)
	// Create an image larger than 1MB: 1024*1024 / 4 = 262144 pixels.
	// An image 600x600 = 360000 pixels = 1.37MB > 1MB budget.
	bigImg := makeImage(600, 600)
	c.Put(1, bigImg, 0)

	if c.EntryCount() != 0 {
		t.Error("image exceeding entire budget should not be stored")
	}
}

// --- LRU Eviction Tests ---

func TestImageCache_LRUEviction(t *testing.T) {
	// Create a tiny cache: 1 MB.
	c := NewImageCache(1)

	// Each 100x100 image = 40,000 bytes = ~0.038 MB.
	// We can fit ~26 of them in 1MB.
	imgs := make([]*image.RGBA, 30)
	for i := range imgs {
		imgs[i] = makeImage(100, 100)
		c.Put(uint64(i), imgs[i], 0)
	}

	// The first few entries should have been evicted.
	if c.EntryCount() >= 30 {
		t.Errorf("expected evictions, EntryCount = %d", c.EntryCount())
	}

	// Last entries should still be present.
	_, ok := c.Get(29)
	if !ok {
		t.Error("most recent entry (29) should still be in cache")
	}

	stats := c.Stats()
	if stats.Evictions == 0 {
		t.Error("expected at least one eviction")
	}
	if stats.Size > stats.MaxSize {
		t.Errorf("cache size %d exceeds max %d", stats.Size, stats.MaxSize)
	}
}

func TestImageCache_LRU_AccessPromotes(t *testing.T) {
	// Create a cache that can hold exactly 2 images of 100x100 (40KB each).
	// Use NewImageCacheWithMaxBytes for precise control.
	c := newImageCacheWithMaxBytes(80001) // Just over 2 images of 100x100.

	img1 := makeImage(100, 100) // 40000 bytes
	img2 := makeImage(100, 100) // 40000 bytes
	img3 := makeImage(100, 100) // 40000 bytes

	c.Put(1, img1, 0) // [1]
	c.Put(2, img2, 0) // [2, 1]

	// Access img1 to promote it to front: [1, 2].
	_, _ = c.Get(1)

	// Insert img3: should evict img2 (LRU), not img1.
	c.Put(3, img3, 0)

	if c.Contains(2) {
		t.Error("entry 2 should have been evicted (LRU)")
	}
	if !c.Contains(1) {
		t.Error("entry 1 should still be present (recently accessed)")
	}
	if !c.Contains(3) {
		t.Error("entry 3 should be present (just inserted)")
	}
}

// newImageCacheWithMaxBytes creates an ImageCache with the given max size
// in bytes (for precise testing). Not exported — test-only.
func newImageCacheWithMaxBytes(maxBytes int64) *ImageCache {
	return &ImageCache{
		entries: make(map[uint64]*imageCacheEntry),
		lru:     newLRUList(),
		maxSize: maxBytes,
	}
}

// --- Invalidate Tests ---

func TestImageCache_Invalidate_RemovesEntry(t *testing.T) {
	c := NewImageCache(1)
	img := makeImage(10, 10)
	c.Put(1, img, 0)

	c.Invalidate(1)

	if c.Contains(1) {
		t.Error("entry should be removed after Invalidate")
	}
	if c.EntryCount() != 0 {
		t.Errorf("EntryCount = %d, want 0", c.EntryCount())
	}
	if c.Size() != 0 {
		t.Errorf("Size = %d, want 0", c.Size())
	}

	stats := c.Stats()
	if stats.Evictions != 1 {
		t.Errorf("Evictions = %d, want 1", stats.Evictions)
	}
}

func TestImageCache_Invalidate_Nonexistent(t *testing.T) {
	c := NewImageCache(1)
	// Should not panic or error.
	c.Invalidate(999)

	if c.EntryCount() != 0 {
		t.Error("should remain empty")
	}
}

func TestImageCache_InvalidateAll(t *testing.T) {
	c := NewImageCache(1)
	for i := range 5 {
		c.Put(uint64(i), makeImage(10, 10), 0)
	}

	if c.EntryCount() != 5 {
		t.Fatalf("pre-condition: EntryCount = %d, want 5", c.EntryCount())
	}

	c.InvalidateAll()

	if c.EntryCount() != 0 {
		t.Errorf("EntryCount = %d after InvalidateAll, want 0", c.EntryCount())
	}
	if c.Size() != 0 {
		t.Errorf("Size = %d after InvalidateAll, want 0", c.Size())
	}

	stats := c.Stats()
	if stats.Evictions != 5 {
		t.Errorf("Evictions = %d, want 5", stats.Evictions)
	}
}

// --- Version Tests ---

func TestImageCache_GetVersion_Exists(t *testing.T) {
	c := NewImageCache(1)
	c.Put(1, makeImage(10, 10), 42)

	ver, ok := c.GetVersion(1)
	if !ok {
		t.Fatal("expected entry to exist")
	}
	if ver != 42 {
		t.Errorf("version = %d, want 42", ver)
	}
}

func TestImageCache_GetVersion_NotFound(t *testing.T) {
	c := NewImageCache(1)

	ver, ok := c.GetVersion(999)
	if ok {
		t.Error("expected not found")
	}
	if ver != 0 {
		t.Errorf("version = %d, want 0", ver)
	}
}

// --- Statistics Tests ---

func TestImageCache_Stats_HitsAndMisses(t *testing.T) {
	c := NewImageCache(1)
	c.Put(1, makeImage(10, 10), 0)

	// 2 hits
	_, _ = c.Get(1)
	_, _ = c.Get(1)
	// 1 miss
	_, _ = c.Get(999)

	stats := c.Stats()
	if stats.Hits != 2 {
		t.Errorf("Hits = %d, want 2", stats.Hits)
	}
	if stats.Misses != 1 {
		t.Errorf("Misses = %d, want 1", stats.Misses)
	}
	// HitRate = 2/3 ~ 0.666
	if stats.HitRate < 0.65 || stats.HitRate > 0.68 {
		t.Errorf("HitRate = %f, want ~0.666", stats.HitRate)
	}
}

func TestImageCache_Stats_EmptyCache(t *testing.T) {
	c := NewImageCache(1)
	stats := c.Stats()

	if stats.HitRate != 0 {
		t.Errorf("HitRate = %f, want 0 (no accesses)", stats.HitRate)
	}
	if stats.Size != 0 {
		t.Errorf("Size = %d, want 0", stats.Size)
	}
}

func TestImageCache_ResetStats(t *testing.T) {
	c := NewImageCache(1)
	c.Put(1, makeImage(10, 10), 0)
	_, _ = c.Get(1) // hit
	_, _ = c.Get(2) // miss

	c.ResetStats()

	stats := c.Stats()
	if stats.Hits != 0 || stats.Misses != 0 || stats.Evictions != 0 {
		t.Errorf("after ResetStats: Hits=%d Misses=%d Evictions=%d, want all 0",
			stats.Hits, stats.Misses, stats.Evictions)
	}
}

// --- Thread Safety Tests ---

func TestImageCache_ConcurrentAccess(t *testing.T) {
	c := NewImageCache(4) // 4MB budget
	const numGoroutines = 16
	const opsPerGoroutine = 100

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for g := range numGoroutines {
		go func(id int) {
			defer wg.Done()
			for i := range opsPerGoroutine {
				key := uint64(id*opsPerGoroutine + i)
				img := makeImage(10, 10)
				c.Put(key, img, uint64(i))
				_, _ = c.Get(key)
				_ = c.Contains(key)
				_, _ = c.GetVersion(key)
			}
		}(g)
	}

	wg.Wait()

	// Verify cache is in a consistent state.
	stats := c.Stats()
	if stats.Size < 0 {
		t.Errorf("Size = %d, should not be negative", stats.Size)
	}
	if stats.Size > stats.MaxSize {
		t.Errorf("Size %d exceeds MaxSize %d", stats.Size, stats.MaxSize)
	}
}

func TestImageCache_ConcurrentPutAndInvalidate(t *testing.T) {
	c := NewImageCache(2)
	const numGoroutines = 8
	const opsPerGoroutine = 50

	var wg sync.WaitGroup
	wg.Add(numGoroutines * 2) // half Put, half Invalidate

	for g := range numGoroutines {
		go func(id int) {
			defer wg.Done()
			for i := range opsPerGoroutine {
				key := uint64(id*opsPerGoroutine + i)
				c.Put(key, makeImage(10, 10), 0)
			}
		}(g)
		go func(id int) {
			defer wg.Done()
			for i := range opsPerGoroutine {
				key := uint64(id*opsPerGoroutine + i)
				c.Invalidate(key)
			}
		}(g)
	}

	wg.Wait()

	// Must not panic, size must be consistent.
	stats := c.Stats()
	if stats.Size < 0 {
		t.Errorf("Size = %d, should not be negative", stats.Size)
	}
}

// --- imageRGBASize Tests ---

func TestImageRGBASize(t *testing.T) {
	tests := []struct {
		name string
		img  *image.RGBA
		want int64
	}{
		{"nil", nil, 0},
		{"0x0", makeImage(0, 0), 0},
		{"1x1", makeImage(1, 1), 4},
		{"10x10", makeImage(10, 10), 400},
		{"100x100", makeImage(100, 100), 40000},
		{"256x256", makeImage(256, 256), 262144},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := imageRGBASize(tt.img)
			if got != tt.want {
				t.Errorf("imageRGBASize = %d, want %d", got, tt.want)
			}
		})
	}
}
