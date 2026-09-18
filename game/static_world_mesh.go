package game

import (
	"image"
	"image/color"

	"github.com/kivutar/goro/render"
	"github.com/kivutar/goro/res"
)

const (
	retainedWorldMeshMaxVertices = 65535
	gndRetainedChunkSize         = 16
)

type retainedWorldMesh struct {
	mesh *render.WorldMesh
}

type gndRetainedMeshCache struct {
	gnd            *res.GND
	rsw            *res.RSW
	groundLightmap gndGroundLightmap
	chunks         map[gndRetainedChunkKey][]retainedWorldMesh
}

type gndRetainedChunkKey struct {
	x int
	y int
}

type retainedMeshKey struct {
	texture      *render.Image
	lightTexture *render.Image
	options      render.DrawTrianglesOptions
}

type retainedMeshBuilder struct {
	texture      *render.Image
	lightTexture *render.Image
	options      render.DrawTrianglesOptions
	vertices     []render.Vertex3D
	indices      []uint16
	meshes       []retainedWorldMesh
}

func (b *retainedMeshBuilder) addTriangle(a, c, d render.Vertex3D) {
	if b.texture == nil {
		return
	}
	if len(b.vertices)+3 > retainedWorldMeshMaxVertices {
		b.flush()
	}
	base := uint16(len(b.vertices))
	b.vertices = append(b.vertices, a, c, d)
	b.indices = append(b.indices, base, base+1, base+2)
}

func (b *retainedMeshBuilder) addQuad(vertices [4]render.Vertex3D, indices []uint16) {
	if b.texture == nil {
		return
	}
	if len(b.vertices)+len(vertices) > retainedWorldMeshMaxVertices {
		b.flush()
	}
	base := uint16(len(b.vertices))
	b.vertices = append(b.vertices, vertices[:]...)
	for _, index := range indices {
		b.indices = append(b.indices, base+index)
	}
}

func (b *retainedMeshBuilder) flush() {
	if len(b.vertices) == 0 || len(b.indices) == 0 || b.texture == nil {
		b.vertices = b.vertices[:0]
		b.indices = b.indices[:0]
		return
	}
	mesh := render.NewWorldMeshWithLightmap(b.vertices, b.indices, b.texture, b.lightTexture, &b.options)
	b.meshes = append(b.meshes, retainedWorldMesh{mesh: mesh})
	b.vertices = nil
	b.indices = nil
}

func (m *WorldMode) drawGNDMeshes(screen *render.Frame, manager *res.Manager, gnd *res.GND, rsw *res.RSW, projection sceneProjection) {
	if gnd == nil {
		return
	}
	cache := m.gndMeshCache
	if cache == nil || cache.gnd != gnd || cache.rsw != rsw {
		cache = &gndRetainedMeshCache{gnd: gnd, rsw: rsw, groundLightmap: buildGNDGroundLightmap(gnd), chunks: make(map[gndRetainedChunkKey][]retainedWorldMesh)}
		m.gndMeshCache = cache
	}
	width := screen.Bounds().Dx()
	height := screen.Bounds().Dy()
	startX, endX, startY, endY, ok := gndDrawBounds(gnd, projection, width, height)
	if !ok {
		return
	}
	startChunkX := startX / gndRetainedChunkSize
	endChunkX := endX / gndRetainedChunkSize
	startChunkY := startY / gndRetainedChunkSize
	endChunkY := endY / gndRetainedChunkSize
	for chunkY := startChunkY; chunkY <= endChunkY; chunkY++ {
		for chunkX := startChunkX; chunkX <= endChunkX; chunkX++ {
			key := gndRetainedChunkKey{x: chunkX, y: chunkY}
			meshes, ok := cache.chunks[key]
			if !ok {
				x0 := chunkX * gndRetainedChunkSize
				y0 := chunkY * gndRetainedChunkSize
				x1 := minInt(gnd.Width-1, x0+gndRetainedChunkSize-1)
				y1 := minInt(gnd.Height-1, y0+gndRetainedChunkSize-1)
				meshes = m.buildGNDMeshChunk(manager, gnd, rsw, cache.groundLightmap, x0, x1, y0, y1)
				cache.chunks[key] = meshes
			}
			for _, mesh := range meshes {
				screen.DrawWorldMesh(mesh.mesh)
			}
		}
	}
}

