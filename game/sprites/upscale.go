package sprites

import (
	"image"
	"sync/atomic"

	"github.com/kivutar/goro/render"
)

// Sprite frames arrive during play. One worker and a bounded queue keep xBR
// off the render thread; a busy queue leaves the original visible until a later
// draw can enqueue it. The cache shares results across actors and anchor offsets.
type Upscaler struct {
	queue chan *spriteUpscaleJob
	stop  chan struct{}
	cache map[*render.Image]*spriteUpscaleJob
}

type spriteUpscaleJob struct {
	source *render.Image
	ready  atomic.Pointer[render.Image]
	image  *render.Image // Adopted by the map on the game/render thread.
}

func NewUpscaler() *Upscaler {
	s := &Upscaler{
		queue: make(chan *spriteUpscaleJob, 8),
		stop:  make(chan struct{}),
		cache: make(map[*render.Image]*spriteUpscaleJob),
	}
	go s.run()
	return s
}

func (s *Upscaler) run() {
	for {
		// Check cancellation before taking another queued frame on map changes.
		select {
		case <-s.stop:
			return
		default:
		}
		select {
		case <-s.stop:
			return
		case job := <-s.queue:
			job.ready.Store(upscaleSpriteImage(job.source))
		}
	}
}

func upscaleSpriteImage(src *render.Image) *render.Image {
	// Composed sprites store straight alpha in an RGBA buffer. Expose its
	// actual encoding so xBR does not unpremultiply and brighten translucent pixels.
	pix := src.RGBA()
	straight := &image.NRGBA{Pix: pix.Pix, Stride: pix.Stride, Rect: pix.Rect}
	return render.NewImageFromStraightAlpha(render.UpscaleTexture2x(straight))
}

// Texture returns the original until the worker finishes. adopt registers a
// completed image with its owner on the game/render thread.
func (s *Upscaler) Texture(src *render.Image, adopt func(*render.Image) *render.Image) *render.Image {
	if job := s.cache[src]; job != nil {
		if job.image == nil {
			if ready := job.ready.Load(); ready != nil {
				job.image = adopt(ready)
			}
		}
		if job.image != nil {
			return job.image
		}
	} else {
		job := &spriteUpscaleJob{source: src}
		select {
		case s.queue <- job:
			s.cache[src] = job
		default:
		}
	}
	return src
}

// Close stops taking work. The owner releases images already adopted by Texture.
func (s *Upscaler) Close() {
	if s == nil {
		return
	}
	select {
	case <-s.stop:
	default:
		close(s.stop)
	}
}
