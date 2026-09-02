package listview

// heightMode represents how item heights are determined.
type heightMode uint8

const (
	heightFixed    heightMode = iota // all items have the same height
	heightCallback                   // user provides height per index
	heightLazy                       // measure on scroll, estimate unmeasured
)

// heightManager manages item heights and cumulative offsets for all three
// height modes. It provides O(1) lookup for fixed height and O(log n) binary
// search for variable heights.
type heightManager struct {
	mode  heightMode
	count int

	// Fixed mode fields.
	fixedHeight float32

	// Callback mode fields.
	heightFn func(index int) float32

	// Lazy mode fields.
	estimatedHeight float32

	// Shared: cumulative offsets. offsets[i] = sum of heights[0..i-1].
	// offsets[0] = 0, offsets[count] = total height.
	// Only allocated for callback and lazy modes.
	offsets []float32

	// Lazy mode: measured heights. -1 means not yet measured.
	measured     []float32
	numMeasured  int
	measuredSum  float32
	offsetsDirty bool
}

// newHeightManager creates a height manager based on the config.
func newHeightManager(cfg *config) heightManager {
	hm := heightManager{
		count: cfg.ResolvedItemCount(),
	}

	switch {
	case cfg.hasFixedHeight():
		hm.mode = heightFixed
		hm.fixedHeight = cfg.fixedItemHeight

	case cfg.hasItemHeightFn():
		hm.mode = heightCallback
		hm.heightFn = cfg.itemHeightFn
		hm.buildCallbackOffsets()

	default:
		hm.mode = heightLazy
		hm.estimatedHeight = cfg.estimatedItemHeight
		if hm.estimatedHeight <= 0 {
			hm.estimatedHeight = defaultEstimatedHeight
		}
		hm.initLazy()
	}

	return hm
}

// updateCount adjusts the height manager for a new item count.
func (h *heightManager) updateCount(newCount int) {
	if newCount == h.count {
		return
	}
	oldCount := h.count
	h.count = newCount

	switch h.mode {
	case heightFixed:
		// Nothing to update.

	case heightCallback:
		h.buildCallbackOffsets()

	case heightLazy:
		if newCount > oldCount {
			// Extend slices.
			for i := oldCount; i < newCount; i++ {
				h.measured = append(h.measured, -1)
			}
		} else if newCount < oldCount {
			h.measured = h.measured[:newCount]
			// Recompute measured stats.
			h.numMeasured = 0
			h.measuredSum = 0
			for _, m := range h.measured {
				if m >= 0 {
					h.numMeasured++
					h.measuredSum += m
				}
			}
		}
		h.offsetsDirty = true
	}
}

// totalHeight returns the total height of all items.
func (h *heightManager) totalHeight() float32 {
	switch h.mode {
	case heightFixed:
		return float32(h.count) * h.fixedHeight

	case heightCallback:
		if len(h.offsets) > h.count {
			return h.offsets[h.count]
		}
		return 0

	case heightLazy:
		h.ensureOffsets()
		if len(h.offsets) > h.count {
			return h.offsets[h.count]
		}
		return float32(h.count) * h.currentEstimate()
	}
	return 0
}

// heightAt returns the height of the item at the given index.
func (h *heightManager) heightAt(index int) float32 {
	if index < 0 || index >= h.count {
		return 0
	}

	switch h.mode {
	case heightFixed:
		return h.fixedHeight

	case heightCallback:
		if h.heightFn != nil {
			return h.heightFn(index)
		}
		return 0

	case heightLazy:
		if index < len(h.measured) && h.measured[index] >= 0 {
			return h.measured[index]
		}
		return h.currentEstimate()
	}
	return 0
}

// offsetAt returns the Y offset of the item at the given index.
func (h *heightManager) offsetAt(index int) float32 {
	if index <= 0 {
		return 0
	}
	if index > h.count {
		index = h.count
	}

	switch h.mode {
	case heightFixed:
		return float32(index) * h.fixedHeight

	case heightCallback:
		if index < len(h.offsets) {
			return h.offsets[index]
		}
		return 0

	case heightLazy:
		h.ensureOffsets()
		if index < len(h.offsets) {
			return h.offsets[index]
		}
		return float32(index) * h.currentEstimate()
	}
	return 0
}

