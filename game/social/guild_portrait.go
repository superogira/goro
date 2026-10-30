package social

import (
	"image"
	"math"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/game/scene"
	"github.com/kivutar/goro/render"
	"github.com/kivutar/goro/res"
	"github.com/kivutar/goro/session"
)

type guildPortraitKey struct {
	job, head, palette, sex uint16
}

type guildPortraitCache struct {
	resources *res.Manager
	images    map[guildPortraitKey]image.Image
}

func (m *System) GuildMemberPortrait(ctx client.Context, member session.GuildMember) image.Image {
	if ctx.Resources == nil || !member.Online() {
		return nil
	}
	if m.guildPortraits.resources != ctx.Resources {
		m.guildPortraits = guildPortraitCache{resources: ctx.Resources, images: make(map[guildPortraitKey]image.Image)}
	}
	key := guildPortraitKey{member.Job, member.HeadType, member.HeadPalette, member.Sex}
	if img, loaded := m.guildPortraits.images[key]; loaded {
		return img
	}
	m.guildPortraits.images[key] = nil
	job, head, sex, palette := int(member.Job), int(member.HeadType), byte(member.Sex), int(member.HeadPalette)
	view, _ := scene.LoadSpriteView(ctx.Resources,
		res.PlayerHeadResourceCandidates(job, head, sex, "act"),
		res.PlayerHeadResourceCandidates(job, head, sex, "spr"),
		res.PlayerHeadPaletteResourceCandidates(job, head, sex, palette, "pal"), "guild portrait")
	if view == nil || len(view.Act.Actions) == 0 || len(view.Act.Actions[0].Animations) == 0 {
		return nil
	}
	portrait, ok := scene.ComposeSingleSpriteBillboard(view, view.Act.Actions[0].Animations[0])
	if !ok {
		return nil
	}
	const size = 24
	bounds := portrait.Image.Bounds()
	scale := math.Min(1, math.Min(float64(size)/float64(bounds.Dx()), float64(size)/float64(bounds.Dy())))
	img := render.NewImage(size, size)
	var opts render.DrawImageOptions
	opts.GeoM.Scale(scale, scale)
	opts.GeoM.Translate((size-float64(bounds.Dx())*scale)/2, (size-float64(bounds.Dy())*scale)/2)
	opts.Filter = render.FilterLinear
	img.DrawImage(portrait.Image, &opts)
	m.guildPortraits.images[key] = img.RGBA()
	return img.RGBA()
}
