package render

import (
	"container/list"
	"sync"
	"sync/atomic"
	"unsafe"

	"github.com/gogpu/gg/scene"
	"github.com/gogpu/gg/svg"
	"github.com/gogpu/ui/widget"
)

// Icon cache configuration constants.
const (
	// defaultIconCacheMaxEntries is the default maximum number of rasterized
	// icon images cached in Level 2.
	defaultIconCacheMaxEntries = 256
)

// iconImageKey uniquely identifies a rasterized icon image.
// The cache stores *scene.Image keyed by the SVG data identity (pointer),
// output dimensions, and fill color.
type iconImageKey struct {
	svgPtr uintptr // pointer to SVG data (SliceData for []byte, string data ptr)
	width  int
	height int
	color  uint32 // packed RGBA (8 bits per channel)
}

// iconImageEntry is a single entry in the Level 2 rasterized image cache.
type iconImageEntry struct {
	key     iconImageKey
	img     *scene.Image
	element *list.Element
}

// IconCacheStats contains cache statistics for monitoring.
type IconCacheStats struct {
	// DocEntries is the number of parsed SVG documents in Level 1.
	DocEntries int
	// ImageEntries is the number of rasterized images in Level 2.
	ImageEntries int
	// MaxImageEntries is the maximum number of Level 2 entries.
	MaxImageEntries int
	// Hits is the number of Level 2 cache hits.
	Hits uint64
	// Misses is the number of Level 2 cache misses.
	Misses uint64
	// HitRate is the Level 2 hit rate (0.0 to 1.0).
	HitRate float64
	// Evictions is the number of Level 2 entries evicted.
	Evictions uint64
	// DocHits is the number of Level 1 cache hits.
	DocHits uint64
	// DocMisses is the number of Level 1 cache misses.
	DocMisses uint64
}

// iconCache provides a 2-level LRU cache for SVG icon rendering.
//
// Level 1 caches parsed [svg.Document] by SVG XML pointer, avoiding
// repeated XML parsing (typically ~0.5ms per parse). Documents are
// lightweight (a few KB each) and never evicted — the set of distinct
// SVG icons in a UI is bounded and small.
//
// Level 2 caches rasterized [scene.Image] by (svgPtr, width, height, color),
// avoiding repeated CPU rasterization (typically ~0.15ms per icon). Images
// are evicted via LRU when the entry count exceeds the configured maximum.
//
// The cache is a package-level singleton shared across all SceneCanvas
// instances so it survives RepaintBoundary re-recording. It is protected
// by sync.Mutex for thread safety, though in practice all access occurs on
// the main/UI thread.
type iconCache struct {
	mu sync.Mutex

	// Level 1: parsed SVG documents by data pointer.
	docs    map[uintptr]*svg.Document
	docHits atomic.Uint64
	docMiss atomic.Uint64

	// Level 2: rasterized scene images by composite key.
	images   map[iconImageKey]*iconImageEntry
	lru      *list.List // front = most recent
	maxItems int

	// Level 2 statistics (atomic for lock-free reads).
	hits      atomic.Uint64
	misses    atomic.Uint64
	evictions atomic.Uint64
}

// globalIconCache is the package-level singleton icon cache.
// Shared across all SceneCanvas instances, survives boundary re-recording.
var globalIconCache = newIconCache(defaultIconCacheMaxEntries)

// newIconCache creates an icon cache with the specified maximum Level 2 entries.
func newIconCache(maxItems int) *iconCache {
	if maxItems <= 0 {
		maxItems = defaultIconCacheMaxEntries
	}
	return &iconCache{
		docs:     make(map[uintptr]*svg.Document),
		images:   make(map[iconImageKey]*iconImageEntry),
		lru:      list.New(),
		maxItems: maxItems,
	}
}

// getDoc retrieves or parses an SVG document for the given XML data.
// Level 1 cache: keyed by the data pointer of the []byte slice header.
// Returns nil if parsing fails.
func (c *iconCache) getDoc(svgXML []byte) *svg.Document {
	ptr := svgSlicePtr(svgXML)

	c.mu.Lock()
	if doc, ok := c.docs[ptr]; ok {
		c.mu.Unlock()
		c.docHits.Add(1)
		return doc
	}
	c.mu.Unlock()

	// Parse outside the lock — svg.Parse is pure computation.
	doc, err := svg.Parse(svgXML)
	if err != nil {
		c.docMiss.Add(1)
		return nil
	}

	c.mu.Lock()
	// Double-check: another goroutine may have inserted while we parsed.
	if existing, ok := c.docs[ptr]; ok {
		c.mu.Unlock()
		c.docHits.Add(1)
		return existing
	}
	c.docs[ptr] = doc
	c.mu.Unlock()

	c.docMiss.Add(1)
	return doc
}

