package maps

import (
	"image"
	"testing"

	"github.com/kivutar/goro/render"
)

func TestMapTextureUploadsRespectBytesAndAllowOneOversizedImage(t *testing.T) {
	m := &Textures{Uploads: []*render.Image{render.NewImage(1024, 1024), render.NewImage(1024, 1024), render.NewImage(2048, 2048)}}
	f := render.NewFrame(100, 100)
	m.PrepareUploads(f)
	if m.UploadBatch != 2 {
		t.Fatal("upload byte budget exceeded")
	}
	m.FrameSubmitted()
	f.BeginFrame()
	m.PrepareUploads(f)
	if m.UploadBatch != 1 {
		t.Fatal("single large texture cannot make progress")
	}
}

func TestMapTextureUploadBudgetIncludesMipmaps(t *testing.T) {
	texture := render.NewImageWithMipmaps(image.NewRGBA(image.Rect(0, 0, 1024, 1024)))
	m := &Textures{Uploads: []*render.Image{texture, texture}}
	m.PrepareUploads(render.NewFrame(100, 100))
	if m.UploadBatch != 1 {
		t.Fatal("two 4 MiB base levels fit the budget, but their mip chains do not")
	}
}
