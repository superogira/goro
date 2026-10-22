package game

import (
	"image/color"
	"math"
	"os"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/db"
	"github.com/kivutar/goro/game/effects"
	"github.com/kivutar/goro/game/internal/testutil"
	maps "github.com/kivutar/goro/game/map"
	"github.com/kivutar/goro/game/scene"
	"github.com/kivutar/goro/game/sprites"
	"github.com/kivutar/goro/input"
	"github.com/kivutar/goro/network"
	"github.com/kivutar/goro/render"
	"github.com/kivutar/goro/res"
	"github.com/kivutar/goro/session"
	gameui "github.com/kivutar/goro/ui"
	worldstate "github.com/kivutar/goro/world"
)

func TestLevel99AuraAddsNormalEffectsForLocalPlayer(t *testing.T) {
	mode := &WorldMode{}
	now := time.Unix(10, 0)
	ctx := testutil.Level99AuraTestContext(99, false)

	mode.effects().SyncLevel99AuraEffects(ctx, now)

	if got := level99AuraEffectIDsForActor(mode.effects().Active, 2000000); !reflect.DeepEqual(got, effects.Level99AuraNormalEffects) {
		t.Fatalf("aura effects = %v, want %v", got, effects.Level99AuraNormalEffects)
	}
	for _, effect := range mode.effects().Active {
		if !effect.Persistent {
			t.Fatalf("effect %d is not persistent", effect.EffectID)
		}
		if effect.Expires.Sub(now) < effects.Level99AuraLifetime {
			t.Fatalf("effect %d lifetime = %s, want at least %s", effect.EffectID, effect.Expires.Sub(now), effects.Level99AuraLifetime)
		}
	}
}

func TestLevel99AuraUsesSimpleEffectWhenLessEffectsEnabled(t *testing.T) {
	mode := &WorldMode{}
	now := time.Unix(10, 0)
	ctx := testutil.Level99AuraTestContext(99, true)

	mode.effects().SyncLevel99AuraEffects(ctx, now)

	if got := level99AuraEffectIDsForActor(mode.effects().Active, 2000000); !reflect.DeepEqual(got, effects.Level99AuraSimpleEffects) {
		t.Fatalf("aura effects = %v, want %v", got, effects.Level99AuraSimpleEffects)
	}

	ctx.Session.LessEffects = false
	mode.effects().SyncLevel99AuraEffects(ctx, now.Add(time.Second))
	if got := level99AuraEffectIDsForActor(mode.effects().Active, 2000000); !reflect.DeepEqual(got, effects.Level99AuraNormalEffects) {
		t.Fatalf("normal aura effects after preference change = %v, want %v", got, effects.Level99AuraNormalEffects)
	}
}

func TestLevel99AuraRemovesEffectsBelowLevel(t *testing.T) {
	mode := &WorldMode{}
	now := time.Unix(10, 0)
	ctx := testutil.Level99AuraTestContext(99, false)
	mode.effects().SyncLevel99AuraEffects(ctx, now)

	ctx.Session.Progress.BaseLevel = 98
	mode.effects().SyncLevel99AuraEffects(ctx, now.Add(time.Second))

	if got := level99AuraEffectIDsForActor(mode.effects().Active, 2000000); len(got) != 0 {
		t.Fatalf("aura effects = %v, want none", got)
	}
}

func TestLevel99AuraAddsEffectsForParsedRemoteActorLevel(t *testing.T) {
	mode := &WorldMode{}
	ctx := testutil.Level99AuraTestContext(1, false)

	mode.upsertNetworkActor(ctx, network.ActorEntry{
		ID:       300,
		X:        12,
		Y:        34,
		Level:    99,
		HasLevel: true,
	})

	if got := level99AuraEffectIDsForActor(mode.effects().Active, 300); !reflect.DeepEqual(got, effects.Level99AuraNormalEffects) {
		t.Fatalf("remote aura effects = %v, want %v", got, effects.Level99AuraNormalEffects)
	}

	mode.applyActorVanish(ctx, network.ActorVanish{ID: 300, Reason: actorVanishLogout})
	if got := level99AuraEffectIDsForActor(mode.effects().Active, 300); len(got) != 0 {
		t.Fatalf("remote aura effects after vanish = %v, want none", got)
	}
}

func TestLevel99AuraRemovedWhenDeadActorCleanupRemovesActor(t *testing.T) {
	mode := &WorldMode{}
	now := time.Unix(10, 0)
	ctx := testutil.Level99AuraTestContext(1, false)
	ctx.World.Actors[300] = worldstate.Actor{
		ID:       300,
		X:        12,
		Y:        34,
		Level:    99,
		HasLevel: true,
	}
	mode.effects().SyncLevel99AuraEffects(ctx, now)
	mode.actorDeaths = map[uint32]time.Time{300: now.Add(-time.Second)}

	mode.cleanupDeadActors(ctx, now)

	if _, ok := ctx.World.Actors[300]; ok {
		t.Fatal("dead actor was not removed")
	}
	if got := level99AuraEffectIDsForActor(mode.effects().Active, 300); len(got) != 0 {
		t.Fatalf("remote aura effects after death cleanup = %v, want none", got)
	}
}

func level99AuraEffectIDsForActor(effects []effects.Instance, actorID uint32) []int {
	out := make([]int, 0, len(effects))
	for _, effect := range effects {
		if effect.ActorID == actorID && isLevel99AuraEffect(effect.EffectID) {
			out = append(out, effect.EffectID)
		}
	}
	sort.Ints(out)
	return out
}

func TestRuwachUsesSightSoundOncePerActivation(t *testing.T) {
	ctx := testutil.MonkEffectContext()
	m := &WorldMode{}
	for _, state := range []uint32{db.EffectStateRuwach, db.EffectStateRuwach, db.EffectStateRuwach | db.EffectStateFalcon} {
		m.applyActorStateChange(ctx, network.ActorStateChange{ID: 300, EffectState: state})
	}
	if len(m.scheduledSounds) != 1 || !reflect.DeepEqual(m.scheduledSounds[0].paths, []string{"effect\\ef_sight.wav"}) {
		t.Fatalf("Ruwach sounds = %+v", m.scheduledSounds)
	}
	m.applyActorStateChange(ctx, network.ActorStateChange{ID: 300})
	m.applyActorStateChange(ctx, network.ActorStateChange{ID: 300, EffectState: db.EffectStateRuwach})
	if len(m.scheduledSounds) != 2 || len(m.effects().Active) != 1 {
		t.Fatal("Ruwach did not restart on a new activation")
	}
}

func TestOverthrustAfterimages(t *testing.T) {
	ctx := testutil.ArcherEffectContext()
	start := time.Unix(100, 0)
	projection := testutil.Projection(800, 600, 10, 20, 0)
	billboard := &scene.Billboard{Image: render.NewImage(20, 40)}
	billboard.Image.Fill(color.RGBA{255, 255, 255, 255})
	for _, status := range []uint16{db.StatusOverthrust, db.StatusOverthrustmax} {
		m := &WorldMode{}
		m.applyStatusEffectChange(ctx, network.StatusEffectChange{ActorID: 2000000, StatusID: status, Active: true})
		entry := scene.ActorDrawEntry{Actor: ctx.World.Player, IsPlayer: true, WorldX: 10.5, WorldY: 20.5, Scale: 1}
		tint := actorStateTint(entry.Actor)
		if tint != (color.RGBA{250, 150, 150, 255}) {
			t.Fatalf("incorrect native Overthrust tint: %v", tint)
		}
		for tick := 0; tick <= 100; tick++ {
			frame := render.NewFrame(800, 600)
			entry.WorldX = 10.5 + float64(tick)/10
			m.drawActorSpriteWithEffects3D(ctx, frame, projection, entry, billboard, 1, 1, 1, tint, render.BlendSourceOver, start.Add(time.Duration(tick)*db.EffectFrameDuration))
			if len(m.effects().ActorBlurs[entry.Actor.ID]) > 5 {
				t.Fatal("afterimages must remain bounded during long movement")
			}
			if tick == 5 {
				commands := reflect.ValueOf(frame).Elem().FieldByName("worldBillboards")
				if commands.Len() != 3 {
					t.Fatal("expected two frozen poses and the current actor")
				}
				old := commands.Index(0)
				if old.FieldByName("Center").Index(0).Float() != 10.5 || math.Abs(old.FieldByName("ColorA").Float()-140.0/255) > .005 {
					t.Fatal("afterimage must keep its captured position and fade by eight per tick")
				}
				if old.FieldByName("Options").FieldByName("DepthWrite").Bool() {
					t.Fatal("fading afterimages must not occlude the actor or other sprites")
				}
			}
		}
		m.applyStatusEffectChange(ctx, network.StatusEffectChange{ActorID: 2000000, StatusID: status})
		entry.Actor = ctx.World.Player
		m.effects().DrawActorBlurs(render.NewFrame(800, 600), projection, entry, billboard, 1, 1, 1, tint, start.Add(124*db.EffectFrameDuration))
		if len(m.effects().ActorBlurs) != 0 {
			t.Fatal("afterimages survived the end of the buff and fade")
		}
	}
}

func TestOverthrustAfterimageVisibilityAndRemoval(t *testing.T) {
	ctx := testutil.ArcherEffectContext()
	entry := scene.ActorDrawEntry{Actor: ctx.World.Player, IsPlayer: true, WorldX: 10.5, WorldY: 20.5, Scale: 1}
	entry.Actor.Opt3State = db.Opt3Overthrust
	projection := testutil.Projection(800, 600, 10, 20, 0)
	billboard := &scene.Billboard{Image: render.NewImage(20, 40)}
	billboard.Image.Fill(color.RGBA{255, 255, 255, 255})
	for _, kind := range []string{"dyed", "monster", "hidden", "removed", "remote"} {
		m := &WorldMode{}
		e := entry
		switch kind {
		case "dyed":
			e.Actor.BodyPal = 1
		case "monster":
			e.IsPlayer = false
			e.Actor = ctx.World.Actors[300]
			e.Actor.Opt3State = db.Opt3Overthrust
		case "hidden":
			m.effects().DrawActorBlurs(render.NewFrame(800, 600), projection, e, billboard, 1, 1, 1, color.RGBA{255, 255, 255, 255}, time.Now())
			e.Actor.EffectState = db.EffectStateHide
		case "remote":
			e.IsPlayer, e.Actor.Appearance = false, true
		}
		m.effects().DrawActorBlurs(render.NewFrame(800, 600), projection, e, billboard, 1, 1, 1, color.RGBA{255, 255, 255, 255}, time.Now())
		if kind == "removed" || kind == "remote" {
			if len(m.effects().ActorBlurs) != 1 {
				t.Fatal("visible player did not produce an afterimage")
			}
			m.clearRemovedActorState(e.Actor.ID)
		}
		if len(m.effects().ActorBlurs) != 0 {
			t.Fatalf("%s actor retained afterimages", kind)
		}
	}
}

func TestBardNativeAnimationFrames(t *testing.T) {
	ctx := testutil.ArcherEffectContext()
	start := time.Unix(100, 0)
	action := res.ACTAction{Animations: make([]res.ACTAnimation, 6)}
	for _, tc := range []struct {
		skill  uint16
		frames []int
	}{{db.SkillBaWhistle, []int{1, 2, 3, 1, 2, 3}}, {db.SkillBDRokisweil, []int{1, 2, 3, 2, 1, 2}}, {db.SkillBaFrostjoke, []int{4, 5, 4, 5, 4, 5}}, {db.SkillBaPangvoice, []int{4, 5, 4, 5, 4, 5}}} {
		m := &WorldMode{}
		m.startSkillActionAnimation(ctx, 2000000, ctx.World.Player, skillAction(tc.skill), start, time.Second)
		for i, frame := range tc.frames {
			now := start.Add(time.Duration(i)*10*db.EffectFrameDuration + time.Millisecond)
			anim, _ := m.actorAnimation(2000000, now)
			state := sprites.State{}
			applyActorAnimationToSpriteState(&state, anim, true)
			if got := sprites.BodyMotionForState(action, state, start, now); got != frame {
				t.Fatalf("skill %d tick %d: motion %d, want %d", tc.skill, i, got, frame)
			}
		}
		if tc.skill == db.SkillBaFrostjoke || tc.skill == db.SkillBaPangvoice {
			anim, _ := m.actorAnimation(2000000, start.Add(1800*time.Millisecond))
			if anim.actionFamily != scene.SpriteActionIdle {
				t.Fatal("shout animation must stop after 1800ms")
			}
		} else {
			m.applySkillNoDamageNotify(ctx, network.SkillNoDamageNotify{SkillID: db.SkillBDAdaptation, SourceID: 2000000, TargetID: 2000000, Result: 1})
			if m.actorAnims[2000000].actionFamily != scene.SpriteActionIdle {
				t.Fatal("Adaptation must return to standing")
			}
		}
	}
	m := &WorldMode{}
	m.startSkillActionAnimation(ctx, 2000000, ctx.World.Player, attackSkillActionSpec, start, time.Second)
	m.stopPerformance(ctx, network.StatusEffectChange{StatusID: db.StatusBdplaying})
	if len(m.actorAnims) != 2 {
		t.Fatal("ending a song cancelled an unrelated attack")
	}
}

func TestGypsySkillAnimationsWithClientAssets(t *testing.T) {
	ctx := testutil.ArcherEffectContext()
	ctx.Resources = testutil.RealDataManager(t)
	ctx.World.Player.Job, ctx.World.Player.Weapon, ctx.World.Player.Sex = db.JobDancerH, 1950, 0 // Rope
	ctx.Session.Selected.Job, ctx.Session.Selected.Weapon = db.JobDancerH, 1950
	view, status := sprites.LoadPlayer(ctx.Resources, ctx.Session.Selected, 0, false)
	if view == nil || !strings.Contains(view.Body.ActSource, res.PlayerJobToken(db.JobDancerH)) {
		t.Fatalf("must load the Gypsy body, without a Novice fallback: %s", status)
	}
	m := &WorldMode{playerView: view}
	for direction := 0; direction < 8; direction++ {
		ctx.World.Player.Dir = direction
		a := network.ActorActionNotify{SourceID: 2000000, TargetID: 300, SkillID: db.SkillCGArrowvulcan, SourceSpeed: 500, HitCount: 9, Damage: 100}
		m.applyActorActionNotify(ctx, a)
		anim := m.actorAnims[a.SourceID]
		if anim.actionFamily != sprites.ActionPCAttack3 || anim.next == nil || anim.next.actionFamily != sprites.ActionPCAttack2 {
			t.Fatal("Gypsy must shoot with ACT 88, then use her whip attack for additional hits")
		}
		for _, step := range []actorAnimation{anim, *anim.next, *anim.next.next} {
			index, action, ok := sprites.ResolveAction(view.Body.Act, step.actionFamily, direction)
			if !ok || index/8 != step.actionFamily || step.hasFixedMotion && step.fixedMotion >= len(action.Animations) {
				t.Fatal("Gypsy attack must use its requested action and frame without fallback")
			}
			state := sprites.State{Direction: direction}
			applyActorAnimationToSpriteState(&state, step, true)
			billboard, ok := sprites.HumanoidBillboard(view, state, step.started)
			if !ok || billboard == nil || billboard.Image == nil {
				t.Fatalf("Gypsy attack failed to render for direction %d", direction)
			}
		}
		m.applyGroundSkillNotify(ctx, network.GroundSkillNotify{SkillID: db.SkillCGHermode, SourceID: a.SourceID})
		dance := m.actorAnims[a.SourceID]
		_, action, ok := sprites.ResolveAction(view.Body.Act, sprites.ActionPCSkill, direction)
		if !ok || len(action.Animations) < 4 || !dance.pingPong {
			t.Fatal("Gypsy must have the complete three-frame dance, without falling back to another action")
		}
		state := sprites.State{Direction: direction}
		applyActorAnimationToSpriteState(&state, dance, true)
		for i, want := range []int{1, 2, 3, 2, 1} {
			at := dance.started.Add(time.Duration(i) * dance.speed)
			if got := sprites.BodyMotionForState(action, state, dance.started, at); got != want {
				t.Fatalf("Gypsy dance frame %d: got %d, want %d", i, got, want)
			}
		}
	}
}

func TestClownGypsyMarionetteLinkLifetime(t *testing.T) {
	for _, end := range []uint16{db.StatusMarionetteMaster, db.StatusMarionette} {
		ctx := testutil.ArcherEffectContext()
		m := &WorldMode{}
		m.applyStatusEffectChange(ctx, network.StatusEffectChange{StatusID: db.StatusMarionetteMaster, ActorID: 2000000, Active: true})
		m.applyStatusEffectChange(ctx, network.StatusEffectChange{StatusID: db.StatusMarionette, ActorID: 300, Active: true})
		m.applySkillNoDamageNotify(ctx, network.SkillNoDamageNotify{SkillID: db.SkillCGMarionette, SourceID: 100, TargetID: 300, Result: 1})
		if len(m.effects().Active) != 3 || len(m.scheduledSounds) != 1 {
			t.Fatal("Marionette needs one link, two body glows, and one sound")
		}
		link := m.effects().Active[2]
		if link.EffectID != effects.MarionetteLink || link.ActorID != 2000000 || link.TargetID != 300 || !link.Expires.IsZero() {
			t.Fatal("link must connect the caster's canonical ID to the recipient")
		}
		id := uint32(100)
		if end == db.StatusMarionette {
			id = 300
		}
		m.applyStatusEffectChange(ctx, network.StatusEffectChange{StatusID: end, ActorID: id})
		if len(m.effects().Active) != 1 || m.effects().Active[0].EffectID != effects.PinkBody {
			t.Fatal("ending either endpoint must remove the link and its own body glow")
		}
	}
	ctx := testutil.ArcherEffectContext()
	m := &WorldMode{}
	n := network.SkillNoDamageNotify{SkillID: db.SkillCGMarionette, SourceID: 2000000, TargetID: 300, Result: 1}
	m.applySkillNoDamageNotify(ctx, n)
	m.effects().RemoveActor(300)
	if len(m.effects().Active) != 1 || m.effects().Active[0].ActorID != n.SourceID {
		t.Fatal("recipient leaving view must remove the link as well as its body effect")
	}
	m = &WorldMode{}
	n.TargetID = 999
	m.applySkillNoDamageNotify(ctx, n)
	if len(m.effects().Active) != 0 {
		t.Fatal("missing recipient must not leave an immortal link")
	}
}

func TestCreatorCastAnimationsWithClientAssets(t *testing.T) {
	ctx := testutil.ArcherEffectContext()
	ctx.Resources = testutil.RealDataManager(t)
	ctx.World.Player.Job, ctx.Session.Selected.Job = db.JobAlchemistH, db.JobAlchemistH
	start := time.Unix(100, 0)
	for _, sex := range []uint8{0, 1} {
		ctx.World.Player.Sex = sex
		view, status := sprites.LoadPlayer(ctx.Resources, ctx.Session.Selected, sex, false)
		if view == nil || !strings.Contains(view.Body.ActSource, res.PlayerJobToken(db.JobAlchemistH)) {
			t.Fatalf("must load the Creator body without a Novice fallback: %s", status)
		}
		m := &WorldMode{playerView: view}
		for _, skill := range []uint16{db.SkillCRSlimpitcher, db.SkillCRAciddemonstration} {
			m.startSkillSourceCastAnimation(ctx, 2000000, skill, 3*time.Second, start)
			anim := m.actorAnims[2000000]
			if anim.actionFamily != sprites.ActionPCAttack1 || anim.duration != time.Second || anim.speed != 10*db.EffectFrameDuration || anim.length != 5 || anim.loop {
				t.Fatal("Creator throw must use the finite native five-frame ACT 40 sequence")
			}
			for direction := 0; direction < 8; direction++ {
				index, action, ok := sprites.ResolveAction(view.Body.Act, sprites.ActionPCAttack1, direction)
				if !ok || index/8 != sprites.ActionPCAttack1 || len(action.Animations) < 5 {
					t.Fatal("Creator body lacks its full throwing sequence")
				}
				state := sprites.State{Direction: direction}
				applyActorAnimationToSpriteState(&state, anim, true)
				for i, want := range []int{0, 1, 2, 3, 4, 4} {
					at := start.Add(time.Duration(i)*anim.speed + time.Millisecond)
					if got := sprites.BodyMotionForState(action, state, start, at); got != want {
						t.Fatalf("throw step %d: got frame %d, want %d", i, got, want)
					}
				}
			}
			if _, ok := m.actorAnimation(2000000, start.Add(time.Second)); ok {
				t.Fatal("throw pose outlived its native one-second duration")
			}
		}
	}
	for _, tc := range []struct {
		skill      uint16
		pose, aura int
	}{
		{db.SkillCRSlimpitcher, sprites.ActionPCAttack1, effects.BeginSpell2},
		{db.SkillCRAciddemonstration, sprites.ActionPCAttack1, effects.BeginSpell2},
		{db.SkillCRFullprotection, scene.SpriteActionIdle, effects.BeginSpell2},
		{db.SkillCRCultivation, sprites.ActionPCReadyFight, effects.BashBegin},
	} {
		m := &WorldMode{}
		m.startSkillSourceCastAnimation(ctx, 2000000, tc.skill, time.Second, start)
		m.addSkillCastEffects(ctx, tc.skill, 1, 2000000, 0, 0, 0, time.Second, start, "test")
		if m.actorAnims[2000000].actionFamily != tc.pose || len(m.effects().Active) != 2 || m.effects().Active[1].EffectID != tc.aura {
			t.Fatalf("incorrect cast pose or aura for skill %d", tc.skill)
		}
	}
}

