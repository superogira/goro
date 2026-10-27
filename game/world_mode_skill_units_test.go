package game

import (
	"encoding/binary"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/db"
	"github.com/kivutar/goro/game/effects"
	"github.com/kivutar/goro/game/internal/testutil"
	"github.com/kivutar/goro/game/scene"
	"github.com/kivutar/goro/game/sprites"
	"github.com/kivutar/goro/network"
	"github.com/kivutar/goro/render"
	"github.com/kivutar/goro/res"
	"github.com/kivutar/goro/session"
	worldstate "github.com/kivutar/goro/world"
)

func TestAlchemistCastAndGroundOwnership(t *testing.T) {
	ctx := testutil.ArcherEffectContext()
	ctx.World.Player.Job, ctx.Session.Selected.Job = db.JobAlchemist, db.JobAlchemist
	start := time.Unix(100, 0)
	for _, tc := range []struct {
		skill        uint16
		pose, effect int
	}{
		{db.SkillAMPharmacy, sprites.ActionPCReadyFight, effects.BashBegin},
		{db.SkillAMDemonstration, sprites.ActionPCReadyFight, effects.BashBegin},
		{db.SkillAMSpheremine, sprites.ActionPCReadyFight, effects.BashBegin},
		{db.SkillAMAcidterror, scene.SpriteActionIdle, effects.BeginSpell2},
		{db.SkillAMPotionpitcher, scene.SpriteActionIdle, effects.BeginSpell2},
		{db.SkillAMCannibalize, scene.SpriteActionIdle, effects.BeginSpell2},
		{db.SkillAMCpWeapon, scene.SpriteActionIdle, effects.BeginSpell2},
		{db.SkillAMCpShield, scene.SpriteActionIdle, effects.BeginSpell2},
		{db.SkillAMCpArmor, scene.SpriteActionIdle, effects.BeginSpell2},
		{db.SkillAMCpHelm, scene.SpriteActionIdle, effects.BeginSpell2},
		{db.SkillAMCallhomun, scene.SpriteActionIdle, effects.CoupleCasting},
		{db.SkillAMRest, scene.SpriteActionIdle, effects.CoupleCasting},
		{db.SkillAMResurrecthomun, scene.SpriteActionIdle, effects.CoupleCasting},
		{db.SkillAMBerserkpitcher, sprites.ActionPCAttack1, effects.BeginSpell2},
		{db.SkillAMTwilight1, sprites.ActionPCSkill, effects.Twilight1},
		{db.SkillAMTwilight2, sprites.ActionPCSkill, effects.Twilight2},
		{db.SkillAMTwilight3, sprites.ActionPCSkill, effects.Twilight3},
	} {
		m := &WorldMode{}
		m.startSkillSourceCastAnimation(ctx, 2000000, tc.skill, 3*time.Second, start)
		m.addSkillCastEffects(ctx, tc.skill, 1, 2000000, 0, 0, 0, 3*time.Second, start, "test")
		if m.actorAnims[2000000].actionFamily != tc.pose || len(m.effects().Active) != 2 || m.effects().Active[1].EffectID != tc.effect {
			t.Fatalf("incorrect cast pose or aura for skill %d", tc.skill)
		}
	}
	for _, skill := range []uint16{db.SkillAMDemonstration, db.SkillAMCannibalize, db.SkillAMSpheremine} {
		m := &WorldMode{}
		m.applyGroundSkillNotify(ctx, network.GroundSkillNotify{SourceID: 2000000, SkillID: skill, X: 20, Y: 20})
		if len(m.effects().Active)+len(m.scheduledSounds) != 0 || m.actorAnims[2000000].started.IsZero() {
			t.Fatal("ground notification must animate the caster and leave the visual to its unit/entity")
		}
		if skill != db.SkillAMDemonstration {
			continue
		}
		m.applySkillUnitEntry(ctx, network.SkillUnitEntry{ID: 500, UnitID: 177, X: 20, Y: 20, Visible: true})
		if len(m.effects().Active) != 1 || m.effects().Active[0].EffectID != effects.Demonstration || !m.effects().Active[0].Persistent {
			t.Fatal("Demonstration must belong to its persistent ground unit")
		}
		m.applySkillUnitDisappear(network.SkillUnitDisappear{ID: 500})
		if len(m.effects().Active) != 0 {
			t.Fatal("Demonstration survived unit removal")
		}
	}
	for _, skill := range []uint16{db.SkillAMPharmacy, db.SkillAMTwilight1, db.SkillAMTwilight2, db.SkillAMTwilight3} {
		m := &WorldMode{}
		m.applySkillNoDamageNotify(ctx, network.SkillNoDamageNotify{SourceID: 2000000, TargetID: 2000000, SkillID: skill, Result: 1})
		if len(m.effects().Active)+len(m.scheduledSounds) != 0 || m.actorAnims[2000000].started.IsZero() {
			t.Fatal("crafting completion must animate the caster without a release flare or sound")
		}
	}
}

func TestAssassinPoisonParticleMotionAndFieldLifetime(t *testing.T) {
	for _, id := range []int{effects.EnchantPoison, effects.VenomDust2} {
		effect := effects.Instance{EffectID: id, Starts: time.Unix(100, 0)}
		life := 40
		if id == effects.VenomDust2 {
			life = 10
		}
		first := effects.RisingParticleAt(effect, 0, 0)
		later := effects.RisingParticleAt(effect, 0, life-3)
		if first.Alpha <= 0 || later.Position.Y <= first.Position.Y || later.Angle <= first.Angle || effects.RisingParticleAt(effect, 0, life-1).Alpha != 0 {
			t.Fatalf("effect %d has wrong rise, rotation or fade: %+v / %+v", id, first, later)
		}
		if id == effects.VenomDust2 && (first.Position.X != 0 || first.Position.Z != 0) {
			t.Fatal("Venom Dust particles must stay inside their field cell")
		}
	}
	ctx := testutil.AssassinEffectContext()
	m := &WorldMode{}
	m.applySkillUnitEntry(ctx, network.SkillUnitEntry{ID: 900, CreatorID: 2000000, UnitID: 146, X: 12, Y: 20, Visible: true})
	if len(m.effects().Active) != 1 || !m.effects().Active[0].Persistent || m.effects().Active[0].EffectID != effects.VenomDust2 {
		t.Fatal("Venom Dust must be owned by the server's ground unit")
	}
	m.applySkillUnitDisappear(network.SkillUnitDisappear{ID: 900})
	if len(m.effects().Active) != 0 {
		t.Fatal("Venom Dust survived its field's removal")
	}
}

func TestBardPerformanceOwnership(t *testing.T) {
	ctx := testutil.ArcherEffectContext()
	for _, skill := range []uint16{db.SkillBDLullaby, db.SkillBDRichmankim, db.SkillBDEternalchaos, db.SkillBDDrumbattlefield, db.SkillBDRingnibelungen, db.SkillBDRokisweil, db.SkillBDIntoabyss, db.SkillBDSiegfried, db.SkillBaDissonance, db.SkillBaWhistle, db.SkillBaAssassincross, db.SkillBaPoembragi, db.SkillBaAppleidun} {
		for _, ground := range []bool{false, true} {
			m := &WorldMode{}
			if ground {
				m.applyGroundSkillNotify(ctx, network.GroundSkillNotify{SkillID: skill, SourceID: 2000000})
			} else {
				m.applySkillNoDamageNotify(ctx, network.SkillNoDamageNotify{SkillID: skill, SourceID: 2000000, TargetID: 2000000, Result: 1})
			}
			sounds := 1
			if skill == db.SkillBaDissonance || skill == db.SkillBaWhistle {
				sounds = 0
			}
			if len(m.effects().Active) != 0 || len(m.scheduledSounds) != sounds || !m.actorAnims[2000000].loop {
				t.Fatalf("skill %d ground=%t: activation must only start performance and its one sound", skill, ground)
			}
			m.stopPerformance(ctx, network.StatusEffectChange{StatusID: db.StatusBdplaying, ActorID: 100})
			if len(m.actorAnims) != 0 {
				t.Fatal("performance end must clear both local actor aliases")
			}
		}
	}
	m := &WorldMode{}
	for unit := uint16(158); unit <= 170; unit++ {
		entry := network.SkillUnitEntry{ID: uint32(unit), UnitID: unit, X: 15, Y: 20, Visible: true}
		m.applySkillUnitEntry(ctx, entry)
		started := m.effects().Active[len(m.effects().Active)-1].Starts
		m.applySkillUnitEntry(ctx, entry)
		if !m.effects().Active[len(m.effects().Active)-1].Starts.Equal(started) {
			t.Fatal("ground-unit refresh restarted the song animation")
		}
	}
	if len(m.effects().Active) != 13 || len(m.scheduledSounds) != 0 {
		t.Fatal("each ground unit must own one silent visual")
	}
	if !m.applySkillUnitLookChange(ctx, network.ActorLookChange{ID: 169, Value: 166}) || m.effects().Active[len(m.effects().Active)-1].EffectID != effects.BottomDissonanceGround {
		t.Fatal("overlapping songs must change to Dissonance")
	}
	for unit := uint32(158); unit <= 170; unit++ {
		m.applySkillUnitDisappear(network.SkillUnitDisappear{ID: unit})
	}
	if len(m.effects().Active) != 0 {
		t.Fatal("song symbols survived removal of their ground units")
	}
}

func TestBardPangVoiceAndSongAssets(t *testing.T) {
	ctx := testutil.ArcherEffectContext()
	ctx.Resources = testutil.RealDataManager(t)
	start := time.Unix(100, 0)
	projection := testutil.Projection(800, 600, 15, 20, 0)
	for unit := uint16(158); unit <= 170; unit++ {
		m := &WorldMode{}
		m.applySkillUnitEntry(ctx, network.SkillUnitEntry{ID: 500, UnitID: unit, X: 15, Y: 20, Visible: true})
		effect := m.effects().Active[0]
		spec, _ := effects.Lookup(effect.EffectID)
		for _, file := range spec.Components[0].TextureFiles {
			if m.effects().EffectFileTexture(ctx.Resources, file) == nil {
				t.Fatalf("missing texture %s", file)
			}
		}
		frame := render.NewFrame(800, 600)
		m.effects().Draw(frame, ctx, projection, effect.Starts.Add(20*db.EffectFrameDuration), false)
		if reflect.ValueOf(frame).Elem().FieldByName("worldCommands").Len()+reflect.ValueOf(frame).Elem().FieldByName("worldBillboards").Len() == 0 {
			t.Fatalf("song unit %d drew nothing", unit)
		}
		commands := reflect.ValueOf(frame).Elem().FieldByName("worldCommands")
		if unit == 160 || unit == 165 {
			vertices := commands.Index(0).FieldByName("Vertices")
			for i := 0; i < vertices.Len(); i++ {
				v := vertices.Index(i)
				radius := math.Hypot(v.FieldByName("X").Float()-15.5, v.FieldByName("Z").Float()-20.5)
				if radius < .399 || radius > .801 {
					t.Fatal("song light must rise vertically at its native radius")
				}
			}
		}
		if unit == 166 || unit == 168 {
			vertices := commands.Index(0).FieldByName("Vertices")
			if vertices.Index(0).FieldByName("Y").Float() <= vertices.Index(3).FieldByName("Y").Float() {
				t.Fatal("song ribbon texture is vertically inverted")
			}
		}
	}
	for skill, spec := range db.SkillEffects {
		if isBardPerformance(skill) {
			for _, sound := range spec.SFX {
				if _, err := ctx.Resources.ReadFile("data/wav/" + sound); err != nil {
					t.Fatal(err)
				}
			}
		}
	}
	m := &WorldMode{}
	m.effects().AddBetween(ctx, effects.FVoice, 300, 2000000, start)
	effect := m.effects().Active[0]
	if !effect.HasSpriteFrame || effect.ActorOrigin.X != 20.5 || effect.ProjectileDirection.X != -10 {
		t.Fatal("Pang Voice must capture its target and caster direction")
	}
	frame := render.NewFrame(800, 600)
	m.effects().Draw(frame, ctx, projection, start.Add(6*db.EffectFrameDuration), false)
	if reflect.ValueOf(frame).Elem().FieldByName("worldCommands").Len()+reflect.ValueOf(frame).Elem().FieldByName("worldBillboards").Len() == 0 {
		t.Fatal("Pang Voice drew nothing")
	}
	for initial := 0; initial <= 10; initial++ {
		process, alpha, height := initial, 0, 0.0
		for n := 1; n < 200; n++ {
			process++
			if process <= 10 {
				alpha += 25
			} else {
				alpha -= 5
				if alpha <= 0 {
					alpha, process, height = 0, 0, 0
				}
				height += .04
			}
			a, h := effects.SongOutParticle(n, initial)
			if a != alpha || math.Abs(h-height) > 1e-9 {
				t.Fatalf("Roki's veil diverged at initial %d tick %d", initial, n)
			}
		}
	}
}

