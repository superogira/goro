package sprites

import (
	"bytes"
	"image"
	"image/color"
	"testing"
	"testing/synctest"

	"github.com/kivutar/goro/render"
)

func TestSpriteUpscalingKeepsStraightAlpha(t *testing.T) {
	source := render.NewImage(3, 2)
	want := color.RGBA{R: 210, G: 120, B: 60, A: 80}
	for y := range 2 {
		for x := range 3 {
			source.RGBA().SetRGBA(x, y, want)
		}
	}
	before := bytes.Clone(source.RGBA().Pix)
	upscaled := upscaleSpriteImage(source)
	if upscaled.Bounds() != image.Rect(0, 0, 6, 4) {
		t.Fatalf("upscaled bounds = %v", upscaled.Bounds())
	}
	for y := range 4 {
		for x := range 6 {
			if got := upscaled.RGBA().RGBAAt(x, y); got != want {
				t.Fatalf("translucent pixel = %v, want %v", got, want)
			}
		}
	}
	if !bytes.Equal(before, source.RGBA().Pix) {
		t.Fatal("upscaling modified the shared source")
	}
}

func TestSpriteUpscalingQueueDoesNotBlockAndStopsOnExit(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		// Hold the worker until the queue is full, then leave the map.
		s := &Upscaler{
			queue: make(chan *spriteUpscaleJob, 1),
			stop:  make(chan struct{}),
			cache: make(map[*render.Image]*spriteUpscaleJob),
		}
		images := &render.ImageGroup{}
		defer images.Release()
		first, second := render.NewImage(4, 4), render.NewImage(5, 5)
		if s.Texture(first, images.Add) != first || s.Texture(second, images.Add) != second || len(s.cache) != 1 {
			t.Fatal("busy worker must leave the original visible without growing the queue")
		}
		job := <-s.queue
		if s.Texture(second, images.Add) != second || len(s.cache) != 2 {
			t.Fatal("visible sprite was not retried after the queue had room")
		}
		s.Close()
		s.Close()
		exited := false
		go func() { s.run(); exited = true }()
		synctest.Wait()
		if !exited || job.ready.Load() != nil || s.cache[second].ready.Load() != nil {
			t.Fatal("worker processed queued sprites after leaving the map")
		}
	})
}