func (m *WorldMode) buildGNDMeshChunk(manager *res.Manager, gnd *res.GND, rsw *res.RSW, groundLightmap gndGroundLightmap, startX, endX, startY, endY int) []retainedWorldMesh {
	if gnd == nil {
		return nil
	}
	startX = maxInt(0, startX)
	startY = maxInt(0, startY)
	endX = minInt(gnd.Width-1, endX)
	endY = minInt(gnd.Height-1, endY)
	if startX > endX || startY > endY {
		return nil
	}
	if m.whitePixel == nil {
		m.whitePixel = render.NewImage(1, 1)
		m.whitePixel.Fill(color.White)
	}
	lighting := m.sceneLighting(rsw)
	topNormals := m.smoothGNDTopNormals(gnd)
	builders := make(map[retainedMeshKey]*retainedMeshBuilder)
	builderFor := func(texture, lightTexture *render.Image, options *render.DrawTrianglesOptions) *retainedMeshBuilder {
		if texture == nil || options == nil {
			return nil
		}
		key := retainedMeshKey{texture: texture, lightTexture: lightTexture, options: *options}
		builder := builders[key]
		if builder == nil {
			builder = &retainedMeshBuilder{texture: texture, lightTexture: lightTexture, options: *options}
			builders[key] = builder
		}
		return builder
	}
	addTextured := func(texture *render.Image, verts [4]modelPoint3, uvs [4]texturePoint, indices []uint16, tints [4]color.RGBA, options *render.DrawTrianglesOptions) {
		builder := builderFor(texture, nil, options)
		if builder == nil {
			return
		}
		bounds := texture.Bounds()
		w, h := float32(bounds.Dx()), float32(bounds.Dy())
		builder.addQuad([4]render.Vertex3D{
			texturedSurfaceVertex3D(verts[0], uvs[0], tints[0], w, h),
			texturedSurfaceVertex3D(verts[1], uvs[1], tints[1], w, h),
			texturedSurfaceVertex3D(verts[2], uvs[2], tints[2], w, h),
			texturedSurfaceVertex3D(verts[3], uvs[3], tints[3], w, h),
		}, indices)
	}
	addColored := func(verts [4]modelPoint3, indices []uint16, tints [4]color.RGBA, options *render.DrawTrianglesOptions) {
		builder := builderFor(m.whitePixel, nil, options)
		if builder == nil {
			return
		}
		builder.addQuad([4]render.Vertex3D{
			coloredSurfaceVertex3D(verts[0], 0, 0, tints[0]),
			coloredSurfaceVertex3D(verts[1], 1, 0, tints[1]),
			coloredSurfaceVertex3D(verts[2], 1, 1, tints[2]),
			coloredSurfaceVertex3D(verts[3], 0, 1, tints[3]),
		}, indices)
	}
	addLightmapped := func(texture *render.Image, verts [4]modelPoint3, uvs [4]texturePoint, baseTints [4]color.RGBA, lightUVs [4]texturePoint, lightScales [4]modelPoint3) {
		if texture == nil || groundLightmap.image == nil {
			addTextured(texture, verts, uvs, quadIndices012023, scaleSurfaceVertexTints(baseTints, lightScales), groundTextureDrawOptions())
			return
		}
		textureBounds := texture.Bounds()
		textureWidth := float32(textureBounds.Dx())
		textureHeight := float32(textureBounds.Dy())
		options := groundTextureDrawOptions()
		builder := builderFor(texture, groundLightmap.image, options)
		if builder == nil {
			return
		}
		var vertices [4]render.Vertex3D
		for i := range vertices {
			base := baseTints[i]
			lightScale := lightScales[i]
			tint := color.RGBA{
				R: clampColor(float64(base.R) * lightScale.x),
				G: clampColor(float64(base.G) * lightScale.y),
				B: clampColor(float64(base.B) * lightScale.z),
				A: base.A,
			}
			vertices[i] = lightmappedSurfaceVertex3D(verts[i], uvs[i], lightUVs[i], tint, textureWidth, textureHeight)
		}
		builder.addQuad(vertices, quadIndices012023)
	}

	for y := startY; y <= endY; y++ {
		for x := startX; x <= endX; x++ {
			cell, ok := gnd.Cell(x, y)
			if !ok {
				continue
			}
			if cell.Top >= 0 {
				if surface, ok := gnd.Surface(cell.Top); ok {
					vertexOrder := [4]int{0, 1, 3, 2}
					verts := [4]modelPoint3{
						{x: float64(x) * 2, y: float64(cell.Heights[0]), z: float64(y) * 2},
						{x: float64(x+1) * 2, y: float64(cell.Heights[1]), z: float64(y) * 2},
						{x: float64(x+1) * 2, y: float64(cell.Heights[3]), z: float64(y+1) * 2},
						{x: float64(x) * 2, y: float64(cell.Heights[2]), z: float64(y+1) * 2},
					}
					uvs := surfaceUVs(surface, vertexOrder)
					baseTints := topGNDSurfaceBaseTints(gnd, x, y, surface.Color)
					normals := gndTopNormalsAt(topNormals, gnd, x, y)
					if texture := m.groundTexture(manager, gndTextureName(gnd, surface.TextureID)); texture != nil {
						if _, ok := gnd.Lightmap(surface.LightmapID); ok {
							addLightmapped(texture, verts, uvs, baseTints, groundLightmapTileUVs(x, y), vertexLightScales(lighting, normals))
						} else {
							addTextured(texture, verts, uvs, quadIndices012023, surfaceVertexTints(baseTints, cell.Heights, normals, lighting), groundTextureDrawOptions())
						}
					} else {
						addColored(verts, quadIndices012023, groundSurfaceVertexColors(gndTextureName(gnd, surface.TextureID), surface.Color, cell.Heights, normals, lighting), worldOpaqueTriangleDrawOptions(render.FilterNearest, render.AddressUnsafe))
					}
				}
			}
			if cell.Front >= 0 && y+1 < gnd.Height {
				neighbor, neighborOK := gnd.Cell(x, y+1)
				surface, surfaceOK := gnd.Surface(cell.Front)
				if neighborOK && surfaceOK {
					vertexOrder := [4]int{0, 1, 3, 2}
					verts := [4]modelPoint3{
						{x: float64(x) * 2, y: float64(cell.Heights[2]), z: float64(y+1) * 2},
						{x: float64(x+1) * 2, y: float64(cell.Heights[3]), z: float64(y+1) * 2},
						{x: float64(x+1) * 2, y: float64(neighbor.Heights[1]), z: float64(y+1) * 2},
						{x: float64(x) * 2, y: float64(neighbor.Heights[0]), z: float64(y+1) * 2},
					}
					heights := [4]float32{cell.Heights[2], cell.Heights[3], neighbor.Heights[1], neighbor.Heights[0]}
					normals := uniformGNDNormals(modelPoint3{z: 1})
					addGNDRetainedSurface(m, manager, gnd, surface, vertexOrder, verts, heights, normals, lighting, addTextured, addColored)
				}
			}
			if cell.Right >= 0 && x+1 < gnd.Width {
				neighbor, neighborOK := gnd.Cell(x+1, y)
				surface, surfaceOK := gnd.Surface(cell.Right)
				if neighborOK && surfaceOK {
					vertexOrder := [4]int{0, 1, 3, 2}
					verts := [4]modelPoint3{
						{x: float64(x+1) * 2, y: float64(cell.Heights[3]), z: float64(y+1) * 2},
						{x: float64(x+1) * 2, y: float64(cell.Heights[1]), z: float64(y) * 2},
						{x: float64(x+1) * 2, y: float64(neighbor.Heights[0]), z: float64(y) * 2},
						{x: float64(x+1) * 2, y: float64(neighbor.Heights[2]), z: float64(y+1) * 2},
					}
					heights := [4]float32{cell.Heights[3], cell.Heights[1], neighbor.Heights[0], neighbor.Heights[2]}
					normals := uniformGNDNormals(modelPoint3{x: 1})
					addGNDRetainedSurface(m, manager, gnd, surface, vertexOrder, verts, heights, normals, lighting, addTextured, addColored)
				}
			}
		}
	}
	var meshes []retainedWorldMesh
	for _, builder := range builders {
		builder.flush()
		meshes = append(meshes, builder.meshes...)
	}
	return meshes
}

