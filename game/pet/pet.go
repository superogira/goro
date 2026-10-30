package pet

import (
	"fmt"
	rand "math/rand/v2"
	"strings"
	"time"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/game/scene"
	"github.com/kivutar/goro/game/skills"
	"github.com/kivutar/goro/game/social"
	"github.com/kivutar/goro/game/sprites"
	"github.com/kivutar/goro/glog"
	"github.com/kivutar/goro/input"
	"github.com/kivutar/goro/network"
	"github.com/kivutar/goro/session"
	gameui "github.com/kivutar/goro/ui"
)

const CaptureSkillID = 0xFFFF

const (
	StateInit        uint8 = 0
	StateFriendly    uint8 = 1
	StateHunger      uint8 = 2
	StateAccessory   uint8 = 3
	StatePerformance uint8 = 4
)

const (
	petTalkFeeding = iota
	petTalkHunting
	petTalkDanger
	petTalkDead
	petTalkNormal
	petTalkPerformanceSpecial
	petTalkLevelUp
	petTalkPerformance1
	petTalkPerformance2
	petTalkPerformance3
	petTalkConnect
)

const (
	petTalkFriendlyThreshold = 900
	petTalkCooldown          = 10 * time.Second
)

var petEmotionTable = [5][5][7]int{
	{
		{32, 32, 29, 29, 7, 23, 28},
		{32, 32, 29, 29, 7, 7, 5},
		{20, 32, -1, 32, 5, 20, 10},
		{33, -1, -1, 10, 20, 10, 6},
		{18, -1, -1, 20, 33, 2, 6},
	},
	{
		{20, 32, 29, 29, 32, 32, 5},
		{20, 32, -1, 32, 29, 32, 32},
		{33, -1, -1, 10, 20, 10, 6},
		{18, -1, -1, 20, 33, 2, 6},
		{15, -1, -1, 19, 3, 2, 21},
	},
	{
		{32, 32, -1, 32, 29, 5, 32},
		{20, -1, -1, 10, 10, 10, 20},
		{33, -1, -1, 20, 33, 2, 6},
		{18, -1, -1, 19, 18, 2, 21},
		{15, 21, -1, 23, 4, 33, 21},
	},
	{
		{32, -1, -1, 10, 10, 10, 20},
		{32, -1, -1, 20, 20, 20, 10},
		{20, -1, -1, 19, 21, 2, 6},
		{33, 21, 19, 23, 3, 33, 21},
		{18, 21, 26, 28, 30, 22, 18},
	},
	{
		{23, -1, -1, 20, 20, 20, 9},
		{32, -1, -1, 19, 20, 0, 0},
		{32, 21, 19, 23, 18, 33, 21},
		{20, 21, 26, 28, 3, 18, 18},
		{20, 21, 26, 28, 30, 18, 30},
	},
}

type CaptureState struct {
	Active  bool
	Started time.Time
}

func CaptureTargetSkill() session.Skill {
	return session.Skill{
		ID:    CaptureSkillID,
		Name:  "Capture Monster",
		Type:  skills.SkillTargetPet,
		Range: 9,
	}
}

func (m *System) StartCapture(ctx client.Context) {
	m.bindings.Skills.Pending = skills.PendingTarget{}
	m.bindings.Skills.PendingText = skills.PendingTextTarget{}
	m.PendingCapture = CaptureState{Active: true, Started: time.Now()}
	m.bindings.Console.AddSystemMessage("Select a monster to capture.")
	glog.Debugf("pet capture target pending")
}

func (m *System) CancelCapture(source string) {
	if !m.PendingCapture.Active {
		return
	}
	m.PendingCapture = CaptureState{}
	glog.Debugf("pet capture canceled source=%s", source)
}

func (m *System) CancelCaptureFromInput(ctx client.Context) bool {
	if !m.PendingCapture.Active || ctx.Input == nil {
		return false
	}
	if !ctx.Input.JustPressed(input.KeyEscape) && !ctx.Input.MouseJustPressed(input.MouseButtonRight) {
		return false
	}
	m.CancelCapture("input")
	return true
}

