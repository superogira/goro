package battle

import (
	"image/color"
	"math"
	"time"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/game/actors"
	"github.com/kivutar/goro/game/companion"
	"github.com/kivutar/goro/game/npc"
	"github.com/kivutar/goro/game/scene"
	"github.com/kivutar/goro/glog"
	"github.com/kivutar/goro/network"
	"github.com/kivutar/goro/render"
	gameui "github.com/kivutar/goro/ui"
	"github.com/kivutar/goro/world"
)

type Life struct {
	HP        int
	MaxHP     int
	SP        int
	MaxSP     int
	HasSP     bool
	Player    bool
	Friendly  bool
	Hunger    int
	MaxHunger int
	HasHunger bool
}

type CastBar struct {
	Started  time.Time
	Duration time.Duration
	Color    color.RGBA
}

func (m *System) ApplyActorHPUpdate(update network.ActorHPUpdate) {
	if update.ID == 0 || update.MaxHP <= 0 {
		return
	}
	hp := update.HP
	if hp < 0 {
		hp = 0
	}
	if hp > update.MaxHP {
		hp = update.MaxHP
	}
	if m.ActorLife == nil {
		m.ActorLife = make(map[uint32]Life)
	}
	m.ActorLife[update.ID] = Life{
		HP:    hp,
		MaxHP: update.MaxHP,
	}
	glog.Debugf("actor hp id=%d hp=%d max_hp=%d tiny=%t", update.ID, hp, update.MaxHP, update.Tiny)
}

func ActorCastBarProgress(bar CastBar, now time.Time) (float64, bool) {
	if bar.Duration <= 0 || bar.Started.IsZero() {
		return 0, false
	}
	progress := float64(now.Sub(bar.Started)) / float64(bar.Duration)
	if progress < 0 {
		progress = 0
	}
	if progress >= 1 {
		return 1, false
	}
	return progress, true
}

func actorLifeBarHeight(life Life) float64 {
	height := actorOverlayBarHeight
	if life.HasSP {
		height += actorOverlayBarRowSpacing
	}
	if life.HasHunger {
		height += actorOverlayBarRowSpacing
	}
	return height
}

func ActorNameBelowLifeBarY(baseY, scale float64, life Life) float64 {
	return actors.ActorLifeBarY(baseY, scale) + actorLifeBarHeight(life) + 3
}

const (
	actorOverlayBarWidth      = 60.0
	actorOverlayBarRowHeight  = 3.0
	actorOverlayBarRowSpacing = 4.0
	actorOverlayBarHeight     = actorOverlayBarRowHeight + 2
)

func actorOverlayBarX(centerX float64) float64 {
	return math.Round(centerX - actorOverlayBarWidth/2)
}

func actorOverlayBarY(y float64) float64 {
	return math.Round(y)
}

func actorOverlayBarFillWidth(ratio float64) float64 {
	if ratio <= 0 || math.IsNaN(ratio) || math.IsInf(ratio, 0) {
		return 0
	}
	if ratio > 1 {
		ratio = 1
	}
	return math.Round((actorOverlayBarWidth - 2) * ratio)
}