func TestClownGypsyMoonlitStatusOwnership(t *testing.T) {
	ctx := testutil.ArcherEffectContext()
	m := &WorldMode{}
	m.applyStatusEffectChange(ctx, network.StatusEffectChange{StatusID: db.StatusMoon, ActorID: 100, Active: true})
	if len(m.effects().Active) != 1 || m.effects().Active[0].EffectID != effects.Moonlit || !m.effects().Active[0].Expires.IsZero() {
		t.Fatal("Moonlit status must create a persistent sphere, including on entry into view")
	}
	start := m.effects().Active[0].Starts
	m.applySkillNoDamageNotify(ctx, network.SkillNoDamageNotify{SkillID: db.SkillCGMoonlit, SourceID: 2000000, TargetID: 2000000, Result: 1})
	m.applySkillUnitEntry(ctx, network.SkillUnitEntry{ID: 500, UnitID: 181, X: 10, Y: 20, Visible: true})
	if len(m.effects().Active) != 1 || !m.effects().Active[0].Starts.Equal(start) || len(m.scheduledSounds) != 1 {
		t.Fatal("activation and ground units must not duplicate or restart the sphere")
	}
	m.applyStatusEffectChange(ctx, network.StatusEffectChange{StatusID: db.StatusMoon, ActorID: 2000000})
	if len(m.effects().Active) != 0 || ctx.World.Player.Opt3State&db.Opt3Moonlit != 0 {
		t.Fatal("Moonlit sphere survived its status ending")
	}
}

func TestClownGypsyHermodeGroundUnitOwnership(t *testing.T) {
	ctx := testutil.ArcherEffectContext()
	m := &WorldMode{}
	m.applyGroundSkillNotify(ctx, network.GroundSkillNotify{SkillID: db.SkillCGHermode, SourceID: 2000000})
	if len(m.effects().Active) != 0 || len(m.scheduledSounds) != 1 || !m.actorAnims[2000000].loop {
		t.Fatal("Hermode ground activation must only begin the dance and sound")
	}
	unit := network.SkillUnitEntry{ID: 500, UnitID: 185, X: 15, Y: 20, Visible: true}
	m.applySkillUnitEntry(ctx, unit)
	if len(m.effects().Active) != 1 || m.effects().Active[0].EffectID != effects.BottomHermode || len(m.scheduledSounds) != 1 {
		t.Fatal("Hermode's unit must own one silent cube")
	}
	start := m.effects().Active[0].Starts
	m.applySkillUnitEntry(ctx, unit)
	if !m.effects().Active[0].Starts.Equal(start) {
		t.Fatal("refresh must not restart the cube")
	}
	m.applySkillUnitDisappear(network.SkillUnitDisappear{ID: unit.ID})
	m.applyStatusEffectChange(ctx, network.StatusEffectChange{StatusID: db.StatusBdplaying, ActorID: 100})
	if len(m.effects().Active)+len(m.actorAnims) != 0 {
		t.Fatal("Hermode ground removal and status ending must stop the cube and dance")
	}
}

func TestDancerPerformanceOwnership(t *testing.T) {
	ctx := testutil.ArcherEffectContext()
	for _, skill := range []uint16{db.SkillDCUglydance, db.SkillDCHumming, db.SkillDCDontforgetme, db.SkillDCFortunekiss, db.SkillDCServiceforyou} {
		for _, ground := range []bool{false, true} {
			m := &WorldMode{}
			if ground {
				m.applyGroundSkillNotify(ctx, network.GroundSkillNotify{SkillID: skill, SourceID: 2000000})
			} else {
				m.applySkillNoDamageNotify(ctx, network.SkillNoDamageNotify{SkillID: skill, SourceID: 2000000, TargetID: 2000000, Result: 1})
			}
			sounds := 1
			if skill == db.SkillDCUglydance {
				sounds = 0
			}
			if len(m.effects().Active) != 0 || len(m.scheduledSounds) != sounds || !m.actorAnims[2000000].pingPong {
				t.Fatalf("dance %d ground=%t must start only its pose and sound", skill, ground)
			}
			m.stopPerformance(ctx, network.StatusEffectChange{StatusID: db.StatusBdplaying, ActorID: 100})
			if len(m.actorAnims) != 0 {
				t.Fatal("dance continued after status end")
			}
		}
	}
	m := &WorldMode{}
	for unit := uint16(171); unit <= 175; unit++ {
		m.applySkillUnitEntry(ctx, network.SkillUnitEntry{ID: uint32(unit), UnitID: unit, X: 15, Y: 20, Visible: true})
	}
	if len(m.effects().Active) != 5 || len(m.scheduledSounds) != 0 {
		t.Fatal("five dance units must create five silent visuals")
	}
	if !m.applySkillUnitLookChange(ctx, network.ActorLookChange{ID: 172, Value: 171}) || m.effects().Active[len(m.effects().Active)-1].EffectID != effects.BottomUglyDanceGround {
		t.Fatal("overlapping dances must change to Ugly Dance")
	}
	for unit := uint32(171); unit <= 175; unit++ {
		m.applySkillUnitDisappear(network.SkillUnitDisappear{ID: unit})
	}
	if len(m.effects().Active) != 0 {
		t.Fatal("dance visual survived ground unit removal")
	}
}

func TestDancerWinkAndClientAssets(t *testing.T) {
	ctx := testutil.ArcherEffectContext()
	ctx.Resources = testutil.RealDataManager(t)
	projection := testutil.Projection(800, 600, 15, 20, 0)
	for unit := uint16(171); unit <= 175; unit++ {
		m := &WorldMode{}
		m.applySkillUnitEntry(ctx, network.SkillUnitEntry{ID: 500, UnitID: unit, X: 15, Y: 20, Visible: true})
		effect := m.effects().Active[0]
		frame := render.NewFrame(800, 600)
		m.effects().Draw(frame, ctx, projection, effect.Starts.Add(20*db.EffectFrameDuration), false)
		commands := reflect.ValueOf(frame).Elem().FieldByName("worldCommands")
		billboards := reflect.ValueOf(frame).Elem().FieldByName("worldBillboards")
		if commands.Len()+billboards.Len() == 0 {
			t.Fatalf("dance unit %d drew nothing", unit)
		}
		if unit == 173 || unit == 175 {
			vertices := commands.Index(0).FieldByName("Vertices")
			count, red, green := 16, 155.0/255, 1.0
			if unit == 175 {
				count, red, green = 8, 1, 150.0/255
			}
			if vertices.Len() != count || math.Abs(vertices.Index(0).FieldByName("ColorR").Float()-red) > .001 || math.Abs(vertices.Index(0).FieldByName("ColorG").Float()-green) > .001 {
				t.Fatalf("dance unit %d has incorrect ribbon count or color", unit)
			}
		}
	}
	for skill, spec := range db.SkillEffects {
		if isDancerPerformance(skill) || skill == db.SkillDCWinkcharm {
			for _, sound := range spec.SFX {
				if _, err := ctx.Resources.ReadFile("data/wav/" + sound); err != nil {
					t.Fatal(err)
				}
			}
		}
	}
	m := &WorldMode{}
	n := network.SkillNoDamageNotify{SkillID: db.SkillDCWinkcharm, SourceID: 2000000, TargetID: 300, Result: 1}
	m.applySkillNoDamageNotify(ctx, n)
	if len(m.effects().Active) != 2 || len(m.scheduledSounds) != 1 {
		t.Fatal("Wink needs its kiss emote, directional sprite, and one sound")
	}
	if e := m.effects().Active[0]; e.EffectID != effects.Emotion || e.ActorID != n.SourceID || e.SpriteFrameOverride != 30 {
		t.Fatal("Wink must display the kiss emote on the caster")
	}
	e := m.effects().Active[1]
	if e.EffectID != effects.Wink || e.ActorID != n.SourceID || e.TargetID != n.TargetID || e.ActorOrigin.X != 20.5 || e.ProjectileDirection.X != -10 {
		t.Fatal("Wink must capture the target position and caster-to-target orientation")
	}
	frame := render.NewFrame(800, 600)
	m.effects().Draw(frame, ctx, projection, e.Starts.Add(6*db.EffectFrameDuration), false)
	if reflect.ValueOf(frame).Elem().FieldByName("worldBillboards").Len() == 0 {
		t.Fatal("Wink sprite did not draw")
	}
	m.effects().RemoveActor(n.SourceID)
	if len(m.effects().Active) != 0 {
		t.Fatal("Wink must disappear when its caster leaves view")
	}
	m = &WorldMode{}
	start := time.Now()
	m.startSkillSourceCastAnimation(ctx, n.SourceID, n.SkillID, time.Second, start)
	m.addSkillCastEffects(ctx, n.SkillID, 0, n.SourceID, n.TargetID, 0, 0, time.Second, start, "test")
	if anim := m.actorAnims[n.SourceID]; !anim.loop || !anim.pingPong || anim.actionFamily != sprites.ActionPCSkill {
		t.Fatal("Wink must dance during its cast")
	}
	if len(m.effects().Active) != 2 {
		t.Fatal("Wink cast needs only the normal circle and lock-on, without an aura")
	}
}

func TestGunslingerGroundAndCoinLifetime(t *testing.T) {
	ctx := testutil.ArcherEffectContext()
	m := &WorldMode{}
	for unit := uint16(190); unit <= 194; unit++ {
		m.applySkillUnitEntry(ctx, network.SkillUnitEntry{ID: 777, UnitID: unit, X: 12, Y: 20, Visible: true})
		if len(m.effects().Active) != 1 || m.effects().Active[0].EffectID != 638+int(unit-190) || !m.effects().Active[0].Persistent {
			t.Fatalf("grenade %d: %+v", unit, m.effects().Active)
		}
		m.applySkillUnitDisappear(network.SkillUnitDisappear{ID: 777})
		if len(m.effects().Active) != 0 {
			t.Fatal("grenade survived unit removal")
		}
	}
	ctx.World.Player.Job = db.JobGunslinger
	ctx.Session.Selected.Job = db.JobGunslinger
	for _, count := range []uint16{1, 5, 6, 10, 0} {
		if !applySpiritSphereCount(ctx, network.SpiritSphereCount{ActorID: 2000000, Count: count}) {
			t.Fatal("missing local actor")
		}
		m.syncActorSpiritEffects(ctx, 2000000)
		if count == 0 {
			if len(m.effects().Active) != 0 {
				t.Fatal("coins survived consumption")
			}
			continue
		}
		if len(m.effects().Active) != 1 || m.effects().Active[0].EffectID != effects.Chookgi3 || m.effects().Active[0].HitCount != int(count) || m.effects().Active[0].GunslingerCoins == nil {
			t.Fatalf("count %d: %+v", count, m.effects().Active)
		}
		e := m.effects().Active[0]
		p := e.GunslingerCoins
		m.syncActorSpiritEffects(ctx, 2000000)
		if m.effects().Active[0].GunslingerCoins != p {
			t.Fatal("unchanged count restarted coins")
		}
		stepped, jumped := effects.NewGunslingerCoins(e), effects.NewGunslingerCoins(e)
		for tick := 0; tick < 100; tick++ {
			stepped.Advance(e, e.Starts.Add(time.Duration(tick)*db.EffectFrameDuration))
		}
		jumped.Advance(e, e.Starts.Add(99*db.EffectFrameDuration))
		if !reflect.DeepEqual(stepped, jumped) {
			t.Fatal("coins depend on render frame rate")
		}
	}
}

