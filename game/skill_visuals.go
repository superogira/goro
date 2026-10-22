package game

import (
	"image/color"
	"math"
	"slices"
	"strconv"
	"time"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/db"
	"github.com/kivutar/goro/game/companion"
	"github.com/kivutar/goro/game/effects"
	"github.com/kivutar/goro/game/scene"
	"github.com/kivutar/goro/game/sprites"
	"github.com/kivutar/goro/glog"
	"github.com/kivutar/goro/network"
	"github.com/kivutar/goro/render"
	"github.com/kivutar/goro/res"
)

func (m *WorldMode) addItemUseEffect(ctx client.Context, ack network.UseItemAck) {
	if ack.Result == 0 {
		return
	}
	spec := itemUseEffectSpec(ack.ItemID)
	effectIDs := spec.effectIDs
	casterEffectIDs := spec.effectIDsOnCaster
	if len(effectIDs) == 0 && len(casterEffectIDs) == 0 {
		return
	}
	actorID := itemUseEffectActorID(ctx, ack, spec.target)
	for _, effectID := range effectIDs {
		targetID := actorID
		if effects.EffectDetachesLocalActor(effectID) && scene.IsLocalActor(ctx, targetID) {
			targetID = 0
		}
		if m.effects().Add(ctx, effectID, targetID) {
			glog.Debugf("item effect item=%d actor=%d effect=%d", ack.ItemID, targetID, effectID)
		}
	}
	for _, effectID := range casterEffectIDs {
		casterID := actorID
		if effects.EffectDetachesLocalActor(effectID) && scene.IsLocalActor(ctx, casterID) {
			casterID = 0
		}
		if m.effects().Add(ctx, effectID, casterID) {
			glog.Debugf("item caster effect item=%d actor=%d effect=%d", ack.ItemID, casterID, effectID)
		}
	}
}

func itemUseEffectActorID(ctx client.Context, ack network.UseItemAck, target itemEffectTarget) uint32 {
	ackActorID := itemUseAckActorID(ctx, ack)
	if target == itemEffectTargetMercenary && (ackActorID == 0 || scene.IsLocalActor(ctx, ackActorID)) {
		if actorID := visibleMercenaryActorID(ctx); actorID != 0 {
			return actorID
		}
	}
	return ackActorID
}

func itemUseAckActorID(ctx client.Context, ack network.UseItemAck) uint32 {
	actorID := ack.AID
	if actorID == 0 && ctx.Session != nil {
		actorID = ctx.Session.AccountID
		if actorID == 0 {
			actorID = ctx.Session.CharID
		}
	}
	return actorID
}

func visibleMercenaryActorID(ctx client.Context) uint32 {
	if ctx.World == nil {
		return 0
	}
	if ctx.Session != nil && ctx.Session.Mercenary.ID != 0 {
		if actor, ok := ctx.World.Actors[ctx.Session.Mercenary.ID]; ok && companion.IsMercenary(actor) {
			return actor.ID
		}
	}
	return companion.FindActorID(ctx, scene.ActorObjectTypeMercenary)
}

func (m *WorldMode) applySkillNoDamageNotify(ctx client.Context, notify network.SkillNoDamageNotify) {
	if notify.Result == 0 {
		return
	}
	if notify.SkillID == db.SkillHWSouldrain || notify.SkillID == db.SkillPFSoulchange || (notify.SkillID >= db.SkillSGSunWarm && notify.SkillID <= db.SkillSGStarWarm) {
		if _, ok, _ := scene.ActorForCombatID(ctx, notify.SourceID); !ok {
			return
		}
		if _, ok, _ := scene.ActorForCombatID(ctx, notify.TargetID); !ok {
			return
		}
	}
	now := time.Now()
	if notify.SkillID == db.SkillPRSanctuary || notify.SkillID == db.SkillMGSrecovery {
		if actor, ok, _ := scene.ActorForCombatID(ctx, notify.TargetID); ok && notify.Amount > 0 {
			kind := damageFloaterRecoveryHP
			if notify.SkillID == db.SkillMGSrecovery {
				kind = damageFloaterRecoverySP
			}
			m.addDamageFloater(notify.TargetID, actor.X, actor.Y, strconv.FormatUint(uint64(notify.Amount), 10), damageFloaterWhite, kind, now)
		}
		return
	}
	m.applySkillNameBubble(ctx, notify.SourceID, notify.SkillID, now)
	m.startSkillNoDamageSourceAnimation(ctx, notify, now)
	if m.applyHunterTrapSkill(ctx, notify, now) {
		return
	}
	if notify.SkillID == db.SkillPaSacrifice {
		// Activation is silent; each sacrificed attack launches BASH3D.
		return
	}
	m.applyFalconSkillNoDamageNotify(ctx, notify, now)
	m.applyClownGypsySkill(ctx, notify, now)
	m.applyAlchemistSkill(ctx, notify, now)
	m.applyHighWizardSkill(ctx, notify, now)
	m.applyProfessorSkill(ctx, notify, now)
	if notify.SkillID == db.SkillNJTatamigaeshi {
		m.scheduleSound(now, "effect\\다다미뒤집기.wav")
	}
	if notify.SkillID == db.SkillGSGlittering {
		m.scheduleSound(now, "effect\\플립.wav")
	} else if notify.SkillID == db.SkillGSCracker {
		m.scheduleSound(now, "effect\\크래커.wav")
	}
	m.applyTaekwonSkill(ctx, notify, now)
	m.applyStarGladiatorSkill(ctx, notify, now)
	m.applyWinkCharm(ctx, notify)
	for _, sound := range db.SkillEffects[notify.SkillID].SFX {
		m.scheduleSoundAtActor(now, notify.TargetID, sound)
	}
	effectIDs := skillEffectIDs(notify.SkillID)
	if notify.SkillID == db.SkillALHeal {
		effectIDs = []int{effects.HealEffectID(notify.Amount)}
	}
	for _, effectID := range effectIDs {
		actorID := notify.TargetID
		// The status packet may already have installed the persistent aura.
		if effectID == effects.SteelBody || effectID == effects.AsuraBody || effectID == effects.BabyBody || effectID == effects.Assumptio {
			if scene.IsLocalActor(ctx, actorID) {
				actorID = scene.LocalSkillTarget(ctx)
			}
			if m.effects().HasActive(effectID, actorID, now) {
				continue
			}
		}
		if m.effects().AddBetween(ctx, effectID, actorID, notify.SourceID, now) {
			glog.Debugf("skill effect skill=%d src=%d target=%d effect=%d amount=%d", notify.SkillID, notify.SourceID, notify.TargetID, effectID, notify.Amount)
		}
	}
	for _, effectID := range skillEffectOnCasterIDs(notify.SkillID) {
		if m.effects().AddAt(ctx, effectID, notify.SourceID, now) {
			glog.Debugf("skill caster effect skill=%d src=%d target=%d effect=%d amount=%d", notify.SkillID, notify.SourceID, notify.TargetID, effectID, notify.Amount)
		}
	}
	for _, effectID := range skillSuccessEffectIDs(notify.SkillID) {
		if notify.SkillID == db.SkillALLResurrection && effectID == effects.Revive {
			if actor, ok, _ := scene.ActorForCombatID(ctx, notify.TargetID); !ok || !res.HasPlayerJobToken(int(actor.Job)) {
				continue
			}
		}
		if m.effects().AddBetween(ctx, effectID, notify.TargetID, notify.SourceID, now) {
			glog.Debugf("skill success effect skill=%d src=%d target=%d effect=%d amount=%d", notify.SkillID, notify.SourceID, notify.TargetID, effectID, notify.Amount)
		}
	}
	for _, effectID := range skillSuccessEffectSelfIDs(notify.SkillID) {
		if effectID == effects.RedBody || effectID == effects.LKConcentration {
			actorID := notify.SourceID
			if scene.IsLocalActor(ctx, actorID) {
				actorID = scene.LocalSkillTarget(ctx)
			}
			if effectID == effects.LKConcentration {
				m.effects().Remove(effectID, actorID)
			}
			m.setActorStateEffect(ctx, actorID, effectID, true)
			continue
		}
		if effectID == effects.SpearQuicken || effectID == effects.AuraBlade || effectID == effects.Overthrust {
			m.effects().Remove(effectID, notify.SourceID)
		}
		if m.effects().AddAt(ctx, effectID, notify.SourceID, now) {
			glog.Debugf("skill success self effect skill=%d src=%d target=%d effect=%d amount=%d", notify.SkillID, notify.SourceID, notify.TargetID, effectID, notify.Amount)
		}
	}
	m.applySoulLinkerMonsterSkill(ctx, notify)
}

func (m *WorldMode) applySkillCastNotify(ctx client.Context, notify network.SkillCastNotify) {
	now := time.Now()
	m.applySkillNameBubble(ctx, notify.SourceID, notify.SkillID, now)
	if notify.SkillID == db.SkillChSoulcollect {
		m.effects().AddAt(ctx, effects.Portal5, notify.SourceID, now)
	}
	if notify.SkillID == db.SkillCRAciddemonstration && notify.SourceID != notify.TargetID {
		_, sourceOK, _ := scene.ActorForCombatID(ctx, notify.SourceID)
		if _, targetOK, _ := scene.ActorForCombatID(ctx, notify.TargetID); sourceOK && targetOK {
			m.effects().AddBetween(ctx, effects.ThrowItem4, notify.TargetID, notify.SourceID, now)
		}
	}
	if notify.DelayTime == 0 {
		if notify.SkillID == db.SkillTKHighjump {
			m.startSkillCastSourceAnimation(ctx, notify, 0, now)
		}
		return
	}
	duration := time.Duration(notify.DelayTime) * time.Millisecond
	m.startSkillCastSourceAnimation(ctx, notify, duration, now)
	if !skillHidesCastBar(notify.SkillID) {
		m.startActorCastBar(ctx, notify.SourceID, duration, now)
	}
	m.addSkillCastEffects(ctx, notify.SkillID, notify.Property, notify.SourceID, notify.TargetID, int(notify.X), int(notify.Y), duration, now, "server")
}