// getImage retrieves a cached rasterized image by its composite key.
// On cache hit, the entry is promoted to the front of the LRU list.
// Returns nil on cache miss.
func (c *iconCache) getImage(key iconImageKey) *scene.Image {
	c.mu.Lock()
	entry, ok := c.images[key]
	if !ok {
		c.mu.Unlock()
		c.misses.Add(1)
		return nil
	}
	c.lru.MoveToFront(entry.element)
	img := entry.img
	c.mu.Unlock()

	c.hits.Add(1)
	return img
}

// putImage stores a rasterized image in the Level 2 cache.
// Evicts the least recently used entry if the cache is full.
func (c *iconCache) putImage(key iconImageKey, img *scene.Image) {
	if img == nil {
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	// Replace existing entry with the same key.
	if existing, ok := c.images[key]; ok {
		c.lru.Remove(existing.element)
		delete(c.images, key)
	}

	// Evict LRU entries until under capacity.
	for c.lru.Len() >= c.maxItems {
		back := c.lru.Back()
		if back == nil {
			break
		}
		victim := back.Value.(*iconImageEntry)
		c.lru.Remove(back)
		delete(c.images, victim.key)
		c.evictions.Add(1)
	}

	entry := &iconImageEntry{
		key: key,
		img: img,
	}
	entry.element = c.lru.PushFront(entry)
	c.images[key] = entry
}

// invalidateImages clears all Level 2 rasterized images.
// Level 1 parsed documents are preserved because they are color-independent.
// Call this on theme change — colors change, parsed structure does not.
func (c *iconCache) invalidateImages() {
	c.mu.Lock()
	defer c.mu.Unlock()

	evicted := uint64(len(c.images))
	c.images = make(map[iconImageKey]*iconImageEntry)
	c.lru.Init()

	if evicted > 0 {
		c.evictions.Add(evicted)
	}
}

// invalidateAll clears both Level 1 and Level 2 caches entirely.
func (c *iconCache) invalidateAll() {
	c.mu.Lock()
	defer c.mu.Unlock()

	evicted := uint64(len(c.images))
	c.docs = make(map[uintptr]*svg.Document)
	c.images = make(map[iconImageKey]*iconImageEntry)
	c.lru.Init()

	if evicted > 0 {
		c.evictions.Add(evicted)
	}
}

// stats returns current cache statistics.
func (c *iconCache) stats() IconCacheStats {
	c.mu.Lock()
	docEntries := len(c.docs)
	imageEntries := len(c.images)
	maxItems := c.maxItems
	c.mu.Unlock()

	hits := c.hits.Load()
	misses := c.misses.Load()

	var hitRate float64
	total := hits + misses
	if total > 0 {
		hitRate = float64(hits) / float64(total)
	}

	return IconCacheStats{
		DocEntries:      docEntries,
		ImageEntries:    imageEntries,
		MaxImageEntries: maxItems,
		Hits:            hits,
		Misses:          misses,
		HitRate:         hitRate,
		Evictions:       c.evictions.Load(),
		DocHits:         c.docHits.Load(),
		DocMisses:       c.docMiss.Load(),
	}
}

// resetStats resets all hit/miss/eviction counters to zero.
func (c *iconCache) resetStats() {
	c.hits.Store(0)
	c.misses.Store(0)
	c.evictions.Store(0)
	c.docHits.Store(0)
	c.docMiss.Store(0)
}

// --- Public API for external access ---

// InvalidateIconImages clears all cached rasterized SVG icon images.
// Parsed SVG documents (Level 1) are preserved because they are
// color-independent. Call this when the theme changes.
func InvalidateIconImages() {
	globalIconCache.invalidateImages()
}

// InvalidateIconCache clears the entire icon cache (both levels).
func InvalidateIconCache() {
	globalIconCache.invalidateAll()
}

// IconCacheStatsSnapshot returns current icon cache statistics.
func IconCacheStatsSnapshot() IconCacheStats {
	return globalIconCache.stats()
}

// --- Helper functions ---

// packColor packs a widget.Color into a uint32 (8 bits per RGBA channel).
// This produces a deterministic key from float32 color values.
func packColor(color widget.Color) uint32 {
	r, g, b, a := color.RGBA8()
	return uint32(r)<<24 | uint32(g)<<16 | uint32(b)<<8 | uint32(a)
}

// svgSlicePtr extracts the data pointer from a []byte slice header.
// Icons are typically go:embed constants, so the pointer is stable for
// the process lifetime. This is the identity key for Level 1 caching.
func svgSlicePtr(data []byte) uintptr {
	if len(data) == 0 {
		return 0
	}
	return uintptr(unsafe.Pointer(unsafe.SliceData(data)))
}

// svgStringPtr extracts the data pointer from a string header.
// SVG path data strings are typically string constants, so the pointer
// is stable for the process lifetime.
func svgStringPtr(s string) uintptr {
	if s == "" {
		return 0
	}
	return uintptr(unsafe.Pointer(unsafe.StringData(s)))
}