func (m *System) ApplyCaptureResult(ctx client.Context, result network.PetCaptureResult) {
	if m.SlotMachine.Active {
		m.SlotMachine.setResult(result.Success)
	} else if result.Success {
		m.bindings.Console.AddBlueMessage("Pet capture succeeded.")
	} else {
		m.bindings.Console.AddErrorMessage("Pet capture failed.")
	}
	glog.Debugf("pet capture result success=%t", result.Success)
}

func (m *System) ApplyEggList(ctx client.Context, list network.PetEggList) {
	glog.Debugf("pet egg list indexes=%v", list.Indexes)
	if len(list.Indexes) == 0 {
		m.bindings.Console.AddErrorMessage("No pet eggs available.")
		return
	}
	m.bindings.Windows.Egg.OpenList(ctx, list)
}

func (m *System) ApplyProperty(ctx client.Context, property network.PetProperty) {
	glog.Debugf("pet property name=%q level=%d fullness=%d relationship=%d accessory=%d job=%d modified=%t", property.Name, property.Level, property.Fullness, property.Relationship, property.AccessoryID, property.Job, property.Modified)
	if !m.HasProperty {
		m.OldFullness = property.Fullness
	} else {
		m.OldFullness = m.Property.Fullness
	}
	m.Property = property
	m.HasProperty = true
	if m.InfoRequested || m.bindings.Windows.Info.IsOpen() {
		m.InfoRequested = false
		m.bindings.Windows.Info.OpenInfo(ctx, property)
	}
}

func (m *System) ApplyFeedResult(ctx client.Context, result network.PetFeedResult) {
	name := fmt.Sprintf("item %d", result.ItemID)
	if ctx.Resources != nil {
		if resolved, ok := ctx.Resources.ItemDisplayName(int(result.ItemID), true); ok && strings.TrimSpace(resolved) != "" {
			name = resolved
		}
	}
	if result.Success {
		m.bindings.Console.AddBlueMessage("Fed pet with %s.", name)
		m.sendFeedEmotion(ctx)
		m.sendTalk(ctx, petTalkFeeding)
	} else {
		m.bindings.Console.AddErrorMessage("Failed to feed pet with %s.", name)
	}
	glog.Debugf("pet feed result success=%t item=%d", result.Success, result.ItemID)
}

func (m *System) ApplyStateChange(ctx client.Context, change network.PetStateChange) {
	glog.Debugf("pet state type=%d id=%d data=%d", change.Type, change.ID, change.Data)
	switch change.Type {
	case StateInit:
		m.ID = change.ID
	case StateFriendly:
		m.Property.Relationship = uint16(change.Data)
	case StateHunger:
		m.OldFullness = m.Property.Fullness
		m.Property.Fullness = uint16(change.Data)
	case StateAccessory:
		m.applyAccessoryChange(ctx, change.ID, change.Data)
	case StatePerformance:
		m.startPerformance(ctx, change.ID, change.Data)
	}
	if m.HasProperty && m.bindings.Windows.Info.IsOpen() {
		m.bindings.Windows.Info.OpenInfo(ctx, m.Property)
	}
}

func (m *System) applyAccessoryChange(ctx client.Context, id uint32, accessoryID uint32) {
	if accessoryID == 0 {
		delete(m.bindings.Actors.PetAccessoryIDs, id)
		return
	}
	if m.bindings.Actors.PetAccessoryIDs == nil {
		m.bindings.Actors.PetAccessoryIDs = make(map[uint32]uint32)
	}
	m.bindings.Actors.PetAccessoryIDs[id] = accessoryID
	if ctx.World != nil {
		if actor, ok := ctx.World.Actors[id]; ok {
			_ = m.bindings.Actors.NonPCSpriteView(ctx, actor)
		}
	}
}

