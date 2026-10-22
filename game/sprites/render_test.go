package sprites

import (
	"image/color"
	"math"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/kivutar/goro/db"
	"github.com/kivutar/goro/game/internal/testutil"
	"github.com/kivutar/goro/game/scene"
	"github.com/kivutar/goro/render"
	"github.com/kivutar/goro/res"
)

func TestNormalizeDirectionIndex(t *testing.T) {
	cases := map[int]int{
		0:  0,
		7:  7,
		8:  0,
		-1: 7,
		-9: 7,
	}
	for input, want := range cases {
		if got := scene.NormalizeDirectionIndex(input); got != want {
			t.Fatalf("normalizeDirectionIndex(%d) = %d, want %d", input, got, want)
		}
	}
}

func TestSpriteDrawFilterUsesLinearSampling(t *testing.T) {
	if got := scene.SpriteDrawFilter(); got != render.FilterLinear {
		t.Fatalf("sprite draw filter = %v, want linear", got)
	}
}

func TestSpriteCompositionFilterKeepsSourcePixelsCrisp(t *testing.T) {
	if got := scene.SpriteCompositionFilter(); got != render.FilterNearest {
		t.Fatalf("sprite composition filter = %v, want nearest", got)
	}
}

func TestSpriteViewImagePreservesStraightAlphaRGBAFrame(t *testing.T) {
	view := &scene.SpriteView{
		Spr: &res.SPR{
			RGBAIndex: 0,
			Frames: []res.SPRFrame{{
				Type:   res.SPRFrameRGBA,
				Width:  1,
				Height: 1,
				Data:   []byte{64, 200, 240, 255},
			}},
		},
		Images: make(map[scene.SpriteFrameKey]*render.Image),
	}

	img, ok := scene.SpriteViewImage(view, 0, res.SPRFrameRGBA)
	if !ok {
		t.Fatal("spriteViewImage failed")
	}
	got := img.RGBA().RGBAAt(0, 0)
	want := color.RGBA{R: 255, G: 240, B: 200, A: 64}
	if got != want {
		t.Fatalf("RGBA sprite frame = %+v, want %+v", got, want)
	}
}

func TestActorSpriteWorldZLiftsSpritesAboveTerrain(t *testing.T) {
	if got := scene.ActorSpriteWorldZ(12.5); math.Abs(got-12.7) > 1e-9 {
		t.Fatalf("actor sprite world z = %.3f, want 12.700", got)
	}
}

func TestActorShadowDepthBiasUsesTinyGroundDecalNudge(t *testing.T) {
	if scene.ActorShadowDepthBias != scene.GroundDecalDepthBias {
		t.Fatalf("actor shadow depth bias = %.10f, want shared ground decal bias %.10f", scene.ActorShadowDepthBias, scene.GroundDecalDepthBias)
	}
	if scene.ActorShadowDepthBias >= 0.001 {
		t.Fatalf("actor shadow depth bias = %.10f, want smaller than old overdraw-prone bias", scene.ActorShadowDepthBias)
	}
}

func TestSpriteDirectionFromWorldDirShowsBackForNorth(t *testing.T) {
	cases := map[int]int{
		0:  4,
		1:  3,
		2:  2,
		3:  1,
		4:  0,
		5:  7,
		6:  6,
		7:  5,
		8:  4,
		-1: 5,
	}
	for input, want := range cases {
		if got := DirectionForCamera(input, 0); got != want {
			t.Fatalf("spriteDirectionFromWorldDirForCamera(%d, 0) = %d, want %d", input, got, want)
		}
	}
}

func TestSpriteDirectionFromWorldDirAccountsForCameraYaw(t *testing.T) {
	cases := []struct {
		name      string
		direction int
		cameraYaw float64
		want      int
	}{
		{name: "outdoor north", direction: 4, cameraYaw: 0, want: 0},
		{name: "indoor north", direction: 4, cameraYaw: -45, want: 7},
		{name: "indoor south", direction: 0, cameraYaw: -45, want: 3},
		{name: "east camera north", direction: 4, cameraYaw: 90, want: 2},
		{name: "rounded yaw", direction: 4, cameraYaw: 44, want: 1},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			if got := DirectionForCamera(tt.direction, tt.cameraYaw); got != tt.want {
				t.Fatalf("spriteDirectionFromWorldDirForCamera(%d, %.1f) = %d, want %d", tt.direction, tt.cameraYaw, got, tt.want)
			}
		})
	}
}

func TestResolveSpriteActionPrefersFamilyAndDirection(t *testing.T) {
	act := &res.ACT{Actions: make([]res.ACTAction, 16)}
	act.Actions[11] = res.ACTAction{Animations: []res.ACTAnimation{{}}, DelayMS: 100}
	index, _, ok := ResolveAction(act, scene.SpriteActionWalk, 3)
	if !ok {
		t.Fatal("expected action")
	}
	if index != 11 {
		t.Fatalf("index = %d, want 11", index)
	}
}

func TestResolveSpriteActionFallsBackToFamilyBase(t *testing.T) {
	act := &res.ACT{Actions: make([]res.ACTAction, 16)}
	act.Actions[8] = res.ACTAction{Animations: []res.ACTAnimation{{}}, DelayMS: 100}
	index, _, ok := ResolveAction(act, scene.SpriteActionWalk, 4)
	if !ok {
		t.Fatal("expected action")
	}
	if index != 8 {
		t.Fatalf("index = %d, want 8", index)
	}
}

