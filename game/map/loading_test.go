package maps

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"
	"testing/synctest"
	"time"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/render"
	"github.com/kivutar/goro/res"
	worldstate "github.com/kivutar/goro/world"
)

func TestLoadingWaitsForFramesAndYieldsBetweenAssets(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		scene, ctx := mapTextureFixture(t)
		calls := 0
		load := func() { calls++; time.Sleep(9 * time.Millisecond) }
		l := &Loading{stage: loadingTextures, tasks: []func(){load, load}}
		l.Advance(ctx, scene)
		if calls != 0 {
			t.Fatal("loaded before the cover was presented")
		}
		l.Presented = true
		l.Advance(ctx, scene)
		if calls != 1 || l.Percent(0, 0) <= 30 || l.Prepared() {
			t.Fatal("did not yield after the first expensive asset")
		}
		l.Advance(ctx, scene)
		if calls != 1 {
			t.Fatal("skipped frame allowed another batch")
		}
		l.Presented = true
		l.Advance(ctx, scene)
		if calls != 2 || l.Percent(0, 0) != 85 {
			t.Fatal("second batch did not finish texture preparation")
		}
	})
}

func TestLoadingArtworkFallbackAndRelease(t *testing.T) {
	for _, name := range []string{"loading07.jpg", "loading00.jpg", "missing"} {
		t.Run(name, func(t *testing.T) {
			root := t.TempDir()
			if name != "missing" {
				path := filepath.Join(root, "data/texture/유저인터페이스", name)
				if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
					t.Fatal(err)
				}
				img := image.NewRGBA(image.Rect(0, 0, 2, 1))
				img.SetRGBA(0, 0, color.RGBA{R: 255, B: 255, A: 255})
				var data bytes.Buffer
				if err := png.Encode(&data, img); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(path, data.Bytes(), 0600); err != nil {
					t.Fatal(err)
				}
			}
			ctx := client.Context{Resources: &res.Manager{Root: root}, World: worldstate.New()}
			l := NewLoading(ctx)
			defer l.Release()
			if name == "missing" {
				if l.background != nil {
					t.Fatal("missing artwork should use a plain cover")
				}
			} else if l.background == nil || l.background.RGBA().RGBAAt(0, 0).A != 255 {
				t.Fatal("did not find the available artwork or magenta was made transparent")
			}
			background := l.background
			l.Draw(render.NewFrame(800, 600), 0, 255)
			l.Release()
			if len(l.tasks) != 0 || (background != nil && background.RGBA() != nil) {
				t.Fatal("leaving loading retained tasks or artwork")
			}
		})
	}
}

func TestLoadingProgressIncludesSubmittedUploadsAndWarmup(t *testing.T) {
	l := &Loading{stage: loadingPrepared, uploads: 4}
	scene := &System{}
	defer scene.Images.Release()
	for range 4 {
		scene.Images.Uploads = append(scene.Images.Uploads, scene.Images.Own(render.NewImage(1, 1)))
	}
	f := render.NewFrame(800, 600)
	if l.Percent(len(scene.Images.Uploads), 0) != 90 {
		t.Fatal("CPU preparation consumed GPU progress")
	}
	scene.Images.PrepareUploads(f)
	if l.Percent(len(scene.Images.Uploads), 0) != 90 {
		t.Fatal("unsubmitted uploads advanced progress")
	}
	scene.Images.FrameSubmitted()
	if l.Percent(len(scene.Images.Uploads), 0) != 98 {
		t.Fatal("uploads did not advance progress")
	}
	if l.Percent(0, 1) != 99 || l.Percent(0, FadePrewarmFrames) != 100 {
		t.Fatal("progress did not wait for scene warmup")
	}
}

func TestLoadingCompletesWithOptionalAssetsMissing(t *testing.T) {
	ctx := client.Context{Resources: &res.Manager{Root: t.TempDir()}, World: worldstate.New()}
	ctx.World.MapName = "missing.gat"
	l := NewLoading(ctx)
	defer l.Release()
	scene := &System{}
	last := 0
	for range 10 {
		l.Presented = true
		done := l.Advance(ctx, scene)
		progress := l.Percent(0, 0)
		if progress < last || progress >= 100 {
			t.Fatalf("premature or regressed progress: %d -> %d", last, progress)
		}
		last = progress
		if done {
			if !l.Prepared() || l.RSWSource != "missing.gat" {
				t.Fatal("missing RSW lost fallback BGM identity")
			}
			if l.Advance(ctx, scene) {
				t.Fatal("reported completion twice")
			}
			return
		}
	}
	t.Fatal("missing optional resources prevented loading")
}