func TestHighWizardDamageAndGroundPackets(t *testing.T) {
	ctx := testutil.ArcherEffectContext()
	ctx.World.Player.Job, ctx.Session.Selected.Job = db.JobWizardH, db.JobWizardH
	for _, tc := range []struct {
		skill                 uint16
		effect, sounds, begin int
	}{
		{db.SkillHWMagiccrasher, 380, 1, 0}, {db.SkillHWNapalmvulcan, 401, 5, 1},
	} {
		m := &WorldMode{}
		a := network.ActorActionNotify{SourceID: 2000000, TargetID: 300, SkillID: tc.skill, SourceSpeed: 500, Damage: 100, HitCount: 3, Action: network.ActorActionSkill}
		m.applyActorActionNotify(ctx, a)
		if len(m.effects().Active) != 4+tc.begin {
			t.Fatalf("skill %d needs its release effect, optional caster flare, and three hits", tc.skill)
		}
		release := m.effects().Active[tc.begin]
		if release.EffectID != tc.effect || !release.Starts.Equal(m.actorAnims[2000000].started) || !release.Starts.Before(m.effects().Active[tc.begin+1].Starts) {
			t.Fatal("release effect must precede the hit flashes")
		}
		if len(m.scheduledSounds) != tc.sounds+1+tc.begin {
			t.Fatalf("skill %d scheduled %d sounds, want %d", tc.skill, len(m.scheduledSounds), tc.sounds+1+tc.begin)
		}
		if tc.skill == db.SkillHWNapalmvulcan {
			for i, sound := range m.scheduledSounds[1+tc.begin:] {
				if sound.at.Sub(release.Starts) != time.Duration(20+10*i)*db.EffectFrameDuration {
					t.Fatal("Napalm Vulcan sound must play at ticks 20, 30, 40, 50, 60 regardless of hit count")
				}
			}
		}
		a.Action = 5
		m.applyActorActionNotify(ctx, a)
		if len(m.effects().Active) != 4+tc.begin {
			t.Fatal("splash packet replayed caster/impact effects")
		}
		a.Action, a.Damage = network.ActorActionSkill, 0
		m.effects().Active = nil
		m.scheduledSounds = nil
		m.applyActorActionNotify(ctx, a)
		if len(m.effects().Active) != 1+tc.begin || len(m.scheduledSounds) != tc.sounds+tc.begin {
			t.Fatal("a miss must retain release effects and sounds, without hit flashes")
		}
	}
	m := &WorldMode{}
	m.applyGroundSkillNotify(ctx, network.GroundSkillNotify{SourceID: 2000000, SkillID: db.SkillHWGanbantein, X: 22, Y: 21})
	if len(m.effects().Active) != 1 || m.effects().Active[0].EffectID != 223 || m.effects().Active[0].X != 22 || m.effects().Active[0].ActorID != 0 || m.actorAnims[2000000].actionFamily != sprites.ActionPCSkill || len(m.scheduledSounds) != 1 || m.scheduledSounds[0].positioned {
		t.Fatal("Ganbantein needs its ground rings, skill pose and centered sound")
	}
	m = &WorldMode{}
	m.applyGroundSkillNotify(ctx, network.GroundSkillNotify{SourceID: 2000000, SkillID: db.SkillHWGravitation, X: 22, Y: 21})
	if len(m.effects().Active) != 0 || len(m.scheduledSounds) != 1 || m.scheduledSounds[0].paths[0] != "effect\\wizard_earthspike.wav" {
		t.Fatal("Gravitation ground notification must leave the field to unit entries")
	}
	m.applySkillUnitEntry(ctx, network.SkillUnitEntry{ID: 500, UnitID: 184, X: 22, Y: 21, Visible: true})
	if len(m.effects().Active) != 1 || m.effects().Active[0].EffectID != 522 || !m.effects().Active[0].Persistent {
		t.Fatal("Gravitation unit must own its stone and ice spikes")
	}
	m.applySkillUnitDisappear(network.SkillUnitDisappear{ID: 500})
	if len(m.effects().Active) != 0 {
		t.Fatal("Gravitation field survived its unit")
	}
	m.applyActorActionNotify(ctx, network.ActorActionNotify{SkillID: db.SkillHWGravitation, SourceID: 500, TargetID: 300, Action: network.ActorActionSkill, Damage: 100, HitCount: 1})
	if len(m.effects().Active) != 1 || m.effects().Active[0].EffectID != effects.Hit1 || m.effects().Active[0].ActorID != 300 {
		t.Fatal("Gravitation damage must flash on the victim without duplicating the field")
	}
}

func TestHunterTrapPlacementAndTrigger(t *testing.T) {
	for _, tc := range []struct {
		skill, unit uint16
		effect      int
	}{
		{db.SkillHTLandmine, 147, effects.FirePillarBomb},
		{db.SkillHTAnklesnare, 145, 0}, {db.SkillHTShockwave, 148, 0},
		{db.SkillHTSandman, 149, effects.Sandman}, {db.SkillHTFlasher, 150, effects.Flasher},
		{db.SkillHTFreezingtrap, 151, effects.FreezingTrap}, {db.SkillHTBlastmine, 143, effects.BlastMineBomb},
		{db.SkillHTClaymoretrap, 152, effects.Claymore}, {db.SkillHTTalkiebox, 153, 0},
	} {
		t.Run(db.SkillResourceName[tc.skill], func(t *testing.T) {
			ctx := testutil.ArcherEffectContext()
			m := &WorldMode{}
			m.applyGroundSkillNotify(ctx, network.GroundSkillNotify{SourceID: 2000000, SkillID: tc.skill, X: 15, Y: 20})
			if len(m.effects().Active) != 0 || m.actorAnims[2000000].actionFamily != scene.SpriteActionPickup {
				t.Fatal("placing a trap must play pickup without detonating it")
			}
			m.applySkillUnitEntry(ctx, network.SkillUnitEntry{ID: 500, UnitID: tc.unit, X: 15, Y: 20, Visible: true})
			if tc.unit == 148 && (len(m.effects().Active) != 1 || m.effects().Active[0].EffectID != effects.Shockwave || !m.effects().Active[0].Persistent) {
				t.Fatal("Shock Wave must loop on the trap")
			}
			before := len(m.effects().Active)
			m.applyActorActionNotify(ctx, network.ActorActionNotify{SourceID: 500, TargetID: 300, SkillID: tc.skill, Damage: 100, HitCount: 3, Action: 5})
			hits := 0
			for _, e := range m.effects().Active[before:] {
				if e.EffectID != effects.Hit1 || e.ActorID != 300 {
					t.Fatal("victim damage must only create the small impact, not the trap explosion")
				}
				hits++
			}
			if (tc.unit == 147 || tc.unit == 151 || tc.unit == 143 || tc.unit == 152) && hits != 3 {
				t.Fatal("missing trap damage impacts")
			}
			look := uint32(db.SkillUnitUsedTraps)
			if tc.unit == 147 {
				look = 141
			}
			if !m.applySkillUnitLookChange(ctx, network.ActorLookChange{ID: 500, Value: look}) {
				t.Fatal("unhandled trap look")
			}
			m.applySkillUnitDisappear(network.SkillUnitDisappear{ID: 500})
			if tc.effect == 0 {
				if len(m.effects().Active) != 0 {
					t.Fatal("expired trap retained its aura")
				}
				return
			}
			if len(m.effects().Active) != hits+1 {
				t.Fatalf("got %d explosions", len(m.effects().Active)-hits)
			}
			e := m.effects().Active[hits]
			if e.EffectID != tc.effect || e.ActorID != 0 || e.Persistent || e.X != 15 || e.Y != 20 {
				t.Fatalf("incorrect trap burst: %+v", e)
			}
			if tc.unit == 143 {
				found := false
				for _, sound := range m.scheduledSounds {
					if sound.paths[0] == "effect\\hunter_blastmine.wav" {
						found = true
						if !sound.at.Equal(e.Starts.Add(20 * db.EffectFrameDuration)) {
							t.Fatal("Blast Mine sound must wait twenty ticks")
						}
					}
				}
				if !found {
					t.Fatal("missing Blast Mine sound")
				}
			}
		})
	}
}

func TestHunterTrapUtilityTargetsTrapAndSurvivesRemoval(t *testing.T) {
	for _, skill := range []uint16{db.SkillHTRemovetrap, db.SkillHTSpringtrap} {
		ctx := testutil.ArcherEffectContext()
		ctx.World.Player.Job = db.JobHunter
		ctx.Session.Selected.Job = db.JobHunter
		ctx.Session.Selected.Option = db.EffectStateFalcon
		m := &WorldMode{}
		m.applySkillUnitEntry(ctx, network.SkillUnitEntry{ID: 500, UnitID: 149, X: 15, Y: 25, Visible: true})
		n := network.SkillNoDamageNotify{SourceID: 2000000, TargetID: 500, SkillID: skill}
		m.applySkillNoDamageNotify(ctx, n)
		if len(m.effects().Active) != 0 {
			t.Fatal("failed trap skill emitted effect")
		}
		n.Result = 1
		m.applySkillNoDamageNotify(ctx, n)
		m.applySkillUnitDisappear(network.SkillUnitDisappear{ID: 500})
		if len(m.effects().Active) != 1 || m.effects().Active[0].ActorID != 0 || m.effects().Active[0].X != 15 || m.effects().Active[0].Y != 25 {
			t.Fatal("effect must stay at removed trap")
		}
		if skill == db.SkillHTSpringtrap && (m.falcons[100] == nil || !m.falcons[100].attacking) {
			t.Fatal("Spring Trap did not send the falcon")
		}
	}
}

func TestHunterAnkleSnareClosesOnce(t *testing.T) {
	ctx := testutil.ArcherEffectContext()
	m := &WorldMode{}
	m.applySkillUnitEntry(ctx, network.SkillUnitEntry{ID: 500, UnitID: 145, X: 15, Y: 20, Visible: false})
	m.applySkillUnitUpdate(network.SkillUnitUpdate{ID: 500})
	if !m.applySkillUnitLookChange(ctx, network.ActorLookChange{ID: 500, Value: 145}) {
		t.Fatal("Ankle Snare close was ignored")
	}
	u := m.skillUnitModels[500]
	rsm := &res.RSM{AnimLength: 1000}
	for _, tc := range []struct {
		age    time.Duration
		frame  int
		moving bool
	}{{0, 3, true}, {db.EffectFrameDuration, 103, true}, {time.Second, 903, false}} {
		frame, moving := skillUnitRSMFrame(rsm, res.RSWModel{}, u, u.animationStarted.Add(tc.age))
		if frame != tc.frame || moving != tc.moving {
			t.Fatalf("age %s: frame=%d moving=%t", tc.age, frame, moving)
		}
	}
}

func TestHunterTrapTargetAndTalkieBox(t *testing.T) {
	ctx := testutil.ArcherEffectContext()
	m := &WorldMode{}
	m.applySkillUnitEntry(ctx, network.SkillUnitEntry{ID: 500, UnitID: 153, X: 15, Y: 20, Visible: true})
	projection := testutil.Projection(800, 600, 15, 20, 0)
	p := projection.Project(15.5, 20.5, 0)
	skill := session.Skill{ID: db.SkillHTRemovetrap, Type: skillTargetTrap, Level: 1, Range: 2}
	actor, ok := m.clickedSkillTarget(ctx, projection, skill, int(p.X), int(p.Y), time.Now())
	if !ok || actor.ID != 500 {
		t.Fatal("trap could not be selected")
	}
	c := skillController{mode: m}
	pending := pendingSkillTarget{skill: skill, targetID: 500}
	if x, y, ok := c.targetCell(ctx, pending, time.Now()); !ok || x != 15 || y != 20 {
		t.Fatal("trap chase lost target")
	}
	m.applySpeechBubble(ctx, network.ChatMessage{GID: 500, Text: "Beware : trap!"}, time.Now())
	m.applySkillUnitLookChange(ctx, network.ActorLookChange{ID: 500, Value: 140})
	b := m.speechBubbles[500]
	if b.ground == nil || *b.ground != [2]int{15, 20} || b.text != "Beware : trap!" {
		t.Fatalf("wrong Talkie Box bubble: %+v", b)
	}
	if _, _, ok := c.targetCell(ctx, pending, time.Now()); ok {
		t.Fatal("chase retained removed trap")
	}
	if _, ok := m.clickedSkillTarget(ctx, projection, skill, int(p.X), int(p.Y), time.Now()); ok {
		t.Fatal("used trap still selectable")
	}
	m.effects().AddAt(ctx, effects.BlitzBeat, 300, time.Now())
	m.applySpeechBubble(ctx, network.ChatMessage{GID: 300, Text: "Actor : hello"}, time.Now())
	if b := m.speechBubbles[300]; b.ground != nil || b.text != "hello" {
		t.Fatal("an actor's visual effects must not turn its speech into a ground bubble")
	}
}

func TestNinjaReleaseAndGroundOwnership(t *testing.T) {
	for _, tc := range []struct {
		skill  uint16
		id     int
		owner  uint32
		sounds int
	}{{527, 0, 0, 1}, {536, 635, 300, 1}, {539, 636, 2000000, 1}, {541, 622, 2000000, 1}, {531, 0, 0, 0}, {532, 0, 0, 0}, {543, 0, 0, 0}, {529, 0, 0, 0}} {
		ctx := testutil.ArcherEffectContext()
		m := &WorldMode{}
		m.applySkillNoDamageNotify(ctx, network.SkillNoDamageNotify{SkillID: tc.skill, SourceID: 2000000, TargetID: 300, Result: 1})
		if len(m.scheduledSounds) != tc.sounds {
			t.Fatalf("skill %d sounds=%d", tc.skill, len(m.scheduledSounds))
		}
		if tc.id == 0 {
			if len(m.effects().Active) != 0 {
				t.Fatalf("skill %d invented an effect", tc.skill)
			}
		} else if len(m.effects().Active) != 1 || m.effects().Active[0].EffectID != tc.id || m.effects().Active[0].ActorID != tc.owner {
			t.Fatalf("skill %d wrong effect owner: %+v", tc.skill, m.effects().Active)
		}
		if tc.skill == 527 && m.scheduledSounds[0].positioned {
			t.Fatal("Tatami's sound is centered")
		}
		m.effects().Active, m.scheduledSounds = nil, nil
		m.addSkillEffect(ctx, network.ActorActionNotify{SkillID: tc.skill, SourceID: 2000000, TargetID: 300}, time.Now(), time.Now())
		if len(m.effects().Active)+len(m.scheduledSounds) != 0 {
			t.Fatalf("skill %d replayed its release on damage", tc.skill)
		}
	}
	for _, tc := range []struct {
		skill, unit uint16
		id          int
	}{{527, 188, 631}, {535, 189, 634}, {538, 187, 620}} {
		ctx := testutil.ArcherEffectContext()
		m := &WorldMode{}
		expectEffectIDs(t, "ground is unit-owned", skillGroundEffectIDs(tc.skill))
		m.applySkillUnitEntry(ctx, network.SkillUnitEntry{ID: 777, UnitID: tc.unit, X: 12, Y: 20, Visible: true})
		if len(m.effects().Active) != 1 || !m.effects().Active[0].Persistent || m.effects().Active[0].EffectID != tc.id || (tc.unit != 189 && len(m.scheduledSounds) != 0) {
			t.Fatalf("unit %d ownership: %+v", tc.unit, m.effects().Active)
		}
		m.applySkillUnitDisappear(network.SkillUnitDisappear{ID: 777})
		if len(m.effects().Active) != 0 {
			t.Fatal("ground effect outlived its server unit")
		}
	}
}