func (m *System) DrawActorLifeBar(screen *render.Frame, ctx client.Context, entry scene.ActorDrawEntry) {
	life, ok := m.ActorLifeForDisplay(ctx, entry.Actor)
	if !ok {
		return
	}
	ratio := actorLifeRatio(life.HP, life.MaxHP)
	height := actorLifeBarHeight(life)
	x := actorOverlayBarX(entry.ScreenX)
	y := actorOverlayBarY(actors.ActorLifeBarY(entry.ScreenY, entry.Scale))
	fillWidth := actorOverlayBarFillWidth(ratio)
	fill := color.RGBA{R: 255, G: 0, B: 231, A: 255}
	if life.Player || life.Friendly {
		fill = gameui.PlayerHPBarColor
		if ratio < 0.25 {
			fill = color.RGBA{R: 255, G: 0, B: 0, A: 255}
		}
	} else if ratio < 0.25 {
		fill = color.RGBA{R: 255, G: 255, B: 0, A: 255}
	}
	render.DrawRect(screen, x, y, actorOverlayBarWidth, height, color.RGBA{R: 16, G: 24, B: 156, A: 255})
	render.DrawRect(screen, x+1, y+1, actorOverlayBarWidth-2, height-2, color.RGBA{R: 66, G: 66, B: 66, A: 255})
	if fillWidth > 0 {
		render.DrawRect(screen, x+1, y+1, fillWidth, actorOverlayBarRowHeight, fill)
	}
	rowY := y + 1
	if life.HasSP {
		rowY += actorOverlayBarRowSpacing
		spRatio := actorLifeRatio(life.SP, life.MaxSP)
		render.DrawRect(screen, x, rowY-1, actorOverlayBarWidth, 1, color.RGBA{R: 16, G: 24, B: 156, A: 255})
		if spWidth := actorOverlayBarFillWidth(spRatio); spWidth > 0 {
			render.DrawRect(screen, x+1, rowY, spWidth, actorOverlayBarRowHeight, gameui.PlayerSPBarColor)
		}
	}
	if life.HasHunger {
		rowY += actorOverlayBarRowSpacing
		hungerRatio := actorLifeRatio(life.Hunger, life.MaxHunger)
		hungerFill := gameui.HungerBarFillColor(life.Hunger, life.MaxHunger)
		render.DrawRect(screen, x+1, rowY-1, actorOverlayBarWidth-2, 1, color.RGBA{R: 66, G: 66, B: 66, A: 255})
		if hungerWidth := actorOverlayBarFillWidth(hungerRatio); hungerWidth > 0 {
			render.DrawRect(screen, x+1, rowY, hungerWidth, actorOverlayBarRowHeight, hungerFill)
		}
	}
}

func actorLifeRatio(value, maxValue int) float64 {
	if maxValue <= 0 {
		return 0
	}
	ratio := float64(value) / float64(maxValue)
	if ratio < 0 {
		return 0
	}
	if ratio > 1 {
		return 1
	}
	return ratio
}

func (m *System) DrawActorCastBar(screen *render.Frame, entry scene.ActorDrawEntry, now time.Time) {
	if entry.Actor.ID == 0 {
		return
	}
	bar, ok := m.serverProgressBar(entry.IsPlayer, now)
	if !ok && m.ActorCastBars != nil {
		bar, ok = m.ActorCastBars[entry.Actor.ID]
	}
	if !ok {
		return
	}
	ratio, active := ActorCastBarProgress(bar, now)
	if !active {
		delete(m.ActorCastBars, entry.Actor.ID)
		return
	}
	x := actorOverlayBarX(entry.ScreenX)
	y := actorOverlayBarY(actors.ActorCastBarY(entry.ScreenY, entry.Scale))
	fillWidth := actorOverlayBarFillWidth(ratio)
	render.DrawRect(screen, x, y, actorOverlayBarWidth, actorOverlayBarHeight, color.RGBA{R: 16, G: 24, B: 156, A: 255})
	render.DrawRect(screen, x+1, y+1, actorOverlayBarWidth-2, actorOverlayBarHeight-2, color.RGBA{R: 66, G: 66, B: 66, A: 255})
	if fillWidth > 0 {
		fill := bar.Color
		if fill.A == 0 {
			fill = color.RGBA{R: 0, G: 255, B: 0, A: 255}
		}
		render.DrawRect(screen, x+1, y+1, fillWidth, actorOverlayBarRowHeight, fill)
	}
}