func TestHighWizardMagicPowerLifecycle(t *testing.T) {
	ctx := testutil.ArcherEffectContext()
	m := &WorldMode{}
	change := network.StatusEffectChange{StatusID: db.StatusMagicpower, ActorID: 2000000, Active: true}
	m.applyStatusEffectChange(ctx, change)
	if len(m.effects().Active) != 1 || m.effects().Active[0].EffectID != 382 || !m.effects().Active[0].Persistent || len(m.scheduledSounds) != 0 {
		t.Fatal("status entry must restore the weapon sparkle silently")
	}
	n := network.SkillNoDamageNotify{SkillID: db.SkillHWMagicpower, SourceID: 2000000, TargetID: 2000000, Result: 1}
	for i := 0; i < 2; i++ {
		m.applySkillNoDamageNotify(ctx, n)
		count := 0
		for _, e := range m.effects().Active {
			if e.EffectID == 382 {
				count++
			}
		}
		if count != 1 {
			t.Fatal("recasting must replace the weapon sparkle")
		}
	}
	if len(m.scheduledSounds) != 2 || m.scheduledSounds[0].positioned || m.scheduledSounds[0].paths[0] != "effect\\마법력 증폭.wav" {
		t.Fatal("each successful activation must play one centered amplification sound")
	}
	change.Active = false
	m.applyStatusEffectChange(ctx, change)
	if m.effects().HasActive(effects.LightBlade, 2000000, time.Now()) {
		t.Fatal("weapon sparkle survived buff removal")
	}
	ctx.World.Player.Opt3State = db.Opt3Lightblade
	m.syncActorSpiritEffects(ctx, 2000000)
	if !m.effects().HasActive(effects.LightBlade, 2000000, time.Now()) {
		t.Fatal("actor entry must restore the weapon sparkle")
	}
}

func TestHighWizardClientAssets(t *testing.T) {
	ctx := testutil.ArcherEffectContext()
	ctx.Resources = testutil.RealDataManager(t)
	ctx.World.Player.Job, ctx.Session.Selected.Job = db.JobWizardH, db.JobWizardH
	start := time.Unix(100, 0)
	for _, sex := range []uint8{0, 1} {
		view, status := sprites.LoadPlayer(ctx.Resources, ctx.Session.Selected, sex, false)
		if view == nil || !strings.Contains(view.Body.ActSource, res.PlayerJobToken(db.JobWizardH)) {
			t.Fatalf("must load the High Wizard body without fallback: %s", status)
		}
		m := &WorldMode{playerView: view}
		m.startSkillNoDamageSourceAnimation(ctx, network.SkillNoDamageNotify{SkillID: db.SkillHWSouldrain, SourceID: 2000000, TargetID: 300}, start)
		for dir := 0; dir < 8; dir++ {
			_, action, ok := sprites.ResolveAction(view.Body.Act, 11, dir)
			if !ok || len(action.Animations) < 4 {
				t.Fatal("High Wizard lacks Soul Drain's forced ACT sequence")
			}
			state := sprites.State{Direction: dir}
			applyActorAnimationToSpriteState(&state, m.actorAnims[2000000], true)
			for tick, want := range []int{0, 1, 2, 3, 3, 3} {
				if got := sprites.BodyMotionForState(action, state, start, start.Add(time.Duration(tick)*10*db.EffectFrameDuration+time.Millisecond)); got != want {
					t.Fatalf("Soul Drain frame %d: got %d, want %d", tick, got, want)
				}
			}
		}
	}
	for _, tc := range []struct {
		skill uint16
		aura  int
	}{
		{db.SkillHWMagiccrasher, 55}, {db.SkillHWMagicpower, 16}, {db.SkillHWNapalmvulcan, 55}, {db.SkillHWGanbantein, 58}, {db.SkillHWGravitation, 55},
	} {
		m := &WorldMode{}
		m.addSkillCastEffects(ctx, tc.skill, 3, 2000000, 0, 0, 0, time.Second, start, "test")
		if len(m.effects().Active) != 2 || m.effects().Active[1].EffectID != tc.aura {
			t.Fatalf("incorrect native casting aura for %d", tc.skill)
		}
	}
	projection := testutil.Projection(800, 600, 20, 20, 0)
	for _, tc := range []struct{ effect, tick, billboards, meshes int }{
		{378, 25, 21, 0}, {378, 60, 0, 0}, {223, 9, 0, 1}, {223, 34, 0, 0}, {522, 25, 0, 2}, {401, 20, 0, 0},
	} {
		m := &WorldMode{}
		if tc.effect == 378 {
			m.effects().AddBetween(ctx, tc.effect, 2000000, 300, start)
		} else {
			m.effects().AddCellLifetime(ctx, tc.effect, 0, 20, 20, start, 0, false)
		}
		frame := render.NewFrame(800, 600)
		m.effects().Draw(frame, ctx, projection, start.Add(time.Duration(tc.tick)*db.EffectFrameDuration), false)
		v := reflect.ValueOf(frame).Elem()
		if v.FieldByName("worldBillboards").Len() != tc.billboards || v.FieldByName("worldCommands").Len() != tc.meshes {
			t.Fatalf("effect %d frame %d rendered the wrong geometry", tc.effect, tc.tick)
		}
		if tc.effect == 522 && v.FieldByName("worldCommands").Index(0).FieldByName("Vertices").Len() != 24 {
			t.Fatal("Gravitation must draw two four-sided spikes per texture")
		}
	}
	for _, sound := range []string{"effect\\마법력 증폭.wav", "effect\\매직 크래쉬.wav", "effect\\EF_NapalmBeat.wav", "effect\\EF_FireWall.wav", "effect\\wizard_earthspike.wav"} {
		if _, err := ctx.Resources.ReadFile("data/wav/" + sound); err != nil {
			t.Fatal(err)
		}
	}
}

func TestLordKnightBerserkState(t *testing.T) {
	ctx := testutil.KnightEffectContext()
	m := &WorldMode{}
	change := network.StatusEffectChange{StatusID: db.StatusBerserk, ActorID: 2000000, Active: true}
	m.applyStatusEffectChange(ctx, change)
	m.applySkillNoDamageNotify(ctx, network.SkillNoDamageNotify{SkillID: db.SkillLKBerserk, SourceID: 2000000, TargetID: 2000000, Result: 1})
	count := 0
	for _, effect := range m.effects().Active {
		if effect.EffectID == effects.RedBody {
			count++
			if !effect.Persistent || !effect.Expires.IsZero() {
				t.Fatal("Berserk must follow server state")
			}
		}
	}
	if count != 1 || len(m.scheduledSounds) != 1 {
		t.Fatal("status plus skill notification duplicated Berserk")
	}
	change.Active = false
	m.applyStatusEffectChange(ctx, change)
	if m.effects().HasActive(effects.RedBody, 2000000, time.Now()) {
		t.Fatal("Berserk lingered after status ended")
	}
	actor := ctx.World.Actors[300]
	actor.Opt3State, actor.HasState = db.Opt3Berserk, true
	ctx.World.Actors[300] = actor
	m = &WorldMode{}
	m.syncCurrentActorEffectStateEffects(ctx)
	if !m.effects().HasActive(effects.RedBody, 300, time.Now()) || len(m.scheduledSounds) != 0 {
		t.Fatal("existing Berserk state must restore silently")
	}
	m.removeActorEffectStateEffects(300)
	if m.effects().HasActive(effects.RedBody, 300, time.Now()) {
		t.Fatal("vanished actor retained Berserk")
	}
}

func TestLordKnightConcentrationState(t *testing.T) {
	for _, actorID := range []uint32{0, 100, 2000000, 300} {
		for _, statusFirst := range []bool{false, true} {
			ctx := testutil.KnightEffectContext()
			m := &WorldMode{}
			owner := actorID
			if owner != 300 {
				owner = 2000000
			}
			change := network.StatusEffectChange{StatusID: db.StatusLkconcentration, ActorID: actorID, Active: true}
			if statusFirst {
				m.applyStatusEffectChange(ctx, change)
			}
			notify := network.SkillNoDamageNotify{SkillID: db.SkillLKConcentration, SourceID: owner, TargetID: owner, Result: 1}
			m.applySkillNoDamageNotify(ctx, notify)
			if !statusFirst {
				m.applyStatusEffectChange(ctx, change)
			}
			later := time.Now().Add(time.Hour)
			if !m.effects().HasActive(effects.LKConcentration, owner, later) || len(m.effects().Active) != 1 || len(m.scheduledSounds) != 1 {
				t.Fatalf("actor=%d statusFirst=%t: Concentration must persist with one activation sound", actorID, statusFirst)
			}
			m.applySkillNoDamageNotify(ctx, notify)
			if len(m.effects().Active) != 1 || len(m.scheduledSounds) != 2 {
				t.Fatal("recasting must replace the activation effect and play its sound once")
			}
			m.removeActorEffectStateEffects(owner)
			if len(m.effects().Active) != 0 {
				t.Fatal("vanishing retained Concentration")
			}
			if owner == 2000000 {
				m = &WorldMode{}
				m.syncCurrentActorEffectStateEffects(ctx)
				if !m.effects().HasActive(effects.LKConcentration, owner, later) || len(m.scheduledSounds) != 0 {
					t.Fatal("map entry must restore the active local buff without replaying its sound")
				}
			} else {
				m.applyStatusEffectChange(ctx, change)
			}
			change.Active = false
			m.applyStatusEffectChange(ctx, change)
			if m.effects().HasActive(effects.LKConcentration, owner, time.Now()) {
				t.Fatal("Concentration tint survived status end")
			}
		}
	}
	ctx := testutil.KnightEffectContext()
	ctx.Session.Statuses.Active = map[uint16]session.StatusEffect{
		db.StatusLkconcentration: {HasDuration: true, ExpiresAt: time.Now().Add(-time.Second)},
	}
	m := &WorldMode{}
	m.syncCurrentActorEffectStateEffects(ctx)
	if len(m.effects().Active) != 0 {
		t.Fatal("map entry restored an expired Concentration buff")
	}
}

