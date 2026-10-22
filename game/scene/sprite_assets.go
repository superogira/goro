package scene

import (
	"fmt"
	"time"

	"github.com/kivutar/goro/render"
	"github.com/kivutar/goro/res"
)

func LoadSpriteView(manager *res.Manager, actCandidates []string, sprCandidates []string, palCandidates []string, label string) (*SpriteView, string) {
	act, actSource, err := manager.LoadACT(actCandidates)
	if err != nil {
		return nil, fmt.Sprintf("%s act: %v", label, err)
	}
	spr, sprSource, err := manager.LoadSPR(sprCandidates)
	if err != nil {
		return nil, fmt.Sprintf("%s spr: %v", label, err)
	}
	palette, paletteSource, paletteStatus := loadSpritePalette(manager, palCandidates)
	return &SpriteView{
		Spr:           spr,
		Act:           act,
		ActSource:     actSource,
		Source:        sprSource,
		Palette:       palette,
		PaletteSource: paletteSource,
		Images:        make(map[SpriteFrameKey]*render.Image),
		Billboards:    make(map[SpriteBillboardKey]*Billboard),
		Started:       time.Now(),
	}, fmt.Sprintf("%s: %s actions=%d frames=%d%s", label, sprSource, len(act.Actions), len(spr.Frames), paletteStatus)
}

func loadSpritePalette(manager *res.Manager, candidates []string) (*res.Palette, string, string) {
	if len(candidates) == 0 {
		return nil, "", ""
	}
	palette, source, err := manager.LoadPAL(candidates)
	if err != nil {
		if source == "" {
			return nil, "", " palette=default"
		}
		return nil, "", fmt.Sprintf(" palette=%s parse-error=%v", source, err)
	}
	return palette, source, fmt.Sprintf(" palette=%s", source)
}