// gndLightmapTexelsPerTile is the resolution of the world-aligned lightmap
// texture per GND tile. Source lightmaps are 8x8 per tile; 4x4 keeps the
// full-map texture small while still capturing the shadow gradients, and the
// 2x2 block average absorbs the dithered intermediate texels.
const gndLightmapTexelsPerTile = 4

// gndLightmapBlurRadius softens the per-tile shadow masks. The baked GND
// shadows jump straight between lit and shadowed values, so without a blur
// they render as angular blobs whose edges follow the tile grid.
const gndLightmapBlurRadius = 2

type gndGroundLightmap struct {
	image *render.Image
}

func buildGNDGroundLightmap(gnd *res.GND) gndGroundLightmap {
	if gnd == nil || gnd.Width <= 0 || gnd.Height <= 0 || len(gnd.Lightmaps) == 0 {
		return gndGroundLightmap{}
	}
	width := gnd.Width * gndLightmapTexelsPerTile
	height := gnd.Height * gndLightmapTexelsPerTile
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for i := 3; i < len(img.Pix); i += 4 {
		img.Pix[i] = 255
	}
	for y := 0; y < gnd.Height; y++ {
		for x := 0; x < gnd.Width; x++ {
			cell, ok := gnd.Cell(x, y)
			if !ok || cell.Top < 0 {
				continue
			}
			surface, ok := gnd.Surface(cell.Top)
			if !ok {
				continue
			}
			lightmap, ok := gnd.Lightmap(surface.LightmapID)
			if !ok {
				continue
			}
			for ty := 0; ty < gndLightmapTexelsPerTile; ty++ {
				for tx := 0; tx < gndLightmapTexelsPerTile; tx++ {
					r, g, b, a := downsampleGNDLightmapTexel(lightmap, tx, ty)
					pixel := img.PixOffset(x*gndLightmapTexelsPerTile+tx, y*gndLightmapTexelsPerTile+ty)
					img.Pix[pixel+0] = r
					img.Pix[pixel+1] = g
					img.Pix[pixel+2] = b
					img.Pix[pixel+3] = a
				}
			}
		}
	}
	blurGNDLightmap(img, gndLightmapBlurRadius)
	return gndGroundLightmap{image: render.NewImageFromImage(img)}
}