func TestLordKnightRealAssets(t *testing.T) {
	root := os.Getenv("GORO_DATA_DIR")
	if root == "" {
		t.Skip("set GORO_DATA_DIR to validate original Lord Knight assets")
	}
	manager, err := res.NewManager(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, archive := range manager.Archives {
		t.Cleanup(func() { _ = archive.Close() })
	}
	ctx := testutil.KnightEffectContext()
	ctx.Resources = manager
	start := time.Unix(100, 0)
	for _, skill := range []uint16{db.SkillLKConcentration, db.SkillLKBerserk} {
		for _, sound := range db.SkillEffects[skill].SFX {
			if _, err := manager.ReadFile("data/wav/" + sound); err != nil {
				t.Fatal(err)
			}
		}
	}
	for _, id := range []int{effects.AuraBlade, effects.Magnum2, effects.Bash3D3, effects.Bash3D4, effects.Guard, effects.LKConcentration} {
		m := &WorldMode{}
		spec, _ := effects.Lookup(id)
		for _, sound := range spec.Sfx {
			if _, err := manager.ReadFile("data/wav/" + sound); err != nil {
				t.Fatal(err)
			}
		}
		m.effects().AddAt(ctx, id, 300, start)
		drew := false
		for _, tick := range []int{0, 15, 24, 30, 40, 60} {
			frame := render.NewFrame(800, 600)
			m.effects().Draw(frame, ctx, testutil.Projection(800, 600, 30.5, 20.5, 0), start.Add(time.Duration(tick)*db.EffectFrameDuration), false)
			drew = drew || reflect.ValueOf(frame).Elem().FieldByName("worldCommands").Len() > 0
		}
		if !drew {
			t.Fatalf("effect %d drew nothing", id)
		}
	}
	m := &WorldMode{}
	m.applyStatusEffectChange(ctx, network.StatusEffectChange{StatusID: db.StatusLkconcentration, ActorID: 300, Active: true})
	effect := m.effects().Active[0]
	spec, _ := effects.Lookup(effects.LKConcentration)
	duration := m.effects().WorldEffectResolvedComponentDuration(ctx, spec, spec.Components[0])
	for _, cycle := range []int{0, 1, 2} {
		frame := render.NewFrame(800, 600)
		at := effect.Starts.Add(time.Duration(cycle)*duration + duration/2)
		m.effects().Draw(frame, ctx, testutil.Projection(800, 600, 30.5, 20.5, 0), at, false)
		draws := reflect.ValueOf(frame).Elem().FieldByName("worldCommands").Len()
		if (draws > 0) != (cycle == 0) {
			t.Fatalf("Concentration activation must play exactly once: cycle=%d draws=%d", cycle, draws)
		}
		if !m.effects().HasActive(effects.LKConcentration, 300, at) {
			t.Fatal("Concentration tint expired with its activation STR")
		}
	}
	for _, tc := range []struct {
		id     int
		delays []time.Duration
	}{
		{effects.Magnum2, []time.Duration{20 * db.EffectFrameDuration, 30 * db.EffectFrameDuration}},
		{effects.Bash3D3, []time.Duration{0}},
		{effects.Bash3D4, []time.Duration{22 * db.EffectFrameDuration}},
	} {
		spec, _ := effects.Lookup(tc.id)
		if !reflect.DeepEqual(spec.SfxDelays, tc.delays) {
			t.Fatalf("effect %d sound timing", tc.id)
		}
	}

}

func TestMageSpriteSequencesAndSightLifetime(t *testing.T) {
	ctx := testutil.KnightEffectContext()
	start := time.Unix(100, 0)
	projection := testutil.Projection(800, 600, 20, 20, 0)
	for _, tc := range []struct {
		skill uint16
		file  string
		tick  int
		draws int
	}{
		{db.SkillMGFireball, "fireball", 19, 0},
		{db.SkillMGFireball, "fireball", 20, 1},
		{db.SkillMGFireball, "fireball", 32, 4},
		{db.SkillMGFireball, "fireball", 52, 0},
		{db.SkillMGSoulstrike, "particle1", 5, 0},
		{db.SkillMGSoulstrike, "particle1", 6, 12},
		{db.SkillMGSoulstrike, "particle1", 17, 24},
	} {
		m := &WorldMode{effectSystem: effects.System{EffectViews: map[string]*scene.SpriteView{tc.file: testutil.EffectRenderTestSprite()}}}
		m.addSkillBeforeHitEffect(ctx, network.ActorActionNotify{SkillID: tc.skill, SourceID: 2000000, TargetID: 300, HitCount: 4}, start)
		frame := render.NewFrame(800, 600)
		m.effects().Draw(frame, ctx, projection, start.Add(time.Duration(tc.tick)*db.EffectFrameDuration), false)
		if got := reflect.ValueOf(frame).Elem().FieldByName("worldBillboards").Len(); got != tc.draws {
			t.Fatalf("skill %d tick %d: got %d sprites, want %d", tc.skill, tc.tick, got, tc.draws)
		}
	}
	m := &WorldMode{effectSystem: effects.System{EffectViews: map[string]*scene.SpriteView{"sight": testutil.EffectRenderTestSprite(), "data\\sprite\\shadow": testutil.EffectRenderTestSprite()}}}
	m.applyActorEffectStateEffects(ctx, 300, 0, db.EffectStateSight)
	m.applyActorEffectStateEffects(ctx, 300, db.EffectStateSight, db.EffectStateSight)
	if len(m.effects().Active) != 1 || len(m.scheduledSounds) != 1 || !m.effects().Active[0].Expires.IsZero() {
		t.Fatal("Sight must persist without replaying its activation sound")
	}
	frame := render.NewFrame(800, 600)
	m.effects().Draw(frame, ctx, projection, m.effects().Active[0].Starts.Add(time.Minute), false)
	if got := reflect.ValueOf(frame).Elem().FieldByName("worldBillboards").Len(); got != 20 {
		t.Fatalf("Sight lost its ten flames and ten shadows: %d", got)
	}
	m.applyActorEffectStateEffects(ctx, 300, db.EffectStateSight, 0)
	if len(m.effects().Active) != 0 {
		t.Fatal("Sight remained after the state ended")
	}
}

func TestSpiritSpherePacketsUpdateCountsAndLifecycle(t *testing.T) {
	for _, actorID := range []uint32{2000000, 300} {
		ctx := testutil.MonkEffectContext()
		m := &WorldMode{}
		for _, count := range []uint16{1, 3, 3, 5, 2, 0, 1} {
			m.handleNetworkPacket(ctx, testutil.SpiritSpherePacket(network.PacketZCSpirits, actorID, count), time.Now())
			actor, _, _ := scene.ActorForCombatID(ctx, actorID)
			if actor.SpiritSpheres != count || !actor.HasSpiritSpheres {
				t.Fatalf("actor %d count=%d, want %d", actorID, actor.SpiritSpheres, count)
			}
			if m.effects().HasActive(effects.Chookgi, actorID, time.Now().Add(time.Hour)) != (count > 0) {
				t.Fatalf("sphere effect does not follow count %d: %+v", count, m.effects().Active)
			}
			if count > 0 && len(m.effects().Active) != 1 {
				t.Fatal("sphere updates duplicate the effect")
			}
		}
		m.applyActorStateChange(ctx, network.ActorStateChange{ID: actorID, EffectState: db.EffectStateCloak})
		if m.effects().HasActive(effects.Chookgi, actorID, time.Now()) {
			t.Fatal("spheres reveal a cloaked actor")
		}
		m.applyActorStateChange(ctx, network.ActorStateChange{ID: actorID})
		if !m.effects().HasActive(effects.Chookgi, actorID, time.Now()) {
			t.Fatal("spheres did not return on reveal")
		}
		m.applyActorVanish(ctx, network.ActorVanish{ID: actorID})
		if m.effects().HasActive(effects.Chookgi, actorID, time.Now()) {
			t.Fatal("sphere effect survived actor removal")
		}
	}
}

func TestSpiritEffectsSurviveLoginAndActorRefresh(t *testing.T) {
	ctx := testutil.MonkEffectContext()
	for _, id := range []uint32{2000000, 300} {
		if handled, err := applyLoginActorBootstrapPacket(ctx, testutil.SpiritSpherePacket(network.PacketZCSpirits2, id, 3)); !handled || err != nil {
			t.Fatal("login dropped sphere packet")
		}
		applyLoginStatusEffectPacket(ctx, testStatusEffectChangePacket(db.StatusExplosionspirits, id, true))
	}
	m := &WorldMode{}
	m.syncCurrentActorEffectStateEffects(ctx)
	if len(m.effects().Active) != 4 {
		t.Fatalf("login effects = %+v, want spheres and Fury for both actors", m.effects().Active)
	}
	original := append([]effects.Instance(nil), m.effects().Active...)
	m.upsertNetworkActor(ctx, network.ActorEntry{ID: 300, X: 13, Y: 20, Job: db.JobMonk, Appearance: true, HasState: true})
	// Account and character IDs refer to the same actor, without duplicating its aura.
	m.applyStatusEffectChange(ctx, network.StatusEffectChange{StatusID: db.StatusExplosionspirits, ActorID: 150000, Active: true})
	if !reflect.DeepEqual(m.effects().Active, original) || ctx.World.Actors[300].SpiritSpheres != 3 {
		t.Fatal("refresh restarted or duplicated state effects")
	}
	for _, id := range []uint32{2000000, 300} {
		m.applyStatusEffectChange(ctx, network.StatusEffectChange{StatusID: db.StatusExplosionspirits, ActorID: id, Active: false})
		if m.effects().HasActive(effects.Gumgang, id, time.Now()) {
			t.Fatal("Fury aura survived status removal")
		}
	}
	// A new WorldMode is created on a map transition, retaining local state.
	next := &WorldMode{}
	next.syncCurrentActorEffectStateEffects(ctx)
	if !next.effects().HasActive(effects.Chookgi, 2000000, time.Now()) {
		t.Fatal("map transition lost local spheres")
	}
}

func TestMonkStateEffectsDrawExactCountsAndKeepLooping(t *testing.T) {
	ctx := testutil.MonkEffectContext()
	ctx.Resources = &res.Manager{}
	m := &WorldMode{mapScene: maps.System{Textures: make(map[string]*render.Image)}}
	for _, id := range []int{effects.Chookgi, effects.Gumgang} {
		spec, _ := effects.Lookup(id)
		for _, component := range spec.Components {
			for _, path := range append(component.TextureFiles, component.TextureFile) {
				m.mapScene.Textures["__effectfile_"+path+"|blackkey=true"] = render.NewImage(8, 8)
			}
		}
	}
	projection := testutil.Projection(800, 600, 12.5, 20.5, 0)
	for _, count := range []uint16{1, 3, 5, 0} {
		m.handleNetworkPacket(ctx, testutil.SpiritSpherePacket(network.PacketZCSpirits, 300, count), time.Now())
		frame := render.NewFrame(800, 600)
		m.effects().Draw(frame, ctx, projection, time.Now().Add(time.Hour), false)
		if got := reflect.ValueOf(frame).Elem().FieldByName("worldBillboards").Len(); got != int(count)*2 {
			t.Fatalf("count %d drew %d billboards, want %d", count, got, count*2)
		}
	}
	m.applyStatusEffectChange(ctx, network.StatusEffectChange{StatusID: db.StatusExplosionspirits, ActorID: 300, Active: true})
	for _, elapsed := range []time.Duration{0, time.Second, time.Hour} {
		frame := render.NewFrame(800, 600)
		m.effects().Draw(frame, ctx, projection, m.effects().Active[0].Starts.Add(elapsed), false)
		commands := reflect.ValueOf(frame).Elem().FieldByName("worldBillboards")
		if commands.Len() < 3 || commands.Len() > 4 {
			t.Fatalf("Fury at %s drew %d flashes, want 3 or 4", elapsed, commands.Len())
		}
		if commands.Index(0).FieldByName("ColorB").Float() != 1 {
			t.Fatal("Fury must use the original blue tint")
		}
	}
}

func TestMonkEffectsRenderWithOldROAssets(t *testing.T) {
	ctx := testutil.MonkEffectContext()
	ctx.Resources = testutil.RealDataManager(t)
	m := &WorldMode{}
	m.handleNetworkPacket(ctx, testutil.SpiritSpherePacket(network.PacketZCSpirits, 300, 3), time.Now())
	m.applyStatusEffectChange(ctx, network.StatusEffectChange{StatusID: db.StatusExplosionspirits, ActorID: 300, Active: true})
	projection := testutil.Projection(800, 600, 12.5, 20.5, 0)
	for _, elapsed := range []time.Duration{time.Second, 20 * time.Second, time.Hour} {
		frame := render.NewFrame(800, 600)
		m.effects().Draw(frame, ctx, projection, m.effects().Active[0].Starts.Add(elapsed), false)
		count := reflect.ValueOf(frame).Elem().FieldByName("worldBillboards").Len()
		if count < 9 || len(m.mapScene.TextureMiss) != 0 {
			t.Fatalf("Monk effects at %s: %d billboards, missing=%v", elapsed, count, m.mapScene.TextureMiss)
		}
	}
}

func TestSteelBodyAuraPacketOrderAndLifetime(t *testing.T) {
	for _, statusFirst := range []bool{true, false} {
		ctx := testutil.MonkEffectContext()
		m := &WorldMode{}
		status := func(active bool) {
			m.applyStatusEffectChange(ctx, network.StatusEffectChange{StatusID: db.StatusSteelbody, ActorID: 300, Active: active})
		}
		if statusFirst {
			status(true)
		}
		m.applySkillNoDamageNotify(ctx, network.SkillNoDamageNotify{SourceID: 300, TargetID: 300, SkillID: db.SkillMOSteelbody, Result: 1})
		if !statusFirst {
			status(true)
		}
		status(true)
		if len(m.scheduledSounds) != 1 {
			t.Fatalf("statusFirst=%t: duplicate activation sounds: %+v", statusFirst, m.scheduledSounds)
		}
		auras := 0
		for _, effect := range m.effects().Active {
			if effect.EffectID == effects.SteelBody {
				auras++
				if effect.Expired(time.Now().Add(time.Hour)) || !effect.Persistent {
					t.Fatal("Steel Body aura does not follow status lifetime")
				}
			}
		}
		if auras != 1 || !m.effects().HasActive(db.SkillEffectSteelBodyRing, 300, time.Now()) {
			t.Fatalf("Steel Body effects = %+v", m.effects().Active)
		}
		status(false)
		if m.effects().HasActive(effects.SteelBody, 300, time.Now()) {
			t.Fatal("Steel Body aura survived status removal")
		}
	}
}

func TestSteelBodyDoesNotReviveExpiredActivation(t *testing.T) {
	ctx := testutil.MonkEffectContext()
	m := &WorldMode{}
	m.effects().AddAt(ctx, effects.SteelBody, 2000000, time.Now().Add(-time.Minute))
	m.applyStatusEffectChange(ctx, network.StatusEffectChange{StatusID: db.StatusSteelbody, ActorID: 150000, Active: true})
	m.applySkillNoDamageNotify(ctx, network.SkillNoDamageNotify{SourceID: 150000, TargetID: 150000, SkillID: db.SkillMOSteelbody, Result: 1})
	active := 0
	for _, effect := range m.effects().Active {
		if effect.EffectID == effects.SteelBody && !effect.Expired(time.Now()) {
			active++
			if effect.ActorID != 2000000 || !effect.Persistent {
				t.Fatal("local aliases duplicated the aura")
			}
		}
	}
	if active != 1 {
		t.Fatalf("active Steel Body auras = %d, want 1", active)
	}
}

func TestMonkTwoFrameSkillsKeepTheirOriginalTiming(t *testing.T) {
	for _, skillID := range []uint16{db.SkillMOInvestigate, db.SkillMOFingeroffensive} {
		for _, sourceSpeed := range []int32{80, 2000} {
			ctx := testutil.MonkEffectContext()
			m := &WorldMode{}
			m.applyActorActionNotify(ctx, network.ActorActionNotify{
				SourceID: 2000000, TargetID: 300, SkillID: skillID, SourceSpeed: sourceSpeed,
				Action: network.ActorActionSkill, HitCount: 1, Damage: 100,
			})
			started := m.actorAnims[150000].started
			action := res.ACTAction{DelayMS: 100, Animations: make([]res.ACTAnimation, 8)}
			for _, tc := range []struct {
				elapsed time.Duration
				frame   int
			}{
				{0, 3}, {150 * time.Millisecond, 3}, {170 * time.Millisecond, 4}, {1099 * time.Millisecond, 4},
			} {
				anim, active := m.actorAnimation(150000, started.Add(tc.elapsed))
				if !active {
					t.Fatalf("skill %d ended at %s with source speed %d", skillID, tc.elapsed, sourceSpeed)
				}
				state := sprites.State{}
				applyActorAnimationToSpriteState(&state, anim, true)
				if frame := sprites.BodyMotionForState(action, state, started, started.Add(tc.elapsed)); frame != tc.frame {
					t.Fatalf("skill %d at %s: frame=%d, want %d", skillID, tc.elapsed, frame, tc.frame)
				}
			}
			if _, active := m.actorAnimation(150000, started.Add(1100*time.Millisecond)); active {
				t.Fatalf("skill %d did not release its pose after 1100ms", skillID)
			}
		}
	}
}

func TestBladeStopFreezesAndReleasesBothActors(t *testing.T) {
	ctx := testutil.MonkEffectContext()
	m := &WorldMode{}
	now := time.Now()
	m.setActorAction(ctx, 300, actorAnimation{actionFamily: sprites.ActionPCAttack1, started: now, fixedMotion: 2, hasFixedMotion: true})
	m.applyActorBladeStop(ctx, network.ActorBladeStop{SourceID: 2000000, TargetID: 300, Active: true})
	if anim := m.actorAnims[150000]; anim.actionFamily != sprites.ActionPCSkill || anim.fixedMotion != 4 || anim.play {
		t.Fatalf("caster pose = %+v", anim)
	}
	if anim := m.actorAnims[300]; anim.actionFamily != sprites.ActionPCAttack1 || anim.play || !anim.hasFixedMotion || anim.fixedMotion != 2 {
		t.Fatalf("victim was put in another animation: %+v", anim)
	}
	for _, id := range []uint32{2000000, 300} {
		actor, _, _ := scene.ActorForCombatID(ctx, id)
		if tint := actorStateTint(actor); tint != (color.RGBA{R: 64, G: 64, B: 64, A: 255}) {
			t.Fatalf("Blade Stop tint = %v", tint)
		}
		if anim, ok := m.actorAnimation(id, time.Now().Add(time.Minute)); !ok || !anim.hasFixedMotion {
			t.Fatalf("actor %d resumed before Blade Stop release", id)
		}
	}
	m.applyActorBladeStop(ctx, network.ActorBladeStop{SourceID: 2000000, TargetID: 300})
	for _, id := range []uint32{2000000, 150000, 300} {
		actor, _, _ := scene.ActorForCombatID(ctx, id)
		if actor.Opt3State&db.Opt3Bladestop != 0 || !m.actorAnims[id].play || m.actorAnims[id].hasFixedMotion {
			t.Fatalf("actor %d still stopped after release", id)
		}
	}
}

func TestNinjaPosesAndMirrorImage(t *testing.T) {
	for _, skill := range []uint16{523, 527, 528, 534, 537, 540, 543} {
		spec := skillAction(skill)
		first, last, duration := 0, 1, 500*time.Millisecond
		if skill == 523 || skill == 534 || skill == 540 || skill == 543 {
			first, last, duration = 2, 5, time.Second
		}
		total := time.Duration(0)
		for frame := first; frame <= last; frame++ {
			if spec.action != skillActorActionSkill || !spec.hasFrame || spec.frame != frame || spec.play {
				t.Fatalf("skill %d wrong ACT pose %+v", skill, spec)
			}
			total += spec.duration
			if frame < last {
				if spec.next == nil {
					t.Fatal("missing pose")
				}
				spec = *spec.next
			}
		}
		if total != duration || spec.next != nil {
			t.Fatalf("skill %d wrong animation lifetime", skill)
		}
	}
	ctx := testutil.ArcherEffectContext()
	m := &WorldMode{}
	m.applyStatusEffectChange(ctx, network.StatusEffectChange{ActorID: 2000000, StatusID: db.StatusNjBunsinjyutsu, Active: true})
	entry := scene.ActorDrawEntry{Actor: ctx.World.Player, IsPlayer: true, WorldX: 10.5, WorldY: 20.5}
	entry.Actor.BodyPal = 1 // Mirror Image also works on dyed bodies.
	tint := actorStateTint(entry.Actor)
	if tint != (color.RGBA{155, 155, 255, 255}) {
		t.Fatalf("mirror tint=%v", tint)
	}
	billboard := &scene.Billboard{Image: render.NewImage(10, 20)}
	projection := testutil.Projection(800, 600, 10, 20, 0)
	start := time.Unix(100, 0)
	for tick := 0; tick <= 20; tick++ {
		m.effects().DrawActorBlurs(render.NewFrame(800, 600), projection, entry, billboard, 1, 1, 1, tint, start.Add(time.Duration(tick)*db.EffectFrameDuration))
	}
	if len(m.effects().ActorBlurs[100]) != 2 {
		t.Fatal("Mirror Image must emit every twenty ticks")
	}
	m.applyStatusEffectChange(ctx, network.StatusEffectChange{ActorID: 2000000, StatusID: db.StatusNjBunsinjyutsu})
	entry.Actor = ctx.World.Player
	m.effects().DrawActorBlurs(render.NewFrame(800, 600), projection, entry, billboard, 1, 1, 1, tint, start.Add(44*db.EffectFrameDuration))
	if len(m.effects().ActorBlurs) != 0 {
		t.Fatal("Mirror Image survived status removal")
	}
}

func TestOriginalBodyColourWindowsAndRestoration(t *testing.T) {
	starts := time.Unix(100, 0)
	base := color.RGBA{R: 255, G: 255, B: 255, A: 173}
	for _, id := range []int{effects.MagicCrasher, effects.TransBlueBody} {
		spec, _ := effects.Lookup(id)
		mode := WorldMode{effectSystem: effects.System{Active: []effects.Instance{{EffectID: id, ActorID: 42, Starts: starts, Expires: starts.Add(spec.Duration)}}}}
		for _, frame := range []int{-1, 0, 29, 30, 60, 61, 199, 200} {
			now := starts.Add(time.Duration(frame) * db.EffectFrameDuration)
			got := mode.actorBodyColorTint(42, base, now)
			if got.A != base.A {
				t.Fatal("body effect changed transparency")
			}
			wantBlend := render.BlendSourceOver
			if got := mode.actorRenderBlend(42, now); got != wantBlend {
				t.Fatalf("effect %d frame %d: blend %v, want %v", id, frame, got, wantBlend)
			}
			if id == effects.MagicCrasher {
				active := frame >= 30 && frame <= 60
				if (got != base) != active {
					t.Fatalf("Magic Crasher at frame %d: tint %+v", frame, got)
				}
				if again := mode.actorBodyColorTint(42, base, now); again != got {
					t.Fatal("flash changes within a tick")
				}
			} else {
				want := base
				if frame >= 0 && frame < 200 {
					want.R, want.G = uint8(205-frame), uint8(205-frame)
				}
				if got != want {
					t.Fatalf("blue transition at frame %d: %+v, want %+v", frame, got, want)
				}
			}
			if mode.actorBodyColorTint(99, base, now) != base {
				t.Fatal("effect tinted another actor")
			}
		}
		// A second cast replaces the first colour instead of multiplying two tints.
		second := mode.effects().Active[0]
		second.Starts = starts.Add(10 * db.EffectFrameDuration)
		second.Expires = second.Starts.Add(spec.Duration)
		mode.effects().Active = append(mode.effects().Active, second)
		onlySecond := WorldMode{effectSystem: effects.System{Active: []effects.Instance{second}}}
		now := starts.Add(40 * db.EffectFrameDuration)
		if mode.actorBodyColorTint(42, base, now) != onlySecond.actorBodyColorTint(42, base, now) {
			t.Fatal("recasting compounds body colour")
		}
	}
}

func TestPaladinNativeTiming(t *testing.T) {
	ctx := testutil.KnightEffectContext()
	start := time.Unix(100, 0)
	m := &WorldMode{}
	m.effects().AddBetween(ctx, effects.ShieldBoomer, 300, 2000000, start)
	flight := m.effects().Active[0].Shield
	origin := flight.Images[0].Position
	flight.Advance(10, origin)
	if math.Abs(flight.Images[0].Position.X-origin.X-4) > 1e-6 || flight.Phase != 0 {
		t.Fatal("shield must travel two native units per frame")
	}
	before := *flight
	flight.Advance(10, origin)
	if *flight != before {
		t.Fatal("drawing twice advanced the shield twice")
	}
	flight.Advance(499, origin)
	if flight.Phase != 2 || flight.Images[0].Alpha != 0 {
		t.Fatal("shield did not turn back and fade")
	}
	effect := effects.Instance{ActorOrigin: scene.Point3{Y: 2}, Starts: start}
	if math.Abs(effects.PressureImage(effect, 32).Position.Y-5.8) > 1e-6 || effects.PressureImage(effect, 72).Alpha != 0 {
		t.Fatal("Pressure must land at frame 32 and fade over 40 frames")
	}
	pressure, _ := effects.Lookup(effects.Pressure)
	sacrifice, _ := effects.Lookup(effects.Bash3D)
	if pressure.SfxDelays[0] != 3*db.EffectFrameDuration || pressure.CameraShakeDelay != 32*db.EffectFrameDuration || sacrifice.SfxDelays[0] != 22*db.EffectFrameDuration {
		t.Fatal("native audio or impact timing changed")
	}
	m.effects().Active = []effects.Instance{{EffectID: effects.Shrink, ActorID: 300, Starts: start}}
	base := color.RGBA{255, 255, 255, 255}
	if m.actorBodyColorTint(300, base, start.Add(6*db.EffectFrameDuration)) != (color.RGBA{250, 250, 100, 255}) ||
		m.actorBodyColorTint(300, base, start.Add(7*db.EffectFrameDuration)) != base ||
		m.actorBodyColorTint(300, base, start.Add(26*db.EffectFrameDuration)) != base {
		t.Fatal("Shrink must blink only during frames 5..25")
	}
}

func TestAssumptioFollowsStatusInEitherPacketOrder(t *testing.T) {
	for _, statusFirst := range []bool{false, true} {
		ctx := testutil.MonkEffectContext()
		m := &WorldMode{}
		status := network.StatusEffectChange{StatusID: db.StatusAssumptio, ActorID: 300, Active: true}
		if statusFirst {
			m.applyStatusEffectChange(ctx, status)
		}
		m.applySkillNoDamageNotify(ctx, network.SkillNoDamageNotify{SkillID: db.SkillHPAssumptio, SourceID: 2000000, TargetID: 300, Result: 1})
		m.applyStatusEffectChange(ctx, status)
		m.applyStatusEffectChange(ctx, status)
		if len(m.effects().Active) != 2 || len(m.scheduledSounds) != 1 || !m.effects().HasActive(effects.Assumptio, 300, time.Now().Add(time.Hour)) {
			t.Fatal("Assumptio must have one persistent glow and one activation")
		}
		next := &WorldMode{}
		next.syncCurrentActorEffectStateEffects(ctx)
		if len(next.effects().Active) != 1 || next.effects().Active[0].EffectID != effects.Assumptio || len(next.scheduledSounds) != 0 {
			t.Fatal("actor entry should restore the glow without replaying activation")
		}
		status.Active = false
		m.applyStatusEffectChange(ctx, status)
		if m.effects().HasActive(effects.Assumptio, 300, time.Now()) {
			t.Fatal("Assumptio glow survived status removal")
		}
	}
}

func TestCloseConfineStatusOwnsBothRings(t *testing.T) {
	ctx := testutil.RogueSkillContext()
	m := &WorldMode{}
	for _, status := range []network.StatusEffectChange{
		{StatusID: db.StatusRgCconfineM, ActorID: 2000000, Active: true},
		{StatusID: db.StatusRgCconfineS, ActorID: 300, Active: true},
	} {
		m.applyStatusEffectChange(ctx, status)
		m.applyStatusEffectChange(ctx, status)
	}
	if len(m.effects().Active) != 2 || len(m.scheduledSounds) != 0 {
		t.Fatalf("expected one silent ring per participant: %+v, %+v", m.effects().Active, m.scheduledSounds)
	}
	for _, effect := range m.effects().Active {
		if effect.EffectID != effects.NPCStop2 || effect.Expired(time.Now().Add(time.Hour)) {
			t.Fatalf("ring must stay until status removal: %+v", effect)
		}
	}
	m.applyStatusEffectChange(ctx, network.StatusEffectChange{StatusID: db.StatusRgCconfineM, ActorID: 100})
	if len(m.effects().Active) != 1 || m.effects().Active[0].ActorID != 300 {
		t.Fatalf("local character/account aliases did not remove the same ring: %+v", m.effects().Active)
	}
	m.applySpecialEffectNotify(ctx, network.SpecialEffectNotify{AID: 300, EffectID: uint32(effects.NPCStop2Del)})
	if len(m.effects().Active) != 0 {
		t.Fatal("explicit ring deletion did not remove it")
	}
	status := network.StatusEffectChange{StatusID: db.StatusRgCconfineS, ActorID: 300, Active: true, HasDuration: true, Duration: time.Second}
	m.applyStatusEffectChange(ctx, status)
	m.effects().Active[0].Expires = time.Now().Add(-time.Second)
	m.applyStatusEffectChange(ctx, status)
	active := 0
	for _, effect := range m.effects().Active {
		if !effect.Expired(time.Now()) {
			active++
			if effect.Expired(time.Now().Add(500*time.Millisecond)) || !effect.Expired(time.Now().Add(2*time.Second)) {
				t.Fatal("ring lost the status deadline")
			}
		}
	}
	if active != 1 {
		t.Fatal("refresh revived an expired ring alongside its replacement")
	}
}

func TestCloseConfineBodyAndRingRendering(t *testing.T) {
	ctx := testutil.RogueSkillContext()
	starts := time.Unix(100, 0)
	m := &WorldMode{effectSystem: effects.System{Active: []effects.Instance{{EffectID: effects.QuakeBody4, ActorID: 300, Starts: starts, Expires: starts.Add(60 * db.EffectFrameDuration)}}}}
	billboard := &scene.Billboard{Image: render.NewImage(64, 64), AnchorX: 32, AnchorY: 60}
	projection := testutil.Projection(800, 600, 16.5, 20.5, 0)
	entry := scene.ActorDrawEntry{Actor: ctx.World.Actors[300], WorldX: 16.5, WorldY: 20.5}
	for _, tick := range []int{0, 20, 21, 22, 59, 60, 61} {
		frame := render.NewFrame(800, 600)
		m.drawActorSpriteWithEffects3D(ctx, frame, projection, entry, billboard, 1, 1, 1, color.RGBA{255, 255, 255, 255}, render.BlendSourceOver, starts.Add(time.Duration(tick)*db.EffectFrameDuration))
		draw := reflect.ValueOf(frame).Elem().FieldByName("worldBillboards").Index(0)
		red := draw.FieldByName("ColorR").Float() > draw.FieldByName("ColorG").Float()
		if red != (tick == 22) {
			t.Fatalf("wrong Close Confine tint at tick %d", tick)
		}
	}
	if billboard.AnchorX != 32 || billboard.AnchorY != 60 || ctx.World.Actors[300].X != 16 {
		t.Fatal("Close Confine modified cached geometry or actor position")
	}
	m = &WorldMode{effectSystem: effects.System{EffectViews: map[string]*scene.SpriteView{"cconfine": testutil.EffectRenderTestSprite()}}}
	m.applyStatusEffectChange(ctx, network.StatusEffectChange{StatusID: db.StatusRgCconfineS, ActorID: 300, Active: true})
	frame := render.NewFrame(800, 600)
	m.effects().Draw(frame, ctx, projection, time.Now().Add(time.Minute), false)
	draws := reflect.ValueOf(frame).Elem().FieldByName("worldBillboards")
	if draws.Len() != 1 {
		t.Fatal("ring did not keep its final sprite frame")
	}
	draw := draws.Index(0)
	if math.Abs(draw.FieldByName("ColorA").Float()-100.0/255) > .001 || draw.FieldByName("Options").FieldByName("DepthTest").Bool() {
		t.Fatal("Close Confine ring must use native translucency and overlay rendering")
	}
}

func TestSniperBodyFlashAndRotation(t *testing.T) {
	ctx := testutil.ArcherEffectContext()
	start := time.Unix(100, 0)
	for _, tc := range []struct {
		id, frame int
		flash     bool
	}{
		{effects.FalconAssault, 29, false}, {effects.FalconAssault, 30, true},
		{effects.FalconAssault, 35, false}, {effects.FalconAssault, 40, true},
		{effects.FalconAssault, 50, true}, {effects.FalconAssault, 51, false},
		{effects.Portal4, 4, false}, {effects.Portal4, 5, true},
		{effects.Portal4, 25, true}, {effects.Portal4, 26, false},
	} {
		m := &WorldMode{}
		m.effects().AddAt(ctx, tc.id, 300, start)
		now := start.Add(time.Duration(tc.frame) * db.EffectFrameDuration)
		if (m.actorRenderBlend(300, now) == render.BlendLighter) != tc.flash {
			t.Fatalf("effect %d frame %d: incorrect body blend", tc.id, tc.frame)
		}
		if tc.id == effects.Portal4 && tc.flash && m.actorBodyColorTint(300, color.RGBA{255, 255, 255, 255}, now) != (color.RGBA{220, 250, 220, 255}) {
			t.Fatal("Wind Walk's body flash must be pale green")
		}
	}
	m := &WorldMode{}
	m.effects().AddAt(ctx, effects.FalconAssault, 2000000, start)
	for _, tc := range []struct{ frame, direction int }{{29, 0}, {30, 1}, {31, 1}, {32, 2}, {41, 0}, {54, 1}, {55, 0}} {
		if d := m.skillEffectDirection(ctx, 100, 0, start.Add(time.Duration(tc.frame)*db.EffectFrameDuration)); d != tc.direction {
			t.Fatalf("frame %d: direction %d, want %d (including local account/character alias)", tc.frame, d, tc.direction)
		}
	}
}

func TestSoulLinkerPlayerGlowsAndKaiteStatus(t *testing.T) {
	ctx := testutil.SoulLinkerEffectContext()
	starts := time.Unix(100, 0)
	m := &WorldMode{}
	billboard := &scene.Billboard{Image: render.NewImage(64, 64), AnchorX: 32, AnchorY: 60}
	projection := testutil.Projection(800, 600, 10.5, 20.5, 0)
	draw := func(now time.Time) reflect.Value {
		frame := render.NewFrame(800, 600)
		entry := scene.ActorDrawEntry{Actor: ctx.World.Player, WorldX: 10.5, WorldY: 20.5, Scale: 1, Shadow: 1}
		m.drawActorSpriteWithEffects3D(ctx, frame, projection, entry, billboard, .5, .6, .8, color.RGBA{100, 120, 200, 200}, render.BlendSourceOver, now)
		commands := reflect.ValueOf(frame).Elem().FieldByName("worldBillboards")
		depthWriters := 0
		for i := 0; i < commands.Len(); i++ {
			if commands.Index(i).FieldByName("Options").FieldByName("DepthWrite").Bool() {
				depthWriters++
			}
		}
		if depthWriters != 1 {
			t.Fatalf("only the body should occlude translucent geometry, got %d silhouettes", depthWriters)
		}
		return commands
	}
	base := draw(starts).Index(0)
	for _, id := range []int{effects.Hated, effects.Kaizel, effects.BlueBody, effects.ReflectBody} {
		// The player's sprite uses CharID, while the effect arrives with AccountID.
		m.effects().Active = []effects.Instance{{EffectID: id, ActorID: ctx.Session.AccountID, Starts: starts}}
		commands := draw(starts.Add(40 * db.EffectFrameDuration))
		if commands.Len() != 3 {
			t.Fatalf("effect %d: want body and two blue passes, got %d", id, commands.Len())
		}
		for _, field := range []string{"ColorR", "ColorG", "ColorB", "ColorA", "AnchorX", "AnchorY"} {
			if base.FieldByName(field).Float() != commands.Index(0).FieldByName(field).Float() {
				t.Fatalf("effect %d changed player's %s", id, field)
			}
		}
		if draw(starts.Add(200*db.EffectFrameDuration)).Len() != 1 {
			t.Fatalf("effect %d glow did not end", id)
		}
	}
	m.effects().Active = nil
	m.applyStatusEffectChange(ctx, network.StatusEffectChange{StatusID: db.StatusKaite, ActorID: ctx.Session.AccountID, Active: true})
	if draw(starts).Len() != 5 {
		t.Fatal("Kaite needs four silhouettes and the player body")
	}
	m.applyStatusEffectChange(ctx, network.StatusEffectChange{StatusID: db.StatusKaite, ActorID: ctx.Session.AccountID, Active: false})
	if draw(starts).Len() != 1 {
		t.Fatal("Kaite silhouettes survived the server clearing the status")
	}
}

func TestEskeFollowsMonsterOptionAndEndsOnClear(t *testing.T) {
	ctx := testutil.SoulLinkerEffectContext()
	m := &WorldMode{}
	m.handleNetworkPacket(ctx, testutil.SoulLinkerSkillPacket(db.SkillSLSke, 1), time.Now())
	actor := ctx.World.Actors[300]
	if actor.Opt3State&db.Opt3Energycoat == 0 {
		t.Fatal("Eske did not set the monster's native option bit")
	}
	if tint := actorStateTint(actor); tint != (color.RGBA{255, 255, 255, 255}) {
		t.Fatalf("Eske should not use the player's Energy Coat tint: %v", tint)
	}
	starts := m.effects().Active[0].Starts
	projection := testutil.Projection(800, 600, 18.5, 20.5, 0)
	billboard := &scene.Billboard{Image: render.NewImage(64, 64), AnchorX: 32, AnchorY: 60}
	draw := func(at time.Time) int {
		frame := render.NewFrame(800, 600)
		entry := scene.ActorDrawEntry{Actor: ctx.World.Actors[300], WorldX: 18.5, WorldY: 20.5}
		m.drawActorSpriteWithEffects3D(ctx, frame, projection, entry, billboard, 1, 1, 1, color.RGBA{255, 255, 255, 255}, render.BlendSourceOver, at)
		return reflect.ValueOf(frame).Elem().FieldByName("worldBillboards").Len()
	}
	if draw(starts.Add(time.Second)) != 11 {
		t.Fatal("Eske burst needs ten expanding outlines")
	}
	if draw(starts.Add(time.Minute)) != 5 {
		t.Fatal("Eske must retain four faint outlines after the burst")
	}
	m.handleNetworkPacket(ctx, testutil.NpcEffectUpdatePacket(300, 0), time.Now())
	if len(m.effects().Active) != 0 || draw(starts.Add(time.Second)) != 1 {
		t.Fatal("Eske persisted after its server status ended")
	}
	// The same option bit on a player still means Energy Coat, not Eske.
	ctx.World.Player.Opt3State = db.Opt3Energycoat
	if actorStateTint(ctx.World.Player) == (color.RGBA{255, 255, 255, 255}) {
		t.Fatal("monster Eske changed player Energy Coat tint")
	}
}

func TestEsmaImpactLayersAndReadyNotification(t *testing.T) {
	ctx := testutil.SoulLinkerEffectContext()
	m := &WorldMode{}
	soulLinkerTestTextures(m)
	starts := time.Unix(100, 0)
	projection := testutil.Projection(800, 600, 18.5, 20.5, 0)
	m.effects().AddAt(ctx, effects.HitLine6, 300, starts)
	m.effects().AddAt(ctx, effects.HitTexture, 300, starts)
	for _, tick := range []int{25, 50, 120} {
		frame := render.NewFrame(800, 600)
		m.effects().Draw(frame, ctx, projection, starts.Add(time.Duration(tick)*db.EffectFrameDuration), false)
		mesh := reflect.ValueOf(frame).Elem().FieldByName("worldCommands")
		quads := reflect.ValueOf(frame).Elem().FieldByName("worldBillboards")
		if (mesh.Len() > 0) != (tick == 50) || (quads.Len() > 0) != (tick == 50) {
			t.Fatalf("wrong impact timing at %d: streaks=%d, flash=%d", tick, mesh.Len(), quads.Len())
		}
	}
	m.effects().Active = nil
	m.applyStatusEffectChange(ctx, network.StatusEffectChange{StatusID: db.StatusSmaReady, ActorID: ctx.Session.AccountID, Active: true})
	if len(m.effects().Active) != 1 || m.effects().Active[0].EffectID != effects.SmaReady || len(m.scheduledSounds) != 0 {
		t.Fatal("Esma readiness must launch its silent particle effect")
	}
	m.applyStatusEffectChange(ctx, network.StatusEffectChange{StatusID: db.StatusSmaReady, ActorID: ctx.Session.AccountID, Active: false})
	if len(m.effects().Active) != 1 {
		t.Fatal("consuming readiness must neither replay nor cut off the native one-shot sparkle")
	}
}

func TestSoulLinkAuraFollowsStatusAndKeepsPlayerColors(t *testing.T) {
	for _, statusFirst := range []bool{false, true} {
		ctx := testutil.SoulLinkerEffectContext()
		m := &WorldMode{}
		status := network.StatusEffectChange{StatusID: db.StatusSoullink, ActorID: 2000000, Active: true}
		if statusFirst {
			m.applyStatusEffectChange(ctx, status)
		}
		m.applySkillNoDamageNotify(ctx, network.SkillNoDamageNotify{SkillID: db.SkillSLMonk, SourceID: 2000000, TargetID: 2000000, Result: 1})
		if !statusFirst {
			m.applyStatusEffectChange(ctx, status)
		}
		count := 0
		var start time.Time
		for _, e := range m.effects().Active {
			if e.EffectID == effects.AsuraBody {
				count++
				start = e.Starts
				if !e.Persistent || !e.Expires.IsZero() {
					t.Fatal("aura must last until status removal")
				}
			}
		}
		if count != 1 {
			t.Fatalf("statusFirst=%v: %d auras", statusFirst, count)
		}
		if actorStateTint(ctx.World.Player) != (color.RGBA{255, 255, 255, 255}) {
			t.Fatal("Soul Link must preserve base sprite color")
		}
		frame := render.NewFrame(800, 600)
		billboard := &scene.Billboard{Image: render.NewImage(64, 64), AnchorX: 32, AnchorY: 60}
		entry := scene.ActorDrawEntry{Actor: ctx.World.Player, WorldX: 10.5, WorldY: 20.5}
		m.drawActorSpriteWithEffects3D(ctx, frame, testutil.Projection(800, 600, 10.5, 20.5, 0), entry, billboard, 1, 1, 1, color.RGBA{255, 255, 255, 255}, render.BlendSourceOver, start.Add(time.Minute))
		commands := reflect.ValueOf(frame).Elem().FieldByName("worldBillboards")
		if commands.Len() != 3 {
			t.Fatalf("want body and two silhouettes, got %d", commands.Len())
		}
		if got := commands.Index(1).FieldByName("ColorA").Float(); math.Abs(got-200.0/255) > 1e-6 {
			t.Fatalf("wrong aura alpha: %v", got)
		}
		status.Active = false
		m.applyStatusEffectChange(ctx, status)
		if m.effects().HasActive(effects.AsuraBody, 150000, time.Now()) {
			t.Fatal("aura survived status removal")
		}
	}
}

func TestEswooShrinksUntilServerClearsIt(t *testing.T) {
	for _, snapshotFirst := range []bool{false, true} {
		ctx := testutil.SoulLinkerEffectContext()
		m := &WorldMode{}
		if snapshotFirst {
			m.handleNetworkPacket(ctx, testutil.NpcEffectUpdatePacket(300, db.Opt3Overthrust), time.Now())
		}
		// rAthena does not send the SWOO status icon to monsters.
		m.handleNetworkPacket(ctx, testutil.SoulLinkerSkillPacket(db.SkillSLSwoo, 1), time.Now())
		count := 0
		for _, e := range m.effects().Active {
			if e.EffectID == effects.BabyBody {
				count++
				for _, tc := range []struct {
					age  time.Duration
					size float64
				}{{0, 1}, {45 * db.EffectFrameDuration, .5}, {time.Minute, .5}} {
					if got := m.actorBodySizeMultiplier(300, e.Starts.Add(tc.age)); math.Abs(got-tc.size) > 1e-8 {
						t.Fatalf("Eswoo size at %v = %v", tc.age, got)
					}
				}
			}
		}
		if count != 1 {
			t.Fatalf("snapshotFirst=%v: %d shrink effects", snapshotFirst, count)
		}
		if tint := actorStateTint(ctx.World.Actors[300]); tint != (color.RGBA{255, 255, 255, 255}) {
			t.Fatalf("monster got Overthrust tint: %v", tint)
		}
		m.handleNetworkPacket(ctx, testutil.NpcEffectUpdatePacket(300, 0), time.Now())
		for _, e := range m.effects().Active {
			if e.EffectID == effects.BabyBodyBack {
				if got := m.actorBodySizeMultiplier(300, e.Starts); got != .5 {
					t.Fatalf("return starts at %v", got)
				}
				if got := m.actorBodySizeMultiplier(300, e.Starts.Add(15*db.EffectFrameDuration)); got != 1 {
					t.Fatalf("return ends at %v", got)
				}
			}
		}
		if !m.effects().HasActive(effects.BabyBodyBack, 300, time.Now()) {
			t.Fatal("missing return animation")
		}
		m.removeActorEffectStateEffects(300)
		if m.effects().HasActive(effects.BabyBodyBack, 300, time.Now()) {
			t.Fatal("size effect survived actor leaving")
		}
	}
}

func TestMonsterEffectUpdatesPreserveActorDuringLoginAndPlay(t *testing.T) {
	for _, login := range []bool{false, true} {
		ctx := testutil.SoulLinkerEffectContext()
		actor := ctx.World.Actors[300]
		actor.BodyState, actor.HealthState = db.BodyStateStun, db.HealthStatePoison
		actor.HasState = true
		actor.Moving, actor.MoveStarted = true, time.Now().Add(-time.Second)
		ctx.World.Actors[300] = actor
		m := &WorldMode{}
		packet := testutil.NpcEffectUpdatePacket(300, db.Opt3Overthrust|db.Opt3Energycoat|db.Opt3Steelbody)
		if login {
			if handled, err := applyLoginActorBootstrapPacket(ctx, packet); !handled || err != nil {
				t.Fatal("login dropped the monster effect snapshot")
			}
			m.syncCurrentActorEffectStateEffects(ctx)
		} else {
			m.handleNetworkPacket(ctx, packet, time.Now())
		}
		actor.Opt3State = db.Opt3Overthrust | db.Opt3Energycoat | db.Opt3Steelbody
		actor.Level, actor.HasLevel = 75, true
		if !reflect.DeepEqual(ctx.World.Actors[300], actor) {
			t.Fatalf("partial snapshot changed other actor data: %+v", ctx.World.Actors[300])
		}
		for _, id := range []int{effects.BabyBody, effects.SteelBody} {
			if !m.effects().HasActive(id, 300, time.Now().Add(time.Hour)) {
				t.Fatalf("missing persistent effect %d", id)
			}
		}
		m.handleNetworkPacket(ctx, testutil.NpcEffectUpdatePacket(300, 0), time.Now())
		if ctx.World.Actors[300].Opt3State != 0 || m.effects().HasActive(effects.SteelBody, 300, time.Now()) || m.effects().HasActive(effects.BabyBody, 300, time.Now()) {
			t.Fatal("monster effects survived the authoritative clear")
		}
		m.handleNetworkPacket(ctx, testutil.NpcEffectUpdatePacket(999, db.Opt3Overthrust), time.Now())
		if _, ok := ctx.World.Actors[999]; ok {
			t.Fatal("effect update created an unknown actor")
		}
	}
}

func TestStarGladiatorWarmthStatusAndSkillPackets(t *testing.T) {
	for _, status := range []uint16{db.StatusSgWarm, db.StatusSgSunWarm, db.StatusSgMoonWarm, db.StatusSgStarWarm} {
		ctx := testutil.ArcherEffectContext()
		ctx.World.Player.Job = db.JobStar
		m := &WorldMode{}
		m.applyStatusEffectChange(ctx, network.StatusEffectChange{StatusID: status, Active: true})
		if len(m.effects().Active) != 2 || len(m.scheduledSounds) != 0 {
			t.Fatal("Warmth status needs persistent lightning and body glow, without recasting particles or sound")
		}
		for _, e := range m.effects().Active {
			if !e.Persistent || !e.Expires.IsZero() || e.ActorID != 2000000 {
				t.Fatalf("Warmth status effect lifetime or local alias: %+v", e)
			}
		}
		m.applySkillNoDamageNotify(ctx, network.SkillNoDamageNotify{SkillID: 428, SourceID: 100, TargetID: 100, Result: 1})
		if len(m.effects().Active) != 3 || m.effects().Active[2].EffectID != effects.Hated2 || m.effects().Active[2].BuffParticles == nil || len(m.scheduledSounds) != 1 || m.scheduledSounds[0].at.Sub(m.effects().Active[2].Starts) != 10*db.EffectFrameDuration {
			t.Fatal("Warmth release must restart its state effects, add red particles, and delay the sound ten ticks")
		}
		anim := m.actorAnims[100]
		if anim.actionFamily != sprites.ActionPCSkill || anim.fixedMotion != 3 || !anim.hasFixedMotion || anim.duration != time.Second {
			t.Fatalf("Warmth release pose: %+v", anim)
		}
		m.applyStatusEffectChange(ctx, network.StatusEffectChange{StatusID: status, Active: false})
		if len(m.effects().Active) != 1 || m.effects().Active[0].EffectID != effects.Hated2 {
			t.Fatal("status removal must stop lightning and red glow while letting release particles finish")
		}
	}
}

func TestFusionPresentationDoesNotAdvanceUnrelatedAnimations(t *testing.T) {
	for _, landing := range []bool{false, true} {
		name := "ordinary actor"
		if landing {
			name = "landing actor"
		}
		t.Run(name, func(t *testing.T) {
			m := &WorldMode{}
			now := time.Unix(100, 0)
			actor := worldstate.Actor{ID: 300}
			if landing {
				actor.Job, actor.EffectState = db.JobStar, db.EffectStateFlying
				m.effects().FusionHeight(actor, now.Add(-300*db.EffectFrameDuration), false)
				m.effects().FusionHeight(actor, now.Add(-db.EffectFrameDuration), false)
				actor.EffectState = 0
			}
			anim := actorAnimation{
				actionFamily: sprites.ActionPCSkill,
				started:      now.Add(-2 * time.Second),
				duration:     time.Second,
				next:         &actorAnimation{actionFamily: scene.SpriteActionIdle, loop: true},
			}
			m.actorAnims = map[uint32]actorAnimation{actor.ID: anim}
			m.starFusionHeight(actor, now)
			if got := m.actorAnims[actor.ID]; !reflect.DeepEqual(got, anim) {
				t.Fatal("Fusion presentation advanced a non-flying actor's animation queue")
			}
		})
	}
}

func TestStarDevilReconcilesLevelStatusAndMapEntry(t *testing.T) {
	ctx := testutil.ArcherEffectContext()
	m := &WorldMode{}
	m.applyStatusEffectChange(ctx, network.StatusEffectChange{StatusID: db.StatusDevil1, Active: true})
	if len(m.effects().Active) != 0 {
		t.Fatal("Demon must wait for its learned level")
	}
	ctx.Session.Skills.List = []session.Skill{{ID: db.SkillSGDevil, Level: 10}}
	m.syncStarDevil(ctx)
	m.syncStarDevil(ctx)
	if len(m.effects().Active) != 2 || m.effects().Active[0].EffectID != 484 || !m.effects().Active[0].Persistent || len(m.scheduledSounds) != 1 {
		t.Fatal("Demon level ten must install a single persistent mask and introduction")
	}
	next := &WorldMode{}
	next.syncCurrentActorEffectStateEffects(ctx)
	if len(next.effects().Active) != 2 || next.effects().Active[0].EffectID != 484 {
		t.Fatal("map entry did not restore Demon")
	}
	ctx.Session.Skills.List[0].Level = 1
	m.syncStarDevil(ctx)
	for _, e := range m.effects().Active {
		if e.EffectID == 484 {
			t.Fatal("relearned Demon kept the old radius")
		}
	}
	m.applyStatusEffectChange(ctx, network.StatusEffectChange{StatusID: db.StatusDevil1, Active: false})
	if len(m.effects().Active) != 0 {
		t.Fatal("Demon removal left its mask or rings")
	}
	m.applyStatusEffectChange(ctx, network.StatusEffectChange{StatusID: db.StatusDevil1, ActorID: 300, Active: true})
	if len(m.effects().Active) != 0 {
		t.Fatal("another actor's Demon blinded the player")
	}
}

func TestTaekwonReadyPosesBelongToSkillPackets(t *testing.T) {
	ctx := testutil.ArcherEffectContext()
	ctx.World.Player.Job, ctx.Session.Selected.Job = db.JobTaekwon, db.JobTaekwon
	for _, tc := range []struct {
		skill, status uint16
		family, frame int
	}{
		{412, db.StatusStormkickOn, sprites.ActionPCSkill, 0},
		{414, db.StatusDownkickOn, sprites.ActionPCSkill, 2},
		{416, db.StatusTurnkickOn, sprites.ActionPCSkill, 3},
		{418, db.StatusCounterOn, sprites.ActionPCSkill, 4},
		{420, db.StatusDodgeOn, scene.SpriteActionPickup, 1},
	} {
		m := &WorldMode{}
		for _, status := range []uint16{tc.status, tc.status + 1} {
			for _, active := range []bool{true, false} {
				m.applyStatusEffectChange(ctx, network.StatusEffectChange{ActorID: 2000000, StatusID: status, Active: active})
			}
		}
		if len(m.actorAnims)+len(m.effects().Active) != 0 {
			t.Fatal("stance status packets must only update the status icons")
		}
		m.applySkillNoDamageNotify(ctx, network.SkillNoDamageNotify{SkillID: tc.skill, SourceID: 2000000, TargetID: 2000000, Result: 1})
		anim := m.actorAnims[100]
		if anim.actionFamily != tc.family || !anim.hasFixedMotion || anim.fixedMotion != tc.frame || anim.play || anim.duration != 2*time.Second {
			t.Fatalf("skill %d ready animation: %+v", tc.skill, anim)
		}
		if len(m.effects().Active) != 1 || m.effects().Active[0].EffectID != effects.TaeReady {
			t.Fatalf("skill %d must flash blue: %+v", tc.skill, m.effects().Active)
		}
		if _, ok := m.actorAnimation(100, anim.started.Add(2*time.Second)); ok {
			t.Fatal("ready pose did not expire")
		}
	}
}

func TestTaekwonRunPacketOrderAndStopMarker(t *testing.T) {
	ctx := testutil.ArcherEffectContext()
	m := &WorldMode{}
	run := network.SkillNoDamageNotify{SkillID: 411, SourceID: 2000000, TargetID: 2000000, Result: 1}
	status := network.StatusEffectChange{ActorID: 100, StatusID: db.StatusRun, Active: true}
	m.applyStatusEffectChange(ctx, status)
	if len(m.effects().Active) != 0 {
		t.Fatal("initial status must not start or stop running")
	}
	m.applySkillNoDamageNotify(ctx, run)
	if !m.effects().HasActive(effects.Run, scene.LocalSkillTarget(ctx), time.Now()) {
		t.Fatal("Run skill did not start footprints")
	}
	status.Active = false
	m.applyStatusEffectChange(ctx, status)
	m.applySkillNoDamageNotify(ctx, run)
	ids := []int{}
	for _, e := range m.effects().Active {
		ids = append(ids, e.EffectID)
	}
	if !reflect.DeepEqual(ids, []int{effects.StopRun, effects.StopEffect}) {
		t.Fatalf("terminal Run packet restarted running: %v", ids)
	}
	if len(m.scheduledSounds) != 1 || m.scheduledSounds[0].paths[0] != "effect\\T_효과음1.wav" {
		t.Fatal("Run stop sound must play once")
	}
}

func TestWizardProjectilesAndSight(t *testing.T) {
	ctx := testutil.KnightEffectContext()
	start := time.Unix(100, 0)
	projection := testutil.Projection(800, 600, 20, 20, 0)
	m := &WorldMode{effectSystem: effects.System{EffectViews: map[string]*scene.SpriteView{"waterball": testutil.EffectRenderTestSprite(), "sight": testutil.EffectRenderTestSprite(), "data\\sprite\\shadow": testutil.EffectRenderTestSprite()}}}
	m.addSkillEffect(ctx, network.ActorActionNotify{SkillID: db.SkillWZWaterball, SourceID: 2000000, TargetID: 300, HitCount: 25}, start, start.Add(time.Second))
	if len(m.effects().Active) != 1 || len(m.scheduledSounds) != 1 {
		t.Fatal("Water Ball duplicated its launch")
	}
	frame := render.NewFrame(800, 600)
	m.effects().Draw(frame, ctx, projection, start.Add(20*db.EffectFrameDuration), false)
	if reflect.ValueOf(frame).Elem().FieldByName("worldBillboards").Len() != 12 {
		t.Fatal("Water Ball needs one head with its eleven trailing sprites")
	}

	m.effects().Active, m.scheduledSounds = nil, nil
	m.applyStatusEffectChange(ctx, network.StatusEffectChange{StatusID: db.StatusWzSightblaster, ActorID: 2000000, Active: true})
	m.applyStatusEffectChange(ctx, network.StatusEffectChange{StatusID: db.StatusWzSightblaster, ActorID: 100, Active: true})
	if len(m.effects().Active) != 1 || !m.effects().Active[0].Expires.IsZero() || len(m.scheduledSounds) != 0 {
		t.Fatal("Sight Blaster must persist once, silently")
	}
	frame = render.NewFrame(800, 600)
	m.effects().Draw(frame, ctx, projection, m.effects().Active[0].Starts.Add(time.Minute), false)
	cmds := reflect.ValueOf(frame).Elem().FieldByName("worldBillboards")
	if cmds.Len() == 0 {
		t.Fatal("Sight Blaster stopped drawing")
	}
	for i := 0; i < cmds.Len(); i++ {
		if a := cmds.Index(i).FieldByName("ColorA").Float(); a <= 0 || a > 50.0/255 {
			t.Fatalf("Sight Blaster alpha wrapped: %g", a)
		}
	}
	m.applyStatusEffectChange(ctx, network.StatusEffectChange{StatusID: db.StatusWzSightblaster, ActorID: 2000000, Active: false})
	if len(m.effects().Active) != 0 {
		t.Fatal("Sight Blaster survived status removal")
	}

	m.effects().AddAt(ctx, effects.SightTrasher, 2000000, start)
	for _, tick := range []int{0, 5, 15, 84, 85} {
		frame := render.NewFrame(800, 600)
		m.effects().Draw(frame, ctx, projection, start.Add(time.Duration(tick)*db.EffectFrameDuration), false)
		got := reflect.ValueOf(frame).Elem().FieldByName("worldBillboards").Len()
		want := 16 * (min(tick, 15)/5 + 1)
		if tick >= 84 {
			want = 0
		}
		if got != want {
			t.Fatalf("Sightrasher tick %d: %d sprites, want %d", tick, got, want)
		}
	}
}

func TestApplyLocalActorLookChangeUpdatesSelectedCharacter(t *testing.T) {
	sessionState := &session.Session{
		AccountID: 100,
		CharID:    200,
		Selected:  session.Character{ID: 200, Job: 0, Hair: 1},
		Characters: []session.Character{
			{ID: 200, Job: 0, Hair: 1},
		},
	}
	ctx := client.Context{
		Session: sessionState,
		World:   worldstate.New(),
	}
	look := network.ActorLookChange{
		ID:    200,
		Type:  2,
		Value: uint32(2101)<<16 | 1201,
	}

	if !applyActorLookChange(ctx, look) {
		t.Fatal("local look change should request player sprite reload")
	}
	if sessionState.Selected.Weapon != 1201 || sessionState.Selected.Shield != 2101 {
		t.Fatalf("selected appearance = weapon %d shield %d", sessionState.Selected.Weapon, sessionState.Selected.Shield)
	}
	if sessionState.Characters[0].Weapon != 1201 || sessionState.Characters[0].Shield != 2101 {
		t.Fatalf("character appearance = weapon %d shield %d", sessionState.Characters[0].Weapon, sessionState.Characters[0].Shield)
	}
	if ctx.World.Player.Weapon != 1201 || ctx.World.Player.Shield != 2101 {
		t.Fatalf("world player appearance = weapon %d shield %d", ctx.World.Player.Weapon, ctx.World.Player.Shield)
	}
}

func TestApplyStatusEffectChangeTracksLocalStatus(t *testing.T) {
	sessionState := &session.Session{AccountID: 2000000, CharID: 150000}
	ctx := client.Context{Session: sessionState}
	mode := &WorldMode{}

	mode.applyStatusEffectChange(ctx, network.StatusEffectChange{
		StatusID:    10,
		ActorID:     2000000,
		Active:      true,
		HasDuration: true,
		Duration:    30 * time.Second,
	})
	effect, ok := sessionState.Statuses.Active[10]
	if !ok {
		t.Fatal("status was not tracked")
	}
	if !effect.HasDuration || effect.ExpiresAt.IsZero() || effect.Source != 2000000 {
		t.Fatalf("effect = %+v", effect)
	}

	mode.applyStatusEffectChange(ctx, network.StatusEffectChange{
		StatusID: 10,
		ActorID:  2000000,
		Active:   false,
	})
	if _, ok := sessionState.Statuses.Active[10]; ok {
		t.Fatal("inactive status was not removed")
	}
}

func TestApplyStatusEffectChangeIgnoresRemoteActor(t *testing.T) {
	sessionState := &session.Session{AccountID: 2000000, CharID: 150000}
	ctx := client.Context{Session: sessionState}
	mode := &WorldMode{}

	mode.applyStatusEffectChange(ctx, network.StatusEffectChange{
		StatusID: 12,
		ActorID:  110000000,
		Active:   true,
	})
	if len(sessionState.Statuses.Active) != 0 {
		t.Fatalf("remote status changed local list: %+v", sessionState.Statuses.Active)
	}
}

func TestApplyFalconStatusUpdatesLocalActorAndSelectedOption(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{
		ID:          150000,
		Job:         db.JobHunter,
		EffectState: db.EffectStateRuwach,
		HasState:    true,
	}
	sessionState := &session.Session{
		AccountID: 2000000,
		CharID:    150000,
		Selected:  session.Character{ID: 150000, Job: db.JobHunter},
		Characters: []session.Character{
			{ID: 150000, Job: db.JobHunter},
		},
	}
	ctx := client.Context{Session: sessionState, World: world}
	mode := &WorldMode{}

	mode.applyStatusEffectChange(ctx, network.StatusEffectChange{
		StatusID: db.StatusFalcon,
		ActorID:  2000000,
		Active:   true,
	})
	if world.Player.EffectState&db.EffectStateFalcon == 0 {
		t.Fatalf("world player effect state = 0x%08X, want falcon bit", world.Player.EffectState)
	}
	if world.Player.EffectState&db.EffectStateRuwach == 0 {
		t.Fatalf("world player effect state = 0x%08X, want existing state preserved", world.Player.EffectState)
	}
	if sessionState.Selected.Option&db.EffectStateFalcon == 0 {
		t.Fatalf("selected option = 0x%08X, want falcon bit", sessionState.Selected.Option)
	}
	if sessionState.Characters[0].Option&db.EffectStateFalcon == 0 {
		t.Fatalf("character option = 0x%08X, want falcon bit", sessionState.Characters[0].Option)
	}
	if _, ok := sessionState.Statuses.Active[db.StatusFalcon]; !ok {
		t.Fatal("falcon status should still be tracked for the status icon")
	}

	mode.applyStatusEffectChange(ctx, network.StatusEffectChange{
		StatusID: db.StatusFalcon,
		ActorID:  2000000,
		Active:   false,
	})
	if world.Player.EffectState&db.EffectStateFalcon != 0 {
		t.Fatalf("world player effect state = 0x%08X, want falcon bit cleared", world.Player.EffectState)
	}
	if sessionState.Selected.Option&db.EffectStateFalcon != 0 {
		t.Fatalf("selected option = 0x%08X, want falcon bit cleared", sessionState.Selected.Option)
	}
	if _, ok := sessionState.Statuses.Active[db.StatusFalcon]; ok {
		t.Fatal("inactive falcon status should be removed from status tracking")
	}
}

func TestApplyFalconStatusUpdatesRemoteActor(t *testing.T) {
	world := worldstate.New()
	world.Actors[110000001] = worldstate.Actor{ID: 110000001, Job: db.JobHunter, HasState: true}
	ctx := client.Context{
		Session: &session.Session{AccountID: 2000000, CharID: 150000},
		World:   world,
	}
	mode := &WorldMode{}

	mode.applyStatusEffectChange(ctx, network.StatusEffectChange{
		StatusID: db.StatusFalcon,
		ActorID:  110000001,
		Active:   true,
	})
	actor := world.Actors[110000001]
	if actor.EffectState&db.EffectStateFalcon == 0 {
		t.Fatalf("remote actor effect state = 0x%08X, want falcon bit", actor.EffectState)
	}
	if len(ctx.Session.Statuses.Active) != 0 {
		t.Fatalf("remote falcon status changed local list: %+v", ctx.Session.Statuses.Active)
	}
}

func TestApplyTrickDeadStatusHoldsDeathPose(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 150000, Job: 0, X: 10, Y: 20}
	sessionState := &session.Session{AccountID: 2000000, CharID: 150000}
	ctx := client.Context{Session: sessionState, World: world}
	mode := &WorldMode{}

	mode.applyStatusEffectChange(ctx, network.StatusEffectChange{
		StatusID: db.StatusTrickdead,
		ActorID:  2000000,
		Active:   true,
	})
	anim, ok := mode.actorAnimation(150000, time.Now())
	if !ok || anim.actionFamily != sprites.ActionPCDeath || !anim.holdFinal {
		t.Fatalf("trick dead animation = %+v ok=%t, want held death pose", anim, ok)
	}

	mode.applyStatusEffectChange(ctx, network.StatusEffectChange{
		StatusID: db.StatusTrickdead,
		ActorID:  2000000,
		Active:   false,
	})
	anim, ok = mode.actorAnimation(150000, time.Now())
	if !ok || anim.actionFamily != scene.SpriteActionIdle || anim.holdFinal {
		t.Fatalf("trick dead inactive animation = %+v ok=%t, want idle", anim, ok)
	}
}