func TestResolveSpriteActionUsesDirectionForEightActionSprites(t *testing.T) {
	act := &res.ACT{Actions: make([]res.ACTAction, 8)}
	for i := range act.Actions {
		act.Actions[i] = res.ACTAction{Animations: []res.ACTAnimation{{}}, DelayMS: 100}
	}
	index, _, ok := ResolveAction(act, scene.SpriteActionWalk, 5)
	if !ok {
		t.Fatal("expected action")
	}
	if index != 5 {
		t.Fatalf("index = %d, want direction action 5", index)
	}
}

func TestResolveSpriteActionUsesDirectIndexForCompactNonPCAct(t *testing.T) {
	act := &res.ACT{Actions: make([]res.ACTAction, 5)}
	act.Actions[ActionNonPCAttack] = res.ACTAction{Animations: []res.ACTAnimation{{}}, DelayMS: 100}
	index, _, ok := ResolveAction(act, ActionNonPCAttack, 4)
	if !ok {
		t.Fatal("expected action")
	}
	if index != ActionNonPCAttack {
		t.Fatalf("index = %d, want %d", index, ActionNonPCAttack)
	}
}

func TestPreferNonPCActUpgradeForLegacyMonsterAct(t *testing.T) {
	if !preferNonPCActUpgrade(1002, 8, 72) {
		t.Fatal("expected richer monster ACT to upgrade legacy directional ACT")
	}
	if preferNonPCActUpgrade(1002, 40, 72) {
		t.Fatal("did not expect upgrade when current ACT already has action families")
	}
	if preferNonPCActUpgrade(47, 8, 72) {
		t.Fatal("did not expect NPC ACT upgrade")
	}
	if preferNonPCActUpgrade(1002, 8, 16) {
		t.Fatal("did not expect small candidate ACT upgrade")
	}
}

func TestGR2ResourcesDoNotUseSpriteFallbacks(t *testing.T) {
	tests := []struct {
		resource string
		want     bool
	}{
		{resource: "Guildflag90_1.gr2", want: true},
		{resource: `data\sprite\npc\empelium90_0.gr2`, want: true},
		{resource: "unknown.gr2", want: true},
		{resource: "OBJ_FLAG_A", want: false},
	}
	for _, tt := range tests {
		t.Run(tt.resource, func(t *testing.T) {
			if got := isGR2Resource(tt.resource); got != tt.want {
				t.Fatalf("isGR2Resource(%q) = %v, want %v", tt.resource, got, tt.want)
			}
		})
	}
}

func TestNoSpriteNPCJobsDoNotLoadFallbackSprite(t *testing.T) {
	manager := &res.Manager{}
	for _, job := range []int{scene.ActorJobWarpPortal, scene.ActorJobWarpPortalActive, scene.ActorJobWarpPortalWaiting, scene.ActorJobHiddenNPC, scene.ActorJobHiddenWarpNPC, scene.ActorJobClearNPC} {
		if view, status := LoadNonPC(manager, job, "nonpc"); view != nil || !strings.HasSuffix(status, " no-sprite") {
			t.Fatalf("job %d attempted to load a sprite: %s", job, status)
		}
	}
}

func TestPoringDeathActionRealData(t *testing.T) {
	manager := testutil.RealDataManager(t)
	view, status := LoadNonPC(manager, 1002, "poring")
	if view == nil {
		t.Fatalf("Poring sprite failed: %s", status)
	}
	if len(view.Act.Actions) < 40 {
		t.Fatalf("Poring ACT was not upgraded: actions=%d status=%s", len(view.Act.Actions), status)
	}
	index, action, ok := ResolveAction(view.Act, ActionNonPCDeath, 0)
	if !ok {
		t.Fatalf("Poring death action missing: %s", status)
	}
	t.Logf("Poring death action index=%d frames=%d delay=%.1f source=%s status=%s", index, len(action.Animations), action.DelayMS, view.ActSource, status)
	if index != ActionNonPCDeath*8 {
		t.Fatalf("Poring death action index=%d, want %d", index, ActionNonPCDeath*8)
	}
	if len(action.Animations) < 2 {
		t.Fatalf("Poring death frames=%d, want animated death", len(action.Animations))
	}
}

func TestActFitsSPRRejectsMissingFrame(t *testing.T) {
	spr := &res.SPR{Frames: make([]res.SPRFrame, 2), RGBAIndex: 2}
	act := &res.ACT{Actions: []res.ACTAction{{
		Animations: []res.ACTAnimation{{Layers: []res.ACTLayer{{Index: 1}}}},
	}}}
	if !actFitsSPR(act, spr) {
		t.Fatal("expected ACT to fit SPR")
	}
	act.Actions[0].Animations[0].Layers[0].Index = 2
	if actFitsSPR(act, spr) {
		t.Fatal("expected ACT with missing frame to be rejected")
	}
}

func TestSpriteMotionIndexUsesActionDelay(t *testing.T) {
	action := res.ACTAction{
		Animations: []res.ACTAnimation{{}, {}, {}},
		DelayMS:    100,
	}
	started := time.Unix(0, 0)
	got := MotionIndex(action, started, started.Add(250*time.Millisecond), true)
	if got != 2 {
		t.Fatalf("motion index = %d, want 2", got)
	}
}

func TestHumanoidIdleUsesFirstBodyMotion(t *testing.T) {
	action := res.ACTAction{
		Animations: []res.ACTAnimation{{}, {}, {}},
		DelayMS:    100,
	}
	got := BodyMotionForState(action, State{ActionFamily: scene.SpriteActionIdle}, time.Unix(0, 0), time.Unix(0, 0).Add(250*time.Millisecond))
	if got != 0 {
		t.Fatalf("idle body motion = %d, want 0", got)
	}
}