func (m *System) ApplyAction(ctx client.Context, action network.PetAction) {
	if action.Data < 5000 {
		emotionType := uint8(action.Data / 10)
		m.bindings.Emotion(ctx, network.EmotionNotify{GID: action.ID, Type: emotionType})
		glog.Debugf("pet emotion actor=%d data=%d type=%d", action.ID, action.Data, emotionType)
		return
	}
	if ctx.Resources == nil {
		glog.Debugf("pet talk actor=%d data=%d ignored=no_resources", action.ID, action.Data)
		return
	}
	text, ok := ctx.Resources.PetTalk(action.Data)
	if !ok {
		glog.Debugf("pet talk actor=%d data=%d ignored=missing_text", action.ID, action.Data)
		return
	}
	m.ApplyTalk(ctx, action.ID, text, time.Now())
	glog.Debugf("pet talk actor=%d data=%d text=%q", action.ID, action.Data, text)
}

func (m *System) ApplyTalk(ctx client.Context, id uint32, text string, now time.Time) {
	text = strings.TrimSpace(text)
	if text == "" {
		return
	}
	name := strings.TrimSpace(m.Property.Name)
	if ctx.World != nil {
		if actor, ok := ctx.World.Actors[id]; ok && strings.TrimSpace(actor.Name) != "" {
			name = strings.TrimSpace(actor.Name)
		}
	}
	if name == "" {
		name = "Pet"
	}
	chat := network.ChatMessage{GID: id, Text: fmt.Sprintf("%s : %s", name, text)}
	m.bindings.Actors.ApplySpeechBubble(ctx, chat, now)
	social.AddConsoleMessage(m.bindings.Console, ctx.Resources, chat)
}

func (m *System) MaybeSendHuntingTalk(ctx client.Context, now time.Time) {
	if rand.IntN(10) >= 3 {
		return
	}
	m.sendTalkThrottled(ctx, petTalkHunting, now)
}

func (m *System) maybeSendDangerTalk(ctx client.Context, now time.Time) {
	m.sendTalkThrottled(ctx, petTalkDanger, now)
}

func (m *System) sendTalkThrottled(ctx client.Context, action int, now time.Time) {
	if !m.LastTalk.IsZero() && m.LastTalk.Add(petTalkCooldown).After(now) {
		return
	}
	if m.sendTalk(ctx, action) {
		m.LastTalk = now
	}
}

func (m *System) sendTalk(ctx client.Context, action int) bool {
	if ctx.Network == nil || !m.HasProperty || m.Property.Relationship <= petTalkFriendlyThreshold {
		return false
	}
	job := m.talkJob(ctx)
	if job == 0 {
		return false
	}
	data := petTalkNumber(job, action, petHungryState(int(m.OldFullness)))
	if err := ctx.Network.SendPetAct(data); err != nil {
		glog.Warnf("send pet talk failed action=%d data=%d: %v", action, data, err)
		return false
	}
	glog.Debugf("pet talk requested action=%d data=%d", action, data)
	return true
}

func (m *System) sendFeedEmotion(ctx client.Context) bool {
	if ctx.Network == nil || !m.HasProperty {
		return false
	}
	emotion := petEmotion(petHungryState(int(m.OldFullness)), petFriendlyState(int(m.Property.Relationship)), petTalkFeeding)
	if emotion < 0 {
		return false
	}
	data := uint32(emotion*10 + 2)
	if err := ctx.Network.SendPetAct(data); err != nil {
		glog.Warnf("send pet feed emotion failed emotion=%d data=%d: %v", emotion, data, err)
		return false
	}
	glog.Debugf("pet feed emotion requested emotion=%d data=%d", emotion, data)
	return true
}

func (m *System) ApplyTalkParameterChange(ctx client.Context, change network.ParameterChange, previousHP int, previousBaseLevel int) {
	if ctx.Session == nil {
		return
	}
	now := time.Now()
	switch change.VarID {
	case network.StatusHP:
		hp := ctx.Session.Vitals.HP
		maxHP := ctx.Session.Vitals.MaxHP
		if maxHP <= 0 {
			maxHP = int(ctx.Session.Selected.MaxHP)
		}
		if hp <= 1 && previousHP > 1 {
			if m.sendTalk(ctx, petTalkDead) {
				m.LastTalk = now
			}
			return
		}
		if maxHP > 0 && hp <= maxHP/4 {
			m.maybeSendDangerTalk(ctx, now)
		}
	case network.StatusBaseLevel:
		if ctx.Session.Progress.BaseLevel > previousBaseLevel {
			if m.sendTalk(ctx, petTalkLevelUp) {
				m.LastTalk = now
			}
		}
	}
}