func downsampleGNDLightmapTexel(lightmap res.GNDLightmap, tx, ty int) (uint8, uint8, uint8, uint8) {
	step := 8 / gndLightmapTexelsPerTile
	sumR, sumG, sumB, sumA, count := 0, 0, 0, 0, 0
	for y := ty * step; y < (ty+1)*step; y++ {
		for x := tx * step; x < (tx+1)*step; x++ {
			c := posterizeGNDLightmapColor(lightmap.Color[y][x])
			sumR += int(c.R)
			sumG += int(c.G)
			sumB += int(c.B)
			sumA += int(lightmap.Alpha[y][x])
			count++
		}
	}
	return uint8(sumR / count), uint8(sumG / count), uint8(sumB / count), uint8(sumA / count)
}

// blurGNDLightmap applies two box-blur sweeps (≈ a triangular kernel) over the
// RGBA texels with edge clamping.
func blurGNDLightmap(img *image.RGBA, radius int) {
	if radius <= 0 {
		return
	}
	bounds := img.Bounds()
	width, height := bounds.Dx(), bounds.Dy()
	if width < 2 || height < 2 {
		return
	}
	temp := image.NewRGBA(bounds)
	for pass := 0; pass < 2; pass++ {
		boxBlurRGBAHorizontal(img, temp, radius)
		boxBlurRGBAVertical(temp, img, radius)
	}
}

func boxBlurRGBAHorizontal(src, dst *image.RGBA, radius int) {
	bounds := src.Bounds()
	width, height := bounds.Dx(), bounds.Dy()
	window := radius*2 + 1
	for y := 0; y < height; y++ {
		row := src.Pix[y*src.Stride:]
		out := dst.Pix[y*dst.Stride:]
		sumR, sumG, sumB, sumA := 0, 0, 0, 0
		for i := -radius; i <= radius; i++ {
			p := clampInt(i, 0, width-1) * 4
			sumR += int(row[p])
			sumG += int(row[p+1])
			sumB += int(row[p+2])
			sumA += int(row[p+3])
		}
		for x := 0; x < width; x++ {
			p := x * 4
			out[p] = uint8(sumR / window)
			out[p+1] = uint8(sumG / window)
			out[p+2] = uint8(sumB / window)
			out[p+3] = uint8(sumA / window)
			add := clampInt(x+radius+1, 0, width-1) * 4
			sub := clampInt(x-radius, 0, width-1) * 4
			sumR += int(row[add]) - int(row[sub])
			sumG += int(row[add+1]) - int(row[sub+1])
			sumB += int(row[add+2]) - int(row[sub+2])
			sumA += int(row[add+3]) - int(row[sub+3])
		}
	}
}