func TestNonPCIdleBodyMotionAnimates(t *testing.T) {
	action := res.ACTAction{
		Animations: []res.ACTAnimation{{}, {}, {}},
		DelayMS:    100,
	}
	got := BodyMotionForState(action, State{ActionFamily: scene.SpriteActionIdle, LoopIdle: true}, time.Unix(0, 0), time.Unix(0, 0).Add(250*time.Millisecond))
	if got != 2 {
		t.Fatalf("nonpc idle body motion = %d, want 2", got)
	}
}

func TestHumanoidWalkBodyMotionAnimates(t *testing.T) {
	action := res.ACTAction{
		Animations: []res.ACTAnimation{{}, {}, {}},
		DelayMS:    100,
	}
	got := BodyMotionForState(action, State{ActionFamily: scene.SpriteActionWalk}, time.Unix(0, 0), time.Unix(0, 0).Add(250*time.Millisecond))
	if got != 2 {
		t.Fatalf("walk body motion = %d, want 2", got)
	}
}

func TestHumanoidWalkBodyMotionScalesWithMoveSpeed(t *testing.T) {
	action := res.ACTAction{
		Animations: []res.ACTAnimation{{}, {}, {}},
		DelayMS:    150,
	}
	started := time.Unix(0, 0)
	state := State{ActionFamily: scene.SpriteActionWalk, MoveSpeedMS: 400}
	if got := BodyMotionForState(action, state, started, started.Add(399*time.Millisecond)); got != 0 {
		t.Fatalf("walk body motion before slow step = %d, want 0", got)
	}
	if got := BodyMotionForState(action, state, started, started.Add(400*time.Millisecond)); got != 1 {
		t.Fatalf("walk body motion after slow step = %d, want 1", got)
	}
}

func TestWalkBodyMotionUsesDistancePhase(t *testing.T) {
	action := res.ACTAction{
		Animations: []res.ACTAnimation{{}, {}, {}},
		DelayMS:    150,
	}
	started := time.Unix(0, 0)
	state := State{ActionFamily: scene.SpriteActionWalk, MoveSpeedMS: 400, WalkDistance: 1}
	if got := BodyMotionForState(action, state, started, started.Add(399*time.Millisecond)); got != 1 {
		t.Fatalf("walk body motion = %d, want distance-driven 1", got)
	}
}

func TestTransientBodyMotionPlaysOnceFromStateStart(t *testing.T) {
	action := res.ACTAction{
		Animations: []res.ACTAnimation{{}, {}, {}},
		DelayMS:    100,
	}
	started := time.Unix(10, 0)
	got := BodyMotionForState(action, State{ActionFamily: ActionPCAttack2, Started: started}, time.Unix(0, 0), started.Add(250*time.Millisecond))
	if got != 2 {
		t.Fatalf("attack body motion = %d, want 2", got)
	}
	got = BodyMotionForState(action, State{ActionFamily: ActionPCAttack2, Started: started}, time.Unix(0, 0), started.Add(500*time.Millisecond))
	if got != 2 {
		t.Fatalf("finished attack body motion = %d, want final frame 2", got)
	}
}

func TestAttachmentDeltaMatchesAttachPointAttribute(t *testing.T) {
	base := res.ACTAnimation{Pos: []res.ACTPosition{
		{X: 1, Y: 2, Attr: 7},
		{X: 30, Y: 40, Attr: 2},
	}}
	attached := res.ACTAnimation{Pos: []res.ACTPosition{
		{X: 10, Y: 15, Attr: 2},
	}}
	x, y := attachmentDelta(base, attached)
	if x != 20 || y != 25 {
		t.Fatalf("attachment delta = (%d,%d), want (20,25)", x, y)
	}
}

func TestHumanoidIdleHeadMotionDoesNotCycle(t *testing.T) {
	action := res.ACTAction{
		Animations: []res.ACTAnimation{{}, {}, {}},
		DelayMS:    100,
	}
	if got := selectHeadMotion(State{ActionFamily: scene.SpriteActionIdle}, 2, action); got != 0 {
		t.Fatalf("idle head motion = %d, want 0", got)
	}
	if got := selectHeadMotion(State{ActionFamily: scene.SpriteActionIdle, HeadDir: 2, HeadTurn: true}, 0, action); got != 2 {
		t.Fatalf("idle turned head motion = %d, want 2", got)
	}
}

func TestHumanoidWalkHeadMotionFollowsBodyMotion(t *testing.T) {
	action := res.ACTAction{
		Animations: []res.ACTAnimation{{}, {}, {}},
		DelayMS:    100,
	}
	if got := selectHeadMotion(State{ActionFamily: scene.SpriteActionWalk}, 2, action); got != 2 {
		t.Fatalf("walk head motion = %d, want 2", got)
	}
}

func TestHumanoidTransientHeadMotionFollowsBodyMotion(t *testing.T) {
	action := res.ACTAction{
		Animations: []res.ACTAnimation{{}, {}, {}},
		DelayMS:    100,
	}
	if got := selectHeadMotion(State{ActionFamily: ActionPCAttack2}, 2, action); got != 2 {
		t.Fatalf("attack head motion = %d, want 2", got)
	}
}

func TestPlayerRenderLayerOrderMatchesDefaultOpenMidgardFallback(t *testing.T) {
	got := playerRenderLayerOrder(nil, 0, 0)
	want := [8]int{7, 0, 1, 4, 3, 2, 5, 6}
	if got != want {
		t.Fatalf("order = %v, want %v", got, want)
	}
}

