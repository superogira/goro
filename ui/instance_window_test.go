package ui

import (
	"testing"
	"time"

	"github.com/gogpu/ui/event"
	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/uitest"
	"github.com/gogpu/ui/widget"
	"github.com/kivutar/goro/input"
	"github.com/kivutar/goro/session"
)

func TestInstanceWindowLayoutAndDestroyButton(t *testing.T) {
	for _, leader := range []bool{true, false} {
		manager := NewManager()
		ctx := Context{Session: &session.Session{AccountID: 7}, UIManager: manager, ScreenW: 1024, ScreenH: 768}
		role := uint32(1)
		if leader {
			role = 0
		}
		ctx.Session.Party.Members = []session.PartyMember{{AccountID: 7, Role: role}}
		ctx.Session.Instance = session.Instance{Name: "Endless Tower", ExpiresAt: 1800007200, IdleExpiresAt: 1800000300}
		w := &InstanceWindow{}
		w.Rebind(ctx, func() {})
		wc := widget.NewContext()
		manager.root.Layout(wc, geometry.Tight(geometry.Sz(1024, 768)))
		manager.root.Draw(wc, &uitest.MockCanvas{})
		body := w.content.Children()[1].Children()[0]
		bounds := body.(interface{ ScreenBounds() geometry.Rect }).ScreenBounds()
		for _, row := range body.Children() {
			r := row.(interface{ ScreenBounds() geometry.Rect }).ScreenBounds()
			if !bounds.ContainsRect(r) || r.Height() == 0 {
				t.Fatalf("clipped instance row %v inside %v", r, bounds)
			}
		}
		footer := w.content.Children()[2].Children()[1].Children()[1]
		destroy := footer.Children()[1].(interface{ ScreenBounds() geometry.Rect })
		p := destroy.ScreenBounds().Center()
		manager.root.Event(wc, event.NewMouseEvent(event.MousePress, event.ButtonLeft, event.ButtonStateLeft, p, p, event.ModNone))
		manager.root.Event(wc, event.NewMouseEvent(event.MouseRelease, event.ButtonLeft, 0, p, p, event.ModNone))
		if w.Confirm.IsOpen() != leader {
			t.Fatalf("leader=%t: destroy button confirmation=%t", leader, w.Confirm.IsOpen())
		}
		console := ChatConsole{OnInstanceWindow: func() { w.Toggle(ctx) }}
		if !console.SendText(ctx, "/instance") || w.IsOpen() || w.Confirm.IsOpen() {
			t.Fatal("local command did not close the instance window and prompt")
		}
		if !console.SendText(ctx, "/instance") || !w.IsOpen() {
			t.Fatal("local command did not reopen the instance window")
		}
	}
}

func TestInstanceWindowLifecycleAndCountdown(t *testing.T) {
	ctx := Context{Session: &session.Session{AccountID: 7}, Input: input.NewState(), UIManager: NewManager(), ScreenW: 1024, ScreenH: 768}
	ctx.Session.Party.Members = []session.PartyMember{{AccountID: 7}}
	now := time.Unix(1800000000, 0)
	w := &InstanceWindow{}
	w.Rebind(ctx, func() {})
	if w.IsOpen() {
		t.Fatal("empty window opened")
	}
	ctx.Session.Instance = session.Instance{Name: "Endless Tower", Queued: true, QueuePosition: 2}
	w.Sync(ctx)
	if !w.IsOpen() {
		t.Fatal("reservation did not open window")
	}
	ctx.Session.Instance = session.Instance{Name: "Endless Tower", ExpiresAt: uint32(now.Unix() + 7200), IdleExpiresAt: uint32(now.Unix() + 300)}
	w.UpdatePresentation(ctx, now)
	if w.lifetime.Get() != "Time left: 02:00:00" || w.idleTime.Get() != "Entry time left: 00:05:00" {
		t.Fatalf("timers = %q, %q", w.lifetime.Get(), w.idleTime.Get())
	}
	content := w.content
	w.UpdatePresentation(ctx, now.Add(time.Second))
	if w.content != content || w.lifetime.Get() != "Time left: 01:59:59" {
		t.Fatal("tick rebuilt tree or did not advance")
	}
	w.UpdatePresentation(ctx, now.Add(3*time.Hour))
	if w.lifetime.Get() != "Time left: 00:00:00" || !w.IsOpen() || ctx.Session.Instance.Name == "" {
		t.Fatal("client expired a server-owned instance")
	}
	w.Close()
	w.UpdatePresentation(ctx, now)
	w.Rebind(ctx, func() {})
	if w.IsOpen() {
		t.Fatal("refresh reopened hidden window")
	}
	ctx.Session.Instance.IdleExpiresAt = 0
	w.Sync(ctx)
	if !w.IsOpen() || w.canDestroy {
		t.Fatal("entry did not update window")
	}
	ctx.Input.SetKey(input.KeyEscape, true)
	w.Update(ctx)
	if !w.IsOpen() {
		t.Fatal("passive window captured Escape")
	}
	ctx.Session.Instance = session.Instance{}
	w.Sync(ctx)
	if w.IsOpen() {
		t.Fatal("destroyed instance stayed open")
	}
	if instanceTimeRemaining(0, now) != "--:--:--" {
		t.Fatal("zero deadline counted down")
	}
}