func TestTrickDeadSkillDoesNotStartDefaultSkillAction(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 150000, Job: 0, X: 10, Y: 20}
	sessionState := &session.Session{AccountID: 2000000, CharID: 150000}
	ctx := client.Context{Session: sessionState, World: world}
	mode := &WorldMode{}

	if action := skillAction(db.SkillNVTrickdead); !action.defined || action.action != skillActorActionNone {
		t.Fatalf("trick dead skill action = %+v, want no source action", action)
	}
	mode.applySkillNoDamageNotify(ctx, network.SkillNoDamageNotify{
		SkillID:  db.SkillNVTrickdead,
		SourceID: 2000000,
		TargetID: 2000000,
		Result:   1,
	})
	if len(mode.actorAnims) != 0 {
		t.Fatalf("trick dead skill animation = %+v, want none before status", mode.actorAnims)
	}
}

func TestBackSlideSkillDoesNotStartDefaultSkillAction(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20}
	sessionState := &session.Session{AccountID: 2000000, CharID: 150000}
	ctx := client.Context{Session: sessionState, World: world}
	mode := &WorldMode{}

	if action := skillAction(db.SkillTFBacksliding); !action.defined || action.action != skillActorActionNone {
		t.Fatalf("back slide skill action = %+v, want no source action", action)
	}
	mode.applySkillNoDamageNotify(ctx, network.SkillNoDamageNotify{
		SkillID:  db.SkillTFBacksliding,
		SourceID: 2000000,
		TargetID: 2000000,
		Result:   1,
	})
	if len(mode.actorAnims) != 0 {
		t.Fatalf("back slide skill animation = %+v, want none before jump packet", mode.actorAnims)
	}
}