func (m *WorldMode) startActorCastBar(ctx client.Context, sourceID uint32, duration time.Duration, started time.Time) {
	if ctx.Config.Headless || sourceID == 0 || duration <= 0 {
		return
	}
	if started.IsZero() {
		started = time.Now()
	}
	bar := actorCastBar{
		started:  started,
		duration: duration,
		color:    color.RGBA{R: 0, G: 255, B: 0, A: 255},
	}
	m.setActorCastBar(sourceID, bar)
	if ctx.Session == nil || !scene.IsLocalActor(ctx, sourceID) {
		return
	}
	m.setActorCastBar(ctx.Session.AccountID, bar)
	m.setActorCastBar(ctx.Session.CharID, bar)
}

func (m *WorldMode) setActorCastBar(actorID uint32, bar actorCastBar) {
	if actorID == 0 {
		return
	}
	if m.actorCastBars == nil {
		m.actorCastBars = make(map[uint32]actorCastBar)
	}
	m.actorCastBars[actorID] = bar
}

func (m *WorldMode) clearActorCastBar(ctx client.Context, actorID uint32) {
	if actorID == 0 || m.actorCastBars == nil {
		return
	}
	delete(m.actorCastBars, actorID)
	if ctx.Session == nil || !scene.IsLocalActor(ctx, actorID) {
		return
	}
	delete(m.actorCastBars, ctx.Session.AccountID)
	delete(m.actorCastBars, ctx.Session.CharID)
}

func (m *WorldMode) startSkillNoDamageSourceAnimation(ctx client.Context, notify network.SkillNoDamageNotify, now time.Time) {
	source, ok, _ := scene.ActorForCombatID(ctx, notify.SourceID)
	if !ok {
		return
	}
	m.faceSkillSource(ctx, notify.SourceID, notify.TargetID, 0, 0)
	if notify.SkillID == db.SkillKNAutocounter {
		// Native SetAction's second argument is speed; the frozen motion is zero.
		m.setCombatActorAction(ctx, notify.SourceID, actorAnimation{
			actionFamily: attackActionFamilyForActorWithResources(ctx.Resources, source),
			started:      now, duration: 1500 * time.Millisecond,
			hasPlay: true, hasFixedMotion: true, fixedMotion: 0, autoCounter: true,
		})
		return
	}
	if notify.SkillID == db.SkillHWSouldrain {
		m.setCombatActorAction(ctx, notify.SourceID, actorAnimation{
			actionFamily: sprites.ActionPCAttack3, started: now, duration: time.Second,
			play: true, hasPlay: true, length: 4, hasLength: true,
			speed: 10 * db.EffectFrameDuration, hasSpeed: true,
			keepMovement: true,
		})
		return
	}
	if notify.SkillID == db.SkillLKAurablade {
		m.setCombatActorAction(ctx, notify.SourceID, actorAnimation{
			actionFamily: sprites.ActionPCReadyFight, started: now, duration: time.Second,
			play: true, hasPlay: true, length: 6, hasLength: true,
			speed: 10 * db.EffectFrameDuration, hasSpeed: true,
		})
		return
	}
	m.startSkillActionAnimation(ctx, notify.SourceID, source, skillAction(notify.SkillID), now, scene.DefaultAttackAnimationDuration)
}

func (m *WorldMode) startSkillCastSourceAnimation(ctx client.Context, notify network.SkillCastNotify, duration time.Duration, now time.Time) {
	m.faceSkillSource(ctx, notify.SourceID, notify.TargetID, int(notify.X), int(notify.Y))
	m.startSkillSourceCastAnimation(ctx, notify.SourceID, notify.SkillID, duration, now)
}

func (m *WorldMode) startSkillSourceCastAnimation(ctx client.Context, sourceID uint32, skillID uint16, duration time.Duration, now time.Time) {
	source, ok, _ := scene.ActorForCombatID(ctx, sourceID)
	if !ok {
		return
	}
	if skillID == db.SkillTKHighjump && res.HasPlayerJobToken(int(source.Job)) {
		m.startSkillActionAnimation(ctx, sourceID, source, taekwonJumpSequence(true), now, duration)
		if scene.IsLocalActor(ctx, sourceID) {
			m.effects().Remove(effects.LandBody, ctx.World.Player.ID)
			m.effects().Remove(effects.JumpBody, ctx.World.Player.ID)
			sourceID = scene.LocalSkillTarget(ctx)
		}
		m.effects().Remove(effects.LandBody, sourceID)
		m.effects().Remove(effects.JumpBody, sourceID)
		m.effects().AddAt(ctx, effects.JumpBody, sourceID, now)
		m.effects().AddAt(ctx, effects.Peong, sourceID, now)
		return
	}
	if spec, ok := gunslingerCastAction(skillID, now); ok && res.HasPlayerJobToken(int(source.Job)) {
		m.startSkillActionAnimation(ctx, sourceID, source, spec, now, duration)
		return
	}
	if ninjaCastingSkill(skillID) && res.HasPlayerJobToken(int(source.Job)) {
		spec := skillActionSpec{defined: true, action: skillActorActionSkill, hasFrame: true, frame: 2 + int(now.UnixNano()&1), duration: 500 * time.Millisecond}
		m.startSkillActionAnimation(ctx, sourceID, source, spec, now, duration)
		return
	}
	if skillID == db.SkillDCWinkcharm {
		spec := danceSkillActionSpec()
		spec.next = &idleSkillActionSpec
		m.startSkillActionAnimation(ctx, sourceID, source, spec, now, duration)
		return
	}
	if (skillID == db.SkillCRSlimpitcher || skillID == db.SkillCRAciddemonstration) && res.HasPlayerJobToken(int(source.Job)) {
		m.startSkillActionAnimation(ctx, sourceID, source, skillActionSpec{
			defined: true, action: skillActorActionAttack1, play: true, length: 5,
			speed: 10 * db.EffectFrameDuration, duration: time.Second, next: &idleSkillActionSpec,
		}, now, time.Second)
		return
	}
	actionFamily := skillCastActionFamilyForActor(source, skillID)
	if skillID == db.SkillBSHammerfall {
		actionFamily = attackActionFamilyForActorWithResources(ctx.Resources, source)
	}
	if (skillID == db.SkillSNSharpshooting || skillID == db.SkillCGArrowvulcan || skillID == db.SkillChSoulcollect || skillID == db.SkillMOCallspirits || skillID == db.SkillMOKitranslation) && res.HasPlayerJobToken(int(source.Job)) {
		m.setCombatActorAction(ctx, sourceID, actorAnimation{
			actionFamily: actionFamily, started: now, duration: duration,
			hasPlay: true, hasFixedMotion: true, fixedMotion: 0,
		})
		return
	}
	if actionFamily == scene.SpriteActionPickup {
		duration = m.actorActionDuration(ctx, source, actionFamily, 300*time.Millisecond)
	}
	m.startCombatAnimation(ctx, sourceID, actionFamily, now, duration)
}

func (m *WorldMode) faceSkillSource(ctx client.Context, sourceID, targetID uint32, cellX, cellY int) {
	source, sourceOK, sourceLocal := scene.ActorForCombatID(ctx, sourceID)
	if !sourceOK {
		return
	}
	if target, targetOK, _ := scene.ActorForCombatID(ctx, targetID); targetOK {
		m.faceCombatSource(ctx, source, sourceLocal, target)
		return
	}
	if x, y, ok := m.skillUnitCell(targetID); ok {
		cellX, cellY = x, y
	}
	if cellX == 0 && cellY == 0 {
		return
	}
	dir := directionFromDelta(source.X, source.Y, cellX, cellY, source.Dir)
	if sourceLocal {
		ctx.World.Player.Dir = dir
		ctx.World.Dir = dir
		return
	}
	source.Dir = dir
	upsertActor(ctx, source)
}

