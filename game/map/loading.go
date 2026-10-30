package maps

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"math/rand/v2"
	"time"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/render"
)

const (
	loadingMap = iota
	loadingModels
	loadingTextures
	loadingSounds
	loadingPrepared
)

// Loading prepares one map on the game thread, yielding between assets so the
// loading screen can redraw. GPU uploads and scene warmup follow preparation.
type Loading struct {
	RSWSource  string
	Presented  bool
	stage      int
	tasks      []func()
	completed  int
	uploads    int
	images     render.ImageGroup
	background *render.Image
	unblockUI  func()
}

func NewLoading(ctx client.Context) *Loading {
	l := &Loading{RSWSource: ctx.World.MapName}
	l.blockUI(ctx)
	if ctx.Resources != nil {
		for _, index := range append(rand.Perm(10), -1) {
			name := fmt.Sprintf("loading%02d.jpg", index+1)
			candidates := []string{
				"data/texture/유저인터페이스/" + name,
				"texture/유저인터페이스/" + name,
				"data/texture/interface/" + name,
				name,
			}
			for _, candidate := range candidates {
				data, err := ctx.Resources.ReadFileExact(candidate)
				if err != nil {
					continue
				}
				// Loading artwork is opaque: sprite magenta-keying does not apply.
				if img, _, err := image.Decode(bytes.NewReader(data)); err == nil {
					l.background = l.images.Add(render.NewImageFromImage(img))
					break
				}
			}
			if l.background != nil {
				break
			}
		}
	}
	l.tasks = []func(){
		func() { ctx.World.GND, _, _ = LoadGND(ctx.Resources, ctx.World.MapName) },
		func() {
			rsw, source, err := LoadRSW(ctx.Resources, ctx.World.MapName)
			if err == nil {
				ctx.World.RSW = rsw
				l.RSWSource = source
			}
		},
	}
	return l
}

func (l *Loading) Prepared() bool { return l.stage == loadingPrepared }

// Advance reports completion once. Each batch waits for a submitted frame;
// skipped GPU frames cannot turn loading back into a blocking loop.
func (l *Loading) Advance(ctx client.Context, scene *System) bool {
	l.blockUI(ctx)
	if !l.Presented || l.Prepared() {
		return false
	}
	l.Presented = false
	deadline := time.Now().Add(8 * time.Millisecond)
	for l.completed < len(l.tasks) {
		l.tasks[l.completed]()
		l.tasks[l.completed] = nil
		l.completed++
		if time.Now().After(deadline) {
			break
		}
	}
	if l.completed < len(l.tasks) {
		return false
	}
	l.stage++
	l.completed = 0
	switch l.stage {
	case loadingModels:
		l.tasks = ModelLoadTasks(ctx)
	case loadingTextures:
		l.tasks = scene.TextureLoadTasks(ctx)
	case loadingSounds:
		l.tasks = SoundLoadTasks(ctx)
	case loadingPrepared:
		l.tasks = nil
		l.uploads = len(scene.Images.Uploads)
		return true
	}
	return false
}

func (l *Loading) Percent(pendingUploads, coveredFrames int) int {
	if l.Prepared() {
		progress := 98
		if l.uploads > 0 {
			progress -= 8 * pendingUploads / l.uploads
		}
		return min(100, progress+2*coveredFrames/FadePrewarmFrames)
	}
	// Stage weights reserve the final ten percent for uploads and warmup.
	limits := [...]int{0, 10, 30, 85, 90}
	progress := limits[l.stage]
	if len(l.tasks) > 0 {
		progress += (limits[l.stage+1] - progress) * l.completed / len(l.tasks)
	}
	return progress
}

func (l *Loading) Draw(screen *render.Frame, percent int, alpha uint8) {
	width, height := float64(screen.Bounds().Dx()), float64(screen.Bounds().Dy())
	render.DrawRect(screen, 0, 0, width, height, color.RGBA{A: alpha})
	if l.background != nil {
		var opts render.DrawImageOptions
		opts.GeoM.Scale(width/float64(l.background.Bounds().Dx()), height/float64(l.background.Bounds().Dy()))
		opts.ColorScale.ScaleAlpha(float32(alpha) / 255)
		opts.Filter = render.FilterLinear
		screen.DrawImage(l.background, &opts)
	}
	w, h := render.SnapScreenPoint(screen, min(240.0, width-16), 15)
	x, y := render.SnapScreenPoint(screen, (width-w)/2, height*0.75)
	// Snap each inset once so opposite edges have the same physical thickness.
	borderX, borderY := render.SnapScreenPoint(screen, 1, 1)
	padX, padY := render.SnapScreenPoint(screen, 2, 2)
	render.DrawRect(screen, x, y, w, h, color.RGBA{G: 255, B: 255, A: alpha})
	render.DrawRect(screen, x+borderX, y+borderY, w-2*borderX, h-2*borderY, color.RGBA{R: 140, G: 140, B: 140, A: alpha})
	render.DrawRect(screen, x+padX, y+padY, float64(percent)*(w-2*padX)/100, h-2*padY, color.RGBA{R: 66, G: 99, B: 165, A: alpha})
	text := fmt.Sprintf("%d%%", percent)
	// Overlay labels include padding around the glyphs.
	render.DrawCenteredUITextAtSize(screen, text, width/2, y-4, color.RGBA{R: 255, G: 255, A: alpha}, 11, false)
}

func (l *Loading) Release() {
	if l.unblockUI != nil {
		l.unblockUI()
		l.unblockUI = nil
	}
	l.images.Release()
	l.background = nil
	l.tasks = nil
}

func (l *Loading) blockUI(ctx client.Context) {
	// Startup can create the map before the renderer attaches the UI app.
	if app, ok := ctx.UIApp.(interface{ BlockInput() func() }); ok && l.unblockUI == nil {
		l.unblockUI = app.BlockInput()
	}
}