func TestPlayerRenderLayerOrderKeepsBodyBeforeHead(t *testing.T) {
	imf := &res.IMF{Layers: make([]res.IMFLayer, 2)}
	for layer := range imf.Layers {
		imf.Layers[layer].Actions = []res.IMFAction{{
			Motions: []res.IMFMotion{{Priority: int32(1 - layer)}},
		}}
	}
	got := playerRenderLayerOrder(imf, 0, 0)
	bodyIndex, headIndex := -1, -1
	for index, layer := range got {
		if layer == 0 && bodyIndex < 0 {
			bodyIndex = index
		}
		if layer == 1 && headIndex < 0 {
			headIndex = index
		}
	}
	if bodyIndex < 0 || headIndex < 0 || bodyIndex > headIndex {
		t.Fatalf("body/head order = %v, body=%d head=%d", got, bodyIndex, headIndex)
	}
}

func TestPlayerRenderLayerOrderRestoresMissingHeadLayer(t *testing.T) {
	order := ensureRenderLayerPresent([8]int{0, 0, 5, 6, 7, 4, 3, 2}, 1)
	if order != [8]int{0, 1, 5, 6, 7, 4, 3, 2} {
		t.Fatalf("order = %v", order)
	}
}

func TestSpriteLayerCenterTreatsPositiveYAsScreenDown(t *testing.T) {
	_, y := scene.SpriteLayerCenter(5, 5, res.ACTLayer{Y: 3})
	if y != 8 {
		t.Fatalf("layer center Y = %.1f, want 8.0", y)
	}
}

func TestSpriteLayerAngleMatchesHighPriest(t *testing.T) {
	source := render.NewImage(3, 3)
	source.RGBA().SetRGBA(1, 0, color.RGBA{R: 255, A: 255})
	target := render.NewImage(21, 21)

	scene.DrawSpriteLayerWithOptions(target, source, res.ACTLayer{
		ScaleX: 1,
		ScaleY: 1,
		Angle:  90,
		Color:  [4]float32{1, 1, 1, 1},
	}, 10, 10, false)

	minX, _, maxX, _, opaque := billboardOpaqueBounds(target)
	if !opaque {
		t.Fatal("rotated layer drew no pixels")
	}
	if minX <= 10 || maxX <= 10 {
		t.Fatalf("positive ACT angle rotated to x=%d..%d, want right of center like HighPriest RenderSprite", minX, maxX)
	}
}

func TestSingleSpriteBillboardPreservesRotatedLayers(t *testing.T) {
	source := render.NewImage(4, 40)
	source.Fill(color.White)
	view := &scene.SpriteView{
		Spr:    &res.SPR{Frames: []res.SPRFrame{{Width: 4, Height: 40}}},
		Images: map[scene.SpriteFrameKey]*render.Image{{}: source},
	}
	anim := res.ACTAnimation{Layers: []res.ACTLayer{{
		X: 21, Y: 8, ScaleX: 1, ScaleY: 1, Angle: 90, Color: [4]float32{1, 1, 1, 1},
	}}}
	for _, ignore := range []bool{false, true} {
		billboard, ok := scene.ComposeSingleSpriteBillboardWithOptions(view, anim, ignore)
		if !ok {
			t.Fatal("could not compose rotated sprite")
		}
		minX, minY, maxX, maxY, opaque := billboardOpaqueBounds(billboard.Image)
		wantW, wantH := 40, 4
		if ignore {
			wantW, wantH = 4, 40
		}
		if !opaque || maxX-minX+1 != wantW || maxY-minY+1 != wantH {
			t.Fatalf("ignore angle=%t: visible sprite is %dx%d, want %dx%d without clipping", ignore, maxX-minX+1, maxY-minY+1, wantW, wantH)
		}
	}
}

func TestDebugPlayerSpriteBillboard(t *testing.T) {
	if os.Getenv("GORO_DEBUG_PLAYER_SPRITE") != "1" {
		t.Skip("set GORO_DEBUG_PLAYER_SPRITE=1")
	}
	manager := testutil.RealDataManager(t)
	for _, sex := range []byte{0, 1} {
		view, status := LoadHumanoid(manager, Appearance{Head: 1, Sex: sex}, "debug player")
		if view == nil {
			t.Logf("sex=%d load failed: %s", sex, status)
			continue
		}
		billboard, ok := HumanoidBillboard(view, State{ActionFamily: scene.SpriteActionIdle, Direction: 0}, time.Now())
		if !ok {
			t.Logf("sex=%d billboard failed: %s", sex, status)
			continue
		}
		bodyIndex, bodyAction, ok := ResolveAction(view.Body.Act, scene.SpriteActionIdle, 0)
		if !ok {
			t.Fatalf("sex=%d missing body action", sex)
		}
		minX, minY, maxX, maxY := spriteAnimationBounds(view.Body, bodyAction.Animations[0], scene.HumanoidBillboardAnchorX, scene.HumanoidBillboardAnchorY, 0, 0)
		t.Logf("sex=%d %s action=%d bounds=(%.1f,%.1f)-(%.1f,%.1f) anchor=(%.0f,%.0f) image=%dx%d", sex, status, bodyIndex, minX, minY, maxX, maxY, billboard.AnchorX, billboard.AnchorY, scene.HumanoidBillboardWidth, scene.HumanoidBillboardHeight)
	}
}

