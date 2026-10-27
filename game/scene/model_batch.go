package scene

import (
	"github.com/kivutar/goro/render"
)

type ModelDrawBatch struct {
	Screen  *render.Frame
	Texture *render.Image
	Options render.DrawTrianglesOptions
	Verts   []render.Vertex3D
	Indices []uint16
}

func (b *ModelDrawBatch) AddTriangle(a, c, d render.Vertex3D) {
	if b.Texture == nil {
		return
	}
	if len(b.Verts)+3 > MaxWorldMeshVertices {
		b.Flush()
	}
	base := uint16(len(b.Verts))
	b.Verts = append(b.Verts, a, c, d)
	b.Indices = append(b.Indices, base, base+1, base+2)
}

func (b *ModelDrawBatch) Flush() {
	if len(b.Verts) == 0 || len(b.Indices) == 0 || b.Texture == nil {
		b.Verts = b.Verts[:0]
		b.Indices = b.Indices[:0]
		return
	}
	b.Screen.DrawTriangles3DOwned(b.Verts, b.Indices, b.Texture, &b.Options)
	b.Verts = nil
	b.Indices = nil
}

func (b *ModelDrawBatch) Clear() {
	b.Screen = nil
	b.Texture = nil
	b.Options = render.DrawTrianglesOptions{}
	b.Verts = nil
	b.Indices = nil
}

const MaxWorldMeshVertices = 65535