func (m *WorldMode) applyGroundSkillNotify(ctx client.Context, notify network.GroundSkillNotify) {
	if m.applyHunterGroundSkill(ctx, notify, time.Now()) || m.applyPerformanceGroundSkill(ctx, notify, time.Now()) || m.applyCreatorGroundSkill(ctx, notify, time.Now()) {
		return
	}
	if notify.SkillID == db.SkillACShower {
		now := time.Now()
		m.faceSkillSource(ctx, notify.SourceID, 0, int(notify.X), int(notify.Y))
		if actor, ok, _ := scene.ActorForCombatID(ctx, notify.SourceID); ok {
			m.startSkillActionAnimation(ctx, notify.SourceID, actor, skillAction(notify.SkillID), now, scene.DefaultAttackAnimationDuration)
			family := skillActionFamilyForActorWithResources(ctx.Resources, actor, notify.SkillID)
			if action, ok := m.actorResolvedAction(ctx, actor, family); ok {
				act := m.actorActionACT(ctx, actor)
				motion := actionAttackMarkerMotion(act, action)
				delay := combatMotionDelayFromAction(action, scene.DefaultAttackAnimationDuration, motion)
				m.scheduleSkillActionSoundAtActor(now.Add(delay), notify.SourceID, notify.SkillID, actionSFXCandidatesForActor(actor, act, action, motion)...)
			}
		}
		m.applySkillNameBubble(ctx, notify.SourceID, notify.SkillID, now)
		m.effects().AddArrowShower(ctx, notify.SourceID, float64(notify.X), float64(notify.Y), true, now)
		return
	}
	switch notify.SkillID {
	case db.SkillALPneuma, db.SkillALWarp, db.SkillMGSafetywall, db.SkillMGFirewall, db.SkillPRSanctuary, db.SkillHPBasilica,
		db.SkillWZFirepillar, db.SkillWZIcewall, db.SkillWZQuagmire,
		db.SkillAMDemonstration, db.SkillAMCannibalize, db.SkillAMSpheremine, db.SkillHWGravitation,
		db.SkillPFFogwall, db.SkillPFSpiderweb, db.SkillGSGrounddrift:
		now := time.Now()
		m.faceSkillSource(ctx, notify.SourceID, 0, int(notify.X), int(notify.Y))
		if actor, ok, _ := scene.ActorForCombatID(ctx, notify.SourceID); ok {
			m.startSkillActionAnimation(ctx, notify.SourceID, actor, skillAction(notify.SkillID), now, scene.DefaultAttackAnimationDuration)
		}
		if notify.SkillID == db.SkillHWGravitation {
			m.scheduleSound(now, "effect\\wizard_earthspike.wav")
		}
		if notify.SkillID == db.SkillHPBasilica {
			m.scheduleSoundAtActor(now, notify.SourceID, "effect\\바실리카.wav")
		}
		// The subsequent skill-unit entry owns the ground visual and its lifetime.
		return
	}
	// rAthena sends Battle Chant's cast notification at 0,0 and follows it with
	// one UNT_GOSPEL entry per field cell. Play EF_BOTTOM_GOSPEL once on the
	// caster here; the unit entries own the persistent ground visuals.
	if notify.SkillID == db.SkillPaGospel && notify.X == 0 && notify.Y == 0 {
		for _, effectID := range skillEffectIDs(notify.SkillID) {
			if m.effects().AddIfMissing(ctx, effectID, notify.SourceID) {
				glog.Debugf("ground skill caster effect skill=%d src=%d level=%d effect=%d", notify.SkillID, notify.SourceID, notify.Level, effectID)
			}
		}
		return
	}
	effectIDs := skillGroundEffectIDs(notify.SkillID)
	if len(effectIDs) == 0 {
		effectIDs = skillEffectIDs(notify.SkillID)
	}
	if len(effectIDs) == 0 {
		return
	}
	now := time.Now()
	switch notify.SkillID {
	case db.SkillPRMagnus, db.SkillPRBenedictio, db.SkillMGThunderstorm,
		db.SkillBSHammerfall, db.SkillHWGanbantein,
		db.SkillWZStormgust, db.SkillWZVermilion, db.SkillWZMeteor, db.SkillWZHeavendrive,
		db.SkillSAVolcano, db.SkillSADeluge, db.SkillSAViolentgale, db.SkillSALandprotector:
		if notify.SkillID != db.SkillWZMeteor {
			m.faceSkillSource(ctx, notify.SourceID, 0, int(notify.X), int(notify.Y))
		}
		// Falling meteors must not interrupt a walk begun after casting.
		if actor, ok, _ := scene.ActorForCombatID(ctx, notify.SourceID); ok &&
			(notify.SkillID != db.SkillWZMeteor || !scene.ActorIsMovingAt(actor, now)) {
			spec := skillAction(notify.SkillID)
			if notify.SkillID == db.SkillWZMeteor {
				// Each meteor holds the release pose instead of turning toward its impact.
				spec = skillActionSpec{defined: true, action: skillActorActionSkill, hasFrame: true}
				actor.Dir = 0
				if action, ok := m.actorResolvedAction(ctx, actor, sprites.ActionPCSkill); ok {
					spec.frame = max(0, len(action.Animations)-1)
				}
			}
			m.startSkillActionAnimation(ctx, notify.SourceID, actor, spec, now, scene.DefaultAttackAnimationDuration)
		}
		if notify.SkillID == db.SkillHWGanbantein {
			m.scheduleSound(now, "effect\\EF_FireWall.wav")
		}
		for _, effectID := range effectIDs {
			m.effects().AddCellLifetime(ctx, effectID, 0, int(notify.X), int(notify.Y), now, 0, false)
		}
		return
	}
	m.applyFalconGroundSkillNotify(ctx, notify, now)
	for _, effectID := range effectIDs {
		if m.effects().AddCellIfMissing(ctx, effectID, int(notify.X), int(notify.Y), now) {
			glog.Debugf("ground skill effect skill=%d src=%d level=%d cell=%d,%d effect=%d", notify.SkillID, notify.SourceID, notify.Level, notify.X, notify.Y, effectID)
		}
	}
}

func (m *WorldMode) applySkillUnitEntry(ctx client.Context, entry network.SkillUnitEntry) {
	if entry.HasMessage {
		m.applyGraffitiEntry(ctx, entry)
		return
	}
	m.removeGraffiti(entry.ID)
	if m.applySkillUnitModelEntry(entry) {
		if entry.UnitID == 148 { // Shock Wave Trap owns its looping STR.
			m.effects().AddCellLifetime(ctx, effects.Shockwave, entry.ID, int(entry.X), int(entry.Y), time.Now(), effects.SkillUnitEffectFallbackDuration, true)
		}
		return
	}
	if !entry.Visible {
		return
	}
	effectIDs := skillUnitEffectIDs(entry.UnitID)
	if len(effectIDs) == 0 {
		return
	}
	now := time.Now()
	for _, effectID := range effectIDs {
		if m.effects().AddCellLifetime(ctx, effectID, entry.ID, int(entry.X), int(entry.Y), now, effects.SkillUnitEffectFallbackDuration, true) {
			if entry.UnitID == 189 {
				effect := m.effects().Active[len(m.effects().Active)-1]
				if effects.HitEffectRandom(effect, 0, 8) == 0 {
					m.scheduleSoundAtWorldEffect(now, effect, "effect\\화염진.wav")
				}
			}
			glog.Debugf("skill unit effect unit=%d id=%d creator=%d cell=%d,%d effect=%d", entry.UnitID, entry.ID, entry.CreatorID, entry.X, entry.Y, effectID)
		}
	}
}

func (m *WorldMode) applySkillUnitLookChange(ctx client.Context, look network.ActorLookChange) bool {
	if look.Type != 0 || look.ID == 0 {
		return false
	}
	if m.applyUsedTrapLookChange(ctx, look) {
		return true
	}
	if m.applyTrapModelLookChange(look) {
		return true
	}
	effectIDs := skillUnitEffectIDs(uint16(look.Value))
	if len(effectIDs) == 0 {
		return false
	}
	x, y, ok := m.skillUnitCell(look.ID)
	if !ok {
		return false
	}
	m.effects().RemoveUnit(look.ID)
	now := time.Now()
	for _, effectID := range effectIDs {
		if m.effects().AddCellLifetime(ctx, effectID, look.ID, x, y, now, effects.SkillUnitEffectFallbackDuration, true) {
			glog.Debugf("skill unit effect changed id=%d unit=%d cell=%d,%d effect=%d", look.ID, look.Value, x, y, effectID)
		}
	}
	return true
}

func (m *WorldMode) applySkillUnitDisappear(disappear network.SkillUnitDisappear) {
	if disappear.ID == 0 {
		return
	}
	m.removeGraffiti(disappear.ID)
	removedEffects := m.effects().RemoveUnit(disappear.ID)
	removedModel := m.removeSkillUnitModelOnly(disappear.ID)
	if removedEffects || removedModel {
		glog.Debugf("skill unit effect removed id=%d", disappear.ID)
	}
}

func (m *WorldMode) skillUnitCell(id uint32) (int, int, bool) {
	if x, y, ok := m.skillUnitModelCell(id); ok {
		return x, y, true
	}
	for _, effect := range m.effects().Active {
		if effect.ActorID == id {
			return effect.X, effect.Y, true
		}
	}
	return 0, 0, false
}

func (m *WorldMode) applySpecialEffectNotify(ctx client.Context, notify network.SpecialEffectNotify) {
	effectID := specialEffectID(notify.EffectID)
	if effectID <= 0 {
		return
	}
	if scene.IsLocalActor(ctx, notify.AID) {
		switch notify.EffectID {
		case network.SpecialEffectBaseLevelUp,
			network.SpecialEffectSuperNoviceBaseLevelUp,
			network.SpecialEffectTaekwonBaseLevelUp:
			m.ui.levelUpNotifications.NotifyBase()
		case network.SpecialEffectJobLevelUp,
			network.SpecialEffectSuperNoviceJobLevelUp:
			m.ui.levelUpNotifications.NotifyJob()
		}
	}
	if effectID == effects.NPCStop2Del {
		m.effects().Remove(effects.NPCStop2, notify.AID)
		return
	}
	spec, _ := effects.Lookup(effectID)
	if craftingResultSpecialEffect(notify.EffectID) {
		m.replaceCraftingResultEffect(ctx, notify, effectID, spec)
		return
	}
	// Each Kaupe dodge is a separate hit, even while an earlier dodge fades.
	if effectID != effects.StormKick4 && m.effects().HasActive(effectID, notify.AID, time.Now()) {
		return
	}
	if m.effects().Add(ctx, effectID, notify.AID) {
		glog.Debugf("special effect actor=%d special=%d effect=%d sfx=%v", notify.AID, notify.EffectID, effectID, spec.Sfx)
	}
}

func (m *WorldMode) applyMVPNotify(ctx client.Context, notify network.MVPNotify) {
	if m.effects().Add(ctx, effects.Mvp, notify.AID) {
		glog.Debugf("mvp effect actor=%d effect=%d", notify.AID, effects.Mvp)
	}
}

func (m *WorldMode) applySkillFailAck(ctx client.Context, ack network.SkillFailAck) {
	if ack.Result != 0 {
		return
	}
	m.clearSenseRequest(ack.SkillID)
	message := skillFailMessage(ack)
	glog.Debugf("skill fail ack skill=%d num=%d item=%d result=%d cause=%d msg=%q", ack.SkillID, ack.Number, ack.ItemID, ack.Result, ack.Cause, message)
	m.ui.console.AddErrorMessage("%s", message)
}

func specialEffectID(effectID uint32) int {
	if mapped, ok := networkSpecialEffectIDs[effectID]; ok {
		return mapped
	}
	if _, ok := effects.Lookup(int(effectID)); ok {
		return int(effectID)
	}
	return 0
}

var networkSpecialEffectIDs = map[uint32]int{
	network.SpecialEffectBaseLevelUp:            effects.BaseLevelUp,
	network.SpecialEffectJobLevelUp:             effects.JobLevelUp,
	network.SpecialEffectRefineFailure:          effects.RefineFail,
	network.SpecialEffectRefineSuccess:          effects.RefineOK,
	network.SpecialEffectGameOver:               0,
	network.SpecialEffectPharmacySuccess:        effects.PharmacyOK,
	network.SpecialEffectPharmacyFailure:        effects.PharmacyFail,
	network.SpecialEffectSuperNoviceBaseLevelUp: effects.BaseLevelUp,
	network.SpecialEffectSuperNoviceJobLevelUp:  effects.JobLevelUp,
	network.SpecialEffectTaekwonBaseLevelUp:     effects.BaseLevelUp,
}