func TestPriestGroundSpellsUseUnitLifetime(t *testing.T) {
	ctx := testutil.KnightEffectContext()
	for _, tc := range []struct {
		skill, unit uint16
		effect      int
	}{
		{db.SkillMGSafetywall, 126, effects.SafetyWall},
		{db.SkillMGFirewall, 127, effects.FireWall},
		{db.SkillPRSanctuary, 131, effects.BottomSanc},
		{db.SkillHPBasilica, 180, effects.BottomBasilica},
	} {
		m := &WorldMode{}
		m.applyGroundSkillNotify(ctx, network.GroundSkillNotify{SkillID: tc.skill, SourceID: 2000000, X: 15, Y: 20})
		if len(m.effects().Active) != 0 {
			t.Fatalf("skill %d created an unowned ground effect", tc.skill)
		}
		if tc.skill == db.SkillHPBasilica && (len(m.scheduledSounds) != 1 || m.scheduledSounds[0].paths[0] != "effect\\바실리카.wav") {
			t.Fatal("Basilica must play its sound once at release")
		}
		m.applySkillUnitEntry(ctx, network.SkillUnitEntry{ID: 900, UnitID: tc.unit, X: 15, Y: 20, Visible: true})
		if len(m.effects().Active) != 1 || m.effects().Active[0].EffectID != tc.effect || m.effects().Active[0].ActorID != 900 {
			t.Fatalf("skill %d: missing unit-owned effect", tc.skill)
		}
		m.applySkillUnitDisappear(network.SkillUnitDisappear{ID: 900})
		if len(m.effects().Active) != 0 {
			t.Fatalf("skill %d left an orphan effect", tc.skill)
		}
	}
	for skill, id := range map[uint16]int{db.SkillPRBenedictio: effects.Benedictio, db.SkillPRMagnus: effects.Magnus} {
		m := &WorldMode{}
		m.applyGroundSkillNotify(ctx, network.GroundSkillNotify{SkillID: skill, SourceID: 2000000, X: 15, Y: 20})
		if len(m.effects().Active) != 1 || m.effects().Active[0].EffectID != id || m.effects().Active[0].ActorID != 0 || m.effects().Active[0].X != 15 {
			t.Fatal("ground spell must launch its STR at the selected cell")
		}
		m.applyGroundSkillNotify(ctx, network.GroundSkillNotify{SkillID: skill, SourceID: 2000000, X: 15, Y: 20})
		if len(m.effects().Active) != 2 || len(m.scheduledSounds) != 2 {
			t.Fatal("a second cast at the same cell must play its own activation")
		}
	}
	for skill, kind := range map[uint16]damageFloaterKind{db.SkillPRSanctuary: damageFloaterRecoveryHP, db.SkillMGSrecovery: damageFloaterRecoverySP} {
		m := &WorldMode{}
		m.applySkillNoDamageNotify(ctx, network.SkillNoDamageNotify{SkillID: skill, SourceID: 2000000, TargetID: 300, Result: 1, Amount: 777})
		if len(m.effects().Active)+len(m.scheduledSounds)+len(m.actorAnims) != 0 || len(m.damageFloaters) != 1 || m.damageFloaters[0].text != "777" || m.damageFloaters[0].kind != kind {
			t.Fatalf("skill %d: recovery should show the amount without a spell animation or sound", skill)
		}
	}
}

func TestProfessorGroundUnits(t *testing.T) {
	ctx := testutil.ArcherEffectContext()
	for _, tc := range []struct {
		skill, unit uint16
		effect      int
	}{{404, 182, 405}, {405, 183, 404}} {
		m := &WorldMode{}
		m.applyGroundSkillNotify(ctx, network.GroundSkillNotify{SkillID: tc.skill, SourceID: 2000000, X: 14, Y: 20, Level: 1})
		if len(m.effects().Active) != 0 || len(m.scheduledSounds) != 0 || m.actorAnims[2000000].actionFamily != sprites.ActionPCSkill {
			t.Fatal("ground notification must animate the caster without duplicating the unit's effect")
		}
		m.applySkillUnitEntry(ctx, network.SkillUnitEntry{ID: 900, UnitID: tc.unit, X: 14, Y: 20, Visible: true})
		if len(m.effects().Active) != 1 || m.effects().Active[0].EffectID != tc.effect || !m.effects().Active[0].Persistent {
			t.Fatal("ground unit must own the native persistent effect")
		}
		m.applySkillUnitDisappear(network.SkillUnitDisappear{ID: 900})
		if len(m.effects().Active) != 0 {
			t.Fatal("ground visual survived unit deletion")
		}
	}
}

func TestSageGroundSpellOwnership(t *testing.T) {
	for _, tc := range []struct {
		skill, unit uint16
		cast, field int
	}{
		{db.SkillSAVolcano, 154, effects.Volcano, effects.BottomVolcano},
		{db.SkillSADeluge, 155, effects.Deluge, effects.BottomDeluge},
		{db.SkillSAViolentgale, 156, effects.ViolentGale, effects.BottomViolent},
		{db.SkillSALandprotector, 157, effects.LandProtector, effects.BottomLand},
	} {
		ctx := testutil.KnightEffectContext()
		m := &WorldMode{}
		notify := network.GroundSkillNotify{SkillID: tc.skill, SourceID: 2000000, X: 15, Y: 20}
		m.applyGroundSkillNotify(ctx, notify)
		if len(m.effects().Active) != 1 {
			t.Fatalf("skill %d: missing activation", tc.skill)
		}
		cast := m.effects().Active[0]
		if cast.EffectID != tc.cast || cast.ActorID != 0 || cast.X != 15 || cast.Y != 20 || cast.Persistent {
			t.Fatalf("skill %d: activation must stay at the selected cell", tc.skill)
		}
		if len(m.actorAnims) == 0 {
			t.Fatalf("skill %d: caster has no release animation", tc.skill)
		}
		// A recast at the same location must replay the brief activation.
		m.applyGroundSkillNotify(ctx, notify)
		if len(m.effects().Active) != 2 {
			t.Fatalf("skill %d: recast was suppressed", tc.skill)
		}
		for i := uint32(0); i < 3; i++ {
			entry := network.SkillUnitEntry{ID: 900 + i, UnitID: tc.unit, X: 15 + uint16(i), Y: 20, Visible: true}
			m.applySkillUnitEntry(ctx, entry)
			m.applySkillUnitEntry(ctx, entry)
		}
		if len(m.effects().Active) != 5 {
			t.Fatalf("skill %d: duplicated or missing field cells", tc.skill)
		}
		for i, field := range m.effects().Active[2:] {
			if field.EffectID != tc.field || !field.Persistent || field.X != 15+i || field.ActorID != uint32(900+i) {
				t.Fatalf("skill %d: incorrect field cell: %+v", tc.skill, field)
			}
		}
		m.applySkillUnitDisappear(network.SkillUnitDisappear{ID: 901})
		if len(m.effects().Active) != 4 {
			t.Fatal("removing one field cell removed its neighbors or left an orphan")
		}
		m.applySkillUnitDisappear(network.SkillUnitDisappear{ID: 900})
		m.applySkillUnitDisappear(network.SkillUnitDisappear{ID: 902})
		if len(m.effects().Active) != 2 {
			t.Fatal("field deletion left persistent visuals")
		}
	}
}

func TestWizardGroundSpellOwnership(t *testing.T) {
	ctx := testutil.KnightEffectContext()
	for _, tc := range []struct {
		skill, unit uint16
		id          int
	}{
		{db.SkillWZFirepillar, 135, effects.FirePillarOn},
		{db.SkillWZFirepillar, 136, effects.FirePillarBomb},
		{db.SkillWZIcewall, 141, effects.IceWall},
		{db.SkillWZQuagmire, 142, effects.Quagmire},
	} {
		m := &WorldMode{}
		m.applyGroundSkillNotify(ctx, network.GroundSkillNotify{SkillID: tc.skill, SourceID: 2000000, X: 15, Y: 20})
		if len(m.effects().Active) != 0 {
			t.Fatalf("skill %d created an effect outside its unit lifetime", tc.skill)
		}
		m.applySkillUnitEntry(ctx, network.SkillUnitEntry{ID: 900, UnitID: tc.unit, X: 15, Y: 20, Visible: true})
		if len(m.effects().Active) != 1 || m.effects().Active[0].EffectID != tc.id {
			t.Fatalf("unit %d has no visual", tc.unit)
		}
		m.applySkillUnitDisappear(network.SkillUnitDisappear{ID: 900})
		if len(m.effects().Active) != 0 {
			t.Fatalf("unit %d left an orphan visual", tc.unit)
		}
	}
	for _, skill := range []uint16{db.SkillWZStormgust, db.SkillWZVermilion, db.SkillWZMeteor, db.SkillWZHeavendrive} {
		m := &WorldMode{}
		for i := 0; i < 2; i++ {
			m.applyGroundSkillNotify(ctx, network.GroundSkillNotify{SkillID: skill, SourceID: 2000000, X: 15, Y: 20})
		}
		if len(m.effects().Active) != 2 || m.effects().Active[0].ActorID != 0 || m.effects().Active[0].X != 15 {
			t.Fatalf("skill %d must play every ground notification at the notified cell", skill)
		}
	}
}

func TestSkillUnitPositionUpdatesDoNotCreateActors(t *testing.T) {
	for _, packetID := range []uint16{0x0088, 0x01FF} {
		t.Run(fmt.Sprintf("0x%04X", packetID), func(t *testing.T) {
			world := worldstate.New()
			world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20, Moving: true}
			ctx := client.Context{Session: &session.Session{AccountID: 2000000}, World: world}
			mode := &WorldMode{}
			mode.applySkillUnitEntry(ctx, network.SkillUnitEntry{
				ID: 9001, CreatorID: world.Player.ID, UnitID: 141, X: 12, Y: 20, Visible: true,
			})
			data := make([]byte, 10)
			binary.LittleEndian.PutUint16(data, packetID)
			binary.LittleEndian.PutUint32(data[2:], 9001)
			binary.LittleEndian.PutUint16(data[6:], 12)
			binary.LittleEndian.PutUint16(data[8:], 20)
			packet := network.Packet{ID: packetID, Data: data}
			mode.handleNetworkPacket(ctx, packet, time.Now())
			if len(world.Actors) != 0 {
				t.Fatalf("Ice Wall position update created a phantom actor: %+v", world.Actors)
			}
			if len(mode.effects().Active) != 1 || mode.effects().Active[0].EffectID != effects.IceWall {
				t.Fatal("position update removed the Ice Wall visual")
			}
			mode.applySkillUnitDisappear(network.SkillUnitDisappear{ID: 9001})
			mode.handleNetworkPacket(ctx, packet, time.Now())
			if len(world.Actors) != 0 || len(mode.effects().Active) != 0 {
				t.Fatal("late position update resurrected the expired skill unit")
			}

			// Real Novices and the local player must still receive position fixes.
			world.UpsertActor(worldstate.Actor{ID: 300, Job: db.JobNovice, Name: "Alice", Appearance: true, X: 5, Y: 6, Moving: true})
			binary.LittleEndian.PutUint32(data[2:], 300)
			mode.handleNetworkPacket(ctx, packet, time.Now())
			actor := world.Actors[300]
			if actor.X != 12 || actor.Y != 20 || actor.Moving || actor.Name != "Alice" || !actor.Appearance {
				t.Fatalf("known actor was not updated correctly: %+v", actor)
			}
			binary.LittleEndian.PutUint32(data[2:], world.Player.ID)
			mode.handleNetworkPacket(ctx, packet, time.Now())
			if world.Player.X != 12 || world.Player.Y != 20 || world.Player.Moving || ctx.Session.PlayerX != 12 || ctx.Session.PlayerY != 20 {
				t.Fatal("local position update stopped working")
			}
		})
	}
}

