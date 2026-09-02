package render

import (
	"container/list"
	"image"
	"sync"
	"sync/atomic"
)

// Default cache configuration constants.
const (
	// DefaultImageCacheMaxSizeMB is the default maximum cache size in megabytes.
	DefaultImageCacheMaxSizeMB = 64
	// imageCacheBytesPerMB is the number of bytes in a megabyte.
	imageCacheBytesPerMB = 1024 * 1024
	// imageCacheBytesPerPixel is the number of bytes per RGBA pixel.
	imageCacheBytesPerPixel = 4
)

// ImageCache provides a centralized LRU cache for RepaintBoundary pixel
// buffers (*image.RGBA). It is the ui-side equivalent of gg's scene.LayerCache,
// but stores image.RGBA instead of gg.Pixmap.
//
// The cache evicts least recently used entries when the memory limit is
// exceeded. Entries are keyed by a monotonic uint64 ID assigned to each
// RepaintBoundary at creation time.
//
// Thread Safety: All methods are safe for concurrent access. The cache uses
// a sync.RWMutex for read/write separation and atomic counters for statistics.
type ImageCache struct {
	mu      sync.RWMutex
	entries map[uint64]*imageCacheEntry // cacheKey -> entry
	lru     *list.List                  // LRU order (front = most recent)
	size    int64                       // Current memory usage in bytes.
	maxSize int64                       // Memory budget in bytes.

	// Statistics (atomic for lock-free reads).
	hits      atomic.Uint64
	misses    atomic.Uint64
	evictions atomic.Uint64
}

// imageCacheEntry represents a single cached image with metadata.
type imageCacheEntry struct {
	key     uint64
	img     *image.RGBA
	size    int64 // Memory size in bytes.
	version uint64
	element *list.Element
}

// ImageCacheStats contains cache statistics for monitoring.
type ImageCacheStats struct {
	// Size is the current memory usage in bytes.
	Size int64
	// MaxSize is the memory budget in bytes.
	MaxSize int64
	// Entries is the number of cached entries.
	Entries int
	// Hits is the number of cache hits.
	Hits uint64
	// Misses is the number of cache misses.
	Misses uint64
	// HitRate is the cache hit rate (0.0 to 1.0).
	HitRate float64
	// Evictions is the number of entries evicted.
	Evictions uint64
}

// NewImageCache creates a new image cache with the specified maximum size.
// The maxSizeMB parameter sets the memory budget in megabytes.
// If maxSizeMB is zero or negative, [DefaultImageCacheMaxSizeMB] (64MB) is used.
func NewImageCache(maxSizeMB int) *ImageCache {
	if maxSizeMB <= 0 {
		maxSizeMB = DefaultImageCacheMaxSizeMB
	}
	return &ImageCache{
		entries: make(map[uint64]*imageCacheEntry),
		lru:     list.New(),
		maxSize: int64(maxSizeMB) * imageCacheBytesPerMB,
	}
}

// DefaultImageCache creates a new image cache with the default 64MB limit.
func DefaultImageCache() *ImageCache {
	return NewImageCache(DefaultImageCacheMaxSizeMB)
}

// Get retrieves a cached image by its key.
// Returns the image and true if found, nil and false otherwise.
// On cache hit, the entry is moved to the front of the LRU list.
func (c *ImageCache) Get(key uint64) (*image.RGBA, bool) {
	c.mu.RLock()
	_, ok := c.entries[key]
	c.mu.RUnlock()

	if !ok {
		c.misses.Add(1)
		return nil, false
	}

	// Move to front (requires write lock).
	c.mu.Lock()
	// Re-check after acquiring write lock (entry may have been evicted).
	entry, ok := c.entries[key]
	if !ok {
		c.mu.Unlock()
		c.misses.Add(1)
		return nil, false
	}
	c.lru.MoveToFront(entry.element)
	img := entry.img
	c.mu.Unlock()

	c.hits.Add(1)
	return img, true
}

// GetVersion returns the version of a cached entry if it exists.
// Returns 0 and false if the entry is not found.
// This does not update the LRU order.
func (c *ImageCache) GetVersion(key uint64) (uint64, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if entry, ok := c.entries[key]; ok {
		return entry.version, true
	}
	return 0, false
}