func TestApplyActorJumpPositionMovesLocalPlayer(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20, Dir: 4, Moving: true}
	sessionState := &session.Session{AccountID: 2000000, PlayerX: 10, PlayerY: 20}
	ctx := client.Context{Session: sessionState, World: world}

	applyActorJumpPosition(ctx, network.ActorJumpPosition{ID: 2000000, X: 7, Y: 20})

	if world.Player.X != 7 || world.Player.Y != 20 || world.Player.Moving {
		t.Fatalf("player after jump = %+v, want stopped at 7,20", world.Player)
	}
	if sessionState.PlayerX != 7 || sessionState.PlayerY != 20 {
		t.Fatalf("session position = %d,%d, want 7,20", sessionState.PlayerX, sessionState.PlayerY)
	}
}

func TestApplyMapAcceptEnterMarksAdminPlayer(t *testing.T) {
	world := worldstate.New()
	sessionState := &session.Session{AccountID: 2000000, CharID: 150000, AdminList: []uint32{2000000}}
	ctx := client.Context{Session: sessionState, World: world}

	applyMapAcceptEnter(ctx, network.MapAcceptEnter{X: 10, Y: 20, Dir: 4})

	if !world.Player.IsAdmin {
		t.Fatalf("world player admin state = false")
	}
}

func TestApplyMapAcceptEnterResetsWorldForChangedSelectedCharacter(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{
		ID:           150000,
		Job:          db.JobAlchemist,
		HasCartState: true,
		HasCart:      true,
		CartNum:      4,
		EffectState:  db.EffectStateCart4,
		Opt3State:    db.Opt3Quicken,
		HasState:     true,
	}
	world.Actors[300] = worldstate.Actor{ID: 300, Name: "stale actor"}
	world.Items[400] = worldstate.FloorItem{ID: 400, ItemID: 501}
	sessionState := &session.Session{
		AccountID: 2000000,
		CharID:    150001,
		Sex:       1,
		Selected: session.Character{
			ID:     150001,
			Name:   "Wizard",
			Job:    db.JobWizard,
			Hair:   7,
			Weapon: 1601,
		},
		Progress: session.Progress{BaseLevel: 42},
	}
	ctx := client.Context{Session: sessionState, World: world}

	applyMapAcceptEnter(ctx, network.MapAcceptEnter{X: 20, Y: 30, Dir: 3})

	if world.Player.ID != 150001 || world.Player.Name != "Wizard" || world.Player.Job != db.JobWizard || world.Player.Weapon != 1601 {
		t.Fatalf("world player identity/appearance = %+v", world.Player)
	}
	if world.Player.HasCart || world.Player.CartNum != 0 || world.Player.EffectState&actorEffectCartMask != 0 {
		t.Fatalf("old cart state leaked into selected character: %+v", world.Player)
	}
	if world.Player.Opt3State != 0 {
		t.Fatalf("old opt3 state leaked into selected character: 0x%08X", world.Player.Opt3State)
	}
	if world.Player.X != 20 || world.Player.Y != 30 || world.Player.Dir != 3 || world.Dir != 3 {
		t.Fatalf("world player position = %+v world dir=%d", world.Player, world.Dir)
	}
	if len(world.Actors) != 0 || len(world.Items) != 0 {
		t.Fatalf("stale world state survived character switch actors=%+v items=%+v", world.Actors, world.Items)
	}
}