func craftingResultSpecialEffect(effectID uint32) bool {
	switch effectID {
	case network.SpecialEffectRefineFailure,
		network.SpecialEffectRefineSuccess,
		network.SpecialEffectPharmacySuccess,
		network.SpecialEffectPharmacyFailure:
		return true
	default:
		return false
	}
}

func (m *WorldMode) replaceCraftingResultEffect(ctx client.Context, notify network.SpecialEffectNotify, effectID int, spec effects.Spec) {
	for _, activeID := range craftingResultEffectGroup(effectID) {
		m.effects().Remove(activeID, notify.AID)
	}
	if m.effects().Add(ctx, effectID, notify.AID) {
		glog.Debugf("special effect actor=%d special=%d effect=%d sfx=%v restart=true", notify.AID, notify.EffectID, effectID, spec.Sfx)
	}
}

func craftingResultEffectGroup(effectID int) []int {
	switch effectID {
	case effects.RefineOK, effects.RefineFail:
		return []int{effects.RefineOK, effects.RefineFail}
	case effects.PharmacyOK, effects.PharmacyFail:
		return []int{effects.PharmacyOK, effects.PharmacyFail}
	default:
		return []int{effectID}
	}
}

func skillFailMessage(ack network.SkillFailAck) string {
	if ack.Cause == 0 {
		if messages, ok := skillFailMessagesBySkill[ack.SkillID]; ok {
			if message, ok := messages[ack.Number]; ok {
				return message
			}
		}
	}
	if message, ok := skillFailMessagesByCause[ack.Cause]; ok {
		return message
	}
	return "Action failed."
}

var skillFailMessagesBySkill = map[uint16]map[uint32]string{
	1: {
		0: "Basic skill failed.",
		1: "Cannot use emotions.",
		2: "Cannot sit.",
		3: "Cannot chat.",
		4: "Cannot form a party.",
		5: "Cannot shout.",
		6: "Cannot PK.",
		7: "Cannot align.",
	},
	50: {
		0: "Steal failed.",
	},
}

var skillFailMessagesByCause = map[uint8]string{
	0: "Action failed.",
	1: "Not enough SP.",
	2: "Not enough HP.",
	4: "Action is still on cooldown.",
	5: "Not enough Zeny.",
	9: "Too much weight.",
}

func (m *WorldMode) addSkillCastEffects(ctx client.Context, skillID uint16, property uint32, sourceID, targetID uint32, cellX, cellY int, duration time.Duration, starts time.Time, source string) {
	if duration <= 0 || sourceID == 0 {
		return
	}
	if targetID == 0 && (cellX != 0 || cellY != 0) {
		markerSize := skillCastGroundSampleSize(skillID)
		markerRotationSpeed := skillCastGroundSampleRotationRadiansPerSecond(skillID)
		if m.effects().AddCellRotationIfMissing(ctx, effects.GroundSample, 0, cellX, cellY, starts, duration, markerSize, markerRotationSpeed) {
			glog.Debugf("skill cast ground marker source=%s skill=%d src=%d cell=%d,%d delay_ms=%d", source, skillID, sourceID, cellX, cellY, duration.Milliseconds())
		}
	}
	if m.effects().AddBetweenIfMissing(ctx, effects.CastRing, sourceID, 0, starts, duration) {
		glog.Debugf("skill cast circle source=%s skill=%d src=%d target=%d delay_ms=%d", source, skillID, sourceID, targetID, duration.Milliseconds())
	}
	if targetID != 0 && targetID != sourceID {
		if m.effects().AddBetweenIfMissing(ctx, effects.LockOnTarget, targetID, 0, starts, duration) {
			glog.Debugf("skill cast target lockon source=%s skill=%d src=%d target=%d delay_ms=%d", source, skillID, sourceID, targetID, duration.Milliseconds())
		}
	}
	if skillID == db.SkillACChargearrow || skillID == db.SkillSNFalconassault || skillID == db.SkillBaMusicalstrike || skillID == db.SkillDCThrowarrow ||
		(skillID >= db.SkillBSIron && skillID <= db.SkillBSMaximize && skillID != db.SkillBSRepairweapon) || skillID == db.SkillBSAdrenaline2 ||
		skillID == db.SkillWSMeltdown || skillID == db.SkillWSCartboost || skillID == db.SkillWSOverthrustmax ||
		skillID == db.SkillWSCarttermination || skillID == db.SkillWSWeaponrefine ||
		skillID == db.SkillAMDemonstration || skillID == db.SkillAMSpheremine || skillID == db.SkillAMPharmacy || skillID == db.SkillCRCultivation || skillID == db.SkillHWMagicpower || skillID == db.SkillPFFogwall {
		m.effects().AddAt(ctx, effects.BashBegin, sourceID, starts)
		return
	}
	if skillHidesCastAura(skillID) {
		return
	}
	if skillID == db.SkillMOExtremityfist {
		effectID := effects.BeginAsura
		if actor, ok, _ := scene.ActorForCombatID(ctx, sourceID); ok && actor.Job == db.JobMonkH {
			effectID = effects.BeginAsura11
		}
		// The glyph sequence has its own timing, independent of cast speed.
		m.effects().AddAt(ctx, effectID, sourceID, starts)
		return
	}
	effectID := skillCastAuraEffectID(property)
	if skillID == db.SkillLKSpiralpierce {
		effectID = effects.PierceBody
	}
	if skillID == db.SkillKNChargeatk || skillID == db.SkillPaPressure || skillID == db.SkillPaShieldchain ||
		skillID == db.SkillHPAssumptio || skillID == db.SkillHPBasilica || skillID == db.SkillPRRedemptio ||
		skillID == db.SkillSNSharpshooting || skillID == db.SkillSNWindwalk || skillID == db.SkillBaPangvoice ||
		skillID == db.SkillCGArrowvulcan || skillID == db.SkillCGTarotcard || skillID == db.SkillHWGanbantein {
		effectID = effects.BeginSpell6
	}
	switch skillID {
	case db.SkillTKRun, db.SkillTKSevenwind, db.SkillTKMission, db.SkillSGFeel, db.SkillSGHate, db.SkillSGFusion, db.SkillNJHuuma, db.SkillNJBunsinjyutsu, db.SkillNJNen:
		effectID = effects.BlueCasting
	case db.SkillACConcentration:
		effectID = effects.IncAgiDex
	case db.SkillSTChasewalk:
		m.effects().AddAt(ctx, effects.CastSpin, sourceID, starts)
		return
	case db.SkillSTPreserve:
		effectID = effects.BeginSpell7
	case db.SkillSTRejectsword, db.SkillSTFullstrip:
		effectID = effects.BashBegin
	case db.SkillASCBreaker, db.SkillASCMeteorassault:
		effectID = effects.BeginSpell7
	case db.SkillASCEdp, db.SkillASCCdp:
		effectID = effects.BashBegin
	case db.SkillAMCallhomun, db.SkillAMRest, db.SkillAMResurrecthomun:
		effectID = effects.CoupleCasting
	case db.SkillAMTwilight1, db.SkillAMTwilight2, db.SkillAMTwilight3:
		effectID = effects.Twilight1 + int(skillID-db.SkillAMTwilight1)
	case db.SkillSAElementwater:
		effectID = effects.BeginSpell2
	case db.SkillSAElementfire:
		effectID = effects.BeginSpell3
	case db.SkillSAElementground:
		effectID = effects.BeginSpell4
	case db.SkillSAElementwind:
		effectID = effects.BeginSpell5
	}
	if _, ok := gunslingerCastAction(skillID, starts); ok {
		effectID = effects.BlueCasting
	}
	if db.IsSoulLinkerActiveSkill(skillID) {
		effectID = effects.BlueCasting
	}
	if effectID <= 0 {
		return
	}
	if m.effects().AddBetweenIfMissing(ctx, effectID, sourceID, targetID, starts, duration) {
		glog.Debugf("skill cast aura source=%s skill=%d src=%d target=%d property=%d effect=%d delay_ms=%d", source, skillID, sourceID, targetID, property, effectID, duration.Milliseconds())
	}
}

func skillCastGroundSampleSize(skillID uint16) float64 {
	if size, ok := db.SkillGroundCastClientScopeSize(skillID); ok {
		return size
	}
	return db.SkillGroundCastSize(skillID, 0)
}

func skillCastGroundSampleRotationRadiansPerSecond(skillID uint16) float64 {
	speed, _ := db.SkillGroundCastClientRotationRadiansPerSecond(skillID)
	return speed
}

type itemEffectSpec struct {
	effectIDs         []int
	effectIDsOnCaster []int
	target            itemEffectTarget
}

type itemEffectTarget uint8

const (
	itemEffectTargetAck itemEffectTarget = iota
	itemEffectTargetMercenary
)

// This mirrors reference client's DB/Items/ItemEffect.js shape. Keep item effects as
// data so future imports can preserve array-valued effectId/effectIdOnCaster.
var itemEffectSpecs = buildItemEffects()