func TestPlayerSpriteCompositionIncludesHeadWithWeapon(t *testing.T) {
	manager := testutil.RealDataManager(t)
	view, status := LoadHumanoid(manager, Appearance{
		Job:    0,
		Head:   1,
		Sex:    0,
		Weapon: 1201,
	}, "debug player")
	if view == nil {
		t.Skipf("debug player unavailable: %s", status)
	}
	bodyActionIndex, bodyAction, ok := ResolveAction(view.Body.Act, scene.SpriteActionIdle, 0)
	if !ok {
		t.Fatalf("body action missing: %s", status)
	}
	order := playerRenderLayerOrder(view.imf, bodyActionIndex, 0)
	if !layerOrderContains(order, 1) {
		t.Fatalf("render order omits head layer: order=%v status=%s", order, status)
	}
	bodyAnim := bodyAction.Animations[0]
	headAnim, ok := actionAnimation(view.head.Act, bodyActionIndex, 0)
	if !ok {
		t.Fatalf("head action missing: %s", status)
	}
	resolvedLayer := view.imf.LayerForPriority(1, bodyActionIndex, 0)
	if resolvedLayer < 0 {
		resolvedLayer = 1
	}
	if resolvedLayer < 0 || resolvedLayer >= len(headAnim.Layers) {
		t.Fatalf("head resolved layer %d out of %d: %s", resolvedLayer, len(headAnim.Layers), status)
	}
	pointX, pointY := view.imf.Point(resolvedLayer, bodyActionIndex, 0)
	dx, dy := attachmentDelta(bodyAnim, headAnim)
	drawLayer := visibleACTLayerIndex(headAnim, resolvedLayer, 1)
	if drawLayer < 0 {
		t.Fatalf("no visible head layer: resolved=%d status=%s", resolvedLayer, status)
	}
	if drawLayer != resolvedLayer {
		pointX, pointY = view.imf.Point(drawLayer, bodyActionIndex, 0)
	}
	layer := headAnim.Layers[drawLayer]
	minX, minY, maxX, maxY := spriteLayerBounds(view.head, layer, scene.HumanoidBillboardAnchorX+float64(pointX+dx), scene.HumanoidBillboardAnchorY+float64(pointY+dy))
	if minY > scene.HumanoidBillboardAnchorY-35 {
		t.Fatalf("head bounds too low: bounds=(%.1f,%.1f)-(%.1f,%.1f) point=(%d,%d) attach=(%d,%d) layer=%+v status=%s", minX, minY, maxX, maxY, pointX, pointY, dx, dy, layer, status)
	}
	t.Logf("head bounds=(%.1f,%.1f)-(%.1f,%.1f) point=(%d,%d) attach=(%d,%d) layer=%+v", minX, minY, maxX, maxY, pointX, pointY, dx, dy, layer)
}

func TestMercenaryHumanoidSpriteCompositionRealWhenConfigured(t *testing.T) {
	manager := testutil.RealDataManager(t)
	cases := []struct {
		name   string
		job    int
		sex    byte
		death  int
		attack int
	}{
		{name: "archer", job: 6017, sex: 1, death: ActionNonPCDeath, attack: ActionPCAttack2},
		{name: "lancer", job: 6027, sex: 0, death: ActionPCDeath, attack: ActionPCAttack3},
		{name: "sword", job: 6037, sex: 0, death: ActionPCDeath, attack: ActionPCAttack2},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			view, status := LoadMercenary(manager, Appearance{Job: tc.job, Head: 1, Sex: tc.sex}, "mercenary")
			if view == nil {
				t.Skipf("mercenary sprite unavailable: %s", status)
			}
			if view.Body == nil || view.head == nil {
				t.Fatalf("mercenary view body=%t head=%t status=%s", view.Body != nil, view.head != nil, status)
			}
			for _, action := range []int{tc.attack, ActionPCHurt, tc.death} {
				if _, _, ok := ResolveAction(view.Body.Act, action, 0); !ok {
					t.Fatalf("missing body action=%d status=%s", action, status)
				}
			}
			if _, ok := HumanoidBillboard(view, State{ActionFamily: tc.attack, Direction: 0}, time.Now()); !ok {
				t.Fatalf("attack billboard unavailable status=%s", status)
			}
			if _, ok := HumanoidBillboard(view, State{ActionFamily: tc.death, Direction: 0}, time.Now()); !ok {
				t.Fatalf("death billboard unavailable action=%d status=%s", tc.death, status)
			}
		})
	}
}

func TestMercenaryAttackBillboardDoesNotClipRealWhenConfigured(t *testing.T) {
	manager := testutil.RealDataManager(t)
	for _, tc := range []struct {
		name   string
		job    int
		sex    byte
		attack int
	}{
		{name: "archer", job: 6017, sex: 0, attack: ActionPCAttack2},
		{name: "lancer", job: 6027, sex: 1, attack: ActionPCAttack3},
		{name: "sword", job: 6037, sex: 1, attack: ActionPCAttack2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			view, status := LoadMercenary(manager, Appearance{Job: tc.job, Head: 1, Sex: tc.sex}, "mercenary")
			if view == nil {
				t.Skipf("mercenary sprite unavailable: %s", status)
			}
			actionFamily := tc.attack
			renderDirection := DirectionForCamera(0, 0)
			_, action, ok := ResolveAction(view.Body.Act, actionFamily, renderDirection)
			if !ok {
				t.Fatalf("mercenary attack action missing: %s", status)
			}
			for motion := range action.Animations {
				billboard, ok := HumanoidBillboard(view, State{
					ActionFamily:   actionFamily,
					Direction:      0,
					FixedMotion:    motion,
					HasFixedMotion: true,
				}, time.Now())
				if !ok {
					t.Fatalf("attack motion %d billboard unavailable: %s", motion, status)
				}
				minX, minY, maxX, maxY, opaque := billboardOpaqueBounds(billboard.Image)
				if !opaque {
					t.Fatalf("attack motion %d drew no opaque pixels: %s", motion, status)
				}
				bounds := billboard.Image.Bounds()
				if minX <= 0 || minY <= 0 || maxX >= bounds.Dx()-1 || maxY >= bounds.Dy()-1 {
					t.Fatalf("attack motion %d touches billboard edge opaque=(%d,%d)-(%d,%d) image=%dx%d anchor=(%.0f,%.0f) status=%s", motion, minX, minY, maxX, maxY, bounds.Dx(), bounds.Dy(), billboard.AnchorX, billboard.AnchorY, status)
				}
			}
		})
	}
}

