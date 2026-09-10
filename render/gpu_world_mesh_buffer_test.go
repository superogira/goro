package render

import (
	"math/rand/v2"
	"slices"
	"testing"
)

func TestWorldMeshBufferReusesAndCoalescesFreedRanges(t *testing.T) {
	page := &worldMeshBufferPage{free: []worldMeshBufferRange{{size: 16384}}}
	a, _ := page.allocate(4096)
	b, _ := page.allocate(4096)
	c, _ := page.allocate(4096)
	d, _ := page.allocate(4096)
	if _, ok := page.allocate(4096); ok {
		t.Fatal("allocation succeeded in a full page")
	}
	b.release()
	reused, ok := page.allocate(4096)
	if !ok || reused.offset != 4096 {
		t.Fatal("freed space was not reused")
	}
	a.release()
	c.release()
	reused.release() // Joins free ranges on both sides.
	large, ok := page.allocate(12288)
	if !ok || large.offset != 0 || d.offset != 12288 {
		t.Fatal("adjacent free ranges were not merged, or a live allocation moved")
	}
	d.release()
	large.release()
	large.release() // Releasing a cleared allocation must be harmless.
	if page.allocations != 0 || !slices.Equal(page.free, []worldMeshBufferRange{{size: 16384}}) {
		t.Fatal("releasing all allocations did not recover the entire page")
	}
}

func TestWorldMeshBufferVisibilityChurnDoesNotOverlapLiveGeometry(t *testing.T) {
	const pageSize = 65536
	page := &worldMeshBufferPage{free: []worldMeshBufferRange{{size: pageSize}}}
	rng := rand.New(rand.NewPCG(1, 2))
	var live []worldMeshBufferAllocation
	for range 2000 {
		if len(live) > 0 && rng.IntN(3) == 0 {
			i := rng.IntN(len(live))
			live[i].release()
			live = slices.Delete(live, i, i+1)
		} else if allocation, ok := page.allocate(nextBufferCapacity(rng.IntN(8192) + 1)); ok {
			live = append(live, allocation)
		}
		if page.allocations != len(live) {
			t.Fatal("page lost track of live allocations")
		}
		ranges := slices.Clone(page.free)
		for _, allocation := range live {
			ranges = append(ranges, allocation.worldMeshBufferRange)
			if allocation.offset%worldVertexStride != 0 {
				t.Fatal("geometry offset cannot be expressed as BaseVertex")
			}
		}
		slices.SortFunc(ranges, func(a, b worldMeshBufferRange) int { return a.offset - b.offset })
		end := 0
		for _, r := range ranges {
			if r.offset != end || r.size <= 0 {
				t.Fatalf("overlap or lost space at %d: %+v", end, ranges)
			}
			end += r.size
		}
		if end != pageSize {
			t.Fatalf("accounted for %d bytes, want %d", end, pageSize)
		}
		for i := 1; i < len(page.free); i++ {
			if page.free[i-1].offset+page.free[i-1].size >= page.free[i].offset {
				t.Fatal("free ranges are not sorted and coalesced")
			}
		}
	}
}