func TestApplyMapAcceptEnterKeepsWorldStateForSameCharacter(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{
		ID:           150000,
		Job:          db.JobAlchemist,
		HasCartState: true,
		HasCart:      true,
		CartNum:      4,
		EffectState:  db.EffectStateCart4,
		Opt3State:    db.Opt3Quicken,
		HasState:     true,
	}
	world.Actors[300] = worldstate.Actor{ID: 300, Name: "same map actor"}
	sessionState := &session.Session{
		AccountID: 2000000,
		CharID:    150000,
		Playing:   true,
		Selected:  session.Character{ID: 150000, Job: db.JobAlchemist},
	}
	ctx := client.Context{Session: sessionState, World: world}

	applyMapAcceptEnter(ctx, network.MapAcceptEnter{X: 20, Y: 30, Dir: 3})

	if !world.Player.HasCartState || !world.Player.HasCart || world.Player.CartNum != 4 || world.Player.EffectState&db.EffectStateCart4 == 0 {
		t.Fatalf("same-character cart state was not preserved: %+v", world.Player)
	}
	if world.Player.Opt3State&db.Opt3Quicken == 0 {
		t.Fatalf("same-character opt3 state was not preserved: 0x%08X", world.Player.Opt3State)
	}
	if len(world.Actors) != 1 {
		t.Fatalf("same-character actors were cleared: %+v", world.Actors)
	}
}

func TestApplyMapAcceptEnterResetsWorldForCharacterSelectReentry(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{
		ID:           150000,
		Job:          db.JobAlchemist,
		HasCartState: true,
		HasCart:      true,
		CartNum:      4,
		EffectState:  db.EffectStateCart4,
		Opt3State:    db.Opt3Quicken,
		HasState:     true,
	}
	world.Actors[300] = worldstate.Actor{ID: 300, Name: "stale actor"}
	sessionState := &session.Session{
		AccountID: 2000000,
		CharID:    150000,
		Playing:   false,
		Selected:  session.Character{ID: 150000, Job: db.JobAlchemist},
	}
	ctx := client.Context{Session: sessionState, World: world}

	applyMapAcceptEnter(ctx, network.MapAcceptEnter{X: 20, Y: 30, Dir: 3})

	if world.Player.HasCart || world.Player.CartNum != 0 || world.Player.EffectState&actorEffectCartMask != 0 {
		t.Fatalf("cart state leaked through character-select re-entry: %+v", world.Player)
	}
	if world.Player.Opt3State != 0 {
		t.Fatalf("opt3 state leaked through character-select re-entry: 0x%08X", world.Player.Opt3State)
	}
	if len(world.Actors) != 0 {
		t.Fatalf("stale actors survived character-select re-entry: %+v", world.Actors)
	}
}

func TestApplyPushCartStatusTracksLocalAndRemoteActors(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 150000, Job: 5}
	world.UpsertActor(worldstate.Actor{ID: 110000001, X: 10, Y: 20, Job: 5, Appearance: true})
	ctx := client.Context{
		Session: &session.Session{AccountID: 2000000, CharID: 150000},
		World:   world,
	}
	mode := &WorldMode{}

	mode.applyStatusEffectChange(ctx, network.StatusEffectChange{
		StatusID:  db.StatusOnPushCart,
		ActorID:   2000000,
		Active:    true,
		HasValues: true,
		Values:    [3]int32{4, 0, 0},
	})
	if !world.Player.HasCartState || !world.Player.HasCart || world.Player.CartNum != 4 {
		t.Fatalf("local cart state = %+v", world.Player)
	}
	if len(ctx.Session.Statuses.Active) != 0 {
		t.Fatalf("pushcart should not create a buff icon: %+v", ctx.Session.Statuses.Active)
	}
	mode.applyActorStateChange(ctx, network.ActorStateChange{
		ID:          2000000,
		BodyState:   0,
		HealthState: 0,
		EffectState: 0,
	})
	if !world.Player.HasCartState || world.Player.HasCart || world.Player.CartNum != 0 || world.Player.EffectState&actorEffectCartMask != 0 {
		t.Fatalf("local cart state after actor state refresh = %+v", world.Player)
	}
	mode.applyActorStateChange(ctx, network.ActorStateChange{
		ID:          2000000,
		BodyState:   0,
		HealthState: 0,
		EffectState: db.EffectStateCart2,
	})
	if !world.Player.HasCartState || !world.Player.HasCart || world.Player.CartNum != 2 {
		t.Fatalf("local cart state after change-cart actor state = %+v", world.Player)
	}

	mode.applyStatusEffectChange(ctx, network.StatusEffectChange{
		StatusID:  db.StatusOnPushCart,
		ActorID:   110000001,
		Active:    true,
		HasValues: true,
		Values:    [3]int32{2, 0, 0},
	})
	remote := world.Actors[110000001]
	if !remote.HasCartState || !remote.HasCart || remote.CartNum != 2 {
		t.Fatalf("remote cart state = %+v", remote)
	}
	mode.applyActorStateChange(ctx, network.ActorStateChange{
		ID:          110000001,
		BodyState:   0,
		HealthState: 0,
		EffectState: 0,
	})
	remote = world.Actors[110000001]
	if !remote.HasCartState || remote.HasCart || remote.CartNum != 0 || remote.EffectState&actorEffectCartMask != 0 {
		t.Fatalf("remote cart state after actor state refresh = %+v", remote)
	}

	mode.applyStatusEffectChange(ctx, network.StatusEffectChange{
		StatusID: db.StatusOnPushCart,
		ActorID:  110000001,
		Active:   false,
	})
	remote = world.Actors[110000001]
	if !remote.HasCartState || remote.HasCart || remote.EffectState&actorEffectCartMask != 0 {
		t.Fatalf("inactive remote cart state = %+v", remote)
	}
}

func TestApplyActorStateChangeSyncsSelectedWeddingOption(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 150000, Job: db.JobWizard, HasState: true}
	sessionState := &session.Session{
		AccountID: 2000000,
		CharID:    150000,
		Selected:  session.Character{ID: 150000, Job: db.JobWizard},
		Characters: []session.Character{
			{ID: 150000, Job: db.JobWizard},
		},
	}
	ctx := client.Context{Session: sessionState, World: world}
	mode := &WorldMode{}

	mode.applyActorStateChange(ctx, network.ActorStateChange{
		ID:          2000000,
		BodyState:   0,
		HealthState: 0,
		EffectState: db.EffectStateWedding,
	})
	if world.Player.EffectState&db.EffectStateWedding == 0 {
		t.Fatalf("world player effect state = 0x%08X, want wedding bit", world.Player.EffectState)
	}
	if sessionState.Selected.Option&db.EffectStateWedding == 0 {
		t.Fatalf("selected option = 0x%08X, want wedding bit", sessionState.Selected.Option)
	}
	if sessionState.Characters[0].Option&db.EffectStateWedding == 0 {
		t.Fatalf("character option = 0x%08X, want wedding bit", sessionState.Characters[0].Option)
	}

	mode.applyActorStateChange(ctx, network.ActorStateChange{
		ID:          2000000,
		BodyState:   0,
		HealthState: 0,
		EffectState: 0,
	})
	if world.Player.EffectState&db.EffectStateWedding != 0 {
		t.Fatalf("world player effect state = 0x%08X, want wedding bit cleared", world.Player.EffectState)
	}
	if sessionState.Selected.Option&db.EffectStateWedding != 0 {
		t.Fatalf("selected option = 0x%08X, want wedding bit cleared", sessionState.Selected.Option)
	}
	if sessionState.Characters[0].Option&db.EffectStateWedding != 0 {
		t.Fatalf("character option = 0x%08X, want wedding bit cleared", sessionState.Characters[0].Option)
	}
}

func TestActorCartStateFromEffectUsesReferenceCartNumbers(t *testing.T) {
	actor := worldstate.Actor{Job: 5, EffectState: db.EffectStateCart3}
	hasCart, cartNum := actorCartState(actor)
	if !hasCart || cartNum != 3 {
		t.Fatalf("cart from effect = %t, %d", hasCart, cartNum)
	}
	actor = worldstate.Actor{Job: 23, EffectState: db.EffectStateCart5}
	hasCart, cartNum = actorCartState(actor)
	if !hasCart || cartNum != 0 {
		t.Fatalf("super novice cart from effect = %t, %d", hasCart, cartNum)
	}
}

func TestActorHasFalconUsesReferenceEffectBitAndJobs(t *testing.T) {
	if !actorHasFalcon(worldstate.Actor{Job: db.JobHunter, EffectState: db.EffectStateFalcon}) {
		t.Fatal("hunter falcon bit should create a falcon")
	}
	if !actorHasFalcon(worldstate.Actor{Job: db.JobHunterH, EffectState: db.EffectStateFalcon}) {
		t.Fatal("sniper falcon bit should create a falcon")
	}
	if actorHasFalcon(worldstate.Actor{Job: db.JobKnight, EffectState: db.EffectStateFalcon}) {
		t.Fatal("non-hunter falcon bit should not create a falcon sprite")
	}
	if actorHasFalcon(worldstate.Actor{Job: db.JobHunter}) {
		t.Fatal("hunter without falcon bit should not create a falcon")
	}
}

func TestFalconStateFollowsAsIndependentEntity(t *testing.T) {
	mode := &WorldMode{}
	start := time.Unix(100, 0)
	actor := worldstate.Actor{
		ID:          150011,
		Job:         db.JobHunter,
		EffectState: db.EffectStateFalcon,
		X:           10,
		Y:           20,
		Dir:         4,
		Speed:       150,
	}

	falcon := mode.updateFalconState(actor, start)
	if falcon == nil {
		t.Fatal("falcon state was not created")
	}
	if falcon.x != 10 || falcon.y != 20 || falcon.moving {
		t.Fatalf("initial falcon = x %.2f y %.2f moving=%t, want owner cell and idle", falcon.x, falcon.y, falcon.moving)
	}

	actor.X = 13
	falcon = mode.updateFalconState(actor, start.Add(falconRetargetInterval+time.Millisecond))
	if falcon == nil || !falcon.moving {
		t.Fatalf("falcon = %+v, want independent follow movement", falcon)
	}
	if falcon.moveSpeedMS != 100 {
		t.Fatalf("falcon speed = %d, want owner speed - 50", falcon.moveSpeedMS)
	}
	if len(falcon.path) == 0 {
		t.Fatal("falcon follow path is empty")
	}
	end := falcon.path[len(falcon.path)-1]
	if end.X != 12 || end.Y != 20 {
		t.Fatalf("falcon follow endpoint = %d,%d, want to stop one cell from owner", end.X, end.Y)
	}

	falcon.advance(start.Add(falconRetargetInterval + 50*time.Millisecond))
	if falcon.x <= 10 || falcon.x >= 13 {
		t.Fatalf("falcon x = %.2f, want independent position between old position and owner", falcon.x)
	}
}

func TestFalconFollowUsesRobrowserRangeFloor(t *testing.T) {
	if got := falconFollowDistance(11.9, 20, 10, 20); got != 1 {
		t.Fatalf("falcon follow distance = %d, want floored euclidean distance", got)
	}
	path := falconFollowPath(10, 20, 12, 21, falconStopRange)
	end := path[len(path)-1]
	if end.X != 11 || end.Y != 21 {
		t.Fatalf("diagonal-ish falcon endpoint = %d,%d, want floor(distance) <= 1 stop", end.X, end.Y)
	}
}

func TestFalconSpriteStateUsesIdleGlideAction(t *testing.T) {
	falcon := &falconRenderState{direction: 3, moving: true}
	state := falcon.spriteState(45)
	if state.ActionFamily != scene.SpriteActionIdle || !state.LoopIdle {
		t.Fatalf("falcon sprite state = %+v, want looping idle glide action", state)
	}
	if state.ActionFamily == scene.SpriteActionWalk {
		t.Fatal("falcon follow should not use character walk action")
	}

	falcon.attacking = true
	state = falcon.spriteState(45)
	if state.ActionFamily != scene.SpriteActionWalk {
		t.Fatalf("falcon attack sprite state = %+v, want walk action like robr", state)
	}
}

func TestFalconNoDamageSkillAttackUsesRobrowserOvershoot(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 150006, X: 10, Y: 20, Dir: 4}
	world.Actors[300] = worldstate.Actor{ID: 300, Job: 1002, X: 12, Y: 20}
	ctx := client.Context{
		Session: &session.Session{
			AccountID: 2000000,
			CharID:    150006,
			Selected: session.Character{
				ID:     150006,
				Job:    db.JobHunter,
				Option: db.EffectStateFalcon,
			},
		},
		World: world,
	}
	mode := &WorldMode{}

	mode.applySkillNoDamageNotify(ctx, network.SkillNoDamageNotify{
		SkillID:  db.SkillHTBlitzbeat,
		SourceID: 2000000,
		TargetID: 300,
		Result:   1,
	})
	falcon := mode.falcons[150006]
	if falcon == nil || !falcon.attacking {
		t.Fatalf("falcon = %+v, want attacking state", falcon)
	}
	if falcon.moveSpeedMS != falconAttackMoveSpeedMS {
		t.Fatalf("falcon attack speed = %d, want %d", falcon.moveSpeedMS, falconAttackMoveSpeedMS)
	}
	end := falcon.path[len(falcon.path)-1]
	if end.X != 17 || end.Y != 20 {
		t.Fatalf("falcon attack endpoint = %d,%d, want robr overshoot past target", end.X, end.Y)
	}
}

func TestFalconActorActionSkillAttackForFalconAssault(t *testing.T) {
	world := worldstate.New()
	world.Actors[200] = worldstate.Actor{
		ID:          200,
		Job:         db.JobHunterH,
		X:           10,
		Y:           20,
		Dir:         4,
		EffectState: db.EffectStateFalcon,
		HasState:    true,
	}
	world.Actors[300] = worldstate.Actor{ID: 300, Job: 1002, X: 8, Y: 19}
	ctx := client.Context{
		Session: &session.Session{AccountID: 2000000, CharID: 150006},
		World:   world,
	}
	mode := &WorldMode{}

	mode.applyActorActionNotify(ctx, network.ActorActionNotify{
		SkillID:  db.SkillSNFalconassault,
		SourceID: 200,
		TargetID: 300,
	})
	falcon := mode.falcons[200]
	if falcon == nil || !falcon.attacking {
		t.Fatalf("falcon = %+v, want Falcon Assault attack state", falcon)
	}
	end := falcon.path[len(falcon.path)-1]
	if end.X != 3 || end.Y != 14 {
		t.Fatalf("falcon assault endpoint = %d,%d, want robr diagonal overshoot", end.X, end.Y)
	}
}

func TestFalconDetectingWaitsForGroundSkill(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 150006, X: 10, Y: 20, Dir: 4}
	ctx := client.Context{
		Session: &session.Session{
			AccountID: 2000000,
			CharID:    150006,
			Selected: session.Character{
				ID:     150006,
				Job:    db.JobHunter,
				Option: db.EffectStateFalcon,
			},
		},
		World: world,
	}
	mode := &WorldMode{}

	mode.applySkillCastNotify(ctx, network.SkillCastNotify{
		SkillID:  db.SkillHTDetecting,
		SourceID: 2000000,
		X:        12,
		Y:        22,
	})
	if mode.falcons[150006] != nil {
		t.Fatal("Detecting must not launch its falcon before the cast completes")
	}
	mode.applyGroundSkillNotify(ctx, network.GroundSkillNotify{SourceID: 2000000, SkillID: db.SkillHTDetecting, X: 12, Y: 22})
	falcon := mode.falcons[150006]
	if falcon == nil || !falcon.attacking {
		t.Fatalf("falcon = %+v, want Detecting attack even with zero cast delay", falcon)
	}
	end := falcon.path[len(falcon.path)-1]
	if end.X != 17 || end.Y != 27 {
		t.Fatalf("detecting endpoint = %d,%d, want robr ground overshoot", end.X, end.Y)
	}
}

func TestFalconAttackReturnsToFollowTarget(t *testing.T) {
	start := time.Unix(100, 0)
	state := &falconRenderState{
		x:           10,
		y:           20,
		direction:   4,
		moveSpeedMS: 100,
		hasTarget:   true,
		targetX:     10,
		targetY:     20,
	}
	state.startAttack(12, 20, start)
	mode := &WorldMode{falcons: map[uint32]*falconRenderState{150006: state}}
	actor := worldstate.Actor{
		ID:          150006,
		Job:         db.JobHunter,
		EffectState: db.EffectStateFalcon,
		X:           10,
		Y:           20,
		Speed:       150,
	}

	mode.updateFalconState(actor, start.Add(falconAttackReturnDelay+time.Millisecond))
	if state.attacking {
		t.Fatal("falcon should leave attacking state after robr return delay")
	}
	if state.moveSpeedMS != falconAttackMoveSpeedMS {
		t.Fatalf("return speed = %d, want attack speed retained like robr", state.moveSpeedMS)
	}
	end := state.path[len(state.path)-1]
	if end.X != 10 || end.Y != 20 {
		t.Fatalf("return endpoint = %d,%d, want last follow target", end.X, end.Y)
	}
}

func TestDrawSceneModelsAndActorsRunsFalconPass(t *testing.T) {
	world := worldstate.New()
	world.MapName = "hugel"
	world.Player = worldstate.Actor{ID: 150006, X: 209, Y: 220, Dir: 4}
	ctx := client.Context{
		Session: &session.Session{
			AccountID: 2000000,
			CharID:    150006,
			Selected: session.Character{
				ID:     150006,
				Job:    db.JobHunter,
				Option: db.EffectStateFalcon,
			},
		},
		Resources: &res.Manager{},
		World:     world,
	}
	mode := &WorldMode{
		falconViews: map[int]*scene.SpriteView{
			db.JobHunter: {},
		},
	}
	screen := render.NewFrame(800, 600)
	projection := testutil.Projection(800, 600, scene.CellCenter(209), scene.CellCenter(220), 0)

	actors := mode.drawSceneModelsAndActors(screen, ctx, projection, time.Unix(100, 0))
	if len(actors) == 0 {
		t.Fatal("no actor entries collected")
	}
	if _, ok := mode.falcons[150006]; !ok {
		t.Fatal("mixed scene draw path did not run falcon pass")
	}
}

func TestCollectSceneActorEntriesUsesSelectedCharacterCartOption(t *testing.T) {
	world := worldstate.New()
	world.MapName = "prontera"
	world.Player = worldstate.Actor{ID: 150004, X: 10, Y: 20, Dir: 4}
	ctx := client.Context{
		Session: &session.Session{
			AccountID: 2000000,
			CharID:    150004,
			AdminList: []uint32{2000000},
			Selected: session.Character{
				ID:     150004,
				Job:    5,
				Option: db.EffectStateCart1,
			},
		},
		World: world,
	}
	mode := &WorldMode{}
	screen := render.NewFrame(800, 600)
	projection := testutil.Projection(800, 600, scene.CellCenter(10), scene.CellCenter(20), 0)

	entries := mode.collectSceneActorEntries(screen, ctx, projection)
	if len(entries) == 0 {
		t.Fatal("no scene actor entries collected")
	}
	if hasCart, cartNum := actorCartState(entries[0].Actor); !hasCart || cartNum != 1 {
		t.Fatalf("local cart from selected character option = has %t num %d actor %+v", hasCart, cartNum, entries[0].Actor)
	}
	if !entries[0].Actor.IsAdmin || !world.Player.IsAdmin {
		t.Fatalf("local admin state not applied: entry=%t world=%t", entries[0].Actor.IsAdmin, world.Player.IsAdmin)
	}
}

