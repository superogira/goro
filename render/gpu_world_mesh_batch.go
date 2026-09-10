package render

import "fmt"

type worldMeshVersion struct {
	mesh    *WorldMesh
	version uint64
}

// WorldMesh geometry and render keys are immutable between versions. Reuse the
// grouping while the submitted sequence is unchanged, including duplicates.
type worldMeshSubmissionCache struct {
	meshes   []worldMeshVersion
	revision uint64
}

func (c *worldMeshSubmissionCache) matches(commands []WorldMeshCommand) bool {
	if c.revision == 0 || len(c.meshes) != len(commands) {
		return false
	}
	for i, command := range commands {
		if c.meshes[i].mesh != command.Mesh || (command.Mesh != nil && c.meshes[i].version != command.Mesh.version) {
			return false
		}
	}
	return true
}

func (c *worldMeshSubmissionCache) remember(commands []WorldMeshCommand) {
	clear(c.meshes)
	c.meshes = reserveSlice(c.meshes, len(commands))
	for _, command := range commands {
		entry := worldMeshVersion{mesh: command.Mesh}
		if command.Mesh != nil {
			entry.version = command.Mesh.version
		}
		c.meshes = append(c.meshes, entry)
	}
	c.revision++
}

// gpuWorldMeshBatch combines the currently visible meshes sharing a render key.
// Keep their order, including duplicate submissions, so merging draws preserves
// depth and blending results. Visibility changes rebuild only affected batches.
type gpuWorldMeshBatch struct {
	allocation  worldMeshBufferAllocation
	firstIndex  uint32
	indexCount  uint32
	meshes      []worldMeshVersion
	width       int
	height      int
	lightWidth  int
	lightHeight int
	revision    uint64
}

func (b *gpuWorldMeshBatch) matches(batch worldMeshBatch, width, height, lightWidth, lightHeight int) bool {
	meshes := batch.meshes
	if b.width != width || b.height != height || b.lightWidth != lightWidth || b.lightHeight != lightHeight || len(b.meshes) != len(meshes) {
		return false
	}
	// Only reuse validation for the exact submission revision that produced it.
	// Grouping can change while the camera is disabled and no batches upload.
	if batch.revision != 0 && b.revision == batch.revision {
		return true
	}
	for i, mesh := range meshes {
		if b.meshes[i].mesh != mesh || b.meshes[i].version != mesh.version {
			return false
		}
	}
	return true
}

func (b *gpuWorldMeshBatch) remember(batch worldMeshBatch, width, height, lightWidth, lightHeight int) {
	meshes := batch.meshes
	clear(b.meshes)
	b.meshes = reserveSlice(b.meshes, len(meshes))
	for _, mesh := range meshes {
		b.meshes = append(b.meshes, worldMeshVersion{mesh: mesh, version: mesh.version})
	}
	b.width, b.height = width, height
	b.lightWidth, b.lightHeight = lightWidth, lightHeight
	b.revision = batch.revision
}

func (b *gpuWorldMeshBatch) release() {
	b.allocation.release()
	*b = gpuWorldMeshBatch{}
}

func (r *gpuRenderer) pruneWorldMeshBatchCache() {
	// Retain only visible render keys, including across map changes. Otherwise
	// old batches would keep both their GPU buffers and entire map textures alive.
	for key, batch := range r.worldMeshBatchCache {
		if _, visible := r.worldMeshBatchByKey[key]; !visible {
			batch.release()
			delete(r.worldMeshBatchCache, key)
		}
	}
	r.pruneWorldMeshBufferPages()
}

func (r *gpuRenderer) ensureWorldMeshBatch(batch worldMeshBatch) (*gpuWorldMeshBatch, error) {
	width, height := batch.key.texture.Bounds().Dx(), batch.key.texture.Bounds().Dy()
	if width <= 0 || height <= 0 {
		return nil, fmt.Errorf("world mesh batch texture has invalid size")
	}
	lw, lh := lightTextureSize(batch.key.lightTexture, width, height)
	if r.worldMeshBatchCache == nil {
		r.worldMeshBatchCache = make(map[drawBatchKey]*gpuWorldMeshBatch)
	}
	cached := r.worldMeshBatchCache[batch.key]
	if cached == nil {
		cached = &gpuWorldMeshBatch{}
		r.worldMeshBatchCache[batch.key] = cached
	}
	if cached.matches(batch, width, height, lw, lh) {
		cached.revision = batch.revision
		return cached, nil
	}
	floats, indices := worldMeshBatchGPUData(r.worldMeshBatchFloats, r.worldMeshBatchIndices, batch, width, height, lw, lh)
	r.worldMeshBatchFloats, r.worldMeshBatchIndices = floats, indices
	vertexBytes := len(floats) * 4
	size := vertexBytes + len(indices)*4
	if cached.allocation.page == nil || cached.allocation.size < size {
		cached.allocation.release()
		allocation, err := r.allocateWorldMeshBuffer(size)
		if err != nil {
			return nil, err
		}
		cached.allocation = allocation
	}
	allocation := cached.allocation
	if err := r.queue.WriteBuffer(allocation.page.buf, uint64(allocation.offset), floatBytes(floats)); err != nil {
		return nil, err
	}
	if err := r.queue.WriteBuffer(allocation.page.buf, uint64(allocation.offset+vertexBytes), u32Bytes(indices)); err != nil {
		return nil, err
	}
	cached.firstIndex = uint32((allocation.offset + vertexBytes) / 4)
	cached.indexCount = uint32(len(indices))
	cached.remember(batch, width, height, lw, lh)
	return cached, nil
}

func worldMeshBatchGPUData(floats []float32, indices []uint32, batch worldMeshBatch, width, height, lightWidth, lightHeight int) ([]float32, []uint32) {
	vertexCount, indexCount := 0, 0
	for _, mesh := range batch.meshes {
		vertexCount += len(mesh.vertices)
		indexCount += len(mesh.indices)
	}
	floats = reserveSlice(floats, vertexCount*worldVertexFloatCount)
	indices = reserveSlice(indices, indexCount)
	for _, mesh := range batch.meshes {
		floats, indices = appendWorldCommand(floats, indices, WorldCommand{
			Vertices: mesh.vertices,
			Indices:  mesh.indices,
			Options:  mesh.options,
		}, width, height, lightWidth, lightHeight)
	}
	return floats, indices
}
