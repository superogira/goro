package pet

import (
	"math"
	"time"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/game/scene"
	"github.com/kivutar/goro/glog"
	"github.com/kivutar/goro/input"
	"github.com/kivutar/goro/render"
	"github.com/kivutar/goro/res"
)

const (
	petSlotMachineWidth   = 270
	petSlotMachineHeight  = 260
	petSlotMachineAnchorX = 140
	petSlotMachineAnchorY = 165
)

type petSlotMachinePhase int

const (
	petSlotMachineWaiting petSlotMachinePhase = iota
	petSlotMachineSpinning
	petSlotMachineSuccess
	petSlotMachineFail
)

type SlotMachineState struct {
	Active    bool
	TargetID  uint32
	Phase     petSlotMachinePhase
	Started   time.Time
	Attempted bool
	Result    bool
}

func (s *SlotMachineState) setResult(success bool) {
	s.Phase = petSlotMachineSpinning
	s.Started = time.Now()
	s.Result = success
}

func (m *System) OpenSlotMachine(ctx client.Context, targetID uint32) {
	if ctx.Config.Render.NoUI {
		if ctx.Network != nil {
			if err := ctx.Network.SendTryCaptureMonster(targetID); err != nil {
				glog.Warnf("pet capture send failed target=%d: %v", targetID, err)
			}
		}
		return
	}
	if m.loadSlotMachineView(ctx.Resources) == nil {
		if ctx.Network != nil {
			if err := ctx.Network.SendTryCaptureMonster(targetID); err != nil {
				glog.Warnf("pet capture send failed target=%d: %v", targetID, err)
			}
		}
		return
	}
	m.SlotMachine = SlotMachineState{
		Active:   true,
		TargetID: targetID,
		Phase:    petSlotMachineWaiting,
		Started:  time.Now(),
	}
}

func (m *System) UpdateSlotMachine(ctx client.Context) bool {
	if !m.SlotMachine.Active || ctx.Input == nil {
		return false
	}
	if ctx.Input.JustPressed(input.KeyEscape) || ctx.Input.MouseJustPressed(input.MouseButtonRight) {
		m.SlotMachine = SlotMachineState{}
		return true
	}
	if !ctx.Input.MouseJustPressed(input.MouseButtonLeft) {
		return false
	}
	x, y := petSlotMachineOrigin(ctx)
	if !petSlotMachineContains(ctx.Input.MouseX, ctx.Input.MouseY, x, y) {
		return true
	}
	if m.SlotMachine.Phase != petSlotMachineWaiting || m.SlotMachine.Attempted {
		return true
	}
	if ctx.Network == nil {
		m.bindings.Console.AddErrorMessage("Capture failed: not connected.")
		return true
	}
	if err := ctx.Network.SendTryCaptureMonster(m.SlotMachine.TargetID); err != nil {
		m.bindings.Console.AddErrorMessage("Capture failed.")
		glog.Warnf("pet capture send failed target=%d: %v", m.SlotMachine.TargetID, err)
		return true
	}
	m.SlotMachine.Attempted = true
	glog.Debugf("pet capture try target=%d", m.SlotMachine.TargetID)
	return true
}

func (m *System) DrawSlotMachine(screen *render.Frame, ctx client.Context, now time.Time) {
	if !m.SlotMachine.Active {
		return
	}
	view := m.loadSlotMachineView(ctx.Resources)
	if view == nil || view.Act == nil || len(view.Act.Actions) == 0 {
		return
	}
	actionIndex, motion, visible := m.slotMachineFrame(view.Act, now)
	if !visible {
		return
	}
	if actionIndex < 0 || actionIndex >= len(view.Act.Actions) {
		return
	}
	action := view.Act.Actions[actionIndex]
	if motion < 0 || motion >= len(action.Animations) {
		return
	}
	key := scene.SpriteBillboardKey{ActionIndex: actionIndex, Motion: motion}
	billboard, ok := view.Billboards[key]
	if !ok {
		var baseOK bool
		billboard, baseOK = scene.ComposeSingleSpriteBillboard(view, action.Animations[motion])
		if !baseOK {
			return
		}
		view.Billboards[key] = billboard
	}
	x, y := petSlotMachineOrigin(ctx)
	var opts render.DrawImageOptions
	opts.GeoM.Translate(float64(x)+petSlotMachineAnchorX-billboard.AnchorX, float64(y)+petSlotMachineAnchorY-billboard.AnchorY)
	opts.Filter = scene.SpriteDrawFilter()
	opts.Blend = render.BlendSourceOver
	screen.DrawImage(billboard.Image, &opts)
}