func buildItemEffects() map[uint16]itemEffectSpec {
	specs := map[uint16]itemEffectSpec{}
	add := func(effectID int, itemIDs ...uint16) {
		for _, itemID := range itemIDs {
			itemEffect := specs[itemID]
			itemEffect.effectIDs = append(itemEffect.effectIDs, effectID)
			specs[itemID] = itemEffect
		}
	}
	target := func(target itemEffectTarget, itemIDs ...uint16) {
		for _, itemID := range itemIDs {
			itemEffect := specs[itemID]
			itemEffect.target = target
			specs[itemID] = itemEffect
		}
	}
	add(effects.PotionRed,
		501, 507, 512, 513, 515, 516, 545, 549, 557, 562, 563, 564, 565, 566, 567, 568, 569, 570, 571, 572,
		574, 575, 576, 577, 578, 579, 580, 581, 583, 584, 585, 586, 587, 588, 589, 590, 591, 592, 593, 594,
		595, 596, 597, 598, 607, 608, 663, 669, 680, 685, 11505, 11507, 11508, 11509, 11510, 11511, 11512,
		11513, 11514, 11515, 11516, 11517, 11519, 11520, 11521, 11522, 11525, 11526, 11527, 11528, 11529,
		11530, 11531, 11532, 11533, 11534, 11536, 11537, 11538, 11539, 11540, 11541, 11542, 11543, 11544,
		11545, 11546, 11547, 11550, 11551, 11552, 11553, 11554, 11555, 11567, 11568, 11570, 11577, 11578,
		11580, 11581, 11582, 11588, 11589, 11590, 11592, 11596, 11597, 11598, 11599, 11600, 11602, 11605,
		11701, 11702, 11703, 11704, 11705, 11706, 11707, 11708, 11709, 11710, 11711, 11712, 11713, 12021,
		12022, 12046, 12047, 12048, 12051, 12052, 12053, 12056, 12057, 12058, 12061, 12063, 12066, 12067,
		12068, 12101, 12102, 12131, 12133, 12188, 12192, 12195, 12196, 12197, 12202, 12203, 12204, 12205,
		12206, 12207, 12226, 12227, 12228, 12229, 12230, 12231, 12233, 12234, 12245, 12257, 12271, 12274,
		12275, 12292, 12293, 12331, 12332, 12335, 12336, 12337, 12436, 12601, 12624, 12704, 12709, 12711,
		12717, 12718, 12719, 12720, 12721, 12722, 12723, 12724, 12734, 12735, 12736, 12737, 12738, 14522,
		14523, 14524, 14551, 14552, 14553, 14575, 14576, 14577, 14578, 14579, 14580, 14672, 14673, 22567,
		22568, 22624, 22657, 22658, 22659, 22686, 22770, 22771, 22772, 22773, 22774, 22775, 22776, 22985)
	add(effects.PotionOrange, 502, 582, 599, 11506, 11569)
	add(effects.PotionYellow, 503, 508, 546, 11500, 11523, 11566, 11574, 11594)
	add(effects.PotionWhite, 504, 509, 547, 11501, 11503, 11524, 11548, 11557, 11558, 11565, 11573, 12428)
	add(effects.PotionBlue, 505, 510, 514, 11502, 11504, 11518, 11549, 11572, 11593)
	add(effects.PotionGreen, 506, 511, 11571, 11595)
	add(effects.Food,
		517, 518, 519, 520, 521, 522, 523, 525, 526, 528, 529, 530, 531, 532, 534, 535, 536, 537, 538, 539,
		540, 541, 542, 543, 544, 548, 550, 551, 552, 553, 554, 555, 556, 12017, 12184, 12185, 12298, 12404,
		12422, 12423, 12424, 12425, 12426, 12427, 12459, 12460, 12461, 12462, 12463, 12464, 12465, 12515,
		12516, 12517, 12518, 12519, 12520, 12521, 12529, 12531, 12648, 12676, 12679, 12680, 12683, 12684,
		12774, 12831, 14534, 14535, 14537, 14541, 14542, 14543, 14544, 14600, 14614, 16254, 16481, 22546)
	add(effects.FoodBlue, 533)
	add(effects.FoodChocolate, 558, 559, 560, 561, 573, 11535, 11583, 11584, 11585, 11586, 11587, 12062, 12322, 22555, 22556, 22557)
	add(effects.ItemFast, 645, 12241, 17263, 22542)
	add(effects.ItemFast2, 656, 12242, 17264, 22544)
	add(effects.ItemFast3, 657, 12243, 17265, 22543)
	add(effects.ItemSlow, 12016, 22545)
	add(effects.BoxThunder, 12028)
	add(effects.BoxResentment, 12030)
	add(effects.BoxDrowsiness, 12031)
	add(effects.BoxSunlight, 12033)
	add(effects.ResistPotion, 12118, 12119, 12120, 12121, 12299)
	add(effects.StatFoodSTR, 12041, 12042, 12043, 12044, 12045, 12071, 12072, 12073, 12074, 12075)
	add(effects.StatFoodINT, 12049, 12050, 12076, 12077, 12078, 12079, 12080, 14554, 14555, 14556)
	add(effects.StatFoodVIT, 12054, 12055, 12081, 12082, 12083, 12084, 12085, 14557, 14558, 14559)
	add(effects.StatFoodAGI, 12059, 12060, 12086, 12087, 12088, 12089, 12090, 14560, 14561, 14562)
	add(effects.StatFoodDEX, 12064, 12065, 12091, 12092, 12093, 12094, 12095, 14563, 14564, 14565)
	add(effects.StatFoodLUK, 12069, 12070, 12096, 12097, 12098, 12099, 12100, 14566, 14567, 14568)
	add(effects.Firecracker, 12018)
	add(effects.Firecracker7, 12326)
	add(effects.Firecracker1, 12788)
	add(effects.Firecracker2, 14546)
	add(effects.Firecracker3, 14547)
	add(effects.Firecracker4, 14548)
	add(effects.Firecracker5, 14549)
	add(effects.Firecracker6, 14550)
	add(effects.ItemAccel, 662, 12262)

	// reference client does not route Butterfly Wing through ItemEffect; keep Goro's
	// local map-change anticipation here but use the same table shape.
	add(effects.Teleportation, 602)

	// eAthena/rAthena send the normal item-use ACK with the player's AID even
	// though these scripts apply to sd->md. Keep the robr effect IDs above, but
	// bind their visual target to the current mercenary when one is visible.
	target(itemEffectTargetMercenary, 12184, 12185, 12241, 12242, 12243)
	return specs
}

func itemUseEffectSpec(itemID uint16) itemEffectSpec {
	return itemEffectSpecs[itemID]
}

type skillEffectSpec struct {
	effectTiming           db.SkillEffectTiming
	effectIDs              []int
	effectIDsOnCaster      []int
	beforeHitEffectIDs     []int
	beforeHitEffectIDsSelf []int
	hitEffectIDs           []int
	hitEffectIDsOnCaster   []int
	successEffectIDs       []int
	successEffectIDsSelf   []int
	beginCastEffectIDs     []int
	groundEffectIDs        []int
	hideCastBar            bool
	hideCastAura           bool
	action                 skillActionSpec
	forceGroundTarget      bool
	forceSelfTarget        bool
	passive                bool
}

func skillEffectSpecFor(skillID uint16) skillEffectSpec {
	return importedSkillEffectSpec(skillID)
}

func importedSkillEffectSpec(skillID uint16) skillEffectSpec {
	out := skillEffectSpec{}
	if spec, ok := db.SkillEffects[skillID]; ok {
		out.effectTiming = spec.EffectTiming
		out.effectIDs = copyIntSlice(spec.EffectIDs)
		out.effectIDsOnCaster = copyIntSlice(spec.EffectIDsOnCaster)
		out.beforeHitEffectIDs = copyIntSlice(spec.BeforeHitEffectIDs)
		out.beforeHitEffectIDsSelf = copyIntSlice(spec.BeforeHitEffectIDsSelf)
		out.hitEffectIDs = copyIntSlice(spec.HitEffectIDs)
		out.hitEffectIDsOnCaster = copyIntSlice(spec.HitEffectIDsOnCaster)
		out.successEffectIDs = copyIntSlice(spec.SuccessEffectIDs)
		out.successEffectIDsSelf = copyIntSlice(spec.SuccessEffectIDsSelf)
		out.beginCastEffectIDs = copyIntSlice(spec.BeginCastEffectIDs)
		out.groundEffectIDs = copyIntSlice(spec.GroundEffectIDs)
		out.hideCastBar = spec.HideCastBar
		out.hideCastAura = spec.HideCastAura
	}
	if action, ok := importedSkillActionSpec(skillID); ok {
		out.action = action
	}
	return out
}

func copyIntSlice(in []int) []int {
	if len(in) == 0 {
		return nil
	}
	return append([]int(nil), in...)
}

func skillSuccessEffectIDs(skillID uint16) []int {
	return skillEffectSpecFor(skillID).successEffectIDs
}

func skillSuccessEffectSelfIDs(skillID uint16) []int {
	return skillEffectSpecFor(skillID).successEffectIDsSelf
}

func skillEffectIDs(skillID uint16) []int {
	return skillEffectSpecFor(skillID).effectIDs
}

func skillEffectOnCasterIDs(skillID uint16) []int {
	return skillEffectSpecFor(skillID).effectIDsOnCaster
}

func skillBeginEffectIDs(skillID uint16) []int {
	return skillEffectSpecFor(skillID).beginCastEffectIDs
}

func skillBeforeHitEffectIDs(skillID uint16) []int {
	return skillEffectSpecFor(skillID).beforeHitEffectIDs
}

func skillBeforeHitEffectSelfIDs(skillID uint16) []int {
	return skillEffectSpecFor(skillID).beforeHitEffectIDsSelf
}

func skillGroundEffectIDs(skillID uint16) []int {
	return skillEffectSpecFor(skillID).groundEffectIDs
}

func skillHidesCastBar(skillID uint16) bool {
	return skillEffectSpecFor(skillID).hideCastBar
}

func skillHidesCastAura(skillID uint16) bool {
	return skillEffectSpecFor(skillID).hideCastAura
}

func skillForcesGroundTarget(skillID uint16) bool {
	return skillEffectSpecFor(skillID).forceGroundTarget
}

func skillForcesSelfTarget(skillID uint16) bool {
	return skillEffectSpecFor(skillID).forceSelfTarget
}

func skillForcesPassive(skillID uint16) bool {
	return skillEffectSpecFor(skillID).passive
}

type skillUnitEffectSpec struct {
	effectIDs []int
}