func (m *System) ActorLifeForDisplay(ctx client.Context, actor world.Actor) (Life, bool) {
	if actor.ID == 0 {
		return Life{}, false
	}
	if npc.SpecialVisualForActor(actor) != npc.SpecialVisualNone {
		return Life{}, false
	}
	if scene.IsLocalActor(ctx, actor.ID) {
		return localPlayerLifeForDisplay(ctx)
	}
	if life, ok := partyMemberLifeForDisplay(ctx, actor); ok {
		return life, true
	}
	if life, ok := m.homunculusLifeForDisplay(ctx, actor); ok {
		return life, true
	}
	if life, ok := m.mercenaryLifeForDisplay(ctx, actor); ok {
		return life, true
	}
	// Monster HP bars are a 2012+ client feature. The 2008 client exposes
	// monster HP through WZ_ESTIMATION/Sense instead, so keep the combat HP
	// cache hidden from the normal actor overlay.
	return Life{}, false
}

func (m *System) homunculusLifeForDisplay(ctx client.Context, actor world.Actor) (Life, bool) {
	if ctx.Session == nil || actor.ID == 0 {
		return Life{}, false
	}
	hom := ctx.Session.Homunculus
	if !hom.Active {
		return Life{}, false
	}
	if hom.ID != 0 {
		if actor.ID != hom.ID {
			return Life{}, false
		}
	} else if !actor.HasObjectType || actor.ObjectType != scene.ActorObjectTypeHomunculus {
		return Life{}, false
	}

	life := Life{
		HP:       hom.HP,
		MaxHP:    hom.MaxHP,
		SP:       hom.SP,
		MaxSP:    hom.MaxSP,
		HasSP:    hom.MaxSP > 0,
		Friendly: true,
	}
	if m.ActorLife != nil {
		if cached, ok := m.ActorLife[actor.ID]; ok {
			if cached.MaxHP > 0 {
				life.HP = cached.HP
				life.MaxHP = cached.MaxHP
			}
			if cached.HasSP && cached.MaxSP > 0 {
				life.SP = cached.SP
				life.MaxSP = cached.MaxSP
				life.HasSP = true
			}
			if cached.HasHunger && cached.MaxHunger > 0 {
				life.Hunger = cached.Hunger
				life.MaxHunger = cached.MaxHunger
				life.HasHunger = true
			}
		}
	}
	if !life.HasHunger && hom.Hunger > 0 {
		life.Hunger = hom.Hunger
		life.MaxHunger = companion.HungerMax
		life.HasHunger = true
	}
	if life.MaxHP <= 0 || life.HP < 0 {
		return Life{}, false
	}
	life.HP = scene.ClampInt(life.HP, 0, life.MaxHP)
	if life.HasSP {
		life.SP = scene.ClampInt(life.SP, 0, life.MaxSP)
	}
	if life.HasHunger {
		life.Hunger = scene.ClampInt(life.Hunger, 0, life.MaxHunger)
	}
	return life, true
}

func (m *System) mercenaryLifeForDisplay(ctx client.Context, actor world.Actor) (Life, bool) {
	if ctx.Session == nil || actor.ID == 0 {
		return Life{}, false
	}
	merc := ctx.Session.Mercenary
	if !merc.Active {
		return Life{}, false
	}
	if merc.ID != 0 {
		if actor.ID != merc.ID {
			return Life{}, false
		}
	} else if !actor.HasObjectType || actor.ObjectType != scene.ActorObjectTypeMercenary {
		return Life{}, false
	}

	life := Life{
		HP:       merc.HP,
		MaxHP:    merc.MaxHP,
		SP:       merc.SP,
		MaxSP:    merc.MaxSP,
		HasSP:    merc.MaxSP > 0,
		Friendly: true,
	}
	if m.ActorLife != nil {
		if cached, ok := m.ActorLife[actor.ID]; ok {
			if cached.MaxHP > 0 {
				life.HP = cached.HP
				life.MaxHP = cached.MaxHP
			}
			if cached.HasSP && cached.MaxSP > 0 {
				life.SP = cached.SP
				life.MaxSP = cached.MaxSP
				life.HasSP = true
			}
		}
	}
	if life.MaxHP <= 0 || life.HP < 0 {
		return Life{}, false
	}
	life.HP = scene.ClampInt(life.HP, 0, life.MaxHP)
	if life.HasSP {
		life.SP = scene.ClampInt(life.SP, 0, life.MaxSP)
	}
	return life, true
}

