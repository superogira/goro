package maps

import "github.com/kivutar/goro/render"

const (
	textureUploadBytes = 8 << 20
	TextureUploadCount = 32
)

type Textures struct {
	Images      *render.ImageGroup
	Uploads     []*render.Image
	UploadBatch int
}

func (t *Textures) Own(img *render.Image) *render.Image {
	if img == nil {
		return nil
	}
	if t.Images == nil {
		t.Images = &render.ImageGroup{}
	}
	return t.Images.Add(img)
}

func (t *Textures) Release() {
	t.Images.Release()
	t.Images = nil
	t.Uploads = nil
	t.UploadBatch = 0
}

// Keep the map covered until all requested uploads have been submitted, then
// render the usual two warmup frames. A skipped GPU frame retries the same batch.
func (t *Textures) PrepareUploads(screen *render.Frame) bool {
	t.UploadBatch = 0
	bytes := 0
	for _, img := range t.Uploads {
		size := img.ByteSize()
		if t.UploadBatch > 0 && (bytes+size > textureUploadBytes || t.UploadBatch >= TextureUploadCount) {
			break
		}
		screen.PrepareImage(img)
		t.UploadBatch++
		bytes += size
	}
	return t.UploadBatch > 0
}

// FrameSubmitted reports whether this submission completed an upload batch.
func (t *Textures) FrameSubmitted() bool {
	if t.UploadBatch > 0 {
		clear(t.Uploads[:t.UploadBatch])
		t.Uploads = t.Uploads[t.UploadBatch:]
		t.UploadBatch = 0
		return true
	}
	return false
}