func TestGroundSkillNotifyAddsCellEffect(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20}
	sessionState := &session.Session{AccountID: 2000000}
	mode := &WorldMode{}
	ctx := client.Context{Session: sessionState, World: world}

	mode.applyGroundSkillNotify(ctx, network.GroundSkillNotify{SkillID: 21, SourceID: 2000000, Level: 4, X: 123, Y: 456})
	if len(mode.effects().Active) != 1 {
		t.Fatalf("world effects = %d, want 1", len(mode.effects().Active))
	}
	if effect := mode.effects().Active[0]; effect.ActorID != 0 || effect.EffectID != effects.ThunderStorm || effect.X != 123 || effect.Y != 456 {
		t.Fatalf("effect = %+v", effect)
	}

	mode.applyGroundSkillNotify(ctx, network.GroundSkillNotify{SkillID: 21, SourceID: 2000000, Level: 4, X: 123, Y: 456})
	if len(mode.effects().Active) != 2 {
		t.Fatalf("repeated Thunderstorm casts = %d, want 2", len(mode.effects().Active))
	}
}

func TestPneumaGroundSkillAndUnitDoNotDuplicate(t *testing.T) {
	for _, unitFirst := range []bool{false, true} {
		ctx := testutil.KnightEffectContext()
		mode := &WorldMode{}
		cast := network.GroundSkillNotify{SkillID: db.SkillALPneuma, SourceID: 2000000, Level: 1, X: 12, Y: 23}
		unit := network.SkillUnitEntry{ID: 9001, CreatorID: 2000000, UnitID: 133, X: 12, Y: 23, Visible: true}
		if unitFirst {
			mode.applySkillUnitEntry(ctx, unit)
		}
		mode.applyGroundSkillNotify(ctx, cast)
		mode.applySkillUnitEntry(ctx, unit)
		if len(mode.effects().Active) != 1 {
			t.Fatalf("Pneuma must render once: %+v", mode.effects().Active)
		}
		effect := mode.effects().Active[0]
		if effect.EffectID != effects.Pneuma || effect.ActorID != unit.ID || !effect.Persistent || effect.X != 12 || effect.Y != 23 {
			t.Fatal("Pneuma must belong to its ground unit")
		}
		if mode.actorAnims[2000000].actionFamily != sprites.ActionPCSkill {
			t.Fatal("Pneuma release must animate its caster")
		}
		mode.applySkillUnitDisappear(network.SkillUnitDisappear{ID: unit.ID})
		if len(mode.effects().Active) != 0 {
			t.Fatal("Pneuma remained visible after its unit disappeared")
		}
	}
}

func TestGospelGroundSkillNotifyPlaysOneCastSoundOnCaster(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20}
	mode := &WorldMode{}
	ctx := client.Context{Session: &session.Session{AccountID: 2000000}, World: world}
	notify := network.GroundSkillNotify{SkillID: db.SkillPaGospel, SourceID: 2000000, Level: 10}

	mode.applyGroundSkillNotify(ctx, notify)
	mode.applyGroundSkillNotify(ctx, notify)

	if len(mode.effects().Active) != 1 {
		t.Fatalf("Gospel cast effects = %+v, want one deduplicated caster effect", mode.effects().Active)
	}
	effect := mode.effects().Active[0]
	if effect.EffectID != effects.BottomGospel || effect.ActorID != 2000000 {
		t.Fatalf("Gospel cast effect = %+v, want EF_BOTTOM_GOSPEL on caster", effect)
	}
	if len(mode.scheduledSounds) != 1 || len(mode.scheduledSounds[0].paths) != 1 || mode.scheduledSounds[0].paths[0] != "effect\\가스펠.wav" {
		t.Fatalf("Gospel cast sounds = %+v, want one Gospel sound", mode.scheduledSounds)
	}
}

func TestGospelSkillUnitUsesPersistentGroundVisualWithoutSound(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20}
	mode := &WorldMode{}
	ctx := client.Context{Session: &session.Session{AccountID: 2000000}, World: world}

	mode.applySkillUnitEntry(ctx, network.SkillUnitEntry{ID: 9179, CreatorID: 2000000, UnitID: 179, X: 12, Y: 34, Visible: true})

	if len(mode.effects().Active) != 1 {
		t.Fatalf("Gospel unit effects = %+v, want one", mode.effects().Active)
	}
	effect := mode.effects().Active[0]
	if effect.EffectID != effects.GospelGround || effect.ActorID != 9179 || effect.X != 12 || effect.Y != 34 || !effect.Persistent {
		t.Fatalf("Gospel unit effect = %+v, want persistent 370_ground at unit cell", effect)
	}
	if len(mode.scheduledSounds) != 0 {
		t.Fatalf("Gospel unit sounds = %+v, want cast sound owned by poseffect only", mode.scheduledSounds)
	}
}

func TestSkillUnitEntryAddsAndRemovesCellEffect(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20}
	sessionState := &session.Session{AccountID: 2000000}
	mode := &WorldMode{}
	ctx := client.Context{Session: sessionState, World: world}

	mode.applySkillUnitEntry(ctx, network.SkillUnitEntry{ID: 9001, CreatorID: 2000000, UnitID: 126, X: 123, Y: 456, Visible: true})
	if len(mode.effects().Active) != 1 {
		t.Fatalf("world effects = %d, want 1", len(mode.effects().Active))
	}
	if effect := mode.effects().Active[0]; effect.ActorID != 9001 || effect.EffectID != effects.SafetyWall || effect.X != 123 || effect.Y != 456 {
		t.Fatalf("effect = %+v", effect)
	}

	mode.applySkillUnitDisappear(network.SkillUnitDisappear{ID: 9001})
	if len(mode.effects().Active) != 0 {
		t.Fatalf("world effects after disappear = %d, want 0", len(mode.effects().Active))
	}
}

func TestFireWallSkillUnitEntryUsesPersistentEffect(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20}
	mode := &WorldMode{}
	ctx := client.Context{Session: &session.Session{AccountID: 2000000}, World: world}

	mode.applySkillUnitEntry(ctx, network.SkillUnitEntry{ID: 9007, CreatorID: 2000000, UnitID: 127, X: 12, Y: 34, Visible: true})
	if len(mode.effects().Active) != 1 {
		t.Fatalf("world effects = %d, want 1", len(mode.effects().Active))
	}
	effect := mode.effects().Active[0]
	if effect.ActorID != 9007 || effect.EffectID != effects.FireWall || effect.X != 12 || effect.Y != 34 {
		t.Fatalf("effect = %+v", effect)
	}
	if !effect.Persistent {
		t.Fatalf("fire wall skill unit effect is not persistent")
	}
	if effect.Expires.Sub(effect.Starts) < effects.SkillUnitEffectFallbackDuration {
		t.Fatalf("fire wall lifetime = %s, want skill unit fallback", effect.Expires.Sub(effect.Starts))
	}
	if effect.Duration != 0 {
		t.Fatalf("fire wall animation override = %s, want native component timing", effect.Duration)
	}
}

func TestSkillUnitEntryRefreshesExistingPersistentCellEffect(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20}
	mode := &WorldMode{}
	ctx := client.Context{Session: &session.Session{AccountID: 2000000}, World: world}

	mode.applySkillUnitEntry(ctx, network.SkillUnitEntry{ID: 9008, CreatorID: 2000000, UnitID: 172, X: 12, Y: 34, Visible: true})
	if len(mode.effects().Active) != 1 {
		t.Fatalf("world effects = %d, want 1", len(mode.effects().Active))
	}
	first := mode.effects().Active[0]

	mode.applySkillUnitEntry(ctx, network.SkillUnitEntry{ID: 9008, CreatorID: 2000000, UnitID: 172, X: 13, Y: 35, Visible: true})
	if len(mode.effects().Active) != 1 {
		t.Fatalf("world effects after refresh = %d, want 1: %+v", len(mode.effects().Active), mode.effects().Active)
	}
	effect := mode.effects().Active[0]
	if effect.ActorID != 9008 || effect.EffectID != effects.BottomHummingGround || effect.X != 13 || effect.Y != 35 {
		t.Fatalf("effect after refresh = %+v, want moved humming ground effect", effect)
	}
	if !effect.Starts.Equal(first.Starts) {
		t.Fatalf("effect start changed from %s to %s, want loop phase preserved", first.Starts, effect.Starts)
	}
	if effect.Expires.Before(first.Expires) {
		t.Fatalf("effect expiry moved backward from %s to %s", first.Expires, effect.Expires)
	}
}

func TestWizardSkillUnitEntrySchedulesCellSound(t *testing.T) {
	for _, tc := range []struct {
		name   string
		unitID uint16
		path   string
	}{
		{name: "ice wall", unitID: 141, path: "effect\\wizard_icewall.wav"},
		{name: "fire pillar", unitID: 136, path: "effect\\wizard_fire_pillar_b.wav"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			world := worldstate.New()
			world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20}
			mode := &WorldMode{}
			ctx := client.Context{Session: &session.Session{AccountID: 2000000}, World: world}

			mode.applySkillUnitEntry(ctx, network.SkillUnitEntry{ID: 9101, CreatorID: 2000000, UnitID: tc.unitID, X: 12, Y: 34, Visible: true})

			if len(mode.scheduledSounds) != 1 {
				t.Fatalf("scheduled sounds = %+v, want one", mode.scheduledSounds)
			}
			sound := mode.scheduledSounds[0]
			if len(sound.paths) != 1 || sound.paths[0] != tc.path {
				t.Fatalf("sound paths = %+v, want %s", sound.paths, tc.path)
			}
			if !sound.positioned || sound.actorID != 9101 || !sound.hasPosition {
				t.Fatalf("sound position metadata = %+v", sound)
			}
			x, y, ok := scheduledSoundPosition(ctx, sound, time.Now())
			if !ok || x != 12 || y != 34 {
				t.Fatalf("sound position = %.0f,%.0f ok=%t, want 12,34 true", x, y, ok)
			}
		})
	}
}

func TestStormGustGroundNotifySchedulesSound(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20}
	world.Actors[1100] = worldstate.Actor{ID: 1100, X: 12, Y: 34}
	mode := &WorldMode{}
	ctx := client.Context{Session: &session.Session{AccountID: 2000000}, World: world}

	mode.applyGroundSkillNotify(ctx, network.GroundSkillNotify{SkillID: db.SkillWZStormgust, SourceID: 2000000, X: 12, Y: 34})

	if len(mode.scheduledSounds) != 1 {
		t.Fatalf("scheduled sounds = %+v, want one", mode.scheduledSounds)
	}
	sound := mode.scheduledSounds[0]
	if len(sound.paths) != 1 || sound.paths[0] != "effect\\wizard_stormgust.wav" {
		t.Fatalf("sound paths = %+v, want storm gust sfx", sound.paths)
	}
	x, y, ok := scheduledSoundPosition(ctx, sound, time.Now())
	if !ok || x != 12 || y != 34 {
		t.Fatalf("sound position = %.0f,%.0f ok=%t, want target position 12,34 true", x, y, ok)
	}
}

func TestQuagmireSkillUnitEntriesBuildFiveByFiveZone(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20}
	mode := &WorldMode{}
	ctx := client.Context{Session: &session.Session{AccountID: 2000000}, World: world}

	const centerX, centerY = 50, 60
	entryID := uint32(9100)
	for dy := -2; dy <= 2; dy++ {
		for dx := -2; dx <= 2; dx++ {
			mode.applySkillUnitEntry(ctx, network.SkillUnitEntry{ID: entryID, CreatorID: 2000000, UnitID: 142, X: uint16(centerX + dx), Y: uint16(centerY + dy), Visible: true})
			entryID++
		}
	}

	if len(mode.effects().Active) != 25 {
		t.Fatalf("quagmire world effects = %d, want 25", len(mode.effects().Active))
	}
	cells := make(map[worldstate.WalkStep]bool, len(mode.effects().Active))
	for _, effect := range mode.effects().Active {
		if effect.EffectID != effects.Quagmire {
			t.Fatalf("effect id = %d, want quagmire: %+v", effect.EffectID, effect)
		}
		if !effect.Persistent {
			t.Fatalf("quagmire unit effect is not persistent: %+v", effect)
		}
		cells[worldstate.WalkStep{X: effect.X, Y: effect.Y}] = true
	}
	for y := centerY - 2; y <= centerY+2; y++ {
		for x := centerX - 2; x <= centerX+2; x++ {
			if !cells[worldstate.WalkStep{X: x, Y: y}] {
				t.Fatalf("missing quagmire cell %d,%d in %+v", x, y, cells)
			}
		}
	}
}

func TestTrapSkillUnitEntryAddsRuntimeRSMModel(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20}
	mode := &WorldMode{}
	ctx := client.Context{Session: &session.Session{AccountID: 2000000}, World: world}

	mode.applySkillUnitEntry(ctx, network.SkillUnitEntry{ID: 9201, CreatorID: 2000000, UnitID: 145, X: 12, Y: 34, Visible: true})

	if len(mode.effects().Active) != 0 {
		t.Fatalf("world effects = %+v, want none for trap placement model", mode.effects().Active)
	}
	unit, ok := mode.skillUnitModels[9201]
	if !ok {
		t.Fatal("trap model was not tracked")
	}
	if unit.unitID != 145 || unit.x != 12 || unit.y != 34 || unit.modelPath != db.SkillUnitModels[145].ModelPath {
		t.Fatalf("trap model = %+v", unit)
	}
	if len(unit.modelFallbacks) != 1 || unit.modelFallbacks[0] != "effect\\trap03.rsm" {
		t.Fatalf("trap fallbacks = %+v", unit.modelFallbacks)
	}
	if unit.scale != 0.15 || !unit.hasFixedFrame || unit.fixedFrame != 3 {
		t.Fatalf("trap render controls = scale %.3f fixed=%t frame=%d", unit.scale, unit.hasFixedFrame, unit.fixedFrame)
	}
}