func TestMercenaryAttackBillboardIncludesDefaultWeaponRealWhenConfigured(t *testing.T) {
	manager := testutil.RealDataManager(t)
	for _, tc := range []struct {
		name   string
		job    int
		sex    byte
		attack int
	}{
		{name: "archer", job: 6017, sex: 0, attack: ActionPCAttack2},
		{name: "lancer", job: 6027, sex: 1, attack: ActionPCAttack3},
		{name: "sword", job: 6037, sex: 1, attack: ActionPCAttack2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			view, status := LoadMercenary(manager, Appearance{Job: tc.job, Head: 1, Sex: tc.sex}, "mercenary")
			if view == nil {
				t.Skipf("mercenary sprite unavailable: %s", status)
			}
			if view.weapon == nil {
				t.Fatalf("mercenary default weapon unavailable: %s", status)
			}
			actionFamily := tc.attack
			renderDirection := DirectionForCamera(0, 0)
			bodyActionIndex, action, ok := ResolveAction(view.Body.Act, actionFamily, renderDirection)
			if !ok {
				t.Fatalf("mercenary attack action missing: %s", status)
			}
			withoutWeaponView := *view
			withoutWeaponView.weapon = nil
			withoutWeaponView.weaponLight = nil
			withoutWeaponView.billboards = make(map[humanoidBillboardKey]*scene.Billboard)
			changed := false
			expectedVisible := false
			for motion := range action.Animations {
				state := State{
					ActionFamily:   actionFamily,
					Direction:      0,
					FixedMotion:    motion,
					HasFixedMotion: true,
				}
				withWeapon, ok := HumanoidBillboard(view, state, time.Now())
				if !ok {
					t.Fatalf("mercenary attack billboard unavailable with weapon: %s", status)
				}
				withoutWeapon, ok := HumanoidBillboard(&withoutWeaponView, state, time.Now())
				if !ok {
					t.Fatalf("mercenary attack billboard unavailable without weapon: %s", status)
				}
				if overlayMotionHasVisibleLayer(view.weapon, view.Body, bodyActionIndex, motion) || overlayMotionHasVisibleLayer(view.weaponLight, view.Body, bodyActionIndex, motion) {
					expectedVisible = true
				}
				if imageRGBAFingerprint(withWeapon.Image) != imageRGBAFingerprint(withoutWeapon.Image) {
					changed = true
					break
				}
			}
			if !expectedVisible {
				t.Logf("mercenary default weapon overlay has no visible layers for action=%d status=%s", bodyActionIndex, status)
				return
			}
			if !changed {
				t.Fatalf("mercenary attack frames did not change after drawing default weapon: %s", status)
			}
		})
	}
}

func TestMercenaryUsesDedicatedWeaponOverlayRealWhenConfigured(t *testing.T) {
	manager := testutil.RealDataManager(t)
	for _, tc := range []struct {
		name       string
		job        int
		sex        byte
		weaponPath string
		lightPath  string
	}{
		{name: "archer", job: 6017, sex: 0, weaponPath: `data\sprite\인간족\용병\활용병_활.spr`},
		{name: "lancer", job: 6027, sex: 1, weaponPath: `data\sprite\인간족\용병\창용병_창.spr`, lightPath: `data\sprite\인간족\용병\창용병_창_검광.spr`},
		{name: "sword", job: 6037, sex: 1, weaponPath: `data\sprite\인간족\용병\검용병_검.spr`, lightPath: `data\sprite\인간족\용병\검용병_검_검광.spr`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			view, status := LoadMercenary(manager, Appearance{Job: tc.job, Head: 1, Sex: tc.sex}, "mercenary")
			if view == nil {
				t.Skipf("mercenary sprite unavailable: %s", status)
			}
			if view.weapon == nil || !strings.Contains(view.weapon.Source, tc.weaponPath) {
				t.Fatalf("weapon source = %q, want dedicated mercenary path %q status=%s", spriteSource(view.weapon), tc.weaponPath, status)
			}
			if tc.lightPath == "" {
				if view.weaponLight != nil {
					t.Fatalf("weapon light source = %q, want none status=%s", spriteSource(view.weaponLight), status)
				}
				return
			}
			if view.weaponLight == nil || !strings.Contains(view.weaponLight.Source, tc.lightPath) {
				t.Fatalf("weapon light source = %q, want dedicated mercenary path %q status=%s", spriteSource(view.weaponLight), tc.lightPath, status)
			}
		})
	}
}

