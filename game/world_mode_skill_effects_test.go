package game

import (
	"encoding/binary"
	"image/color"
	"math"
	"reflect"
	"strconv"
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
	"github.com/kivutar/goro/world"
	worldstate "github.com/kivutar/goro/world"
)

func TestAcolyteCastAuraAndPoses(t *testing.T) {
	for _, skill := range []uint16{
		db.SkillALRuwach, db.SkillALPneuma, db.SkillALTeleport, db.SkillALWarp,
		db.SkillALHeal, db.SkillALIncagi, db.SkillALDecagi, db.SkillALHolywater,
		db.SkillALCrucis, db.SkillALAngelus, db.SkillALBlessing, db.SkillALCure, db.SkillALHolylight,
	} {
		ctx := testutil.KnightEffectContext()
		m := &WorldMode{}
		m.applySkillCastNotify(ctx, network.SkillCastNotify{
			SkillID: skill, SourceID: 2000000, TargetID: 300, Property: 6, DelayTime: 1000,
		})
		if !m.effects().HasActive(effects.BeginSpell6, 2000000, time.Now()) || m.effects().HasActive(effects.BeginSpell, 2000000, time.Now()) {
			t.Fatalf("skill %d: native cast aura must follow the holy property", skill)
		}
		pose := sprites.ActionPCReadyFight
		if skill == db.SkillALIncagi {
			pose = sprites.ActionPCSkill
		} else if skill == db.SkillALBlessing {
			pose = scene.SpriteActionIdle
		}
		if m.actorAnims[2000000].actionFamily != pose {
			t.Fatalf("skill %d: wrong casting pose", skill)
		}
	}
}

func TestAlchemistBerserkPotionAndCoupleCasting(t *testing.T) {
	ctx := testutil.ArcherEffectContext()
	ctx.Resources = testutil.RealDataManager(t)
	start := time.Unix(100, 0)
	projection := testutil.Projection(800, 600, 20.5, 20.5, 0)
	m := &WorldMode{}
	m.effects().AddAt(ctx, effects.ItemFast3, 300, start)
	if len(m.scheduledSounds) != 1 || !m.scheduledSounds[0].at.Equal(start.Add(18*db.EffectFrameDuration)) {
		t.Fatal("Berserk Potion sound must coincide with the tick-18 shake")
	}
	spec, _ := effects.Lookup(effects.ItemFast3)
	if spec.Components[0].StrFile != "버서크" {
		t.Fatal("Berserk Potion lost its STR")
	}
	for _, tick := range []int{0, 36, 148} {
		frame := render.NewFrame(800, 600)
		m.effects().DrawBerserkPotion(frame, ctx, spec.Components[1], m.effects().Active[0], start.Add(time.Duration(tick)*db.EffectFrameDuration))
		commands := reflect.ValueOf(frame).Elem().FieldByName("worldCommands")
		want := 1
		if tick == 36 {
			want = 2
		}
		if tick == 148 {
			want = 0
		}
		if commands.Len() != want {
			t.Fatalf("wrong pillar/streak lifetime at tick %d", tick)
		}
	}
	texture := render.NewImage(12, 24)
	texture.Fill(color.White)
	billboard := &scene.Billboard{Image: texture, AnchorX: 6, AnchorY: 24}
	entry := scene.ActorDrawEntry{Actor: ctx.World.Actors[300], WorldX: 20.5, WorldY: 20.5}
	for _, tick := range []int{0, 150, 151} {
		frame := render.NewFrame(800, 600)
		m.drawActorSpriteWithEffects3D(ctx, frame, projection, entry, billboard, 1, 1, 1, color.RGBA{255, 255, 255, 255}, render.BlendSourceOver, start.Add(time.Duration(tick)*db.EffectFrameDuration))
		draw := reflect.ValueOf(frame).Elem().FieldByName("worldBillboards").Index(0)
		green := 120.0 / 255
		if tick == 151 {
			green = 1
		}
		if math.Abs(draw.FieldByName("ColorG").Float()-green) > .001 {
			t.Fatal("Berserk Potion tint must end at tick 151")
		}
	}
	m = &WorldMode{}
	m.effects().AddAt(ctx, effects.CoupleCasting, 300, start)
	frame := render.NewFrame(800, 600)
	m.effects().Draw(frame, ctx, projection, start.Add(180*db.EffectFrameDuration), false)
	if m.effects().Active[0].BlueCasting.Tick != 181 {
		t.Fatal("standalone Couple Casting must use its declared lifetime")
	}
	commands := reflect.ValueOf(frame).Elem().FieldByName("worldCommands")
	if commands.Len() != 1 {
		t.Fatal("Couple Casting stopped after one second")
	}
	v := commands.Index(0).FieldByName("Vertices").Index(0)
	if math.Abs(v.FieldByName("ColorG").Float()-89.0/255) > .001 || math.Abs(v.FieldByName("ColorB").Float()-182.0/255) > .001 {
		t.Fatal("Couple Casting must use its native pink tint")
	}
}

func TestArcherConcentrationCast(t *testing.T) {
	ctx := testutil.ArcherEffectContext()
	m := &WorldMode{}
	m.applySkillCastNotify(ctx, network.SkillCastNotify{
		SkillID: db.SkillACConcentration, SourceID: 2000000, TargetID: 2000000, DelayTime: 4000, Property: 3,
	})
	if len(m.effects().Active) != 2 || m.effects().Active[1].EffectID != effects.IncAgiDex || m.effects().Active[1].Duration != 4*time.Second || len(m.scheduledSounds) != 1 || m.scheduledSounds[0].paths[0] != "effect\\ef_incagidex.wav" {
		t.Fatal("Concentration must use its AGI/DEX casting label and sound, regardless of element")
	}
	label := effects.AgilityLabel(false, 119, 240)
	if math.Abs(label.Y+125.55) > .001 || label.Alpha != 200 {
		t.Fatal("Concentration's label must follow the cast duration instead of ending after 100 ticks")
	}
	ctx.Resources = testutil.RealDataManager(t)
	frame := render.NewFrame(800, 600)
	effect := m.effects().Active[1]
	spec, _ := effects.Lookup(effects.IncAgiDex)
	projection := testutil.Projection(800, 600, 10, 20, 0)
	m.effects().DrawAgilityEffect(frame, ctx, projection, spec.Components[0], effect, 10.5, 20.5, 0, effect.Starts.Add(120*db.EffectFrameDuration))
	if reflect.ValueOf(frame).Elem().FieldByName("worldCommands").Len() == 0 {
		t.Fatal("Concentration's rising streaks must remain visible during a longer cast")
	}
}

func TestArcherArrowShowerSpreadAroundMovingTarget(t *testing.T) {
	ctx := testutil.ArcherEffectContext()
	now := time.Unix(100, 0)
	target := ctx.World.Actors[300]
	target.Moving = true
	target.FromX, target.FromY, target.ToX, target.ToY = 20, 20, 21, 19
	target.MoveStarted, target.MoveDuration = now.Add(-750*time.Millisecond), time.Second
	ctx.World.Actors[300] = target
	m := &WorldMode{}
	m.addSkillEffect(ctx, network.ActorActionNotify{SourceID: 2000000, TargetID: 300, SkillID: db.SkillACShower}, now, now)
	arrows := m.effects().Active[0].ArrowTargets
	if arrows[0].X != 21.25 || arrows[0].Z != 19.75 {
		t.Fatalf("center arrow must aim at the interpolated position: %+v", arrows[0])
	}
	var centerX, centerZ float64
	for _, arrow := range arrows[1:] {
		centerX += arrow.X / 8
		centerZ += arrow.Z / 8
	}
	if centerX != 21.5 || centerZ != 19.5 {
		t.Fatalf("spread center = %g,%g, want center of containing cell 21,19", centerX, centerZ)
	}
}

func TestArcherEffectsWithClientAssets(t *testing.T) {
	ctx := testutil.ArcherEffectContext()
	ctx.Resources = testutil.RealDataManager(t)
	m := &WorldMode{}
	start := time.Unix(100, 0)
	m.effects().AddArrowShower(ctx, 2000000, 20, 20, false, start)
	effect := m.effects().Active[0]
	spec, _ := effects.Lookup(effects.ArrowShower)
	projection := testutil.Projection(800, 600, 15, 20, 0)
	for _, tc := range []struct {
		delay time.Duration
		count int
	}{{-1, 0}, {48 * time.Millisecond, 9}, {192 * time.Millisecond, 0}} {
		frame := render.NewFrame(800, 600)
		m.effects().DrawArrowShower(frame, ctx, projection, spec.Components[0], effect, effect.Starts.Add(tc.delay))
		if got := reflect.ValueOf(frame).Elem().FieldByName("worldBillboards").Len(); got != tc.count {
			t.Fatalf("delay %s: drew %d arrows, want %d", tc.delay, got, tc.count)
		}
	}
	for _, id := range []int{effects.IncAgiDex, effects.Concentration} {
		if !m.effects().AddAt(ctx, id, 2000000, start) {
			t.Fatalf("could not add effect %d", id)
		}
	}
	for _, sound := range m.scheduledSounds {
		for _, path := range sound.paths {
			if _, err := ctx.Resources.ReadFile("data/wav/" + path); err != nil {
				t.Error(err)
			}
		}
	}
	frame := render.NewFrame(800, 600)
	m.effects().Draw(frame, ctx, projection, start.Add(30*db.EffectFrameDuration), false)
	if reflect.ValueOf(frame).Elem().FieldByName("worldCommands").Len() == 0 {
		t.Fatal("Concentration's rising streaks did not render")
	}
}

func TestSonicBlowCasterEffectsAndEightImpacts(t *testing.T) {
	ctx := testutil.AssassinEffectContext()
	m := &WorldMode{}
	starts := time.Unix(100, 0)
	hitAt := starts.Add(400 * time.Millisecond)
	action := network.ActorActionNotify{SkillID: db.SkillASSonicblow, SourceID: 2000000, TargetID: 300, Damage: 80, HitCount: 8}
	m.addSkillEffect(ctx, action, starts, hitAt)
	m.addSkillHitEffect(ctx, action, hitAt)
	if len(m.effects().Active) != 10 {
		t.Fatalf("effects = %+v", m.effects().Active)
	}
	for i, effect := range m.effects().Active {
		if i < 2 {
			if effect.EffectID != []int{effects.SonicBlow, effects.SonicBlow2}[i] || effect.ActorID != 2000000 || !effect.Starts.Equal(starts) {
				t.Fatalf("Sonic Blow cast effect on wrong actor or delayed: %+v", effect)
			}
		} else if effect.EffectID != effects.SonicBlowHit || effect.ActorID != 300 || !effect.Starts.Equal(hitAt.Add(time.Duration(i-2)*scene.MultiHitDelay)) {
			t.Fatalf("wrong Sonic Blow impact: %+v", effect)
		}
	}
}

func TestVenomKnifeFlightAndAfterimages(t *testing.T) {
	ctx := testutil.AssassinEffectContext()
	m := &WorldMode{}
	starts := time.Unix(100, 0)
	m.addSkillBeforeHitEffect(ctx, network.ActorActionNotify{SkillID: db.SkillASVenomknife, SourceID: 2000000, TargetID: 300}, starts)
	if len(m.effects().Active) != 1 {
		t.Fatal("Venom Knife must launch exactly one projectile")
	}
	effect := m.effects().Active[0]
	first, angle, alpha := effects.ThrowItemAt(effect, 1)
	if math.Abs(first.X-11.1) > 1e-9 || angle != scene.DegreesToRadians(15) || alpha != 255 {
		t.Fatalf("knife should immediately advance three native units: %+v, %g, %d", first, angle, alpha)
	}
	near := effect
	near.ProjectileDirection.X = 1.2
	_, _, nearAlpha := effects.ThrowItemAt(near, 10)
	_, _, farAlpha := effects.ThrowItemAt(effect, 10)
	if nearAlpha >= farAlpha {
		t.Fatal("flight time must depend on distance")
	}
	if _, _, alpha := effects.ThrowItemAt(effect, 30); alpha != 0 {
		t.Fatal("knife must fade after arriving")
	}
	m.mapScene.Textures = map[string]*render.Image{"__effectfile_유저인터페이스/item/베넘나이프.bmp|blackkey=true": render.NewImage(8, 8)}
	frame := render.NewFrame(800, 600)
	projection := testutil.Projection(800, 600, 14, 20, 0)
	m.effects().Draw(frame, ctx, projection, starts.Add(6*db.EffectFrameDuration), false)
	commands := reflect.ValueOf(frame).Elem().FieldByName("worldBillboards")
	if commands.Len() != 4 {
		t.Fatalf("knife and three afterimages: got %d commands", commands.Len())
	}
	for i, process := range []int{7, 6, 4, 2} {
		want, _, _ := effects.ThrowItemAt(effect, process)
		billboard := commands.Index(i)
		if math.Abs(billboard.FieldByName("Center").Index(0).Float()-want.X) > 1e-5 {
			t.Fatalf("afterimage %d did not use the previous two-tick sample", i)
		}
	}
}

func TestGrimtoothCastPackets(t *testing.T) {
	for _, tc := range []struct {
		name   string
		damage int32
	}{
		{"normal", -30000},
		{"hidden WoE damage", 1},
		{"hallucination", 1234},
		{"hallucination zero", 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := testutil.AssassinEffectContext()
			ctx.World.Actors[301] = world.Actor{ID: 301, X: 16, Y: 21, Job: 1002}
			ctx.World.Actors[2000001] = world.Actor{ID: 2000001, X: 10, Y: 21, Job: db.JobAssassin}
			m := &WorldMode{}
			starts := time.Unix(100, 0)
			action := network.ActorActionNotify{SkillID: db.SkillASGrimtooth, SkillLevel: 5,
				SourceID: 2000000, TargetID: 300, ServerTick: 1000, SourceSpeed: 500,
				Damage: tc.damage, HitCount: 1, Action: network.ActorActionSkill}
			m.addSkillEffect(ctx, action, starts, starts.Add(500*time.Millisecond))
			if len(m.effects().Active) != 1 || len(m.scheduledSounds) != 1 {
				t.Fatalf("cast must start a trail and sound regardless of damage: %+v", m.effects().Active)
			}
			// The preamble may refer to a splash victim, not the primary target.
			action.TargetID, action.Damage, action.TargetSpeed = 301, 30, 200
			m.addSkillEffect(ctx, action, starts.Add(time.Millisecond), starts.Add(500*time.Millisecond))
			action.Action = 5 // splash damage must never launch a trail
			m.addSkillEffect(ctx, action, starts.Add(2*time.Millisecond), starts.Add(500*time.Millisecond))
			if len(m.effects().Active) != 1 || len(m.scheduledSounds) != 1 {
				t.Fatal("damage packets duplicated the trail or sound")
			}
			action.Action = network.ActorActionSkill
			action.SourceID = 2000001
			m.addSkillEffect(ctx, action, starts, starts)
			action.SourceID, action.ServerTick = 2000000, 1500
			m.addSkillEffect(ctx, action, starts.Add(500*time.Millisecond), starts.Add(time.Second))
			if len(m.effects().Active) != 3 || len(m.scheduledSounds) != 3 {
				t.Fatal("another caster or subsequent cast was suppressed while the first trail was active")
			}
			m = &WorldMode{}
			action.Action = 5
			m.addSkillEffect(ctx, action, starts, starts)
			if len(m.effects().Active) != 0 || len(m.scheduledSounds) != 0 {
				t.Fatal("a splash-only notification started a trail")
			}
			action.Action, action.SourceID = network.ActorActionSkill, 999
			m.addSkillEffect(ctx, action, starts, starts)
			if len(m.effects().Active) != 0 || len(m.scheduledSounds) != 0 {
				t.Fatal("a missing caster started a trail at the victim")
			}
		})
	}
}

func TestGrimtoothPreambleAndFixedTrail(t *testing.T) {
	ctx := testutil.AssassinEffectContext()
	m := &WorldMode{}
	starts := time.Unix(100, 0)
	action := network.ActorActionNotify{SkillID: db.SkillASGrimtooth, SourceID: 2000000, TargetID: 300, Damage: -30000, Action: network.ActorActionSkill}
	m.addSkillEffect(ctx, action, starts, starts.Add(time.Second))
	action.Damage = 30
	m.addSkillEffect(ctx, action, starts, starts)
	m.addSkillHitEffect(ctx, action, starts)
	action.Action = 5 // secondary victim's damage-only packet
	m.addSkillEffect(ctx, action, starts, starts)
	if len(m.effects().Active) != 2 || m.effects().Active[0].EffectID != effects.Grimtooth || m.effects().Active[1].EffectID != effects.GrimtoothAtk {
		t.Fatalf("one cast must produce one trail and one impact per victim: %+v", m.effects().Active)
	}
	trail := m.effects().Active[0]
	if trail.ActorID != 2000000 || m.scheduledSounds[0].actorID != 2000000 || trail.ActorOrigin.X != 10.5 || trail.ActorOrigin.Z != 20.5 || math.Abs(trail.ProjectileDirection.X-.4) > 1e-9 || !trail.Starts.Equal(starts) {
		t.Fatalf("trail does not start at caster toward target: %+v", trail)
	}
	m.mapScene.Textures = map[string]*render.Image{"__effectfile_effect/stone.bmp|blackkey=true": render.NewImage(8, 8)}
	m.effects().Active = m.effects().Active[:1]
	projection := testutil.Projection(800, 600, 14, 20, 0)
	draw := func() reflect.Value {
		frame := render.NewFrame(800, 600)
		m.effects().Draw(frame, ctx, projection, starts.Add(30*db.EffectFrameDuration), false)
		commands := reflect.ValueOf(frame).Elem().FieldByName("worldCommands")
		if commands.Len() != 1 {
			t.Fatalf("expected one batched trail, got %d commands", commands.Len())
		}
		return commands.Index(0).FieldByName("Vertices")
	}
	before := draw()
	if before.Len() != 11*12 {
		t.Fatalf("tick 30 should have eleven horns: %d vertices", before.Len())
	}
	ctx.World.Player.X = 80
	delete(ctx.World.Actors, 300)
	after := draw()
	for i := 0; i < before.Len(); i++ {
		for _, axis := range []string{"X", "Y", "Z"} {
			if before.Index(i).FieldByName(axis).Float() != after.Index(i).FieldByName(axis).Float() {
				t.Fatal("moving actors pulled an already emitted Grimtooth trail")
			}
		}
	}
	spike := effects.GrimtoothSpike(trail, 0, 6)
	if spike.Origin != effects.GrimtoothSpike(trail, 0, 20).Origin {
		t.Fatal("trail spike kept moving after the native seven updates")
	}
	if effects.GrimtoothSpike(trail, 0, 39).Alpha != 0 {
		t.Fatal("trail spike did not finish fading")
	}
	impact := effects.Instance{EffectID: effects.GrimtoothAtk}
	if effects.GrimtoothSpike(impact, 0, 10).Origin != effects.GrimtoothSpike(impact, 0, 989).Origin || effects.GrimtoothSpike(impact, 0, 999).Alpha != 0 {
		t.Fatal("impact spikes should stop after eleven ticks and fade at the end")
	}
}

func TestBlacksmithHammerFallAndCastPoses(t *testing.T) {
	ctx := testutil.MerchantEffectContext()
	ctx.World.Player.Job, ctx.World.Player.Weapon = db.JobBlacksmith, db.WeaponAxe
	ctx.Session.Selected.Job, ctx.Session.Selected.Weapon = db.JobBlacksmith, db.WeaponAxe
	ctx.Session.Selected.ID = 100
	start := time.Unix(100, 0)
	m := &WorldMode{}
	m.startSkillSourceCastAnimation(ctx, 2000000, db.SkillBSHammerfall, time.Second, start)
	if m.actorAnims[2000000].actionFamily != attackActionFamilyForActorWithResources(ctx.Resources, ctx.World.Player) {
		t.Fatal("Hammer Fall must cast with the equipped weapon's attack pose")
	}
	m.applyGroundSkillNotify(ctx, network.GroundSkillNotify{SkillID: db.SkillBSHammerfall, SourceID: 2000000, X: 15, Y: 20})
	m.applyGroundSkillNotify(ctx, network.GroundSkillNotify{SkillID: db.SkillBSHammerfall, SourceID: 2000000, X: 15, Y: 20})
	if len(m.effects().Active) != 2 || m.actorAnims[2000000].actionFamily != sprites.ActionPCSkill {
		t.Fatal("each Hammer Fall must play its release pose and create a fresh ground impact")
	}
	for _, e := range m.effects().Active {
		if e.EffectID != effects.CrashEarth || e.ActorID != 0 || e.X != 15 || e.Y != 20 {
			t.Fatal("Hammer Fall must stay at its target cell")
		}
	}
	for _, tc := range []struct {
		skill uint16
		aura  int
	}{
		{db.SkillBSHammerfall, effects.BashBegin}, {db.SkillBSAdrenaline, effects.BashBegin},
		{db.SkillBSMaximize, effects.BashBegin}, {db.SkillBSRepairweapon, effects.BeginSpell3},
	} {
		m := &WorldMode{}
		m.addSkillCastEffects(ctx, tc.skill, 3, 2000000, 300, 0, 0, time.Second, start, "test")
		if len(m.effects().Active) != 3 || m.effects().Active[2].EffectID != tc.aura {
			t.Fatalf("skill %d has the wrong casting aura", tc.skill)
		}
		for _, s := range m.scheduledSounds {
			if s.paths[0] == "effect\\black_adrenalinerush_a.wav" || s.paths[0] == "effect\\black_maximize_power_circle.wav" {
				t.Fatal("buff audio must wait for the success packet")
			}
		}
	}
}

func TestBlacksmithEffectsWithClientAssets(t *testing.T) {
	ctx := testutil.MerchantEffectContext()
	ctx.Resources = testutil.RealDataManager(t)
	start := time.Unix(100, 0)
	projection := testutil.Projection(800, 600, 10, 20, 0)
	for _, tc := range []struct {
		id    int
		ticks []int
	}{
		{effects.HasteUp, []int{0, 30}}, {effects.RepairWeapon, []int{20, 55}},
		{effects.CrashEarth, []int{30}}, {effects.WeaponPerfect, []int{0}},
		{effects.MaximizePower, []int{0, 40, 47, 65}}, {effects.Overthrust, []int{0, 0}},
	} {
		m := &WorldMode{}
		m.effects().AddAt(ctx, tc.id, 2000000, start)
		if len(m.scheduledSounds) != len(tc.ticks) {
			t.Fatalf("effect %d: incorrect sound count", tc.id)
		}
		for i, sound := range m.scheduledSounds {
			if sound.at.Sub(start) != time.Duration(tc.ticks[i])*db.EffectFrameDuration {
				t.Fatalf("effect %d sound %d has incorrect timing", tc.id, i)
			}
			if _, err := ctx.Resources.ReadFile("data/wav/" + sound.paths[0]); err != nil {
				t.Fatal(err)
			}
		}
		frame := render.NewFrame(800, 600)
		m.effects().Draw(frame, ctx, projection, start.Add(20*db.EffectFrameDuration), false)
		value := reflect.ValueOf(frame).Elem()
		if value.FieldByName("commands").Len()+value.FieldByName("worldCommands").Len()+value.FieldByName("worldBillboards").Len() == 0 {
			t.Fatalf("effect %d drew nothing with client assets", tc.id)
		}
		if tc.id == effects.HasteUp {
			if got := value.FieldByName("worldBillboards").Len(); got != 4 {
				t.Fatalf("Adrenaline Rush needs four orbiting particles, got %d", got)
			}
			center := value.FieldByName("worldBillboards").Index(0).FieldByName("Center")
			if math.Abs(center.Index(1).Float()-(.07+(2.1+.462)/5)) > .001 {
				t.Fatal("Adrenaline particles have the wrong rising trajectory")
			}
		}
		if tc.id == effects.CrashEarth {
			spec, _ := effects.Lookup(tc.id)
			if spec.CameraShakeDelay != 35*db.EffectFrameDuration || m.scheduledSounds[0].paths[0] != "effect\\wizard_fire_pillar_b.wav" {
				t.Fatal("Hammer Fall must use the native impact sound and quake tick")
			}
		}
	}
}

func TestBlueCastingFlaresDuringCastAndEndsWithIt(t *testing.T) {
	ctx := testutil.SoulLinkerEffectContext()
	m := &WorldMode{}
	soulLinkerTestTextures(m)
	start := time.Unix(100, 0)
	for _, duration := range []int{6, 60, 180} {
		m.effects().Active = nil
		m.effects().AddBetweenDuration(ctx, effects.BlueCasting, 300, 0, start, time.Duration(duration)*db.EffectFrameDuration)
		e := m.effects().Active[0]
		duration = max(duration, 56)
		if e.Expires.Sub(e.Starts) != time.Duration(duration)*db.EffectFrameDuration {
			t.Fatal("short casts need the native minimum flare duration")
		}
		spec, _ := effects.Lookup(effects.BlueCasting)
		frame := render.NewFrame(800, 600)
		m.effects().DrawBlueCastingEffect(frame, ctx, spec.Components[0], e, 18.5, 20.5, 0, start.Add(15*db.EffectFrameDuration))
		if reflect.ValueOf(frame).Elem().FieldByName("worldCommands").Len() != 1 {
			t.Fatal("missing cast rings")
		}
		one, many := &effects.CastingState{}, &effects.CastingState{}
		one.Advance(duration, duration)
		for i := 0; i <= duration; i++ {
			many.Advance(i, duration)
		}
		if !reflect.DeepEqual(one, many) {
			t.Fatal("cast rings depend on render rate")
		}
		frame = render.NewFrame(800, 600)
		m.effects().Draw(frame, ctx, testutil.Projection(800, 600, 18.5, 20.5, 0), start.Add(time.Duration(duration+1)*db.EffectFrameDuration), false)
		if reflect.ValueOf(frame).Elem().FieldByName("worldCommands").Len() != 0 || len(m.effects().Active) != 0 {
			t.Fatal("rings survived the cast")
		}
	}
}

func TestAsuraNativeGlyphTimingAndPalette(t *testing.T) {
	ctx := testutil.MonkEffectContext()
	ctx.Resources = testutil.RealDataManager(t)
	start := time.Unix(100, 0)
	projection := testutil.Projection(800, 600, 12, 20, 0)
	for _, id := range []int{328, 510} {
		m := &WorldMode{}
		m.effects().AddAt(ctx, id, 300, start)
		effect := m.effects().Active[0]
		spec, _ := effects.Lookup(id)
		if len(spec.Components) != 2 || spec.Duration != 350*db.EffectFrameDuration || len(m.scheduledSounds) != 1 {
			t.Fatal("Asura needs native rings, glyphs, and one casting sound")
		}
		for _, tick := range []int{0, 1, 21, 291} {
			frame := render.NewFrame(800, 600)
			m.effects().DrawEffectGlyphs(frame, ctx, projection, spec.Components[1], effect, start.Add(time.Duration(tick)*db.EffectFrameDuration))
			draws := reflect.ValueOf(frame).Elem().FieldByName("worldBillboards")
			count := 0
			if tick == 1 {
				count = 10
			}
			if tick == 21 {
				count = 20
			}
			shade := 0.0
			if id == 510 {
				count += count / 10
				shade = 10.0 / 255
			}
			if draws.Len() != count {
				t.Fatalf("Asura %d tick %d: %d glyph layers, want %d", id, tick, draws.Len(), count)
			}
			for i := 0; i < draws.Len(); i++ {
				draw := draws.Index(i)
				if math.Abs(draw.FieldByName("ColorR").Float()-shade) > .00001 || draw.FieldByName("Options").FieldByName("Blend").Int() != int64(render.BlendSourceOver) || math.Abs(draw.FieldByName("Center").Index(1).Float()-6.07) > .0001 {
					t.Fatal("Asura must use the 2008 dark glyph palette, normal alpha, and native height")
				}
			}
		}
	}
}

func TestChampionEffectsRenderWithNativeAssets(t *testing.T) {
	ctx := testutil.MonkEffectContext()
	ctx.Resources = testutil.RealDataManager(t)
	ctx.World.Player.Job, ctx.Session.Selected.Job = db.JobMonkH, db.JobMonkH
	start := time.Unix(100, 0)
	projection := testutil.Projection(800, 600, 12, 20, 0)
	for _, sex := range []uint8{0, 1} {
		view, status := sprites.LoadPlayer(ctx.Resources, ctx.Session.Selected, sex, false)
		if view == nil {
			t.Fatalf("Champion sprite missing: %s", status)
		}
		for dir := 0; dir < 8; dir++ {
			_, action, ok := sprites.ResolveAction(view.Body.Act, sprites.ActionPCSkill, dir)
			if !ok || len(action.Animations) <= 4 {
				t.Fatal("Champion is missing the fixed skill poses")
			}
		}
	}
	for _, tc := range []struct{ id, tick, end int }{{376, 35, 100}, {377, 10, 50}, {402, 10, 80}, {512, 30, 125}, {263, 10, 65}} {
		m := &WorldMode{}
		m.effects().AddBetween(ctx, tc.id, 300, 2000000, start)
		for _, sound := range m.scheduledSounds {
			if _, err := ctx.Resources.ReadFile("data/wav/" + sound.paths[0]); err != nil {
				t.Fatal(err)
			}
		}
		for _, tick := range []int{tc.tick, tc.end} {
			frame := render.NewFrame(800, 600)
			m.effects().Draw(frame, ctx, projection, start.Add(time.Duration(tick)*db.EffectFrameDuration), false)
			commands := reflect.ValueOf(frame).Elem().FieldByName("worldCommands")
			if (commands.Len() > 0) != (tick == tc.tick) {
				t.Fatalf("effect %d tick %d: commands %d", tc.id, tick, commands.Len())
			}
			if tc.id == 512 && tick == tc.tick {
				vertices := commands.Index(0).FieldByName("Vertices")
				if vertices.Len() != 32*4 {
					t.Fatal("Chain Crush needs thirty-two flat streaks")
				}
				for i := 0; i < vertices.Len(); i++ {
					if math.Abs(vertices.Index(i).FieldByName("Y").Float()-1.27) > .0001 {
						t.Fatal("Chain Crush streaks must be six native units above ground")
					}
				}
			}
		}
	}
}

func TestClownGypsyEffectsWithClientAssets(t *testing.T) {
	ctx := testutil.ArcherEffectContext()
	ctx.Resources = testutil.RealDataManager(t)
	start := time.Unix(100, 0)
	projection := testutil.Projection(800, 600, 15, 20, 0)
	ids := []int{effects.TripleAttack3, effects.Moonlit, effects.MarionetteLink, effects.BottomHermode}
	for id := effects.TarotCard1; id <= effects.TarotCard14; id++ {
		ids = append(ids, id)
	}
	for _, id := range ids {
		m := &WorldMode{}
		m.effects().AddBetween(ctx, id, 300, 2000000, start)
		for _, sound := range m.scheduledSounds {
			if _, err := ctx.Resources.ReadFile("data/wav/" + sound.paths[0]); err != nil {
				t.Fatal(err)
			}
			if id == effects.TripleAttack3 && sound.at.Sub(start) != 30*db.EffectFrameDuration {
				t.Fatal("Arrow Vulcan sound must wait 30 ticks")
			}
		}
		frame := render.NewFrame(800, 600)
		m.effects().Draw(frame, ctx, projection, start.Add(65*db.EffectFrameDuration), false)
		commands := reflect.ValueOf(frame).Elem().FieldByName("worldCommands")
		billboards := reflect.ValueOf(frame).Elem().FieldByName("worldBillboards")
		if commands.Len()+billboards.Len() == 0 {
			t.Fatalf("Clown effect %d drew nothing with actual client assets", id)
		}
		if id >= effects.TarotCard1 && id <= effects.TarotCard14 {
			cmd := billboards.Index(0)
			if math.Abs(cmd.FieldByName("Center").Index(1).Float()-6.07) > .001 || len(m.scheduledSounds) != 0 {
				t.Fatal("Tarot card must hover at native height and remain silent")
			}
		}
	}
	for _, skill := range []uint16{db.SkillCGMoonlit, db.SkillCGHermode, db.SkillCGLongingfreedom, db.SkillCGTarotcard} {
		for _, sound := range db.SkillEffects[skill].SFX {
			if _, err := ctx.Resources.ReadFile("data/wav/" + sound); err != nil {
				t.Fatal(err)
			}
		}
	}
}

func TestCreatorEffectRenderingWithClientAssets(t *testing.T) {
	ctx := testutil.ArcherEffectContext()
	ctx.Resources = testutil.RealDataManager(t)
	start := time.Unix(100, 0)
	projection := testutil.Projection(800, 600, 20, 20, 0)
	m := &WorldMode{}
	m.effects().AddBetween(ctx, effects.ThrowItem4, 300, 2000000, start)
	for _, tc := range []struct{ tick, billboards int }{{4, 0}, {5, 1}, {9, 3}, {10, 4}, {20, 8}, {60, 0}} {
		frame := render.NewFrame(800, 600)
		m.effects().Draw(frame, ctx, projection, start.Add(time.Duration(tc.tick)*db.EffectFrameDuration), false)
		billboards := reflect.ValueOf(frame).Elem().FieldByName("worldBillboards")
		if billboards.Len() != tc.billboards {
			t.Fatalf("bottle frame %d: got %d billboards, want %d", tc.tick, billboards.Len(), tc.billboards)
		}
		for i := 0; i < billboards.Len(); i++ {
			if billboards.Index(i).FieldByName("ColorG").Float() != 1 {
				t.Fatal("Acid Demonstration bottles must be untinted")
			}
		}
	}
	for variant, id := range []int{effects.Slim, effects.Slim2, effects.Slim3} {
		m := &WorldMode{}
		m.effects().AddCellLifetime(ctx, id, 0, 22, 21, start, 0, false)
		frame := render.NewFrame(800, 600)
		m.effects().Draw(frame, ctx, projection, start.Add(25*db.EffectFrameDuration), false)
		commands := reflect.ValueOf(frame).Elem().FieldByName("worldCommands")
		billboards := reflect.ValueOf(frame).Elem().FieldByName("worldBillboards")
		if billboards.Len() != 4 || commands.Len() != 1 {
			t.Fatal("potion burst needs its falling icon, three afterimages, and a ground wave")
		}
		wantVertices := 4
		if variant == 0 {
			wantVertices = 8
		}
		if commands.Index(0).FieldByName("Vertices").Len() != wantVertices {
			t.Fatal("red potion must draw its ground wave twice")
		}
		m.effects().Draw(render.NewFrame(800, 600), ctx, projection, start.Add(5*time.Second), false)
		if len(m.effects().Active) != 0 {
			t.Fatal("potion burst failed to expire")
		}
	}
	m = &WorldMode{}
	m.effects().AddAt(ctx, effects.AcidDemon, 300, start)
	actor := ctx.World.Actors[300]
	actor.X = 60
	ctx.World.Actors[300] = actor
	for _, tick := range []int{19, 43, 44} {
		frame := render.NewFrame(800, 600)
		m.effects().Draw(frame, ctx, projection, start.Add(time.Duration(tick)*db.EffectFrameDuration), false)
		commands := reflect.ValueOf(frame).Elem().FieldByName("worldCommands")
		if tick == 44 {
			if commands.Len() != 0 {
				t.Fatal("Acid Demonstration rings survived their fade")
			}
			continue
		}
		if commands.Len() != 1 || commands.Index(0).FieldByName("Vertices").Len() != 168 {
			t.Fatal("Acid Demonstration must batch four rings using the original 20-segment meshes")
		}
		v := commands.Index(0).FieldByName("Vertices").Index(0)
		if math.Abs(v.FieldByName("X").Float()-20.5) > 1 || math.Abs(v.FieldByName("ColorG").Float()-50.0/255) > .001 {
			t.Fatal("impact rings must remain at the original victim position with their native magenta tint")
		}
	}
	texture := m.mapScene.Textures["__effectfile_effect/ring_black.tga|blackkey=false"]
	if texture == nil {
		t.Fatal("Acid Demonstration must retain the TGA's black pixels")
	}
	source, _, err := res.LoadImageExact(ctx.Resources, []string{"data/texture/effect/ring_black.tga"})
	if err != nil {
		t.Fatal(err)
	}
	partialAlpha := 0
	for y := 0; y < source.Bounds().Dy(); y++ {
		for x := 0; x < source.Bounds().Dx(); x++ {
			want := color.NRGBAModel.Convert(source.At(x, y)).(color.NRGBA)
			got := texture.RGBA().RGBAAt(x, y)
			if got.R != want.R || got.G != want.G || got.B != want.B || got.A != want.A {
				t.Fatalf("impact texture changed straight-alpha pixel at %d,%d: %v != %v", x, y, got, want)
			}
			if want.A > 0 && want.A < 255 {
				partialAlpha++
			}
		}
	}
	if partialAlpha == 0 {
		t.Fatal("fixture lacks the TGA's translucent edges")
	}
	spec, _ := effects.Lookup(effects.AcidDemon)
	if spec.CameraShakeDelay != 5*db.EffectFrameDuration || len(m.scheduledSounds) != 1 || !m.scheduledSounds[0].at.Equal(start) {
		t.Fatal("impact sound must start immediately and quake at tick five")
	}
}

func TestGunslingerNativePosesAndCasting(t *testing.T) {
	ctx := testutil.ArcherEffectContext()
	ctx.World.Player.Job = db.JobGunslinger
	for _, tc := range []struct {
		skill       uint16
		first, last int
		family      int
		duration    time.Duration
	}{
		{500, 4, 5, sprites.ActionPCSkill, 500 * time.Millisecond},
		{501, 0, 3, sprites.ActionPCSkill, time.Second}, {504, 0, 3, sprites.ActionPCSkill, time.Second},
		{505, 0, 3, sprites.ActionPCSkill, time.Second}, {506, 0, 3, sprites.ActionPCSkill, time.Second},
		{516, 0, 3, sprites.ActionPCSkill, time.Second}, {517, 0, 3, sprites.ActionPCSkill, time.Second},
		{508, 0, 2, sprites.ActionPCAttack1, 500 * time.Millisecond}, {521, 0, 2, sprites.ActionPCAttack1, 500 * time.Millisecond},
	} {
		for _, seed := range []int64{0, 1} {
			anim := skillAction(tc.skill).actorAnimationForActorWithResources(nil, ctx.World.Player, time.Unix(100, seed), time.Second)
			var total time.Duration
			for i := tc.first; i <= tc.last; i++ {
				frame := i
				if tc.last == 3 && seed == 1 {
					frame = 3 - i
				}
				if !anim.hasFixedMotion || anim.fixedMotion != frame || anim.actionFamily != tc.family {
					t.Fatalf("skill %d pose %d = %+v", tc.skill, frame, anim)
				}
				total += anim.duration
				if i < tc.last {
					if anim.next == nil {
						t.Fatal("missing next pose")
					}
					anim = *anim.next
				}
			}
			if total != tc.duration || anim.next != nil {
				t.Fatalf("skill %d lifetime=%s want %s", tc.skill, total, tc.duration)
			}
		}
	}
	for _, id := range []uint16{502, 503, 507, 512, 513, 514, 515, 518, 519, 520} {
		if skillAction(id).action != skillActorActionAttack {
			t.Fatalf("skill %d must use weapon attack", id)
		}
	}
	for _, id := range []uint16{503, 504, 505, 512, 513, 514, 515, 517, 518, 519, 520, 521} {
		m := &WorldMode{}
		now := time.Unix(100, 0)
		m.startSkillSourceCastAnimation(ctx, 2000000, id, 2*time.Second, now)
		a := m.actorAnims[2000000]
		family, frame := sprites.ActionPCSkill, 1
		if id >= 518 {
			family, frame = sprites.ActionPCAttack1, 0
		}
		if a.actionFamily != family || !a.hasFixedMotion || a.fixedMotion != frame || a.duration != 500*time.Millisecond {
			t.Fatalf("skill %d cast pose %+v", id, a)
		}
		m.addSkillCastEffects(ctx, id, 0, 2000000, 300, 0, 0, time.Second, now, "test")
		found := false
		for _, e := range m.effects().Active {
			found = found || e.EffectID == effects.BlueCasting
		}
		if !found {
			t.Fatalf("skill %d missing blue casting rings", id)
		}
	}
}

func TestGunslingerAssetsAndTiming(t *testing.T) {
	ctx := testutil.ArcherEffectContext()
	ctx.Resources = testutil.RealDataManager(t)
	m := &WorldMode{}
	start := time.Unix(100, 0)
	projection := testutil.Projection(800, 600, 10, 20, 0)
	for _, tc := range []struct{ id, tick int }{{456, 30}, {584, 3}, {623, 30}, {627, 40}, {628, 30}, {629, 30}, {637, 3}, {638, 3}, {639, 3}, {640, 3}, {641, 3}, {642, 3}, {643, 3}, {644, 3}, {645, 3}, {647, 3}, {648, 32}} {
		m.effects().Active = nil
		m.scheduledSounds = nil
		if !m.effects().AddBetween(ctx, tc.id, 300, 2000000, start) {
			t.Fatalf("missing effect %d", tc.id)
		}
		e := m.effects().Active[0]
		e.HitCount = 8
		spec, _ := effects.Lookup(tc.id)
		frame := render.NewFrame(800, 600)
		for _, c := range spec.Components {
			files := append([]string{}, c.TextureFiles...)
			if c.TextureFile != "" {
				files = append(files, c.TextureFile)
			}
			for _, file := range files {
				if m.effects().EffectFileTexture(ctx.Resources, file) == nil {
					t.Fatalf("effect %d missing texture %s", tc.id, file)
				}
			}
			if c.SpriteFile != "" && m.effects().EffectSpriteView(ctx.Resources, c.SpriteFile) == nil {
				t.Fatalf("missing sprite %s", c.SpriteFile)
			}
			m.effects().DrawFuncEffect(frame, ctx, projection, e, c, 0, 20.5, 20.5, 0, 0, start.Add(time.Duration(tc.tick)*db.EffectFrameDuration))
		}
		v := reflect.ValueOf(frame).Elem()
		if v.FieldByName("worldCommands").Len()+v.FieldByName("worldBillboards").Len() == 0 {
			t.Fatalf("effect %d drew nothing", tc.id)
		}
	}
	for id, tick := range map[int]int{584: 5, 623: 28, 627: 21, 637: 10, 643: 20, 644: 0, 648: 25} {
		m.scheduledSounds = nil
		m.effects().AddBetween(ctx, id, 300, 2000000, start)
		if len(m.scheduledSounds) != 1 || m.scheduledSounds[0].at.Sub(start) != time.Duration(tick)*db.EffectFrameDuration {
			t.Fatalf("effect %d sound timing %+v", id, m.scheduledSounds)
		}
		if id == 648 && m.scheduledSounds[0].positioned {
			t.Fatal("Triple Action sound should be centered")
		}
	}
	// Body overlays are visible for both buff colours, then disappear.
	for _, id := range []int{548, 625, 626} {
		m.effects().Active = nil
		m.effects().AddAt(ctx, id, 300, start)
		billboard := &scene.Billboard{Image: render.NewImage(10, 20)}
		entry := scene.ActorDrawEntry{Actor: ctx.World.Actors[300], WorldX: 20.5, WorldY: 20.5}
		frame := render.NewFrame(800, 600)
		m.effects().DrawActorSpriteBodyEffects3D(ctx, frame, projection, entry, billboard, 1, 1, 1, color.RGBA{255, 255, 255, 255}, render.BlendSourceOver, 0, start.Add(5*db.EffectFrameDuration))
		if reflect.ValueOf(frame).Elem().FieldByName("worldBillboards").Len() < 2 {
			t.Fatalf("body flash %d missing", id)
		}
	}
}

func TestGunslingerSpriteDelayAndGroundLoops(t *testing.T) {
	ctx := testutil.ArcherEffectContext()
	ctx.Resources = testutil.RealDataManager(t)
	m := &WorldMode{}
	start := time.Unix(100, 0)
	projection := testutil.Projection(800, 600, 10, 20, 0)
	for _, id := range []int{effects.TripleAction, effects.LightningS} {
		m.effects().Active = nil
		m.effects().AddAt(ctx, id, 300, start)
		if id == effects.LightningS {
			m.effects().Active[0].Persistent = true
			m.effects().Active[0].Expires = time.Time{}
		}
		for _, tick := range []int{0, 16, 29, 30, 80, 100} {
			frame := render.NewFrame(800, 600)
			m.effects().Draw(frame, ctx, projection, start.Add(time.Duration(tick)*db.EffectFrameDuration), false)
			n := reflect.ValueOf(frame).Elem().FieldByName("worldBillboards").Len()
			if id == effects.TripleAction {
				if (n > 0) != (tick == 30) {
					t.Fatalf("Triple Action tick %d draws=%d", tick, n)
				}
			} else {
				want := 1
				if tick >= 16 && tick%16 < 7 {
					want = 2
				}
				if n != want {
					t.Fatalf("grenade tick %d draws=%d want %d", tick, n, want)
				}
			}
		}
	}
	ctx.Session.Selected.Job = db.JobGunslinger
	ctx.Session.Selected.Weapon = 0
	for _, sex := range []uint8{0, 1} {
		view, status := sprites.LoadPlayer(ctx.Resources, ctx.Session.Selected, sex, false)
		if view == nil {
			t.Fatalf("Gunslinger sprite missing: %s", status)
		}
		for direction := 0; direction < 8; direction++ {
			for family, minFrames := range map[int]int{sprites.ActionPCSkill: 6, sprites.ActionPCAttack1: 3} {
				_, action, ok := sprites.ResolveAction(view.Body.Act, family, direction)
				if !ok || len(action.Animations) < minFrames {
					t.Fatalf("missing Gunslinger pose sex=%d direction=%d family=%d", sex, direction, family)
				}
			}
		}
	}
	for _, path := range []string{"effect\\플립.wav", "effect\\크래커.wav"} {
		if _, err := ctx.Resources.ReadFile("data/wav/" + path); err != nil {
			t.Fatal(err)
		}
	}
	for _, id := range []int{456, 584, 623, 627, 628, 637, 643, 644, 648} {
		spec, _ := effects.Lookup(id)
		for _, path := range spec.Sfx {
			if _, err := ctx.Resources.ReadFile("data/wav/" + path); err != nil {
				t.Fatal(err)
			}
		}
	}
}

func TestHunterEffectsWithClientAssets(t *testing.T) {
	ctx := testutil.ArcherEffectContext()
	ctx.Resources = testutil.RealDataManager(t)
	projection := testutil.Projection(800, 600, 20, 20, 0)
	start := time.Unix(100, 0)
	for _, tc := range []struct{ id, frame int }{{effects.BlitzBeat, 10}, {effects.BlitzBeat, 40}, {effects.Flasher, 30}, {effects.Detecting, 20}, {effects.RemoveTrap, 20}, {effects.Shockwave, 20}} {
		m := &WorldMode{mapScene: maps.System{WhitePixel: render.NewImage(1, 1)}}
		m.effects().AddAt(ctx, tc.id, 300, start)
		frame := render.NewFrame(800, 600)
		m.effects().Draw(frame, ctx, projection, start.Add(time.Duration(tc.frame)*db.EffectFrameDuration), false)
		if reflect.ValueOf(frame).Elem().FieldByName("worldCommands").Len()+reflect.ValueOf(frame).Elem().FieldByName("worldBillboards").Len() == 0 {
			t.Fatalf("effect %d frame %d drew nothing", tc.id, tc.frame)
		}
		for _, sound := range m.scheduledSounds {
			for _, path := range sound.paths {
				if _, err := ctx.Resources.ReadFile("data/wav/" + path); err != nil {
					t.Error(err)
				}
			}
		}
	}
}

func TestKnightAttackEffects(t *testing.T) {
	starts := time.Unix(100, 0)
	hitAt := starts.Add(300 * time.Millisecond)
	for _, tc := range []struct {
		skill uint16
		begin int
		hits  []int
	}{
		{db.SkillKNPierce, effects.PierceSelf, []int{effects.Pierce}},
		{db.SkillKNSpearstab, effects.SpearStabSelf, []int{effects.Pierce}},
		{db.SkillKNBowlingbash, effects.BowlingSelf, []int{effects.Hit1, effects.BowlingBash, effects.Hit1, effects.Hit1}},
		{db.SkillKNSpearboomerang, effects.SpearBmrSelf, []int{effects.Hit4}},
		{db.SkillKNChargeatk, -1, []int{effects.BashHit}},
	} {
		ctx := testutil.KnightEffectContext()
		m := &WorldMode{}
		action := network.ActorActionNotify{SkillID: tc.skill, SourceID: 2000000, TargetID: 300, Damage: 90, HitCount: 3, Action: 8}
		m.addSkillBeginEffect(ctx, action, starts)
		m.addSkillEffect(ctx, action, starts, hitAt)
		if tc.begin >= 0 {
			if len(m.effects().Active) != 1 || m.effects().Active[0].EffectID != tc.begin || m.effects().Active[0].ActorID != 2000000 || !m.effects().Active[0].Starts.Equal(starts) {
				t.Fatalf("skill %d must start its caster STR before impact: %+v", tc.skill, m.effects().Active)
			}
		} else if len(m.effects().Active) != 0 {
			t.Fatalf("skill %d invented a startup flash", tc.skill)
		}
		before := len(m.effects().Active)
		m.addSkillHitEffect(ctx, action, hitAt)
		if len(m.effects().Active) != before+len(tc.hits) {
			t.Fatalf("skill %d replayed its impact for each damage number: %+v", tc.skill, m.effects().Active)
		}
		for i, id := range tc.hits {
			effect := m.effects().Active[before+i]
			wantAt := hitAt
			if tc.skill == db.SkillKNBowlingbash && i > 1 {
				wantAt = wantAt.Add(time.Duration(i-1) * scene.MultiHitDelay)
			}
			if effect.EffectID != id || effect.ActorID != 300 || !effect.Starts.Equal(wantAt) {
				t.Fatalf("skill %d impact has wrong owner, visual or timing: %+v", tc.skill, effect)
			}
		}
		action.Damage = 0
		m = &WorldMode{}
		m.addSkillHitEffect(ctx, action, hitAt)
		if len(m.effects().Active) != 0 {
			t.Fatalf("skill %d played a hit on a miss", tc.skill)
		}
	}
}

func TestKnightBrandishStartsBothSTRsImmediately(t *testing.T) {
	ctx := testutil.KnightEffectContext()
	m := &WorldMode{}
	now := time.Now()
	action := network.ActorActionNotify{SkillID: db.SkillKNBrandishspear, SourceID: 2000000, TargetID: 300, Damage: 50}
	m.addSkillBeginEffect(ctx, action, now)
	m.addSkillEffect(ctx, action, now, now.Add(time.Second))
	if len(m.effects().Active) != 2 || len(m.scheduledSounds) != 0 {
		t.Fatalf("Brandish effects or extra audio: %+v, %+v", m.effects().Active, m.scheduledSounds)
	}
	for i, want := range []struct {
		id    int
		actor uint32
	}{{effects.BrandishSpear2, 2000000}, {effects.BrandishSpear, 300}} {
		effect := m.effects().Active[i]
		if effect.EffectID != want.id || effect.ActorID != want.actor || !effect.Starts.Equal(now) {
			t.Fatalf("Brandish phase %d is delayed or misplaced: %+v", i, effect)
		}
	}
}

func TestKnightChargeAttackCast(t *testing.T) {
	ctx := testutil.KnightEffectContext()
	m := &WorldMode{}
	m.applySkillCastNotify(ctx, network.SkillCastNotify{SkillID: db.SkillKNChargeatk, SourceID: 2000000, TargetID: 300, DelayTime: 1000})
	if m.actorAnims[2000000].actionFamily != scene.SpriteActionIdle {
		t.Fatal("Charge Attack must charge while standing")
	}
	found := false
	for _, effect := range m.effects().Active {
		if effect.EffectID == effects.BeginSpell6 {
			found = true
		}
	}
	if !found {
		t.Fatal("Charge Attack is missing its native casting aura")
	}
}

func TestKnightSpearFlightAndSounds(t *testing.T) {
	ctx := testutil.KnightEffectContext()
	m := &WorldMode{}
	starts := time.Unix(100, 0)
	m.addSkillBeforeHitEffect(ctx, network.ActorActionNotify{SkillID: db.SkillKNSpearboomerang, SourceID: 2000000, TargetID: 300, HitCount: 3}, starts)
	if len(m.effects().Active) != 1 || len(m.scheduledSounds) != 4 {
		t.Fatalf("Spear Boomerang must launch one four-image trail: %+v", m.effects().Active)
	}
	effect := m.effects().Active[0]
	if effect.ActorID != 2000000 || effect.TargetID != 300 {
		t.Fatal("spear travels in the wrong direction")
	}
	for i, sound := range m.scheduledSounds {
		if !sound.at.Equal(starts.Add(time.Duration(i*4) * db.EffectFrameDuration)) {
			t.Fatal("spear sounds are not spaced four native ticks apart")
		}
	}
	for _, age := range []int{0, 6, 13} {
		point, ok := effects.SpearBoomerangPosition(effect, age)
		if !ok || math.Abs(point.X-(10.5+20*float64(age+1)/15)) > .001 || math.Abs(point.Y-3.07) > .001 {
			t.Fatalf("unexpected spear position at tick %d: %+v", age, point)
		}
	}
	for _, age := range []int{-1, 14, 15, 16} {
		if _, ok := effects.SpearBoomerangPosition(effect, age); ok {
			t.Fatalf("spear visible before launch or after reaching target: %d", age)
		}
	}
}

func TestMageCastsUseElementalAura(t *testing.T) {
	ctx := testutil.KnightEffectContext()
	for _, skill := range []uint16{db.SkillMGNapalmbeat, db.SkillMGSafetywall, db.SkillMGSoulstrike, db.SkillMGColdbolt,
		db.SkillMGFrostdiver, db.SkillMGStonecurse, db.SkillMGFireball, db.SkillMGFirewall, db.SkillMGFirebolt,
		db.SkillMGLightningbolt, db.SkillMGThunderstorm, db.SkillMGEnergycoat} {
		// Native PROPERTY_WATER through PROPERTY_SAINT select colored auras;
		// neutral, dark, ghost, and undead retain EF_BEGINSPELL.
		for property, aura := range []int{12, 54, 56, 55, 57, 59, 58, 12, 12, 12} {
			m := &WorldMode{}
			m.applySkillCastNotify(ctx, network.SkillCastNotify{SkillID: skill, SourceID: 2000000, TargetID: 300, DelayTime: 1000, Property: uint32(property)})
			if len(m.effects().Active) != 3 || m.effects().Active[2].EffectID != aura || m.effects().Active[2].Duration != time.Second {
				t.Fatalf("skill %d property %d: expected casting aura %d for one second", skill, property, aura)
			}
		}
	}
}

func TestMageProjectileSequencesAndSounds(t *testing.T) {
	ctx := testutil.KnightEffectContext()
	start := time.Unix(100, 0)
	for _, tc := range []struct {
		skill uint16
		ticks []int
	}{
		{db.SkillMGFireball, []int{20, 24, 28, 32}},
		{db.SkillMGSoulstrike, []int{6, 17, 28, 39}},
		{db.SkillMGColdbolt, []int{12}},
		{db.SkillMGFirebolt, []int{12}},
	} {
		m := &WorldMode{}
		m.addSkillBeforeHitEffect(ctx, network.ActorActionNotify{SkillID: tc.skill, SourceID: 2000000, TargetID: 300, HitCount: 4}, start)
		if len(m.effects().Active) != 1 || len(m.scheduledSounds) != len(tc.ticks) {
			t.Fatalf("skill %d: expected a single sequence and %d sounds", tc.skill, len(tc.ticks))
		}
		for i, tick := range tc.ticks {
			sound := m.scheduledSounds[i]
			if !sound.at.Equal(start.Add(time.Duration(tick)*db.EffectFrameDuration)) || len(sound.paths) != 1 || strings.Contains(sound.paths[0], "%d") {
				t.Fatalf("skill %d: wrong sound or timing at tick %d", tc.skill, tick)
			}
		}
		if tc.skill == db.SkillMGColdbolt || tc.skill == db.SkillMGFirebolt {
			duration := 112 * db.EffectFrameDuration
			if tc.skill == db.SkillMGFirebolt {
				duration = 160 * db.EffectFrameDuration
			}
			if effect := m.effects().Active[0]; effect.HitCount != 4 || effect.Duration != duration {
				t.Fatal("bolt sequence must preserve all projectiles")
			}
		}
	}
}

func TestMageFrostDiverTrailAndBurst(t *testing.T) {
	ctx := testutil.KnightEffectContext()
	start := time.Unix(100, 0)
	m := &WorldMode{mapScene: maps.System{Textures: map[string]*render.Image{"__effectfile_effect/ice.tga|blackkey=true": render.NewImage(32, 32)}}}
	m.addSkillBeforeHitEffect(ctx, network.ActorActionNotify{SkillID: db.SkillMGFrostdiver, SourceID: 2000000, TargetID: 300}, start)
	effect := m.effects().Active[0]
	if effect.ActorID != 2000000 || effect.TargetID != 300 || !effect.Starts.Equal(start) || effects.FrostDiverTrailCount(effect) != 48 {
		t.Fatal("Frost Diver must travel from the caster to the target, starting at action time")
	}
	spec, _ := effects.Lookup(effects.FrostDiver)
	for _, tick := range []int{0, 10, 38, 48, 86, 87} {
		frame := render.NewFrame(800, 600)
		m.effects().DrawFrostDiverSpikes(frame, ctx, spec.Components[0], effect, effect.ActorOrigin, start.Add(time.Duration(tick)*db.EffectFrameDuration))
		cmds := reflect.ValueOf(frame).Elem().FieldByName("worldCommands")
		if tick >= 87 {
			if cmds.Len() != 0 {
				t.Fatal("ice trail survived its last spike")
			}
			continue
		}
		want := min(48, tick+1) - max(0, tick-39)
		if cmds.Len() != 1 || cmds.Index(0).FieldByName("Vertices").Len() != want*12 {
			t.Fatalf("tick %d: incorrect ice spike count", tick)
		}
	}
	spec, _ = effects.Lookup(effects.FrostDiverHit)
	frame := render.NewFrame(800, 600)
	m.effects().DrawFrostDiverSpikes(frame, ctx, spec.Components[0], effects.Instance{EffectID: effects.FrostDiverHit, Starts: start}, effect.ActorOrigin, start.Add(20*db.EffectFrameDuration))
	cmd := reflect.ValueOf(frame).Elem().FieldByName("worldCommands").Index(0)
	vertices := cmd.FieldByName("Vertices")
	if vertices.Len() != 8*12 || render.Blend(cmd.FieldByName("Options").FieldByName("Blend").Int()) != render.BlendSourceOver {
		t.Fatal("Frost Diver impact must be eight translucent ice spikes")
	}
	for i := 0; i < vertices.Len(); i += 3 {
		base, tip := vertices.Index(i), vertices.Index(i+1)
		if tip.FieldByName("Y").Float() <= base.FieldByName("Y").Float() {
			t.Fatal("ice spike points into the ground")
		}
	}
}

func TestMageRealAssets(t *testing.T) {
	ctx := testutil.KnightEffectContext()
	ctx.Resources = testutil.RealDataManager(t)
	start := time.Unix(100, 0)
	for _, id := range []int{effects.SoulStrike, effects.FireBall, effects.FrostDiver, effects.FrostDiverHit, effects.FireBolt, effects.ColdBolt, effects.Sight, effects.StoneCurse, effects.EnergyCoat, effects.SafetyWall, effects.FireWall, effects.ThunderStorm} {
		t.Run(strconv.Itoa(id), func(t *testing.T) {
			m := &WorldMode{}
			if !m.effects().AddBetween(ctx, id, 300, 2000000, start) {
				t.Fatal("effect unavailable")
			}
			for _, sound := range m.scheduledSounds {
				for _, path := range sound.paths {
					if _, err := ctx.Resources.ReadFile("data/wav/" + path); err != nil {
						t.Fatal(err)
					}
				}
			}
			draws := 0
			for _, tick := range []int{0, 12, 20, 30, 45} {
				frame := render.NewFrame(800, 600)
				m.effects().Draw(frame, ctx, testutil.Projection(800, 600, 20, 20, 0), start.Add(time.Duration(tick)*db.EffectFrameDuration), false)
				v := reflect.ValueOf(frame).Elem()
				draws += v.FieldByName("worldCommands").Len() + v.FieldByName("worldBillboards").Len() + v.FieldByName("commands").Len()
			}
			if draws == 0 {
				t.Fatal("effect produced no geometry with real assets")
			}
		})
	}
	if _, err := ctx.Resources.ReadFile("data/wav/_stonecurse.wav"); err != nil {
		t.Fatal(err)
	}
}

func TestMerchantAttackEffects(t *testing.T) {
	ctx := testutil.MerchantEffectContext()
	start := time.Unix(100, 0)
	hit := start.Add(500 * time.Millisecond)
	for _, skill := range []uint16{db.SkillMCMammonite, db.SkillMCCartrevolution} {
		for _, damage := range []int32{0, 100} {
			m := &WorldMode{}
			action := network.ActorActionNotify{SkillID: skill, SourceID: 2000000, TargetID: 300, Damage: damage, Action: network.ActorActionSkill}
			m.addSkillBeginEffect(ctx, action, start)
			m.addSkillBeforeHitEffect(ctx, action, start)
			m.addSkillEffect(ctx, action, start, hit)
			m.addSkillHitEffect(ctx, action, hit)
			begin, impact := effects.BashBegin, effects.Mammonite
			if skill == db.SkillMCCartrevolution {
				begin, impact = effects.CartRevolution, effects.CartRevolution
			}
			want := 1
			if damage > 0 {
				want += 2
			}
			if len(m.effects().Active) != want || m.effects().Active[0].EffectID != begin || m.effects().Active[0].ActorID != 2000000 || !m.effects().Active[0].Starts.Equal(start) {
				t.Fatalf("skill %d damage %d: wrong caster effects: %+v", skill, damage, m.effects().Active)
			}
			if damage > 0 {
				if e := m.effects().Active[2]; e.EffectID != impact || e.ActorID != 300 || !e.Starts.Equal(hit) {
					t.Fatalf("skill %d: impact must appear on the victim at hit time: %+v", skill, e)
				}
			}
			if skill == db.SkillMCCartrevolution {
				for i, effect := range []effects.Instance{m.effects().Active[0], m.effects().Active[len(m.effects().Active)-1]} {
					if damage == 0 && i == 1 {
						break
					}
					for j, tick := range []int{7, 20} {
						if sound := m.scheduledSounds[i*2+j]; !sound.at.Equal(effect.Starts.Add(time.Duration(tick) * db.EffectFrameDuration)) {
							t.Fatalf("Cart Revolution sound must coincide with burst %d: %+v", tick, sound)
						}
					}
				}
			}
			before, sounds := len(m.effects().Active), len(m.scheduledSounds)
			action.Action = 5 // Secondary splash notification.
			m.addSkillBeginEffect(ctx, action, start)
			m.addSkillBeforeHitEffect(ctx, action, start)
			m.addSkillEffect(ctx, action, start, hit)
			m.addSkillHitEffect(ctx, action, hit)
			if len(m.effects().Active) != before || len(m.scheduledSounds) != sounds {
				t.Fatal("splash victims must not replay the skill")
			}
		}
	}
	for _, weapon := range []int16{0, 1, 6} {
		actor := ctx.World.Player
		actor.Weapon = weapon
		if got := skillAction(db.SkillMCMammonite).actionFamilyForActorWithResources(nil, actor); got != attackActionFamilyForActorWithResources(nil, actor) {
			t.Fatal("Mammonite must use the equipped weapon's normal attack animation")
		}
	}
}

func TestMammoniteCoinBounces(t *testing.T) {
	ctx := testutil.MerchantEffectContext()
	ctx.World.GAT = &res.GAT{Width: 30, Height: 30, Cells: make([]res.GATCell, 900)}
	for i := range ctx.World.GAT.Cells {
		ctx.World.GAT.Cells[i].Heights = [4]float32{4, 4, 4, 4}
	}
	m := &WorldMode{}
	start := time.Unix(100, 0)
	m.effects().AddAt(ctx, effects.Mammonite, 300, start)
	effect := m.effects().Active[0]
	if effect.CoinTrails == nil {
		t.Fatal("missing five coin trajectories")
	}
	bounces := 0
	for _, trail := range effect.CoinTrails {
		if len(trail) < 40 || len(trail) > 69 || trail[0].Y <= 6 || trail[5].Y <= trail[0].Y {
			t.Fatal("coins must rise from ten native units above the victim's ground height")
		}
		for age := 1; age < len(trail); age++ {
			if trail[age].Y < 4 && trail[age].Y < trail[age-1].Y {
				bounces++
				if age+1 < len(trail) && trail[age+1].Y <= trail[age].Y {
					t.Fatal("coin failed to bounce off terrain")
				}
			}
		}
	}
	if bounces < 5 || len(m.scheduledSounds) != bounces {
		t.Fatalf("one sound per bounce: bounces=%d sounds=%d", bounces, len(m.scheduledSounds))
	}
	for _, sound := range m.scheduledSounds {
		if !sound.at.After(start.Add(20 * db.EffectFrameDuration)) {
			t.Fatal("coin audio must wait until a coin hits the ground")
		}
	}
	ctx.Session.LessEffects = true
	m = &WorldMode{}
	m.effects().AddAt(ctx, effects.Mammonite, 300, start)
	if m.effects().Active[0].CoinTrails != nil || len(m.scheduledSounds) != 0 {
		t.Fatal("minimal Mammonite must omit coins and bounce sounds")
	}
}

func TestMerchantEffectsWithClientAssets(t *testing.T) {
	ctx := testutil.MerchantEffectContext()
	ctx.Resources = testutil.RealDataManager(t)
	m := &WorldMode{}
	start := time.Unix(100, 0)
	for _, id := range []int{effects.Mammonite, effects.CartRevolution, effects.Loud} {
		m.effects().AddAt(ctx, id, 300, start)
		spec, _ := effects.Lookup(id)
		for _, component := range spec.Components {
			if component.StrFile != "" && m.effects().LoadWorldEffectSTR(ctx.Resources, component.StrFile, component.TexturePath) == nil {
				t.Fatalf("missing STR %q", component.StrFile)
			}
		}
	}
	for _, sound := range m.scheduledSounds {
		for _, path := range sound.paths {
			if _, err := ctx.Resources.ReadFile("data/wav/" + path); err != nil {
				t.Error(err)
			}
		}
	}
	projection := testutil.Projection(800, 600, 11, 20, 0)
	coin, _ := effects.Lookup(effects.Mammonite)
	for _, tc := range []struct{ tick, count int }{{19, 0}, {20, 5}, {21, 10}, {35, 10}, {90, 0}} {
		frame := render.NewFrame(800, 600)
		m.effects().DrawMammoniteCoins(frame, ctx, projection, coin.Components[1], m.effects().Active[0], start.Add(time.Duration(tc.tick)*db.EffectFrameDuration))
		if count := reflect.ValueOf(frame).Elem().FieldByName("worldBillboards").Len(); count != tc.count {
			t.Fatalf("tick %d: got %d coins/afterimages, want %d", tc.tick, count, tc.count)
		}
	}
	cart, _ := effects.Lookup(effects.CartRevolution)
	for _, tc := range []struct{ tick, count int }{{6, 0}, {8, 2}, {21, 4}, {30, 2}, {40, 0}} {
		frame := render.NewFrame(800, 600)
		m.effects().DrawCartRevolutionEffect(frame, ctx, cart.Components[1], m.effects().Active[1], start.Add(time.Duration(tc.tick)*db.EffectFrameDuration))
		if count := reflect.ValueOf(frame).Elem().FieldByName("worldCommands").Len(); count != tc.count {
			t.Fatalf("tick %d: got %d burst meshes, want %d", tc.tick, count, tc.count)
		}
	}
}

func TestMonkBuffsFollowActorEntrySnapshots(t *testing.T) {
	for _, job := range []int16{db.JobMonk, db.JobSuperNovice} {
		ctx := testutil.MonkEffectContext()
		delete(ctx.World.Actors, 300)
		m := &WorldMode{}
		data := make([]byte, 60)
		binary.LittleEndian.PutUint32(data[2:], 300)
		binary.LittleEndian.PutUint16(data[16:], uint16(job))
		applyEntry := func(state uint32) {
			t.Helper()
			binary.LittleEndian.PutUint32(data[44:], state)
			entry, ok, err := network.ParseActorEntry(network.Packet{ID: 0x02EE, Data: data})
			if err != nil || !ok {
				t.Fatalf("parse entry: ok=%t err=%v", ok, err)
			}
			m.upsertNetworkActor(ctx, entry)
		}
		assertAuras := func(active bool) {
			t.Helper()
			for _, id := range []int{effects.Gumgang, effects.SteelBody} {
				if m.effects().HasActive(id, 300, time.Now().Add(time.Hour)) != active {
					t.Fatalf("job %d effect %d active != %t", job, id, active)
				}
			}
		}
		applyEntry(db.Opt3Explosionspirits | db.Opt3Steelbody)
		assertAuras(true)
		original := append([]effects.Instance(nil), m.effects().Active...)
		applyEntry(db.Opt3Explosionspirits | db.Opt3Steelbody)
		m.upsertNetworkActor(ctx, network.ActorEntry{ID: 300, Moving: true, ToX: 15, ToY: 20})
		assertAuras(true)
		if !reflect.DeepEqual(m.effects().Active, original) {
			t.Fatal("actor refresh restarted the auras")
		}
		applyEntry(0)
		assertAuras(false)
	}
}

func TestThrowSpiritSphereSizeUsesSphereCount(t *testing.T) {
	ctx := testutil.MonkEffectContext()
	m := &WorldMode{}
	starts := time.Now()
	for _, count := range []uint16{1, 5} {
		m.addSkillEffect(ctx, network.ActorActionNotify{SourceID: 2000000, TargetID: 300, SkillID: db.SkillMOFingeroffensive, SkillLevel: 5, HitCount: count}, starts, starts)
	}
	small, large := m.effects().Active[0].Size, m.effects().Active[1].Size
	if math.Abs(large/small-4.0/2.4) > 1e-6 {
		t.Fatalf("sphere sizes = %g, %g; want scaling by the number of spheres used", small, large)
	}
}

func TestMonkAttackEffectsStartAtRelease(t *testing.T) {
	for _, tc := range []struct {
		skill  uint16
		target []int
		caster []int
	}{
		{db.SkillMOTripleattack, nil, []int{effects.TripleAttack}},
		{db.SkillMOInvestigate, []int{effects.Chimto}, []int{effects.TeiHit2}},
		{db.SkillMOFingeroffensive, nil, []int{effects.Tanji}},
		{db.SkillMOExtremityfist, []int{effects.TeiHit1X}, []int{effects.Quake}},
		{db.SkillMOChaincombo, []int{effects.TeiHit1, effects.ChainCombo}, []int{effects.Gumgang3}},
		{db.SkillMOCombofinish, nil, []int{effects.Gumgang3, effects.HitLine, effects.Quake}},
	} {
		ctx := testutil.MonkEffectContext()
		m := &WorldMode{}
		release := time.Unix(100, 0)
		hit := release.Add(800 * time.Millisecond)
		action := network.ActorActionNotify{SkillID: tc.skill, SourceID: 2000000, TargetID: 300, SkillLevel: 5, Damage: 100, HitCount: 3}
		m.addSkillEffect(ctx, action, release, hit)
		if len(m.effects().Active) != len(tc.target)+len(tc.caster) {
			t.Fatalf("skill %d effects = %+v", tc.skill, m.effects().Active)
		}
		for _, effect := range m.effects().Active {
			wantActor := uint32(0)
			for _, id := range tc.target {
				if effect.EffectID == id {
					wantActor = 300
				}
			}
			for _, id := range tc.caster {
				if effect.EffectID == id {
					wantActor = 2000000
				}
			}
			if wantActor == 0 || effect.ActorID != wantActor || !effect.Starts.Equal(release) {
				t.Fatalf("skill %d misplaced or delayed effect: %+v", tc.skill, effect)
			}
		}
		if tc.skill == db.SkillMOChaincombo {
			if len(m.scheduledSounds) != 1 || m.scheduledSounds[0].paths[0] != "effect\\mon_연환.wav" {
				t.Fatalf("Chain Combo sounds = %+v; must not play Fury", m.scheduledSounds)
			}
		}
		if tc.skill == db.SkillMOTripleattack {
			if len(m.scheduledSounds) != 3 {
				t.Fatalf("Triple Attack sounds = %+v", m.scheduledSounds)
			}
			for i, sound := range m.scheduledSounds {
				want := "effect\\ef_hit2.wav"
				if i == 1 {
					want = "effect\\ef_hit4.wav"
				}
				if sound.paths[0] != want || !sound.at.Equal(release.Add(time.Duration(10*(i+1))*db.EffectFrameDuration)) {
					t.Fatalf("Triple Attack sound %d: %+v", i, sound)
				}
			}
		}
	}
}

func TestMonkImpactMarkersAndSplash(t *testing.T) {
	ctx := testutil.MonkEffectContext()
	starts := time.Unix(100, 0)
	for _, tc := range []struct {
		skill uint16
		want  map[int]int
	}{
		{db.SkillMOTripleattack, map[int]int{effects.Hit1: 3}},
		{db.SkillMOInvestigate, map[int]int{effects.Hit1: 3}},
		{db.SkillMOFingeroffensive, map[int]int{effects.Hit1: 3, effects.BashHit: 1}},
		{db.SkillMOExtremityfist, map[int]int{effects.Hit1: 3}},
		{db.SkillMOChaincombo, map[int]int{effects.Hit1: 3, effects.SonicBlowHit: 3}},
		{db.SkillMOCombofinish, map[int]int{effects.Hit1: 3, effects.BashHit: 1}},
		{db.SkillMOBalkyoung, map[int]int{effects.Hit3: 3}},
	} {
		action := network.ActorActionNotify{SkillID: tc.skill, SourceID: 2000000, TargetID: 300, Damage: 100, HitCount: 3}
		m := &WorldMode{}
		m.addSkillHitEffect(ctx, action, starts)
		got := make(map[int]int)
		for _, effect := range m.effects().Active {
			if effect.ActorID != 300 {
				t.Fatalf("skill %d impact on wrong actor", tc.skill)
			}
			got[effect.EffectID]++
		}
		if !reflect.DeepEqual(got, tc.want) {
			t.Fatalf("skill %d hit markers = %v, want %v", tc.skill, got, tc.want)
		}
		action.Action = 5
		m = &WorldMode{}
		m.addSkillEffect(ctx, action, starts, starts)
		m.addSkillHitEffect(ctx, action, starts)
		// The original explicitly exempts Combo Finish from ACTION_SPLASH suppression.
		if (len(m.effects().Active) > 0) != (tc.skill == db.SkillMOCombofinish) {
			t.Fatalf("skill %d splash effects = %+v", tc.skill, m.effects().Active)
		}
	}
}

func TestBladeStopCapturesVictimsPlayingFrame(t *testing.T) {
	ctx := testutil.MonkEffectContext()
	actions := make([]res.ACTAction, 13*8)
	for i := range actions {
		actions[i] = res.ACTAction{DelayMS: 100, Animations: make([]res.ACTAnimation, 4)}
	}
	started := time.Now().Add(-150 * time.Millisecond)
	m := &WorldMode{playerView: &sprites.Humanoid{Body: &scene.SpriteView{Act: &res.ACT{Actions: actions}}, Started: started}}
	m.setCombatActorAction(ctx, 2000000, actorAnimation{actionFamily: sprites.ActionPCAttack1, started: started, play: true, hasPlay: true, duration: time.Second})
	m.applyActorBladeStop(ctx, network.ActorBladeStop{SourceID: 300, TargetID: 2000000, Active: true})
	anim := m.actorAnims[150000]
	want := scene.MinInt(3, int(anim.started.Sub(started)/(100*time.Millisecond)))
	if want == 0 || anim.fixedMotion != want || anim.play {
		t.Fatalf("victim pose = %+v, want frozen frame %d", anim, want)
	}
	if later, ok := m.actorAnimation(150000, time.Now().Add(time.Minute)); !ok || later.fixedMotion != want {
		t.Fatal("Blade Stop victim resumed animation before release")
	}
}

func TestNinjaWindAndPetalsAreFrameRateIndependent(t *testing.T) {
	ctx := testutil.ArcherEffectContext()
	m := &WorldMode{}
	start := time.Unix(100, 0)
	m.addSkillBeforeHitEffect(ctx, network.ActorActionNotify{SkillID: 540, HitCount: 3, SourceID: 2000000, TargetID: 300}, start)
	e := m.effects().Active[0]
	var stepped, jumped effects.NinjaWind
	for tick := 0; tick < 16; tick++ {
		stepped.Advance(ctx, e, start.Add(time.Duration(tick)*db.EffectFrameDuration))
	}
	jumped.Advance(ctx, e, start.Add(15*db.EffectFrameDuration))
	if !reflect.DeepEqual(stepped, jumped) {
		t.Fatal("wind path depends on frame rate")
	}
	if stepped.Heads[0].Samples[0].Position.Z == stepped.Heads[1].Samples[0].Position.Z {
		t.Fatal("alternating wind blades must curve on different sides")
	}
	delete(ctx.World.Actors, 300)
	stepped.Advance(ctx, e, start.Add(100*db.EffectFrameDuration))
	for _, h := range stepped.Heads {
		for _, s := range h.Samples {
			if s.Alpha != 0 {
				t.Fatal("wind did not fade after losing its target")
			}
		}
	}
	ctx = testutil.ArcherEffectContext()
	m = &WorldMode{}
	m.effects().AddAt(ctx, effects.Kouenka, 300, start)
	e = m.effects().Active[0]
	var a, b effects.NinjaPetals
	for tick := 0; tick < 30; tick++ {
		a.Advance(ctx, e, start.Add(time.Duration(tick)*db.EffectFrameDuration))
	}
	b.Advance(ctx, e, start.Add(29*db.EffectFrameDuration))
	if !reflect.DeepEqual(a, b) {
		t.Fatal("petals depend on frame rate")
	}
	if a.Particles[0].Position.Y <= e.ActorOrigin.Y || a.Particles[19].Age != 11 {
		t.Fatal("petals must rise and appear one per tick")
	}
}

func TestNinjaNativeAssetsAndRendering(t *testing.T) {
	ctx := testutil.ArcherEffectContext()
	ctx.Resources = testutil.RealDataManager(t)
	ctx.World.Player.Job, ctx.Session.Selected.Job = db.JobNinja, db.JobNinja
	ctx.World.Player.Weapon, ctx.Session.Selected.Weapon = 0, 0
	for _, sex := range []uint8{0, 1} {
		view, status := sprites.LoadPlayer(ctx.Resources, ctx.Session.Selected, sex, false)
		if view == nil {
			t.Fatalf("missing Ninja sprite: %s", status)
		}
		for direction := 0; direction < 8; direction++ {
			_, action, ok := sprites.ResolveAction(view.Body.Act, sprites.ActionPCSkill, direction)
			if !ok || len(action.Animations) < 6 {
				t.Fatalf("Ninja sex %d direction %d lacks native skill poses", sex, direction)
			}
		}
	}
	m := &WorldMode{}
	start := time.Unix(100, 0)
	projection := testutil.Projection(800, 600, 10, 20, 0)
	for _, id := range []int{613, 614, 615, 616, 618, 619, 620, 621, 622, 624, 630, 631, 632, 633, 634, 635, 636} {
		m.effects().Active = nil
		if !m.effects().AddBetween(ctx, id, 300, 2000000, start) {
			t.Fatalf("effect %d missing", id)
		}
		e := m.effects().Active[0]
		e.HitCount = 3
		spec, _ := effects.Lookup(id)
		for _, c := range spec.Components {
			if c.SpriteFile != "" && m.effects().EffectSpriteView(ctx.Resources, c.SpriteFile) == nil {
				t.Fatalf("missing SPR %s", c.SpriteFile)
			}
			if c.TextureFile != "" && m.effects().EffectFileTexture(ctx.Resources, c.TextureFile) == nil {
				t.Fatalf("missing texture %s", c.TextureFile)
			}
		}
		if id == 618 || id == 621 || id == 622 || id == 624 || id == 631 || id == 634 {
			frame := render.NewFrame(800, 600)
			for _, c := range spec.Components {
				m.effects().DrawFuncEffect(frame, ctx, projection, e, c, 0, 20.5, 20.5, 0, 0, start.Add(10*db.EffectFrameDuration))
			}
			v := reflect.ValueOf(frame).Elem()
			if v.FieldByName("worldCommands").Len()+v.FieldByName("worldBillboards").Len() == 0 {
				t.Fatalf("effect %d produced no draw commands", id)
			}
		}
	}
}

func TestMagicCrasherBlendReachesActorDraws(t *testing.T) {
	sprite := &scene.SpriteView{
		Spr:    &res.SPR{Frames: []res.SPRFrame{{Type: res.SPRFrameRGBA, Width: 8, Height: 8, Data: testutil.SolidRGBAFrame(8, 8)}}},
		Act:    &res.ACT{Actions: []res.ACTAction{{DelayMS: 100, Animations: []res.ACTAnimation{{Layers: []res.ACTLayer{{Index: 0, SPRType: res.SPRFrameRGBA, ScaleX: 1, ScaleY: 1, Color: [4]float32{1, 1, 1, 1}}}}}}}},
		Images: make(map[scene.SpriteFrameKey]*render.Image), Billboards: make(map[scene.SpriteBillboardKey]*scene.Billboard),
	}
	mode := &WorldMode{
		playerView: &sprites.Humanoid{Body: sprite},
		nonPCViews: map[int]*scene.SpriteView{1002: sprite},
	}
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 42, X: 10, Y: 20}
	ctx := client.Context{World: world, Session: &session.Session{AccountID: 43, CharID: 44}}
	projection := testutil.Projection(800, 600, 10.5, 20.5, 0)
	for _, target := range []uint32{42, 43, 44, 99} {
		for _, expired := range []bool{false, true} {
			now := time.Now()
			effect := effects.Instance{EffectID: effects.MagicCrasher, ActorID: target, Starts: now.Add(-700 * time.Millisecond), Expires: now.Add(time.Second)}
			if expired {
				effect.Expires = now.Add(-time.Millisecond)
			}
			mode.effects().Active = []effects.Instance{effect}
			for _, local := range []bool{false, true} {
				for _, stealth := range []scene.StealthView{scene.StealthVisible, scene.StealthSilhouette} {
					entry := scene.ActorDrawEntry{Actor: worldstate.Actor{ID: 99, Job: 1002}, WorldX: 10.5, WorldY: 20.5, Scale: 1, Stealth: stealth}
					frame := render.NewFrame(800, 600)
					var drawn bool
					if local {
						entry.Actor = world.Player
						drawn = mode.drawPlayerSprite3D(ctx, frame, projection, entry, 0, 0, 1, 1)
					} else {
						drawn = mode.drawNonPCSprite3D(frame, ctx, projection, entry, 0, 1)
					}
					commands := reflect.ValueOf(frame).Elem().FieldByName("worldBillboards")
					if !drawn || commands.Len() != 1 {
						t.Fatalf("local=%t: missing actor billboard", local)
					}
					want := render.BlendSourceOver
					if !expired && stealth == scene.StealthVisible && (local == (target != 99)) {
						want = render.BlendLighter
					}
					options := commands.Index(0).FieldByName("Options")
					if got := render.Blend(options.FieldByName("Blend").Int()); got != want {
						t.Fatalf("target=%d local=%t expired=%t stealth=%d: blend %v, want %v", target, local, expired, stealth, got, want)
					}
					if !options.FieldByName("DepthTest").Bool() || !options.FieldByName("DepthWrite").Bool() {
						t.Fatal("flash changed actor depth options")
					}
				}
			}
		}
	}
}

func TestPriestCasting(t *testing.T) {
	ctx := testutil.KnightEffectContext()
	for _, skill := range []uint16{
		db.SkillMGSafetywall, db.SkillALLResurrection, db.SkillPRImpositio, db.SkillPRSuffragium,
		db.SkillPRAspersio, db.SkillPRBenedictio, db.SkillPRSanctuary, db.SkillPRStrecovery,
		db.SkillPRKyrie, db.SkillPRMagnificat, db.SkillPRGloria, db.SkillPRLexdivina,
		db.SkillPRTurnundead, db.SkillPRLexaeterna, db.SkillPRMagnus,
		db.SkillHPAssumptio, db.SkillHPBasilica, db.SkillPRRedemptio,
	} {
		m := &WorldMode{}
		// Ordinary auras follow the property; High Priest and Redemptio stay white.
		m.applySkillCastNotify(ctx, network.SkillCastNotify{SkillID: skill, SourceID: 2000000, TargetID: 300, DelayTime: 1000, Property: 3})
		aura := effects.BeginSpell3
		if skill == db.SkillHPAssumptio || skill == db.SkillHPBasilica || skill == db.SkillPRRedemptio {
			aura = effects.BeginSpell6
		}
		if !m.effects().HasActive(aura, 2000000, time.Now()) {
			t.Fatalf("skill %d: wrong cast aura", skill)
		}
		pose := sprites.ActionPCReadyFight
		if skill == db.SkillPRRedemptio {
			pose = scene.SpriteActionIdle
		}
		if m.actorAnims[2000000].actionFamily != pose {
			t.Fatalf("skill %d: wrong cast pose", skill)
		}
	}
}

func TestProfessorClientAssets(t *testing.T) {
	ctx := testutil.ArcherEffectContext()
	ctx.Resources = testutil.RealDataManager(t)
	ctx.World.Player.Job, ctx.Session.Selected.Job = db.JobSageH, db.JobSageH
	start := time.Unix(100, 0)
	for _, sex := range []uint8{0, 1} {
		view, status := sprites.LoadPlayer(ctx.Resources, ctx.Session.Selected, sex, false)
		if view == nil || !strings.Contains(view.Body.ActSource, res.PlayerJobToken(db.JobSageH)) {
			t.Fatalf("must load the Professor body without fallback: %s", status)
		}
		for dir := 0; dir < 8; dir++ {
			for _, family := range []int{sprites.ActionPCReadyFight, sprites.ActionPCSkill} {
				if _, a, ok := sprites.ResolveAction(view.Body.Act, family, dir); !ok || len(a.Animations) == 0 {
					t.Fatal("Professor casting/release ACT is missing")
				}
			}
		}
	}
	for _, skill := range []uint16{373, 374, 375, 402, 403, 404, 405, 482} {
		m := &WorldMode{}
		m.applySkillCastNotify(ctx, network.SkillCastNotify{SkillID: skill, SourceID: 2000000, TargetID: 300, Property: 3, DelayTime: 1000})
		aura := 55
		if skill == 404 {
			aura = 16
		}
		if m.actorAnims[2000000].actionFamily != sprites.ActionPCReadyFight || len(m.effects().Active) != 3 || m.effects().Active[2].EffectID != aura {
			t.Fatalf("skill %d needs casting aura %d and the ready pose", skill, aura)
		}
	}
	projection := testutil.Projection(800, 600, 20, 20, 0)
	for _, tc := range []struct{ effect, tick, billboards, vertices int }{
		{383, 0, 21, 0}, {383, 4, 105, 0}, {383, 60, 84, 0}, {383, 64, 0, 0},
		{384, 19, 0, 80}, {384, 77, 0, 0}, {404, 90, 0, 4}, {405, 90, 0, 24},
	} {
		m := &WorldMode{}
		m.effects().AddBetween(ctx, tc.effect, 2000000, 300, start)
		frame := render.NewFrame(800, 600)
		m.effects().Draw(frame, ctx, projection, start.Add(time.Duration(tc.tick)*db.EffectFrameDuration), false)
		v := reflect.ValueOf(frame).Elem()
		commands := v.FieldByName("worldCommands")
		vertices := 0
		for i := 0; i < commands.Len(); i++ {
			vertices += commands.Index(i).FieldByName("Vertices").Len()
		}
		if v.FieldByName("worldBillboards").Len() != tc.billboards || vertices != tc.vertices {
			t.Fatalf("effect %d tick %d: billboards=%d vertices=%d", tc.effect, tc.tick, v.FieldByName("worldBillboards").Len(), vertices)
		}
		if tc.effect == 405 {
			cmd := commands.Index(0)
			if cmd.FieldByName("Options").FieldByName("Blend").Int() != int64(render.BlendSourceOver) {
				t.Fatal("Wall of Fog needs translucent grey walls, not additive light")
			}
		}
	}
	m := &WorldMode{}
	for _, file := range []string{"소울번", "사람효과"} {
		if m.effects().LoadWorldEffectSTR(ctx.Resources, file, "") == nil {
			t.Fatalf("missing native STR %s", file)
		}
	}
	for _, sound := range []string{"소울 체인지", "매직 크래쉬", "마법력 증폭", "swordman_provoke", "priest_suffragium"} {
		if _, err := ctx.Resources.ReadFile("data/wav/effect/" + sound + ".wav"); err != nil {
			t.Fatal(err)
		}
	}
}

func TestWorldDrawPlacesEffectsBeforeAndAfterEntities(t *testing.T) {
	const effectID = -1
	db.EffectSpecs[effectID] = db.EffectSpec{Components: []db.EffectComponent{
		{Kind: db.EffectComponentSPR, SpriteFile: "after", Duration: time.Hour},
		{Kind: db.EffectComponentSPR, SpriteFile: "before", Duration: time.Hour, RenderBefore: true},
	}}
	t.Cleanup(func() { delete(db.EffectSpecs, effectID) })
	for _, terrain := range []string{"gat", "gnd", "models"} {
		t.Run(terrain, func(t *testing.T) {
			world := worldstate.New()
			world.Player = worldstate.Actor{ID: 42, X: 1, Y: 1}
			world.Actors[100] = worldstate.Actor{ID: 100, Job: 1002, X: 1, Y: 1}
			world.GAT = &res.GAT{Width: 4, Height: 4, Cells: make([]res.GATCell, 16)}
			if terrain != "gat" {
				world.GND = &res.GND{Width: 2, Height: 2, Cells: make([]res.GNDCell, 4)}
			}
			if terrain == "models" {
				world.RSW = &res.RSW{}
				world.RSM = map[string]*res.RSM{"empty": {}}
			}
			ctx := client.Context{World: world, Resources: &res.Manager{}, Session: &session.Session{CharID: 42}}
			ctx.Config.Render.NoUI = true
			before, actor, after := testutil.EffectRenderTestSprite(), testutil.EffectRenderTestSprite(), testutil.EffectRenderTestSprite()
			starts := time.Now().Add(-time.Second)
			mode := &WorldMode{
				effectSystem: effects.System{EffectViews: map[string]*scene.SpriteView{"before": before, "after": after}, Active: []effects.Instance{
					{EffectID: effectID, X: 1, Y: 1, Starts: starts, Expires: starts.Add(time.Hour)},
					{EffectID: effectID, X: 1, Y: 1, Starts: starts, Expires: starts},
					{EffectID: effectID, X: 1, Y: 1, Starts: starts.Add(time.Hour)},
				}},
				nonPCViews: map[int]*scene.SpriteView{1002: actor},
			}
			frame := render.NewFrame(800, 600)
			mode.Draw(ctx, frame)
			commands := reflect.ValueOf(frame).Elem().FieldByName("worldBillboards")
			if commands.Len() != 3 {
				t.Fatalf("draws = %d, want one effect behind, one actor, one effect in front", commands.Len())
			}
			for i, view := range []*scene.SpriteView{before, actor, after} {
				want := reflect.ValueOf(view.Billboards[scene.SpriteBillboardKey{}].Image).Pointer()
				if commands.Index(i).FieldByName("Texture").Pointer() != want {
					t.Fatalf("draw %d has wrong texture/order", i)
				}
			}
			if len(mode.effects().Active) != 2 {
				t.Fatal("expired effects should be removed and scheduled effects retained")
			}
		})
	}
}

func TestRogueAttackOwnersAndTiming(t *testing.T) {
	ctx := testutil.RogueSkillContext()
	starts := time.Unix(100, 0)
	for _, skill := range []uint16{db.SkillRGBackstap, db.SkillRGIntimidate} {
		m := &WorldMode{}
		// The action effect also plays on a miss.
		action := network.ActorActionNotify{SkillID: skill, SourceID: 2000000, TargetID: 300, Damage: 0}
		m.addSkillEffect(ctx, action, starts, starts.Add(time.Second))
		m.addSkillHitEffect(ctx, action, starts.Add(time.Second))
		if len(m.effects().Active) != 1 {
			t.Fatalf("skill %d: effects = %+v", skill, m.effects().Active)
		}
		effect := m.effects().Active[0]
		if effect.ActorID != action.SourceID || !effect.Starts.Equal(starts) {
			t.Fatalf("skill %d must start on the caster at action time: %+v", skill, effect)
		}
		if skill == db.SkillRGBackstap {
			if effect.ActorOrigin.X != 16.5 || effect.ProjectileDirection.X != 6 || effect.ProjectileDirection.Z != 0 {
				t.Fatalf("Back Stab must start at the victim and continue away from the caster: %+v", effect)
			}
			if len(m.scheduledSounds) != 1 || !m.scheduledSounds[0].at.Equal(starts.Add(20*db.EffectFrameDuration)) {
				t.Fatalf("Back Stab sound = %+v", m.scheduledSounds)
			}
		} else {
			if effect.ActorOrigin.X != 10.5 || len(m.scheduledSounds) != 2 ||
				!m.scheduledSounds[0].at.Equal(starts) || !m.scheduledSounds[1].at.Equal(starts.Add(35*db.EffectFrameDuration)) {
				t.Fatalf("Snatch origin or sounds: %+v, %+v", effect, m.scheduledSounds)
			}
		}
	}
}

func TestSageCastColors(t *testing.T) {
	for _, tc := range []struct {
		skill    uint16
		property uint32
		effect   int
	}{
		{db.SkillSAVolcano, 3, effects.BeginSpell3},
		{db.SkillSADeluge, 1, effects.BeginSpell2},
		{db.SkillSAViolentgale, 4, effects.BeginSpell5},
		{db.SkillSALandprotector, 2, effects.BeginSpell4},
		{db.SkillSAAutospell, 0, effects.BeginSpell},
		{db.SkillSASpellbreaker, 0, effects.BeginSpell},
		{db.SkillSADispell, 0, effects.BeginSpell},
		{db.SkillSAAbracadabra, 0, effects.BeginSpell},
		{db.SkillSAElementwater, 0, effects.BeginSpell2},
		{db.SkillSAElementfire, 0, effects.BeginSpell3},
		{db.SkillSAElementground, 0, effects.BeginSpell4},
		{db.SkillSAElementwind, 0, effects.BeginSpell5},
	} {
		ctx := testutil.KnightEffectContext()
		m := &WorldMode{}
		m.applySkillCastNotify(ctx, network.SkillCastNotify{
			SkillID: tc.skill, SourceID: 2000000, TargetID: 300, DelayTime: 1000, Property: tc.property,
		})
		if len(m.effects().Active) != 3 || m.effects().Active[2].EffectID != tc.effect {
			t.Fatalf("skill %d: wrong casting color for property %d", tc.skill, tc.property)
		}
	}
}

func TestSageRealAssets(t *testing.T) {
	ctx := testutil.KnightEffectContext()
	ctx.Resources = testutil.RealDataManager(t)
	start := time.Unix(100, 0)
	for _, id := range []int{
		effects.Volcano, effects.Deluge, effects.ViolentGale, effects.LandProtector,
		effects.BottomVolcano, effects.BottomDeluge, effects.BottomViolent, effects.BottomLand,
		effects.MagicRod, effects.SpellBreaker, effects.Dispell,
		effects.FlameLauncher, effects.FrostWeapon, effects.LightningLoad, effects.SeismicWeapon,
	} {
		t.Run(strconv.Itoa(id), func(t *testing.T) {
			m := &WorldMode{}
			if !m.effects().AddAt(ctx, id, 300, start) {
				t.Fatal("missing effect")
			}
			for _, sound := range m.scheduledSounds {
				for _, path := range sound.paths {
					if _, err := ctx.Resources.ReadFile("data/wav/" + path); err != nil {
						t.Fatal(err)
					}
				}
			}
			draws := 0
			for _, tick := range []int{0, 15, 60, 150} {
				frame := render.NewFrame(800, 600)
				m.effects().Draw(frame, ctx, testutil.Projection(800, 600, 20, 20, 0), start.Add(time.Duration(tick)*db.EffectFrameDuration), false)
				v := reflect.ValueOf(frame).Elem()
				draws += v.FieldByName("worldCommands").Len() + v.FieldByName("worldBillboards").Len() + v.FieldByName("commands").Len()
			}
			if draws == 0 {
				t.Fatal("effect produced no geometry with real assets")
			}
		})
	}
}

func TestSniperEffectsWithClientAssets(t *testing.T) {
	ctx := testutil.ArcherEffectContext()
	ctx.Resources = testutil.RealDataManager(t)
	start := time.Unix(100, 0)
	projection := testutil.Projection(800, 600, 15, 20, 0)
	for _, id := range []int{effects.TrueSight, effects.Portal4, effects.TripleAttack2} {
		m := &WorldMode{}
		m.effects().AddBetween(ctx, id, 300, 2000000, start)
		for _, sound := range m.scheduledSounds {
			if _, err := ctx.Resources.ReadFile("data/wav/" + sound.paths[0]); err != nil {
				t.Fatal(err)
			}
			if id == effects.TripleAttack2 && sound.at.Sub(start) != 30*db.EffectFrameDuration {
				t.Fatal("Sharp Shooting sound should wait 30 ticks")
			}
		}
		at := 10
		if id == effects.TripleAttack2 {
			at = 60
		}
		frame := render.NewFrame(800, 600)
		m.effects().Draw(frame, ctx, projection, start.Add(time.Duration(at)*db.EffectFrameDuration), false)
		if reflect.ValueOf(frame).Elem().FieldByName("worldCommands").Len() == 0 {
			t.Fatalf("effect %d drew nothing", id)
		}
		frame = render.NewFrame(800, 600)
		m.effects().Draw(frame, ctx, projection, start.Add(90*db.EffectFrameDuration), false)
		if reflect.ValueOf(frame).Elem().FieldByName("worldCommands").Len() != 0 {
			t.Fatalf("effect %d still drew after particles faded", id)
		}
	}
	effect := effects.Instance{ActorOrigin: scene.Point3{X: 10}, ProjectileDirection: scene.Point3{Z: 8}, Starts: start}
	for i := 0; i < 16; i++ {
		from, to, _ := effects.SharpShootingLine(effect, i, 60)
		if math.Abs(to.X-from.X) > 1e-6 || to.Z-from.Z < 8 || from.Y < 1.6 || from.Y > 2.4 {
			t.Fatal("Sharp Shooting rays have incorrect direction, length, or height")
		}
		_, _, alpha := effects.SharpShootingLine(effect, i, 40)
		if alpha != 0 {
			t.Fatal("Sharp Shooting ray appeared before native delay")
		}
	}
}

func TestSoulLinkerSkillSequenceMatchesOriginalClient(t *testing.T) {
	for _, tc := range []struct {
		skill              uint16
		projectile, impact int
	}{
		{db.SkillSLStin, effects.Stin, effects.QuakeBody3},
		{db.SkillSLStun, effects.Stin3, effects.HitLine4},
	} {
		ctx := testutil.SoulLinkerEffectContext()
		m := &WorldMode{}
		starts := time.Unix(100, 0)
		hitAt := starts.Add(500 * time.Millisecond)
		action := network.ActorActionNotify{SourceID: 2000000, TargetID: 300, SkillID: tc.skill, Damage: 10}
		m.addSkillEffect(ctx, action, starts, hitAt)
		m.addSkillHitEffect(ctx, action, hitAt)
		if len(m.effects().Active) != 3 {
			t.Fatalf("skill %d: effects = %+v", tc.skill, m.effects().Active)
		}
		for i, id := range []int{tc.projectile, tc.impact, effects.BlueHit} {
			effect := m.effects().Active[i]
			want := starts
			if i == 2 {
				want = hitAt
			}
			if effect.EffectID != id || effect.ActorID != 300 || !effect.Starts.Equal(want) {
				t.Fatalf("skill %d: wrong effect order, owner or timing: %+v", tc.skill, effect)
			}
		}
		projectile := m.effects().Active[0]
		if projectile.ActorOrigin.X != 10.5 || projectile.ActorOrigin.Z != 20.5 || projectile.ProjectileDirection != (scene.Point3{X: 1}) {
			t.Fatalf("projectile starts at target instead of caster: %+v", projectile)
		}
		if len(m.scheduledSounds) != 1 {
			t.Fatalf("skill %d: sounds = %+v", tc.skill, m.scheduledSounds)
		}
		m.effects().Active = nil
		action.Damage = 0
		m.addSkillHitEffect(ctx, action, hitAt)
		if len(m.effects().Active) != 0 {
			t.Fatal("a miss must not flash the target blue")
		}
	}
}

func TestEstinNativeMotionAndTrail(t *testing.T) {
	distance, alpha := effects.EstinHead(10)
	wantDistance, speed := .8, .8
	for i := 2; i <= 10; i++ {
		speed *= 1.02
		wantDistance += speed
	}
	if math.Abs(distance-wantDistance/5) > 1e-9 || alpha != 225.0/255 {
		t.Fatalf("end of rise = %g/%g", distance, alpha)
	}
	if _, alpha := effects.EstinHead(55); alpha != 0 {
		t.Fatal("Estin head should finish fading at tick 55")
	}
	ctx := testutil.SoulLinkerEffectContext()
	m := &WorldMode{}
	soulLinkerTestTextures(m)
	starts := time.Unix(100, 0)
	m.effects().AddBetween(ctx, effects.Stin, 300, 2000000, starts)
	projection := testutil.Projection(800, 600, 14.5, 20.5, 0)
	draw := func() *render.Frame {
		frame := render.NewFrame(800, 600)
		m.effects().Draw(frame, ctx, projection, starts.Add(16*db.EffectFrameDuration), false)
		return frame
	}
	frame := draw()
	commands := reflect.ValueOf(frame).Elem().FieldByName("worldCommands")
	if commands.Len() != 1 {
		t.Fatal("Estin should draw one batched wind trail")
	}
	vertices := commands.Index(0).FieldByName("Vertices")
	if vertices.Len() != 16 {
		t.Fatalf("trail vertices=%d, want four squares", vertices.Len())
	}
	for i := 0; i < 16; i++ {
		if math.Abs(vertices.Index(i).FieldByName("Y").Float()-m.effects().Active[0].ActorOrigin.Y) > 1e-5 {
			t.Fatal("Estin must use world-horizontal quads as in RenderSTIN")
		}
	}
	ctx.World.Player.X, ctx.World.Player.Y = 50, 50
	delete(ctx.World.Actors, 300)
	again := reflect.ValueOf(draw()).Elem().FieldByName("worldCommands").Index(0).FieldByName("Vertices")
	for i := 0; i < vertices.Len(); i++ {
		for _, axis := range []string{"X", "Y", "Z"} {
			if vertices.Index(i).FieldByName(axis).Float() != again.Index(i).FieldByName(axis).Float() {
				t.Fatal("movement/removal bent an already launched projectile")
			}
		}
	}
}

func TestEstunEmitsParticlesAndStops(t *testing.T) {
	ctx := testutil.SoulLinkerEffectContext()
	m := &WorldMode{}
	soulLinkerTestTextures(m)
	starts := time.Unix(100, 0)
	m.effects().AddBetween(ctx, effects.Stin3, 300, 2000000, starts)
	effect := m.effects().Active[0]
	spec, _ := effects.Lookup(effects.Stin3)
	projection := testutil.Projection(800, 600, 14.5, 20.5, 0)
	for _, tc := range []struct{ frame, count int }{{-1, 0}, {0, 0}, {1, 4}, {10, 40}, {199, 196}, {230, 76}, {250, 0}} {
		frame := render.NewFrame(800, 600)
		m.effects().DrawRisingBuffParticles(frame, ctx, projection, spec.Components[0], effect, starts.Add(time.Duration(tc.frame)*db.EffectFrameDuration))
		draws := reflect.ValueOf(frame).Elem().FieldByName("worldBillboards")
		if draws.Len() != tc.count {
			t.Fatalf("tick %d: %d particles, want %d", tc.frame, draws.Len(), tc.count)
		}
		for i := 0; i < draws.Len(); i++ {
			if draws.Index(i).FieldByName("Center").Index(1).Float() <= effect.ActorOrigin.Y {
				t.Fatal("particles must rise above the emitter")
			}
		}
	}
}

func TestEstunImpactStreaksHaveBoundedLifetime(t *testing.T) {
	ctx := testutil.SoulLinkerEffectContext()
	m := &WorldMode{}
	soulLinkerTestTextures(m)
	spec, _ := effects.Lookup(effects.HitLine4)
	starts := time.Unix(100, 0)
	effect := effects.Instance{EffectID: effects.HitLine4, ActorID: 300, Starts: starts}
	for _, tick := range []int{5, 30, 80, 200} {
		frame := render.NewFrame(800, 600)
		m.effects().DrawHitLineBurst(frame, ctx, spec.Components[0], effect, 18.5, 20.5, 0, starts.Add(time.Duration(tick)*db.EffectFrameDuration))
		commands := reflect.ValueOf(frame).Elem().FieldByName("worldCommands")
		if (commands.Len() > 0) != (tick == 30) {
			t.Fatalf("tick %d: impact draws=%d", tick, commands.Len())
		}
		if tick == 30 {
			vertices := commands.Index(0).FieldByName("Vertices")
			indices := commands.Index(0).FieldByName("Indices")
			for i := 0; i < indices.Len(); i++ {
				if int(indices.Index(i).Uint()) >= vertices.Len() {
					t.Fatal("invalid streak mesh index")
				}
			}
		}
	}
}

func TestEsmaSequenceAndHoming(t *testing.T) {
	ctx := testutil.SoulLinkerEffectContext()
	starts := time.Unix(100, 0)
	m := &WorldMode{}
	soulLinkerTestTextures(m)
	m.addSkillEffect(ctx, network.ActorActionNotify{SourceID: 2000000, TargetID: 300, SkillID: db.SkillSLSma, Damage: 100}, starts, starts.Add(500*time.Millisecond))
	if len(m.effects().Active) != 4 {
		t.Fatalf("Esma effects = %+v", m.effects().Active)
	}
	for i, id := range []int{effects.FourWayBody, effects.Stin2, effects.HitLine6, effects.HitTexture} {
		if e := m.effects().Active[i]; e.EffectID != id || e.ActorID != 300 || !e.Starts.Equal(starts) {
			t.Fatalf("wrong Esma component/timing: %+v", e)
		}
	}
	if len(m.scheduledSounds) != 5 {
		t.Fatalf("Esma sounds = %+v", m.scheduledSounds)
	}
	for i, sound := range m.scheduledSounds {
		if !sound.at.Equal(starts.Add(time.Duration(5+15*i) * db.EffectFrameDuration)) {
			t.Fatalf("wrong Esma sound timing: %+v", sound)
		}
	}
	e := m.effects().Active[1]
	e.Esma.Advance(ctx, e, starts.Add(10*db.EffectFrameDuration))
	left, right := e.Esma.Heads[0].Samples[0], e.Esma.Heads[1].Samples[0]
	if left.Position.Z <= e.ActorOrigin.Z || right.Position.Z >= e.ActorOrigin.Z || math.Abs(left.Position.X-right.Position.X) > 1e-8 {
		t.Fatalf("Esma must leave the caster in opposite arcs: %v %v", left, right)
	}
	// A second render pass in the same tick must not advance the simulation.
	before := *e.Esma
	e.Esma.Advance(ctx, e, starts.Add(10*db.EffectFrameDuration))
	if *e.Esma != before {
		t.Fatal("drawing twice advanced Esma")
	}
	actor := ctx.World.Actors[300]
	actor.X, actor.Y = 24, 30
	ctx.World.Actors[300] = actor
	e.Esma.Advance(ctx, e, starts.Add(11*db.EffectFrameDuration))
	wantHistory := before.Heads[0].Samples[1]
	wantHistory.Alpha *= .8
	if e.Esma.Heads[0].Samples[2] != wantHistory {
		t.Fatal("moving target rewrote an old trail position")
	}
	for i := range e.Esma.Heads {
		step := scene.Sub3(e.Esma.Heads[i].Samples[0].Position, before.Heads[i].Samples[0].Position)
		if math.Abs(math.Hypot(step.X, step.Z)-.4) > 1e-8 {
			t.Fatal("moving target teleported an Esma head")
		}
	}
	delete(ctx.World.Actors, 300)
	e.Esma.Advance(ctx, e, starts.Add(100*db.EffectFrameDuration))
	for _, head := range e.Esma.Heads {
		for _, sample := range head.Samples {
			if sample.Alpha > 0 {
				t.Fatal("Esma must fade when its target leaves")
			}
		}
	}
}

func TestSoulLinkerParticleAndKaizelTiming(t *testing.T) {
	ctx := testutil.SoulLinkerEffectContext()
	starts := time.Unix(100, 0)
	m := &WorldMode{}
	soulLinkerTestTextures(m)
	projection := testutil.Projection(800, 600, 18.5, 20.5, 0)
	for _, tc := range []struct{ id, tick, count int }{
		{effects.Hated, 0, 0}, {effects.Hated, 1, 4}, {effects.Hated, 5, 8}, {effects.Hated, 135, 0},
		{effects.SmaReady, 39, 0}, {effects.SmaReady, 41, 4}, {effects.SmaReady, 175, 0},
	} {
		frame := render.NewFrame(800, 600)
		spec, _ := effects.Lookup(tc.id)
		m.effects().AddAt(ctx, tc.id, 300, starts)
		effect := m.effects().Active[len(m.effects().Active)-1]
		m.effects().DrawRisingBuffParticles(frame, ctx, projection, spec.Components[0], effect, starts.Add(time.Duration(tc.tick)*db.EffectFrameDuration))
		if n := reflect.ValueOf(frame).Elem().FieldByName("worldBillboards").Len(); n != tc.count {
			t.Fatalf("effect %d tick %d: got %d particles, want %d", tc.id, tc.tick, n, tc.count)
		}
	}
	spec, _ := effects.Lookup(effects.Kaizel)
	for _, tick := range []int{14, 15, 25, 45} {
		frame := render.NewFrame(800, 600)
		m.effects().DrawKaizelEffect(frame, ctx, spec.Components[1], effects.Instance{Starts: starts}, 18.5, 20.5, 0, starts.Add(time.Duration(tick)*db.EffectFrameDuration))
		commands := reflect.ValueOf(frame).Elem().FieldByName("worldCommands")
		if (commands.Len() > 0) != (tick == 15 || tick == 25) {
			t.Fatalf("Kaizel burst at wrong tick %d", tick)
		}
		if commands.Len() > 0 && commands.Index(0).FieldByName("Vertices").Len() != 8*3*4 {
			t.Fatal("Kaizel needs eight blades of three quads each")
		}
	}
}

func TestSoulLinkerBuffParticlesKeepEmissionPositions(t *testing.T) {
	for _, id := range []int{effects.Hated, effects.Kaizel, effects.SmaReady} {
		ctx := testutil.SoulLinkerEffectContext()
		m := &WorldMode{}
		soulLinkerTestTextures(m)
		starts := time.Unix(100, 0)
		m.effects().AddAt(ctx, id, 300, starts)
		projection := testutil.Projection(800, 600, 18.5, 20.5, 0)
		first := 0
		if id == effects.SmaReady {
			first = 40
		}
		draw := func(tick int) reflect.Value {
			frame := render.NewFrame(800, 600)
			m.effects().Draw(frame, ctx, projection, starts.Add(time.Duration(tick)*db.EffectFrameDuration), false)
			return reflect.ValueOf(frame).Elem().FieldByName("worldBillboards")
		}
		before := draw(first + 1)
		actor := ctx.World.Actors[300]
		actor.X += 10
		ctx.World.Actors[300] = actor
		after := draw(first + 5)
		if before.Len() != 4 || after.Len() != 8 {
			t.Fatalf("effect %d: particles before/after movement = %d/%d", id, before.Len(), after.Len())
		}
		for i := 0; i < 4; i++ {
			for _, axis := range []int{0, 2} {
				if before.Index(i).FieldByName("Center").Index(axis).Float() != after.Index(i).FieldByName("Center").Index(axis).Float() {
					t.Fatalf("effect %d: movement relocated an emitted particle", id)
				}
			}
			if after.Index(i+4).FieldByName("Center").Index(0).Float() < 27 {
				t.Fatalf("effect %d: new particles did not follow the actor", id)
			}
		}
	}
}

func TestSoulLinkerMonsterSkillSuccessOwnsActivation(t *testing.T) {
	for _, skill := range []uint16{db.SkillSLSwoo, db.SkillSLSke, db.SkillSLSka} {
		ctx := testutil.SoulLinkerEffectContext()
		m := &WorldMode{}
		m.handleNetworkPacket(ctx, testutil.SoulLinkerSkillPacket(skill, 0), time.Now())
		if ctx.World.Actors[300].Opt3State != 0 || len(m.effects().Active) != 0 {
			t.Fatalf("failed skill %d changed monster effects", skill)
		}
		m.handleNetworkPacket(ctx, testutil.SoulLinkerSkillPacket(skill, 1), time.Now())
		if ctx.World.Actors[300].Opt3State == 0 {
			t.Fatalf("skill %d still requires a status-icon packet", skill)
		}
		m.upsertNetworkActor(ctx, network.ActorEntry{ID: 300, Job: 1002, X: 18, Y: 20, HasState: true, HasOpt3State: true})
		if m.effects().HasActive(effects.BabyBody, 300, time.Now()) || m.effects().HasActive(effects.SteelBody, 300, time.Now()) || m.effects().HasActive(effects.AsuraBodyMonster, 300, time.Now()) {
			t.Fatalf("skill %d ignored a newer actor snapshot", skill)
		}
	}
}

func TestKaupeRendersEveryDodge(t *testing.T) {
	ctx := testutil.SoulLinkerEffectContext()
	m := &WorldMode{}
	var w network.Writer
	w.Uint16(network.PacketZCNotifyEffect2)
	w.Uint32(300)
	w.Uint32(effects.StormKick4)
	packet := network.Packet{ID: network.PacketZCNotifyEffect2, Data: w.Bytes()}
	m.handleNetworkPacket(ctx, packet, time.Now())
	first := m.effects().Active[0]
	m.handleNetworkPacket(ctx, packet, time.Now())
	if len(m.effects().Active) != 2 || len(m.scheduledSounds) != 2 || m.effects().Active[0] != first || m.effects().Active[0].SoulLights == m.effects().Active[1].SoulLights {
		t.Fatal("successive dodges must have independent effects and sounds")
	}
}

func TestSoulLinkAndKaupeParticleTimingIndependentOfFrameRate(t *testing.T) {
	ctx := testutil.SoulLinkerEffectContext()
	for _, id := range []int{effects.SoulLink, effects.StormKick4} {
		m := &WorldMode{}
		soulLinkerTestTextures(m)
		start := time.Unix(100, 0)
		m.effects().AddAt(ctx, id, 300, start)
		e := m.effects().Active[0]
		one, many := &effects.SoulLights{}, &effects.SoulLights{}
		right := scene.Point3{X: 1}
		one.Advance(ctx, e, right, start.Add(100*db.EffectFrameDuration))
		for i := 0; i <= 100; i++ {
			many.Advance(ctx, e, right, start.Add(time.Duration(i)*db.EffectFrameDuration))
		}
		if !reflect.DeepEqual(one, many) {
			t.Fatalf("effect %d depends on render rate", id)
		}
		projection := testutil.Projection(800, 600, 18.5, 20.5, 0)
		frame := render.NewFrame(800, 600)
		m.effects().Draw(frame, ctx, projection, start.Add(30*db.EffectFrameDuration), false)
		if reflect.ValueOf(frame).Elem().FieldByName("worldCommands").Len() == 0 {
			t.Fatalf("effect %d drew nothing", id)
		}
		e.SoulLights.Advance(ctx, e, right, start.Add(300*db.EffectFrameDuration))
		if len(e.SoulLights.Particles) != 0 {
			t.Fatalf("effect %d particles never die", id)
		}
		if id == effects.StormKick4 && (len(m.scheduledSounds) != 1 || m.scheduledSounds[0].at.Sub(start) != 15*db.EffectFrameDuration) {
			t.Fatal("Kaupe dodge sound timing")
		}
	}
	for _, tc := range []struct {
		age, layer int
		alpha      float64
	}{{0, 10, 0}, {19, 10, 240.0 / 255}, {100, 10, 240.0 / 255}, {148, 10, 0}, {129, 8, 0}} {
		_, a := effects.SoulLinkGlyph(tc.age, tc.layer)
		if math.Abs(a-tc.alpha) > 1e-8 {
			t.Fatalf("glyph age=%d layer=%d alpha=%v", tc.age, tc.layer, a)
		}
	}
}

func TestStalkerClientAssets(t *testing.T) {
	ctx := testutil.ArcherEffectContext()
	ctx.Resources = testutil.RealDataManager(t)
	ctx.World.Player.Job, ctx.Session.Selected.Job = db.JobRogueH, db.JobRogueH
	for _, sex := range []uint8{0, 1} {
		view, status := sprites.LoadPlayer(ctx.Resources, ctx.Session.Selected, sex, false)
		if view == nil {
			t.Fatalf("Stalker sprite missing: %s", status)
		}
		for dir := 0; dir < 8; dir++ {
			for _, family := range []int{scene.SpriteActionIdle, sprites.ActionPCReadyFight, sprites.ActionPCSkill} {
				if _, action, ok := sprites.ResolveAction(view.Body.Act, family, dir); !ok || len(action.Animations) == 0 {
					t.Fatal("Stalker casting/release ACT missing")
				}
			}
		}
	}
	start := time.Unix(100, 0)
	projection := testutil.Projection(800, 600, 20, 20, 0)
	for _, tc := range []struct{ id, tick, billboards, vertices int }{
		{495, 0, 0, 0}, {495, 50, 32, 0}, {495, 109, 0, 0},
		{496, 10, 0, 180}, {496, 44, 0, 0},
		{db.SkillEffectChaseWalkSmoke, 0, 1, 0},
	} {
		m := &WorldMode{}
		if !m.effects().AddAt(ctx, tc.id, 300, start) {
			t.Fatalf("missing effect %d", tc.id)
		}
		frame := render.NewFrame(800, 600)
		m.effects().Draw(frame, ctx, projection, start.Add(time.Duration(tc.tick)*db.EffectFrameDuration), false)
		draws := reflect.ValueOf(frame).Elem()
		commands := draws.FieldByName("worldCommands")
		vertices := 0
		for i := 0; i < commands.Len(); i++ {
			vertices += commands.Index(i).FieldByName("Vertices").Len()
		}
		if tc.id == 495 && tc.tick == 50 {
			billboards := draws.FieldByName("worldBillboards")
			for i := 0; i < billboards.Len(); i++ {
				draw := billboards.Index(i)
				c := draw.FieldByName("Center")
				dx, dy, dz := c.Index(0).Float()-20.5, c.Index(1).Float()-.07, c.Index(2).Float()-20.5
				radius := math.Sqrt(dx*dx + dy*dy + dz*dz)
				if radius < 3.19 || radius > 9.01 || draw.FieldByName("Options").FieldByName("Blend").Int() != int64(render.BlendLighter) {
					t.Fatal("Full Strip fragments need native travel distance and additive blending")
				}
			}
		}
		if draws.FieldByName("worldBillboards").Len() != tc.billboards || vertices != tc.vertices {
			t.Fatalf("effect %d tick %d: billboards=%d vertices=%d", tc.id, tc.tick, draws.FieldByName("worldBillboards").Len(), vertices)
		}
	}
	guard := &WorldMode{}
	guard.effects().AddAt(ctx, 496, 300, start)
	before := render.NewFrame(800, 600)
	guard.effects().Draw(before, ctx, projection, start.Add(10*db.EffectFrameDuration), false)
	actorBefore := ctx.World.Actors[300]
	actorAfter := actorBefore
	actorAfter.Dir = (actorBefore.Dir + 4) % 8
	actorAfter.X += 1
	ctx.World.Actors[300] = actorAfter
	after := render.NewFrame(800, 600)
	guard.effects().Draw(after, ctx, projection, start.Add(10*db.EffectFrameDuration), false)
	for c := 0; c < 3; c++ {
		a := reflect.ValueOf(before).Elem().FieldByName("worldCommands").Index(c).FieldByName("Vertices")
		b := reflect.ValueOf(after).Elem().FieldByName("worldCommands").Index(c).FieldByName("Vertices")
		for v := 0; v < a.Len(); v++ {
			if math.Abs(b.Index(v).FieldByName("X").Float()-a.Index(v).FieldByName("X").Float()-1) > .00001 || b.Index(v).FieldByName("Z").Float() != a.Index(v).FieldByName("Z").Float() {
				t.Fatal("Preserve must follow position but keep its initial rotation")
			}
		}
	}
	ctx.World.Actors[300] = actorBefore
	for _, id := range []int{392, 495, 496, 501} {
		spec, _ := effects.Lookup(id)
		for _, sound := range spec.Sfx {
			if _, err := ctx.Resources.ReadFile("data/wav/" + sound); err != nil {
				t.Fatalf("effect %d sound %s: %v", id, sound, err)
			}
		}
	}
	m := &WorldMode{}
	view := m.effects().EffectSpriteView(ctx.Resources, "smoke")
	if view == nil || len(view.Act.Actions[0].Animations) == 0 {
		t.Fatal("native smoke ACT missing")
	}
	m.effects().AddAt(ctx, db.SkillEffectChaseWalkSmoke, 300, start)
	e := m.effects().Active[0]
	if e.Expires.Sub(start) != time.Duration(len(view.Act.Actions[0].Animations))*(5*db.EffectFrameDuration/6) {
		t.Fatal("smoke must play once at native speed")
	}
	actor := ctx.World.Actors[300]
	actor.X += 10
	ctx.World.Actors[300] = actor
	frame := render.NewFrame(800, 600)
	m.effects().Draw(frame, ctx, projection, start, false)
	draw := reflect.ValueOf(frame).Elem().FieldByName("worldBillboards").Index(0)
	if math.Abs(draw.FieldByName("Center").Index(0).Float()-20.5) > .001 {
		t.Fatal("smoke followed the hidden actor instead of staying at release position")
	}
}

func TestStarGladiatorNativeAssetsAndGeometry(t *testing.T) {
	ctx := testutil.ArcherEffectContext()
	ctx.Resources = testutil.RealDataManager(t)
	start := time.Now()
	projection := testutil.Projection(800, 600, 10, 20, 0)
	for _, tc := range []struct{ id, tick, end int }{{418, 50, -1}, {572, 20, 301}, {488, 90, 205}, {487, 20, 151}, {475, 30, -1}, {484, 30, -1}} {
		m := &WorldMode{}
		m.effects().AddAt(ctx, tc.id, 2000000, start)
		for _, tick := range []int{tc.tick, tc.end} {
			if tick < 0 {
				continue
			}
			frame := render.NewFrame(800, 600)
			m.effects().Draw(frame, ctx, projection, start.Add(time.Duration(tick)*db.EffectFrameDuration), false)
			v := reflect.ValueOf(frame).Elem()
			commands := v.FieldByName("worldCommands")
			count := commands.Len() + v.FieldByName("worldBillboards").Len()
			if (count > 0) != (tick == tc.tick) {
				t.Fatalf("effect %d tick %d draws=%d", tc.id, tick, count)
			}
			if tc.id == 488 && tick == 90 {
				mesh := commands.Index(0).FieldByName("Vertices")
				if mesh.Len() != 8*21*4 {
					t.Fatalf("Comfort must have eight 315-degree rings: %d vertices", mesh.Len())
				}
			}
		}
		for _, sound := range m.scheduledSounds {
			for _, path := range sound.paths {
				if _, err := ctx.Resources.ReadFile("data/wav/" + path); err != nil {
					t.Fatal(err)
				}
			}
		}
	}
	for _, path := range []string{"T_변신.wav", "T_등록.wav", "_blind.wav"} {
		if _, err := ctx.Resources.ReadFile("data/wav/effect/" + path); err != nil {
			t.Fatal(err)
		}
	}
	ctx.Session.Selected.Job = db.JobStar2
	for _, sex := range []uint8{0, 1} {
		if view, message := sprites.LoadPlayer(ctx.Resources, ctx.Session.Selected, sex, false); view == nil {
			t.Fatal(message)
		}
	}
}

func TestBashImpactOnlyOnHit(t *testing.T) {
	ctx := testutil.SwordmanContext()
	starts := time.Unix(100, 0)
	for _, damage := range []int32{0, 50} {
		m := &WorldMode{}
		action := network.ActorActionNotify{SkillID: db.SkillSMBash, SourceID: 2000000, TargetID: 300, Damage: damage}
		m.addSkillBeginEffect(ctx, action, starts)
		m.addSkillHitEffect(ctx, action, starts.Add(200*time.Millisecond))
		want := 1
		if damage > 0 {
			want = 2
		}
		if len(m.effects().Active) != want || m.effects().Active[0].EffectID != effects.BashBegin || m.effects().Active[0].ActorID != 2000000 {
			t.Fatalf("damage %d: effects=%+v", damage, m.effects().Active)
		}
		if damage > 0 && (m.effects().Active[1].ActorID != 300 || m.effects().Active[1].EffectID != effects.BashHit || !m.effects().Active[1].Starts.Equal(starts.Add(200*time.Millisecond))) {
			t.Fatal("Bash impact did not follow the victim/hit time")
		}
	}
}

func TestTaekwonEffectsRenderWithNativeAssets(t *testing.T) {
	ctx := testutil.ArcherEffectContext()
	ctx.Resources = testutil.RealDataManager(t)
	for _, job := range []int16{db.JobTaekwon, db.JobStar, db.JobLinker} {
		ctx.World.Player.Job, ctx.Session.Selected.Job = job, job
		ctx.World.Player.Weapon, ctx.Session.Selected.Weapon = 0, 0
		for _, sex := range []uint8{0, 1} {
			view, status := sprites.LoadPlayer(ctx.Resources, ctx.Session.Selected, sex, false)
			if view == nil {
				t.Fatalf("job %d sprite missing: %s", job, status)
			}
			for _, pose := range []struct{ family, motion int }{{sprites.ActionPCSkill, 5}, {sprites.ActionPCAttack1, 4}, {sprites.ActionPCAttack2, 8}, {sprites.ActionPCAttack3, 7}, {scene.SpriteActionPickup, 1}, {scene.SpriteActionWalk, 5}} {
				if job == db.JobLinker && pose.family == sprites.ActionPCAttack2 {
					continue // Soul Linkers cannot learn Heel Drop.
				}
				for dir := 0; dir < 8; dir++ {
					_, action, ok := sprites.ResolveAction(view.Body.Act, pose.family, dir)
					if !ok || len(action.Animations) <= pose.motion {
						t.Fatalf("job %d sex %d family %d direction %d lacks motion %d", job, sex, pose.family, dir, pose.motion)
					}
				}
			}
		}
	}
	start := time.Unix(100, 0)
	projection := testutil.Projection(800, 600, 15, 20, 0)
	for _, tc := range []struct{ id, tick, end int }{{411, 60, 200}, {430, 40, 90}, {435, 20, 110}, {439, 20, 60}, {444, 10, 30}, {447, 30, 301}, {448, 30, 301}, {461, 20, 110}, {467, 20, 251}, {473, 20, 251}} {
		m := &WorldMode{}
		if !m.effects().AddBetween(ctx, tc.id, 300, 2000000, start) {
			t.Fatalf("effect %d has no implementation", tc.id)
		}
		spec, _ := effects.Lookup(tc.id)
		for _, component := range spec.Components {
			for _, file := range append([]string{component.TextureFile}, component.TextureFiles...) {
				if file != "" && m.effects().EffectFileTexture(ctx.Resources, file) == nil {
					t.Fatalf("effect %d missing texture %s", tc.id, file)
				}
			}
		}
		for _, tick := range []int{tc.tick, tc.end} {
			frame := render.NewFrame(800, 600)
			m.effects().Draw(frame, ctx, projection, start.Add(time.Duration(tick)*db.EffectFrameDuration), false)
			commands := reflect.ValueOf(frame).Elem()
			count := commands.FieldByName("worldCommands").Len() + commands.FieldByName("worldBillboards").Len()
			if (count > 0) != (tick == tc.tick) {
				t.Fatalf("effect %d tick %d: draws %d", tc.id, tick, count)
			}
		}
	}
	for _, sound := range []string{"T_낙법.wav", "T_피링.wav", "T_바람방출.wav", "T_벽튕김.wav", "T_회피2.wav", "T_돌려차기.wav", "T_내려찍기.wav", "T_날라차기.wav", "T_회오리차기.wav", "T_효과음1.wav"} {
		if _, err := ctx.Resources.ReadFile("data/wav/effect/" + sound); err != nil {
			t.Fatal(err)
		}
	}
}

func TestTaekwonKickEffectsHaveNativeOwnershipAndTiming(t *testing.T) {
	ctx := testutil.ArcherEffectContext()
	start := time.Unix(100, 0)
	for _, tc := range []struct {
		skill   uint16
		effects map[int]uint32
	}{
		{413, map[int]uint32{}}, // The no-damage packet owns Tornado Kick.
		{415, map[int]uint32{effects.PressedBody: 300, effects.HitLine6: 300}},
		{417, map[int]uint32{effects.SpinedBody2: 300, effects.HitLine4: 300}},
		{419, map[int]uint32{effects.KickedBody: 300, effects.HitLine5: 300}},
		{421, map[int]uint32{effects.QuakeBody2: 300, effects.Chemical3: 2000000, effects.JumpKick: 2000000}},
	} {
		m := &WorldMode{}
		a := network.ActorActionNotify{SkillID: tc.skill, SourceID: 2000000, TargetID: 300, Damage: 10, HitCount: 1, Action: network.ActorActionSkill}
		m.addSkillEffect(ctx, a, start, start.Add(time.Second))
		got := map[int]uint32{}
		for _, e := range m.effects().Active {
			got[e.EffectID] = e.ActorID
			if e.Starts != start {
				t.Fatal("kick effects must start when the attack begins")
			}
			if e.EffectID == effects.Chemical3 && (e.TargetID != 300 || e.ProjectileDirection.X <= 0) {
				t.Fatal("Flying Kick streaks must point from caster to victim")
			}
		}
		if !reflect.DeepEqual(got, tc.effects) {
			t.Fatalf("skill %d effects: %v want %v", tc.skill, got, tc.effects)
		}
		m.effects().Active = nil
		m.addSkillHitEffect(ctx, a, start.Add(time.Second))
		if len(m.effects().Active) != 0 {
			t.Fatal("Taekwon skills must not add the ordinary hit flash")
		}
	}
}

func TestStoneFlingFlight(t *testing.T) {
	ctx := testutil.ThiefEffectContext()
	m := &WorldMode{}
	starts := time.Unix(100, 0)
	m.addSkillBeforeHitEffect(ctx, network.ActorActionNotify{SkillID: db.SkillTFThrowstone, SourceID: 2000000, TargetID: 300}, starts)
	effect := m.effects().Active[0]
	first, angle, alpha := effects.ThrowItemAt(effect, 1)
	if math.Abs(first.X-10.9) > 1e-9 || angle != scene.DegreesToRadians(10) || alpha != 255 {
		t.Fatalf("stone must advance two native units and rotate ten degrees per tick: %+v, %g, %d", first, angle, alpha)
	}
	middle, _, _ := effects.ThrowItemAt(effect, 8)
	if math.Abs(middle.Y-effect.ActorOrigin.Y-2) > 1e-9 {
		t.Fatal("stone's arc must rise ten native units above its launch height")
	}
	near := effect
	near.ProjectileDirection.X = 1.2
	_, _, nearAlpha := effects.ThrowItemAt(near, 10)
	_, _, farAlpha := effects.ThrowItemAt(effect, 10)
	if nearAlpha >= farAlpha {
		t.Fatal("stone flight duration must depend on distance")
	}
	m.mapScene.Textures = map[string]*render.Image{"__effectfile_유저인터페이스/item/돌.bmp|blackkey=true": render.NewImage(8, 8)}
	frame := render.NewFrame(800, 600)
	m.effects().Draw(frame, ctx, testutil.Projection(800, 600, 14, 20, 0), starts.Add(6*db.EffectFrameDuration), false)
	if n := reflect.ValueOf(frame).Elem().FieldByName("worldBillboards").Len(); n != 4 {
		t.Fatalf("stone and three afterimages: got %d", n)
	}
}

func TestWhitesmithEffectsWithClientAssets(t *testing.T) {
	ctx := testutil.ArcherEffectContext()
	ctx.Resources = testutil.RealDataManager(t)
	start := time.Unix(100, 0)
	projection := testutil.Projection(800, 600, 20, 20, 0)
	for _, tc := range []struct {
		effect int
		sound  string
	}{
		{effects.Meltdown, "effect\\black_overthrust.wav"}, {effects.CartBoost, "effect\\ef_incagility.wav"},
	} {
		m := &WorldMode{}
		m.effects().AddAt(ctx, tc.effect, 300, start)
		if len(m.scheduledSounds) != 1 || m.scheduledSounds[0].paths[0] != tc.sound || !m.scheduledSounds[0].at.Equal(start) {
			t.Fatal("buff must play its native sound on activation")
		}
		if _, err := ctx.Resources.ReadFile("data/wav/" + tc.sound); err != nil {
			t.Fatal(err)
		}
		frame := render.NewFrame(800, 600)
		m.effects().Draw(frame, ctx, projection, start.Add(20*db.EffectFrameDuration), false)
		v := reflect.ValueOf(frame).Elem()
		if v.FieldByName("commands").Len()+v.FieldByName("worldCommands").Len() == 0 {
			t.Fatalf("effect %d drew nothing with client STR assets", tc.effect)
		}
	}
	m := &WorldMode{}
	m.effects().AddAt(ctx, effects.CartTermination, 300, start)
	if _, err := ctx.Resources.ReadFile("data/wav/" + m.scheduledSounds[0].paths[0]); err != nil {
		t.Fatal(err)
	}
	for _, tick := range []int{29, 55, 93} {
		frame := render.NewFrame(800, 600)
		at := start.Add(time.Duration(tick) * db.EffectFrameDuration)
		m.effects().Draw(frame, ctx, projection, at, false)
		commands := reflect.ValueOf(frame).Elem().FieldByName("worldCommands")
		if tick != 55 {
			if commands.Len() != 0 {
				t.Fatalf("unexpected Cart Termination particles at tick %d", tick)
			}
			continue
		}
		if commands.Len() != 1 {
			t.Fatal("Cart Termination did not draw its particle burst")
		}
		vertices := commands.Index(0).FieldByName("Vertices")
		if vertices.Len() < 24*4 || vertices.Len() > 48*4 {
			t.Fatal("incorrect size of Cart Termination burst")
		}
		center := effects.EffectWorldAnchor(ctx, m.effects().Active[0], at)
		for i := 0; i < vertices.Len(); i += 4 {
			var x, y, z float64
			for j := 0; j < 4; j++ {
				v := vertices.Index(i + j)
				x += v.FieldByName("X").Float() / 4
				y += v.FieldByName("Y").Float() / 4
				z += v.FieldByName("Z").Float() / 4
				if math.Abs(v.FieldByName("ColorA").Float()-224.0/255) > .001 {
					t.Fatal("incorrect particle fade")
				}
			}
			radius := math.Sqrt((x-center.X)*(x-center.X) + (y-center.Y)*(y-center.Y) + (z-center.Z)*(z-center.Z))
			if math.Abs(radius-5.2) > .001 {
				t.Fatal("particles must travel 26 native units before slowing")
			}
		}
	}
}

func TestWizardSoundTiming(t *testing.T) {
	ctx := testutil.KnightEffectContext()
	start := time.Unix(100, 0)
	m := &WorldMode{}
	m.effects().AddAt(ctx, effects.FirePillar, 300, start)
	if len(m.scheduledSounds) != 1 || !m.scheduledSounds[0].at.Equal(start.Add(75*db.EffectFrameDuration)) {
		t.Fatal("Fire Pillar must delay its sound until the pillar erupts")
	}
	m.scheduledSounds = nil
	spec, _ := effects.Lookup(effects.Quagmire)
	for id := uint32(9000); id < 10200; id++ {
		m.effects().ScheduleWorldEffectSound(start, spec, effects.Instance{EffectID: effects.Quagmire, ActorID: id, Starts: start, X: 15, Y: 20})
	}
	if n := len(m.scheduledSounds); n < 60 || n > 140 {
		t.Fatalf("Quagmire played %d sounds for 1200 cells, expected roughly one in twelve", n)
	}
	for _, sound := range m.scheduledSounds {
		x, y, ok := scheduledSoundPosition(ctx, sound, start)
		if !ok || x != 15 || y != 20 {
			t.Fatal("Quagmire sound must stay at its cell")
		}
	}
}

func TestWizardMeteorKeepsReleasePose(t *testing.T) {
	ctx := testutil.KnightEffectContext()
	ctx.World.Player.Job = db.JobWizard
	ctx.World.Player.Dir = 3
	actions := make([]res.ACTAction, 13*8)
	for i := range actions {
		actions[i].Animations = make([]res.ACTAnimation, 4)
	}
	m := &WorldMode{playerView: &sprites.Humanoid{Body: &scene.SpriteView{Act: &res.ACT{Actions: actions}}}}
	for _, x := range []uint16{5, 30} {
		m.applyGroundSkillNotify(ctx, network.GroundSkillNotify{SkillID: db.SkillWZMeteor, SourceID: 2000000, X: x, Y: 20})
		anim := m.actorAnims[100]
		if ctx.World.Player.Dir != 3 || anim.actionFamily != sprites.ActionPCSkill || anim.play || !anim.hasFixedMotion || anim.fixedMotion != 3 {
			t.Fatal("Meteor must hold the last skill frame without turning toward each impact")
		}
	}
}

func TestWizardMeteorPreservesMovement(t *testing.T) {
	for _, sourceID := range []uint32{2000000, 100, 300} {
		ctx := testutil.KnightEffectContext()
		actor, _, local := scene.ActorForCombatID(ctx, sourceID)
		actor.Job, actor.Dir = db.JobWizard, 3
		actor.Moving = true
		actor.FromX, actor.FromY = 10, 20
		actor.ToX, actor.ToY = 20, 20
		actor.MovePath = []worldstate.WalkStep{{X: 10, Y: 20}, {X: 20, Y: 20}}
		actor.MoveStarted = time.Now()
		actor.MoveDuration = 10 * time.Second
		if local {
			ctx.World.Player = actor
		} else {
			ctx.World.Actors[actor.ID] = actor
		}
		m := &WorldMode{}
		for _, x := range []uint16{5, 30} {
			m.applyGroundSkillNotify(ctx, network.GroundSkillNotify{SkillID: db.SkillWZMeteor, SourceID: sourceID, X: x, Y: 20})
			got := ctx.World.Player
			if !local {
				got = ctx.World.Actors[actor.ID]
			}
			if !reflect.DeepEqual(got, actor) || len(m.actorAnims) != 0 {
				t.Fatalf("meteor interrupted actor %d's movement or walking animation", sourceID)
			}
		}
		if len(m.effects().Active) != 2 {
			t.Fatal("moving caster lost the meteor effects")
		}
	}
}

func TestWizardRealAssets(t *testing.T) {
	ctx := testutil.KnightEffectContext()
	ctx.Resources = testutil.RealDataManager(t)
	start := time.Unix(100, 0)
	for _, id := range []int{effects.SightTrasher, effects.SightBlaster, effects.IceWall, effects.EarthSpike, effects.HeavenDrive, effects.StormGust, effects.LordVermilion, effects.MeteorStorm, effects.JupitelThunder, effects.JupitelHit, effects.WaterBall2, effects.Quagmire, effects.FirePillar, effects.FirePillarBomb, effects.FirePillarOn} {
		t.Run(strconv.Itoa(id), func(t *testing.T) {
			m := &WorldMode{}
			if !m.effects().AddBetween(ctx, id, 300, 2000000, start) {
				t.Fatal("missing effect")
			}
			for _, sound := range m.scheduledSounds {
				for _, path := range sound.paths {
					if _, err := ctx.Resources.ReadFile("data/wav/" + path); err != nil {
						t.Fatal(err)
					}
				}
			}
			draws := 0
			for _, tick := range []int{0, 15, 60, 150} {
				frame := render.NewFrame(800, 600)
				m.effects().Draw(frame, ctx, testutil.Projection(800, 600, 20, 20, 0), start.Add(time.Duration(tick)*db.EffectFrameDuration), false)
				v := reflect.ValueOf(frame).Elem()
				draws += v.FieldByName("worldCommands").Len() + v.FieldByName("worldBillboards").Len() + v.FieldByName("commands").Len()
			}
			if draws == 0 {
				t.Fatal("effect produced no geometry with real assets")
			}
		})
	}
}

func TestEffectBodyColorTintFollowsRobrowserFlashWindow(t *testing.T) {
	base := color.RGBA{R: 100, G: 150, B: 200, A: 255}
	starts := time.Unix(10, 0)
	mode := WorldMode{
		effectSystem: effects.System{Active: []effects.Instance{{
			EffectID: effects.BodyColor,
			ActorID:  42,
			Starts:   starts,
			Expires:  starts.Add(300 * time.Millisecond),
		}}},
	}
	if got := mode.actorBodyColorTint(42, base, starts); got != base {
		t.Fatalf("initial tint = %+v, want base %+v", got, base)
	}
	if got := mode.actorBodyColorTint(42, base, starts.Add(50*time.Millisecond)); got != (color.RGBA{R: 177, G: 75, B: 100, A: 255}) {
		t.Fatalf("half flash tint = %+v", got)
	}
	if got := mode.actorBodyColorTint(42, base, starts.Add(100*time.Millisecond)); got != (color.RGBA{R: 255, G: 0, B: 0, A: 255}) {
		t.Fatalf("full flash tint = %+v", got)
	}
	if got := mode.actorBodyColorTint(42, base, starts.Add(300*time.Millisecond)); got != (color.RGBA{R: 255, G: 0, B: 0, A: 255}) {
		t.Fatalf("final flash tint = %+v", got)
	}
	if got := mode.actorBodyColorTint(42, base, starts.Add(301*time.Millisecond)); got != base {
		t.Fatalf("expired tint = %+v, want base %+v", got, base)
	}
	if got := mode.actorBodyColorTint(99, base, starts.Add(100*time.Millisecond)); got != base {
		t.Fatalf("other actor tint = %+v, want base %+v", got, base)
	}
}

func TestMagicCrasher2BodyColorTintUsesOriginalColorWindow(t *testing.T) {
	base := color.RGBA{R: 100, G: 150, B: 200, A: 255}
	starts := time.Unix(10, 0)
	mode := WorldMode{
		effectSystem: effects.System{Active: []effects.Instance{{
			EffectID: effects.MagicCrasher2,
			ActorID:  42,
			Starts:   starts,
			Expires:  starts.Add(100 * db.EffectFrameDuration),
		}}},
	}
	first := mode.actorBodyColorTint(42, base, starts.Add(100*time.Millisecond))
	second := mode.actorBodyColorTint(42, base, starts.Add(100*time.Millisecond))
	if first != second {
		t.Fatalf("random tint should be deterministic per frame: first %+v second %+v", first, second)
	}
	if first == base || first.A != base.A {
		t.Fatalf("active random tint = %+v, want color-channel change with unchanged alpha", first)
	}
	if got := mode.actorBodyColorTint(42, base, starts.Add(61*db.EffectFrameDuration)); got != base {
		t.Fatalf("expired random tint = %+v, want base %+v", got, base)
	}
}

func TestActorBodySizeMultiplierFollowsRobrowserEswooSizes(t *testing.T) {
	starts := time.Unix(10, 0)
	for _, tc := range []struct {
		name     string
		effectID int
		duration time.Duration
		at       time.Duration
		want     float64
	}{
		{"EF_BABYBODY start", effects.BabyBody, 300 * time.Millisecond, 0, 1},
		{"EF_BABYBODY middle", effects.BabyBody, 300 * time.Millisecond, 150 * time.Millisecond, 0.75},
		{"EF_BABYBODY end", effects.BabyBody, 300 * time.Millisecond, 300 * time.Millisecond, 0.5},
		{"EF_BABYBODY2", effects.BabyBody2, 5 * time.Minute, time.Second, 0.5},
		{"EF_GIANTBODY middle", effects.GiantBody, 300 * time.Millisecond, 150 * time.Millisecond, 1.25},
		{"EF_GIANTBODY end", effects.GiantBody, 300 * time.Millisecond, 300 * time.Millisecond, 1.5},
		{"EF_GIANTBODY2", effects.GiantBody2, 5 * time.Minute, time.Second, 1.5},
	} {
		mode := WorldMode{
			effectSystem: effects.System{Active: []effects.Instance{{
				EffectID: tc.effectID,
				ActorID:  42,
				Starts:   starts,
				Expires:  starts.Add(tc.duration),
				Duration: tc.duration,
			}}},
		}
		if got := mode.actorBodySizeMultiplier(42, starts.Add(tc.at)); got != tc.want {
			t.Fatalf("%s multiplier = %f, want %f", tc.name, got, tc.want)
		}
	}
}

func TestRobrowserShockwaveSpec(t *testing.T) {
	shockwave, ok := effects.Lookup(effects.Shockwave)
	if !ok || len(shockwave.Components) != 1 {
		t.Fatalf("EF_SHOCKWAVE spec = %+v ok=%t", shockwave, ok)
	}
	if len(shockwave.Sfx) != 0 {
		t.Fatalf("EF_SHOCKWAVE sfx = %v", shockwave.Sfx)
	}
	if component := shockwave.Components[0]; component.Kind != effects.ComponentSTR || component.StrFile != "shockwave" {
		t.Fatalf("EF_SHOCKWAVE component = %+v", component)
	}
}

func TestWaterBallAndSonicBlowSpecs(t *testing.T) {
	water, ok := effects.Lookup(effects.WaterBall)
	if !ok || len(water.Components) != 1 || water.Duration != 500*time.Millisecond {
		t.Fatalf("EF_WATERBALL spec = %+v ok=%t", water, ok)
	}
	component := water.Components[0]
	if component.Kind != effects.Component3D || len(component.TextureFiles) != 3 || component.TextureFiles[0] != "effect/water_out_a.bmp" || component.TextureFiles[2] != "effect/water_out_c.bmp" {
		t.Fatalf("EF_WATERBALL texture files = %+v", component)
	}
	if component.FrameDelay != 10*time.Millisecond || component.Duration != 500*time.Millisecond || !component.FadeOut || component.PosXRand != 1.5 || component.PosZRand != 1.5 || component.PosYEnd != 3 || !component.PosYSmooth || component.SizeStart != testutil.EffectTableSize(30.5) || !component.RotateWithCamera || !component.BlendAdditive || !component.AttachedEntity {
		t.Fatalf("EF_WATERBALL component = %+v", component)
	}

	sonic, ok := effects.Lookup(effects.SonicBlow)
	if !ok || len(sonic.Components) != 1 || sonic.Duration != 40*db.EffectFrameDuration {
		t.Fatalf("EF_SONICBLOW spec = %+v ok=%t", sonic, ok)
	}
	ring := sonic.Components[0]
	if ring.Kind != effects.ComponentFUNC || ring.FuncAdapter != effects.EffectFuncSonicBlow || ring.TextureFile != "effect/alpha_center.tga" || !ring.AttachedEntity {
		t.Fatalf("EF_SONICBLOW ring = %+v", ring)
	}
	spin, ok := effects.Lookup(effects.SonicBlowHit)
	if !ok || len(spin.Components) != 1 || spin.Components[0].Kind != effects.ComponentFUNC || spin.Components[0].FuncAdapter != effects.EffectFuncHitRing || !spin.Components[0].AttachedEntity {
		t.Fatalf("EF_SONICBLOWHIT spec = %+v ok=%t", spin, ok)
	}
}

func TestCrashEarthSpec(t *testing.T) {
	crash, ok := effects.Lookup(effects.CrashEarth)
	if !ok || len(crash.Components) != 1 {
		t.Fatalf("EF_CRASHEARTH spec = %+v ok=%t", crash, ok)
	}
	if crash.CameraShakeDelay != 35*db.EffectFrameDuration || crash.CameraShake != 650*time.Millisecond {
		t.Fatalf("EF_CRASHEARTH camera shake = delay %s duration %s", crash.CameraShakeDelay, crash.CameraShake)
	}
	if component := crash.Components[0]; component.Kind != effects.ComponentSTR || component.StrFile != "crashearth" || component.AttachedEntity {
		t.Fatalf("EF_CRASHEARTH component = %+v", component)
	}

}

func TestRobrowserQuadHornRuntimeDefaults(t *testing.T) {
	if got := effects.QuadHornDefaultOffset(0); got != 0.5 {
		t.Fatalf("quadHornDefaultOffset(0) = %v, want robr default 0.5", got)
	}
	if got := effects.QuadHornDefaultOffset(-0.2); got != -0.2 {
		t.Fatalf("quadHornDefaultOffset(-0.2) = %v, want -0.2", got)
	}
	effect := effects.Instance{EffectID: effects.EarthSpike, ActorID: 1, Starts: time.Unix(10, 20)}
	if got := effects.QuadHornRange(effect, 1, 0, -0.2); got != -0.2 {
		t.Fatalf("quadHornRange max<min = %v, want -0.2", got)
	}
}

func TestQuadHornUVsMatchSourceFaceStrips(t *testing.T) {
	if len(effects.QuadHornUVs) != 12 {
		t.Fatalf("quadHornUVs len = %d, want 12", len(effects.QuadHornUVs))
	}
	for face := 0; face < 4; face++ {
		u0 := float32(face) * 0.2
		uvs := effects.QuadHornUVs[face*3 : face*3+3]
		want := []scene.UV{
			{U: u0, V: 0},
			{U: u0, V: 1},
			{U: u0 + 0.2, V: 1},
		}
		if !reflect.DeepEqual(uvs, want) {
			t.Fatalf("quad horn face %d uvs = %+v, want %+v", face, uvs, want)
		}
	}
}

func TestQuadHornDefaultRotationKeepsApexAboveBase(t *testing.T) {
	effect := effects.Instance{EffectID: effects.IceWall, ActorID: 1, Starts: time.Unix(10, 20)}
	component := effects.Component{}
	salt := 0
	rotateX := effects.QuadHornRange(effect, salt+5, component.QuadHornRotateXMin, component.QuadHornRotateXMax)
	rotateY := effects.QuadHornRange(effect, salt+6, component.QuadHornRotateYMin, component.QuadHornRotateYMax)
	rotateZ := effects.QuadHornRange(effect, salt+7, component.QuadHornRotateZMin, component.QuadHornRotateZMax)
	height := 3.0
	bottomSize := 0.4
	origin := scene.Point3{Y: height * 0.9}
	apex := scene.Add3(origin, effects.RotateEffectCylinderVector(scene.Point3{Y: height}, rotateX, rotateY, rotateZ))
	base := scene.Add3(origin, effects.RotateEffectCylinderVector(scene.Point3{X: -bottomSize, Y: -height, Z: bottomSize}, rotateX, rotateY, rotateZ))

	if apex.Y <= base.Y {
		t.Fatalf("quad horn apex y = %.3f, base y = %.3f, want apex above base", apex.Y, base.Y)
	}
}

func TestRobrowserOldPortalEffectsZeroToFifty(t *testing.T) {
	entry, ok := effects.Lookup(effects.Entry)
	if !ok {
		t.Fatal("EF_ENTRY spec missing")
	}
	if entry.Duration != 500*time.Millisecond || len(entry.Sfx) != 1 || entry.Sfx[0] != "effect\\ef_portal.wav" {
		t.Fatalf("EF_ENTRY timing/sfx = duration %s sfx %#v", entry.Duration, entry.Sfx)
	}
	if len(entry.Components) != 3 {
		t.Fatalf("EF_ENTRY components = %d, want 3", len(entry.Components))
	}
	for i, component := range entry.Components {
		if component.Kind != effects.ComponentCylinder || component.TextureName != "ring_blue" || component.Duration != 500*time.Millisecond || component.Animation != 1 || !component.Fade || !component.Rotate {
			t.Fatalf("EF_ENTRY component %d = %+v", i, component)
		}
	}
	if entry.Components[0].AttachedEntity || !entry.Components[1].AttachedEntity || !entry.Components[2].AttachedEntity {
		t.Fatalf("EF_ENTRY attachment flags = %t %t %t", entry.Components[0].AttachedEntity, entry.Components[1].AttachedEntity, entry.Components[2].AttachedEntity)
	}
	if entry.Components[0].Height != 7.5 || entry.Components[1].Height != 8 || entry.Components[2].TopSize != 1.5 {
		t.Fatalf("EF_ENTRY dimensions = %+v", entry.Components)
	}

	warp, ok := effects.Lookup(effects.Warp)
	if !ok || len(warp.Components) != 1 {
		t.Fatalf("EF_WARP spec = %+v ok=%t", warp, ok)
	}
	wave := warp.Components[0]
	if wave.Kind != effects.ComponentCylinder || wave.TextureName != "ring_yellow" || wave.Animation != 4 || wave.Duplicate != 4 || wave.DuplicateDelay != 300*time.Millisecond {
		t.Fatalf("EF_WARP wave = %+v", wave)
	}
	if wave.BottomSize != 10 || wave.TopSize != 13 || wave.PosZ != 0.1 || !wave.AttachedEntity {
		t.Fatalf("EF_WARP dimensions = %+v", wave)
	}
	if got := effects.WorldEffectComponentDuration(warp, wave); got != 1900*time.Millisecond {
		t.Fatalf("EF_WARP resolved duration = %s, want 1900ms", got)
	}

	teleport, ok := effects.Lookup(effects.TeleportOld)
	if !ok || len(teleport.Components) != 1 {
		t.Fatalf("EF_TELEPORTATION spec = %+v ok=%t", teleport, ok)
	}
	beam := teleport.Components[0]
	if teleport.Duration != time.Second || len(teleport.Sfx) != 1 || teleport.Sfx[0] != "effect\\ef_teleportation.wav" {
		t.Fatalf("EF_TELEPORTATION timing/sfx = %s %#v", teleport.Duration, teleport.Sfx)
	}
	if beam.Kind != effects.ComponentCylinder || beam.TextureName != "ring_blue" || beam.Animation != 5 || beam.Height != 35 || beam.BottomSize != 0.8 || beam.TopSize != 0.7 || !beam.Rotate {
		t.Fatalf("EF_TELEPORTATION beam = %+v", beam)
	}

	ready, ok := effects.Lookup(effects.ReadyPortalOld)
	if !ok || len(ready.Components) != 1 {
		t.Fatalf("EF_READYPORTAL spec = %+v ok=%t", ready, ok)
	}
	portal := ready.Components[0]
	if ready.Duration != 25*time.Second || len(ready.Sfx) != 1 || ready.Sfx[0] != "effect\\ef_readyportal.wav" {
		t.Fatalf("EF_READYPORTAL timing/sfx = %s %#v", ready.Duration, ready.Sfx)
	}
	if portal.Kind != effects.ComponentCylinder || portal.TextureName != "alpha_down" || portal.Color != (color.RGBA{R: 178, G: 178, B: 255, A: 255}) || portal.Height != 15 || portal.AlphaMax != 0.6 {
		t.Fatalf("EF_READYPORTAL cylinder = %+v", portal)
	}
}

func TestRobrowserOldRestoreEffectsZeroToFifty(t *testing.T) {
	exit, ok := effects.Lookup(effects.Exit)
	if !ok {
		t.Fatal("EF_EXIT spec missing")
	}
	if exit.Duration != 2*time.Second || len(exit.Sfx) != 1 || exit.Sfx[0] != "_heal_effect.wav" || len(exit.Components) != 3 {
		t.Fatalf("EF_EXIT spec = %+v", exit)
	}
	if cylinder := exit.Components[0]; cylinder.Kind != effects.ComponentCylinder || cylinder.TextureName != "alpha_down" || cylinder.Duration != 2*time.Second || cylinder.Animation != 1 || cylinder.AlphaMax != 0.2 || !cylinder.BlendAdditive {
		t.Fatalf("EF_EXIT cylinder = %+v", cylinder)
	}
	if particle := exit.Components[1]; particle.Kind != effects.Component3D || particle.TextureFile != "effect/pok3.tga" || particle.Delay != 400*time.Millisecond || particle.Duplicate != 6 || particle.DuplicateDelay != 80*time.Millisecond || !particle.Sparkling {
		t.Fatalf("EF_EXIT first particle = %+v", particle)
	}
	if particle := exit.Components[2]; particle.Duration != 900*time.Millisecond || particle.Delay != 200*time.Millisecond || particle.Duplicate != 3 || particle.DuplicateDelay != 200*time.Millisecond || particle.PosZEnd != 6 {
		t.Fatalf("EF_EXIT second particle = %+v", particle)
	}

	enhance, ok := effects.Lookup(effects.Enhance)
	if !ok || len(enhance.Components) != 3 {
		t.Fatalf("EF_ENHANCE spec = %+v ok=%t", enhance, ok)
	}
	if enhance.Components[0].TextureName != "alpha_down" || enhance.Components[0].BlendAdditive != true || enhance.Components[0].Duration != 2*time.Second {
		t.Fatalf("EF_ENHANCE cylinder = %+v", enhance.Components[0])
	}
	for _, tc := range []struct {
		index     int
		delay     time.Duration
		duplicate int
	}{
		{index: 1, delay: 500 * time.Millisecond, duplicate: 7},
		{index: 2, delay: 400 * time.Millisecond, duplicate: 3},
	} {
		component := enhance.Components[tc.index]
		if component.Kind != effects.Component3D || component.TextureFile != "effect/ac_center2.tga" || component.Delay != tc.delay || component.Duplicate != tc.duplicate || component.DuplicateDelay != 200*time.Millisecond {
			t.Fatalf("EF_ENHANCE particle %d = %+v", tc.index, component)
		}
		if component.SizeStartX != 2.5*scene.PixelRatio || component.SizeRandY != 15*scene.PixelRatio || component.SizeRandYMiddle != 45*scene.PixelRatio {
			t.Fatalf("EF_ENHANCE particle %d size = %+v", tc.index, component)
		}
	}

	healSP, ok := effects.Lookup(effects.HealSP)
	if !ok || len(healSP.Components) != 3 {
		t.Fatalf("EF_HEALSP spec = %+v ok=%t", healSP, ok)
	}
	if len(healSP.Sfx) != 1 || healSP.Sfx[0] != "_heal_effect.wav" {
		t.Fatalf("EF_HEALSP sfx = %#v", healSP.Sfx)
	}
	blue := color.RGBA{R: 25, G: 128, B: 255, A: 255}
	if cylinder := healSP.Components[0]; cylinder.TextureName != "ring_blue" || cylinder.Color != blue || !cylinder.Rotate || !cylinder.BlendAdditive {
		t.Fatalf("EF_HEALSP cylinder = %+v", cylinder)
	}
	if healSP.Components[1].Color != blue || healSP.Components[2].Color != blue {
		t.Fatalf("EF_HEALSP particle tints = %+v %+v", healSP.Components[1].Color, healSP.Components[2].Color)
	}
}

func TestRobrowserOldBoltSoundAndStatusEffectsZeroToFifty(t *testing.T) {
	glass, ok := effects.Lookup(effects.GlassWall)
	if !ok || len(glass.Components) != 1 {
		t.Fatalf("EF_GLASSWALL spec = %+v ok=%t", glass, ok)
	}
	if component := glass.Components[0]; component.Kind != effects.ComponentSTR || component.StrFile != "effect/safetywall" || component.AttachedEntity {
		t.Fatalf("EF_GLASSWALL component = %+v", component)
	}
	if len(glass.Sfx) != 1 || glass.Sfx[0] != "effect\\ef_glasswall.wav" {
		t.Fatalf("EF_GLASSWALL sfx = %#v", glass.Sfx)
	}

	ice, ok := effects.Lookup(effects.IceArrow)
	if !ok {
		t.Fatal("EF_ICEARROW spec missing")
	}
	if len(ice.Components) != 1 || ice.Components[0].FuncAdapter != effects.EffectFuncMageBolt || len(ice.Sfx) != 1 || ice.Sfx[0] != "effect\\ef_icearrow%d.wav" || ice.SfxRandMin != 1 || ice.SfxRandMax != 3 {
		t.Fatalf("EF_ICEARROW spec = %+v", ice)
	}

	fire, ok := effects.Lookup(effects.FireArrow)
	if !ok {
		t.Fatal("EF_FIREARROW spec missing")
	}
	if len(fire.Components) != 1 || fire.Components[0].FuncAdapter != effects.EffectFuncMageBolt || len(fire.Sfx) != 1 || fire.Sfx[0] != "effect\\ef_firearrow%d.wav" {
		t.Fatalf("EF_FIREARROW spec = %+v", fire)
	}

	incAgiDex, ok := effects.Lookup(effects.IncAgiDex)
	if !ok || len(incAgiDex.Components) != 1 {
		t.Fatalf("EF_INCAGIDEX spec = %+v ok=%t", incAgiDex, ok)
	}
	if len(incAgiDex.Sfx) != 1 || incAgiDex.Sfx[0] != "effect\\ef_incagidex.wav" {
		t.Fatalf("EF_INCAGIDEX sfx = %#v", incAgiDex.Sfx)
	}
	component := incAgiDex.Components[0]
	if component.FuncAdapter != effects.EffectFuncAgility || component.TextureFile != "effect/dex_agi_up.bmp" || incAgiDex.Duration != 100*db.EffectFrameDuration {
		t.Fatalf("EF_INCAGIDEX must reuse the native agility particles: %+v", component)
	}
}

func TestPneumaEffectSpecMatchesRoBrowserSTR(t *testing.T) {
	spec, ok := effects.Lookup(effects.Pneuma)
	if !ok {
		t.Fatal("Pneuma effect spec missing")
	}
	if len(spec.Components) != 1 {
		t.Fatalf("Pneuma components = %d, want 1", len(spec.Components))
	}
	component := spec.Components[0]
	if component.Kind != effects.ComponentSTR || component.StrFile != "pneuma%d" || component.StrRandMin != 1 || component.StrRandMax != 3 || component.AttachedEntity {
		t.Fatalf("Pneuma component = %+v", component)
	}
}

func TestTorchEffectSpecMatchesRoBrowserShape(t *testing.T) {
	spec, ok := effects.Lookup(effects.Torch)
	if !ok {
		t.Fatal("torch effect spec missing")
	}
	if spec.Duration != 24*time.Hour {
		t.Fatalf("torch duration = %s", spec.Duration)
	}
	if len(spec.Components) != 1 {
		t.Fatalf("torch components = %d, want 1", len(spec.Components))
	}
	component := spec.Components[0]
	if component.Kind != effects.Component3D || component.SpriteFile != "torch_01" || !component.SpriteRepeat {
		t.Fatalf("torch component = %+v", component)
	}
	if component.Duration != 600*time.Millisecond || component.SpriteDelay != 100*time.Millisecond {
		t.Fatalf("torch timing = duration %s delay %s", component.Duration, component.SpriteDelay)
	}
	if component.PosX != 0.1 || component.PosZ != 0.8 || component.SizeStart != testutil.EffectTableSize(100) || component.AngleStart != 270 || !component.RotateToTarget {
		t.Fatalf("torch placement = %+v", component)
	}
	ctx := client.Context{World: worldstate.New()}
	angle, ok := effects.EffectSpriteRobrowserRotation(ctx, scene.Projection{}, component, effects.Instance{EffectID: effects.Torch}, 0)
	if !ok || math.Abs(angle+2*math.Pi) > 0.001 {
		t.Fatalf("torch effective angle = %.3f, %v, want -2π radians, true", angle, ok)
	}
}

func TestFireflyEffectSpecUsesFaintSpriteParticles(t *testing.T) {
	spec, ok := effects.Lookup(effects.Firefly)
	if !ok {
		t.Fatal("firefly effect spec missing")
	}
	if len(spec.Components) != 1 {
		t.Fatalf("firefly components = %d, want 1", len(spec.Components))
	}
	component := spec.Components[0]
	if component.Kind != effects.Component3D || component.TextureFile != "" || component.SpriteFile == "" || !component.SpriteRepeat {
		t.Fatalf("firefly component = %+v", component)
	}
	if component.AlphaMax > 0.25 || component.SizeEnd > testutil.EffectTableSize(120) {
		t.Fatalf("firefly should stay faint and moderately sized: %+v", component)
	}
}

func TestBubbleEffectSpecMatchesRoBrowserShape(t *testing.T) {
	spec, ok := effects.Lookup(effects.Bubble)
	if !ok {
		t.Fatal("bubble effect spec missing")
	}
	if len(spec.Components) != 1 {
		t.Fatalf("bubble components = %d, want 1", len(spec.Components))
	}
	component := spec.Components[0]
	if component.Kind != effects.ComponentSTR || component.StrFile != "bubble%d" || component.StrRandMin != 1 || component.StrRandMax != 4 {
		t.Fatalf("bubble component = %+v", component)
	}
}

func TestWorldEffectSpecLookupReturnsCopy(t *testing.T) {
	spec, ok := effects.Lookup(effects.BashBegin)
	if !ok {
		t.Fatal("bash begin effect spec missing")
	}
	spec.Sfx[0] = "mutated.wav"
	spec.Components[0].TextureFiles[0] = "mutated"

	again, ok := effects.Lookup(effects.BashBegin)
	if !ok {
		t.Fatal("bash begin effect spec missing after mutation")
	}
	if again.Sfx[0] != "effect\\ef_bash.wav" || again.Components[0].TextureFiles[0] != "effect/alpha_center.tga" {
		t.Fatalf("catalog mutated: %+v", again)
	}
}

func TestResolveEffectSTRFileUsesDeterministicRandRange(t *testing.T) {
	component := effects.Component{
		StrFile:    "firehit%d",
		StrRandMin: 1,
		StrRandMax: 3,
	}
	effect := effects.Instance{EffectID: effects.FireHit, ActorID: 100, Starts: time.Unix(10, 20)}
	got := effects.ResolveEffectSTRFile(component, effect, false)
	if got != "firehit1" && got != "firehit2" && got != "firehit3" {
		t.Fatalf("resolved STR file = %q, want firehit1..3", got)
	}
	if again := effects.ResolveEffectSTRFile(component, effect, false); again != got {
		t.Fatalf("resolved STR file changed from %q to %q", got, again)
	}
}

func TestResolveEffectSTRFileUsesMinFileForLessEffects(t *testing.T) {
	component := effects.Component{
		StrFile:    "angelus",
		StrMinFile: "jong_mini",
	}
	if got := effects.ResolveEffectSTRFile(component, effects.Instance{}, true); got != "jong_mini" {
		t.Fatalf("resolved STR file = %q, want jong_mini", got)
	}
}

func expectEffectIDs(t *testing.T, label string, got []int, want ...int) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("%s effects = %v, want %v", label, got, want)
	}
}

func TestSwordmanSkillEffectMappings(t *testing.T) {
	expectEffectIDs(t, "SM_BASH begin", skillBeginEffectIDs(5), effects.BashBegin)
	expectEffectIDs(t, "SM_BASH hit", skillHitEffectIDs(5), effects.BashHit)
	expectEffectIDs(t, "SM_PROVOKE success", skillSuccessEffectIDs(6), effects.Provoke)
	expectEffectIDs(t, "SM_MAGNUM damage", skillEffectIDs(7))
	expectEffectIDs(t, "SM_MAGNUM caster damage", skillEffectOnCasterIDs(7))
	expectEffectIDs(t, "SM_MAGNUM success", skillSuccessEffectSelfIDs(7), effects.MagnumBreak, effects.QuakeMagnum)
	expectEffectIDs(t, "SM_ENDURE", skillSuccessEffectSelfIDs(8), effects.Endure)
}

func TestNoviceSkillEffectMappings(t *testing.T) {
	if action := skillAction(db.SkillNVBasic); !action.defined || action.action != skillActorActionNone {
		t.Fatalf("basic skill action = %+v, want no source action", action)
	}
	expectEffectIDs(t, "NV_FIRSTAID", skillEffectIDs(142), effects.FirstAid)
	expectEffectIDs(t, "NV_TRICKDEAD", skillEffectIDs(143))
}

func TestMageSkillEffectMappings(t *testing.T) {
	expectEffectIDs(t, "MG_SIGHT success", skillSuccessEffectIDs(10))
	expectEffectIDs(t, "MG_SIGHT immediate", skillEffectIDs(10))
	expectEffectIDs(t, "MG_NAPALMBEAT hit", skillHitEffectIDs(11), effects.Hit1, effects.BashHit)
	expectEffectIDs(t, "MG_SAFETYWALL ground", skillGroundEffectIDs(12))
	expectEffectIDs(t, "MG_SOULSTRIKE before-hit", skillBeforeHitEffectIDs(13), effects.SoulStrike)
	expectEffectIDs(t, "MG_SOULSTRIKE hit", skillHitEffectIDs(13), effects.Hit1, effects.BashHit)
	expectEffectIDs(t, "MG_COLDBOLT before-hit", skillBeforeHitEffectIDs(14), effects.ColdBolt)
	expectEffectIDs(t, "MG_COLDBOLT hit", skillHitEffectIDs(14), effects.ColdHit)
	expectEffectIDs(t, "MG_FROSTDIVER", skillEffectIDs(15))
	expectEffectIDs(t, "MG_FROSTDIVER before-hit", skillBeforeHitEffectIDs(15), effects.FrostDiver)
	expectEffectIDs(t, "MG_FROSTDIVER hit", skillHitEffectIDs(15), effects.ColdHit, effects.FrostDiverHit)
	expectEffectIDs(t, "MG_STONECURSE", skillSuccessEffectIDs(16), effects.StoneCurse)
	expectEffectIDs(t, "MG_FIREBOLT before-hit", skillBeforeHitEffectIDs(19), effects.FireBolt)
	expectEffectIDs(t, "MG_FIREBALL before-hit", skillBeforeHitEffectIDs(17), effects.FireBall)
	expectEffectIDs(t, "MG_FIREWALL ground", skillGroundEffectIDs(18))
	expectEffectIDs(t, "MG_FIREBALL hit", skillHitEffectIDs(17), effects.FireHit, effects.BashHit)
	expectEffectIDs(t, "MG_FIREWALL hit", skillHitEffectIDs(18), effects.Hit1)
	expectEffectIDs(t, "MG_FIREBOLT hit", skillHitEffectIDs(19), effects.FireHit)
	expectEffectIDs(t, "MG_LIGHTNINGBOLT hit", skillHitEffectIDs(20), effects.WindHit)
	expectEffectIDs(t, "MG_THUNDERSTORM hit", skillHitEffectIDs(21), effects.Hit1)
	expectEffectIDs(t, "MG_LIGHTNINGBOLT before-hit", skillBeforeHitEffectIDs(20))
	expectEffectIDs(t, "MG_LIGHTNINGBOLT", skillEffectIDs(20), effects.LightningBolt)
	expectEffectIDs(t, "MG_THUNDERSTORM", skillEffectIDs(21), effects.ThunderStorm)
	expectEffectIDs(t, "MG_THUNDERSTORM ground", skillGroundEffectIDs(21))
	expectEffectIDs(t, "MG_ENERGYCOAT", skillSuccessEffectSelfIDs(157), effects.EnergyCoat)
	expectEffectIDs(t, "MG_THUNDERSTORM before-hit", skillBeforeHitEffectIDs(21))
	for _, skillID := range []uint16{20, 21} {
		expectEffectIDs(t, "wind skill begin", skillBeginEffectIDs(skillID))
	}
}

func TestFireHitEffectSpec(t *testing.T) {
	hitSpec, ok := effects.Lookup(effects.FireHit)
	if !ok || len(hitSpec.Components) != 1 {
		t.Fatalf("fire hit effect missing or wrong component count: ok=%t components=%d", ok, len(hitSpec.Components))
	}
	hit := hitSpec.Components[0]
	if hit.Kind != effects.ComponentSTR || hit.StrFile != "firehit%d" || hit.StrRandMin != 1 || hit.StrRandMax != 3 || !hit.AttachedEntity {
		t.Fatalf("fire hit STR = %+v", hit)
	}
	if len(hitSpec.Sfx) != 1 || hitSpec.Sfx[0] != "effect\\ef_firehit.wav" {
		t.Fatalf("fire hit sfx = %#v", hitSpec.Sfx)
	}
}

func TestRepeatedWorldEffectDuplicateProgressUsesDuplicateDelay(t *testing.T) {
	component := effects.Component{
		Duration:       4 * time.Second,
		Delay:          200 * time.Millisecond,
		Repeat:         true,
		Duplicate:      4,
		DuplicateDelay: time.Second,
	}
	starts := time.Unix(1000, 0)
	now := starts.Add(2700 * time.Millisecond)
	for _, tc := range []struct {
		duplicate int
		want      float64
	}{
		{0, 0.625},
		{1, 0.375},
		{2, 0.125},
	} {
		got, active := effects.WorldEffectComponentDuplicateProgressForDraw(starts, component, tc.duplicate, component.Duration, now)
		if !active {
			t.Fatalf("duplicate %d inactive, want progress %.3f", tc.duplicate, tc.want)
		}
		if math.Abs(got-tc.want) > 0.000001 {
			t.Fatalf("duplicate %d progress = %.6f, want %.6f", tc.duplicate, got, tc.want)
		}
	}
	if got, active := effects.WorldEffectComponentDuplicateProgressForDraw(starts, component, 3, component.Duration, now); active {
		t.Fatalf("duplicate 3 active with progress %.6f, want delayed until its own start", got)
	}
}

func TestCylinderAnimationThreeScalesRadiusAndPulsesHeightLikeRobrowser(t *testing.T) {
	component := effects.Component{
		Animation:  3,
		BottomSize: 2,
		TopSize:    4,
		Height:     6,
	}
	for _, tc := range []struct {
		progress   float64
		wantBottom float64
		wantTop    float64
		wantHeight float64
	}{
		{0, 2, 4, 0},
		{0.25, 1.5, 3, 3},
		{0.5, 1, 2, 6},
		{0.75, 0.5, 1, 3},
	} {
		gotBottom, gotTop, gotHeight := effects.EffectCylinderAnimatedDimensions(component, tc.progress)
		if math.Abs(gotBottom-tc.wantBottom) > 0.000001 || math.Abs(gotTop-tc.wantTop) > 0.000001 || math.Abs(gotHeight-tc.wantHeight) > 0.000001 {
			t.Fatalf("progress %.2f dimensions = %.6f %.6f %.6f, want %.6f %.6f %.6f", tc.progress, gotBottom, gotTop, gotHeight, tc.wantBottom, tc.wantTop, tc.wantHeight)
		}
	}
}

func TestRegularHitEffectKeepsRobrowserParticleBurst(t *testing.T) {
	spec, ok := effects.Lookup(effects.Hit1)
	if !ok || len(spec.Components) != 2 {
		t.Fatalf("regular hit spec = %+v ok=%t, want sparks and a ring", spec, ok)
	}
	if spec.Duration != 300*time.Millisecond {
		t.Fatalf("duration = %s, want 300ms", spec.Duration)
	}
	component := spec.Components[0]
	if component.Kind != effects.Component3D || component.TextureFile != "effect/pok3.tga" || component.Duration != 300*time.Millisecond {
		t.Fatalf("component resource/timing = %+v", component)
	}
	if component.Duplicate != 4 || component.AlphaMax != 0.8 || !component.FadeIn || !component.FadeOut || !component.Sparkling {
		t.Fatalf("component duplicate/fade = %+v", component)
	}
	if component.PosZ != 1 || component.PosXEndRand != 2 || component.PosYEndRand != 2 || component.PosZEndRand != 2 {
		t.Fatalf("component position = %+v", component)
	}
	if component.SizeStart != testutil.EffectTableSize(10) || component.SizeEnd != testutil.EffectTableSize(10) || component.SizeRand != testutil.EffectTableSize(20) || !component.SizeSmooth {
		t.Fatalf("component size = %+v", component)
	}
}

func TestSkillHitSlashSpecsMatchRobrowser(t *testing.T) {
	for _, tc := range []struct {
		name     string
		effectID int
		kind     effects.ComponentKind
		width    float64
		height   float64
		sfx      string
		overlay  bool
	}{
		{"hit5", effects.Hit5, effects.Component3D, testutil.EffectTableSize(15), testutil.EffectTableSize(200), "effect\\ef_hit5.wav", false},
		{"hit6", effects.Hit6, effects.Component2D, testutil.EffectTableSize(10), testutil.EffectTableSize(150), "effect\\ef_hit6.wav", true},
	} {
		spec, ok := effects.Lookup(tc.effectID)
		if !ok || len(spec.Components) != 2 {
			t.Fatalf("%s spec = %+v ok=%t, want two slash components", tc.name, spec, ok)
		}
		if len(spec.Sfx) != 1 || spec.Sfx[0] != tc.sfx {
			t.Fatalf("%s sfx = %v", tc.name, spec.Sfx)
		}
		for i, component := range spec.Components {
			if component.Kind != tc.kind || component.TextureFile != "effect/lens2.tga" || component.Duration != 400*time.Millisecond || component.AlphaMax != 1 || !component.FadeOut || !component.Rotate || component.Overlay != tc.overlay {
				t.Fatalf("%s component %d = %+v", tc.name, i, component)
			}
			if component.PosZ != 1 || component.SizeStartX != tc.width || component.SizeEndX != tc.width || component.SizeStartY != testutil.EffectTableSize(10) || component.SizeEndY != tc.height {
				t.Fatalf("%s component %d size/position = %+v", tc.name, i, component)
			}
		}
		if spec.Components[0].AngleStart != 90 || spec.Components[0].AngleEnd != 0 || spec.Components[1].AngleStart != 180 || spec.Components[1].AngleEnd != 90 {
			t.Fatalf("%s slash angles = %+v %+v", tc.name, spec.Components[0], spec.Components[1])
		}
	}
}

func TestSpriteRotationUsesRobrowserWorldTrajectory(t *testing.T) {
	component := effects.Component{ToSrc: true, RotateToTarget: true, RotateWithCamera: true}
	tests := []struct {
		name      string
		sourceX   int
		sourceY   int
		targetX   int
		targetY   int
		cameraYaw float64
		want      float64
	}{
		{name: "same row", sourceX: 10, sourceY: 20, targetX: 12, targetY: 20, want: -math.Pi / 2},
		{name: "same column", sourceX: 12, sourceY: 18, targetX: 12, targetY: 20, want: 0},
		{name: "diagonal", sourceX: 10, sourceY: 18, targetX: 12, targetY: 20, want: -math.Pi / 4},
		{name: "camera yaw", sourceX: 10, sourceY: 20, targetX: 12, targetY: 20, cameraYaw: 45, want: -3 * math.Pi / 4},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			world := worldstate.New()
			world.Player = worldstate.Actor{ID: 2000000, X: tt.sourceX, Y: tt.sourceY}
			world.UpsertActor(worldstate.Actor{ID: 300, X: tt.targetX, Y: tt.targetY})
			ctx := client.Context{
				Session: &session.Session{AccountID: 2000000, CharID: 150000},
				World:   world,
			}
			effect := effects.Instance{EffectID: effects.FireBall, ActorID: 300, TargetID: 2000000}
			startX, startY, _, endX, endY, _, ok := effects.EffectTrajectoryEndpoints(ctx, component, effect)
			if !ok {
				t.Fatal("trajectory endpoints missing")
			}
			if math.Hypot(endX-startX, endY-startY) <= 0.001 {
				t.Fatalf("trajectory did not span caster and target: %.2f,%.2f -> %.2f,%.2f", startX, startY, endX, endY)
			}
			projection := testutil.ProjectionYaw(800, 600, float64(tt.targetX), float64(tt.targetY), 0, tt.cameraYaw)
			angle, ok := effects.EffectSpriteRobrowserRotation(ctx, projection, component, effect, 0)
			if !ok {
				t.Fatal("rotation missing")
			}
			wantFromFormula := -(90 - math.Atan2(endY-startY, endX-startX)*180/math.Pi + tt.cameraYaw) * math.Pi / 180
			if math.Abs(angle-wantFromFormula) > 0.001 {
				t.Fatalf("angle = %.3f, want robr formula %.3f", angle, wantFromFormula)
			}
			if math.Abs(angle-tt.want) > 0.001 {
				t.Fatalf("angle = %.3f, want %.3f", angle, tt.want)
			}
		})
	}
}

func TestGroundSampleEffectSpecUsesMagicTargetPlane(t *testing.T) {
	spec, ok := effects.Lookup(effects.GroundSample)
	if !ok {
		t.Fatal("ground sample effect missing")
	}
	if len(spec.Components) != 1 {
		t.Fatalf("components = %d, want 1", len(spec.Components))
	}
	component := spec.Components[0]
	if component.Kind != effects.ComponentFUNC || component.FuncAdapter != effects.EffectFuncGroundSample || component.FuncName != "MagicTarget" || component.TextureFile != "effect/magic_target.tga" || component.SizeStart != 1 {
		t.Fatalf("component = %+v", component)
	}
	if component.Duration != 0 {
		t.Fatalf("ground sample component duration = %s, want inherited cast duration", component.Duration)
	}
}

func TestGroundSampleRotationAngleUsesClientSpeedOverride(t *testing.T) {
	start := time.Unix(10, 0)
	now := start.Add(time.Second)
	speed := 0.5 / 7 * 60
	effect := effects.Instance{Starts: start, GroundSampleRotationRadiansPerSecond: speed}
	if got := effects.GroundSampleRotationAngle(effect, now); math.Abs(got-speed) > 0.001 {
		t.Fatalf("override angle = %.4f, want %.4f", got, speed)
	}
	fallback := effects.Instance{Starts: start}
	wantFallback := 40 * math.Pi / 180
	if got := effects.GroundSampleRotationAngle(fallback, now); math.Abs(got-wantFallback) > 0.001 {
		t.Fatalf("fallback angle = %.4f, want %.4f", got, wantFallback)
	}
}

func TestGroundSampleDrawOptionsClampRotatedMagicTargetTexture(t *testing.T) {
	options := effects.GroundSampleDrawOptions()
	if options.Filter != render.FilterLinear {
		t.Fatalf("filter = %v, want linear", options.Filter)
	}
	if options.Address != render.AddressClampToZero {
		t.Fatalf("address = %v, want clamp-to-zero to avoid repeated rotated MagicTarget corners", options.Address)
	}
	if !options.DepthTest {
		t.Fatal("ground sample depth test is disabled")
	}
}

func TestLockOnTargetEffectSpecMatchesRobrowserCastTargetCircle(t *testing.T) {
	spec, ok := effects.Lookup(effects.LockOnTarget)
	if !ok {
		t.Fatal("lock-on target effect missing")
	}
	if len(spec.Components) != 1 {
		t.Fatalf("components = %d, want 1", len(spec.Components))
	}
	component := spec.Components[0]
	if component.Kind != effects.ComponentFUNC || component.FuncAdapter != effects.EffectFuncLockOnTarget || component.FuncName != "LockOnTarget" || component.TextureFile != "effect/lockon128.tga" || !component.AttachedEntity {
		t.Fatalf("component = %+v", component)
	}
	start := time.Unix(10, 0)
	if got := effects.LockOnTargetSize(start, start); got != 15 {
		t.Fatalf("initial lock-on size = %.1f, want 15", got)
	}
	if got := effects.LockOnTargetSize(start, start.Add(250*time.Millisecond)); got != 3 {
		t.Fatalf("settled lock-on size = %.1f, want 3", got)
	}
	if got := effects.LockOnTargetTint(start, start); got != (color.RGBA{R: 255, G: 255, B: 255, A: 255}) {
		t.Fatalf("initial lock-on tint = %+v", got)
	}
	if got := effects.LockOnTargetTint(start, start.Add(380*time.Millisecond)); got != (color.RGBA{R: 255, G: 12, B: 12, A: 255}) {
		t.Fatalf("low lock-on tint = %+v", got)
	}
}

func TestApplyActorActionNotifyRepeatsFireBoltHits(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20, Dir: 4}
	world.UpsertActor(worldstate.Actor{
		ID:            300,
		X:             11,
		Y:             20,
		Job:           1002,
		ObjectType:    scene.ActorObjectTypeMob,
		HasObjectType: true,
	})
	mode := &WorldMode{}
	ctx := client.Context{
		Session: &session.Session{AccountID: 2000000, CharID: 150000},
		World:   world,
	}

	mode.applyActorActionNotify(ctx, network.ActorActionNotify{
		SkillID:     19,
		SkillLevel:  4,
		SourceID:    2000000,
		TargetID:    300,
		SourceSpeed: 580,
		TargetSpeed: 480,
		Damage:      1008,
		HitCount:    4,
		Action:      network.ActorActionSkill,
	})

	if len(mode.effects().Active) != 5 {
		t.Fatalf("world effects = %d, want one bolt sequence and four impacts", len(mode.effects().Active))
	}
	if effect := mode.effects().Active[0]; effect.EffectID != effects.FireBolt || effect.ActorID != 300 || effect.HitCount != 4 {
		t.Fatalf("wrong bolt sequence: %+v", effect)
	}
	for i := 1; i < 5; i++ {
		effect := mode.effects().Active[i]
		if effect.EffectID != effects.FireHit || effect.ActorID != 300 {
			t.Fatalf("hit effect %d = %+v", i, effect)
		}
		if i > 1 {
			if delay := effect.Starts.Sub(mode.effects().Active[i-1].Starts); delay != scene.MultiHitDelay {
				t.Fatalf("hit delay %d = %s, want %s", i, delay, scene.MultiHitDelay)
			}
		}
	}
	if len(mode.damageFloaters) != 8 {
		t.Fatalf("damage floaters = %d, want 8", len(mode.damageFloaters))
	}
	wantFloaters := []struct {
		text string
		kind damageFloaterKind
	}{
		{text: "252", kind: damageFloaterNormal},
		{text: "252", kind: damageFloaterCombo},
		{text: "252", kind: damageFloaterNormal},
		{text: "504", kind: damageFloaterCombo},
		{text: "252", kind: damageFloaterNormal},
		{text: "756", kind: damageFloaterCombo},
		{text: "252", kind: damageFloaterNormal},
		{text: "1008", kind: damageFloaterCombo},
	}
	for i, want := range wantFloaters {
		floater := mode.damageFloaters[i]
		if floater.text != want.text || floater.kind != want.kind {
			t.Fatalf("floater %d = %+v, want text=%q kind=%d", i, floater, want.text, want.kind)
		}
		if i > 1 {
			if delay := floater.starts.Sub(mode.damageFloaters[i-2].starts); delay != scene.MultiHitDelay {
				t.Fatalf("floater %d delay = %s, want %s", i, delay, scene.MultiHitDelay)
			}
		}
		if floater.kind == damageFloaterCombo {
			if floater.duration != damageFloaterDuration(damageFloaterCombo) {
				t.Fatalf("combo floater %d duration = %s, want %s", i, floater.duration, damageFloaterDuration(damageFloaterCombo))
			}
			wantVisible := damageFloaterComboTransientDuration()
			if i == len(wantFloaters)-1 {
				wantVisible = damageFloaterDuration(damageFloaterCombo)
			}
			if visible := floater.expires.Sub(floater.starts); visible != wantVisible {
				t.Fatalf("combo floater %d visible = %s, want %s", i, visible, wantVisible)
			}
		}
	}
}

func TestLightningBoltStartsWithSkillAction(t *testing.T) {
	for _, tt := range []struct {
		name   string
		hits   uint16
		damage int32
	}{
		{"single hit", 1, 100},
		{"four hits", 4, 400},
		{"ten hits", 10, 1000},
		{"zero damage", 4, 0},
	} {
		t.Run(tt.name, func(t *testing.T) {
			world := worldstate.New()
			world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20}
			world.UpsertActor(worldstate.Actor{ID: 300, X: 11, Y: 20, Job: 1002, ObjectType: scene.ActorObjectTypeMob, HasObjectType: true})
			mode := &WorldMode{}
			ctx := client.Context{Session: &session.Session{AccountID: 2000000}, World: world}
			mode.applyActorActionNotify(ctx, network.ActorActionNotify{
				SkillID: db.SkillMGLightningbolt, SkillLevel: tt.hits,
				SourceID: 2000000, TargetID: 300, SourceSpeed: 580, TargetSpeed: 480,
				Damage: tt.damage, HitCount: tt.hits, Action: network.ActorActionSkill,
			})

			hitCount := int(tt.hits)
			if tt.damage == 0 {
				hitCount = 0
			}
			if len(mode.effects().Active) != 1+hitCount {
				t.Fatalf("effects = %d, want one lightning animation and %d impacts", len(mode.effects().Active), hitCount)
			}
			started := mode.actorAnims[2000000].started
			bolt := mode.effects().Active[0]
			if started.IsZero() || bolt.EffectID != effects.LightningBolt || bolt.ActorID != 300 || !bolt.Starts.Equal(started) {
				t.Fatalf("lightning = %+v, want one animation on target starting with caster at %s", bolt, started)
			}
			for i, hit := range mode.effects().Active[1:] {
				want := started.Add(580*time.Millisecond + time.Duration(i)*200*time.Millisecond)
				if hit.EffectID != effects.WindHit || hit.ActorID != 300 || !hit.Starts.Equal(want) {
					t.Fatalf("impact %d = %+v, want wind hit on target at %s", i, hit, want)
				}
			}
			var damageTimes []time.Time
			for _, floater := range mode.damageFloaters {
				if floater.kind == damageFloaterNormal {
					damageTimes = append(damageTimes, floater.starts)
				}
			}
			if len(damageTimes) != hitCount {
				t.Fatalf("damage numbers = %d, want %d", len(damageTimes), hitCount)
			}
			for i, hitAt := range damageTimes {
				if !hitAt.Equal(mode.effects().Active[i+1].Starts) {
					t.Fatalf("damage %d does not coincide with its impact", i)
				}
			}
		})
	}
}

func TestLightningBoltHasNoExtraImpactFlash(t *testing.T) {
	spec, ok := effects.Lookup(effects.LightningBolt)
	if !ok || len(spec.Components) != 1 || spec.Components[0].StrFile != "lightning" {
		t.Fatal("lightning must play only its STR; impact flashes are scheduled per damage hit")
	}
}

func TestZeroDamageSkillKeepsExecutionEffectsWithoutHitReaction(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20, Dir: 4}
	world.UpsertActor(worldstate.Actor{
		ID:            300,
		X:             11,
		Y:             20,
		Job:           1063,
		ObjectType:    scene.ActorObjectTypeMob,
		HasObjectType: true,
	})
	mode := &WorldMode{}
	ctx := client.Context{
		Session: &session.Session{AccountID: 2000000, CharID: 150000},
		World:   world,
	}

	mode.applyActorActionNotify(ctx, network.ActorActionNotify{
		SkillID:     db.SkillMGSoulstrike,
		SkillLevel:  8,
		SourceID:    2000000,
		TargetID:    300,
		SourceSpeed: 580,
		TargetSpeed: 480,
		HitCount:    4,
		Action:      8,
	})

	if len(mode.effects().Active) != 1 || mode.effects().Active[0].EffectID != effects.SoulStrike || mode.effects().Active[0].HitCount != 4 || mode.effects().Active[0].ActorID != 2000000 || mode.effects().Active[0].TargetID != 300 {
		t.Fatalf("world effects = %+v, want one four-soul sequence from caster to target", mode.effects().Active)
	}
	if _, ok := mode.actorAnims[300]; ok {
		t.Fatal("zero-damage target received a hurt animation")
	}
	if len(mode.scheduledSounds) != 4 {
		t.Fatalf("scheduled sounds = %+v, want four Soul Strike effect sounds", mode.scheduledSounds)
	}
	for i, sound := range mode.scheduledSounds {
		if len(sound.paths) != 1 || sound.paths[0] != "effect\\ef_soulstrike.wav" {
			t.Fatalf("scheduled sound %d = %+v, want only the Soul Strike effect sound", i, sound)
		}
	}
	if len(mode.damageFloaters) != 1 || mode.damageFloaters[0].text != "miss" || mode.damageFloaters[0].kind != damageFloaterMiss {
		t.Fatalf("damage floaters = %+v, want one miss", mode.damageFloaters)
	}
}

func TestZeroDamageSkillKeepsImmediateEffectWithoutHitEffect(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20, Dir: 4}
	world.UpsertActor(worldstate.Actor{
		ID:            300,
		X:             11,
		Y:             20,
		Job:           1063,
		ObjectType:    scene.ActorObjectTypeMob,
		HasObjectType: true,
	})
	mode := &WorldMode{}
	ctx := client.Context{
		Session: &session.Session{AccountID: 2000000, CharID: 150000},
		World:   world,
	}

	mode.applyActorActionNotify(ctx, network.ActorActionNotify{
		SkillID:     db.SkillMGFrostdiver,
		SkillLevel:  1,
		SourceID:    2000000,
		TargetID:    300,
		SourceSpeed: 580,
		TargetSpeed: 480,
		HitCount:    1,
		Action:      network.ActorActionSkill,
	})

	if len(mode.effects().Active) != 1 || mode.effects().Active[0].EffectID != effects.FrostDiver {
		t.Fatalf("world effects = %+v, want Frost Diver execution effect without hit effect", mode.effects().Active)
	}
	wantStart := mode.actorAnims[2000000].started
	if !mode.effects().Active[0].Starts.Equal(wantStart) {
		t.Fatalf("Frost Diver start = %s, want action time %s", mode.effects().Active[0].Starts, wantStart)
	}
}

func TestSkillCastAuraEffectMappings(t *testing.T) {
	tests := []struct {
		property uint32
		want     int
	}{
		{property: 0, want: effects.BeginSpell},
		{property: 1, want: effects.BeginSpell2},
		{property: 2, want: effects.BeginSpell4},
		{property: 3, want: effects.BeginSpell3},
		{property: 4, want: effects.BeginSpell5},
		{property: 5, want: effects.BeginSpell7},
		{property: 6, want: effects.BeginSpell6},
		{property: 7, want: effects.BeginSpell},
		{property: 8, want: effects.BeginSpell},
		{property: 9, want: effects.BeginSpell},
	}
	for _, tt := range tests {
		if got := skillCastAuraEffectID(tt.property); got != tt.want {
			t.Fatalf("cast aura property %d = %d, want %d", tt.property, got, tt.want)
		}
	}
}

func TestSkillVisualMetadataMappings(t *testing.T) {
	if skillAction(5).action != skillActorActionAttack || skillAction(7).action != skillActorActionAttack {
		t.Fatalf("swordman weapon-action skills = bash:%d magnum:%d", skillAction(5).action, skillAction(7).action)
	}
	if skillAction(8).action != skillActorActionReadyFight {
		t.Fatalf("endure action = %d, want ready fight", skillAction(8).action)
	}
	if skillAction(28).action != skillActorActionSkill {
		t.Fatalf("heal action = %d, want default skill action", skillAction(28).action)
	}
	defaultAction := skillAction(28)
	if !defaultAction.play || defaultAction.repeat || defaultAction.next == nil || defaultAction.next.action != skillActorActionIdle || !defaultAction.next.repeat {
		t.Fatalf("default skill action shape = %+v next=%+v, want robr-style skill action followed by repeating idle", defaultAction, defaultAction.next)
	}
	if size := skillCastGroundSampleSize(19); size != 1 {
		t.Fatalf("firebolt marker size = %.1f, want default 1", size)
	}
	for skillID, wantSize := range map[uint16]float64{
		db.SkillACShower:        5,
		db.SkillALPneuma:        5,
		db.SkillASGrimtooth:     5,
		db.SkillASVenomdust:     5,
		db.SkillBSHammerfall:    5,
		db.SkillCRSlimpitcher:   9,
		db.SkillHTBlitzbeat:     5,
		db.SkillHTDetecting:     5,
		db.SkillHWGanbantein:    5,
		db.SkillHWGravitation:   7,
		db.SkillMGFireball:      7,
		db.SkillMGFirewall:      2,
		db.SkillMGNapalmbeat:    5,
		db.SkillMGThunderstorm:  7,
		db.SkillPRBenedictio:    5,
		db.SkillPRMagnus:        9,
		db.SkillPRSanctuary:     7,
		db.SkillSNFalconassault: 5,
		db.SkillWZFirepillar:    5,
		db.SkillWZFrostnova:     5,
		db.SkillWZHeavendrive:   7,
		db.SkillWZMeteor:        11,
		db.SkillWZQuagmire:      7,
		db.SkillWZStormgust:     11,
		db.SkillWZVermilion:     13,
		db.SkillWZWaterball:     5,
	} {
		if size := skillCastGroundSampleSize(skillID); size != wantSize {
			t.Fatalf("skill %d marker size = %.1f, want classic client rendered scope %.1f", skillID, size, wantSize)
		}
		gotSpeed := skillCastGroundSampleRotationRadiansPerSecond(skillID)
		wantSpeed := 0.5 / wantSize * 60
		if math.Abs(gotSpeed-wantSpeed) > 0.001 {
			t.Fatalf("skill %d marker speed = %.4f rad/s, want %.4f", skillID, gotSpeed, wantSpeed)
		}
	}
	if speed := skillCastGroundSampleRotationRadiansPerSecond(19); speed != 0 {
		t.Fatalf("default marker speed override = %.4f, want fallback path", speed)
	}
}

func TestWindHitEffectSpecMatchesRobrowserRandomSTRAndSFX(t *testing.T) {
	spec, ok := effects.Lookup(effects.WindHit)
	if !ok {
		t.Fatal("wind hit effect missing")
	}
	if len(spec.Components) != 1 {
		t.Fatalf("components = %d, want 1", len(spec.Components))
	}
	component := spec.Components[0]
	if component.Kind != effects.ComponentSTR || component.StrFile != "windhit%d" || component.StrRandMin != 1 || component.StrRandMax != 3 || !component.AttachedEntity {
		t.Fatalf("wind hit component = %+v", component)
	}
	if len(spec.Sfx) != 1 || spec.Sfx[0] != "_hit_fist%d.wav" || spec.SfxRandMin != 1 || spec.SfxRandMax != 3 {
		t.Fatalf("wind hit sfx = %+v rand=%d..%d", spec.Sfx, spec.SfxRandMin, spec.SfxRandMax)
	}

	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20}
	mode := &WorldMode{}
	ctx := client.Context{Session: &session.Session{AccountID: 2000000}, World: world}
	if !mode.effects().AddAt(ctx, effects.WindHit, 2000000, time.Unix(10, 20)) {
		t.Fatal("wind hit effect was not added")
	}
	if len(mode.scheduledSounds) != 1 || len(mode.scheduledSounds[0].paths) != 1 {
		t.Fatalf("scheduled sounds = %+v", mode.scheduledSounds)
	}
	path := mode.scheduledSounds[0].paths[0]
	if strings.Contains(path, "%d") || !strings.HasPrefix(path, "_hit_fist") || !strings.HasSuffix(path, ".wav") {
		t.Fatalf("scheduled wind hit sound path = %q", path)
	}
}

func TestSkillCastNotifyAddsDurationAura(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20, Dir: 4}
	world.UpsertActor(worldstate.Actor{ID: 1100, X: 12, Y: 20})
	mode := &WorldMode{}
	ctx := client.Context{Session: &session.Session{AccountID: 2000000, CharID: 150000}, World: world}

	mode.applySkillCastNotify(ctx, network.SkillCastNotify{SourceID: 2000000, TargetID: 1100, SkillID: 20, Property: 4, DelayTime: 2500})

	if len(mode.effects().Active) != 3 {
		t.Fatalf("world effects = %d, want 3", len(mode.effects().Active))
	}
	circle := mode.effects().Active[0]
	if circle.EffectID != effects.CastRing || circle.ActorID != 2000000 || circle.TargetID != 0 || circle.Duration != 2500*time.Millisecond {
		t.Fatalf("circle = %+v", circle)
	}
	lockon := mode.effects().Active[1]
	if lockon.EffectID != effects.LockOnTarget || lockon.ActorID != 1100 || lockon.TargetID != 0 || lockon.Duration != 2500*time.Millisecond {
		t.Fatalf("lockon = %+v", lockon)
	}
	aura := mode.effects().Active[2]
	if aura.EffectID != effects.BeginSpell5 || aura.ActorID != 2000000 || aura.TargetID != 1100 || aura.Duration != 2500*time.Millisecond {
		t.Fatalf("aura = %+v", aura)
	}
	bar, ok := mode.actorCastBars[150000]
	if !ok {
		t.Fatal("local cast bar missing")
	}
	if bar.duration != 2500*time.Millisecond || bar.color != (color.RGBA{R: 0, G: 255, B: 0, A: 255}) {
		t.Fatalf("cast bar = %+v", bar)
	}
	anim, ok := mode.actorAnims[150000]
	if !ok {
		t.Fatal("cast animation missing")
	}
	if anim.actionFamily != sprites.ActionPCReadyFight || anim.duration != 2500*time.Millisecond || anim.hasFixedMotion {
		t.Fatalf("cast animation = %+v", anim)
	}
	if world.Dir != directionFromDelta(10, 20, 12, 20, 4) {
		t.Fatalf("cast dir = %d", world.Dir)
	}
}

func TestArrowRepelCastUsesNativeFlash(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20, Dir: 4}
	world.UpsertActor(worldstate.Actor{ID: 1100, X: 12, Y: 20})
	mode := &WorldMode{}
	ctx := client.Context{Session: &session.Session{AccountID: 2000000, CharID: 150000}, World: world}

	mode.applySkillCastNotify(ctx, network.SkillCastNotify{SourceID: 2000000, TargetID: 1100, SkillID: db.SkillACChargearrow, Property: 4, DelayTime: 1200})

	if len(mode.effects().Active) != 3 {
		t.Fatalf("world effects = %+v, want cast ring, target lock-on, and Bash flash", mode.effects().Active)
	}
	if mode.effects().Active[0].EffectID != effects.CastRing {
		t.Fatalf("effect = %+v, want cast ring", mode.effects().Active[0])
	}
	if mode.effects().Active[1].EffectID != effects.LockOnTarget || mode.effects().Active[1].ActorID != 1100 {
		t.Fatalf("effect = %+v, want target lock-on", mode.effects().Active[1])
	}
	if e := mode.effects().Active[2]; e.EffectID != effects.BashBegin || e.Duration != 0 {
		t.Fatalf("Arrow Repel needs a single flash with its own duration: %+v", e)
	}
	if _, ok := mode.actorCastBars[150000]; !ok {
		t.Fatal("Arrow Repel cast bar should remain visible")
	}
}

func TestSelfTargetSkillCastDoesNotAddLockOnTarget(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20}
	mode := &WorldMode{}
	ctx := client.Context{Session: &session.Session{AccountID: 2000000}, World: world}

	mode.addSkillCastEffects(ctx, db.SkillMGFireball, 3, 2000000, 2000000, 0, 0, 900*time.Millisecond, time.Now(), "self")

	if len(mode.effects().Active) != 2 {
		t.Fatalf("world effects = %+v, want cast ring and aura", mode.effects().Active)
	}
	if mode.effects().Active[0].EffectID != effects.CastRing || mode.effects().Active[1].EffectID != effects.BeginSpell3 {
		t.Fatalf("effects = %+v", mode.effects().Active)
	}
}

func TestSkillCastEffectsDedupeServerAndLocalFallback(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20}
	mode := &WorldMode{}
	ctx := client.Context{Session: &session.Session{AccountID: 2000000}, World: world}

	start := time.Now()
	mode.addSkillCastEffects(ctx, 19, 3, 2000000, 1100, 0, 0, 2800*time.Millisecond, start, "local")
	mode.addSkillCastEffects(ctx, 19, 3, 2000000, 1100, 0, 0, 2800*time.Millisecond, start.Add(20*time.Millisecond), "server")

	if len(mode.effects().Active) != 2 {
		t.Fatalf("world effects = %d, want 2", len(mode.effects().Active))
	}
	if mode.effects().Active[0].EffectID != effects.CastRing || mode.effects().Active[1].EffectID != effects.BeginSpell3 {
		t.Fatalf("effects = %+v", mode.effects().Active)
	}
}

func TestSkillResultAnimationReplacesCastStance(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20, Job: 4, Dir: 4}
	world.UpsertActor(worldstate.Actor{ID: 1100, X: 12, Y: 20})
	mode := &WorldMode{}
	ctx := client.Context{Session: &session.Session{AccountID: 2000000, CharID: 150000}, World: world}

	mode.applySkillCastNotify(ctx, network.SkillCastNotify{SourceID: 2000000, TargetID: 1100, SkillID: 28, DelayTime: 1200})
	if anim, ok := mode.actorAnims[150000]; !ok || anim.actionFamily != sprites.ActionPCReadyFight {
		t.Fatalf("cast stance animation = %+v ok=%t", anim, ok)
	}

	mode.applySkillNoDamageNotify(ctx, network.SkillNoDamageNotify{SourceID: 2000000, TargetID: 1100, SkillID: 28, Amount: 234, Result: 1})
	anim, ok := mode.actorAnims[150000]
	if !ok {
		t.Fatal("skill result animation missing")
	}
	if anim.actionFamily != sprites.ActionPCSkill || anim.hasFixedMotion {
		t.Fatalf("skill result animation = %+v, want delivery skill action", anim)
	}
}

func TestGroundSkillCastEffectsAddGroundSampleMarker(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20}
	mode := &WorldMode{}
	ctx := client.Context{Session: &session.Session{AccountID: 2000000}, World: world}

	start := time.Now()
	mode.addSkillCastEffects(ctx, 21, 4, 2000000, 0, 123, 456, 1800*time.Millisecond, start, "local-ground")

	if len(mode.effects().Active) != 3 {
		t.Fatalf("world effects = %d, want 3", len(mode.effects().Active))
	}
	marker := mode.effects().Active[0]
	wantSpeed := 0.5 / 7 * 60
	if marker.EffectID != effects.GroundSample || marker.ActorID != 0 || marker.X != 123 || marker.Y != 456 || marker.Duration != 1800*time.Millisecond || marker.Size != 7 || math.Abs(marker.GroundSampleRotationRadiansPerSecond-wantSpeed) > 0.001 {
		t.Fatalf("ground marker = %+v", marker)
	}
	if mode.effects().Active[1].EffectID != effects.CastRing || mode.effects().Active[2].EffectID != effects.BeginSpell5 {
		t.Fatalf("cast effects = %+v", mode.effects().Active)
	}
}

func TestAcolyteSkillEffectMappings(t *testing.T) {
	expectEffectIDs(t, "AL_DP passive", skillEffectIDs(22))
	expectEffectIDs(t, "AL_DEMONBANE passive", skillEffectIDs(23))
	expectEffectIDs(t, "AL_RUWACH hit", skillHitEffectIDs(24), effects.Hit1, effects.BashHit)
	expectEffectIDs(t, "AL_PNEUMA uses skill unit", skillGroundEffectIDs(25))
	expectEffectIDs(t, "AL_TELEPORT", skillEffectIDs(26))
	expectEffectIDs(t, "AL_WARP", skillEffectIDs(27))
	expectEffectIDs(t, "AL_HEAL offensive", skillEffectIDs(28), effects.HealOffensive)
	expectEffectIDs(t, "AL_HEAL hit", skillHitEffectIDs(28), effects.Hit1, effects.BashHit)
	expectEffectIDs(t, "AL_INCAGI", skillSuccessEffectIDs(29), effects.IncAgility)
	expectEffectIDs(t, "AL_DECAGI", skillSuccessEffectIDs(30), effects.DecAgility)
	expectEffectIDs(t, "AL_HOLYWATER", skillSuccessEffectSelfIDs(31), effects.Aqua)
	expectEffectIDs(t, "AL_CRUCIS", skillSuccessEffectSelfIDs(32), effects.Signum)
	expectEffectIDs(t, "AL_ANGELUS", skillSuccessEffectSelfIDs(33), effects.Angelus)
	expectEffectIDs(t, "AL_BLESSING", skillSuccessEffectIDs(34), effects.Blessing)
	expectEffectIDs(t, "AL_CURE", skillSuccessEffectIDs(35), effects.Cure)
}

func TestMercenarySupportSkillEffectMappings(t *testing.T) {
	expectEffectIDs(t, "MER_MAGNIFICAT", skillEffectIDs(db.SkillMerMagnificat), effects.Magnificat)
	expectEffectIDs(t, "MER_QUICKEN", skillEffectIDs(db.SkillMerQuicken), effects.TwoHandQuicken)
	expectEffectIDs(t, "MER_PROVOKE success", skillSuccessEffectIDs(db.SkillMerProvoke), effects.Provoke)
	expectEffectIDs(t, "MER_DECAGI", skillEffectIDs(db.SkillMerDecagi), effects.DecAgility)
	expectEffectIDs(t, "MER_LEXDIVINA", skillEffectIDs(db.SkillMerLexdivina), effects.LexDivina)
	expectEffectIDs(t, "MER_ESTIMATION", skillEffectIDs(db.SkillMerEstimation))
	expectEffectIDs(t, "MER_KYRIE", skillEffectIDs(db.SkillMerKyrie), effects.Kyrie)
	expectEffectIDs(t, "MER_BLESSING", skillEffectIDs(db.SkillMerBlessing), effects.Blessing)
	expectEffectIDs(t, "MER_INCAGI", skillEffectIDs(db.SkillMerIncagi), effects.IncAgility)

	for _, tc := range []struct {
		name    string
		skillID uint16
	}{
		{"MER_SIGHT", db.SkillMerSight},
		{"MER_CRASH", db.SkillMerCrash},
		{"MER_REGAIN", db.SkillMerRegain},
		{"MER_TENDER", db.SkillMerTender},
		{"MER_BENEDICTION", db.SkillMerBenediction},
		{"MER_RECUPERATE", db.SkillMerRecuperate},
		{"MER_MENTALCURE", db.SkillMerMentalcure},
		{"MER_COMPRESS", db.SkillMerCompress},
		{"MER_AUTOBERSERK", db.SkillMerAutoberserk},
		{"MER_SCAPEGOAT", db.SkillMerScapegoat},
	} {
		expectEffectIDs(t, tc.name, skillEffectIDs(tc.skillID))
		expectEffectIDs(t, tc.name+" success", skillSuccessEffectIDs(tc.skillID))
	}
}

func TestImportedSkillEffectFallback(t *testing.T) {
	expectEffectIDs(t, "PR_IMPOSITIO imported", skillEffectIDs(db.SkillPRImpositio), effects.Impositio)
	expectEffectIDs(t, "ALL_RESURRECTION success", skillSuccessEffectIDs(db.SkillALLResurrection), effects.Resurrection, effects.Revive)
	expectEffectIDs(t, "PR_SUFFRAGIUM imported", skillEffectIDs(db.SkillPRSuffragium), effects.Suffragium)
	expectEffectIDs(t, "PR_ASPERSIO success", skillSuccessEffectIDs(db.SkillPRAspersio), effects.Aspersio)
	expectEffectIDs(t, "PR_BENEDICTIO ground", skillGroundEffectIDs(db.SkillPRBenedictio), effects.Benedictio)
	expectEffectIDs(t, "PR_SANCTUARY healing has no STR", skillEffectIDs(db.SkillPRSanctuary))
	expectEffectIDs(t, "PR_KYRIE success", skillSuccessEffectSelfIDs(db.SkillPRKyrie), effects.Kyrie)
	expectEffectIDs(t, "PR_MAGNIFICAT success", skillSuccessEffectSelfIDs(db.SkillPRMagnificat), effects.Magnificat)
	expectEffectIDs(t, "PR_GLORIA success", skillSuccessEffectSelfIDs(db.SkillPRGloria), effects.Gloria)
	expectEffectIDs(t, "PR_LEXDIVINA imported", skillEffectIDs(db.SkillPRLexdivina), effects.LexDivina)
	expectEffectIDs(t, "PR_LEXAETERNA imported", skillEffectIDs(db.SkillPRLexaeterna), effects.LexAeterna)
	expectEffectIDs(t, "PR_TURNUNDEAD hit", skillHitEffectIDs(db.SkillPRTurnundead), effects.Hit1, effects.HolyLight)
	expectEffectIDs(t, "PR_MAGNUS damage has no STR", skillEffectIDs(db.SkillPRMagnus))
	expectEffectIDs(t, "PR_MAGNUS ground", skillGroundEffectIDs(db.SkillPRMagnus), effects.Magnus)
	expectEffectIDs(t, "PR_SANCTUARY imported ground", skillGroundEffectIDs(db.SkillPRSanctuary), effects.BottomSanc)
	expectEffectIDs(t, "PR_SLOWPOISON imported", skillEffectIDs(db.SkillPRSlowpoison), effects.SlowPoison)
	expectEffectIDs(t, "PR_STRECOVERY imported", skillEffectIDs(db.SkillPRStrecovery), effects.Recovery)
	expectEffectIDs(t, "PR_REDEMPTIO imported empty", skillEffectIDs(db.SkillPRRedemptio))
	expectEffectIDs(t, "MC_IDENTIFY imported empty", skillEffectIDs(db.SkillMCIdentify))
	expectEffectIDs(t, "MC_VENDING imported empty", skillEffectIDs(db.SkillMCVending))
	expectEffectIDs(t, "MC_CHANGECART imported empty", skillEffectIDs(db.SkillMCChangecart))
	expectEffectIDs(t, "MC_CARTDECORATE imported empty", skillEffectIDs(db.SkillMCCartdecorate))
	expectEffectIDs(t, "KN_PIERCE begin", skillBeginEffectIDs(db.SkillKNPierce), effects.PierceSelf)
	expectEffectIDs(t, "KN_PIERCE imported hit", skillHitEffectIDs(db.SkillKNPierce), effects.Pierce)
	expectEffectIDs(t, "KN_BRANDISHSPEAR imported", skillEffectIDs(db.SkillKNBrandishspear), effects.BrandishSpear)
	expectEffectIDs(t, "KN_BRANDISHSPEAR imported caster", skillBeginEffectIDs(db.SkillKNBrandishspear), effects.BrandishSpear2)
	expectEffectIDs(t, "KN_SPEARSTAB imported caster", skillBeginEffectIDs(db.SkillKNSpearstab), effects.SpearStabSelf)
	expectEffectIDs(t, "KN_SPEARBOOMERANG imported caster", skillBeginEffectIDs(db.SkillKNSpearboomerang), effects.SpearBmrSelf)
	expectEffectIDs(t, "KN_SPEARBOOMERANG imported before hit", skillBeforeHitEffectIDs(db.SkillKNSpearboomerang), effects.SpearBoomerang)
	expectEffectIDs(t, "KN_SPEARBOOMERANG imported hit", skillHitEffectIDs(db.SkillKNSpearboomerang), effects.Hit4)
	expectEffectIDs(t, "KN_TWOHANDQUICKEN imported", skillSuccessEffectSelfIDs(db.SkillKNTwohandquicken), effects.TwoHandQuicken)
	expectEffectIDs(t, "KN_ONEHAND imported", skillSuccessEffectSelfIDs(db.SkillKNOnehand), effects.TwoHandQuicken)
	expectEffectIDs(t, "KN_CHARGEATK imported begin", skillBeginEffectIDs(db.SkillKNChargeatk))
	expectEffectIDs(t, "KN_CHARGEATK imported hit", skillHitEffectIDs(db.SkillKNChargeatk), effects.BashHit)
	expectEffectIDs(t, "KN_BOWLINGBASH imported caster", skillBeginEffectIDs(db.SkillKNBowlingbash), effects.BowlingSelf)
	expectEffectIDs(t, "HT_SKIDTRAP imported", skillGroundEffectIDs(db.SkillHTSkidtrap), effects.SkidTrap)
	expectEffectIDs(t, "HT_LANDMINE imported empty", skillEffectIDs(db.SkillHTLandmine))
	expectEffectIDs(t, "HT_ANKLESNARE imported ground", skillGroundEffectIDs(db.SkillHTAnklesnare))
	expectEffectIDs(t, "HT_SHOCKWAVE imported", skillEffectIDs(db.SkillHTShockwave))
	expectEffectIDs(t, "HT_SHOCKWAVE imported hit", skillHitEffectIDs(db.SkillHTShockwave))
	expectEffectIDs(t, "HT_SANDMAN imported hit", skillHitEffectIDs(db.SkillHTSandman))
	expectEffectIDs(t, "HT_FLASHER imported hit", skillHitEffectIDs(db.SkillHTFlasher))
	expectEffectIDs(t, "HT_FREEZINGTRAP imported hit", skillHitEffectIDs(db.SkillHTFreezingtrap), effects.Hit1)
	expectEffectIDs(t, "HT_BLASTMINE imported hit", skillHitEffectIDs(db.SkillHTBlastmine), effects.Hit1)
	expectEffectIDs(t, "HT_CLAYMORE imported hit", skillHitEffectIDs(db.SkillHTClaymoretrap), effects.Hit1)
	expectEffectIDs(t, "HT_REMOVETRAP imported", skillEffectIDs(db.SkillHTRemovetrap), effects.RemoveTrap)
	expectEffectIDs(t, "HT_BLITZBEAT imported", skillHitEffectIDs(db.SkillHTBlitzbeat), effects.BlitzBeat)
	expectEffectIDs(t, "HT_DETECTING imported", skillGroundEffectIDs(db.SkillHTDetecting), effects.Detecting)
	expectEffectIDs(t, "HT_SPRINGTRAP imported", skillEffectIDs(db.SkillHTSpringtrap), effects.SpringTrap)
	expectEffectIDs(t, "HT_TALKIEBOX imported empty", skillEffectIDs(db.SkillHTTalkiebox))
	expectEffectIDs(t, "HT_POWER imported empty", skillEffectIDs(db.SkillHTPower))
	expectEffectIDs(t, "AS_CLOAKING imported", skillEffectIDs(db.SkillASCloaking), effects.Cloaking)
	expectEffectIDs(t, "AS_SONICBLOW imported", skillEffectIDs(db.SkillASSonicblow))
	expectEffectIDs(t, "AS_SONICBLOW imported caster", skillEffectOnCasterIDs(db.SkillASSonicblow), effects.SonicBlow, effects.SonicBlow2)
	expectEffectIDs(t, "AS_SONICBLOW imported hit", skillHitEffectIDs(db.SkillASSonicblow), effects.SonicBlowHit)
	expectEffectIDs(t, "AS_GRIMTOOTH imported", skillEffectIDs(db.SkillASGrimtooth), effects.Grimtooth)
	expectEffectIDs(t, "AS_GRIMTOOTH imported hit", skillHitEffectIDs(db.SkillASGrimtooth), effects.GrimtoothAtk)
	expectEffectIDs(t, "AS_POISONREACT imported", skillSuccessEffectIDs(db.SkillASPoisonreact), effects.PoisonReact)
	expectEffectIDs(t, "AS_POISONREACT imported hit", skillHitEffectIDs(db.SkillASPoisonreact))
	expectEffectIDs(t, "AS_VENOMDUST imported", skillEffectIDs(db.SkillASVenomdust), effects.VenomDust)
	expectEffectIDs(t, "AS_VENOMDUST imported ground", skillGroundEffectIDs(db.SkillASVenomdust), effects.VenomDust2)
	expectEffectIDs(t, "AS_SPLASHER imported", skillSuccessEffectIDs(db.SkillASSplasher), effects.EnchantPoison)
	expectEffectIDs(t, "NPC_DARKCROSS imported", skillEffectIDs(db.SkillNPCDarkcross), effects.DarkGrandCross)
	expectEffectIDs(t, "NPC_DARKSTRIKE imported", skillEffectIDs(db.SkillNPCDarkstrike), effects.DarkSoulStrike)
	expectEffectIDs(t, "NPC_STOP imported", skillEffectIDs(db.SkillNPCStop), effects.NPCStop)
	expectEffectIDs(t, "NPC_POWERUP imported", skillEffectIDs(db.SkillNPCPowerup))
	expectEffectIDs(t, "NPC_DARKBREATH imported", skillEffectIDs(db.SkillNPCDarkbreath), effects.DarkBreath)
	expectEffectIDs(t, "NPC_DEFENDER imported", skillEffectIDs(db.SkillNPCDefender), effects.Defender)
	expectEffectIDs(t, "NPC_KEEPING imported", skillEffectIDs(db.SkillNPCKeeping), effects.Keeping)
	expectEffectIDs(t, "NPC_BLOODDRAIN imported caster", skillEffectOnCasterIDs(db.SkillNPCBlooddrain), effects.BloodDrain)
	expectEffectIDs(t, "NPC_ENERGYDRAIN imported caster", skillEffectOnCasterIDs(db.SkillNPCEnergydrain), effects.EnergyDrain)
	expectEffectIDs(t, "NPC_EARTHQUAKE imported caster", skillEffectOnCasterIDs(db.SkillNPCEarthquake), effects.NPCEarthquake)
	expectEffectIDs(t, "NPC_DRAGONFEAR imported", skillEffectIDs(db.SkillNPCDragonfear), effects.DragonFear)
	expectEffectIDs(t, "NPC_WIDEBLEEDING imported caster", skillEffectOnCasterIDs(db.SkillNPCWidebleeding), effects.WideBleeding)
	expectEffectIDs(t, "NPC_EVILLAND imported ground", skillGroundEffectIDs(db.SkillNPCEvilland), effects.BottomEvilLand)
	expectEffectIDs(t, "NPC_CRITICALWOUND imported hit", skillHitEffectIDs(db.SkillNPCCriticalwound), effects.CriticalWound)
	expectEffectIDs(t, "RG_STEALCOIN imported success", skillSuccessEffectIDs(db.SkillRGStealcoin), effects.StealCoin, effects.RogueCoin)
	expectEffectIDs(t, "RG_BACKSTAP action", skillEffectIDs(db.SkillRGBackstap), effects.BackStab)
	expectEffectIDs(t, "RG_RAID success", skillSuccessEffectSelfIDs(db.SkillRGRaid), effects.TeiHit3)
	expectEffectIDs(t, "RG_STRIPWEAPON imported success", skillSuccessEffectIDs(db.SkillRGStripweapon), effects.StripWeapon)
	expectEffectIDs(t, "RG_STRIPSHIELD imported success", skillSuccessEffectIDs(db.SkillRGStripshield), effects.StripShield)
	expectEffectIDs(t, "RG_STRIPARMOR imported success", skillSuccessEffectIDs(db.SkillRGStriparmor), effects.StripArmor)
	expectEffectIDs(t, "RG_STRIPHELM imported success", skillSuccessEffectIDs(db.SkillRGStriphelm), effects.StripHelm)
	expectEffectIDs(t, "RG_INTIMIDATE caster", skillEffectOnCasterIDs(db.SkillRGIntimidate), effects.Intimidate)
	expectEffectIDs(t, "RG_GRAFFITI imported empty", skillEffectIDs(db.SkillRGGraffiti))
	expectEffectIDs(t, "RG_FLAGGRAFFITI imported empty", skillEffectIDs(db.SkillRGFlaggraffiti))
	expectEffectIDs(t, "RG_CLEANER imported empty", skillEffectIDs(db.SkillRGCleaner))
	expectEffectIDs(t, "RG_CLOSECONFINE success", skillSuccessEffectIDs(db.SkillRGCloseconfine), effects.QuakeBody4)
	expectEffectIDs(t, "RG_CLOSECONFINE imported ground", skillGroundEffectIDs(db.SkillRGCloseconfine))
	expectEffectIDs(t, "AM_PHARMACY imported empty", skillEffectIDs(db.SkillAMPharmacy))
	expectEffectIDs(t, "AM_DEMONSTRATION imported ground", skillGroundEffectIDs(db.SkillAMDemonstration))
	expectEffectIDs(t, "AM_ACIDTERROR at release", skillEffectIDs(db.SkillAMAcidterror), effects.ThrowItem)
	expectEffectIDs(t, "AM_POTIONPITCHER imported", skillEffectIDs(db.SkillAMPotionpitcher))
	expectEffectIDs(t, "AM_CANNIBALIZE imported empty", skillEffectIDs(db.SkillAMCannibalize))
	expectEffectIDs(t, "AM_SPHEREMINE imported empty", skillEffectIDs(db.SkillAMSpheremine))
	expectEffectIDs(t, "AM_CP_WEAPON imported", skillSuccessEffectIDs(db.SkillAMCpWeapon), effects.ChemicalProt)
	expectEffectIDs(t, "AM_CP_SHIELD imported", skillSuccessEffectIDs(db.SkillAMCpShield), effects.ChemicalProt)
	expectEffectIDs(t, "AM_CP_ARMOR imported", skillSuccessEffectIDs(db.SkillAMCpArmor), effects.ChemicalProt)
	expectEffectIDs(t, "AM_CP_HELM imported", skillSuccessEffectIDs(db.SkillAMCpHelm), effects.ChemicalProt)
	expectEffectIDs(t, "AM_CALLHOMUN imported empty", skillEffectIDs(db.SkillAMCallhomun))
	expectEffectIDs(t, "AM_REST imported empty", skillEffectIDs(db.SkillAMRest))
	expectEffectIDs(t, "AM_RESURRECTHOMUN imported empty", skillEffectIDs(db.SkillAMResurrecthomun))
	expectEffectIDs(t, "CR_AUTOGUARD imported", skillSuccessEffectSelfIDs(db.SkillCRAutoguard), effects.Guard)
	expectEffectIDs(t, "CR_SHIELDCHARGE imported", skillEffectIDs(db.SkillCRShieldcharge), effects.ShieldCharge)
	expectEffectIDs(t, "CR_SHIELDBOOMERANG imported", skillEffectIDs(db.SkillCRShieldboomerang), effects.ShieldBoomer)
	expectEffectIDs(t, "CR_SHIELDBOOMERANG imported before hit", skillBeforeHitEffectIDs(db.SkillCRShieldboomerang))
	expectEffectIDs(t, "CR_REFLECTSHIELD imported", skillSuccessEffectSelfIDs(db.SkillCRReflectshield), effects.ReflectShield)
	expectEffectIDs(t, "CR_HOLYCROSS imported", skillEffectIDs(db.SkillCRHolycross), effects.HolyCross)
	expectEffectIDs(t, "CR_GRANDCROSS imported", skillEffectOnCasterIDs(db.SkillCRGrandcross), effects.GrandCross)
	expectEffectIDs(t, "CR_DEVOTION imported", skillSuccessEffectSelfIDs(db.SkillCRDevotion), effects.Devotion)
	expectEffectIDs(t, "CR_PROVIDENCE imported", skillEffectIDs(db.SkillCRProvidence), effects.Providence)
	expectEffectIDs(t, "CR_DEFENDER imported", skillSuccessEffectSelfIDs(db.SkillCRDefender), effects.CrusaderDef)
	expectEffectIDs(t, "CR_SPEARQUICKEN imported", skillSuccessEffectSelfIDs(db.SkillCRSpearquicken), effects.SpearQuicken)
	expectEffectIDs(t, "MO_CALLSPIRITS imported empty", skillEffectIDs(db.SkillMOCallspirits))
	expectEffectIDs(t, "MO_ABSORBSPIRITS imported self success", skillSuccessEffectSelfIDs(db.SkillMOAbsorbspirits), effects.AbsorbSpirits)
	expectEffectIDs(t, "MO_BODYRELOCATION imported empty", skillEffectIDs(db.SkillMOBodyrelocation))
	expectEffectIDs(t, "MO_INVESTIGATE imported", skillEffectIDs(db.SkillMOInvestigate), effects.TeiHit2, effects.Chimto)
	expectEffectIDs(t, "MO_FINGEROFFENSIVE imported", skillEffectIDs(db.SkillMOFingeroffensive), effects.Tanji)
	expectEffectIDs(t, "MO_STEELBODY imported", skillEffectIDs(db.SkillMOSteelbody), effects.SteelBody, db.SkillEffectSteelBodyRing, effects.Quake)
	expectEffectIDs(t, "MO_BLADESTOP imported empty", skillEffectIDs(db.SkillMOBladestop))
	expectEffectIDs(t, "MO_EXPLOSIONSPIRITS imported caster", skillEffectOnCasterIDs(db.SkillMOExplosionspirits), effects.Gumgang2, effects.Quake)
	expectEffectIDs(t, "MO_EXTREMITYFIST imported", skillEffectIDs(db.SkillMOExtremityfist), effects.TeiHit1X)
	expectEffectIDs(t, "MO_EXTREMITYFIST imported hit", skillHitEffectIDs(db.SkillMOExtremityfist), effects.Hit1)
	expectEffectIDs(t, "MO_TRIPLEATTACK imported", skillEffectIDs(db.SkillMOTripleattack), effects.TripleAttack)
	expectEffectIDs(t, "MO_CHAINCOMBO imported", skillEffectIDs(db.SkillMOChaincombo), effects.TeiHit1, effects.ChainCombo)
	expectEffectIDs(t, "MO_CHAINCOMBO imported caster", skillEffectOnCasterIDs(db.SkillMOChaincombo), effects.Gumgang3)
	expectEffectIDs(t, "MO_COMBOFINISH imported", skillEffectOnCasterIDs(db.SkillMOCombofinish), effects.Gumgang3, effects.HitLine, effects.Quake)
	expectEffectIDs(t, "SA_CASTCANCEL imported empty", skillEffectIDs(db.SkillSACastcancel))
	expectEffectIDs(t, "SA_MAGICROD success on caster", skillSuccessEffectSelfIDs(db.SkillSAMagicrod), effects.MagicRod)
	expectEffectIDs(t, "SA_SPELLBREAKER imported success", skillSuccessEffectIDs(db.SkillSASpellbreaker), effects.SpellBreaker)
	expectEffectIDs(t, "SA_AUTOSPELL imported empty", skillEffectIDs(db.SkillSAAutospell))
	expectEffectIDs(t, "SA_FLAMELAUNCHER imported success", skillSuccessEffectIDs(db.SkillSAFlamelauncher), effects.FlameLauncher)
	expectEffectIDs(t, "SA_FROSTWEAPON imported success", skillSuccessEffectIDs(db.SkillSAFrostweapon), effects.FrostWeapon)
	expectEffectIDs(t, "SA_LIGHTNINGLOADER imported success", skillSuccessEffectIDs(db.SkillSALightningloader), effects.LightningLoad)
	expectEffectIDs(t, "SA_SEISMICWEAPON imported success", skillSuccessEffectIDs(db.SkillSASeismicweapon), effects.SeismicWeapon)
	expectEffectIDs(t, "SA_DISPELL imported success", skillSuccessEffectIDs(db.SkillSADispell), effects.Dispell)
	expectEffectIDs(t, "SA_VOLCANO ground activation", skillGroundEffectIDs(db.SkillSAVolcano), effects.Volcano)
	expectEffectIDs(t, "SA_DELUGE ground activation", skillGroundEffectIDs(db.SkillSADeluge), effects.Deluge)
	expectEffectIDs(t, "SA_VIOLENTGALE ground activation", skillGroundEffectIDs(db.SkillSAViolentgale), effects.ViolentGale)
	expectEffectIDs(t, "SA_LANDPROTECTOR ground activation", skillGroundEffectIDs(db.SkillSALandprotector), effects.LandProtector)
	expectEffectIDs(t, "SA_ABRACADABRA imported empty", skillEffectIDs(db.SkillSAAbracadabra))
	for _, skillID := range []uint16{
		db.SkillSAMonocell,
		db.SkillSAClasschange,
		db.SkillSASummonmonster,
		db.SkillSAReverseorcish,
		db.SkillSADeath,
		db.SkillSAFortune,
		db.SkillSATamingmonster,
		db.SkillSAQuestion,
		db.SkillSAGravity,
		db.SkillSALevelup,
		db.SkillSAInstantdeath,
		db.SkillSAFullrecovery,
		db.SkillSAComa,
	} {
		expectEffectIDs(t, "SA_ABRACADABRA result imported empty", skillEffectIDs(skillID))
	}
	expectEffectIDs(t, "BD_ADAPTATION imported empty", skillEffectIDs(db.SkillBDAdaptation))
	expectEffectIDs(t, "BD_ENCORE imported empty", skillEffectIDs(db.SkillBDEncore))
	expectEffectIDs(t, "BD_LULLABY imported", skillEffectIDs(db.SkillBDLullaby))
	expectEffectIDs(t, "BD_LULLABY imported ground", skillGroundEffectIDs(db.SkillBDLullaby), effects.BottomLullabyGround)
	expectEffectIDs(t, "BD_RICHMANKIM imported", skillEffectIDs(db.SkillBDRichmankim))
	expectEffectIDs(t, "BD_RICHMANKIM imported ground", skillGroundEffectIDs(db.SkillBDRichmankim), effects.BottomRichKimGround)
	expectEffectIDs(t, "BD_ETERNALCHAOS imported", skillEffectIDs(db.SkillBDEternalchaos))
	expectEffectIDs(t, "BD_ETERNALCHAOS imported ground", skillGroundEffectIDs(db.SkillBDEternalchaos), effects.BottomChaosGround)
	expectEffectIDs(t, "BD_DRUMBATTLEFIELD imported", skillEffectIDs(db.SkillBDDrumbattlefield))
	expectEffectIDs(t, "BD_DRUMBATTLEFIELD imported ground", skillGroundEffectIDs(db.SkillBDDrumbattlefield), effects.BottomDrumGround)
	expectEffectIDs(t, "BD_RINGNIBELUNGEN imported", skillEffectIDs(db.SkillBDRingnibelungen))
	expectEffectIDs(t, "BD_RINGNIBELUNGEN imported ground", skillGroundEffectIDs(db.SkillBDRingnibelungen), effects.BottomNibelungGround)
	expectEffectIDs(t, "BD_ROKISWEIL imported", skillEffectIDs(db.SkillBDRokisweil))
	expectEffectIDs(t, "BD_ROKISWEIL imported ground", skillGroundEffectIDs(db.SkillBDRokisweil), effects.BottomRokiGround)
	expectEffectIDs(t, "BD_INTOABYSS imported", skillEffectIDs(db.SkillBDIntoabyss))
	expectEffectIDs(t, "BD_INTOABYSS imported ground", skillGroundEffectIDs(db.SkillBDIntoabyss), effects.BottomAbyssGround)
	expectEffectIDs(t, "BD_SIEGFRIED imported", skillEffectIDs(db.SkillBDSiegfried))
	expectEffectIDs(t, "BD_SIEGFRIED imported ground", skillGroundEffectIDs(db.SkillBDSiegfried), effects.BottomSiegGround)
	expectEffectIDs(t, "BA_MUSICALLESSON imported empty", skillEffectIDs(db.SkillBaMusicallesson))
	expectEffectIDs(t, "BA_MUSICALSTRIKE imported before hit", skillBeforeHitEffectIDs(db.SkillBaMusicalstrike), effects.ArrowShot)
	if skillHidesCastAura(db.SkillBaMusicalstrike) {
		t.Fatal("BA_MUSICALSTRIKE must show the native Bash casting effect")
	}
	expectEffectIDs(t, "BA_DISSONANCE imported ground", skillGroundEffectIDs(db.SkillBaDissonance), effects.BottomDissonanceGround)
	expectEffectIDs(t, "BA_FROSTJOKE success", skillSuccessEffectSelfIDs(db.SkillBaFrostjoke), effects.TalkFrostJoke)
	expectEffectIDs(t, "BA_WHISTLE imported", skillEffectIDs(db.SkillBaWhistle))
	expectEffectIDs(t, "BA_WHISTLE imported ground", skillGroundEffectIDs(db.SkillBaWhistle), effects.BottomWhistleGround)
	expectEffectIDs(t, "BA_ASSASSINCROSS imported", skillEffectIDs(db.SkillBaAssassincross))
	expectEffectIDs(t, "BA_ASSASSINCROSS imported ground", skillGroundEffectIDs(db.SkillBaAssassincross), effects.BottomSinXGround)
	expectEffectIDs(t, "BA_POEMBRAGI imported", skillEffectIDs(db.SkillBaPoembragi))
	expectEffectIDs(t, "BA_POEMBRAGI imported ground", skillGroundEffectIDs(db.SkillBaPoembragi), effects.BottomBragiGround)
	expectEffectIDs(t, "BA_APPLEIDUN imported", skillEffectIDs(db.SkillBaAppleidun))
	expectEffectIDs(t, "BA_APPLEIDUN imported ground", skillGroundEffectIDs(db.SkillBaAppleidun), effects.BottomAppleGround)
	expectEffectIDs(t, "BA_PANGVOICE imported success", skillSuccessEffectIDs(db.SkillBaPangvoice), effects.FVoice)
	expectEffectIDs(t, "DC_DANCINGLESSON imported empty", skillEffectIDs(db.SkillDCDancinglesson))
	expectEffectIDs(t, "DC_THROWARROW imported before hit", skillBeforeHitEffectIDs(db.SkillDCThrowarrow), effects.ArrowShot)
	if skillHidesCastAura(db.SkillDCThrowarrow) {
		t.Fatal("DC_THROWARROW uses the original casting flare")
	}
	expectEffectIDs(t, "DC_UGLYDANCE imported ground", skillGroundEffectIDs(db.SkillDCUglydance), effects.BottomUglyDanceGround)
	expectEffectIDs(t, "DC_SCREAM original caster chat", skillSuccessEffectSelfIDs(db.SkillDCScream), effects.TalkScream)
	expectEffectIDs(t, "DC_HUMMING imported", skillEffectIDs(db.SkillDCHumming))
	expectEffectIDs(t, "DC_HUMMING imported ground", skillGroundEffectIDs(db.SkillDCHumming), effects.BottomHummingGround)
	expectEffectIDs(t, "DC_DONTFORGETME imported", skillEffectIDs(db.SkillDCDontforgetme))
	expectEffectIDs(t, "DC_DONTFORGETME imported ground", skillGroundEffectIDs(db.SkillDCDontforgetme), effects.BottomForgetGround)
	expectEffectIDs(t, "DC_FORTUNEKISS imported", skillEffectIDs(db.SkillDCFortunekiss))
	expectEffectIDs(t, "DC_FORTUNEKISS imported ground", skillGroundEffectIDs(db.SkillDCFortunekiss), effects.BottomFortuneGround)
	expectEffectIDs(t, "DC_SERVICEFORYOU imported", skillEffectIDs(db.SkillDCServiceforyou))
	expectEffectIDs(t, "DC_SERVICEFORYOU imported ground", skillGroundEffectIDs(db.SkillDCServiceforyou), effects.BottomServiceGround)
	expectEffectIDs(t, "DC_WINKCHARM imported success", skillSuccessEffectIDs(db.SkillDCWinkcharm), effects.Wink)
	expectEffectIDs(t, "SL_KAIZEL imported", skillEffectIDs(db.SkillSLKaizel), effects.Kaizel)
	expectEffectIDs(t, "SL_SMA original", skillEffectIDs(db.SkillSLSma), effects.FourWayBody, effects.Stin2, effects.HitLine6, effects.HitTexture)
	expectEffectIDs(t, "SL_SWOO imported", skillEffectIDs(db.SkillSLSwoo), effects.M07, effects.BabyBody)
	expectEffectIDs(t, "SL_SKA imported", skillEffectIDs(db.SkillSLSka), effects.SteelBody, effects.Gumgang2, effects.Quake)
	expectEffectIDs(t, "AM_BERSERKPITCHER imported", skillSuccessEffectIDs(db.SkillAMBerserkpitcher), effects.ItemFast3)
	expectEffectIDs(t, "AM_BERSERKPITCHER imported before hit", skillBeforeHitEffectIDs(db.SkillAMBerserkpitcher))
	expectEffectIDs(t, "AM_TWILIGHT1 imported", skillEffectIDs(db.SkillAMTwilight1))
	expectEffectIDs(t, "AM_TWILIGHT2 imported", skillEffectIDs(db.SkillAMTwilight2))
	expectEffectIDs(t, "AM_TWILIGHT3 imported", skillEffectIDs(db.SkillAMTwilight3))
	expectEffectIDs(t, "CR_ALCHEMY imported empty", skillEffectIDs(db.SkillCRAlchemy))
	expectEffectIDs(t, "CR_SYNTHESISPOTION imported empty", skillEffectIDs(db.SkillCRSynthesispotion))
	expectEffectIDs(t, "CR_SLIMPITCHER imported empty", skillEffectIDs(db.SkillCRSlimpitcher))
	expectEffectIDs(t, "CR_FULLPROTECTION success", skillSuccessEffectIDs(db.SkillCRFullprotection), effects.ChemicalProt, effects.ChemicalBody)
	expectEffectIDs(t, "CR_CULTIVATION imported empty", skillEffectIDs(db.SkillCRCultivation))
	expectEffectIDs(t, "SA_CREATECON imported empty", skillEffectIDs(db.SkillSACreatecon))
	expectEffectIDs(t, "SA_ELEMENTWATER success", skillSuccessEffectIDs(db.SkillSAElementwater), effects.FrostWeapon)
	expectEffectIDs(t, "SA_ELEMENTGROUND success", skillSuccessEffectIDs(db.SkillSAElementground), effects.SeismicWeapon)
	expectEffectIDs(t, "SA_ELEMENTFIRE success", skillSuccessEffectIDs(db.SkillSAElementfire), effects.FlameLauncher)
	expectEffectIDs(t, "SA_ELEMENTWIND success", skillSuccessEffectIDs(db.SkillSAElementwind), effects.LightningLoad)
	expectEffectIDs(t, "MO_KITRANSLATION imported empty", skillEffectIDs(db.SkillMOKitranslation))
	expectEffectIDs(t, "MO_BALKYOUNG imported", skillHitEffectIDs(db.SkillMOBalkyoung), effects.Hit3)
	expectEffectIDs(t, "HP_ASSUMPTIO", skillEffectIDs(db.SkillHPAssumptio), effects.Assumptio, effects.Assumptio2)
	expectEffectIDs(t, "HP_BASILICA imported ground", skillGroundEffectIDs(db.SkillHPBasilica), effects.BottomBasilica)
	expectEffectIDs(t, "HP_MEDITATIO has no robr effect row", skillEffectIDs(db.SkillHPMeditatio))
	expectEffectIDs(t, "HP_MANARECHARGE has no robr effect row", skillEffectIDs(db.SkillHPManarecharge))
	expectEffectIDs(t, "PA_PRESSURE imported before hit", skillEffectIDs(db.SkillPaPressure), effects.Pressure)
	expectEffectIDs(t, "PA_SACRIFICE imported", skillEffectOnCasterIDs(db.SkillPaSacrifice), effects.Bash3D)
	expectEffectIDs(t, "PA_GOSPEL imported", skillEffectIDs(db.SkillPaGospel), effects.BottomGospel)
	expectEffectIDs(t, "PA_GOSPEL imported ground", skillGroundEffectIDs(db.SkillPaGospel), effects.GospelGround)
	expectEffectIDs(t, "PA_SHIELDCHAIN imported before hit", skillEffectIDs(db.SkillPaShieldchain), effects.ShieldBoomerang3)
	expectEffectIDs(t, "CH_PALMSTRIKE imported hit", skillHitEffectIDs(db.SkillChPalmstrike), effects.Hit1)
	expectEffectIDs(t, "CH_TIGERFIST imported", skillEffectOnCasterIDs(db.SkillChTigerfist), effects.Bash3D2, effects.Quake, effects.Gumgang3)
	expectEffectIDs(t, "CH_CHAINCRUSH imported", skillEffectIDs(db.SkillChChaincrush), effects.Chemical2Dash)
	expectEffectIDs(t, "CH_SOULCOLLECT imported begin", skillBeginEffectIDs(db.SkillChSoulcollect))
	expectEffectIDs(t, "SN_SIGHT imported", skillSuccessEffectSelfIDs(db.SkillSNSight), effects.TrueSight)
	expectEffectIDs(t, "SN_FALCONASSAULT imported", skillEffectIDs(db.SkillSNFalconassault), effects.FalconAssault)
	expectEffectIDs(t, "HT_PHANTASMIC imported before hit", skillBeforeHitEffectIDs(db.SkillHTPhantasmic), effects.ArrowShot)
	expectEffectIDs(t, "HT_PHANTASMIC imported hit", skillHitEffectIDs(db.SkillHTPhantasmic), effects.Hit1, effects.BashHit)
	expectEffectIDs(t, "SN_SHARPSHOOTING imported begin", skillBeginEffectIDs(db.SkillSNSharpshooting))
	expectEffectIDs(t, "SN_SHARPSHOOTING imported before hit", skillBeforeHitEffectIDs(db.SkillSNSharpshooting), effects.ArrowShot)
	expectEffectIDs(t, "SN_SHARPSHOOTING imported hit", skillHitEffectIDs(db.SkillSNSharpshooting), effects.Hit1, effects.BashHit)
	expectEffectIDs(t, "SN_WINDWALK imported", skillSuccessEffectSelfIDs(db.SkillSNWindwalk), effects.Portal4)
	expectEffectIDs(t, "WS_CREATECOIN imported empty", skillEffectIDs(db.SkillWSCreatecoin))
	expectEffectIDs(t, "WS_CREATENUGGET imported empty", skillEffectIDs(db.SkillWSCreatenugget))
	expectEffectIDs(t, "WS_SYSTEMCREATE imported empty", skillEffectIDs(db.SkillWSSystemcreate))
	expectEffectIDs(t, "WS_WEAPONREFINE imported empty", skillEffectIDs(db.SkillWSWeaponrefine))
	expectEffectIDs(t, "CG_ARROWVULCAN imported", skillEffectIDs(db.SkillCGArrowvulcan), effects.TripleAttack3)
	expectEffectIDs(t, "CG_ARROWVULCAN imported before hit", skillBeforeHitEffectIDs(db.SkillCGArrowvulcan), effects.ArrowShot)
	expectEffectIDs(t, "CG_MOONLIT imported", skillEffectIDs(db.SkillCGMoonlit))
	expectEffectIDs(t, "CG_MOONLIT imported ground", skillGroundEffectIDs(db.SkillCGMoonlit))
	expectEffectIDs(t, "CG_MARIONETTE imported", skillEffectIDs(db.SkillCGMarionette))
	expectEffectIDs(t, "CG_MARIONETTE imported hit", skillHitEffectIDs(db.SkillCGMarionette))
	expectEffectIDs(t, "CG_LONGINGFREEDOM imported", skillSuccessEffectSelfIDs(db.SkillCGLongingfreedom), effects.ChemicalBody)
	expectEffectIDs(t, "CG_HERMODE imported music", skillEffectIDs(db.SkillCGHermode))
	expectEffectIDs(t, "CG_HERMODE imported ground", skillGroundEffectIDs(db.SkillCGHermode))
	expectEffectIDs(t, "CG_TAROTCARD imported success", skillSuccessEffectIDs(db.SkillCGTarotcard), effects.ChemicalBody)
	expectEffectIDs(t, "CR_ACIDDEMONSTRATION imported", skillEffectIDs(db.SkillCRAciddemonstration), effects.AcidDemon)
	expectEffectIDs(t, "SL_KAAHI imported", skillEffectIDs(db.SkillSLKaahi), effects.Hated)
	expectEffectIDs(t, "WE_MALE imported empty", skillEffectIDs(db.SkillWEMale))
	expectEffectIDs(t, "WE_FEMALE imported empty", skillEffectIDs(db.SkillWEFemale))
	expectEffectIDs(t, "WE_CALLPARTNER imported empty", skillEffectIDs(db.SkillWECallpartner))
	expectEffectIDs(t, "WE_BABY imported", skillEffectIDs(db.SkillWEBaby), 408)
	expectEffectIDs(t, "WE_CALLPARENT imported empty", skillEffectIDs(db.SkillWECallparent))
	expectEffectIDs(t, "WE_CALLBABY imported empty", skillEffectIDs(db.SkillWECallbaby))
	expectEffectIDs(t, "TK_JUMPKICK imported hit", skillHitEffectIDs(db.SkillTKJumpkick))
	expectEffectIDs(t, "AS_VENOMKNIFE imported before hit", skillBeforeHitEffectIDs(db.SkillASVenomknife), effects.ThrowItem6)
	expectEffectIDs(t, "ALL_WEWISH imported", skillEffectIDs(db.SkillALLWewish), effects.ChristmasCarol)
	expectEffectIDs(t, "NPC_VENOMFOG imported", skillEffectIDs(db.SkillNPCVenomfog), effects.VenomFog)
	expectEffectIDs(t, "RK_IGNITIONBREAK imported caster", skillEffectOnCasterIDs(db.SkillRKIgnitionbreak), effects.IgnitionBreak)
	expectEffectIDs(t, "RK_DRAGONBREATH imported hit", skillHitEffectIDs(db.SkillRKDragonbreath), effects.M05)
	expectEffectIDs(t, "RK_DRAGONHOWLING imported", skillEffectIDs(db.SkillRKDragonhowling), effects.DragonHowling)
	expectEffectIDs(t, "RK_MILLENNIUMSHIELD imported", skillEffectIDs(db.SkillRKMillenniumshield), effects.MillenniumShield)
	expectEffectIDs(t, "RK_ENCHANTBLADE imported", skillEffectIDs(db.SkillRKEnchantblade), effects.BerserkPotion2)
	expectEffectIDs(t, "RK_SONICWAVE imported", skillEffectIDs(db.SkillRKSonicwave), effects.HealN)
	expectEffectIDs(t, "WL_WHITEIMPRISON imported", skillEffectIDs(db.SkillWLWhiteimprison), effects.BottomBasilica2)
	expectEffectIDs(t, "WL_FROSTMISTY imported", skillEffectIDs(db.SkillWLFrostmisty), effects.FrostMisty)
	expectEffectIDs(t, "WL_MARSHOFABYSS imported", skillEffectIDs(db.SkillWLMarshofabyss), effects.MarshOfAbyss)
	expectEffectIDs(t, "WL_RECOGNIZEDSPELL imported", skillEffectIDs(db.SkillWLRecognizedspell), effects.Recognized)
	expectEffectIDs(t, "WL_STASIS imported", skillEffectIDs(db.SkillWLStasis), effects.Stasis)
	expectEffectIDs(t, "WL_CRIMSONROCK imported", skillEffectIDs(db.SkillWLCrimsonrock), effects.CrimsonRock)
	expectEffectIDs(t, "WL_HELLINFERNO imported ground", skillGroundEffectIDs(db.SkillWLHellinferno), effects.HellInferno)
	expectEffectIDs(t, "WL_CHAINLIGHTNING_ATK imported", skillEffectIDs(db.SkillWLChainlightningAtk), effects.ChainLightning)
	expectEffectIDs(t, "WL_EARTHSTRAIN imported ground", skillGroundEffectIDs(db.SkillWLEarthstrain), effects.EarthWall)
	expectEffectIDs(t, "WL_TETRAVORTEX imported", skillEffectIDs(db.SkillWLTetravortex), effects.Tetra)
	expectEffectIDs(t, "WL_TETRAVORTEX imported begin", skillBeginEffectIDs(db.SkillWLTetravortex), effects.TetraCasting)
	expectEffectIDs(t, "GC_ROLLINGCUTTER imported", skillEffectIDs(db.SkillGCRollingcutter), effects.CastSpin2)
	expectEffectIDs(t, "AB_JUDEX imported", skillEffectIDs(db.SkillABJudex), effects.FirePillarOn2)
	expectEffectIDs(t, "AB_JUDEX imported hit", skillHitEffectIDs(db.SkillABJudex), effects.HolyLight)
	expectEffectIDs(t, "AB_ADORAMUS imported", skillEffectIDs(db.SkillABAdoramus), effects.Adoramus)
	expectEffectIDs(t, "AB_EPICLESIS imported", skillEffectIDs(db.SkillABEpiclesis), effects.GlassWall4)
	expectEffectIDs(t, "AB_EPICLESIS imported ground", skillGroundEffectIDs(db.SkillABEpiclesis), effects.GlassWall3)
	expectEffectIDs(t, "RA_ARROWSTORM imported", skillEffectIDs(db.SkillRAArrowstorm), effects.ArrowStorm)
	expectEffectIDs(t, "RA_AIMEDBOLT imported", skillEffectIDs(db.SkillRAAimedbolt), effects.AimedBolt)
	expectEffectIDs(t, "RA_AIMEDBOLT imported before hit", skillBeforeHitEffectIDs(db.SkillRAAimedbolt), effects.ArrowShot)
	expectEffectIDs(t, "RA_DETONATOR imported", skillEffectIDs(db.SkillRADetonator), effects.Concentration2)
	expectEffectIDs(t, "NC_POWERSWING imported", skillEffectIDs(db.SkillNCPowerswing), effects.CrashAxe)
	expectEffectIDs(t, "SR_EARTHSHAKER imported", skillEffectIDs(db.SkillSREarthshaker), effects.Electric4)
	expectEffectIDs(t, "GC_DARKCROW imported", skillEffectIDs(db.SkillGCDarkcrow), effects.GCDarkCrow)
	expectEffectIDs(t, "GN_ILLUSIONDOPING imported", skillEffectIDs(db.SkillGNIllusiondoping), effects.GNIllusionDoping)
	expectEffectIDs(t, "RK_LUXANIMA imported", skillEffectIDs(db.SkillRKLuxanima), effects.RKLuxAnima)
	expectEffectIDs(t, "NC_MAGMA_ERUPTION imported", skillEffectIDs(db.SkillNCMagmaEruption), effects.NCMagmaEruption)
	expectEffectIDs(t, "SO_ELEMENTAL_SHIELD imported", skillEffectIDs(db.SkillSOElementalShield), effects.SOElemShield)
	expectEffectIDs(t, "SR_FLASHCOMBO imported", skillEffectIDs(db.SkillSRFlashcombo), effects.SRFlashCombo)
	expectEffectIDs(t, "AB_OFFERTORIUM imported", skillEffectIDs(db.SkillABOffertorium), effects.ABOffertorium)
	expectEffectIDs(t, "WL_TELEKINESIS_INTENSE imported", skillEffectIDs(db.SkillWLTelekinesisIntense), effects.WLTelekinesis)
	expectEffectIDs(t, "ALL_FULL_THROTTLE imported", skillEffectIDs(db.SkillALLFullThrottle), effects.AllFullThrottle)
	expectEffectIDs(t, "SC_BODYPAINT imported", skillEffectIDs(db.SkillSCBodypaint), effects.Stretch)
	expectEffectIDs(t, "SC_ENERVATION imported", skillEffectIDs(db.SkillSCEnervation), effects.Enervation)
	expectEffectIDs(t, "SC_GROOMY imported", skillEffectIDs(db.SkillSCGroomy), effects.Enervation2)
	expectEffectIDs(t, "SC_IGNORANCE imported", skillEffectIDs(db.SkillSCIgnorance), effects.Enervation3)
	expectEffectIDs(t, "SC_LAZINESS imported", skillEffectIDs(db.SkillSCLaziness), effects.Enervation4)
	expectEffectIDs(t, "SC_UNLUCKY imported", skillEffectIDs(db.SkillSCUnlucky), effects.Enervation5)
	expectEffectIDs(t, "SC_WEAKNESS imported", skillEffectIDs(db.SkillSCWeakness), effects.Enervation6)
	expectEffectIDs(t, "SC_MANHOLE imported ground", skillGroundEffectIDs(db.SkillSCManhole), effects.BottomManhole)
	expectEffectIDs(t, "SC_MANHOLE imported success", skillSuccessEffectIDs(db.SkillSCManhole), effects.Manhole)
	expectEffectIDs(t, "SC_DIMENSIONDOOR imported ground", skillGroundEffectIDs(db.SkillSCDimensiondoor), effects.ForestLight6)
	expectEffectIDs(t, "SC_CHAOSPANIC imported ground", skillGroundEffectIDs(db.SkillSCChaospanic), effects.BottomAni)
	expectEffectIDs(t, "SC_MAELSTROM imported ground", skillGroundEffectIDs(db.SkillSCMaelstrom), effects.BottomMaelstrom)
	expectEffectIDs(t, "SC_BLOODYLUST imported ground", skillGroundEffectIDs(db.SkillSCBloodylust), effects.BottomBloodyLust)
	expectEffectIDs(t, "LG_SHIELDPRESS imported before hit", skillBeforeHitEffectIDs(db.SkillLGShieldpress), effects.Pressure2)
	expectEffectIDs(t, "LG_PRESTIGE imported", skillEffectIDs(db.SkillLGPrestige), effects.PrimeCharge2)
	expectEffectIDs(t, "LG_BANDING imported", skillEffectIDs(db.SkillLGBanding), effects.PrimeCharge3)
	expectEffectIDs(t, "LG_INSPIRATION imported", skillEffectIDs(db.SkillLGInspiration), effects.PrimeCharge4)
	expectEffectIDs(t, "SO_FIREWALK imported ground", skillGroundEffectIDs(db.SkillSOFirewalk), effects.FireWall2)
	expectEffectIDs(t, "SO_ELECTRICWALK imported ground", skillGroundEffectIDs(db.SkillSOElectricwalk), effects.Shockwave2)
	expectEffectIDs(t, "SO_DIAMONDDUST imported", skillEffectIDs(db.SkillSODiamonddust), effects.ColdThrow2)
	expectEffectIDs(t, "SO_PSYCHIC_WAVE imported", skillEffectIDs(db.SkillSOPsychicWave), effects.SprPlant10)
	expectEffectIDs(t, "SO_WARMER imported", skillEffectIDs(db.SkillSOWarmer), effects.DemonicFire4)
	expectEffectIDs(t, "SO_VARETYR_SPEAR imported before hit", skillBeforeHitEffectIDs(db.SkillSOVaretyrSpear), effects.Pressure3)
	expectEffectIDs(t, "WM_REVERBERATION imported ground", skillGroundEffectIDs(db.SkillWmReverberation), effects.BotReverb)
	expectEffectIDs(t, "WM_REVERBERATION_MELEE imported", skillEffectIDs(db.SkillWmReverberationMelee), effects.BotReverb2)
	expectEffectIDs(t, "WM_SEVERE_RAINSTORM imported", skillEffectIDs(db.SkillWmSevereRainstorm), effects.RainParticle)
	expectEffectIDs(t, "WM_POEMOFNETHERWORLD imported ground", skillGroundEffectIDs(db.SkillWmPoemofnetherworld), effects.BotReverb2)
	expectEffectIDs(t, "WM_VOICEOFSIREN imported ground", skillGroundEffectIDs(db.SkillWmVoiceofsiren), effects.HeartAsura)
	expectEffectIDs(t, "WM_LULLABY_DEEPSLEEP imported", skillEffectIDs(db.SkillWmLullabyDeepsleep), effects.ChemicalV2)
	expectEffectIDs(t, "WM_SIRCLEOFNATURE imported", skillEffectIDs(db.SkillWmSircleofnature), effects.CirclePower2)
	expectEffectIDs(t, "WM_RANDOMIZESPELL imported", skillEffectIDs(db.SkillWmRandomizespell), effects.Secra2)
	expectEffectIDs(t, "WM_GLOOMYDAY imported", skillEffectIDs(db.SkillWmGloomyday), effects.Dance1)
	expectEffectIDs(t, "WM_SONG_OF_MANA imported ground", skillGroundEffectIDs(db.SkillWmSongOfMana), effects.SprPlant3)
	expectEffectIDs(t, "WM_DANCE_WITH_WUG imported ground", skillGroundEffectIDs(db.SkillWmDanceWithWug), effects.SprPlant2)
	expectEffectIDs(t, "WM_SATURDAY_NIGHT_FEVER imported ground", skillGroundEffectIDs(db.SkillWmSaturdayNightFever), effects.SprPlant4)
	expectEffectIDs(t, "WM_LERADS_DEW imported ground", skillGroundEffectIDs(db.SkillWmLeradsDew), effects.SprPlant5)
	expectEffectIDs(t, "WM_MELODYOFSINK imported ground", skillGroundEffectIDs(db.SkillWmMelodyofsink), effects.SprPlant6)
	expectEffectIDs(t, "WM_BEYOND_OF_WARCRY imported ground", skillGroundEffectIDs(db.SkillWmBeyondOfWarcry), effects.SprPlant7)
	expectEffectIDs(t, "WM_UNLIMITED_HUMMING_VOICE imported ground", skillGroundEffectIDs(db.SkillWmUnlimitedHummingVoice), effects.SprPlant8)
	expectEffectIDs(t, "HAMI_CASTLE imported", skillEffectIDs(db.SkillHamiCastle), effects.HamiCastle)
	expectEffectIDs(t, "HAMI_DEFENCE imported", skillEffectIDs(db.SkillHamiDefence), effects.HamiDefence)
	expectEffectIDs(t, "HAMI_BLOODLUST imported", skillEffectIDs(db.SkillHamiBloodlust), effects.HamiBlood)
	expectEffectIDs(t, "MH_POISON_MIST imported", skillEffectIDs(db.SkillMhPoisonMist), effects.PoisonMist)
	expectEffectIDs(t, "MH_ERASER_CUTTER imported", skillEffectIDs(db.SkillMhEraserCutter), effects.EraserCutter)
	expectEffectIDs(t, "MH_SONIC_CRAW imported", skillEffectIDs(db.SkillMhSonicCraw), effects.SonicClaw)
	expectEffectIDs(t, "MH_MIDNIGHT_FRENZY imported", skillEffectIDs(db.SkillMhMidnightFrenzy), effects.MidnightFrenzy)
	expectEffectIDs(t, "MH_TINDER_BREAKER imported", skillEffectIDs(db.SkillMhTinderBreaker), effects.TinderBreaker)
	expectEffectIDs(t, "MH_LAVA_SLIDE imported", skillEffectIDs(db.SkillMhLavaSlide), effects.LavaSlide)
	expectEffectIDs(t, "MH_VOLCANIC_ASH imported", skillEffectIDs(db.SkillMhVolcanicAsh), effects.VolcanicAsh)
	expectEffectIDs(t, "AB_CHEAL imported", skillEffectIDs(db.SkillABCheal), effects.Heal2)
	expectEffectIDs(t, "AB_HIGHNESSHEAL imported", skillEffectIDs(db.SkillABHighnessheal), effects.Heal4)
	expectEffectIDs(t, "AB_HIGHNESSHEAL imported hit", skillHitEffectIDs(db.SkillABHighnessheal), effects.HealOffensive)
}

func TestRobrowserMiniSTREffectSpecs(t *testing.T) {
	for _, tc := range []struct {
		name string
		id   int
		file string
		min  string
	}{
		{"Mammonite", effects.Mammonite, "maemor", "memor_min"},
		{"Angelus", effects.Angelus, "angelus", "jong_mini"},
		{"Cure", effects.Cure, "cure", "cure_min"},
		{"Gloria", effects.Gloria, "gloria", "gloria_min"},
		{"Magnificat", effects.Magnificat, "magnificat", "magnificat_min"},
		{"Resurrection", effects.Resurrection, "resurrection", "resurrection_min"},
		{"Lex Aeterna", effects.LexAeterna, "lexaeterna", "lexaeterna_min"},
		{"Suffragium", effects.Suffragium, "suffragium", "suffragium_min"},
		{"Storm Gust", effects.StormGust, "stormgust", "storm_min"},
		{"Weapon Perfection", effects.WeaponPerfect, "weaponperfection", "weaponperfection_min"},
		{"Maximize Power", effects.MaximizePower, "maximizepower", "maximize_min"},
		{"Kyrie Eleison", effects.Kyrie, "kyrie", "kyrie_min"},
		{"Christmas Carol", effects.ChristmasCarol, "angelus", "jong_mini"},
	} {
		spec, ok := effects.Lookup(tc.id)
		if !ok || len(spec.Components) == 0 {
			t.Fatalf("%s spec = %+v ok=%t, want a STR component", tc.name, spec, ok)
		}
		component := spec.Components[0]
		if component.Kind != effects.ComponentSTR || component.StrFile != tc.file || component.StrMinFile != tc.min || !component.AttachedEntity {
			t.Fatalf("%s component = %+v, want STR %q min %q attached", tc.name, component, tc.file, tc.min)
		}
	}
}

func TestMVPEffectSpecMatchesRoBrowserSTR(t *testing.T) {
	spec, ok := effects.Lookup(effects.Mvp)
	if !ok {
		t.Fatal("MVP effect spec missing")
	}
	if len(spec.Components) != 1 {
		t.Fatalf("MVP components = %d, want 1", len(spec.Components))
	}
	component := spec.Components[0]
	if component.Kind != effects.ComponentSTR || component.StrFile != "mvp" || !component.AttachedEntity {
		t.Fatalf("MVP component = %+v, want attached STR mvp", component)
	}
	if len(spec.Sfx) != 1 || spec.Sfx[0] != "effect\\st_mvp.wav" {
		t.Fatalf("MVP sfx = %#v", spec.Sfx)
	}
}

func TestImportedSkillActionFallback(t *testing.T) {
	archer := worldstate.Actor{Job: 3}
	if action := skillAction(db.SkillACDouble).actionFamilyForActorWithResources(nil, archer); action != attackActionFamilyForActorWithResources(nil, archer) {
		t.Fatalf("AC_DOUBLE action = %d, want normal weapon attack", action)
	}
	shower := skillAction(db.SkillACShower)
	if action := shower.actionFamilyForActorWithResources(nil, archer); action != attackActionFamilyForActorWithResources(nil, archer) {
		t.Fatalf("AC_SHOWER action = %d, want normal attack family", action)
	}
	if shower.speed != 0 {
		t.Fatalf("AC_SHOWER must use the supplied attack duration: %+v", shower)
	}
	merchant := worldstate.Actor{Job: 5}
	if action := skillAction(db.SkillMCCartrevolution).actionFamilyForActorWithResources(nil, merchant); action != sprites.ActionPCSkill {
		t.Fatalf("MC_CARTREVOLUTION action = %d, want SKILL", action)
	}
	counter := skillAction(db.SkillKNAutocounter)
	if !counter.defined || counter.action != skillActorActionAttack || !counter.hasFrame || counter.frame != 0 || counter.play || counter.next != nil {
		t.Fatalf("KN_AUTOCOUNTER action = %+v, want robr attack frame 0 with play=false next=false", counter)
	}
	relax := skillAction(db.SkillLKTensionrelax)
	if !relax.defined || relax.action != skillActorActionNone {
		t.Fatalf("LK_TENSIONRELAX action = %+v, want robr false action", relax)
	}
	trap := skillAction(db.SkillHTLandmine)
	if !trap.defined || trap.action != skillActorActionPickup || !trap.play || trap.repeat || trap.next == nil || trap.next.action != skillActorActionIdle {
		t.Fatalf("HT_LANDMINE action = %+v, want robr PICKUP then IDLE", trap)
	}
	sight := skillAction(db.SkillSNSight)
	if !sight.defined || sight.action != skillActorActionIdle || !sight.play || !sight.repeat || sight.next != nil {
		t.Fatalf("SN_SIGHT action = %+v, want native standing pose", sight)
	}
	for _, skillID := range []uint16{db.SkillDCWinkcharm, db.SkillBDRokisweil} {
		action := skillAction(skillID)
		if !action.defined || action.action != skillActorActionSkill || !action.play || !action.repeat || action.next != nil || !action.hasFrame || action.frame != 1 || action.length != 3 || action.speed != 10*db.EffectFrameDuration {
			t.Fatalf("dance/play action for skill %d = %+v, want native repeating SKILL frames 1–3", skillID, action)
		}
	}
	sonic := skillAction(db.SkillASSonicblow)
	hits := 0
	sawReadyFight := false
	for spec := &sonic; spec != nil; spec = spec.next {
		if spec.action == skillActorActionReadyFight {
			if !spec.repeat || !spec.play || spec.next != nil {
				t.Fatalf("AS_SONICBLOW ready fight tail = %+v, want robr READYFIGHT loop", spec)
			}
			sawReadyFight = true
			break
		}
		if spec.action != skillActorActionAttack || spec.repeat || !spec.play {
			t.Fatalf("AS_SONICBLOW chain node %d = %+v, want robr ATTACK hit", hits+1, spec)
		}
		hits++
		if hits == 1 {
			if spec.speed != 0 {
				t.Fatalf("AS_SONICBLOW first hit speed = %s, want default", spec.speed)
			}
			continue
		}
		if spec.speed != 30*time.Millisecond {
			t.Fatalf("AS_SONICBLOW hit %d speed = %s, want 30ms", hits, spec.speed)
		}
	}
	if hits != 8 {
		t.Fatalf("AS_SONICBLOW hits = %d, want robr 8-hit chain", hits)
	}
	if !sawReadyFight {
		t.Fatal("AS_SONICBLOW chain missing robr READYFIGHT tail")
	}
}

func TestHunterStringKeyEffectsMatchRobrowser(t *testing.T) {
	cast, ok := effects.Lookup(effects.SharpShootingCast)
	if !ok || cast.Duration != 10*time.Second || len(cast.Components) != 1 {
		t.Fatalf("496_beforecast spec = %+v ok=%t, want one 10s CastRing component", cast, ok)
	}
	component := cast.Components[0]
	if component.Kind != effects.ComponentFUNC || component.FuncName != "CastRing" || component.FuncAdapter != effects.EffectFuncCastRing || component.TextureName != "ring_jadu" {
		t.Fatalf("496_beforecast component identity = %+v", component)
	}
	if component.BottomSize != 0.8 || component.TopSize != 2.45 || component.Height != 2.8 || component.PosZ != 0.08 {
		t.Fatalf("496_beforecast component shape = %+v", component)
	}
	if component.TotalCircleSides != 20 || component.CircleSides != 20 || component.AlphaMax != 0.9 || !component.Fade || !component.Rotate || !component.AttachedEntity {
		t.Fatalf("496_beforecast component flags = %+v", component)
	}
}

func TestBlacksmithStringKeyEffectsMatchRobrowser(t *testing.T) {
	adrenaline, ok := effects.Lookup(effects.AdrenalineCast)
	if !ok || adrenaline.Duration != 500*time.Millisecond || len(adrenaline.Components) != 0 || len(adrenaline.Sfx) != 1 || adrenaline.Sfx[0] != "effect\\black_adrenalinerush_a.wav" {
		t.Fatalf("98_beforecast spec = %+v ok=%t, want robr adrenaline pre-cast sound", adrenaline, ok)
	}

	maximize, ok := effects.Lookup(effects.MaximizeSounds)
	if !ok || maximize.Duration != 950*time.Millisecond || len(maximize.Components) != 0 {
		t.Fatalf("maximize_power_sounds spec = %+v ok=%t, want robr delayed sound row", maximize, ok)
	}
	wantSFX := []string{
		"effect\\black_maximize_power_circle.wav",
		"effect\\black_maximize_power_sword.wav",
		"effect\\black_maximize_power_sword.wav",
		"effect\\black_maximize_power_sword_bic.wav",
	}
	wantDelays := []time.Duration{time.Millisecond, 550 * time.Millisecond, 700 * time.Millisecond, 950 * time.Millisecond}
	if !reflect.DeepEqual(maximize.Sfx, wantSFX) || !reflect.DeepEqual(maximize.SfxDelays, wantDelays) {
		t.Fatalf("maximize_power_sounds sfx = %#v delays %#v", maximize.Sfx, maximize.SfxDelays)
	}

	greed, ok := effects.Lookup(effects.GreedSound)
	if !ok || greed.Duration != 500*time.Millisecond || len(greed.Components) != 0 || len(greed.Sfx) != 1 || greed.Sfx[0] != "effect\\ef_entry.wav" {
		t.Fatalf("ef_greed_sound spec = %+v ok=%t, want robr greed sound", greed, ok)
	}
}

func TestKnightStringKeyEffectsMatchRobrowser(t *testing.T) {
	white, ok := effects.Lookup(effects.WhitePulse)
	if !ok || white.Duration != 500*time.Millisecond || len(white.Components) != 0 || len(white.Sfx) != 0 {
		t.Fatalf("white_pulse spec = %+v ok=%t, want robr no-draw 500ms row", white, ok)
	}

	for _, tc := range []struct {
		name string
		id   int
		wav  string
	}{
		{"spear_hit_sound", effects.SpearHitSound, "_hit_spear.wav"},
		{"enemy_hit_normal1", effects.EnemyHitNormal1, "_enemy_hit_normal1.wav"},
	} {
		spec, ok := effects.Lookup(tc.id)
		if !ok || spec.Duration != 500*time.Millisecond || len(spec.Components) != 0 || len(spec.Sfx) != 1 || spec.Sfx[0] != tc.wav {
			t.Fatalf("%s spec = %+v ok=%t, want sound %q", tc.name, spec, ok, tc.wav)
		}
	}

	quake, ok := effects.Lookup(effects.Quake)
	if !ok || quake.Duration != 650*time.Millisecond || quake.CameraShake != 650*time.Millisecond || len(quake.Components) != 1 {
		t.Fatalf("quake spec = %+v ok=%t, want 650ms CameraQuake", quake, ok)
	}
	quakeComponent := quake.Components[0]
	if quakeComponent.Kind != effects.ComponentFUNC || quakeComponent.FuncName != "CameraQuake" || quakeComponent.Duration != 650*time.Millisecond || !quakeComponent.AttachedEntity {
		t.Fatalf("quake component = %+v", quakeComponent)
	}
}

func TestGospelGroundHasTileAndFloatingCross(t *testing.T) {
	gospel, ok := effects.Lookup(effects.GospelGround)
	if !ok || gospel.Duration != 1500*time.Millisecond || len(gospel.Components) != 2 {
		t.Fatalf("370_ground spec = %+v ok=%t, want two song ground components", gospel, ok)
	}
	tile, cross := gospel.Components[0], gospel.Components[1]
	if tile.Kind != effects.ComponentFUNC || tile.FuncName != "FlatColorTile" || tile.FuncAdapter != effects.EffectFuncFlatColorTile || tile.Color != (color.RGBA{R: 255, G: 255, B: 255, A: 13}) || tile.SizeStart != 1 || !tile.RenderBefore || tile.AttachedEntity {
		t.Fatalf("370_ground tile = %+v", tile)
	}
	if cross.Kind != effects.ComponentFUNC || cross.FuncAdapter != effects.EffectFuncGospelCross || cross.TextureFile != "effect/cross_old.bmp" || !cross.Repeat || cross.RenderBefore {
		t.Fatalf("370_ground cross = %+v", cross)
	}
}

func TestArcherThiefMerchantSkillEffectMappings(t *testing.T) {
	expectEffectIDs(t, "AC_CONCENTRATION", skillSuccessEffectSelfIDs(45), effects.Concentration)
	expectEffectIDs(t, "AC_DOUBLE begin", skillBeginEffectIDs(46), effects.BashBegin)
	expectEffectIDs(t, "AC_DOUBLE before-hit", skillBeforeHitEffectIDs(46), effects.ArrowShot)
	expectEffectIDs(t, "AC_DOUBLE hit", skillHitEffectIDs(46), effects.Hit1, effects.BashHit)
	expectEffectIDs(t, "AC_SHOWER", skillEffectIDs(47), effects.ArrowShower)
	expectEffectIDs(t, "AC_SHOWER hit", skillHitEffectIDs(47), effects.Hit1, effects.BashHit)
	expectEffectIDs(t, "AC_CHARGEARROW hit", skillHitEffectIDs(148), effects.Hit1)
	if skillHidesCastAura(db.SkillACChargearrow) {
		t.Fatal("AC_CHARGEARROW must show the native casting flash")
	}
	expectEffectIDs(t, "TF_DOUBLE passive", skillEffectIDs(48))
	expectEffectIDs(t, "TF_MISS passive", skillEffectIDs(49))
	expectEffectIDs(t, "TF_STEAL success", skillSuccessEffectIDs(50), effects.Steal)
	expectEffectIDs(t, "TF_HIDING", skillEffectIDs(51))
	expectEffectIDs(t, "TF_POISON hit", skillHitEffectIDs(52), effects.BashHit)
	expectEffectIDs(t, "TF_DETOXIFY", skillEffectIDs(53), effects.Detoxication)
	expectEffectIDs(t, "TF_SPRINKLESAND before-hit", skillBeforeHitEffectIDs(149), effects.SprinkleSand)
	expectEffectIDs(t, "TF_BACKSLIDING", skillEffectIDs(150))
	expectEffectIDs(t, "TF_PICKSTONE", skillEffectIDs(151))
	if !skillHidesCastAura(db.SkillTFPickstone) {
		t.Fatal("TF_PICKSTONE should hide cast aura like roBrowser")
	}
	expectEffectIDs(t, "TF_THROWSTONE before-hit", skillBeforeHitEffectIDs(152), effects.ThrowItem3)
	expectEffectIDs(t, "MC_MAMMONITE hit", skillHitEffectIDs(42), effects.Hit1, effects.Mammonite)
	expectEffectIDs(t, "MC_CARTREVOLUTION before hit", skillBeforeHitEffectSelfIDs(153), effects.CartRevolution)
	expectEffectIDs(t, "MC_CARTREVOLUTION hit", skillHitEffectIDs(153), effects.Hit1, effects.CartRevolution)
	expectEffectIDs(t, "MC_LOUD", skillSuccessEffectSelfIDs(155), effects.Loud)
	expectEffectIDs(t, "AL_HOLYLIGHT", skillHitEffectIDs(156), effects.Hit1, effects.HolyLight)
}

func TestArcherProjectileEffects(t *testing.T) {
	shot, ok := effects.Lookup(effects.ArrowShot)
	if !ok || len(shot.Components) != 1 {
		t.Fatalf("arrow shot spec = %#v ok=%t, want one component", shot, ok)
	}
	component := shot.Components[0]
	if component.Kind != effects.Component3D || component.SpriteFile != "data/sprite/npc/skel_archer_arrow" {
		t.Fatalf("arrow shot component = %#v, want skel_archer_arrow 3D sprite", component)
	}
	if !component.ToSrc || !component.RotateToTarget || !component.RotateWithCamera || component.Duration != 140*time.Millisecond {
		t.Fatalf("arrow shot robr flags = %#v", component)
	}

	shower, ok := effects.Lookup(effects.ArrowShower)
	if !ok || len(shower.Components) != 1 {
		t.Fatalf("arrow shower spec = %#v ok=%t, want one component", shower, ok)
	}
	component = shower.Components[0]
	if component.FuncAdapter != effects.EffectFuncArrowShower || shower.Duration != scene.ReferenceBowArrowFlightDuration {
		t.Fatalf("arrow shower must use native flight and spread: %#v", component)
	}
}

func TestBowNormalAttackAddsArrowProjectileEffect(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{X: 10, Y: 20}
	world.Actors[300] = worldstate.Actor{ID: 300, X: 15, Y: 20, Job: 1002, ObjectType: scene.ActorObjectTypeMob, HasObjectType: true}
	ctx := client.Context{
		Session: &session.Session{
			AccountID: 200,
			Selected:  session.Character{ID: 200, Job: 3, Weapon: 11},
		},
		World: world,
	}
	mode := &WorldMode{}

	mode.applyActorActionNotify(ctx, network.ActorActionNotify{
		SourceID:    200,
		TargetID:    300,
		Damage:      12,
		HitCount:    1,
		Action:      0,
		SourceSpeed: 500,
		TargetSpeed: 500,
	})

	if len(mode.effects().Active) != 2 {
		t.Fatalf("world effects = %d, want arrow projectile and regular hit", len(mode.effects().Active))
	}
	projectile := mode.effects().Active[0]
	if projectile.EffectID != effects.ArrowShot || projectile.ActorID != 300 || projectile.TargetID != 200 {
		t.Fatalf("normal bow projectile = %+v", projectile)
	}
	if projectile.Duration != scene.ReferenceBowArrowFlightDuration || projectile.Expires.Sub(projectile.Starts) != scene.ReferenceBowArrowFlightDuration {
		t.Fatalf("normal bow projectile flight = duration %s lifetime %s, want %s", projectile.Duration, projectile.Expires.Sub(projectile.Starts), scene.ReferenceBowArrowFlightDuration)
	}
	hit := mode.effects().Active[1]
	if hit.EffectID != effects.Hit1 || hit.ActorID != 300 || !hit.Starts.Equal(projectile.Expires) {
		t.Fatalf("normal bow hit = %+v projectile=%+v", hit, projectile)
	}
}

func TestArcherMercenaryNormalAttackAddsArrowProjectileEffect(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20}
	world.UpsertActor(worldstate.Actor{
		ID:            400,
		X:             10,
		Y:             20,
		Job:           6017,
		ObjectType:    scene.ActorObjectTypeMercenary,
		HasObjectType: true,
	})
	world.UpsertActor(worldstate.Actor{
		ID:            300,
		X:             15,
		Y:             20,
		Job:           1002,
		ObjectType:    scene.ActorObjectTypeMob,
		HasObjectType: true,
	})
	mode := &WorldMode{}
	ctx := client.Context{Session: &session.Session{AccountID: 2000000}, World: world}

	mode.applyActorActionNotify(ctx, network.ActorActionNotify{
		SourceID:    400,
		TargetID:    300,
		Damage:      12,
		HitCount:    1,
		Action:      0,
		SourceSpeed: 500,
		TargetSpeed: 500,
	})

	if len(mode.effects().Active) != 2 {
		t.Fatalf("world effects = %d, want arrow projectile and regular hit", len(mode.effects().Active))
	}
	projectile := mode.effects().Active[0]
	if projectile.EffectID != effects.ArrowShot || projectile.ActorID != 300 || projectile.TargetID != 400 {
		t.Fatalf("mercenary arrow projectile = %+v", projectile)
	}
	hit := mode.effects().Active[1]
	if hit.EffectID != effects.Hit1 || hit.ActorID != 300 || !hit.Starts.After(projectile.Starts) {
		t.Fatalf("mercenary bow hit = %+v projectile=%+v", hit, projectile)
	}
}

func TestWarpEffectMappings(t *testing.T) {
	expectEffectIDs(t, "AL_TELEPORT begin", skillBeginEffectIDs(26))
	expectEffectIDs(t, "Butterfly Wing item", itemUseEffectSpec(602).effectIDs, effects.Teleportation)
	expectEffectIDs(t, "Fly Wing item", itemUseEffectSpec(601).effectIDs)
}

func TestSpeedPotionItemEffectMappingsMatchRobrowser(t *testing.T) {
	expectEffectIDs(t, "Concentration Potion item", itemUseEffectSpec(645).effectIDs, effects.ItemFast)
	expectEffectIDs(t, "Awakening Potion item", itemUseEffectSpec(656).effectIDs, effects.ItemFast2)
	expectEffectIDs(t, "Berserk Potion item", itemUseEffectSpec(657).effectIDs, effects.ItemFast3)
}

func TestTeleportModalRules(t *testing.T) {
	lv1 := session.Skill{ID: 26, Level: 1, Type: skillTargetEnemy, Range: 9}
	lv2 := session.Skill{ID: 26, Level: 2, Type: skillTargetEnemy, Range: 9}
	if !gameui.TeleportWarpListBypassesModal(lv1, network.WarpPointList{SkillID: 26, MapNames: []string{"Random"}}) {
		t.Fatal("Teleport level 1 should bypass the modal")
	}
	if gameui.TeleportWarpListBypassesModal(lv2, network.WarpPointList{SkillID: 26, MapNames: []string{"Random", "prontera"}}) {
		t.Fatal("Teleport level 2 with a save point should show the modal")
	}
}

func TestWarpPortalListOpensDestinationModal(t *testing.T) {
	mode := &WorldMode{}
	ctx := client.Context{Session: &session.Session{Skills: session.Skills{List: []session.Skill{
		{ID: 27, Level: 4, Type: skillTargetPlace, Range: 9},
	}}}}
	mode.applyWarpPointList(ctx, network.WarpPointList{SkillID: 27, MapNames: []string{"prontera", "geffen", "payon"}})
	if !mode.ui.teleportModal.IsOpen() {
		t.Fatal("warp portal list should open the destination modal")
	}
	if mode.ui.teleportModal.Title() != "Warp Portal" {
		t.Fatalf("modal title = %q", mode.ui.teleportModal.Title())
	}
}

func TestSkillUnitEffectMappings(t *testing.T) {
	expectEffectIDs(t, "UNT_SAFETYWALL", skillUnitEffectIDs(126), effects.SafetyWall)
	expectEffectIDs(t, "UNT_FIREWALL", skillUnitEffectIDs(127), effects.FireWall)
	expectEffectIDs(t, "UNT_WARPPORTAL", skillUnitEffectIDs(128), effects.Portal)
	expectEffectIDs(t, "UNT_PRE_WARPPORTAL", skillUnitEffectIDs(129), effects.ReadyPortal)
	expectEffectIDs(t, "UNT_SANCTUARY", skillUnitEffectIDs(131), effects.BottomSanc)
	expectEffectIDs(t, "UNT_MAGNUS", skillUnitEffectIDs(132), effects.BottomMagnus)
	expectEffectIDs(t, "UNT_PNEUMA", skillUnitEffectIDs(133), effects.Pneuma)
	expectEffectIDs(t, "UNT_FIREPILLAR_WAITING", skillUnitEffectIDs(135), effects.FirePillarOn)
	expectEffectIDs(t, "UNT_ICEWALL", skillUnitEffectIDs(141), effects.IceWall)
	expectEffectIDs(t, "UNT_QUAGMIRE", skillUnitEffectIDs(142), effects.Quagmire)
	expectEffectIDs(t, "UNT_VENOMDUST", skillUnitEffectIDs(146), effects.VenomDust2)
	expectEffectIDs(t, "UNT_VOLCANO", skillUnitEffectIDs(154), effects.BottomVolcano)
	expectEffectIDs(t, "UNT_DELUGE", skillUnitEffectIDs(155), effects.BottomDeluge)
	expectEffectIDs(t, "UNT_VIOLENTGALE", skillUnitEffectIDs(156), effects.BottomViolent)
	expectEffectIDs(t, "UNT_LANDPROTECTOR", skillUnitEffectIDs(157), effects.BottomLand)
	expectEffectIDs(t, "UNT_LULLABY", skillUnitEffectIDs(158), effects.BottomLullabyGround)
	expectEffectIDs(t, "UNT_RICHMANKIM", skillUnitEffectIDs(159), effects.BottomRichKimGround)
	expectEffectIDs(t, "UNT_ETERNALCHAOS", skillUnitEffectIDs(160), effects.BottomChaosGround)
	expectEffectIDs(t, "UNT_DRUMBATTLEFIELD", skillUnitEffectIDs(161), effects.BottomDrumGround)
	expectEffectIDs(t, "UNT_RINGNIBELUNGEN", skillUnitEffectIDs(162), effects.BottomNibelungGround)
	expectEffectIDs(t, "UNT_ROKISWEIL", skillUnitEffectIDs(163), effects.BottomRokiGround)
	expectEffectIDs(t, "UNT_INTOABYSS", skillUnitEffectIDs(164), effects.BottomAbyssGround)
	expectEffectIDs(t, "UNT_SIEGFRIED", skillUnitEffectIDs(165), effects.BottomSiegGround)
	expectEffectIDs(t, "UNT_DISSONANCE", skillUnitEffectIDs(166), effects.BottomDissonanceGround)
	expectEffectIDs(t, "UNT_WHISTLE", skillUnitEffectIDs(167), effects.BottomWhistleGround)
	expectEffectIDs(t, "UNT_ASSASSINCROSS", skillUnitEffectIDs(168), effects.BottomSinXGround)
	expectEffectIDs(t, "UNT_POEMBRAGI", skillUnitEffectIDs(169), effects.BottomBragiGround)
	expectEffectIDs(t, "UNT_APPLEIDUN", skillUnitEffectIDs(170), effects.BottomAppleGround)
	expectEffectIDs(t, "UNT_UGLYDANCE", skillUnitEffectIDs(171), effects.BottomUglyDanceGround)
	expectEffectIDs(t, "UNT_HUMMING", skillUnitEffectIDs(172), effects.BottomHummingGround)
	expectEffectIDs(t, "UNT_DONTFORGETME", skillUnitEffectIDs(173), effects.BottomForgetGround)
	expectEffectIDs(t, "UNT_FORTUNEKISS", skillUnitEffectIDs(174), effects.BottomFortuneGround)
	expectEffectIDs(t, "UNT_SERVICEFORYOU", skillUnitEffectIDs(175), effects.BottomServiceGround)
	expectEffectIDs(t, "UNT_DEMONSTRATION", skillUnitEffectIDs(177), effects.Demonstration)
	expectEffectIDs(t, "UNT_GOSPEL", skillUnitEffectIDs(179), effects.GospelGround)
	expectEffectIDs(t, "UNT_BASILICA", skillUnitEffectIDs(180), effects.BottomBasilica)
	expectEffectIDs(t, "UNT_MOONLIT", skillUnitEffectIDs(181))
	expectEffectIDs(t, "UNT_FOGWALL", skillUnitEffectIDs(182), effects.FogWallGround)
	expectEffectIDs(t, "UNT_SPIDERWEB", skillUnitEffectIDs(183), effects.BottomSpider)
	expectEffectIDs(t, "UNT_GRAVITATION", skillUnitEffectIDs(184), effects.Gravitation)
	expectEffectIDs(t, "UNT_HERMODE", skillUnitEffectIDs(185), effects.BottomHermode)
	expectEffectIDs(t, "UNT_TATAMIGAESHI", skillUnitEffectIDs(188), effects.Tatami)
	expectEffectIDs(t, "UNT_KAEN", skillUnitEffectIDs(189), effects.Kaen)
	expectEffectIDs(t, "UNT_EVILLAND", skillUnitEffectIDs(199), effects.BottomEvilLand)
	expectEffectIDs(t, "UNT_EPICLESIS", skillUnitEffectIDs(202), effects.GlassWall3)
	expectEffectIDs(t, "UNT_EARTHSTRAIN", skillUnitEffectIDs(203), effects.EarthWall)
	expectEffectIDs(t, "UNT_MANHOLE", skillUnitEffectIDs(204), effects.BottomManhole)
	expectEffectIDs(t, "UNT_DIMENSIONDOOR", skillUnitEffectIDs(205), effects.ForestLight6)
	expectEffectIDs(t, "UNT_CHAOSPANIC", skillUnitEffectIDs(206), effects.BottomAni)
	expectEffectIDs(t, "UNT_MAELSTROM", skillUnitEffectIDs(207), effects.BottomMaelstrom)
	expectEffectIDs(t, "UNT_BLOODYLUST", skillUnitEffectIDs(208), effects.BottomBloodyLust)
	expectEffectIDs(t, "UNT_REVERBERATION", skillUnitEffectIDs(218), effects.BotReverb)
	expectEffectIDs(t, "UNT_FIREWALK", skillUnitEffectIDs(220), effects.FireWall2)
	expectEffectIDs(t, "UNT_ELECTRICWALK", skillUnitEffectIDs(221), effects.Shockwave2)
	expectEffectIDs(t, "UNT_NETHERWORLD", skillUnitEffectIDs(222), effects.BotReverb2)
}

func TestQuakeMagnumEffectStartsCameraShake(t *testing.T) {
	spec, ok := effects.Lookup(effects.QuakeMagnum)
	if !ok {
		t.Fatal("quake magnum effect spec missing")
	}
	if spec.CameraShake != 50*time.Millisecond || len(spec.Components) != 0 {
		t.Fatalf("quake magnum spec = %+v, want no-draw 50ms camera shake", spec)
	}
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20}
	starts := time.Unix(100, 0)
	mode := &WorldMode{}
	ctx := client.Context{Session: &session.Session{AccountID: 2000000}, World: world}

	if !mode.effects().AddAt(ctx, effects.QuakeMagnum, 2000000, starts) {
		t.Fatal("add quake magnum effect failed")
	}
	if !mode.cameraShakeStart.Equal(starts) || !mode.cameraShakeEnd.Equal(starts.Add(50*time.Millisecond)) {
		t.Fatalf("camera shake = %s..%s", mode.cameraShakeStart, mode.cameraShakeEnd)
	}
	if x, y := mode.cameraShakeOffset(starts.Add(10 * time.Millisecond)); x == 0 && y == 0 {
		t.Fatalf("camera shake offset = %.3f, %.3f, want non-zero", x, y)
	}
	if x, y := mode.cameraShakeOffset(starts.Add(60 * time.Millisecond)); x != 0 || y != 0 {
		t.Fatalf("expired camera shake offset = %.3f, %.3f, want zero", x, y)
	}
}

func TestSpeedPotionEffectSpecsMatchRobrowserSTRRows(t *testing.T) {
	for _, tc := range []struct {
		name     string
		effectID int
		file     string
	}{
		{"Concentration Potion", effects.ItemFast, "집중"},
		{"Awakening Potion", effects.ItemFast2, "각성"},
	} {
		spec, ok := effects.Lookup(tc.effectID)
		if !ok || len(spec.Components) != 1 {
			t.Fatalf("%s spec = %+v ok=%t, want one STR component", tc.name, spec, ok)
		}
		component := spec.Components[0]
		if component.Kind != effects.ComponentSTR || component.StrFile != tc.file || !component.AttachedEntity {
			t.Fatalf("%s component = %+v, want attached STR %q", tc.name, component, tc.file)
		}
		if len(spec.Sfx) != 1 || spec.Sfx[0] != "effect\\ac_concentration.wav" {
			t.Fatalf("%s sfx = %v, want ac_concentration", tc.name, spec.Sfx)
		}
	}
}

func TestBerserkPotionEffectStartsDelayedCameraShake(t *testing.T) {
	spec, ok := effects.Lookup(effects.ItemFast3)
	if !ok {
		t.Fatal("berserk potion effect spec missing")
	}
	if spec.CameraShake != 200*time.Millisecond || spec.CameraShakeDelay != 18*db.EffectFrameDuration {
		t.Fatalf("berserk potion shake = delay %s duration %s, want 18 ticks/200ms", spec.CameraShakeDelay, spec.CameraShake)
	}
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20}
	starts := time.Unix(100, 0)
	mode := &WorldMode{}
	ctx := client.Context{Session: &session.Session{AccountID: 2000000}, World: world}

	if !mode.effects().AddAt(ctx, effects.ItemFast3, 2000000, starts) {
		t.Fatal("add berserk potion effect failed")
	}
	shakeStart := starts.Add(18 * db.EffectFrameDuration)
	shakeEnd := shakeStart.Add(200 * time.Millisecond)
	if !mode.cameraShakeStart.Equal(shakeStart) || !mode.cameraShakeEnd.Equal(shakeEnd) {
		t.Fatalf("camera shake = %s..%s, want %s..%s", mode.cameraShakeStart, mode.cameraShakeEnd, shakeStart, shakeEnd)
	}
	if x, y := mode.cameraShakeOffset(starts.Add(100 * time.Millisecond)); x != 0 || y != 0 {
		t.Fatalf("early camera shake offset = %.3f, %.3f, want zero", x, y)
	}
	if x, y := mode.cameraShakeOffset(shakeStart.Add(50 * time.Millisecond)); x == 0 && y == 0 {
		t.Fatalf("active camera shake offset = %.3f, %.3f, want non-zero", x, y)
	}
}

func TestTeleportationEffectSpecUsesRobrowserCylinderStack(t *testing.T) {
	spec, ok := effects.Lookup(effects.Teleportation)
	if !ok {
		t.Fatal("teleportation effect spec missing")
	}
	if spec.Duration != 1500*time.Millisecond {
		t.Fatalf("duration = %s, want 1500ms", spec.Duration)
	}
	if !spec.DetachLocalActor {
		t.Fatal("teleportation should detach from the local actor")
	}
	if len(spec.Sfx) != 1 || spec.Sfx[0] != "effect\\ef_teleportation.wav" {
		t.Fatalf("sfx = %#v", spec.Sfx)
	}
	if len(spec.Components) != 4 {
		t.Fatalf("components = %d, want 4", len(spec.Components))
	}
	expected := []struct {
		bottom float64
		top    float64
		height float64
	}{
		{0.3, 0.3, 35},
		{0.6, 0.8, 25},
		{0.8, 1.0, 13},
		{1.0, 1.3, 5},
	}
	for i, want := range expected {
		component := spec.Components[i]
		if component.Kind != effects.ComponentCylinder || component.TextureName != "ring_blue" || component.Duration != 1500*time.Millisecond || component.BlendMode != 2 || !component.BlendAdditive {
			t.Fatalf("component %d = %+v", i, component)
		}
		if component.BottomSize != want.bottom || component.TopSize != want.top || component.Height != want.height {
			t.Fatalf("component %d size = %.1f %.1f %.1f, want %.1f %.1f %.1f", i, component.BottomSize, component.TopSize, component.Height, want.bottom, want.top, want.height)
		}
		if component.FixedPerspective {
			t.Fatalf("component %d uses fixed perspective, want world-space cylinder", i)
		}
		if !component.AttachedEntity {
			t.Fatalf("component %d is not attached to the entity", i)
		}
	}
}

func TestWarpPortalEffectSpecUsesPortal2Cylinders(t *testing.T) {
	spec, ok := effects.Lookup(effects.Portal)
	if !ok {
		t.Fatal("portal effect spec missing")
	}
	if len(spec.Sfx) != 2 || spec.Sfx[0] != "effect\\ef_readyportal.wav" || spec.Sfx[1] != "effect\\ef_portal.wav" {
		t.Fatalf("portal sfx = %#v", spec.Sfx)
	}
	if len(spec.Components) != 4 {
		t.Fatalf("components = %d, want 4", len(spec.Components))
	}
	first := spec.Components[0]
	if first.Kind != effects.ComponentCylinder || first.TextureName != "ring_blue" || first.Duration != 500*time.Millisecond || first.Animation != 4 {
		t.Fatalf("first portal component = %+v", first)
	}
	if !first.Repeat || first.RepeatDelay != -300*time.Millisecond {
		t.Fatalf("first portal repeat = %t delay=%s, want reference client repeat -300ms", first.Repeat, first.RepeatDelay)
	}
	if spec.Components[3].TextureName != "alpha1" || spec.Components[3].PosZ != 2 || spec.Components[3].Height != 1 {
		t.Fatalf("portal cap component = %+v", spec.Components[3])
	}
}

func TestHealEffectSpecUsesRobrowserCylindersAndParticles(t *testing.T) {
	spec, ok := effects.Lookup(effects.Heal)
	if !ok {
		t.Fatal("heal effect spec missing")
	}
	if spec.Duration != 1840*time.Millisecond {
		t.Fatalf("duration = %s, want 1840ms", spec.Duration)
	}
	if len(spec.Sfx) != 1 || spec.Sfx[0] != "_heal_effect.wav" {
		t.Fatalf("sfx = %#v", spec.Sfx)
	}
	if len(spec.Components) != 4 {
		t.Fatalf("components = %d, want 4", len(spec.Components))
	}
	for i, component := range spec.Components[:2] {
		if component.Kind != effects.ComponentCylinder || component.TextureName != "ring_white" || component.Animation != 1 {
			t.Fatalf("component %d = %+v", i, component)
		}
		if component.Duration != 1500*time.Millisecond || component.Height != 8 || component.AlphaMax != 0.2 {
			t.Fatalf("component %d timing/shape = %+v", i, component)
		}
		if component.Color != (color.RGBA{R: 178, G: 255, B: 178, A: 255}) || !component.BlendAdditive {
			t.Fatalf("component %d tint/blend = %+v", i, component)
		}
	}
	firstParticle := spec.Components[2]
	if firstParticle.Kind != effects.Component3D || firstParticle.TextureFile != "effect/pok3.tga" {
		t.Fatalf("first heal particle = %+v", firstParticle)
	}
	if firstParticle.Duration != 1300*time.Millisecond || firstParticle.Delay != 400*time.Millisecond || firstParticle.DuplicateDelay != 10*time.Millisecond || firstParticle.Duplicate != 15 {
		t.Fatalf("first heal particle timing = %+v", firstParticle)
	}
	if firstParticle.AlphaMax != 0.6 || !firstParticle.FadeIn || !firstParticle.FadeOut || firstParticle.Sparkling || firstParticle.SparkNumber != 0 {
		t.Fatalf("first heal particle fade = %+v", firstParticle)
	}
	if firstParticle.PosXRand != 1.5 || firstParticle.PosYRand != 1.5 || firstParticle.PosZEndRand != 2 || firstParticle.PosZEndMiddle != 6 {
		t.Fatalf("first heal particle position = %+v", firstParticle)
	}
	if firstParticle.SizeStart != 9*scene.PixelRatio || firstParticle.SizeEnd != 9*scene.PixelRatio || firstParticle.SizeRand != 2*scene.PixelRatio {
		t.Fatalf("first heal particle size = %+v", firstParticle)
	}
	secondParticle := spec.Components[3]
	if secondParticle.Kind != effects.Component3D || secondParticle.TextureFile != "effect/pok3.tga" {
		t.Fatalf("second heal particle = %+v", secondParticle)
	}
	if secondParticle.Duration != 1100*time.Millisecond || secondParticle.Delay != 200*time.Millisecond || secondParticle.DuplicateDelay != 50*time.Millisecond || secondParticle.Duplicate != 7 {
		t.Fatalf("second heal particle timing = %+v", secondParticle)
	}
	if secondParticle.AlphaMax != 0.6 || !secondParticle.FadeIn || !secondParticle.FadeOut || secondParticle.Sparkling || secondParticle.SparkNumber != 0 {
		t.Fatalf("second heal particle fade = %+v", secondParticle)
	}
	if secondParticle.PosXRand != 1 || secondParticle.PosYRand != 1 || secondParticle.PosZEnd != 5 || secondParticle.PosZStartRand != 1 {
		t.Fatalf("second heal particle position = %+v", secondParticle)
	}
	if secondParticle.SizeStart != 9*scene.PixelRatio || secondParticle.SizeEnd != 9*scene.PixelRatio || secondParticle.SizeRand != 2*scene.PixelRatio {
		t.Fatalf("second heal particle size = %+v", secondParticle)
	}
}

func TestHealOffensiveEffectSpecUsesRobrowserCylindersAndParticles(t *testing.T) {
	spec, ok := effects.Lookup(effects.HealOffensive)
	if !ok {
		t.Fatal("offensive heal effect spec missing")
	}
	if spec.Duration != 1490*time.Millisecond {
		t.Fatalf("duration = %s, want 1490ms", spec.Duration)
	}
	if len(spec.Sfx) != 1 || spec.Sfx[0] != "_heal_effect.wav" {
		t.Fatalf("sfx = %#v", spec.Sfx)
	}
	if len(spec.Components) != 4 {
		t.Fatalf("components = %d, want 4", len(spec.Components))
	}
	for i, component := range spec.Components[:2] {
		if component.Kind != effects.ComponentCylinder || component.TextureName != "ring_white" || component.Animation != 1 {
			t.Fatalf("component %d = %+v", i, component)
		}
		if component.Duration != time.Second || !component.BlendAdditive || component.Color != (color.RGBA{R: 255, G: 255, B: 255, A: 255}) {
			t.Fatalf("component %d timing/tint = %+v", i, component)
		}
	}
	if spec.Components[0].Height != 10 || spec.Components[1].Height != 9 {
		t.Fatalf("cylinder heights = %.1f %.1f", spec.Components[0].Height, spec.Components[1].Height)
	}
	firstParticle := spec.Components[2]
	if firstParticle.Kind != effects.Component3D || firstParticle.TextureFile != "effect/pok3.tga" {
		t.Fatalf("first offensive heal particle = %+v", firstParticle)
	}
	if firstParticle.Duration != time.Second || firstParticle.Delay != 400*time.Millisecond || firstParticle.DuplicateDelay != 10*time.Millisecond || firstParticle.Duplicate != 10 {
		t.Fatalf("first offensive heal particle timing = %+v", firstParticle)
	}
	if firstParticle.AlphaMax != 0.8 || !firstParticle.FadeIn || !firstParticle.FadeOut || !firstParticle.BlendAdditive || !firstParticle.Sparkling || firstParticle.SparkNumber != 2 {
		t.Fatalf("first offensive heal particle fade/blend = %+v", firstParticle)
	}
	if firstParticle.PosXRand != 1.5 || firstParticle.PosYRand != 1.5 || firstParticle.PosZEndRand != 3 || firstParticle.PosZEndMiddle != 6 {
		t.Fatalf("first offensive heal particle position = %+v", firstParticle)
	}
	secondParticle := spec.Components[3]
	if secondParticle.Kind != effects.Component3D || secondParticle.TextureFile != "effect/pok3.tga" {
		t.Fatalf("second offensive heal particle = %+v", secondParticle)
	}
	if secondParticle.Duration != 900*time.Millisecond || secondParticle.Delay != 200*time.Millisecond || secondParticle.DuplicateDelay != 50*time.Millisecond || secondParticle.Duplicate != 5 {
		t.Fatalf("second offensive heal particle timing = %+v", secondParticle)
	}
	if secondParticle.AlphaMax != 0.8 || !secondParticle.FadeIn || !secondParticle.FadeOut || !secondParticle.BlendAdditive || !secondParticle.Sparkling || secondParticle.SparkNumber != 2 {
		t.Fatalf("second offensive heal particle fade/blend = %+v", secondParticle)
	}
	if secondParticle.PosXRand != 1 || secondParticle.PosYRand != 1 || secondParticle.PosZEnd != 6 || secondParticle.PosZStartRand != 1 {
		t.Fatalf("second offensive heal particle position = %+v", secondParticle)
	}
	if secondParticle.SizeStart != 9*scene.PixelRatio || secondParticle.SizeEnd != 9*scene.PixelRatio || secondParticle.SizeRand != 2*scene.PixelRatio {
		t.Fatalf("second offensive heal particle size = %+v", secondParticle)
	}
}

func TestAngelusEffectSpecUsesRobrowserSTR(t *testing.T) {
	spec, ok := effects.Lookup(effects.Angelus)
	if !ok {
		t.Fatal("angelus effect spec missing")
	}
	if len(spec.Sfx) != 1 || spec.Sfx[0] != "effect\\ef_angelus.wav" {
		t.Fatalf("sfx = %#v", spec.Sfx)
	}
	if len(spec.Components) != 1 {
		t.Fatalf("components = %d, want 1", len(spec.Components))
	}
	component := spec.Components[0]
	if component.Kind != effects.ComponentSTR || component.StrFile != "angelus" || component.StrMinFile != "jong_mini" {
		t.Fatalf("STR resource = %+v", component)
	}
	if !component.AttachedEntity || !component.SpriteHead {
		t.Fatalf("STR attachment flags = %+v", component)
	}
}

func TestEmotionEffectSpecUsesEntityAttachmentOffset(t *testing.T) {
	spec, ok := effects.Lookup(effects.Emotion)
	if !ok {
		t.Fatal("emotion effect spec missing")
	}
	if len(spec.Components) != 1 {
		t.Fatalf("components = %d, want 1", len(spec.Components))
	}
	component := spec.Components[0]
	if component.Kind != effects.ComponentSPR || component.SpriteFile != "emotion" {
		t.Fatalf("emotion component = %+v", component)
	}
	if !component.AttachedEntity || component.SpriteYOffset != -100 || component.SpriteHead {
		t.Fatalf("emotion placement = %+v", component)
	}
}

func TestSPREffectShowsEveryFrameBeforeExpiring(t *testing.T) {
	image := render.NewImage(4, 4)
	image.Fill(color.White)
	animation := res.ACTAnimation{Layers: []res.ACTLayer{{
		ScaleX: 1, ScaleY: 1, Color: [4]float32{1, 1, 1, 1},
	}}}
	act := &res.ACT{Actions: make([]res.ACTAction, 20)}
	act.Actions[0] = res.ACTAction{Animations: []res.ACTAnimation{animation}, DelayMS: 50}
	// Pupa's sweating emotion uses action 19: 24 frames at 50 ms each.
	act.Actions[19] = res.ACTAction{Animations: make([]res.ACTAnimation, 24), DelayMS: 50}
	for i := range act.Actions[19].Animations {
		act.Actions[19].Animations[i] = animation
	}
	view := &scene.SpriteView{
		Act:        act,
		Spr:        &res.SPR{Frames: []res.SPRFrame{{Width: 4, Height: 4}}},
		Images:     map[scene.SpriteFrameKey]*render.Image{{}: image},
		Billboards: make(map[scene.SpriteBillboardKey]*scene.Billboard),
	}
	mode := WorldMode{effectSystem: effects.System{EffectViews: map[string]*scene.SpriteView{"emotion": view}}}
	ctx := client.Context{Resources: &res.Manager{}}
	projection := testutil.Projection(800, 600, 0, 0, 0)
	starts := time.Unix(10, 0)
	for _, tc := range []struct {
		name                  string
		action, ms            int
		stop, repeat, visible bool
	}{
		{"first", 19, 0, false, false, true},
		{"last begins", 19, 1150, false, false, true},
		{"last ends", 19, 1199, false, false, true},
		{"expired", 19, 1200, false, false, false},
		{"single frame", 0, 0, false, false, true},
		{"single expired", 0, 50, false, false, false},
		{"hold last", 19, 1200, true, false, true},
		{"repeat", 19, 1200, false, true, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			screen := render.NewFrame(800, 600)
			effect := effects.Instance{Starts: starts, HasSpriteFrame: true, SpriteFrameOverride: tc.action}
			component := effects.Component{SpriteFile: "emotion", SpriteStopAtEnd: tc.stop, SpriteRepeat: tc.repeat}
			mode.effects().DrawSPREffect(screen, ctx, projection, effect, component, 0, 0, 0, starts.Add(time.Duration(tc.ms)*time.Millisecond))
			visible := !reflect.DeepEqual(screen, render.NewFrame(800, 600))
			if visible != tc.visible {
				t.Fatalf("visible=%t, want %t", visible, tc.visible)
			}
		})
	}
}

func TestWorldEffectDuplicateDeltasMatchRobrowserSemantics(t *testing.T) {
	component := effects.Component{
		AlphaMax:      0.2,
		AlphaMaxDelta: 0.2,
		SizeStart:     100 * scene.PixelRatio,
		SizeEnd:       100 * scene.PixelRatio,
		SizeDelta:     -10,
	}
	if got := effects.EffectBillboardAlphaForDuplicate(0.5, component, 2); math.Abs(got-0.6) > 0.001 {
		t.Fatalf("duplicate alpha = %.3f, want 0.6", got)
	}
	sizeX, sizeY := effects.Effect3DSize(component, effects.Instance{}, 0, 0.5, 2)
	want := 80 * scene.PixelRatio
	if math.Abs(sizeX-want) > 0.001 || math.Abs(sizeY-want) > 0.001 {
		t.Fatalf("duplicate size = %.3f x %.3f, want %.3f", sizeX, sizeY, want)
	}
}

func TestEffect3DSpriteScaleUsesRobrowserSpriteUnits(t *testing.T) {
	size := testutil.EffectTableSize(200)
	got := effects.Effect3DSpriteScale(size)
	want := size / 100
	if math.Abs(got-want) > 0.001 {
		t.Fatalf("sprite pixel scale = %.5f, want %.5f", got, want)
	}
	fireballWorldWidth := 64 * got
	wantWidth := 128 * scene.PixelRatio
	if math.Abs(fireballWorldWidth-wantWidth) > 0.001 {
		t.Fatalf("64px fireball width = %.3f, want robr 128/35 %.3f", fireballWorldWidth, wantWidth)
	}
}

func TestEffect3DSpriteDrawOptionsHonorAdditiveBlend(t *testing.T) {
	defaultOptions := effects.Effect3DSpriteDrawOptions(effects.Component{})
	if got := defaultOptions.Blend; got != render.BlendSourceOver {
		t.Fatalf("default sprite effect blend = %v, want source-over", got)
	}
	if got := defaultOptions.DepthBias; got != 0 {
		t.Fatalf("default sprite effect depth bias = %.3f, want 0", got)
	}
	if got := effects.Effect3DSpriteDrawOptions(effects.Component{BlendAdditive: true}).Blend; got != render.BlendLighter {
		t.Fatalf("additive sprite effect blend = %v, want lighter", got)
	}
	if got := effects.Effect3DSpriteDrawOptions(effects.Component{WorldSizedSprite: true}).DepthBias; got != effects.StrEffectDepthBias {
		t.Fatalf("world-sized sprite effect depth bias = %.3f, want %.3f", got, effects.StrEffectDepthBias)
	}
	if effects.Effect3DSpriteDrawOptions(effects.Component{Overlay: true}).DepthTest {
		t.Fatal("overlay sprite effect depth test enabled, want disabled")
	}
}

func TestTexturedEffectBillboardDrawOptionsHonorOverlay(t *testing.T) {
	defaultOptions := effects.TexturedEffectBillboardDrawOptions(false, false)
	if !defaultOptions.DepthTest {
		t.Fatal("default textured effect depth test disabled, want enabled")
	}
	if got := defaultOptions.Blend; got != render.BlendSourceOver {
		t.Fatalf("default textured effect blend = %v, want source-over", got)
	}
	additiveOptions := effects.TexturedEffectBillboardDrawOptions(true, false)
	if got := additiveOptions.Blend; got != render.BlendLighter {
		t.Fatalf("additive textured effect blend = %v, want lighter", got)
	}
	overlayOptions := effects.TexturedEffectBillboardDrawOptions(false, true)
	if overlayOptions.DepthTest {
		t.Fatal("overlay textured effect depth test enabled, want disabled")
	}
}

func TestWorldSizedSpriteBillboardUsesCenterDepthLikeRobrowser(t *testing.T) {
	options := render.DrawTrianglesOptions{DepthBias: 0.01}
	cmd := scene.WorldSpriteBillboardCommand(
		render.WhiteImage(),
		options,
		scene.Point3{X: 1, Y: 2, Z: 3},
		scene.Point3{X: 4, Y: 5, Z: 6},
		scene.Point3{X: 7, Y: 8, Z: 9},
		10,
		20,
		3,
		4,
		color.RGBA{R: 51, G: 102, B: 153, A: 204},
	)

	if cmd.UpAxis != [3]float32{7, 8, 9} {
		t.Fatalf("visible up axis = %+v, want rendered billboard up axis", cmd.UpAxis)
	}
	if cmd.DepthUpAxis != [3]float32{} {
		t.Fatalf("depth up axis = %+v, want center-depth axis", cmd.DepthUpAxis)
	}
	if cmd.DepthBias != options.DepthBias || cmd.Options.DepthBias != options.DepthBias {
		t.Fatalf("depth bias = command %.3f options %.3f, want %.3f", cmd.DepthBias, cmd.Options.DepthBias, options.DepthBias)
	}
}

func TestWorldEffectOrbitReplacesBasePositionLikeRobrowser(t *testing.T) {
	component := effects.Component{
		PosX:           -2,
		PosY:           4,
		OrbitRadiusX:   3,
		OrbitRadiusY:   3,
		OrbitRotations: 8,
		OrbitPhase:     0.7,
		OrbitClockwise: true,
	}
	x, y, _ := (&WorldMode{}).effects().Effect3DOffset(client.Context{}, component, effects.Instance{}, 0, 0, 0, 0, 0, 0)
	angle := -0.7 * math.Pi / 2
	wantX := -math.Cos(angle) * 3
	wantY := math.Sin(angle) * 3
	if math.Abs(x-wantX) > 0.001 || math.Abs(y-wantY) > 0.001 {
		t.Fatalf("orbit offset = %.3f, %.3f; want %.3f, %.3f", x, y, wantX, wantY)
	}
	if math.Abs(x-(-2+wantX)) < 0.001 || math.Abs(y-(4+wantY)) < 0.001 {
		t.Fatalf("orbit offset incorrectly included base position: %.3f, %.3f", x, y)
	}
}

func TestWorldEffectBillboardSparklingAlphaMatchesRobrowser(t *testing.T) {
	component := effects.Component{
		AlphaMax:    1,
		Sparkling:   true,
		SparkNumber: 2,
	}
	if got := effects.EffectBillboardAlphaForDuplicate(0, component, 0); math.Abs(got-1) > 0.001 {
		t.Fatalf("spark alpha at start = %.3f, want 1", got)
	}
	if got := effects.EffectBillboardAlphaForDuplicate(0.25, component, 0); math.Abs(got-0.008) > 0.001 {
		t.Fatalf("spark alpha at quarter = %.3f, want about 0.008", got)
	}
	if got := effects.EffectBillboardAlphaForDuplicate(0.75, component, 0); math.Abs(got-0.067) > 0.001 {
		t.Fatalf("spark alpha at three quarters = %.3f, want about 0.067", got)
	}
}

func TestWorldEffectBillboardAngleCanRotateWithCamera(t *testing.T) {
	projection := testutil.ProjectionYaw(800, 600, 0, 0, 0, 45)
	component := effects.Component{AngleStart: 90, AngleEnd: 180, RotateWithCamera: true}
	got := effects.WorldEffectBillboardAngleForEffect(component, projection, effects.Instance{}, 0, 0.5)
	want := scene.DegreesToRadians(180)
	if math.Abs(got-want) > 0.001 {
		t.Fatalf("angle = %.3f, want %.3f", got, want)
	}
}

func TestEffectCylinderAngleXRotatesHeightAxisLikeRobrowser(t *testing.T) {
	got := effects.RotateEffectCylinderVector(scene.Point3{Y: 1}, -90, 0, 0)
	want := scene.Point3{Z: -1}
	if !testutil.ModelPointNear(got, want, 0.001) {
		t.Fatalf("rotated height axis = %+v, want %+v", got, want)
	}
}

func TestWorldCylinderBandAllowsRobrowserNegativeHeight(t *testing.T) {
	screen := render.NewFrame(320, 240)
	effects.DrawWorldCylinderBandWithBasis(screen, render.WhiteImage(), render.WhiteImage(), 0, 0, 0, 1, 2, -4, color.RGBA{A: 255}, 8, scene.Point3{X: 1}, scene.Point3{Z: 1}, scene.Point3{Y: 1})
	commands := reflect.ValueOf(screen).Elem().FieldByName("worldCommands")
	if commands.Len() != 1 {
		t.Fatalf("world commands = %d, want one negative-height cylinder", commands.Len())
	}
	vertices := commands.Index(0).FieldByName("Vertices")
	if vertices.Len() < 2 {
		t.Fatalf("vertices = %d, want cylinder vertices", vertices.Len())
	}
	bottomY := vertices.Index(0).FieldByName("Y").Float()
	topY := vertices.Index(1).FieldByName("Y").Float()
	if topY >= bottomY {
		t.Fatalf("top Y = %.1f bottom Y = %.1f, want top below bottom", topY, bottomY)
	}
}

func TestSTRAnimationAttachedEntityUsesActorAnchor(t *testing.T) {
	anim := res.STRAnimation{Pos: [2]float32{320, 320}}

	_, groundY := effects.StrAnimationOffset(anim, false)
	_, attachedY := effects.StrAnimationOffset(anim, true)

	if groundY != -0.5 {
		t.Fatalf("ground STR y offset = %.3f, want -0.5", groundY)
	}
	if attachedY != 0 {
		t.Fatalf("attached STR y offset = %.3f, want 0", attachedY)
	}
}

func TestSTRAnimationLocalOffsetFlipsYBeforeRotationLikeRobrowser(t *testing.T) {
	anim := res.STRAnimation{
		Pos:   [2]float32{330, 340},
		Angle: 90,
	}
	gotX, gotY := effects.StrAnimationLocalOffset(anim, 10, 20, true)
	wantX, wantY := -10.0/35.0, -30.0/35.0
	if math.Abs(gotX-wantX) > 0.0001 || math.Abs(gotY-wantY) > 0.0001 {
		t.Fatalf("STR local offset = %.4f %.4f, want %.4f %.4f", gotX, gotY, wantX, wantY)
	}
}

func TestSTRAnimationBlendMatchesRobrowserD3DBlend(t *testing.T) {
	if got := effects.StrAnimationBlend(res.STRAnimation{SrcAlpha: 5, DestAlpha: 7}); got != render.BlendSrcAlphaDstAlpha {
		t.Fatalf("SRC_ALPHA/DST_ALPHA blend = %v, want BlendSrcAlphaDstAlpha", got)
	}
	if got := effects.StrAnimationBlend(res.STRAnimation{SrcAlpha: 5, DestAlpha: 2}); got != render.BlendLighter {
		t.Fatalf("SRC_ALPHA/ONE blend = %v, want BlendLighter", got)
	}
	if got := effects.StrAnimationBlend(res.STRAnimation{SrcAlpha: 5, DestAlpha: 6}); got != render.BlendSourceOver {
		t.Fatalf("regular STR blend = %v, want BlendSourceOver", got)
	}
}

func TestSTRAnimationDrawOptionsDisableFogToMatchRobrowser(t *testing.T) {
	options := effects.StrAnimationDrawOptions(res.STRAnimation{SrcAlpha: 5, DestAlpha: 6})
	if !options.DisableFog {
		t.Fatal("STR draw options enabled map fog, want disabled")
	}
}

func TestSTRAnimationDrawOptionsUseDepthBiasToMatchRobrowser(t *testing.T) {
	options := effects.StrAnimationDrawOptions(res.STRAnimation{SrcAlpha: 5, DestAlpha: 6})
	if options.DepthBias <= 0 {
		t.Fatalf("STR depth bias = %.3f, want positive robr-style camera bias", options.DepthBias)
	}
}

func TestSTRAnimationVertexUsesCenterDepthToMatchRobrowser(t *testing.T) {
	point := scene.Point3{X: 1, Y: 2, Z: 3}
	depthPoint := scene.Point3{X: 4, Y: 5, Z: 6}
	vertex := effects.StrAnimationVertex3D(point, scene.UV{U: 0.25, V: 0.5}, color.RGBA{R: 51, G: 102, B: 153, A: 204}, 80, 40, depthPoint)

	if vertex.X != 1 || vertex.Y != 2 || vertex.Z != 3 {
		t.Fatalf("vertex position = %.1f %.1f %.1f, want rendered point", vertex.X, vertex.Y, vertex.Z)
	}
	if vertex.DepthX != 4 || vertex.DepthY != 5 || vertex.DepthZ != 6 {
		t.Fatalf("vertex depth = %.1f %.1f %.1f, want STR center depth", vertex.DepthX, vertex.DepthY, vertex.DepthZ)
	}
	if vertex.SrcX != 20 || vertex.SrcY != 20 {
		t.Fatalf("vertex uv = %.1f %.1f, want texture pixel coords", vertex.SrcX, vertex.SrcY)
	}
}

func TestLevelUpEffectSpecsUseSTRResources(t *testing.T) {
	base, ok := effects.Lookup(effects.BaseLevelUp)
	if !ok {
		t.Fatal("base level-up effect spec missing")
	}
	if len(base.Components) != 1 || base.Components[0].Kind != effects.ComponentSTR || base.Components[0].StrFile != "angel" || !base.Components[0].AttachedEntity {
		t.Fatalf("base level-up spec = %+v", base)
	}
	if len(base.Sfx) != 1 || base.Sfx[0] != "levelup.wav" {
		t.Fatalf("base level-up sfx = %#v", base.Sfx)
	}
	job, ok := effects.Lookup(effects.JobLevelUp)
	if !ok {
		t.Fatal("job level-up effect spec missing")
	}
	if len(job.Components) != 1 || job.Components[0].Kind != effects.ComponentSTR || job.Components[0].StrFile != "joblvup" {
		t.Fatalf("job level-up spec = %+v", job)
	}
	if len(job.Sfx) != 0 {
		t.Fatalf("job level-up sfx = %#v", job.Sfx)
	}
}

func TestSpecialEffectNotifyAddsLevelUpEffects(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20}
	mode := &WorldMode{}
	ctx := client.Context{Session: &session.Session{AccountID: 2000000}, World: world}

	mode.applySpecialEffectNotify(ctx, network.SpecialEffectNotify{AID: 2000000, EffectID: network.SpecialEffectBaseLevelUp})
	mode.applySpecialEffectNotify(ctx, network.SpecialEffectNotify{AID: 2000000, EffectID: network.SpecialEffectJobLevelUp})

	if len(mode.effects().Active) != 2 {
		t.Fatalf("world effects = %d, want 2", len(mode.effects().Active))
	}
	if mode.effects().Active[0].EffectID != effects.BaseLevelUp || mode.effects().Active[1].EffectID != effects.JobLevelUp {
		t.Fatalf("world effects = %+v", mode.effects().Active)
	}
	if len(mode.scheduledSounds) != 1 {
		t.Fatalf("scheduled sounds = %+v", mode.scheduledSounds)
	}
	if mode.scheduledSounds[0].paths[0] != "levelup.wav" {
		t.Fatalf("base level-up sound = %+v", mode.scheduledSounds[0])
	}
	if !mode.ui.levelUpNotifications.BaseVisible() || !mode.ui.levelUpNotifications.JobVisible() {
		t.Fatal("local level-up effects did not show both availability notifications")
	}
}

func TestRemoteLevelUpEffectDoesNotShowAvailabilityNotification(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20}
	mode := &WorldMode{}
	ctx := client.Context{Session: &session.Session{AccountID: 2000000}, World: world}

	mode.applySpecialEffectNotify(ctx, network.SpecialEffectNotify{AID: 3000000, EffectID: network.SpecialEffectBaseLevelUp})
	mode.applySpecialEffectNotify(ctx, network.SpecialEffectNotify{AID: 3000000, EffectID: network.SpecialEffectJobLevelUp})

	if mode.ui.levelUpNotifications.BaseVisible() || mode.ui.levelUpNotifications.JobVisible() {
		t.Fatal("remote level-up effect showed a local availability notification")
	}
}

func TestLevelUpNotificationActionsOpenStatusAndSkillWindows(t *testing.T) {
	mode := &WorldMode{}
	ctx := client.Context{
		Session:   &session.Session{},
		UIManager: gameui.NewManager(),
		ScreenW:   800,
		ScreenH:   600,
	}

	if !mode.handleLevelUpNotificationAction(ctx, gameui.LevelUpNotificationBase|gameui.LevelUpNotificationJob) {
		t.Fatal("level-up notification action was not handled")
	}
	if !mode.ui.statsWindow.IsOpen() {
		t.Fatal("base notification did not open the status window")
	}
	if !mode.ui.skillWindow.IsOpen() {
		t.Fatal("job notification did not open the skill window")
	}
}

func TestSpecialEffectNotifyMapsServerResultEffects(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20}
	mode := &WorldMode{}
	ctx := client.Context{Session: &session.Session{AccountID: 2000000}, World: world}

	for _, effectID := range []uint32{
		network.SpecialEffectRefineFailure,
		network.SpecialEffectRefineSuccess,
		network.SpecialEffectPharmacySuccess,
		network.SpecialEffectPharmacyFailure,
		network.SpecialEffectGameOver,
	} {
		mode.applySpecialEffectNotify(ctx, network.SpecialEffectNotify{AID: 2000000, EffectID: effectID})
	}

	want := []int{effects.RefineOK, effects.PharmacyFail}
	wantSFX := []string{
		"effect\\bs_refinefailed.wav",
		"effect\\bs_refinesuccess.wav",
		"effect\\p_success.wav",
		"effect\\p_failed.wav",
	}
	if len(mode.effects().Active) != len(want) {
		t.Fatalf("world effects = %d, want %d: %+v", len(mode.effects().Active), len(want), mode.effects().Active)
	}
	for i, effect := range mode.effects().Active {
		if effect.EffectID != want[i] {
			t.Fatalf("world effect %d = %+v, want effect %d", i, effect, want[i])
		}
	}
	if len(mode.scheduledSounds) != len(wantSFX) {
		t.Fatalf("scheduled sounds = %d, want %d: %+v", len(mode.scheduledSounds), len(wantSFX), mode.scheduledSounds)
	}
	for i, sound := range mode.scheduledSounds {
		if len(sound.paths) != 1 || sound.paths[0] != wantSFX[i] {
			t.Fatalf("scheduled sound %d = %+v, want %q", i, sound, wantSFX[i])
		}
	}
}

func TestMakingItemWindowEnterConsumesBeforeConsole(t *testing.T) {
	inputState := input.NewState()
	netClient := network.NewClient(20080910, false)
	mode := &WorldMode{}
	ctx := client.Context{
		Input:   inputState,
		Network: netClient,
		ScreenW: 800,
		ScreenH: 600,
	}
	mode.ui.makingItem.OpenList(ctx, network.MakingItemList{
		Items: []network.MakingItemOption{{ItemID: 501}},
	})

	inputState.SetKey(input.KeyEnter, true)
	if _, err := mode.Update(ctx); err != nil {
		t.Fatal(err)
	}

	if mode.ui.console.Active() {
		t.Fatal("console became active before making item window consumed Enter")
	}
	if !mode.ui.makingItem.IsOpen() {
		t.Fatal("making item window closed even though the disconnected test client could not send")
	}
}

func TestMVPNotifyAddsMVPBannerEffect(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20}
	mode := &WorldMode{}
	ctx := client.Context{Session: &session.Session{AccountID: 2000000}, World: world}

	mode.applyMVPNotify(ctx, network.MVPNotify{AID: 2000000})

	if len(mode.effects().Active) != 1 {
		t.Fatalf("world effects = %d, want 1", len(mode.effects().Active))
	}
	if effect := mode.effects().Active[0]; effect.EffectID != effects.Mvp || effect.ActorID != 2000000 {
		t.Fatalf("world effect = %+v", effect)
	}
	if len(mode.scheduledSounds) != 1 || mode.scheduledSounds[0].paths[0] != "effect\\st_mvp.wav" {
		t.Fatalf("scheduled sounds = %+v", mode.scheduledSounds)
	}
}

func TestParameterChangeLevelUpFallbackIsDeduped(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20}
	sessionState := &session.Session{
		AccountID: 2000000,
		Progress:  session.Progress{BaseLevel: 10, JobLevel: 4},
	}
	mode := &WorldMode{}
	ctx := client.Context{Session: sessionState, World: world}

	mode.applyParameterChange(ctx, network.ParameterChange{VarID: network.StatusBaseLevel, Value: 11})
	mode.applyParameterChange(ctx, network.ParameterChange{VarID: network.StatusBaseLevel, Value: 11})
	mode.applyParameterChange(ctx, network.ParameterChange{VarID: network.StatusJobLevel, Value: 5})

	if len(mode.effects().Active) != 2 {
		t.Fatalf("world effects = %d, want 2", len(mode.effects().Active))
	}
	if mode.effects().Active[0].EffectID != effects.BaseLevelUp || mode.effects().Active[1].EffectID != effects.JobLevelUp {
		t.Fatalf("world effects = %+v", mode.effects().Active)
	}
	if !mode.ui.levelUpNotifications.BaseVisible() || !mode.ui.levelUpNotifications.JobVisible() {
		t.Fatal("parameter level-up fallback did not show both availability notifications")
	}
}

func TestSpecialEffectNotifyDedupesParameterLevelUpFallback(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20}
	sessionState := &session.Session{
		AccountID: 2000000,
		Progress:  session.Progress{JobLevel: 21},
	}
	mode := &WorldMode{}
	ctx := client.Context{Session: sessionState, World: world}

	mode.applyParameterChange(ctx, network.ParameterChange{VarID: network.StatusJobLevel, Value: 22})
	mode.applySpecialEffectNotify(ctx, network.SpecialEffectNotify{AID: 2000000, EffectID: network.SpecialEffectJobLevelUp})

	if len(mode.effects().Active) != 1 {
		t.Fatalf("world effects = %d, want 1", len(mode.effects().Active))
	}
	if effect := mode.effects().Active[0]; effect.EffectID != effects.JobLevelUp || effect.ActorID != 2000000 {
		t.Fatalf("world effect = %+v", effect)
	}
	if len(mode.scheduledSounds) != 0 {
		t.Fatalf("scheduled sounds = %+v, want none for job level-up", mode.scheduledSounds)
	}
}

func soulLinkerTestTextures(m *WorldMode) {
	m.mapScene.Textures = make(map[string]*render.Image)
	for _, id := range []int{effects.Stin, effects.Stin3, effects.HitLine4, effects.Stin2, effects.HitLine6, effects.HitTexture, effects.AsuraBodyMonster, effects.Hated, effects.Kaizel, effects.BlueBody, effects.ReflectBody, effects.SmaReady, effects.SoulLink, effects.AsuraBody, effects.BlueCasting, effects.TaeReady, effects.StormKick4, effects.BabyBodyBack, effects.M07} {
		for _, c := range db.EffectSpecs[id].Components {
			for _, file := range append([]string{c.TextureFile}, c.TextureFiles...) {
				if file != "" {
					m.mapScene.Textures["__effectfile_"+file+"|blackkey=true"] = render.NewImage(8, 8)
				}
			}
		}
	}
}