// setMeasured records the measured height for an item in lazy mode.
func (h *heightManager) setMeasured(index int, height float32) {
	if h.mode != heightLazy || index < 0 || index >= h.count {
		return
	}
	if index >= len(h.measured) {
		return
	}

	old := h.measured[index]
	if old == height {
		return
	}

	if old >= 0 {
		// Update existing measurement.
		h.measuredSum += height - old
	} else {
		// New measurement.
		h.numMeasured++
		h.measuredSum += height
	}
	h.measured[index] = height
	h.offsetsDirty = true
}

// findIndexAtOffset returns the item index containing the given pixel offset
// using binary search on cumulative offsets.
func (h *heightManager) findIndexAtOffset(offset float32) int {
	if offset <= 0 || h.count == 0 {
		return 0
	}

	switch h.mode {
	case heightFixed:
		if h.fixedHeight <= 0 {
			return 0
		}
		idx := int(offset / h.fixedHeight)
		if idx >= h.count {
			idx = h.count - 1
		}
		return idx

	case heightCallback, heightLazy:
		if h.mode == heightLazy {
			h.ensureOffsets()
		}
		return h.binarySearchOffset(offset)
	}
	return 0
}

// visibleRange returns [start, end) of items visible in the viewport,
// including the overscan buffer.
func (h *heightManager) visibleRange(scrollY, viewportH float32, overscan int) (int, int) {
	if h.count == 0 {
		return 0, 0
	}

	start := h.findIndexAtOffset(scrollY)
	end := h.findIndexAtOffset(scrollY+viewportH) + 1

	// Apply overscan.
	start -= overscan
	if start < 0 {
		start = 0
	}
	end += overscan
	if end > h.count {
		end = h.count
	}

	return start, end
}

// binarySearchOffset finds the largest index i where offsets[i] <= offset.
func (h *heightManager) binarySearchOffset(offset float32) int {
	n := len(h.offsets)
	if n == 0 {
		return 0
	}

	lo, hi := 0, n-1
	for lo < hi {
		mid := lo + (hi-lo+1)/2
		if h.offsets[mid] <= offset {
			lo = mid
		} else {
			hi = mid - 1
		}
	}

	// Clamp to valid item range.
	if lo >= h.count {
		lo = h.count - 1
	}
	if lo < 0 {
		lo = 0
	}
	return lo
}

// buildCallbackOffsets builds cumulative offsets from the height callback.
func (h *heightManager) buildCallbackOffsets() {
	h.offsets = make([]float32, h.count+1)
	h.offsets[0] = 0
	for i := 0; i < h.count; i++ {
		h.offsets[i+1] = h.offsets[i] + h.heightFn(i)
	}
}

// initLazy initializes lazy mode measured array.
func (h *heightManager) initLazy() {
	h.measured = make([]float32, h.count)
	for i := range h.measured {
		h.measured[i] = -1
	}
	h.numMeasured = 0
	h.measuredSum = 0
	h.offsetsDirty = true
}

// ensureOffsets rebuilds cumulative offsets for lazy mode if dirty.
func (h *heightManager) ensureOffsets() {
	if !h.offsetsDirty && len(h.offsets) == h.count+1 {
		return
	}

	est := h.currentEstimate()
	h.offsets = make([]float32, h.count+1)
	h.offsets[0] = 0
	for i := 0; i < h.count; i++ {
		ht := est
		if i < len(h.measured) && h.measured[i] >= 0 {
			ht = h.measured[i]
		}
		h.offsets[i+1] = h.offsets[i] + ht
	}
	h.offsetsDirty = false
}

// currentEstimate returns the current height estimate based on measured items.
func (h *heightManager) currentEstimate() float32 {
	if h.numMeasured > 0 {
		return h.measuredSum / float32(h.numMeasured)
	}
	return h.estimatedHeight
}