func TestCollectSceneActorEntriesMergesSelectedCharacterFalconOption(t *testing.T) {
	world := worldstate.New()
	world.MapName = "pay_arche"
	world.Player = worldstate.Actor{
		ID:          150011,
		X:           10,
		Y:           20,
		Dir:         4,
		HasState:    true,
		EffectState: db.EffectStateRuwach,
	}
	ctx := client.Context{
		Session: &session.Session{
			AccountID: 2000000,
			CharID:    150011,
			Selected: session.Character{
				ID:     150011,
				Job:    db.JobHunter,
				Option: db.EffectStateFalcon,
			},
		},
		World: world,
	}
	mode := &WorldMode{}
	screen := render.NewFrame(800, 600)
	projection := testutil.Projection(800, 600, scene.CellCenter(10), scene.CellCenter(20), 0)

	entries := mode.collectSceneActorEntries(screen, ctx, projection)
	if len(entries) == 0 {
		t.Fatal("no scene actor entries collected")
	}
	if entries[0].Actor.EffectState&db.EffectStateFalcon == 0 {
		t.Fatalf("entry effect state = 0x%08X, want selected falcon option merged", entries[0].Actor.EffectState)
	}
	if entries[0].Actor.EffectState&db.EffectStateRuwach == 0 {
		t.Fatalf("entry effect state = 0x%08X, want live effect state preserved", entries[0].Actor.EffectState)
	}
}

func TestCollectSceneActorEntriesMergesSelectedCharacterWeddingOption(t *testing.T) {
	world := worldstate.New()
	world.MapName = "prontera"
	world.Player = worldstate.Actor{
		ID:          150012,
		X:           10,
		Y:           20,
		Dir:         4,
		HasState:    true,
		EffectState: db.EffectStateRuwach,
	}
	ctx := client.Context{
		Session: &session.Session{
			AccountID: 2000000,
			CharID:    150012,
			Selected: session.Character{
				ID:     150012,
				Job:    db.JobWizard,
				Option: db.EffectStateWedding,
			},
		},
		World: world,
	}
	mode := &WorldMode{}
	screen := render.NewFrame(800, 600)
	projection := testutil.Projection(800, 600, scene.CellCenter(10), scene.CellCenter(20), 0)

	entries := mode.collectSceneActorEntries(screen, ctx, projection)
	if len(entries) == 0 {
		t.Fatal("no scene actor entries collected")
	}
	if entries[0].Actor.EffectState&db.EffectStateWedding == 0 {
		t.Fatalf("entry effect state = 0x%08X, want selected wedding option merged", entries[0].Actor.EffectState)
	}
	if entries[0].Actor.EffectState&db.EffectStateRuwach == 0 {
		t.Fatalf("entry effect state = 0x%08X, want live effect state preserved", entries[0].Actor.EffectState)
	}
	if entries[0].Actor.Job != db.JobMarried {
		t.Fatalf("entry visual job = %d, want married job %d", entries[0].Actor.Job, db.JobMarried)
	}
}

func TestCartOffsetBillboardAppliesReferencePixelOffset(t *testing.T) {
	base := &scene.Billboard{AnchorX: 100, AnchorY: 200}
	dx, dy := cartSpriteOffset(2)
	got := cartOffsetBillboard(base, dx, dy)
	if got == base {
		t.Fatal("cart offset should copy billboard")
	}
	if got.AnchorX != 60 || got.AnchorY != 200 {
		t.Fatalf("offset billboard anchor = %.0f, %.0f", got.AnchorX, got.AnchorY)
	}
	if base.AnchorX != 100 || base.AnchorY != 200 {
		t.Fatalf("base billboard mutated = %.0f, %.0f", base.AnchorX, base.AnchorY)
	}
}

func TestCartDirectionOffsetBillboardPlacesShadowAtCart(t *testing.T) {
	base := &scene.Billboard{AnchorX: 100, AnchorY: 200}
	got := cartDirectionOffsetBillboard(base, 5)
	if got == base {
		t.Fatal("cart shadow offset should copy billboard")
	}
	if got.AnchorX != 130 || got.AnchorY != 190 {
		t.Fatalf("cart shadow anchor = %.0f, %.0f, want 130, 190", got.AnchorX, got.AnchorY)
	}
	if base.AnchorX != 100 || base.AnchorY != 200 {
		t.Fatalf("base shadow billboard mutated = %.0f, %.0f", base.AnchorX, base.AnchorY)
	}
}

func TestDrawActorCartShadowUsesLoadedShadowSprite(t *testing.T) {
	shadowView := &scene.SpriteView{
		Spr: &res.SPR{
			RGBAIndex: 0,
			Frames: []res.SPRFrame{{
				Type:   res.SPRFrameRGBA,
				Width:  16,
				Height: 8,
				Data:   testutil.SolidRGBAFrame(16, 8),
			}},
		},
		Act: &res.ACT{Actions: []res.ACTAction{{
			Animations: []res.ACTAnimation{{Layers: []res.ACTLayer{{
				Index:   0,
				SPRType: res.SPRFrameRGBA,
				ScaleX:  1,
				ScaleY:  1,
				Color:   [4]float32{1, 1, 1, 1},
			}}}},
		}}},
		Images:     make(map[scene.SpriteFrameKey]*render.Image),
		Billboards: make(map[scene.SpriteBillboardKey]*scene.Billboard),
	}
	mode := &WorldMode{shadowView: shadowView}
	entry := scene.ActorDrawEntry{
		Actor: worldstate.Actor{
			ID:           300,
			Job:          db.JobMerchant,
			Dir:          2,
			HasCartState: true,
			HasCart:      true,
			CartNum:      1,
		},
		WorldX:      scene.CellCenter(10),
		WorldY:      scene.CellCenter(20),
		Scale:       1,
		Shadow:      1,
		CastShadow:  true,
		ShadowScale: 1,
	}
	projection := testutil.Projection(800, 600, entry.WorldX, entry.WorldY, 0)
	if !mode.drawActorCartShadow3D(render.NewFrame(800, 600), client.Context{}, projection, entry, 0, time.Now()) {
		t.Fatal("cart shadow was not drawn")
	}
	entry.Actor.HasCart = false
	if mode.drawActorCartShadow3D(render.NewFrame(800, 600), client.Context{}, projection, entry, 0, time.Now()) {
		t.Fatal("cart shadow drawn without a cart")
	}
}

func TestApplyActorStateChangeTracksRemoteActorRenderState(t *testing.T) {
	world := worldstate.New()
	world.UpsertActor(worldstate.Actor{ID: 110000001, X: 10, Y: 20, Job: 1002, Speed: 400, Appearance: true})
	ctx := client.Context{World: world}
	mode := &WorldMode{}

	mode.applyActorStateChange(ctx, network.ActorStateChange{
		ID:          110000001,
		BodyState:   db.BodyStateFreeze,
		HealthState: db.HealthStateBlind,
		EffectState: 0x00402000,
	})

	actor := world.Actors[110000001]
	if !actor.HasState || actor.BodyState != db.BodyStateFreeze || actor.HealthState != db.HealthStateBlind || actor.EffectState != 0x00402000 {
		t.Fatalf("actor state = %+v", actor)
	}
	if len(mode.effects().Active) != 1 || mode.effects().Active[0].EffectID != effects.Ruwach || mode.effects().Active[0].ActorID != 110000001 {
		t.Fatalf("actor state effects = %+v, want Ruwach", mode.effects().Active)
	}
	state := mode.nonPCSpriteState(actor, time.Now())
	if state.ActionFamily != scene.SpriteActionIdle || !state.HasPlay || state.Play || !state.HasFixedMotion || state.FixedMotion != 0 {
		t.Fatalf("frozen non-pc sprite state = %+v", state)
	}

	mode.applyActorStateChange(ctx, network.ActorStateChange{
		ID:          110000001,
		BodyState:   db.BodyStateFreeze,
		HealthState: db.HealthStateBlind,
		EffectState: 0,
	})
	if len(mode.effects().Active) != 0 {
		t.Fatalf("actor state effects after clear = %+v, want none", mode.effects().Active)
	}
}

func TestActorEntryWithEffectStateStartsStateEffect(t *testing.T) {
	world := worldstate.New()
	ctx := client.Context{World: world}
	mode := &WorldMode{}

	mode.upsertNetworkActor(ctx, network.ActorEntry{
		ID:          110000001,
		X:           10,
		Y:           20,
		Job:         1002,
		EffectState: db.EffectStateRuwach,
		HasState:    true,
	})
	if len(mode.effects().Active) != 1 || mode.effects().Active[0].EffectID != effects.Ruwach || mode.effects().Active[0].ActorID != 110000001 {
		t.Fatalf("actor entry effects = %+v, want Ruwach", mode.effects().Active)
	}

	mode.upsertNetworkActor(ctx, network.ActorEntry{
		ID:          110000001,
		X:           10,
		Y:           20,
		Job:         1002,
		EffectState: db.EffectStateRuwach,
		HasState:    true,
	})
	if len(mode.effects().Active) != 1 || mode.effects().Active[0].EffectID != effects.Ruwach {
		t.Fatalf("refreshed actor entry effects = %+v, want one Ruwach", mode.effects().Active)
	}

	mode.upsertNetworkActor(ctx, network.ActorEntry{
		ID:          110000001,
		X:           10,
		Y:           20,
		Job:         1002,
		EffectState: 0,
		HasState:    true,
	})
	if len(mode.effects().Active) != 0 {
		t.Fatalf("actor entry effects after clear = %+v, want none", mode.effects().Active)
	}
}

func TestSyncCurrentActorEffectStateEffectsStartsExistingActorEffects(t *testing.T) {
	world := worldstate.New()
	world.UpsertActor(worldstate.Actor{
		ID:          110000001,
		X:           10,
		Y:           20,
		Job:         1002,
		EffectState: db.EffectStateRuwach,
		HasState:    true,
	})
	ctx := client.Context{World: world}
	mode := &WorldMode{}

	mode.syncCurrentActorEffectStateEffects(ctx)
	if len(mode.effects().Active) != 1 || mode.effects().Active[0].EffectID != effects.Ruwach || mode.effects().Active[0].ActorID != 110000001 {
		t.Fatalf("synced actor effects = %+v, want Ruwach", mode.effects().Active)
	}
}

func TestActorVanishRemovesActorEffectStateEffects(t *testing.T) {
	world := worldstate.New()
	world.UpsertActor(worldstate.Actor{ID: 110000001, X: 10, Y: 20, Job: 1002, Appearance: true})
	ctx := client.Context{World: world}
	mode := &WorldMode{}

	mode.applyActorStateChange(ctx, network.ActorStateChange{
		ID:          110000001,
		EffectState: db.EffectStateRuwach,
	})
	if len(mode.effects().Active) != 1 {
		t.Fatalf("world effects before vanish = %+v, want Ruwach", mode.effects().Active)
	}

	mode.applyActorVanish(ctx, network.ActorVanish{ID: 110000001})
	if len(mode.effects().Active) != 0 {
		t.Fatalf("world effects after vanish = %+v, want none", mode.effects().Active)
	}
}

func TestActorStateTintMatchesReferenceBodyAndHealthTints(t *testing.T) {
	tint := actorStateTint(worldstate.Actor{
		BodyState:   db.BodyStateFreeze,
		HealthState: db.HealthStateBlind,
		HasState:    true,
	})
	if tint.R != 0 || tint.G != 20 || tint.B != 40 || tint.A != 255 {
		t.Fatalf("tint = %+v, want frozen blue darkened by blind", tint)
	}
}

func TestEnergyCoatStatusSetsActorOpt3StateAndTint(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20}
	sessionState := &session.Session{AccountID: 2000000, CharID: 150000}
	ctx := client.Context{Session: sessionState, World: world}
	mode := &WorldMode{}

	mode.applyStatusEffectChange(ctx, network.StatusEffectChange{
		StatusID: db.StatusEnergycoat,
		ActorID:  2000000,
		Active:   true,
	})

	if world.Player.Opt3State&db.Opt3Energycoat == 0 {
		t.Fatalf("opt3 state = 0x%08X, want energy coat bit", world.Player.Opt3State)
	}
	tint := actorStateTint(world.Player)
	if tint.R != 127 || tint.G != 127 || tint.B != 216 || tint.A != 255 {
		t.Fatalf("energy coat tint = %+v, want robr OPT3 tint", tint)
	}

	mode.applyStatusEffectChange(ctx, network.StatusEffectChange{
		StatusID: db.StatusEnergycoat,
		ActorID:  2000000,
		Active:   false,
	})
	if world.Player.Opt3State&db.Opt3Energycoat != 0 {
		t.Fatalf("opt3 state = 0x%08X, want energy coat cleared", world.Player.Opt3State)
	}
}

func TestTwoHandQuickenStatusUsesOpt3WithoutSightEffect(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20}
	sessionState := &session.Session{AccountID: 2000000, CharID: 150000}
	ctx := client.Context{Session: sessionState, World: world}
	mode := &WorldMode{}

	mode.applyStatusEffectChange(ctx, network.StatusEffectChange{
		StatusID: db.StatusTwohandquicken,
		ActorID:  2000000,
		Active:   true,
	})

	if world.Player.Opt3State&db.Opt3Quicken == 0 {
		t.Fatalf("opt3 state = 0x%08X, want quicken bit", world.Player.Opt3State)
	}
	if world.Player.EffectState&db.EffectStateSight != 0 {
		t.Fatalf("effect state = 0x%08X, want no Sight bit from quicken", world.Player.EffectState)
	}
	if tint := actorStateTint(world.Player); tint.B != 0 {
		t.Fatalf("quicken tint = %+v, want robr OPT3 quicken blue channel removed", tint)
	}
	if len(mode.effects().Active) != 0 {
		t.Fatalf("world effects = %+v, want no EF_SIGHT from quicken status", mode.effects().Active)
	}
}

func TestActorEntryPreservesExistingOpt3State(t *testing.T) {
	world := worldstate.New()
	world.UpsertActor(worldstate.Actor{ID: 300, X: 10, Y: 20, Job: 1, Opt3State: db.Opt3Quicken, HasState: true})
	ctx := client.Context{World: world}
	mode := &WorldMode{}

	mode.upsertNetworkActor(ctx, network.ActorEntry{
		ID:          300,
		X:           11,
		Y:           20,
		Job:         1,
		EffectState: db.EffectStateRuwach,
		HasState:    true,
	})

	actor := world.Actors[300]
	if actor.Opt3State&db.Opt3Quicken == 0 {
		t.Fatalf("actor opt3 state = 0x%08X, want quicken preserved", actor.Opt3State)
	}
	if actor.EffectState != db.EffectStateRuwach {
		t.Fatalf("actor effect state = 0x%08X, want packet effect state", actor.EffectState)
	}
}

func TestBerserkStatusSetsActorOpt3StateFromImportedTable(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20}
	sessionState := &session.Session{AccountID: 2000000, CharID: 150000}
	ctx := client.Context{Session: sessionState, World: world}
	mode := &WorldMode{}

	mode.applyStatusEffectChange(ctx, network.StatusEffectChange{
		StatusID: db.StatusBerserk,
		ActorID:  2000000,
		Active:   true,
	})

	if world.Player.Opt3State&db.Opt3Berserk == 0 {
		t.Fatalf("opt3 state = 0x%08X, want berserk bit", world.Player.Opt3State)
	}

	mode.applyStatusEffectChange(ctx, network.StatusEffectChange{
		StatusID: db.StatusBerserk,
		ActorID:  2000000,
		Active:   false,
	})
	if world.Player.Opt3State&db.Opt3Berserk != 0 {
		t.Fatalf("opt3 state = 0x%08X, want berserk cleared", world.Player.Opt3State)
	}
}

func TestCollectSceneActorEntriesPreservesLocalOpt3State(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{
		ID:        2000000,
		X:         10,
		Y:         20,
		Opt3State: db.Opt3Energycoat,
		HasState:  true,
	}
	sessionState := &session.Session{
		AccountID: 2000000,
		CharID:    150000,
		Selected:  session.Character{ID: 150000, Job: 2, Option: db.EffectStateCart1},
	}
	screen := render.NewFrame(800, 600)
	ctx := client.Context{Session: sessionState, World: world}
	projection := testutil.Projection(800, 600, 10, 20, 0)
	mode := &WorldMode{}

	entries := mode.collectSceneActorEntries(screen, ctx, projection)
	if len(entries) == 0 {
		t.Fatal("no actor entries collected")
	}
	if entries[0].Actor.Opt3State&db.Opt3Energycoat == 0 {
		t.Fatalf("entry opt3 state = 0x%08X, want energy coat preserved", entries[0].Actor.Opt3State)
	}
	if entries[0].Actor.EffectState&db.EffectStateCart1 == 0 {
		t.Fatalf("entry effect state = 0x%08X, want cart option merged", entries[0].Actor.EffectState)
	}
}

func TestVisibleStatusIconIDsAreKnownAndSorted(t *testing.T) {
	active := map[uint16]session.StatusEffect{
		99:           {ID: 99},
		12:           {ID: 12},
		10:           {ID: 10},
		db.StatusSit: {ID: db.StatusSit},
	}
	ids := gameui.VisibleStatusIconIDs(active)
	if !reflect.DeepEqual(ids, []uint16{10, 12}) {
		t.Fatalf("ids = %+v", ids)
	}
}

func TestApplyRemoteActorLookChangeUpdatesWorldActor(t *testing.T) {
	world := worldstate.New()
	world.UpsertActor(worldstate.Actor{ID: 300, Job: 0, Head: 1, Appearance: true})
	ctx := client.Context{
		Session: &session.Session{AccountID: 100, CharID: 200},
		World:   world,
	}

	if applyActorLookChange(ctx, network.ActorLookChange{ID: 300, Type: 4, Value: 7}) {
		t.Fatal("remote look change should not request local player sprite reload")
	}
	actor := world.Actors[300]
	if actor.HeadTop != 7 {
		t.Fatalf("remote head top = %d, want 7", actor.HeadTop)
	}
}

func TestDirectionFromDeltaUsesRathenaDirectionOrder(t *testing.T) {
	cases := []struct {
		name string
		toX  int
		toY  int
		want int
	}{
		{name: "north", toX: 10, toY: 11, want: 0},
		{name: "northwest", toX: 9, toY: 11, want: 1},
		{name: "west", toX: 9, toY: 10, want: 2},
		{name: "southwest", toX: 9, toY: 9, want: 3},
		{name: "south", toX: 10, toY: 9, want: 4},
		{name: "southeast", toX: 11, toY: 9, want: 5},
		{name: "east", toX: 11, toY: 10, want: 6},
		{name: "northeast", toX: 11, toY: 11, want: 7},
		{name: "long mostly north", toX: 11, toY: 20, want: 0},
		{name: "long mostly west", toX: 0, toY: 11, want: 2},
		{name: "long mostly south", toX: 11, toY: 0, want: 4},
		{name: "long mostly east", toX: 20, toY: 11, want: 6},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			if got := directionFromDelta(10, 10, tt.toX, tt.toY, 6); got != tt.want {
				t.Fatalf("directionFromDelta = %d, want %d", got, tt.want)
			}
		})
	}
	if got := directionFromDelta(10, 10, 10, 10, -1); got != 7 {
		t.Fatalf("stationary fallback = %d, want 7", got)
	}
}

func TestResolveTurnOnlyDirectionUsesHeadBeforeBody(t *testing.T) {
	head, body, ok := resolveTurnOnlyDirection(4, 0, 3)
	if !ok {
		t.Fatal("direction not resolved")
	}
	if head != 1 || body != 4 {
		t.Fatalf("one-octant left turn = head %d body %d, want head 1 body 4", head, body)
	}

	head, body, ok = resolveTurnOnlyDirection(4, 1, 3)
	if !ok {
		t.Fatal("direction not resolved")
	}
	if head != 0 || body != 3 {
		t.Fatalf("repeated left turn = head %d body %d, want head 0 body 3", head, body)
	}
}

func TestResolveTurnOnlyDirectionRotatesBodyForWideTurn(t *testing.T) {
	head, body, ok := resolveTurnOnlyDirection(4, 0, 1)
	if !ok {
		t.Fatal("direction not resolved")
	}
	if head != 1 || body != 2 {
		t.Fatalf("wide left turn = head %d body %d, want head 1 body 2", head, body)
	}

	head, body, ok = resolveTurnOnlyDirection(4, 0, 7)
	if !ok {
		t.Fatal("direction not resolved")
	}
	if head != 2 || body != 6 {
		t.Fatalf("wide right turn = head %d body %d, want head 2 body 6", head, body)
	}
}

