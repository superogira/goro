package social

import (
	"bytes"
	"compress/zlib"
	"fmt"
	"image"
	"image/draw"
	"io"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/game/scene"
	"github.com/kivutar/goro/glog"
	"github.com/kivutar/goro/network"
	"github.com/kivutar/goro/render"
	"github.com/kivutar/goro/res"
	worldstate "github.com/kivutar/goro/world"
)

type guildEmblem struct {
	version          uint32
	requestedVersion uint32
	image            *render.Image
	flagImage        *render.Image
}

const guildFlagEmblemCanvasScale = 2

func (m *System) RequestActorGuildEmblem(ctx client.Context, guildID, version uint32) {
	m.requestGuildEmblem(ctx, guildID, version, false)
}

func (m *System) requestGuildEmblem(ctx client.Context, guildID, version uint32, force bool) {
	if ctx.Config.Headless || guildID == 0 || version == 0 || ctx.Network == nil {
		return
	}
	if m.guildEmblems == nil {
		m.guildEmblems = make(map[uint32]guildEmblem)
	}
	emblem := m.guildEmblems[guildID]
	if emblem.image != nil && emblem.version >= version {
		return
	}
	if !force && emblem.requestedVersion >= version {
		return
	}
	if err := ctx.Network.SendGuildEmblemRequest(guildID); err != nil {
		glog.Warnf("request guild emblem failed guild=%d version=%d: %v", guildID, version, err)
		return
	}
	emblem.requestedVersion = version
	m.guildEmblems[guildID] = emblem
}

func (m *System) ApplyGuildEmblemImage(ctx client.Context, packet network.GuildEmblemImage) {
	if ctx.Config.Headless {
		return
	}
	decodedImage, err := decodeGuildEmblemImage(packet.Data)
	if err != nil {
		glog.Warnf("decode guild emblem failed guild=%d version=%d: %v", packet.GuildID, packet.EmblemVersion, err)
		return
	}
	if m.guildEmblems == nil {
		m.guildEmblems = make(map[uint32]guildEmblem)
	}
	m.guildEmblems[packet.GuildID] = guildEmblem{
		version:          packet.EmblemVersion,
		requestedVersion: packet.EmblemVersion,
		image:            render.NewImageFromImage(decodedImage),
		flagImage:        buildGuildFlagEmblemTexture(decodedImage),
	}
	m.bindings.Windows.GuildWindow.Refresh(ctx)
	glog.Debugf("guild emblem loaded guild=%d version=%d size=%dx%d", packet.GuildID, packet.EmblemVersion, decodedImage.Bounds().Dx(), decodedImage.Bounds().Dy())
}

func buildGuildFlagEmblemTexture(source image.Image) *render.Image {
	if source == nil {
		return nil
	}
	bounds := source.Bounds()
	width, height := bounds.Dx(), bounds.Dy()
	if width <= 0 || height <= 0 {
		return nil
	}
	canvas := image.NewNRGBA(image.Rect(0, 0, width*guildFlagEmblemCanvasScale, height*guildFlagEmblemCanvasScale))
	offset := image.Pt((canvas.Bounds().Dx()-width)/2, (canvas.Bounds().Dy()-height)/2)
	draw.Draw(canvas, image.Rectangle{Min: offset, Max: offset.Add(bounds.Size())}, source, bounds.Min, draw.Src)
	bleedGuildFlagTransparentEdges(canvas)
	return render.NewImageFromStraightAlpha(canvas)
}

func bleedGuildFlagTransparentEdges(img *image.NRGBA) {
	if img == nil {
		return
	}
	source := append([]byte(nil), img.Pix...)
	for y := img.Bounds().Min.Y; y < img.Bounds().Max.Y; y++ {
		for x := img.Bounds().Min.X; x < img.Bounds().Max.X; x++ {
			offset := img.PixOffset(x, y)
			if source[offset+3] != 0 {
				continue
			}
			var red, green, blue, count int
			for dy := -1; dy <= 1; dy++ {
				for dx := -1; dx <= 1; dx++ {
					if dx == 0 && dy == 0 {
						continue
					}
					nx, ny := x+dx, y+dy
					if !image.Pt(nx, ny).In(img.Bounds()) {
						continue
					}
					neighbor := img.PixOffset(nx, ny)
					if source[neighbor+3] == 0 {
						continue
					}
					red += int(source[neighbor])
					green += int(source[neighbor+1])
					blue += int(source[neighbor+2])
					count++
				}
			}
			if count > 0 {
				img.Pix[offset] = byte(red / count)
				img.Pix[offset+1] = byte(green / count)
				img.Pix[offset+2] = byte(blue / count)
			}
		}
	}
}