func TestComposeSingleSpriteBillboardUsesAnimationBounds(t *testing.T) {
	view := &scene.SpriteView{
		Spr: &res.SPR{
			RGBAIndex: 0,
			Frames: []res.SPRFrame{{
				Type:   res.SPRFrameRGBA,
				Width:  20,
				Height: 60,
				Data:   testutil.SolidRGBAFrame(20, 60),
			}},
		},
		Act:        &res.ACT{},
		Images:     make(map[scene.SpriteFrameKey]*render.Image),
		Billboards: make(map[scene.SpriteBillboardKey]*scene.Billboard),
	}
	anim := res.ACTAnimation{Layers: []res.ACTLayer{{
		Index:   0,
		SPRType: res.SPRFrameRGBA,
		X:       3,
		Y:       -20,
		ScaleX:  1,
		ScaleY:  1,
		Color:   [4]float32{1, 1, 1, 1},
	}}}

	billboard, ok := scene.ComposeSingleSpriteBillboard(view, anim)
	if !ok {
		t.Fatal("expected single sprite billboard")
	}
	if billboard.Image.Bounds().Dx() != 28 || billboard.Image.Bounds().Dy() != 68 {
		t.Fatalf("billboard size = %dx%d, want 28x68", billboard.Image.Bounds().Dx(), billboard.Image.Bounds().Dy())
	}
	if billboard.AnchorX != 11 || billboard.AnchorY != 54 {
		t.Fatalf("billboard anchor = %.1f, %.1f, want 11, 54", billboard.AnchorX, billboard.AnchorY)
	}
}

func TestCursorFrameBillboardUsesCompositionAnchorAsHotspot(t *testing.T) {
	view := &scene.SpriteView{
		Spr: &res.SPR{
			RGBAIndex: 0,
			Frames: []res.SPRFrame{{
				Type:   res.SPRFrameRGBA,
				Width:  18,
				Height: 24,
				Data:   testutil.SolidRGBAFrame(18, 24),
			}},
		},
		Act: &res.ACT{Actions: []res.ACTAction{{
			Animations: []res.ACTAnimation{{Layers: []res.ACTLayer{{
				Index:   0,
				SPRType: res.SPRFrameRGBA,
				X:       9,
				Y:       12,
				ScaleX:  1,
				ScaleY:  1,
				Color:   [4]float32{1, 1, 1, 1},
			}}}},
		}}},
		Images:     make(map[scene.SpriteFrameKey]*render.Image),
		Billboards: make(map[scene.SpriteBillboardKey]*scene.Billboard),
	}

	billboard, ok := CursorFrameBillboard(view, 0, 0, 20, 24)
	if !ok {
		t.Fatal("expected cursor billboard")
	}
	if got, want := billboard.Image.Bounds().Dx(), 26; got != want {
		t.Fatalf("cursor width = %d, want %d", got, want)
	}
	if got, want := billboard.Image.Bounds().Dy(), 32; got != want {
		t.Fatalf("cursor height = %d, want %d", got, want)
	}
	if billboard.AnchorX != 4 || billboard.AnchorY != 4 {
		t.Fatalf("cursor anchor = %.1f, %.1f, want 4, 4", billboard.AnchorX, billboard.AnchorY)
	}
}

func TestCursorFrameBillboardDoesNotClipTallTargetCursor(t *testing.T) {
	view := &scene.SpriteView{
		Spr: &res.SPR{
			RGBAIndex: 0,
			Frames: []res.SPRFrame{{
				Type:   res.SPRFrameRGBA,
				Width:  32,
				Height: 64,
				Data:   testutil.SolidRGBAFrame(32, 64),
			}},
		},
		Act: &res.ACT{Actions: []res.ACTAction{{
			Animations: []res.ACTAnimation{{Layers: []res.ACTLayer{{
				Index:   0,
				SPRType: res.SPRFrameRGBA,
				X:       16,
				Y:       32,
				ScaleX:  1,
				ScaleY:  1,
				Color:   [4]float32{1, 1, 1, 1},
			}}}},
		}}},
		Images:     make(map[scene.SpriteFrameKey]*render.Image),
		Billboards: make(map[scene.SpriteBillboardKey]*scene.Billboard),
	}

	billboard, ok := CursorFrameBillboard(view, 0, 0, 20, 50)
	if !ok {
		t.Fatal("expected cursor billboard")
	}
	if got, want := billboard.Image.Bounds().Dy(), 72; got != want {
		t.Fatalf("cursor height = %d, want %d", got, want)
	}
	if got, want := billboard.AnchorY, 4.0; got != want {
		t.Fatalf("cursor anchorY = %.1f, want %.1f", got, want)
	}
}

func layerOrderContains(order [8]int, layer int) bool {
	for _, value := range order {
		if value == layer {
			return true
		}
	}
	return false
}

func billboardOpaqueBounds(img *render.Image) (int, int, int, int, bool) {
	if img == nil || img.RGBA() == nil {
		return 0, 0, 0, 0, false
	}
	rgba := img.RGBA()
	bounds := rgba.Bounds()
	minX, minY := bounds.Dx(), bounds.Dy()
	maxX, maxY := -1, -1
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			if rgba.Pix[rgba.PixOffset(x, y)+3] == 0 {
				continue
			}
			relX := x - bounds.Min.X
			relY := y - bounds.Min.Y
			if relX < minX {
				minX = relX
			}
			if relY < minY {
				minY = relY
			}
			if relX > maxX {
				maxX = relX
			}
			if relY > maxY {
				maxY = relY
			}
		}
	}
	if maxX < minX || maxY < minY {
		return 0, 0, 0, 0, false
	}
	return minX, minY, maxX, maxY, true
}