// Mostly mirrors reference client's DB/Skills/SkillUnit.js: unit id -> effect id.
var skillUnitEffectSpecs = map[uint16]skillUnitEffectSpec{
	126: {effectIDs: []int{effects.SafetyWall}},             // UNT_SAFETYWALL -> EF_GLASSWALL2
	127: {effectIDs: []int{effects.FireWall}},               // UNT_FIREWALL -> EF_FIREWALL
	128: {effectIDs: []int{effects.Portal}},                 // UNT_WARPPORTAL -> EF_PORTAL2
	129: {effectIDs: []int{effects.ReadyPortal}},            // UNT_PRE_WARPPORTAL -> EF_READYPORTAL2
	131: {effectIDs: []int{effects.BottomSanc}},             // UNT_SANCTUARY -> EF_BOTTOM_SANC
	132: {effectIDs: []int{effects.BottomMagnus}},           // UNT_MAGNUS -> EF_BOTTOM_MAG
	133: {effectIDs: []int{effects.Pneuma}},                 // UNT_PNEUMA -> EF_PNEUMA
	135: {effectIDs: []int{effects.FirePillarOn}},           // UNT_FIREPILLAR_WAITING -> EF_FIREPILLARON
	136: {effectIDs: []int{effects.FirePillarBomb}},         // UNT_FIREPILLAR_ACTIVE -> EF_FIREPILLARBOMB
	141: {effectIDs: []int{effects.IceWall}},                // UNT_ICEWALL -> EF_ICEWALL
	142: {effectIDs: []int{effects.Quagmire}},               // UNT_QUAGMIRE -> EF_QUAGMIRE
	146: {effectIDs: []int{effects.VenomDust2}},             // UNT_VENOMDUST -> EF_VENOMDUST2
	154: {effectIDs: []int{effects.BottomVolcano}},          // UNT_VOLCANO -> EF_BOTTOM_VO
	155: {effectIDs: []int{effects.BottomDeluge}},           // UNT_DELUGE -> EF_BOTTOM_DE
	156: {effectIDs: []int{effects.BottomViolent}},          // UNT_VIOLENTGALE -> EF_BOTTOM_VI
	157: {effectIDs: []int{effects.BottomLand}},             // UNT_LANDPROTECTOR -> EF_BOTTOM_LA
	158: {effectIDs: []int{effects.BottomLullabyGround}},    // UNT_LULLABY -> 278_ground
	159: {effectIDs: []int{effects.BottomRichKimGround}},    // UNT_RICHMANKIM -> 279_ground
	160: {effectIDs: []int{effects.BottomChaosGround}},      // UNT_ETERNALCHAOS -> 280_ground
	161: {effectIDs: []int{effects.BottomDrumGround}},       // UNT_DRUMBATTLEFIELD -> 281_ground
	162: {effectIDs: []int{effects.BottomNibelungGround}},   // UNT_RINGNIBELUNGEN -> 282_ground
	163: {effectIDs: []int{effects.BottomRokiGround}},       // UNT_ROKISWEIL -> 283_ground
	164: {effectIDs: []int{effects.BottomAbyssGround}},      // UNT_INTOABYSS -> 284_ground
	165: {effectIDs: []int{effects.BottomSiegGround}},       // UNT_SIEGFRIED -> 285_ground
	166: {effectIDs: []int{effects.BottomDissonanceGround}}, // UNT_DISSONANCE -> 277_ground
	167: {effectIDs: []int{effects.BottomWhistleGround}},    // UNT_WHISTLE -> 286_ground
	168: {effectIDs: []int{effects.BottomSinXGround}},       // UNT_ASSASSINCROSS -> 287_ground
	169: {effectIDs: []int{effects.BottomBragiGround}},      // UNT_POEMBRAGI -> 288_ground
	170: {effectIDs: []int{effects.BottomAppleGround}},      // UNT_APPLEIDUN -> 289_ground
	171: {effectIDs: []int{effects.BottomUglyDanceGround}},  // UNT_UGLYDANCE -> 290_ground
	172: {effectIDs: []int{effects.BottomHummingGround}},    // UNT_HUMMING -> 291_ground
	173: {effectIDs: []int{effects.BottomForgetGround}},     // UNT_DONTFORGETME -> 292_ground
	174: {effectIDs: []int{effects.BottomFortuneGround}},    // UNT_FORTUNEKISS -> 293_ground
	175: {effectIDs: []int{effects.BottomServiceGround}},    // UNT_SERVICEFORYOU -> 294_ground
	177: {effectIDs: []int{effects.Demonstration}},          // UNT_DEMONSTRATION -> EF_DEMONSTRATION
	179: {effectIDs: []int{effects.GospelGround}},           // UNT_GOSPEL -> 370_ground
	180: {effectIDs: []int{effects.BottomBasilica}},         // UNT_BASILICA -> EF_BOTTOM_BASILICA
	181: {},                                                 // UNT_MOONLIT: the caster owns the sphere.
	182: {effectIDs: []int{effects.FogWallGround}},          // UNT_FOGWALL -> EF_BOTTOM_FOGWALL
	183: {effectIDs: []int{effects.BottomSpider}},           // UNT_SPIDERWEB -> EF_BOTTOM_SPIDER
	184: {effectIDs: []int{effects.Gravitation}},            // UNT_GRAVITATION -> EF_GRAVITATION
	185: {effectIDs: []int{effects.BottomHermode}},          // UNT_HERMODE -> EF_BOTTOM_HERMODE
	187: {effectIDs: []int{effects.BottomSuiton}},           // UNT_SUITON
	188: {effectIDs: []int{effects.Tatami}},                 // UNT_TATAMIGAESHI -> EF_TATAMI
	189: {effectIDs: []int{effects.Kaen}},                   // UNT_KAEN -> EF_KAEN
	190: {effectIDs: []int{effects.LightningS}},             // UNT_GROUNDDRIFT_LIGHTNING
	191: {effectIDs: []int{effects.BlindS}},                 // UNT_GROUNDDRIFT_BLIND
	192: {effectIDs: []int{effects.PoisonS}},                // UNT_GROUNDDRIFT_POISON
	193: {effectIDs: []int{effects.FreezingS}},              // UNT_GROUNDDRIFT_FREEZING
	194: {effectIDs: []int{effects.FlareS}},                 // UNT_GROUNDDRIFT_FLARE
	199: {effectIDs: []int{effects.BottomEvilLand}},         // UNT_EVILLAND -> EF_BOTTOM_EVILLAND
	202: {effectIDs: []int{effects.GlassWall3}},             // UNT_EPICLESIS -> EF_GLASSWALL3
	203: {effectIDs: []int{effects.EarthWall}},              // UNT_EARTHSTRAIN -> EF_EARTHWALL
	204: {effectIDs: []int{effects.BottomManhole}},          // UNT_MANHOLE -> EF_BOTTOM_MANHOLE
	205: {effectIDs: []int{effects.ForestLight6}},           // UNT_DIMENSIONDOOR -> EF_FORESTLIGHT6
	206: {effectIDs: []int{effects.BottomAni}},              // UNT_CHAOSPANIC -> EF_BOTTOM_ANI
	207: {effectIDs: []int{effects.BottomMaelstrom}},        // UNT_MAELSTROM -> EF_BOTTOM_MAELSTROM
	208: {effectIDs: []int{effects.BottomBloodyLust}},       // UNT_BLOODYLUST -> EF_BOTTOM_BLOODYLUST
	218: {effectIDs: []int{effects.BotReverb}},              // UNT_REVERBERATION -> EF_BOT_REVERB
	220: {effectIDs: []int{effects.FireWall2}},              // UNT_FIREWALK -> EF_FIREWALL2
	221: {effectIDs: []int{effects.Shockwave2}},             // UNT_ELECTRICWALK -> EF_SHOCKWAVE2
	222: {effectIDs: []int{effects.BotReverb2}},             // UNT_NETHERWORLD -> EF_BOT_REVERB2
}

func skillUnitEffectIDs(unitID uint16) []int {
	return skillUnitEffectSpecs[unitID].effectIDs
}

func skillCastAuraEffectID(property uint32) int {
	switch property {
	case 1:
		return effects.BeginSpell2
	case 2:
		return effects.BeginSpell4
	case 3:
		return effects.BeginSpell3
	case 4:
		return effects.BeginSpell5
	case 5:
		return effects.BeginSpell7
	case 6:
		return effects.BeginSpell6
	default:
		return effects.BeginSpell
	}
}

func skillHitEffectIDs(skillID uint16) []int {
	return skillEffectSpecFor(skillID).hitEffectIDs
}

func skillHitEffectOnCasterIDs(skillID uint16) []int {
	return skillEffectSpecFor(skillID).hitEffectIDsOnCaster
}

func (m *WorldMode) applyAlchemistSkill(ctx client.Context, notify network.SkillNoDamageNotify, now time.Time) {
	effectID, level := effects.ThrowItem2, int(notify.Amount)
	switch notify.SkillID {
	case db.SkillAMPotionpitcher:
	case db.SkillAMBerserkpitcher:
		effectID, level = effects.ThrowItem5, 4
	default:
		return
	}
	if m.effects().AddBetween(ctx, effectID, notify.TargetID, notify.SourceID, now) {
		m.effects().Active[len(m.effects().Active)-1].ItemLevel = level
	}
}

func isBardPerformance(skillID uint16) bool {
	return skillID >= db.SkillBDLullaby && skillID <= db.SkillBDSiegfried ||
		skillID == db.SkillBaDissonance || skillID >= db.SkillBaWhistle && skillID <= db.SkillBaAppleidun
}

// Body effects reuse the composed sprite, including its appearance and scale.
// They never mutate the cached image or the actor's authoritative position.
func (m *WorldMode) drawActorSpriteWithEffects3D(ctx client.Context, screen *render.Frame, projection scene.Projection, entry scene.ActorDrawEntry, billboard *scene.Billboard, scale, visibility, shadow float64, tint color.RGBA, blend render.Blend, now time.Time) {
	m.effects().DrawActor(ctx, screen, projection, entry, billboard, scale, visibility, shadow, tint, blend, now, m.starFusionHeight(entry.Actor, now))
}