func TestRobrowserRSMTrapSkillUnitModelsUseTrapRenderControls(t *testing.T) {
	expected := map[uint16]string{
		143: "외부소품\\트랩03_3.rsm",
		144: "외부소품\\트랩02.rsm",
		145: "외부소품\\트랩01.rsm",
		147: "외부소품\\트랩03.rsm",
		148: "외부소품\\트랩03_6.rsm",
		149: "외부소품\\트랩03_4.rsm",
		150: "외부소품\\트랩03_5.rsm",
		151: "외부소품\\트랩03_2.rsm",
		152: "외부소품\\트랩04.rsm",
		153: "외부소품\\트랩05.rsm",
		210: "event\\3차트랩_변화01.rsm",
		211: "event\\3차트랩_변수01.rsm",
		212: "event\\3차트랩_변지01.rsm",
		213: "event\\3차트랩_변풍01.rsm",
		214: "event\\3차트랩_화01.rsm",
		215: "event\\3차트랩_수01.rsm",
		216: "event\\3차트랩_풍01.rsm",
		217: "event\\3차트랩_지01.rsm",
		229: "event\\3차트랩_가시01.rsm",
	}
	for unitID, wantPath := range expected {
		spec, ok := db.SkillUnitModels[unitID]
		if !ok {
			t.Fatalf("trap unit %d is not mapped", unitID)
		}
		if spec.ModelPath != wantPath {
			t.Fatalf("trap unit %d model = %q, want %q", unitID, spec.ModelPath, wantPath)
		}
		if spec.Scale != 0.15 || !spec.HasFixedFrame || spec.FixedFrame != 3 {
			t.Fatalf("trap unit %d render controls = scale %.3f fixed=%t frame=%d", unitID, spec.Scale, spec.HasFixedFrame, spec.FixedFrame)
		}
	}
}

func TestHiddenTrapSkillUnitUpdateRevealsRuntimeRSMModel(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20}
	mode := &WorldMode{}
	ctx := client.Context{Session: &session.Session{AccountID: 2000000}, World: world}

	mode.applySkillUnitEntry(ctx, network.SkillUnitEntry{ID: 9202, CreatorID: 3000000, UnitID: 151, X: 12, Y: 34, Visible: false})
	if len(mode.skillUnitModels) != 0 {
		t.Fatalf("visible trap models = %+v, want none", mode.skillUnitModels)
	}
	if _, ok := mode.hiddenSkillUnits[9202]; !ok {
		t.Fatal("hidden trap model was not tracked")
	}

	mode.applySkillUnitUpdate(network.SkillUnitUpdate{ID: 9202})

	if _, ok := mode.hiddenSkillUnits[9202]; ok {
		t.Fatalf("hidden trap still tracked after reveal: %+v", mode.hiddenSkillUnits[9202])
	}
	if unit, ok := mode.skillUnitModels[9202]; !ok || unit.unitID != 151 || unit.x != 12 || unit.y != 34 {
		t.Fatalf("visible trap after reveal = %+v ok=%t", unit, ok)
	}
}

func TestUsedTrapLookChangeRemovesModelAndSpawnsTriggerBurst(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20}
	mode := &WorldMode{}
	ctx := client.Context{Session: &session.Session{AccountID: 2000000}, World: world}

	mode.applySkillUnitEntry(ctx, network.SkillUnitEntry{ID: 9203, CreatorID: 2000000, UnitID: 151, X: 12, Y: 34, Visible: true})
	if !mode.applySkillUnitLookChange(ctx, network.ActorLookChange{ID: 9203, Type: 0, Value: uint32(db.SkillUnitUsedTraps)}) {
		t.Fatal("used trap look change was not handled")
	}

	if len(mode.skillUnitModels) != 0 {
		t.Fatalf("trap models after trigger = %+v, want none", mode.skillUnitModels)
	}
	if len(mode.effects().Active) != 1 {
		t.Fatalf("world effects = %+v, want freezing trap burst", mode.effects().Active)
	}
	effect := mode.effects().Active[0]
	if effect.EffectID != effects.FreezingTrap || effect.ActorID != 0 || effect.X != 12 || effect.Y != 34 || effect.Persistent {
		t.Fatalf("trigger burst = %+v", effect)
	}
}

func TestSkillUnitDisappearRemovesRuntimeRSMModel(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20}
	mode := &WorldMode{}
	ctx := client.Context{Session: &session.Session{AccountID: 2000000}, World: world}

	mode.applySkillUnitEntry(ctx, network.SkillUnitEntry{ID: 9204, CreatorID: 2000000, UnitID: 145, X: 12, Y: 34, Visible: true})
	mode.applySkillUnitDisappear(network.SkillUnitDisappear{ID: 9204})

	if len(mode.skillUnitModels) != 0 || len(mode.hiddenSkillUnits) != 0 || len(mode.effects().Active) != 0 {
		t.Fatalf("skill unit state after disappear models=%+v hidden=%+v effects=%+v", mode.skillUnitModels, mode.hiddenSkillUnits, mode.effects().Active)
	}
}

func TestSkillUnitRSMPlacementUsesCellCenterAndTerrainHeight(t *testing.T) {
	world := &worldstate.World{
		GAT: &res.GAT{
			Width:  2,
			Height: 2,
			Cells: []res.GATCell{
				{Heights: [4]float32{1, 1, 1, 1}},
				{Heights: [4]float32{2, 2, 2, 2}},
				{Heights: [4]float32{3, 3, 3, 3}},
				{Heights: [4]float32{7, 7, 7, 7}},
			},
		},
	}
	unit := skillUnitModel{unitID: 145, x: 1, y: 1, modelPath: db.SkillUnitModels[145].ModelPath}

	placement := skillUnitRSMPlacement(world, 9205, unit)

	if placement.BaseX != 1.5 || placement.BaseY != 1.5 {
		t.Fatalf("base = %.1f,%.1f, want cell center 1.5,1.5", placement.BaseX, placement.BaseY)
	}
	if placement.Model.Position.Y != 7 {
		t.Fatalf("model y = %.1f, want terrain height 7", placement.Model.Position.Y)
	}
	if placement.Model.Scale != (res.RSWVector3{X: 1, Y: 1, Z: 1}) {
		t.Fatalf("scale = %+v, want 1,1,1", placement.Model.Scale)
	}
}

func TestTrapSkillUnitRSMPlacementUsesEffectScale(t *testing.T) {
	world := worldstate.New()
	unit := skillUnitModel{unitID: 145, x: 1, y: 1, modelPath: db.SkillUnitModels[145].ModelPath, scale: db.SkillUnitModels[145].Scale}

	placement := skillUnitRSMPlacement(world, 9206, unit)

	want := res.RSWVector3{X: 0.15, Y: 0.15, Z: 0.15}
	if placement.Model.Scale != want {
		t.Fatalf("scale = %+v, want skill-unit trap scale %+v", placement.Model.Scale, want)
	}
}

func TestTrapSkillUnitRSMFrameUsesClassicClientFixedFrame(t *testing.T) {
	rsm := &res.RSM{
		AnimLength: 12,
		Nodes: []res.RSMNode{{
			Name: "root",
			PositionKeyframes: []res.RSMPositionKeyframe{
				{Frame: 0},
				{Frame: 12, Pos: res.RSMVector3{X: 12}},
			},
		}},
	}
	unit := skillUnitModel{hasFixedFrame: true, fixedFrame: 3}

	frame, animated := skillUnitRSMFrame(rsm, res.RSWModel{}, unit, time.UnixMilli(1000))
	if animated || frame != 3 {
		t.Fatalf("trap RSM frame = %d animated=%t, want fixed frame 3", frame, animated)
	}
}

func TestMappedSkillUnitEntriesUsePersistentEffects(t *testing.T) {
	for unitID, spec := range skillUnitEffectSpecs {
		if len(spec.effectIDs) == 0 {
			continue
		}
		world := worldstate.New()
		world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20}
		mode := &WorldMode{}
		ctx := client.Context{Session: &session.Session{AccountID: 2000000}, World: world}
		entryID := uint32(unitID) + 100000

		mode.applySkillUnitEntry(ctx, network.SkillUnitEntry{ID: entryID, CreatorID: 2000000, UnitID: unitID, X: 12, Y: 34, Visible: true})
		if len(mode.effects().Active) != len(spec.effectIDs) {
			t.Fatalf("unit %d world effects = %d, want %d", unitID, len(mode.effects().Active), len(spec.effectIDs))
		}
		for _, effect := range mode.effects().Active {
			if !effect.Persistent {
				t.Fatalf("unit %d effect %d is not persistent", unitID, effect.EffectID)
			}
		}
	}
}

func TestRepeatedSTRKeyIndexLoops(t *testing.T) {
	starts := time.Unix(100, 0)
	now := starts.Add(250 * time.Millisecond)
	if got := effects.StrEffectKeyIndex(starts, now, 60, 12, true); got != 3 {
		t.Fatalf("repeated key index = %.2f, want 3", got)
	}
	if got := effects.StrEffectKeyIndex(starts, now, 60, 12, false); got != 15 {
		t.Fatalf("one-shot key index = %.2f, want 15", got)
	}
}

func TestWarpPortalSkillUnitEntryAddsAndRemovesCellEffect(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20}
	mode := &WorldMode{}
	ctx := client.Context{Session: &session.Session{AccountID: 2000000}, World: world}

	mode.applySkillUnitEntry(ctx, network.SkillUnitEntry{ID: 9003, CreatorID: 2000000, UnitID: 128, X: 30, Y: 40, Visible: true})
	if len(mode.effects().Active) != 1 {
		t.Fatalf("world effects = %d, want 1", len(mode.effects().Active))
	}
	if effect := mode.effects().Active[0]; effect.ActorID != 9003 || effect.EffectID != effects.Portal || effect.X != 30 || effect.Y != 40 {
		t.Fatalf("effect = %+v", effect)
	}

	mode.applySkillUnitDisappear(network.SkillUnitDisappear{ID: 9003})
	if len(mode.effects().Active) != 0 {
		t.Fatalf("world effects after disappear = %d, want 0", len(mode.effects().Active))
	}
}

func TestPreWarpPortalSkillUnitEntryUsesReadyPortalEffectUntilDisappear(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20}
	mode := &WorldMode{}
	ctx := client.Context{Session: &session.Session{AccountID: 2000000}, World: world}

	mode.applySkillUnitEntry(ctx, network.SkillUnitEntry{ID: 9005, CreatorID: 2000000, UnitID: 129, X: 30, Y: 40, Visible: true})
	if len(mode.effects().Active) != 1 {
		t.Fatalf("world effects = %d, want 1", len(mode.effects().Active))
	}
	effect := mode.effects().Active[0]
	if effect.ActorID != 9005 || effect.EffectID != effects.ReadyPortal || effect.X != 30 || effect.Y != 40 {
		t.Fatalf("effect = %+v", effect)
	}
	if effect.Expires.Sub(effect.Starts) < effects.SkillUnitEffectFallbackDuration {
		t.Fatalf("portal lifetime = %s, want skill unit fallback", effect.Expires.Sub(effect.Starts))
	}
	if effect.Duration != 0 {
		t.Fatalf("portal animation override = %s, want native component timing", effect.Duration)
	}

	mode.applySkillUnitDisappear(network.SkillUnitDisappear{ID: 9005})
	if len(mode.effects().Active) != 0 {
		t.Fatalf("world effects after disappear = %d, want 0", len(mode.effects().Active))
	}
}

func TestWarpPortalSkillUnitLookChangeKeepsPortalAtSameCell(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20}
	mode := &WorldMode{}
	ctx := client.Context{Session: &session.Session{AccountID: 2000000}, World: world}

	mode.applySkillUnitEntry(ctx, network.SkillUnitEntry{ID: 9006, CreatorID: 2000000, UnitID: 129, X: 30, Y: 40, Visible: true})
	if len(mode.effects().Active) != 1 || mode.effects().Active[0].EffectID != effects.ReadyPortal {
		t.Fatalf("world effects before look change = %+v, want ready portal", mode.effects().Active)
	}

	if !mode.applySkillUnitLookChange(ctx, network.ActorLookChange{ID: 9006, Type: 0, Value: 128}) {
		t.Fatal("skill unit look change was not handled")
	}
	if len(mode.effects().Active) != 1 {
		t.Fatalf("world effects = %d, want 1: %+v", len(mode.effects().Active), mode.effects().Active)
	}
	effect := mode.effects().Active[0]
	if effect.ActorID != 9006 || effect.EffectID != effects.Portal || effect.X != 30 || effect.Y != 40 {
		t.Fatalf("effect after look change = %+v, want portal on same unit cell", effect)
	}
}

