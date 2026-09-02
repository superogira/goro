package primitives

// RasterCacheConfig controls heuristic parameters for display list stability
// tracking in [RepaintBoundary]. When a boundary's scene is replayed for
// several consecutive frames without invalidation (consecutiveHits >= threshold),
// and the content is complex enough (tag count > minComplexity) and large
// enough (area >= minArea), the boundary is promoted to "stable" status.
//
// Stable boundaries indicate to the compositor that these display lists rarely
// change and are candidates for GPU texture caching (Phase 4: actual texture
// promotion when gg exposes DrawGPUTexture through the Canvas interface).
//
// This follows the Flutter RasterCache pattern (raster_cache.cc):
//   - Track consecutive cache hits per boundary
//   - Promote after N consecutive hits with sufficient complexity
//   - Demote on cache invalidation (boundaryDirty)
//
// All fields have sane defaults via [DefaultRasterCacheConfig].
type RasterCacheConfig struct {
	// PromotionThreshold is the number of consecutive cache hits required
	// before a boundary is considered stable. Flutter uses 3.
	PromotionThreshold int

	// MinComplexity is the minimum number of encoding tags (draw commands)
	// in the scene to qualify for promotion. Simple scenes (e.g., a single
	// rectangle) are cheap to replay and don't benefit from caching.
	// Flutter uses ~20 encoding operations.
	MinComplexity int

	// MinArea is the minimum pixel area (width * height) for promotion.
	// Small boundaries have negligible replay cost and the overhead of
	// tracking/promoting them is not worthwhile.
	MinArea int
}

// DefaultRasterCacheConfig returns the default raster cache configuration.
//
// Values follow Flutter's RasterCache heuristics:
//   - PromotionThreshold: 3 consecutive cache hits
//   - MinComplexity: 20 encoding tags (draw commands)
//   - MinArea: 4096 pixels (64x64)
func DefaultRasterCacheConfig() RasterCacheConfig {
	return RasterCacheConfig{
		PromotionThreshold: defaultPromotionThreshold,
		MinArea:            defaultMinArea,
		MinComplexity:      defaultMinComplexity,
	}
}

// Defaults for raster cache heuristics.
const (
	defaultPromotionThreshold = 3    // consecutive cache hits before promotion
	defaultMinArea            = 4096 // 64x64 pixels
	defaultMinComplexity      = 20   // minimum encoding tags
)

// RasterCacheStats provides observability into the raster cache state of
// a [RepaintBoundary]. Used for diagnostics, benchmarks, and compositor
// optimization decisions.
type RasterCacheStats struct {
	// ConsecutiveHits is the number of consecutive cache hits since the
	// last cache miss (boundary dirty or first draw). Reset to 0 on miss.
	ConsecutiveHits int

	// IsStable is true when the boundary has been promoted to stable status
	// (consecutiveHits >= threshold, complexity and area qualifiers met).
	// Stable boundaries are candidates for GPU texture caching.
	IsStable bool

	// TagCount is the number of encoding tags in the cached scene.
	// Zero when no scene is cached.
	TagCount int

	// Area is the pixel area (width * height) of the boundary.
	Area int

	// TotalPromotions is the cumulative count of times this boundary
	// has been promoted to stable status (for diagnostics).
	TotalPromotions int

	// TotalDemotions is the cumulative count of times this boundary
	// has been demoted from stable status (for diagnostics).
	TotalDemotions int
}

// WithRasterCacheConfig sets a custom raster cache configuration for a
// [RepaintBoundary]. By default, [DefaultRasterCacheConfig] is used.
func WithRasterCacheConfig(cfg RasterCacheConfig) Option {
	return func(rb *RepaintBoundary) {
		rb.rasterCacheCfg = cfg
	}
}