func decodeGuildEmblemImage(data []byte) (image.Image, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("empty emblem")
	}
	zr, err := zlib.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("inflate emblem: %w", err)
	}
	defer zr.Close()
	decoded, err := io.ReadAll(io.LimitReader(zr, 4096))
	if err != nil {
		return nil, fmt.Errorf("read inflated emblem: %w", err)
	}
	img, err := res.DecodeImageData(decoded)
	if err != nil {
		return nil, err
	}
	return img, nil
}

func (m *System) ApplyGuildEmblemChange(ctx client.Context, change network.GuildEmblemChange) {
	if change.ActorID == 0 || ctx.World == nil {
		return
	}
	if scene.IsLocalActor(ctx, change.ActorID) {
		ctx.World.Player.GuildID = change.GuildID
		ctx.World.Player.EmblemVersion = change.EmblemVersion
		if ctx.Session != nil {
			ctx.Session.GuildID = change.GuildID
			ctx.Session.EmblemVersion = change.EmblemVersion
		}
	} else if actor, ok := ctx.World.Actors[change.ActorID]; ok {
		actor.GuildID = change.GuildID
		actor.EmblemVersion = change.EmblemVersion
		ctx.World.Actors[change.ActorID] = actor
	}
	m.RequestActorGuildEmblem(ctx, change.GuildID, change.EmblemVersion)
}

func (m *System) ActorGuildEmblem(ctx client.Context, actor worldstate.Actor, isPlayer bool) *render.Image {
	guildID := actor.GuildID
	version := actor.EmblemVersion
	if isPlayer && ctx.Session != nil {
		if guildID == 0 {
			guildID = ctx.Session.GuildID
			if guildID == 0 {
				guildID = ctx.Session.Guild.ID
			}
		}
		if version == 0 {
			version = ctx.Session.EmblemVersion
		}
	}
	if guildID == 0 || version == 0 {
		return nil
	}
	if m.guildEmblems == nil {
		m.RequestActorGuildEmblem(ctx, guildID, version)
		return nil
	}
	emblem := m.guildEmblems[guildID]
	if emblem.image == nil || emblem.version < version {
		m.RequestActorGuildEmblem(ctx, guildID, version)
		return nil
	}
	return emblem.image
}

func (m *System) SessionGuildEmblemImage(ctx client.Context) image.Image {
	if ctx.Session == nil || m.guildEmblems == nil {
		return nil
	}
	guildID := ctx.Session.GuildID
	version := ctx.Session.EmblemVersion
	if guildID == 0 {
		guildID = ctx.Session.Guild.ID
	}
	if version == 0 {
		version = ctx.Session.Guild.EmblemVersion
	}
	if guildID == 0 || version == 0 {
		return nil
	}
	emblem := m.guildEmblems[guildID]
	if emblem.image == nil || emblem.version < version {
		return nil
	}
	return emblem.image.RGBA()
}

func (m *System) GuildFlagEmblem(ctx client.Context, actor worldstate.Actor) *render.Image {
	if actor.GuildID == 0 || actor.EmblemVersion == 0 {
		return nil
	}
	if m.guildEmblems == nil {
		m.RequestActorGuildEmblem(ctx, actor.GuildID, actor.EmblemVersion)
		return nil
	}
	emblem := m.guildEmblems[actor.GuildID]
	if emblem.flagImage == nil || emblem.version < actor.EmblemVersion {
		m.RequestActorGuildEmblem(ctx, actor.GuildID, actor.EmblemVersion)
		return nil
	}
	return emblem.flagImage
}