func imageRGBAFingerprint(img *render.Image) uint64 {
	if img == nil || img.RGBA() == nil {
		return 0
	}
	rgba := img.RGBA()
	var out uint64 = 1469598103934665603
	for _, value := range rgba.Pix {
		out ^= uint64(value)
		out *= 1099511628211
	}
	return out
}

func overlayMotionHasVisibleLayer(overlay *scene.SpriteView, body *scene.SpriteView, actionIndex, bodyMotion int) bool {
	if overlay == nil || overlay.Act == nil || body == nil || body.Act == nil || len(overlay.Act.Actions) == 0 {
		return false
	}
	resolvedActionIndex := actionIndex % len(overlay.Act.Actions)
	if resolvedActionIndex < 0 {
		resolvedActionIndex += len(overlay.Act.Actions)
	}
	motionIndex := resolveOverlayMotionIndex(overlay.Act, body.Act, resolvedActionIndex, bodyMotion)
	anim, ok := actionAnimation(overlay.Act, resolvedActionIndex, motionIndex)
	if !ok {
		anim, ok = actionAnimation(overlay.Act, resolvedActionIndex, 0)
	}
	return ok && visibleACTLayerIndex(anim) >= 0
}

func spriteSource(view *scene.SpriteView) string {
	if view == nil {
		return ""
	}
	return view.Source
}

func spriteAnimationBounds(view *scene.SpriteView, anim res.ACTAnimation, anchorX, anchorY float64, posX, posY int32) (float64, float64, float64, float64) {
	minX, minY := math.Inf(1), math.Inf(1)
	maxX, maxY := math.Inf(-1), math.Inf(-1)
	for _, layer := range anim.Layers {
		if layer.Index < 0 {
			continue
		}
		frameIndex := int(layer.Index)
		if layer.SPRType == res.SPRFrameRGBA {
			frameIndex += view.Spr.RGBAIndex
		}
		if frameIndex < 0 || frameIndex >= len(view.Spr.Frames) {
			continue
		}
		frame := view.Spr.Frames[frameIndex]
		cx := anchorX + float64(posX) + float64(layer.X)
		cy := anchorY + float64(posY) + float64(layer.Y)
		w := float64(frame.Width)
		h := float64(frame.Height)
		minX = math.Min(minX, cx-w/2)
		minY = math.Min(minY, cy-h/2)
		maxX = math.Max(maxX, cx+w/2)
		maxY = math.Max(maxY, cy+h/2)
	}
	return minX, minY, maxX, maxY
}

func spriteLayerBounds(view *scene.SpriteView, layer res.ACTLayer, centerX, centerY float64) (float64, float64, float64, float64) {
	frameIndex := int(layer.Index)
	if layer.SPRType == res.SPRFrameRGBA {
		frameIndex += view.Spr.RGBAIndex
	}
	if frameIndex < 0 || frameIndex >= len(view.Spr.Frames) {
		return math.Inf(1), math.Inf(1), math.Inf(-1), math.Inf(-1)
	}
	width := float64(view.Spr.Frames[frameIndex].Width)
	height := float64(view.Spr.Frames[frameIndex].Height)
	scaleX := math.Abs(float64(layer.ScaleX))
	scaleY := math.Abs(float64(layer.ScaleY))
	if scaleX == 0 {
		scaleX = 1
	}
	if scaleY == 0 {
		scaleY = 1
	}
	layerCenterX, layerCenterY := scene.SpriteLayerCenter(centerX, centerY, layer)
	halfW := width * scaleX / 2
	halfH := height * scaleY / 2
	return layerCenterX - halfW, layerCenterY - halfH, layerCenterX + halfW, layerCenterY + halfH
}

func TestSeparateEquipmentPreservesComposition(t *testing.T) {
	manager := testutil.RealDataManager(t)
	start := time.Unix(100, 0)
	view, status := LoadHumanoid(manager, Appearance{
		Job: db.JobKnightH, Head: 1, Sex: 1, Weapon: int(db.WeaponSword), Shield: 1,
	}, "test")
	if view == nil {
		t.Fatal(status)
	}
	// Separating blend groups must preserve the original IMF order and anchors.
	for _, imf := range []*res.IMF{view.imf, nil} {
		copy := *view
		copy.imf = imf
		copy.billboards = make(map[humanoidBillboardKey]*scene.Billboard)
		for direction := 0; direction < 8; direction++ {
			state := State{ActionFamily: ActionPCAttack2, Direction: direction, HasFixedMotion: true, FixedMotion: 2}
			whole, ok := HumanoidBillboard(&copy, state, start)
			if !ok {
				t.Fatal("missing Lord Knight attack")
			}
			state.SeparateEquipment = true
			split, ok := HumanoidBillboard(&copy, state, start)
			if !ok || len(split.Parts) == 0 {
				t.Fatal("missing split Lord Knight")
			}
			got, want := render.NewImage(512, 512), render.NewImage(512, 512)
			draw := func(target *render.Image, b *scene.Billboard) {
				options := &render.DrawImageOptions{}
				options.GeoM.Translate(256-b.AnchorX, 256-b.AnchorY)
				target.DrawImage(b.Image, options)
			}
			draw(want, whole)
			for _, part := range split.Parts {
				draw(got, part)
			}
			for i, channel := range want.RGBA().Pix {
				if scene.AbsInt(int(channel)-int(got.RGBA().Pix[i])) > 2 {
					t.Fatalf("split layers changed composition: direction=%d imf=%t byte=%d", direction, imf != nil, i)
				}
			}
			if again, _ := HumanoidBillboard(&copy, state, start); again != split {
				t.Fatal("split sprite was not cached")
			}
		}
	}
}