func TestActorBillboardScreenScaleUsesProjectedReferenceHeight(t *testing.T) {
	if scene.ActorBillboardWorldHeightUnit != 5 {
		t.Fatalf("actor billboard world height = %.1f, want 5.0", scene.ActorBillboardWorldHeightUnit)
	}

	projection := testutil.Projection(800, 600, 10.5, 20.5, 5)

	scale := scene.ActorBillboardScreenScale(projection, 10.5, 20.5, 5)
	if math.Abs(scale-1.04) > 0.01 {
		t.Fatalf("camera billboard scale = %.3f, want about 1.04 at reference client default zoom", scale)
	}
}

func TestActorBillboardScreenScaleDoesNotChangeWithCameraPitch(t *testing.T) {
	const (
		width  = 800
		height = 600
		x      = 10.5
		y      = 20.5
		z      = 5
	)
	want := scene.ActorBillboardScreenScale(scene.NewProjection(width, height, x, y, z, 0, scene.CameraPitch(), 125), x, y, z)
	for _, pitch := range []float64{scene.DefaultCameraMinPitch, 215, 235, scene.DefaultCameraMaxPitch} {
		projection := scene.NewProjection(width, height, x, y, z, 0, pitch, 125)
		if got := scene.ActorBillboardScreenScale(projection, x, y, z); math.Abs(got-want) > 0.000001 {
			t.Fatalf("actor scale at pitch %.1f = %.6f, want %.6f", pitch, got, want)
		}
	}
}

func TestActorBillboardScreenScaleFollowsZoomAndPerspectiveDepth(t *testing.T) {
	const (
		width  = 800
		height = 600
		x      = 10.5
		y      = 20.5
		z      = 5
	)
	zoomedIn := scene.ActorBillboardScreenScale(scene.NewProjection(width, height, x, y, z, 0, 235, scene.DefaultCameraMinZoom), x, y, z)
	zoomedOut := scene.ActorBillboardScreenScale(scene.NewProjection(width, height, x, y, z, 0, 235, scene.DefaultCameraMaxZoom), x, y, z)
	if zoomedIn <= zoomedOut {
		t.Fatalf("actor scale did not follow zoom: zoomed in %.6f, zoomed out %.6f", zoomedIn, zoomedOut)
	}

	nearer := scene.ActorBillboardScreenScale(scene.NewProjection(width, height, x, y, z, 0, 235, 125), x, y-5, z)
	farther := scene.ActorBillboardScreenScale(scene.NewProjection(width, height, x, y, z, 0, 235, 125), x, y+5, z)
	if nearer <= farther {
		t.Fatalf("actor scale did not follow perspective depth: nearer %.6f, farther %.6f", nearer, farther)
	}
}

func TestActorAnchorOutsideViewportKeepsBodyVisibleBelowScreen(t *testing.T) {
	if actorAnchorOutsideViewport(400, 600+150, 800, 600, 1) {
		t.Fatal("actor should remain visible while its body can still overlap the bottom edge")
	}
	if !actorAnchorOutsideViewport(400, 600+260, 800, 600, 1) {
		t.Fatal("actor should be culled after the whole billboard is beyond the bottom edge")
	}
}

func TestActorViewportCullMarginsScaleWithZoom(t *testing.T) {
	_, _, _, bottom1 := actorViewportCullMargins(1)
	_, _, _, bottom2 := actorViewportCullMargins(2)
	if bottom2 <= bottom1 {
		t.Fatalf("bottom margin did not scale: %.1f <= %.1f", bottom2, bottom1)
	}
	left, right, top, bottom := actorViewportCullMargins(0)
	if left <= 0 || right <= 0 || top <= 0 || bottom <= 0 {
		t.Fatalf("invalid fallback margins: %.1f %.1f %.1f %.1f", left, right, top, bottom)
	}
}

func TestClickedAttackTargetPicksMobOnly(t *testing.T) {
	now := time.Now()
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 200, X: 10, Y: 20}
	world.UpsertActor(worldstate.Actor{
		ID:            300,
		X:             11,
		Y:             20,
		ObjectType:    6,
		HasObjectType: true,
	})
	ctx := client.Context{
		Session: &session.Session{AccountID: 100, CharID: 200},
		World:   world,
	}
	projection := testutil.Projection(800, 600, scene.CellCenter(10), scene.CellCenter(20), 0)
	npcPoint := projection.Project(scene.CellCenter(11), scene.CellCenter(20), 0)

	if actor, ok := clickedAttackTarget(ctx, projection, int(npcPoint.X), int(npcPoint.Y), now, nil); ok {
		t.Fatalf("npc should not be attack-clickable: %+v", actor)
	}

	world.UpsertActor(worldstate.Actor{
		ID:            400,
		X:             12,
		Y:             20,
		ObjectType:    5,
		HasObjectType: true,
	})
	mobPoint := projection.Project(scene.CellCenter(12), scene.CellCenter(20), 0)

	actor, ok := clickedAttackTarget(ctx, projection, int(mobPoint.X), int(mobPoint.Y), now, nil)
	if !ok {
		t.Fatal("expected mob hit")
	}
	if actor.ID != 400 {
		t.Fatalf("target id = %d, want 400", actor.ID)
	}
}

func TestDefaultHoveredUIRootDoesNotHideWorldNPCCursor(t *testing.T) {
	ctx, projection := cursorHoverTestContext(worldstate.Actor{
		ID:            300,
		X:             11,
		Y:             20,
		ObjectType:    scene.ActorObjectTypeNPC,
		HasObjectType: true,
	})
	mode := &WorldMode{}

	if got := mode.cursorDesiredAction(ctx, projection, time.Now()); got != cursorActionTalk {
		t.Fatalf("cursor action = %d, want talk", got)
	}
}

func TestDefaultHoveredUIRootDoesNotHideWorldWarpCursor(t *testing.T) {
	ctx, projection := cursorHoverTestContext(worldstate.Actor{
		ID:            300,
		X:             11,
		Y:             20,
		Job:           scene.ActorJobWarpPortal,
		ObjectType:    scene.ActorObjectTypeNPC,
		HasObjectType: true,
	})
	mode := &WorldMode{}

	if got := mode.cursorDesiredAction(ctx, projection, time.Now()); got != cursorActionWarp {
		t.Fatalf("cursor action = %d, want warp", got)
	}
}

func TestCursorActionDefaultOverPC(t *testing.T) {
	ctx, projection := cursorHoverTestContext(worldstate.Actor{
		ID:            300,
		X:             11,
		Y:             20,
		ObjectType:    scene.ActorObjectTypePC,
		HasObjectType: true,
	})
	mode := &WorldMode{}

	if got := mode.cursorDesiredAction(ctx, projection, time.Now()); got != cursorActionDefault {
		t.Fatalf("cursor action = %d, want default", got)
	}
}

func TestCursorActionClickOverVendingBoard(t *testing.T) {
	now := time.Now()
	actor := worldstate.Actor{
		ID:            300,
		X:             11,
		Y:             20,
		ObjectType:    scene.ActorObjectTypePC,
		HasObjectType: true,
		Vending:       true,
		VendingName:   "Fresh Fish",
	}
	ctx, projection := cursorHoverTestContext(actor)
	bounds, ok := vendingBoardActorBounds(ctx, projection, actor, now)
	if !ok {
		t.Fatal("expected vending board bounds")
	}
	ctx.Input.SetMousePosition(int(bounds.x+bounds.w/2), int(bounds.y+bounds.h/2))
	mode := &WorldMode{}

	if got := mode.cursorDesiredAction(ctx, projection, now); got != cursorActionClick {
		t.Fatalf("cursor action = %d, want click", got)
	}
}

func TestCursorMagnetOffsetFollowsTargetSnapSetting(t *testing.T) {
	now := time.Now()
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 200, X: 10, Y: 20}
	world.UpsertActor(worldstate.Actor{
		ID:            300,
		X:             11,
		Y:             20,
		ObjectType:    scene.ActorObjectTypeMob,
		HasObjectType: true,
	})
	projection := testutil.Projection(800, 600, scene.CellCenter(10), scene.CellCenter(20), 0)
	point := projection.Project(scene.CellCenter(11), scene.CellCenter(20), 0)
	inputState := input.NewState()
	inputState.SetMousePosition(int(point.X)+7, int(point.Y)+3)
	ctx := client.Context{
		Input:   inputState,
		Session: &session.Session{AccountID: 100, CharID: 200},
		World:   world,
	}
	mode := &WorldMode{}

	if dx, dy := mode.cursorMagnetOffset(ctx, projection, cursorActionAttack, now); dx != 0 || dy != 0 {
		t.Fatalf("disabled target snap offset = %.1f,%.1f, want zero", dx, dy)
	}
	ctx.Session.SnapTargets = true
	scale := scene.ActorBillboardScreenScale(projection, scene.CellCenter(11), scene.CellCenter(20), 0)
	targetX, targetY := actorPickBoundsCenter(float64(point.X), float64(point.Y), scale)
	inputState.SetMousePosition(int(math.Round(targetX))+7, int(math.Round(targetY))+3)
	wantDX := float64(inputState.MouseX) - targetX
	wantDY := float64(inputState.MouseY) - targetY
	dx, dy := mode.cursorMagnetOffset(ctx, projection, cursorActionAttack, now)
	if math.Abs(dx-wantDX) > 0.1 || math.Abs(dy-wantDY) > 0.1 {
		t.Fatalf("target snap offset = %.1f,%.1f, want %.1f,%.1f", dx, dy, wantDX, wantDY)
	}
	if got := mode.cursorDesiredAction(ctx, projection, now); got != cursorActionAttack {
		t.Fatalf("target snap cursor action = %d, want attack inside snap distance", got)
	}
	inputState.SetMousePosition(int(math.Round(targetX)), int(math.Round(targetY+actorCursorSnapRadius(scale)*1.05)))
	if dx, dy := mode.cursorMagnetOffset(ctx, projection, cursorActionAttack, now); dx != 0 || dy != 0 {
		t.Fatalf("outer target snap offset = %.1f,%.1f, want zero inside pick bounds but outside snap distance", dx, dy)
	}
	if got := mode.cursorDesiredAction(ctx, projection, now); got != cursorActionDefault {
		t.Fatalf("outer target snap cursor action = %d, want default inside pick bounds but outside snap distance", got)
	}
}

func TestCursorMagnetOffsetSnapsTargetSkillToSelf(t *testing.T) {
	now := time.Now()
	world := worldstate.New()
	world.Player = worldstate.Actor{
		ID:            200,
		X:             10,
		Y:             20,
		ObjectType:    scene.ActorObjectTypePC,
		HasObjectType: true,
	}
	projection := testutil.Projection(800, 600, scene.CellCenter(10), scene.CellCenter(20), 0)
	point := projection.Project(scene.CellCenter(10), scene.CellCenter(20), 0)
	inputState := input.NewState()
	inputState.SetMousePosition(int(point.X)+7, int(point.Y)+3)
	ctx := client.Context{
		Input: inputState,
		Session: &session.Session{
			AccountID:   100,
			CharID:      200,
			SnapTargets: true,
		},
		World: world,
	}
	mode := &WorldMode{pendingSkill: pendingSkillTarget{skill: session.Skill{
		ID:    db.SkillAMPotionpitcher,
		Type:  skillTargetFriend,
		Range: 9,
	}}}

	scale := scene.ActorBillboardScreenScale(projection, scene.CellCenter(10), scene.CellCenter(20), 0)
	targetX, targetY := actorPickBoundsCenter(float64(point.X), float64(point.Y), scale)
	inputState.SetMousePosition(int(math.Round(targetX))+7, int(math.Round(targetY))+3)
	wantDX := float64(inputState.MouseX) - targetX
	wantDY := float64(inputState.MouseY) - targetY
	dx, dy := mode.cursorMagnetOffset(ctx, projection, cursorActionTarget2, now)
	if math.Abs(dx-wantDX) > 0.1 || math.Abs(dy-wantDY) > 0.1 {
		t.Fatalf("self target skill snap offset = %.1f,%.1f, want %.1f,%.1f", dx, dy, wantDX, wantDY)
	}
}

func TestCursorMagnetOffsetSnapsTargetSkillToHomunculus(t *testing.T) {
	now := time.Now()
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 200, X: 10, Y: 20}
	world.UpsertActor(worldstate.Actor{
		ID:            300,
		X:             11,
		Y:             20,
		ObjectType:    scene.ActorObjectTypeHomunculus,
		HasObjectType: true,
	})
	projection := testutil.Projection(800, 600, scene.CellCenter(10), scene.CellCenter(20), 0)
	point := projection.Project(scene.CellCenter(11), scene.CellCenter(20), 0)
	inputState := input.NewState()
	inputState.SetMousePosition(int(point.X)+7, int(point.Y)+3)
	ctx := client.Context{
		Input: inputState,
		Session: &session.Session{
			AccountID:   100,
			CharID:      200,
			SnapTargets: true,
		},
		World: world,
	}
	mode := &WorldMode{pendingSkill: pendingSkillTarget{skill: session.Skill{
		ID:    db.SkillAMPotionpitcher,
		Type:  skillTargetFriend,
		Range: 9,
	}}}

	scale := scene.ActorBillboardScreenScale(projection, scene.CellCenter(11), scene.CellCenter(20), 0)
	targetX, targetY := actorPickBoundsCenter(float64(point.X), float64(point.Y), scale)
	inputState.SetMousePosition(int(math.Round(targetX))+7, int(math.Round(targetY))+3)
	wantDX := float64(inputState.MouseX) - targetX
	wantDY := float64(inputState.MouseY) - targetY
	dx, dy := mode.cursorMagnetOffset(ctx, projection, cursorActionTarget2, now)
	if math.Abs(dx-wantDX) > 0.1 || math.Abs(dy-wantDY) > 0.1 {
		t.Fatalf("homunculus target skill snap offset = %.1f,%.1f, want %.1f,%.1f", dx, dy, wantDX, wantDY)
	}
	if got := mode.cursorDesiredAction(ctx, projection, now); got != cursorActionTarget2 {
		t.Fatalf("homunculus target skill cursor action = %d, want target2 inside snap distance", got)
	}
	inputState.SetMousePosition(int(math.Round(targetX)), int(math.Round(targetY+actorCursorSnapRadius(scale)*1.05)))
	if got := mode.cursorDesiredAction(ctx, projection, now); got != cursorActionTarget {
		t.Fatalf("outer homunculus target skill cursor action = %d, want target inside pick bounds but outside snap distance", got)
	}
}

func TestCursorMagnetOffsetFollowsItemSnapSetting(t *testing.T) {
	now := time.Now()
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 200, X: 10, Y: 20}
	item := worldstate.FloorItem{ID: 9001, ItemID: 909, X: 11, Y: 20}
	world.UpsertItem(item)
	projection := testutil.Projection(800, 600, scene.CellCenter(10), scene.CellCenter(20), 0)
	x, y := floorItemWorldPosition(item)
	point := projection.Project(scene.CellCenter(x), scene.CellCenter(y), 0)
	inputState := input.NewState()
	inputState.SetMousePosition(int(point.X)+5, int(point.Y)+2)
	ctx := client.Context{
		Input:   inputState,
		Session: &session.Session{AccountID: 100, CharID: 200},
		World:   world,
	}
	mode := &WorldMode{}

	if dx, dy := mode.cursorMagnetOffset(ctx, projection, cursorActionPick, now); dx != 0 || dy != 0 {
		t.Fatalf("disabled item snap offset = %.1f,%.1f, want zero", dx, dy)
	}
	ctx.Session.SnapItems = true
	scale := scene.ActorBillboardScreenScale(projection, scene.CellCenter(x), scene.CellCenter(y), 0) * 0.42
	targetX, targetY := groundItemPickBoundsCenter(float64(point.X), float64(point.Y), scale)
	inputState.SetMousePosition(int(math.Round(targetX)), int(math.Round(targetY)))
	wantDX := float64(inputState.MouseX) - targetX
	wantDY := float64(inputState.MouseY) - targetY
	dx, dy := mode.cursorMagnetOffset(ctx, projection, cursorActionPick, now)
	if math.Abs(dx-wantDX) > 0.1 || math.Abs(dy-wantDY) > 0.1 {
		t.Fatalf("item snap offset = %.1f,%.1f, want %.1f,%.1f", dx, dy, wantDX, wantDY)
	}
	if got := mode.cursorDesiredAction(ctx, projection, now); got != cursorActionPick {
		t.Fatalf("item snap cursor action = %d, want pick inside snap distance", got)
	}
	inputState.SetMousePosition(int(math.Round(targetX+groundItemCursorSnapRadius(scale)*1.1)), int(math.Round(targetY)))
	if dx, dy := mode.cursorMagnetOffset(ctx, projection, cursorActionPick, now); dx != 0 || dy != 0 {
		t.Fatalf("outer item snap offset = %.1f,%.1f, want zero inside pick bounds but outside snap distance", dx, dy)
	}
	if got := mode.cursorDesiredAction(ctx, projection, now); got != cursorActionDefault {
		t.Fatalf("outer item snap cursor action = %d, want default inside pick bounds but outside snap distance", got)
	}
}

func TestSpriteBillboardScreenCenterUsesImageAndAnchor(t *testing.T) {
	billboard := &scene.Billboard{
		Image:   render.NewImage(40, 20),
		AnchorX: 10,
		AnchorY: 30,
	}
	x, y, ok := spriteBillboardScreenCenter(billboard, scene.Point2{X: 100, Y: 200}, 2)
	if !ok {
		t.Fatal("expected billboard center")
	}
	if x != 120 || y != 160 {
		t.Fatalf("center = %.1f,%.1f, want 120,160", x, y)
	}
}

func TestHidingUsesActorOptionsInsteadOfStatusIcons(t *testing.T) {
	ctx := client.Context{
		World:   worldstate.New(),
		Session: &session.Session{AccountID: 2000000, CharID: 150000},
	}
	ctx.World.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20}
	mode := &WorldMode{}
	// Main's status path also enqueues transition effects by design, so
	// only the visibility semantics are asserted here (upstream additionally
	// counts effects, which does not apply to the richer transition system).
	mode.applyStatusEffectChange(ctx, network.StatusEffectChange{StatusID: db.StatusHiding, ActorID: 2000000, Active: true, HasDuration: true, Duration: time.Second})
	if ctx.PlayerHasEffectState(db.EffectStateHide) {
		t.Fatal("status icon changed actor visibility")
	}
	change := network.ActorStateChange{ID: 2000000, EffectState: db.EffectStateHide}
	mode.applyActorStateChange(ctx, change)
	mode.applyActorStateChange(ctx, change)
	mode.removeExpiredStatusEffects(ctx.Session, time.Now().Add(time.Hour))
	if !ctx.PlayerHasEffectState(db.EffectStateHide) {
		t.Fatal("status icon expiry revealed the player")
	}
	change.EffectState = 0
	mode.applyActorStateChange(ctx, change)
	if ctx.PlayerHasEffectState(db.EffectStateHide) {
		t.Fatal("actor remained hidden after option cleared")
	}
}

func isLevel99AuraEffect(effectID int) bool {
	for _, auraEffectID := range effects.Level99AuraNormalEffects {
		if effectID == auraEffectID {
			return true
		}
	}
	return false
}

func TestWalkingSpriteStateIgnoresReadyFightAnimation(t *testing.T) {
	state := sprites.State{
		ActionFamily: scene.SpriteActionWalk,
		Moving:       true,
		Loop:         true,
	}

	if applyActorAnimationToSpriteState(&state, actorAnimation{actionFamily: sprites.ActionPCReadyFight, loop: true}, true) {
		t.Fatal("ready fight should not override walking state")
	}
	if state.ActionFamily != scene.SpriteActionWalk || !state.Moving {
		t.Fatalf("state = %+v, want still walking", state)
	}

	if !applyActorAnimationToSpriteState(&state, actorAnimation{actionFamily: sprites.ActionPCDeath}, true) {
		t.Fatal("death should override walking state")
	}
	if state.ActionFamily != sprites.ActionPCDeath || state.Moving {
		t.Fatalf("death state = %+v", state)
	}

	nonPC := sprites.State{ActionFamily: scene.SpriteActionWalk, Moving: true, Loop: true}
	if !applyActorAnimationToSpriteState(&nonPC, actorAnimation{actionFamily: sprites.ActionNonPCDeath}, false) {
		t.Fatal("non-pc death should override walking state")
	}
	if nonPC.ActionFamily != sprites.ActionNonPCDeath || nonPC.Moving {
		t.Fatalf("non-pc death state = %+v", nonPC)
	}
}