func (m *WorldMode) skillEffectDirection(ctx client.Context, actorID uint32, direction int, now time.Time) int {
	for _, effect := range m.effects().Active {
		if (effect.EffectID != effects.FalconAssault && effect.EffectID != effects.CastSpin && effect.EffectID != effects.StormKick) || now.Before(effect.Starts) || effect.Expired(now) {
			continue
		}
		if effect.ActorID != actorID && !(scene.IsLocalActor(ctx, actorID) && scene.IsLocalActor(ctx, effect.ActorID)) {
			continue
		}
		frame := int(now.Sub(effect.Starts) / db.EffectFrameDuration)
		turn := 0.0
		switch effect.EffectID {
		case effects.StormKick:
			if frame > 47 {
				continue
			}
			turn = float64(frame+1) * 30
		case effects.FalconAssault:
			if frame < 30 || frame > 54 {
				continue
			}
			turn = float64(frame-29) * 30
		case effects.CastSpin:
			anim, ok := m.actorAnims[effect.ActorID]
			if frame >= 100 || !ok || anim.actionFamily != scene.SpriteActionIdle || !anim.started.Equal(effect.Starts) || now.Sub(anim.started) >= anim.duration {
				continue
			}
			ticks := frame + 1
			turn = float64(12*min(ticks, 16) + 18*scene.ClampInt(ticks-16, 0, 15) + 24*scene.ClampInt(ticks-31, 0, 10) + 48*max(0, ticks-41))
		default:
			continue
		}
		direction = (direction + int(math.Floor((turn+22.5)/45))) % 8
	}
	return direction
}

func (m *WorldMode) applyClownGypsySkill(ctx client.Context, notify network.SkillNoDamageNotify, now time.Time) {
	source, target := notify.SourceID, notify.TargetID
	if scene.IsLocalActor(ctx, source) {
		source = scene.LocalSkillTarget(ctx)
	}
	if scene.IsLocalActor(ctx, target) {
		target = scene.LocalSkillTarget(ctx)
	}
	switch notify.SkillID {
	case db.SkillCGMoonlit:
		if _, ok, _ := scene.ActorForCombatID(ctx, source); ok {
			m.setActorStateEffect(ctx, source, effects.Moonlit, true)
		}
	case db.SkillCGMarionette:
		if _, ok, _ := scene.ActorForCombatID(ctx, source); !ok {
			return
		}
		if _, ok, _ := scene.ActorForCombatID(ctx, target); !ok {
			return
		}
		m.effects().RemoveMarionetteLinks(source)
		if m.effects().AddBetween(ctx, effects.MarionetteLink, source, target, now) {
			effect := &m.effects().Active[len(m.effects().Active)-1]
			effect.Persistent, effect.Expires = true, time.Time{}
		}
		m.setActorStateEffect(ctx, source, effects.PinkBody, true)
		m.setActorStateEffect(ctx, target, effects.PinkBody, true)
	}
}

// The initial shot uses ACT 88. Each additional hit restarts the normal
// weapon attack, as AM_WILL_BE_ATTACK does in the original client.
func (m *WorldMode) startArrowVulcanHits(ctx client.Context, action network.ActorActionNotify, hitAt time.Time) {
	if action.SkillID != db.SkillCGArrowvulcan || skillSplashAction(action) || actionVisualHitCount(action) < 2 {
		return
	}
	anim, ok := m.actorAnims[action.SourceID]
	source, sourceOK, _ := scene.ActorForCombatID(ctx, action.SourceID)
	if !ok || !sourceOK {
		return
	}
	anim.speed = m.actorActionFrameDelay(ctx, source, anim.actionFamily, anim.duration)
	anim.hasSpeed = anim.speed > 0
	count := min(9, actionVisualHitCount(action))
	next := readyFightAnimation(hitAt.Add(time.Duration(count) * scene.MultiHitDelay))
	for hit := count - 1; hit >= 1; hit-- {
		at := hitAt.Add(time.Duration(hit) * scene.MultiHitDelay)
		rest := actorAnimation{
			actionFamily: attackActionFamilyForActorWithResources(ctx.Resources, source),
			started:      at.Add(24 * time.Millisecond), duration: scene.MultiHitDelay - 24*time.Millisecond,
			play: true, hasPlay: true, speed: 24 * time.Millisecond, hasSpeed: true,
			frameOffset: 1, hasFrameOffset: true, next: next,
		}
		next = &actorAnimation{
			actionFamily: rest.actionFamily, started: at, duration: 24 * time.Millisecond,
			hasPlay: true, hasFixedMotion: true, fixedMotion: 4, next: &rest,
		}
	}
	anim.duration = next.started.Sub(anim.started)
	anim.next = next
	m.setCombatActorAction(ctx, action.SourceID, anim)
}

func (m *WorldMode) applyCreatorGroundSkill(ctx client.Context, notify network.GroundSkillNotify, now time.Time) bool {
	if notify.SkillID != db.SkillCRSlimpitcher && notify.SkillID != db.SkillCRCultivation {
		return false
	}
	m.faceSkillSource(ctx, notify.SourceID, 0, int(notify.X), int(notify.Y))
	if actor, ok, _ := scene.ActorForCombatID(ctx, notify.SourceID); ok {
		m.startSkillActionAnimation(ctx, notify.SourceID, actor, skillAction(notify.SkillID), now, scene.DefaultAttackAnimationDuration)
	}
	m.applySkillNameBubble(ctx, notify.SourceID, notify.SkillID, now)
	if notify.SkillID == db.SkillCRCultivation {
		m.effects().AddAt(ctx, effects.BeginSpell6, notify.SourceID, now)
		return true
	}
	effectID := effects.Slim
	if notify.Level >= 10 {
		effectID = effects.Slim3
	} else if notify.Level >= 6 {
		effectID = effects.Slim2
	}
	m.effects().AddCellLifetime(ctx, effectID, 0, int(notify.X), int(notify.Y), now, 0, false)
	// The native sound is centered at a fixed distance of 150, with a 40-unit minimum.
	m.scheduleSoundVolume(now, 40.0/150, "assulter_attack.wav")
	return true
}

func isDancerPerformance(skillID uint16) bool {
	return skillID == db.SkillDCUglydance || skillID >= db.SkillDCHumming && skillID <= db.SkillDCServiceforyou
}

func (m *WorldMode) applyWinkCharm(ctx client.Context, notify network.SkillNoDamageNotify) {
	if notify.SkillID != db.SkillDCWinkcharm {
		return
	}
	if _, ok, _ := scene.ActorForCombatID(ctx, notify.SourceID); !ok {
		return
	}
	if _, ok, _ := scene.ActorForCombatID(ctx, notify.TargetID); !ok {
		return
	}
	m.applyEmotionNotify(ctx, network.EmotionNotify{GID: notify.SourceID, Type: 30}) // ET_CHUP
}

func (m *WorldMode) addGunslingerBeforeHitEffect(ctx client.Context, action network.ActorActionNotify, starts time.Time) bool {
	if action.SkillID < db.SkillGSGlittering || action.SkillID > db.SkillGSGrounddrift {
		return false
	}
	for _, id := range skillBeforeHitEffectIDs(action.SkillID) {
		m.effects().AddBetween(ctx, id, action.TargetID, action.SourceID, starts)
	}
	return true
}

func (m *WorldMode) applyHighWizardSkill(ctx client.Context, notify network.SkillNoDamageNotify, now time.Time) {
	switch notify.SkillID {
	case db.SkillHWSouldrain:
		m.effects().AddBetween(ctx, effects.EnergyDrain2, notify.SourceID, notify.TargetID, now)
	case db.SkillHWMagicpower:
		id := notify.TargetID
		if scene.IsLocalActor(ctx, id) {
			id = scene.LocalSkillTarget(ctx)
		}
		if _, ok, _ := scene.ActorForCombatID(ctx, id); ok {
			m.effects().Remove(effects.LightBlade, id)
			m.setActorStateEffect(ctx, id, effects.LightBlade, true)
			m.scheduleSound(now, "effect\\마법력 증폭.wav")
		}
	}
}

func (m *WorldMode) applyHunterTrapSkill(ctx client.Context, notify network.SkillNoDamageNotify, now time.Time) bool {
	if notify.SkillID != db.SkillHTRemovetrap && notify.SkillID != db.SkillHTSpringtrap {
		return false
	}
	x, y, ok := m.skillUnitCell(notify.TargetID)
	if ok {
		for _, effectID := range skillEffectIDs(notify.SkillID) {
			// The following disappearance packet must not cut off the burst.
			m.effects().AddCellLifetime(ctx, effectID, 0, x, y, now, 0, false)
		}
		if notify.SkillID == db.SkillHTSpringtrap {
			m.startFalconAttackAt(ctx, notify.SourceID, x, y, now)
		}
	}
	return true
}

func (m *WorldMode) applyHunterGroundSkill(ctx client.Context, notify network.GroundSkillNotify, now time.Time) bool {
	switch notify.SkillID {
	case db.SkillHTSkidtrap, db.SkillHTLandmine, db.SkillHTAnklesnare, db.SkillHTShockwave,
		db.SkillHTSandman, db.SkillHTFlasher, db.SkillHTFreezingtrap, db.SkillHTBlastmine,
		db.SkillHTClaymoretrap, db.SkillHTTalkiebox, db.SkillHTDetecting:
	default:
		return false
	}
	m.faceSkillSource(ctx, notify.SourceID, 0, int(notify.X), int(notify.Y))
	if actor, ok, _ := scene.ActorForCombatID(ctx, notify.SourceID); ok {
		m.startSkillActionAnimation(ctx, notify.SourceID, actor, skillAction(notify.SkillID), now, scene.DefaultAttackAnimationDuration)
	}
	m.applyFalconGroundSkillNotify(ctx, notify, now)
	for _, effectID := range skillGroundEffectIDs(notify.SkillID) {
		m.effects().AddCellLifetime(ctx, effectID, 0, int(notify.X), int(notify.Y), now, 0, false)
	}
	return true
}