func (m *System) monsterLifeForSense(actorID uint32) (Life, bool) {
	if m.ActorLife == nil {
		return Life{}, false
	}
	life, ok := m.ActorLife[actorID]
	if !ok || life.MaxHP <= 0 || life.HP < 0 {
		return Life{}, false
	}
	return life, true
}

func localPlayerLifeForDisplay(ctx client.Context) (Life, bool) {
	if ctx.Session == nil {
		return Life{}, false
	}
	hp := ctx.Session.Vitals.HP
	maxHP := ctx.Session.Vitals.MaxHP
	sp := ctx.Session.Vitals.SP
	maxSP := ctx.Session.Vitals.MaxSP
	if maxHP <= 0 {
		character := ctx.Session.SelectedCharacter()
		hp = int(character.HP)
		maxHP = int(character.MaxHP)
	}
	if maxSP <= 0 {
		character := ctx.Session.SelectedCharacter()
		sp = int(character.SP)
		maxSP = int(character.MaxSP)
	}
	if maxHP <= 0 {
		return Life{}, false
	}
	return Life{
		HP:     scene.ClampInt(hp, 0, maxHP),
		MaxHP:  maxHP,
		SP:     scene.ClampInt(sp, 0, maxSP),
		MaxSP:  maxSP,
		HasSP:  maxSP > 0,
		Player: true,
	}, true
}

func (m *System) StartActorCastBar(ctx client.Context, sourceID uint32, duration time.Duration, started time.Time) {
	if ctx.Config.Headless || sourceID == 0 || duration <= 0 {
		return
	}
	if started.IsZero() {
		started = time.Now()
	}
	bar := CastBar{
		Started:  started,
		Duration: duration,
		Color:    color.RGBA{R: 0, G: 255, B: 0, A: 255},
	}
	m.setActorCastBar(sourceID, bar)
	if ctx.Session == nil || !scene.IsLocalActor(ctx, sourceID) {
		return
	}
	m.setActorCastBar(ctx.Session.AccountID, bar)
	m.setActorCastBar(ctx.Session.CharID, bar)
}

func (m *System) setActorCastBar(actorID uint32, bar CastBar) {
	if actorID == 0 {
		return
	}
	if m.ActorCastBars == nil {
		m.ActorCastBars = make(map[uint32]CastBar)
	}
	m.ActorCastBars[actorID] = bar
}

func (m *System) ClearActorCastBar(ctx client.Context, actorID uint32) {
	if actorID == 0 || m.ActorCastBars == nil {
		return
	}
	delete(m.ActorCastBars, actorID)
	if ctx.Session == nil || !scene.IsLocalActor(ctx, actorID) {
		return
	}
	delete(m.ActorCastBars, ctx.Session.AccountID)
	delete(m.ActorCastBars, ctx.Session.CharID)
}

func partyMemberLifeForDisplay(ctx client.Context, actor world.Actor) (Life, bool) {
	if ctx.Session == nil || actor.ID == 0 || !actor.HasObjectType || actor.ObjectType != scene.ActorObjectTypePC {
		return Life{}, false
	}
	for _, member := range ctx.Session.Party.Members {
		if member.AccountID != actor.ID || member.MaxHP <= 0 {
			continue
		}
		return Life{
			HP:     scene.ClampInt(member.HP, 0, member.MaxHP),
			MaxHP:  member.MaxHP,
			Player: true,
		}, true
	}
	return Life{}, false
}

func (m *System) serverProgressBar(localPlayer bool, now time.Time) (CastBar, bool) {
	if m.bindings.Progress.Started.IsZero() || !localPlayer {
		return CastBar{}, false
	}
	bar := CastBar{
		Started:  m.bindings.Progress.Started,
		Duration: m.bindings.Progress.Duration,
		Color:    m.bindings.Progress.Color,
	}
	_, active := ActorCastBarProgress(bar, now)
	return bar, active
}
