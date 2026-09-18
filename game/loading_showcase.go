package game

import (
	"math/rand"
	"strings"

	"github.com/kivutar/goro/db"
	"github.com/kivutar/goro/render"
	"github.com/kivutar/goro/res"
)

// The loading-cover showcase: one random NPC or monster sprite (idle frame)
// drawn mid-screen while a map loads, so the cover is not the same every
// time. Picked fresh for every map load.

type loadingShowcase struct {
	image *render.Image
	name  string
}

// loadingShowcaseRoots are the folders the random pick comes from. Archive
// entry names are normalized to lowercase UTF-8 with '/' separators, so the
// Korean monster folder matches as a literal.
var loadingShowcaseRoots = []string{
	"data/sprite/npc/",
	"data/sprite/몬스터/",
}

// reverseMonsterResourceName maps a monster resource stem (lowercased, as the
// archive keys spell it) back to its job ID, so a monster pick can use its
// proper display name; NPC stems fall back to the prettified file name.
var reverseMonsterResourceName = func() map[string]int {
	out := make(map[string]int, len(db.MonsterResourceName))
	for job, stem := range db.MonsterResourceName {
		if stem == "" {
			continue
		}
		out[strings.ToLower(stem)] = job
	}
	return out
}()

// loadLoadingShowcase picks one random NPC/monster sprite and composes its
// idle frame (action 0, motion 0) into a still image. A pick that fails to
// decode is retried a few times; nil means "draw nothing".
func loadLoadingShowcase(manager *res.Manager) *loadingShowcase {
	if manager == nil {
		return nil
	}
	names := manager.NamesWithPrefix("data/sprite/")
	haveAct := make(map[string]bool, len(names))
	for _, name := range names {
		if strings.HasSuffix(name, ".act") {
			haveAct[strings.TrimSuffix(name, ".act")] = true
		}
	}
	stems := make([]string, 0, len(names))
	for _, root := range loadingShowcaseRoots {
		for _, name := range names {
			if !strings.HasPrefix(name, root) || !strings.HasSuffix(name, ".spr") {
				continue
			}
			stem := strings.TrimSuffix(name, ".spr")
			if haveAct[stem] {
				stems = append(stems, stem)
			}
		}
	}
	if len(stems) == 0 {
		return nil
	}
	for attempt := 0; attempt < 8; attempt++ {
		stem := stems[rand.Intn(len(stems))]
		view, _ := loadSpriteView(manager,
			[]string{stem + ".act"},
			[]string{stem + ".spr"},
			nil,
			"loading showcase")
		if view == nil {
			continue
		}
		billboard, ok := fixedSpriteBillboard(view)
		if !ok || billboard.image == nil || billboard.image.Bounds().Dy() <= 0 {
			continue
		}
		return &loadingShowcase{image: billboard.image, name: loadingShowcaseName(stem)}
	}
	return nil
}

// loadingShowcaseName resolves a display name from the sprite stem: the
// monster name table when the stem belongs to a known monster, otherwise the
// prettified file name (NPC sprites have no name table entry here).
func loadingShowcaseName(stem string) string {
	base := stem
	if idx := strings.LastIndexAny(base, `/\`); idx >= 0 {
		base = base[idx+1:]
	}
	if job, ok := reverseMonsterResourceName[strings.ToLower(base)]; ok {
		if name := db.MonsterDisplayName[job]; name != "" {
			return name
		}
	}
	return displayNameFromResource(base)
}

// drawLoadingShowcase draws the sprite mid-screen on the loading cover.
// Pixel art stays crisp: the scale is a whole 1x-3x step with nearest
// filtering, falling back to a fractional linear scale only when the frame
// is taller than the screen can fit.
func drawLoadingShowcase(screen *render.Frame, s *loadingShowcase) {
	if screen == nil || s == nil || s.image == nil {
		return
	}
	bounds := screen.Bounds()
	width, height := float64(bounds.Dx()), float64(bounds.Dy())
	b := s.image.Bounds()
	if b.Dx() <= 0 || b.Dy() <= 0 {
		return
	}
	target := height * 0.4
	scale := float64(int(target / float64(b.Dy())))
	if scale < 1 {
		scale = 1
	}
	if scale > 3 {
		scale = 3
	}
	filter := render.FilterNearest
	if drawn := float64(b.Dy()) * scale; drawn > height*0.85 {
		scale = height * 0.85 / float64(b.Dy())
		filter = render.FilterLinear
	}
	var opts render.DrawImageOptions
	opts.GeoM.Scale(scale, scale)
	opts.GeoM.Translate((width-float64(b.Dx())*scale)/2, height*0.44-float64(b.Dy())*scale/2)
	opts.Filter = filter
	screen.DrawImage(s.image, &opts)
}
