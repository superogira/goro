package effects

import (
	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/game/companion"
	"github.com/kivutar/goro/game/scene"
	"github.com/kivutar/goro/glog"
	"github.com/kivutar/goro/network"
)

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
	add(PotionRed,
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
	add(PotionOrange, 502, 582, 599, 11506, 11569)
	add(PotionYellow, 503, 508, 546, 11500, 11523, 11566, 11574, 11594)
	add(PotionWhite, 504, 509, 547, 11501, 11503, 11524, 11548, 11557, 11558, 11565, 11573, 12428)
	add(PotionBlue, 505, 510, 514, 11502, 11504, 11518, 11549, 11572, 11593)
	add(PotionGreen, 506, 511, 11571, 11595)
	add(Food,
		517, 518, 519, 520, 521, 522, 523, 525, 526, 528, 529, 530, 531, 532, 534, 535, 536, 537, 538, 539,
		540, 541, 542, 543, 544, 548, 550, 551, 552, 553, 554, 555, 556, 12017, 12184, 12185, 12298, 12404,
		12422, 12423, 12424, 12425, 12426, 12427, 12459, 12460, 12461, 12462, 12463, 12464, 12465, 12515,
		12516, 12517, 12518, 12519, 12520, 12521, 12529, 12531, 12648, 12676, 12679, 12680, 12683, 12684,
		12774, 12831, 14534, 14535, 14537, 14541, 14542, 14543, 14544, 14600, 14614, 16254, 16481, 22546)
	add(FoodBlue, 533)
	add(FoodChocolate, 558, 559, 560, 561, 573, 11535, 11583, 11584, 11585, 11586, 11587, 12062, 12322, 22555, 22556, 22557)
	add(ItemFast, 645, 12241, 17263, 22542)
	add(ItemFast2, 656, 12242, 17264, 22544)
	add(ItemFast3, 657, 12243, 17265, 22543)
	add(ItemSlow, 12016, 22545)
	add(BoxThunder, 12028)
	add(BoxResentment, 12030)
	add(BoxDrowsiness, 12031)
	add(BoxSunlight, 12033)
	add(ResistPotion, 12118, 12119, 12120, 12121, 12299)
	add(StatFoodSTR, 12041, 12042, 12043, 12044, 12045, 12071, 12072, 12073, 12074, 12075)
	add(StatFoodINT, 12049, 12050, 12076, 12077, 12078, 12079, 12080, 14554, 14555, 14556)
	add(StatFoodVIT, 12054, 12055, 12081, 12082, 12083, 12084, 12085, 14557, 14558, 14559)
	add(StatFoodAGI, 12059, 12060, 12086, 12087, 12088, 12089, 12090, 14560, 14561, 14562)
	add(StatFoodDEX, 12064, 12065, 12091, 12092, 12093, 12094, 12095, 14563, 14564, 14565)
	add(StatFoodLUK, 12069, 12070, 12096, 12097, 12098, 12099, 12100, 14566, 14567, 14568)
	add(Firecracker, 12018)
	add(Firecracker7, 12326)
	add(Firecracker1, 12788)
	add(Firecracker2, 14546)
	add(Firecracker3, 14547)
	add(Firecracker4, 14548)
	add(Firecracker5, 14549)
	add(Firecracker6, 14550)
	add(ItemAccel, 662, 12262)

	// reference client does not route Butterfly Wing through ItemEffect; keep Goro's
	// local map-change anticipation here but use the same table shape.
	add(Teleportation, 602)

	// eAthena/rAthena send the normal item-use ACK with the player's AID even
	// though these scripts apply to sd->md. Keep the robr effect IDs above, but
	// bind their visual target to the current mercenary when one is visible.
	target(itemEffectTargetMercenary, 12184, 12185, 12241, 12242, 12243)
	return specs
}

func itemUseEffectSpec(itemID uint16) itemEffectSpec {
	return itemEffectSpecs[itemID]
}

func (m *System) AddItemUseEffect(ctx client.Context, ack network.UseItemAck) {
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
		if EffectDetachesLocalActor(effectID) && scene.IsLocalActor(ctx, targetID) {
			targetID = 0
		}
		if m.Add(ctx, effectID, targetID) {
			glog.Debugf("item effect item=%d actor=%d effect=%d", ack.ItemID, targetID, effectID)
		}
	}
	for _, effectID := range casterEffectIDs {
		casterID := actorID
		if EffectDetachesLocalActor(effectID) && scene.IsLocalActor(ctx, casterID) {
			casterID = 0
		}
		if m.Add(ctx, effectID, casterID) {
			glog.Debugf("item caster effect item=%d actor=%d effect=%d", ack.ItemID, casterID, effectID)
		}
	}
}
