package npc

import (
	"strings"
	"time"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/game/effects"
	maps "github.com/kivutar/goro/game/map"
	"github.com/kivutar/goro/game/scene"
	"github.com/kivutar/goro/game/sprites"
	"github.com/kivutar/goro/render"
	worldstate "github.com/kivutar/goro/world"
)

type SpecialVisual uint8

const (
	SpecialVisualNone SpecialVisual = iota
	SpecialVisualTorch
)

func SpecialVisualForActor(actor worldstate.Actor) SpecialVisual {
	if int(actor.Job) == scene.ActorJobClearNPC {
		switch normalizeSpecialNPCActorName(actor.Name) {
		case "BOBBING TORCH", "WET FIREWOOD":
			return SpecialVisualTorch
		}
	}
	return SpecialVisualNone
}

func normalizeSpecialNPCActorName(name string) string {
	name = scene.SanitizeActorName(name)
	if i := strings.IndexByte(name, '#'); i >= 0 {
		name = name[:i]
	}
	return strings.ToUpper(strings.TrimSpace(name))
}

func JobHasNoShadow(job int) bool {
	if sprites.JobHasNoSprite(job) {
		return true
	}
	switch job {
	case scene.ActorJobClearNPC:
		return true
	default:
		return false
	}
}

func DrawSpecialVisual(screen *render.Frame, ctx client.Context, projection scene.Projection, entry scene.ActorDrawEntry, visual SpecialVisual, now time.Time, fx *effects.System) bool {
	switch visual {
	case SpecialVisualTorch:
		drawPersistentWorldEffectAt(screen, ctx, projection, effects.Torch, entry, now, fx)
		return true
	default:
		return false
	}
}

func drawPersistentWorldEffectAt(screen *render.Frame, ctx client.Context, projection scene.Projection, effectID int, entry scene.ActorDrawEntry, now time.Time, fx *effects.System) {
	spec, ok := effects.Lookup(effectID)
	if !ok {
		return
	}
	effect := persistentWorldEffectForActorEntry(effectID, entry, now)
	for index, component := range spec.Components {
		component = maps.MapEffectComponent(component)
		duration := effects.WorldEffectComponentDuration(spec, component)
		if component.Duration > 0 {
			duration = component.Duration
		}
		if duration <= 0 {
			duration = 600 * time.Millisecond
		}
		elapsed := now.Sub(time.Unix(0, 0))
		componentStart := now.Add(-(elapsed % duration))
		effect.Starts = componentStart
		progress := effects.WorldEffectComponentProgress(componentStart, duration, now)
		fx.DrawWorldEffectComponent(screen, ctx, projection, effect, component, index, entry.WorldX, entry.WorldY, entry.WorldZ+0.07, progress, duration, now)
	}
}

func persistentWorldEffectForActorEntry(effectID int, entry scene.ActorDrawEntry, now time.Time) effects.Instance {
	return effects.Instance{
		EffectID: effectID,
		ActorID:  entry.Actor.ID,
		X:        int(entry.Actor.X),
		Y:        int(entry.Actor.Y),
		Starts:   now,
		Expires:  now.Add(24 * time.Hour),
		Duration: 24 * time.Hour,
	}
}