// Put stores an image in the cache with the given key and version.
// If the cache exceeds its memory budget, least recently used entries are
// evicted. If an entry with the same key exists, it is replaced.
//
// Put is a no-op if img is nil or the image exceeds the entire cache budget.
func (c *ImageCache) Put(key uint64, img *image.RGBA, version uint64) {
	if img == nil {
		return
	}

	entrySize := imageRGBASize(img)
	if entrySize <= 0 {
		return
	}

	// Don't cache if single entry exceeds budget.
	if entrySize > c.maxSize {
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	// Remove existing entry with same key if present.
	if existing, ok := c.entries[key]; ok {
		c.size -= existing.size
		c.lru.Remove(existing.element)
	}

	// Evict entries until we have space.
	c.evictUntilSize(c.maxSize - entrySize)

	// Create new entry.
	entry := &imageCacheEntry{
		key:     key,
		img:     img,
		size:    entrySize,
		version: version,
	}
	entry.element = c.lru.PushFront(entry)
	c.entries[key] = entry
	c.size += entrySize
}

// Invalidate removes a specific entry from the cache by key.
func (c *ImageCache) Invalidate(key uint64) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if entry, ok := c.entries[key]; ok {
		c.lru.Remove(entry.element)
		c.size -= entry.size
		delete(c.entries, entry.key)
		c.evictions.Add(1)
	}
}

// InvalidateAll clears the entire cache.
func (c *ImageCache) InvalidateAll() {
	c.mu.Lock()
	defer c.mu.Unlock()

	evicted := uint64(len(c.entries))
	c.entries = make(map[uint64]*imageCacheEntry)
	c.lru.Init()
	c.size = 0

	if evicted > 0 {
		c.evictions.Add(evicted)
	}
}

// evictUntilSize evicts LRU entries until size is at or below target.
// Must be called with c.mu held.
func (c *ImageCache) evictUntilSize(targetSize int64) {
	for c.size > targetSize && c.lru.Len() > 0 {
		elem := c.lru.Back()
		if elem == nil {
			break
		}

		entry := elem.Value.(*imageCacheEntry)
		c.lru.Remove(elem)
		c.size -= entry.size
		delete(c.entries, entry.key)
		c.evictions.Add(1)
	}
}

// Stats returns current cache statistics.
func (c *ImageCache) Stats() ImageCacheStats {
	c.mu.RLock()
	size := c.size
	maxSize := c.maxSize
	entries := len(c.entries)
	c.mu.RUnlock()

	hits := c.hits.Load()
	misses := c.misses.Load()
	evictions := c.evictions.Load()

	var hitRate float64
	total := hits + misses
	if total > 0 {
		hitRate = float64(hits) / float64(total)
	}

	return ImageCacheStats{
		Size:      size,
		MaxSize:   maxSize,
		Entries:   entries,
		Hits:      hits,
		Misses:    misses,
		HitRate:   hitRate,
		Evictions: evictions,
	}
}

// Size returns the current memory usage in bytes.
func (c *ImageCache) Size() int64 {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.size
}

// MaxSize returns the memory budget in bytes.
func (c *ImageCache) MaxSize() int64 {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.maxSize
}

// EntryCount returns the number of entries in the cache.
func (c *ImageCache) EntryCount() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.entries)
}

// Contains checks if an entry with the given key exists in the cache.
// This does not update the LRU order.
func (c *ImageCache) Contains(key uint64) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	_, ok := c.entries[key]
	return ok
}

// ResetStats resets the hit, miss, and eviction counters to zero.
func (c *ImageCache) ResetStats() {
	c.hits.Store(0)
	c.misses.Store(0)
	c.evictions.Store(0)
}

// imageRGBASize calculates the memory size of an *image.RGBA in bytes.
func imageRGBASize(img *image.RGBA) int64 {
	if img == nil {
		return 0
	}
	bounds := img.Bounds()
	return int64(bounds.Dx()) * int64(bounds.Dy()) * imageCacheBytesPerPixel
}