func TestInstanceConfirmationCannotDestroyChangedInstance(t *testing.T) {
	ctx := Context{Session: &session.Session{AccountID: 7}, UIManager: NewManager(), ScreenW: 1024, ScreenH: 768}
	ctx.Session.Party.Members = []session.PartyMember{{AccountID: 7}}
	ctx.Session.Instance = session.Instance{Name: "Tower", Queued: true, QueuePosition: 2}
	calls := 0
	w := &InstanceWindow{}
	w.Rebind(ctx, func() { calls++ })
	w.requestDestroy(ctx)
	if !w.Confirm.IsOpen() || calls != 0 {
		t.Fatal("destruction did not require confirmation")
	}
	ctx.Session.Instance.QueuePosition = 1
	w.Sync(ctx)
	if !w.Confirm.IsOpen() || w.instance.QueuePosition != 1 {
		t.Fatal("queue advancement dismissed confirmation or left the displayed position stale")
	}
	w.Confirm.Confirm(ctx)
	if calls != 1 || ctx.Session.Instance.Name == "" || !w.IsOpen() {
		t.Fatal("confirmation cleared instance before server reply")
	}
	for _, changed := range []session.Instance{
		{Name: "Other dungeon", Queued: true},
		{Name: "Tower", IdleExpiresAt: 1800000300},
	} {
		previous := ctx.Session.Instance
		w.requestDestroy(ctx)
		ctx.Session.Instance = changed
		w.Sync(ctx)
		if w.Confirm.IsOpen() {
			t.Fatal("dungeon or phase change kept old confirmation")
		}
		w.Confirm.Confirm(ctx)
		if calls != 1 {
			t.Fatal("old confirmation destroyed a different dungeon or phase")
		}
		ctx.Session.Instance = previous
		w.Sync(ctx)
	}
	w.requestDestroy(ctx)
	ctx.Session.Party.Members[0].Role = 1
	w.Confirm.Confirm(ctx)
	if calls != 1 {
		t.Fatal("old leader destroyed instance")
	}
	ctx.Session.Party.Members[0].Role = 0
	w.requestDestroy(ctx)
	ctx.Session.Instance = session.Instance{}
	w.Sync(ctx)
	ctx.Session.Instance = session.Instance{Name: "Tower", Queued: true, QueuePosition: 2}
	w.Sync(ctx)
	if w.Confirm.IsOpen() {
		t.Fatal("new reservation kept old confirmation")
	}
	w.Confirm.Confirm(ctx)
	if calls != 1 {
		t.Fatal("stale confirmation callback destroyed a new reservation")
	}
	w.requestDestroy(ctx)
	next := *w
	next.Rebind(ctx, func() { calls += 10 })
	if next.Confirm.IsOpen() {
		t.Fatal("confirmation survived map transition")
	}
	next.requestDestroy(ctx)
	next.Confirm.Confirm(ctx)
	if calls != 11 {
		t.Fatal("rebound window retained old callback")
	}
}