func boxBlurRGBAVertical(src, dst *image.RGBA, radius int) {
	bounds := src.Bounds()
	width, height := bounds.Dx(), bounds.Dy()
	window := radius*2 + 1
	for x := 0; x < width; x++ {
		sumR, sumG, sumB, sumA := 0, 0, 0, 0
		for i := -radius; i <= radius; i++ {
			y := clampInt(i, 0, height-1)
			p := y*src.Stride + x*4
			sumR += int(src.Pix[p])
			sumG += int(src.Pix[p+1])
			sumB += int(src.Pix[p+2])
			sumA += int(src.Pix[p+3])
		}
		for y := 0; y < height; y++ {
			p := y*dst.Stride + x*4
			dst.Pix[p] = uint8(sumR / window)
			dst.Pix[p+1] = uint8(sumG / window)
			dst.Pix[p+2] = uint8(sumB / window)
			dst.Pix[p+3] = uint8(sumA / window)
			addY := clampInt(y+radius+1, 0, height-1)
			subY := clampInt(y-radius, 0, height-1)
			add := addY*src.Stride + x*4
			sub := subY*src.Stride + x*4
			sumR += int(src.Pix[add]) - int(src.Pix[sub])
			sumG += int(src.Pix[add+1]) - int(src.Pix[sub+1])
			sumB += int(src.Pix[add+2]) - int(src.Pix[sub+2])
			sumA += int(src.Pix[add+3]) - int(src.Pix[sub+3])
		}
	}
}

// groundLightmapTileUVs maps the four corners of tile (x, y) onto the
// world-aligned lightmap texture. Tile corners land exactly on texel
// boundaries so bilinear sampling stays continuous across neighboring tiles.
func groundLightmapTileUVs(x, y int) [4]texturePoint {
	x0 := float32(x * gndLightmapTexelsPerTile)
	y0 := float32(y * gndLightmapTexelsPerTile)
	x1 := float32((x + 1) * gndLightmapTexelsPerTile)
	y1 := float32((y + 1) * gndLightmapTexelsPerTile)
	return [4]texturePoint{
		{u: x0, v: y0},
		{u: x1, v: y0},
		{u: x1, v: y1},
		{u: x0, v: y1},
	}
}

func lightmappedSurfaceVertex3D(point modelPoint3, uv texturePoint, lightUV texturePoint, tint color.RGBA, textureWidth, textureHeight float32) render.Vertex3D {
	vertex := texturedSurfaceVertex3D(point, uv, tint, textureWidth, textureHeight)
	vertex.LightSrcX = lightUV.u
	vertex.LightSrcY = lightUV.v
	return vertex
}

func addGNDRetainedSurface(m *WorldMode, manager *res.Manager, gnd *res.GND, surface res.GNDSurface, vertexOrder [4]int, verts [4]modelPoint3, heights [4]float32, normals [4]modelPoint3, lighting sceneLighting, addTextured func(*render.Image, [4]modelPoint3, [4]texturePoint, []uint16, [4]color.RGBA, *render.DrawTrianglesOptions), addColored func([4]modelPoint3, []uint16, [4]color.RGBA, *render.DrawTrianglesOptions)) {
	baseTints := uniformGNDSurfaceBaseTints(surface.Color)
	uvs := surfaceUVs(surface, vertexOrder)
	textureName := gndTextureName(gnd, surface.TextureID)
	if texture := m.groundTexture(manager, textureName); texture != nil {
		addTextured(texture, verts, uvs, quadIndices012023, surfaceVertexTints(baseTints, heights, normals, lighting), groundTextureDrawOptions())
		return
	}
	addColored(verts, quadIndices012023, groundSurfaceVertexColors(textureName, surface.Color, heights, normals, lighting), worldOpaqueTriangleDrawOptions(render.FilterNearest, render.AddressUnsafe))
}