func TestSkillUnitEntryDispatchesAllMappedUnitEffectArrays(t *testing.T) {
	const unitID uint16 = 65000
	skillUnitEffectSpecs[unitID] = skillUnitEffectSpec{effectIDs: []int{effects.Pneuma, effects.SafetyWall}}
	defer delete(skillUnitEffectSpecs, unitID)

	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20}
	mode := &WorldMode{}
	ctx := client.Context{Session: &session.Session{AccountID: 2000000}, World: world}

	mode.applySkillUnitEntry(ctx, network.SkillUnitEntry{ID: 9004, CreatorID: 2000000, UnitID: unitID, X: 123, Y: 456, Visible: true})

	want := []int{effects.Pneuma, effects.SafetyWall}
	if len(mode.effects().Active) != len(want) {
		t.Fatalf("world effects = %d, want %d: %+v", len(mode.effects().Active), len(want), mode.effects().Active)
	}
	for i, wantEffectID := range want {
		effect := mode.effects().Active[i]
		if effect.ActorID != 9004 || effect.EffectID != wantEffectID || effect.X != 123 || effect.Y != 456 {
			t.Fatalf("effect %d = %+v, want effect %d on unit", i, effect, wantEffectID)
		}
	}

	mode.applySkillUnitDisappear(network.SkillUnitDisappear{ID: 9004})
	if len(mode.effects().Active) != 0 {
		t.Fatalf("world effects after disappear = %d, want 0", len(mode.effects().Active))
	}
}

func TestPneumaSkillUnitEntryAddsAndRemovesCellEffect(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20}
	mode := &WorldMode{}
	ctx := client.Context{Session: &session.Session{AccountID: 2000000}, World: world}

	mode.applySkillUnitEntry(ctx, network.SkillUnitEntry{ID: 9002, CreatorID: 2000000, UnitID: 133, X: 123, Y: 456, Visible: true})
	if len(mode.effects().Active) != 1 {
		t.Fatalf("world effects = %d, want 1", len(mode.effects().Active))
	}
	if effect := mode.effects().Active[0]; effect.ActorID != 9002 || effect.EffectID != effects.Pneuma || effect.X != 123 || effect.Y != 456 {
		t.Fatalf("effect = %+v", effect)
	}

	mode.applySkillUnitDisappear(network.SkillUnitDisappear{ID: 9002})
	if len(mode.effects().Active) != 0 {
		t.Fatalf("world effects after disappear = %d, want 0", len(mode.effects().Active))
	}
}

func TestSkillNoDamageNotifyAddsProvokeEffect(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20}
	world.Actors[1100] = worldstate.Actor{ID: 1100, X: 12, Y: 22}
	sessionState := &session.Session{AccountID: 2000000}
	mode := &WorldMode{}
	ctx := client.Context{Session: sessionState, World: world}

	mode.applySkillNoDamageNotify(ctx, network.SkillNoDamageNotify{SkillID: 6, Amount: 2, TargetID: 1100, SourceID: 2000000, Result: 1})
	if len(mode.effects().Active) != 1 {
		t.Fatalf("world effects = %d, want 1", len(mode.effects().Active))
	}
	if effect := mode.effects().Active[0]; effect.ActorID != 1100 || effect.EffectID != effects.Provoke || effect.X != 12 || effect.Y != 22 {
		t.Fatalf("effect = %+v", effect)
	}
}

func TestSkillNoDamageNotifyAddsSongTalkBeginEffects(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "data"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "data", "dc_scream.txt"), []byte("SCREAM\r\n\tDancer line\r\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "data", "ba_frostjoke.txt"), []byte("FROST JOKE\r\n\tBard line\r\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	manager, err := res.NewManager(dir)
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name     string
		skillID  uint16
		effectID int
		text     string
	}{
		{name: "scream", skillID: db.SkillDCScream, effectID: effects.TalkScream, text: "Dancer line"},
		{name: "frost joke", skillID: db.SkillBaFrostjoke, effectID: effects.TalkFrostJoke, text: "Bard line"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			world := worldstate.New()
			world.Player = worldstate.Actor{ID: 150000, X: 10, Y: 20}
			sessionState := &session.Session{
				AccountID: 2000000,
				CharID:    150000,
				Selected:  session.Character{ID: 150000, Name: "Kivutar", Job: db.JobDancer},
			}
			mode := &WorldMode{}
			ctx := client.Context{Session: sessionState, World: world, Resources: manager}

			mode.applySkillNoDamageNotify(ctx, network.SkillNoDamageNotify{SkillID: tc.skillID, TargetID: 2000000, SourceID: 2000000, Result: 1})
			if len(mode.effects().Active) != 1 {
				t.Fatalf("world effects = %d, want 1", len(mode.effects().Active))
			}
			if effect := mode.effects().Active[0]; effect.ActorID != 2000000 || effect.EffectID != tc.effectID {
				t.Fatalf("effect = %+v, want actor 2000000 effect %d", effect, tc.effectID)
			}
			if bubble, ok := mode.speechBubbles[2000000]; !ok || bubble.text != tc.text {
				t.Fatalf("account bubble = %+v ok=%t, want %q", bubble, ok, tc.text)
			}
			if bubble, ok := mode.speechBubbles[150000]; !ok || bubble.text != tc.text {
				t.Fatalf("char bubble = %+v ok=%t, want %q", bubble, ok, tc.text)
			}
		})
	}
}

func TestSkillNoDamageNotifyAddsStealEffect(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20}
	world.Actors[1100] = worldstate.Actor{ID: 1100, X: 12, Y: 22}
	mode := &WorldMode{}
	ctx := client.Context{Session: &session.Session{AccountID: 2000000}, World: world}

	mode.applySkillNoDamageNotify(ctx, network.SkillNoDamageNotify{SkillID: 50, TargetID: 1100, SourceID: 2000000, Result: 1})
	if len(mode.effects().Active) != 1 {
		t.Fatalf("world effects = %d, want 1", len(mode.effects().Active))
	}
	if effect := mode.effects().Active[0]; effect.ActorID != 1100 || effect.EffectID != effects.Steal || effect.X != 12 || effect.Y != 22 {
		t.Fatalf("effect = %+v", effect)
	}
}

func TestSkillNoDamageNotifyEndureUsesReadyFightAction(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20, Job: 1}
	sessionState := &session.Session{AccountID: 2000000, CharID: 150000, Selected: session.Character{ID: 150000, Job: 1, Hair: 1}}
	mode := &WorldMode{}
	ctx := client.Context{Session: sessionState, World: world}

	mode.applySkillNoDamageNotify(ctx, network.SkillNoDamageNotify{SkillID: 8, TargetID: 2000000, SourceID: 2000000, Result: 1})
	anim, ok := mode.actorAnims[150000]
	if !ok {
		t.Fatal("source animation missing")
	}
	if anim.actionFamily != sprites.ActionPCReadyFight || anim.hasFixedMotion {
		t.Fatalf("source animation = %+v, want reference client READYFIGHT action", anim)
	}
	if len(mode.effects().Active) != 1 || mode.effects().Active[0].EffectID != effects.Endure {
		t.Fatalf("world effects = %+v, want Endure effect", mode.effects().Active)
	}
}

func TestSkillNoDamageNotifyAddsHealEffect(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20, Dir: 4}
	world.Actors[1100] = worldstate.Actor{ID: 1100, X: 12, Y: 22}
	sessionState := &session.Session{AccountID: 2000000, CharID: 150000}
	mode := &WorldMode{}
	ctx := client.Context{Session: sessionState, World: world}

	mode.applySkillNoDamageNotify(ctx, network.SkillNoDamageNotify{SkillID: 28, Amount: 234, TargetID: 1100, SourceID: 2000000, Result: 1})
	if len(mode.effects().Active) != 1 {
		t.Fatalf("world effects = %d, want 1", len(mode.effects().Active))
	}
	if effect := mode.effects().Active[0]; effect.ActorID != 1100 || effect.EffectID != effects.Heal || effect.X != 12 || effect.Y != 22 {
		t.Fatalf("effect = %+v", effect)
	}
	anim, ok := mode.actorAnims[150000]
	if !ok {
		t.Fatal("source cast animation missing")
	}
	if anim.actionFamily != sprites.ActionPCSkill || anim.hasFixedMotion {
		t.Fatalf("source animation = %+v, want reference client DEFAULT skill action", anim)
	}
}

func TestActorActionNotifyHealUsesCastAndOffensiveHealEffect(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20, Dir: 4}
	world.UpsertActor(worldstate.Actor{
		ID:            300,
		X:             11,
		Y:             20,
		Job:           1015,
		ObjectType:    scene.ActorObjectTypeMob,
		HasObjectType: true,
	})
	mode := &WorldMode{}
	ctx := client.Context{
		Session: &session.Session{AccountID: 2000000, CharID: 150000, Selected: session.Character{ID: 150000, Job: 4, Hair: 1}},
		World:   world,
	}

	mode.applyActorActionNotify(ctx, network.ActorActionNotify{
		SkillID:     28,
		SkillLevel:  3,
		SourceID:    2000000,
		TargetID:    300,
		SourceSpeed: 580,
		TargetSpeed: 480,
		Damage:      84,
		HitCount:    1,
		Action:      8,
	})

	anim, ok := mode.actorAnims[150000]
	if !ok {
		t.Fatal("source animation missing")
	}
	if anim.actionFamily != sprites.ActionPCSkill || anim.hasFixedMotion {
		t.Fatalf("source animation = %+v, want reference client DEFAULT skill action", anim)
	}
	found := false
	for _, effect := range mode.effects().Active {
		if effect.EffectID == effects.HealOffensive && effect.ActorID == 300 {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("world effects = %+v, want offensive heal effect on target", mode.effects().Active)
	}
}

func TestActorActionNotifyBashUsesRobrowserWeaponAttackOverride(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20, Dir: 4}
	world.UpsertActor(worldstate.Actor{
		ID:            300,
		X:             11,
		Y:             20,
		Job:           1015,
		ObjectType:    scene.ActorObjectTypeMob,
		HasObjectType: true,
	})
	mode := &WorldMode{}
	ctx := client.Context{
		Session: &session.Session{AccountID: 2000000, CharID: 150000, Selected: session.Character{ID: 150000, Job: 1, Hair: 1, Weapon: 3}},
		World:   world,
	}

	mode.applyActorActionNotify(ctx, network.ActorActionNotify{
		SkillID:     5,
		SkillLevel:  3,
		SourceID:    2000000,
		TargetID:    300,
		SourceSpeed: 580,
		TargetSpeed: 480,
		Damage:      84,
		HitCount:    1,
		Action:      network.ActorActionSkill,
	})

	anim, ok := mode.actorAnims[150000]
	if !ok {
		t.Fatal("source animation missing")
	}
	if anim.actionFamily != sprites.ActionPCAttack2 {
		t.Fatalf("source animation = %+v, want reference client weapon attack override", anim)
	}
}

func TestActorActionNotifyHealDoesNotOverwriteLocalCastWithHurt(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20, Dir: 4}
	mode := &WorldMode{}
	ctx := client.Context{
		Session: &session.Session{AccountID: 2000000, CharID: 150000, Selected: session.Character{ID: 150000, Job: 4, Hair: 1}},
		World:   world,
	}

	mode.applyActorActionNotify(ctx, network.ActorActionNotify{
		SkillID:     28,
		SkillLevel:  3,
		SourceID:    2000000,
		TargetID:    2000000,
		SourceSpeed: 580,
		TargetSpeed: 480,
		Damage:      84,
		HitCount:    1,
		Action:      network.ActorActionSkill,
	})

	anim, ok := mode.actorAnims[150000]
	if !ok {
		t.Fatal("source animation missing")
	}
	if anim.actionFamily != sprites.ActionPCSkill || anim.hasFixedMotion {
		t.Fatalf("source animation = %+v, want reference client DEFAULT skill action", anim)
	}
	if len(mode.effects().Active) != 3 || mode.effects().Active[0].EffectID != effects.HealOffensive || mode.effects().Active[1].EffectID != effects.Hit1 || mode.effects().Active[2].EffectID != effects.BashHit {
		t.Fatalf("world effects = %+v, want offensive Heal followed by the native impact pair", mode.effects().Active)
	}
}

func TestSkillFailAckAddsConsoleErrorWithoutEffect(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20}
	sessionState := &session.Session{AccountID: 2000000}
	mode := &WorldMode{}
	ctx := client.Context{Session: sessionState, World: world}

	mode.applySkillFailAck(ctx, network.SkillFailAck{SkillID: 6, Result: 0, Cause: 0})

	if len(mode.effects().Active) != 0 {
		t.Fatalf("world effects = %d, want 0", len(mode.effects().Active))
	}
	messages := mode.ui.console.Messages()
	if len(messages) != 1 || messages[0].Text != "Action failed." {
		t.Fatalf("console messages = %+v", messages)
	}
}