func (m *System) talkJob(ctx client.Context) int {
	if ctx.World != nil && m.ID != 0 {
		if actor, ok := ctx.World.Actors[m.ID]; ok && actor.Job >= 1000 {
			return int(actor.Job)
		}
	}
	if m.Property.Job >= 1000 {
		return int(m.Property.Job)
	}
	return 0
}

func petTalkNumber(job int, action int, hunger int) uint32 {
	if hunger < 0 {
		hunger = 0
	}
	if hunger > 99 {
		hunger = 99
	}
	if action < 0 {
		action = 0
	}
	return uint32(job*1000 + hunger*10 + action)
}

func petHungryState(fullness int) int {
	switch {
	case fullness > 90 && fullness <= 100:
		return 4
	case fullness > 75 && fullness <= 90:
		return 3
	case fullness > 25 && fullness <= 75:
		return 2
	case fullness > 10 && fullness <= 25:
		return 1
	default:
		return 0
	}
}

func petFriendlyState(relationship int) int {
	switch {
	case relationship > 900 && relationship <= 1000:
		return 4
	case relationship > 750 && relationship <= 900:
		return 3
	case relationship > 250 && relationship <= 750:
		return 2
	case relationship > 100 && relationship <= 250:
		return 1
	default:
		return 0
	}
}

func petEmotion(hunger int, friendly int, action int) int {
	if hunger < 0 || hunger >= len(petEmotionTable) {
		return -1
	}
	if friendly < 0 || friendly >= len(petEmotionTable[hunger]) {
		return -1
	}
	if action < 0 || action >= len(petEmotionTable[hunger][friendly]) {
		return -1
	}
	return petEmotionTable[hunger][friendly][action]
}

func (m *System) startPerformance(ctx client.Context, id uint32, data uint32) {
	if ctx.World == nil {
		return
	}
	actor, ok := ctx.World.Actors[id]
	if !ok {
		return
	}
	action := sprites.ActionNonPCPerf1
	switch data {
	case 2:
		action = sprites.ActionNonPCPerf2
	case 3:
		action = sprites.ActionNonPCPerf3
	case 4:
		action = sprites.ActionNonPCSpecial
	}
	now := time.Now()
	duration := m.bindings.Actors.ActorActionDuration(ctx, actor, action, scene.DefaultAttackAnimationDuration)
	m.bindings.StartAnimation(ctx, id, action, now, duration)
}

func (m *System) HandleContextAction(ctx client.Context, action gameui.PetContextAction) {
	switch action.Kind {
	case gameui.PetContextActionInfo:
		m.OpenInfo(ctx)
	case gameui.PetContextActionFeed:
		m.openFeedConfirm(ctx)
	case gameui.PetContextActionPerformance:
		m.sendCommand(ctx, network.PetCommandPerformance)
	case gameui.PetContextActionBackToEgg:
		m.sendCommand(ctx, network.PetCommandBackToEgg)
	case gameui.PetContextActionUnequipAccessory:
		m.sendCommand(ctx, network.PetCommandUnequipAccessory)
	}
}

func (m *System) OpenInfo(ctx client.Context) {
	if m.HasProperty {
		m.bindings.Windows.Info.OpenInfo(ctx, m.Property)
		return
	}
	m.InfoRequested = true
	m.sendCommand(ctx, network.PetCommandInfo)
}

func (m *System) openFeedConfirm(ctx client.Context) {
	m.bindings.Windows.Confirm.Open(ctx, "Feed Pet", "Are you sure you want to feed your pet?", func() {
		m.sendCommand(ctx, network.PetCommandFeed)
	}, nil)
}

func (m *System) sendCommand(ctx client.Context, command uint8) {
	if ctx.Network == nil {
		m.bindings.Console.AddErrorMessage("Pet command failed: not connected.")
		return
	}
	if err := ctx.Network.SendPetCommand(command); err != nil {
		m.bindings.Console.AddErrorMessage("Pet command failed.")
		glog.Warnf("pet command failed command=%d: %v", command, err)
	}
}