func (m *WorldMode) addMageProjectileEffect(ctx client.Context, action network.ActorActionNotify, starts time.Time) bool {
	count := actionVisualHitCount(action)
	frames := 52
	switch action.SkillID {
	case db.SkillMGFireball, db.SkillMGFrostdiver:
		count = 1
	case db.SkillMGSoulstrike:
		count = min(count, 5)
		frames = 6 + (count-1)*11 + 40
	case db.SkillMGColdbolt, db.SkillMGFirebolt:
		frames = 12 + (count-1)*10 + 70
		if action.SkillID == db.SkillMGFirebolt {
			frames = max(frames, 160)
		}
	default:
		return false
	}
	id := skillBeforeHitEffectIDs(action.SkillID)[0]
	if m.effects().AddBetweenDuration(ctx, id, action.TargetID, action.SourceID, starts, time.Duration(frames)*db.EffectFrameDuration) {
		effect := &m.effects().Active[len(m.effects().Active)-1]
		effect.HitCount = count
		if action.SkillID == db.SkillMGSoulstrike {
			for i := 1; i < count; i++ {
				m.scheduleSoundAtWorldEffect(starts.Add(time.Duration(6+11*i)*db.EffectFrameDuration), *effect, "effect\\ef_soulstrike.wav")
			}
		}
	}
	return true
}

func ninjaCastingSkill(id uint16) bool {
	return id == db.SkillNJHuuma || id == db.SkillNJBunsinjyutsu || id >= db.SkillNJKouenka && id <= db.SkillNJNen
}

func (m *WorldMode) addNinjaBeforeHitEffect(ctx client.Context, action network.ActorActionNotify, starts time.Time) bool {
	if action.SkillID < db.SkillNJSyuriken || action.SkillID > db.SkillNJIssen {
		return false
	}
	for _, id := range skillBeforeHitEffectIDs(action.SkillID) {
		if !m.effects().AddBetween(ctx, id, action.TargetID, action.SourceID, starts) {
			continue
		}
		effect := &m.effects().Active[len(m.effects().Active)-1]
		if id >= effects.ThrowItem7 && id <= effects.ThrowItem10 {
			m.scheduleSound(starts, "effect\\닌자_던지기.wav")
		}
		if id == effects.Stin4 {
			effect.HitCount = actionVisualHitCount(action)
			if effect.HitCount > 36 {
				effect.HitCount = 1
			}
		}
	}
	return true
}

func (m *WorldMode) applyPerformanceGroundSkill(ctx client.Context, notify network.GroundSkillNotify, now time.Time) bool {
	if !isBardPerformance(notify.SkillID) && !isDancerPerformance(notify.SkillID) && notify.SkillID != db.SkillCGHermode && notify.SkillID != db.SkillCGMoonlit {
		return false
	}
	if actor, ok, _ := scene.ActorForCombatID(ctx, notify.SourceID); ok {
		m.startSkillActionAnimation(ctx, notify.SourceID, actor, skillAction(notify.SkillID), now, scene.DefaultAttackAnimationDuration)
	}
	for _, sound := range db.SkillEffects[notify.SkillID].SFX {
		m.scheduleSoundAtActor(now, notify.SourceID, sound)
	}
	if notify.SkillID == db.SkillCGMoonlit {
		m.applyClownGypsySkill(ctx, network.SkillNoDamageNotify{SkillID: notify.SkillID, SourceID: notify.SourceID}, now)
	}
	// Skill-unit packets own each symbol and its lifetime.
	return true
}

func (m *WorldMode) stopPerformance(ctx client.Context, change network.StatusEffectChange) {
	if change.StatusID != db.StatusBdplaying || change.Active {
		return
	}
	id := change.ActorID
	if id == 0 {
		id = scene.LocalSkillTarget(ctx)
	}
	ids := []uint32{id}
	if scene.IsLocalActor(ctx, id) {
		ids = []uint32{ctx.Session.AccountID, ctx.Session.CharID}
	}
	for _, id := range ids {
		if anim, ok := m.actorAnims[id]; ok && anim.loop && anim.actionFamily == sprites.ActionPCSkill && anim.hasFrameOffset && anim.frameOffset == 1 && anim.length == 3 {
			m.clearActorAction(ctx, id)
		}
	}
}

func (m *WorldMode) applyProfessorSkill(ctx client.Context, notify network.SkillNoDamageNotify, now time.Time) {
	switch notify.SkillID {
	case db.SkillPFSoulchange:
		m.effects().AddBetween(ctx, effects.LineLink2, notify.SourceID, notify.TargetID, now)
	case db.SkillPFDoublecasting:
		if _, ok, _ := scene.ActorForCombatID(ctx, notify.TargetID); ok {
			m.scheduleSound(now, "effect\\마법력 증폭.wav")
		}
	}
}

func (m *WorldMode) applyCloseConfineStatus(ctx client.Context, change network.StatusEffectChange) {
	if change.StatusID != db.StatusRgCconfineM && change.StatusID != db.StatusRgCconfineS {
		return
	}
	id := change.ActorID
	if id == 0 || scene.IsLocalActor(ctx, id) {
		id = scene.LocalSkillTarget(ctx)
	}
	m.setActorStateEffect(ctx, id, effects.NPCStop2, change.Active)
	if change.Active && change.HasDuration {
		now := time.Now()
		for i := range m.effects().Active {
			effect := &m.effects().Active[i]
			if effect.EffectID == effects.NPCStop2 && effect.ActorID == id && !effect.Expired(now) {
				effect.Expires = now.Add(change.Duration)
			}
		}
	}
}

func (m *WorldMode) applyStarGladiatorSkill(ctx client.Context, notify network.SkillNoDamageNotify, now time.Time) {
	id := notify.SourceID
	if scene.IsLocalActor(ctx, id) {
		id = scene.LocalSkillTarget(ctx)
	}
	switch notify.SkillID {
	case db.SkillSGFusion:
		if _, ok, _ := scene.ActorForCombatID(ctx, id); !ok {
			return
		}
		m.scheduleSound(now, "effect\\T_변신.wav")
	case db.SkillSGHate:
		if _, ok, _ := scene.ActorForCombatID(ctx, notify.TargetID); ok {
			m.scheduleSound(now, "effect\\T_등록.wav")
			m.effects().Remove(effects.Hated, notify.TargetID)
		}
	case db.SkillSGSunComfort, db.SkillSGMoonComfort, db.SkillSGStarComfort:
		m.effects().Remove(effects.FlowerCast, notify.SourceID)
		m.effects().Remove(effects.FlowerCast, id)
	case db.SkillSGSunWarm, db.SkillSGMoonWarm, db.SkillSGStarWarm:
		actor, ok, local := scene.ActorForCombatID(ctx, id)
		if !ok {
			return
		}
		actor.Opt3State |= db.Opt3Warm
		actor.HasState = true
		if local {
			ctx.World.Player.Opt3State, ctx.World.Player.HasState = actor.Opt3State, true
		} else {
			ctx.World.Actors[id] = actor
		}
		m.effects().Remove(effects.DoubleGumgang, id)
		m.effects().Remove(effects.RedLightBody, id)
		m.syncActorSpiritEffects(ctx, id)
	}
}

// The status is local, but can precede the skill list on login. Reconcile both
// packet orders, restore it on map entry, and ignore repeated status packets.
func (m *WorldMode) syncStarDevil(ctx client.Context) {
	if ctx.Session == nil || ctx.World == nil {
		return
	}
	id, wanted := scene.LocalSkillTarget(ctx), 0
	if status, ok := ctx.Session.Statuses.Active[db.StatusDevil1]; ok && (!status.HasDuration || time.Now().Before(status.ExpiresAt)) {
		if level := sessionSkillLevel(ctx.Session, db.SkillSGDevil); level >= 1 && level <= 10 {
			wanted = effects.Devil1 + level - 1
		}
	}
	now := time.Now()
	for effectID := effects.Devil1; effectID <= effects.Devil10; effectID++ {
		if effectID != wanted {
			m.effects().Remove(effectID, id)
		}
	}
	if wanted == 0 {
		m.effects().Remove(effects.BlackDevil, id)
	} else if !m.effects().HasActive(wanted, id, now) {
		m.setActorStateEffect(ctx, id, wanted, true)
		m.effects().AddAt(ctx, effects.BlackDevil, id, now)
		m.scheduleSoundVolume(now, .4, "effect\\_blind.wav")
	}
}

func (m *WorldMode) applyTaekwonSkill(ctx client.Context, notify network.SkillNoDamageNotify, now time.Time) {
	id := notify.SourceID
	if _, ok, _ := scene.ActorForCombatID(ctx, id); !ok {
		return
	}
	if scene.IsLocalActor(ctx, id) {
		id = scene.LocalSkillTarget(ctx)
	}
	switch notify.SkillID {
	case db.SkillTKRun:
		if m.effects().HasActive(effects.StopRun, id, now) {
			m.effects().AddAt(ctx, effects.StopEffect, id, now)
		} else if !m.effects().HasActive(effects.Run, id, now) {
			m.effects().AddAt(ctx, effects.Run, id, now)
		}
	case db.SkillTKHighjump:
		m.effects().Remove(effects.JumpBody, notify.SourceID)
		m.effects().Remove(effects.JumpBody, id)
		m.scheduledSounds = slices.DeleteFunc(m.scheduledSounds, func(sound scheduledSound) bool {
			return (sound.actorID == notify.SourceID || sound.actorID == id) && sound.at.After(now) && slices.Contains(sound.paths, "effect\\T_회피2.wav")
		})
	case db.SkillTKDodge:
		m.scheduleSoundVolume(now, 4.0/15, "effect\\T_낙법.wav")
	case db.SkillTKMission:
		m.scheduleSoundVolume(now, 4.0/10, "effect\\T_피링.wav")
	case db.SkillTKSevenwind:
		if notify.Amount < 1 || notify.Amount > 7 {
			return
		}
		m.scheduleSound(now, "effect\\T_바람방출.wav")
		for _, effectID := range []int{effects.StormKick3, effects.BeginAsura1 + int(notify.Amount) - 1} {
			m.effects().Remove(effectID, id)
			m.effects().AddAt(ctx, effectID, id, now)
		}
	}
}