func TestStealFailAckUsesSkillSpecificMessage(t *testing.T) {
	mode := &WorldMode{}
	ctx := client.Context{Session: &session.Session{AccountID: 2000000}}

	mode.applySkillFailAck(ctx, network.SkillFailAck{SkillID: 50, Result: 0, Cause: 0})

	messages := mode.ui.console.Messages()
	if len(messages) != 1 || messages[0].Text != "Steal failed." {
		t.Fatalf("console messages = %+v", messages)
	}
}

func TestPickedInventoryItemAddsToExistingStack(t *testing.T) {
	sessionState := &session.Session{
		Inventory: session.Inventory{
			Items: []session.InventoryItem{{Index: 7, ItemID: 512, Type: 0, Amount: 3, Identified: true}},
		},
	}

	addPickedSessionInventoryItem(sessionState, session.InventoryItem{Index: 7, ItemID: 512, Type: 3, Amount: 2})

	if got := sessionState.Inventory.Items[0].Amount; got != 5 {
		t.Fatalf("picked stack amount = %d, want 5", got)
	}
	if got := sessionState.Inventory.Items[0].Type; got != 0 {
		t.Fatalf("picked stack type = %d, want preserved healing type", got)
	}
}

func TestSessionItemFromNetworkMarksEquipmentByType(t *testing.T) {
	item := sessionItemFromNetwork(network.InventoryItem{
		Index:      7,
		ItemID:     1201,
		Type:       5,
		Location:   0x0002,
		Identified: true,
		Amount:     1,
	})
	if !item.Equip {
		t.Fatalf("item = %+v, want equipment item", item)
	}
}

func TestSessionItemFromNetworkDefaultsAmmoLocation(t *testing.T) {
	item := sessionItemFromNetwork(network.InventoryItem{
		Index:      8,
		ItemID:     1750,
		Type:       10,
		Identified: true,
		Amount:     100,
	})
	if !item.Equip || item.Location != db.EquipAmmo {
		t.Fatalf("ammo item = %+v, want equipped ammo location 0x%04X", item, db.EquipAmmo)
	}
}

func TestSessionItemFromNetworkDoesNotMarkCardsAsEquipment(t *testing.T) {
	item := sessionItemFromNetwork(network.InventoryItem{
		Index:      9,
		ItemID:     4001,
		Type:       db.ItemTypeCard,
		Location:   db.EquipAccessory1,
		Identified: true,
		Amount:     1,
		Equip:      true,
		Equipped:   true,
	})
	if item.Equip || item.Equipped || item.Location != db.EquipAccessory1 {
		t.Fatalf("card item = %+v, want non-equipment card with original location", item)
	}
}

func TestInventoryItemListReplacesDifferentItemAtReusedIndex(t *testing.T) {
	sessionState := &session.Session{
		Inventory: session.Inventory{
			Items: []session.InventoryItem{{
				Index:    11,
				ItemID:   1201,
				Type:     5,
				Location: 0x0002,
				Amount:   1,
				Equip:    true,
			}},
		},
	}
	ctx := client.Context{Session: sessionState}

	applyInventoryItemList(ctx, []network.InventoryItem{{
		Index:  11,
		ItemID: 938,
		Type:   3,
		Amount: 2,
	}})

	item := sessionState.Inventory.Items[0]
	if item.Equip || item.Location != 0 || item.Type != 3 || item.ItemID != 938 {
		t.Fatalf("item = %+v, want clean replacement", item)
	}
}

func TestPickedEquipmentKeepsEquipMetadata(t *testing.T) {
	sessionState := &session.Session{}
	addPickedSessionInventoryItem(sessionState, session.InventoryItem{
		Index:      11,
		ItemID:     1201,
		Type:       5,
		Location:   0x0002,
		Identified: true,
		Amount:     1,
		Equip:      inventoryItemTypeIsEquipment(5),
	})
	if len(sessionState.Inventory.Items) != 1 {
		t.Fatalf("item count = %d, want 1", len(sessionState.Inventory.Items))
	}
	item := sessionState.Inventory.Items[0]
	if !item.Equip || item.Location != 0x0002 {
		t.Fatalf("picked item = %+v, want equipment metadata", item)
	}
}

func TestApplyInventoryEquipAckUpdatesEquippedState(t *testing.T) {
	sessionState := &session.Session{
		Inventory: session.Inventory{
			Items: []session.InventoryItem{
				{Index: 1, ItemID: 1201, Type: 4, Location: 0x0002, Equip: true},
				{Index: 2, ItemID: 1202, Type: 4, Location: 0x0002, WearLocation: 0x0002, Equip: true, Equipped: true},
			},
		},
	}
	ctx := client.Context{Session: sessionState}

	applyInventoryEquipAck(ctx, network.InventoryEquipAck{Index: 1, Location: 0x0002, Success: true})
	if !sessionState.Inventory.Items[0].Equipped {
		t.Fatal("equipped item was not marked equipped")
	}
	if sessionState.Inventory.Items[1].Equipped {
		t.Fatal("previous item in same location stayed equipped")
	}

	applyInventoryEquipAck(ctx, network.InventoryEquipAck{Index: 1, Location: 0x0002, Success: true, Unequip: true})
	if sessionState.Inventory.Items[0].Equipped {
		t.Fatal("unequipped item stayed equipped")
	}
}

func TestApplyInventoryEquipAckDefaultsAmmoLocation(t *testing.T) {
	sessionState := &session.Session{
		Inventory: session.Inventory{
			Items: []session.InventoryItem{
				{Index: 3, ItemID: 1750, Type: 10, Amount: 100, Equip: true},
			},
		},
	}
	ctx := client.Context{Session: sessionState}

	applyInventoryEquipAck(ctx, network.InventoryEquipAck{Index: 3, Success: true})

	item := sessionState.Inventory.Items[0]
	if !item.Equipped || item.Location != db.EquipAmmo || item.WearLocation != db.EquipAmmo {
		t.Fatalf("ammo item after equip ack = %+v, want equipped ammo location 0x%04X", item, db.EquipAmmo)
	}
}

func TestApplyEquippedArrowMarksAmmoSlot(t *testing.T) {
	sessionState := &session.Session{
		Inventory: session.Inventory{
			Items: []session.InventoryItem{
				{Index: 3, ItemID: 1750, Type: 10, Amount: 100, Location: db.EquipAmmo, WearLocation: db.EquipAmmo, Equip: true, Equipped: true},
				{Index: 9, ItemID: 1751, Type: 10, Amount: 50},
			},
		},
	}
	ctx := client.Context{Session: sessionState}

	applyEquippedArrow(ctx, network.EquippedArrow{Index: 9})

	if sessionState.Inventory.Items[0].Equipped || sessionState.Inventory.Items[0].WearLocation != 0 {
		t.Fatal("previous arrow stayed equipped")
	}
	item := sessionState.Inventory.Items[1]
	if !item.Equip || !item.Equipped || item.Location != db.EquipAmmo || item.WearLocation != db.EquipAmmo {
		t.Fatalf("arrow item after ZC_EQUIP_ARROW = %+v, want equipped ammo location 0x%04X", item, db.EquipAmmo)
	}
}

func TestInventoryEquipmentRebuildsLocalWeaponAppearanceFromEquippedItem(t *testing.T) {
	sessionState := &session.Session{
		CharID:   150000,
		Selected: session.Character{ID: 150000, Job: 2, Weapon: 10},
		Characters: []session.Character{
			{ID: 150000, Job: 2, Weapon: 10},
		},
	}
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 2000000, Job: 2, Weapon: 10}
	ctx := client.Context{Session: sessionState, World: world}

	applyInventoryItemList(ctx, []network.InventoryItem{
		{Index: 2, ItemID: 1607, Type: 5, Location: 0x0002, WearLocation: 0x0002, Equip: true, Equipped: true, Identified: true},
	})

	if sessionState.Selected.Weapon != 1607 || sessionState.Selected.Shield != 0 {
		t.Fatalf("selected weapon = %d shield = %d, want 1607/0", sessionState.Selected.Weapon, sessionState.Selected.Shield)
	}
	if sessionState.Characters[0].Weapon != 1607 {
		t.Fatalf("character list weapon = %d, want 1607", sessionState.Characters[0].Weapon)
	}
	if world.Player.Weapon != 1607 || world.Player.Shield != 0 {
		t.Fatalf("world player weapon = %d shield = %d, want 1607/0", world.Player.Weapon, world.Player.Shield)
	}
}

func TestApplyStoragePacketsUpdateSessionStorage(t *testing.T) {
	sessionState := &session.Session{}
	ctx := client.Context{Session: sessionState}

	applyStorageAmount(ctx, network.StorageAmount{Amount: 1, MaxAmount: 300})
	applyStorageItemList(ctx, []network.InventoryItem{{Index: 3, ItemID: 512, Type: 0, Amount: 4, Identified: true}})
	if !sessionState.Storage.Open {
		t.Fatal("storage was not marked open")
	}
	if sessionState.Storage.Amount != 1 || sessionState.Storage.MaxAmount != 300 {
		t.Fatalf("storage counts = %d/%d", sessionState.Storage.Amount, sessionState.Storage.MaxAmount)
	}
	if len(sessionState.Storage.Items) != 1 || sessionState.Storage.Items[0].ItemID != 512 || sessionState.Storage.Items[0].Amount != 4 {
		t.Fatalf("storage items = %+v", sessionState.Storage.Items)
	}

	applyStorageItemAdded(ctx, network.InventoryItem{Index: 3, ItemID: 512, Type: 0, Amount: 7, Identified: true})
	if got := sessionState.Storage.Items[0].Amount; got != 7 {
		t.Fatalf("storage amount after replace = %d, want 7", got)
	}
	applyStorageItemRemoved(ctx, network.StorageItemRemoved{Index: 3, Amount: 2})
	if got := sessionState.Storage.Items[0].Amount; got != 5 {
		t.Fatalf("storage amount after remove = %d, want 5", got)
	}
	applyStorageClosed(ctx)
	if sessionState.Storage.Open || len(sessionState.Storage.Items) != 0 {
		t.Fatalf("storage after close = %+v", sessionState.Storage)
	}
}

func TestApplyCartPacketsUpdateSessionCart(t *testing.T) {
	sessionState := &session.Session{}
	ctx := client.Context{Session: sessionState}

	applyCartAmount(ctx, network.CartAmount{Amount: 1, MaxAmount: 100, Weight: 450, MaxWeight: 80000})
	applyCartItemList(ctx, []network.InventoryItem{{Index: 3, ItemID: 512, Type: 0, Amount: 4, Identified: true}})
	if !sessionState.Cart.Open {
		t.Fatal("cart was not marked open")
	}
	if sessionState.Cart.Amount != 1 || sessionState.Cart.MaxAmount != 100 || sessionState.Cart.Weight != 450 || sessionState.Cart.MaxWeight != 80000 {
		t.Fatalf("cart counts = %+v", sessionState.Cart)
	}
	if len(sessionState.Cart.Items) != 1 || sessionState.Cart.Items[0].ItemID != 512 || sessionState.Cart.Items[0].Amount != 4 {
		t.Fatalf("cart items = %+v", sessionState.Cart.Items)
	}

	applyCartItemAdded(ctx, network.InventoryItem{Index: 3, ItemID: 512, Type: 0, Amount: 7, Identified: true})
	if got := sessionState.Cart.Items[0].Amount; got != 7 {
		t.Fatalf("cart amount after replace = %d, want 7", got)
	}
	applyCartItemRemoved(ctx, network.CartItemRemoved{Index: 3, Amount: 2})
	if got := sessionState.Cart.Items[0].Amount; got != 5 {
		t.Fatalf("cart amount after remove = %d, want 5", got)
	}
	applyCartClosed(ctx)
	if sessionState.Cart.Open {
		t.Fatalf("cart after close = %+v", sessionState.Cart)
	}
}

func TestAttackFocusTracksTargetAndAnimationStart(t *testing.T) {
	mode := &WorldMode{}
	first := time.Unix(10, 0)
	second := time.Unix(20, 0)

	mode.focusAttackTarget(100, first)
	if mode.attackFocusID != 100 || !mode.attackFocusStart.Equal(first) {
		t.Fatalf("first focus = id %d start %v", mode.attackFocusID, mode.attackFocusStart)
	}

	mode.focusAttackTarget(100, second)
	if !mode.attackFocusStart.Equal(first) {
		t.Fatalf("same target reset animation start to %v", mode.attackFocusStart)
	}

	mode.focusAttackTarget(200, second)
	if mode.attackFocusID != 200 || !mode.attackFocusStart.Equal(second) {
		t.Fatalf("second focus = id %d start %v", mode.attackFocusID, mode.attackFocusStart)
	}

	mode.clearAttackFocus()
	if mode.attackFocusID != 0 || !mode.attackFocusStart.IsZero() {
		t.Fatalf("clear focus = id %d start %v", mode.attackFocusID, mode.attackFocusStart)
	}
}
