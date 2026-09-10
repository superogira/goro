package render

import (
	"slices"

	"github.com/gogpu/wgpu"
)

const worldMeshBufferPageSize = 4 << 20

type worldMeshBufferRange struct {
	offset, size int
}

// A page stores both vertices and indices. Every allocation starts on a vertex
// boundary, so draws can use BaseVertex and FirstIndex without rebinding buffers.
// Free ranges are sorted and coalesced to reuse space as visibility changes.
type worldMeshBufferPage struct {
	buf         *wgpu.Buffer
	free        []worldMeshBufferRange
	allocations int
}

type worldMeshBufferAllocation struct {
	page *worldMeshBufferPage
	worldMeshBufferRange
}

func (p *worldMeshBufferPage) allocate(size int) (worldMeshBufferAllocation, bool) {
	for i, available := range p.free {
		if available.size < size {
			continue
		}
		p.free[i].offset += size
		p.free[i].size -= size
		if p.free[i].size == 0 {
			p.free = slices.Delete(p.free, i, i+1)
		}
		p.allocations++
		return worldMeshBufferAllocation{page: p, worldMeshBufferRange: worldMeshBufferRange{offset: available.offset, size: size}}, true
	}
	return worldMeshBufferAllocation{}, false
}

func (a *worldMeshBufferAllocation) release() {
	if a.page == nil {
		return
	}
	p := a.page
	i, _ := slices.BinarySearchFunc(p.free, a.offset, func(r worldMeshBufferRange, offset int) int { return r.offset - offset })
	p.free = slices.Insert(p.free, i, a.worldMeshBufferRange)
	if i > 0 && p.free[i-1].offset+p.free[i-1].size == p.free[i].offset {
		p.free[i-1].size += p.free[i].size
		p.free = slices.Delete(p.free, i, i+1)
		i--
	}
	if i+1 < len(p.free) && p.free[i].offset+p.free[i].size == p.free[i+1].offset {
		p.free[i].size += p.free[i+1].size
		p.free = slices.Delete(p.free, i+1, i+2)
	}
	p.allocations--
	*a = worldMeshBufferAllocation{}
}

func (r *gpuRenderer) allocateWorldMeshBuffer(size int) (worldMeshBufferAllocation, error) {
	// Geometric capacity growth lets a visible batch grow without relocating on
	// every added mesh. Capacities are multiples of worldVertexStride.
	capacity := nextBufferCapacity(size)
	for _, page := range r.worldMeshBufferPages {
		if allocation, ok := page.allocate(capacity); ok {
			return allocation, nil
		}
	}
	pageSize := max(worldMeshBufferPageSize, capacity)
	buf, err := r.dev.CreateBuffer(&wgpu.BufferDescriptor{
		Label: "goro-world-mesh-page",
		Size:  uint64(pageSize),
		Usage: wgpu.BufferUsageVertex | wgpu.BufferUsageIndex | wgpu.BufferUsageCopyDst,
	})
	if err != nil {
		return worldMeshBufferAllocation{}, err
	}
	page := &worldMeshBufferPage{buf: buf, free: []worldMeshBufferRange{{size: pageSize}}}
	r.worldMeshBufferPages = append(r.worldMeshBufferPages, page)
	allocation, _ := page.allocate(capacity)
	return allocation, nil
}

func (r *gpuRenderer) pruneWorldMeshBufferPages() {
	pages := r.worldMeshBufferPages[:0]
	for _, page := range r.worldMeshBufferPages {
		if page.allocations == 0 {
			page.buf.Release()
			page.buf = nil
		} else {
			pages = append(pages, page)
		}
	}
	clear(r.worldMeshBufferPages[len(pages):])
	r.worldMeshBufferPages = pages
}