func (m *System) slotMachineFrame(act *res.ACT, now time.Time) (int, int, bool) {
	phase := m.SlotMachine.Phase
	actionIndex := int(phase)
	if actionIndex < 0 || actionIndex >= len(act.Actions) {
		return 0, 0, false
	}
	action := act.Actions[actionIndex]
	if len(action.Animations) == 0 {
		return 0, 0, false
	}
	delay := float64(action.DelayMS)
	if delay <= 0 {
		delay = 150
	}
	elapsed := float64(now.Sub(m.SlotMachine.Started) / time.Millisecond)
	switch phase {
	case petSlotMachineWaiting:
		motion := int(math.Floor(elapsed/delay*2)) % len(action.Animations)
		return actionIndex, motion, true
	case petSlotMachineSpinning:
		maxFrame := len(action.Animations)
		if m.SlotMachine.Result {
			maxFrame += 7
		} else {
			maxFrame += 3
		}
		motion := int(math.Floor(elapsed / delay))
		if motion >= maxFrame {
			if m.SlotMachine.Result {
				m.SlotMachine.Phase = petSlotMachineSuccess
			} else {
				m.SlotMachine.Phase = petSlotMachineFail
			}
			m.SlotMachine.Started = now
			return m.slotMachineFrame(act, now)
		}
		return actionIndex, motion % len(action.Animations), true
	case petSlotMachineSuccess, petSlotMachineFail:
		motion := int(math.Floor(elapsed / delay))
		if motion >= len(action.Animations) {
			if elapsed >= float64(len(action.Animations))*delay+500 {
				m.SlotMachine = SlotMachineState{}
				return actionIndex, len(action.Animations) - 1, false
			}
			return actionIndex, len(action.Animations) - 1, true
		}
		return actionIndex, motion, true
	default:
		return 0, 0, false
	}
}

func (m *System) loadSlotMachineView(manager *res.Manager) *scene.SpriteView {
	if manager == nil || m.slotMachineMiss {
		return nil
	}
	if m.slotMachineView != nil {
		return m.slotMachineView
	}
	view, status := loadSlotMachineSpriteView(manager)
	if view == nil {
		m.slotMachineMiss = true
		glog.Warnf("pet slot machine sprite unavailable: %s", status)
		return nil
	}
	m.slotMachineView = view
	glog.Debugf("pet slot machine sprite resources %s", status)
	return view
}

func loadSlotMachineSpriteView(manager *res.Manager) (*scene.SpriteView, string) {
	return scene.LoadSpriteView(manager,
		[]string{"data\\sprite\\slotmachine.act", "data/sprite/slotmachine.act"},
		[]string{"data\\sprite\\slotmachine.spr", "data/sprite/slotmachine.spr"},
		nil,
		"pet slotmachine",
	)
}

func petSlotMachineOrigin(ctx client.Context) (int, int) {
	w, h := ctx.ScreenSize()
	return w/2 - petSlotMachineWidth/2, h/2 - petSlotMachineHeight/2
}

func petSlotMachineContains(x, y, rx, ry int) bool {
	return x >= rx && x < rx+petSlotMachineWidth && y >= ry && y < ry+petSlotMachineHeight
}
