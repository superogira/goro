package ui

import (
	"testing"

	"github.com/gogpu/ui/event"
	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/widget"
	"github.com/kivutar/goro/client"
)

func TestLevelUpNotificationsUseOriginalScreenCorners(t *testing.T) {
	manager := NewManager()
	ctx := client.Context{UIManager: manager, ScreenW: 800, ScreenH: 600}
	var notifications LevelUpNotifications
	notifications.NotifyBase()
	notifications.NotifyJob()

	if action := notifications.Update(ctx); action != LevelUpNotificationNone {
		t.Fatalf("initial action = %d", action)
	}
	manager.root.Layout(widget.NewContext(), geometry.Tight(geometry.Sz(800, 600)))

	if got := notifications.job.root.(*positionedOverlay).Bounds(); got != geometry.NewRect(0, 557, 43, 43) {
		t.Fatalf("job notification bounds = %+v", got)
	}
	if got := notifications.base.root.(*positionedOverlay).Bounds(); got != geometry.NewRect(757, 557, 43, 43) {
		t.Fatalf("base notification bounds = %+v", got)
	}
	if !manager.PointerBlocked(10, 570) || !manager.PointerBlocked(780, 570) {
		t.Fatal("visible notifications did not block pointer input")
	}
}

func TestLevelUpNotificationClickDismissesAndReportsAction(t *testing.T) {
	manager := NewManager()
	ctx := client.Context{UIManager: manager, ScreenW: 800, ScreenH: 600}
	var notifications LevelUpNotifications
	notifications.NotifyBase()
	notifications.NotifyJob()
	notifications.Update(ctx)
	manager.root.Layout(widget.NewContext(), geometry.Tight(geometry.Sz(800, 600)))

	point := geometry.Pt(780, 570)
	if !manager.root.Event(widget.NewContext(), event.NewMouseEvent(event.MousePress, event.ButtonLeft, event.ButtonStateLeft, point, point, event.ModNone)) {
		t.Fatal("base notification press was not consumed")
	}
	if action := notifications.Update(ctx); action != LevelUpNotificationNone {
		t.Fatalf("press action = %d, want none before release", action)
	}
	if !notifications.BaseVisible() {
		t.Fatal("base notification closed while its mouse press was still held")
	}
	if !manager.PointerBlocked(780, 570) {
		t.Fatal("base notification stopped blocking while its mouse press was held")
	}
	if !manager.root.Event(widget.NewContext(), event.NewMouseEvent(event.MouseRelease, event.ButtonLeft, 0, point, point, event.ModNone)) {
		t.Fatal("base notification release was not consumed")
	}
	if action := notifications.Update(ctx); action != LevelUpNotificationBase {
		t.Fatalf("click action = %d, want base", action)
	}
	if notifications.BaseVisible() {
		t.Fatal("base notification remained visible after activation")
	}
	if !notifications.JobVisible() {
		t.Fatal("base activation dismissed the job notification")
	}
	if len(manager.overlays) != 1 || manager.overlays[0] != notifications.job.root {
		t.Fatalf("published overlays after click = %d", len(manager.overlays))
	}
}

func TestLevelUpNotificationTracksHoverAndPressedStates(t *testing.T) {
	w := newLevelUpNotificationWidget(nil)
	w.Layout(widget.NewContext(), geometry.Tight(geometry.Sz(43, 43)))
	point := geometry.Pt(20, 20)
	w.Event(widget.NewContext(), event.NewMouseEvent(event.MouseEnter, event.ButtonNone, 0, point, point, event.ModNone))
	if !w.hovered {
		t.Fatal("notification did not enter hover state")
	}
	w.Event(widget.NewContext(), event.NewMouseEvent(event.MousePress, event.ButtonLeft, event.ButtonStateLeft, point, point, event.ModNone))
	if !w.pressed {
		t.Fatal("notification did not enter pressed state")
	}
	w.Event(widget.NewContext(), event.NewMouseEvent(event.MouseLeave, event.ButtonNone, 0, point, point, event.ModNone))
	if w.hovered {
		t.Fatal("notification retained hover state after leave")
	}
	if w.pressed {
		t.Fatal("notification retained pressed state after leave")
	}
}

func TestLevelUpNotificationRebindPreservesStateAndRefreshesCallbacks(t *testing.T) {
	manager := NewManager()
	ctx := client.Context{UIManager: manager, ScreenW: 800, ScreenH: 600}
	var original LevelUpNotifications
	original.NotifyJob()
	original.Update(ctx)

	next := original
	next.Rebind(ctx)
	next.Update(ctx)
	manager.root.Layout(widget.NewContext(), geometry.Tight(geometry.Sz(800, 600)))
	point := geometry.Pt(20, 570)
	manager.root.Event(widget.NewContext(), event.NewMouseEvent(event.MousePress, event.ButtonLeft, event.ButtonStateLeft, point, point, event.ModNone))
	manager.root.Event(widget.NewContext(), event.NewMouseEvent(event.MouseRelease, event.ButtonLeft, 0, point, point, event.ModNone))

	if action := next.Update(ctx); action != LevelUpNotificationJob {
		t.Fatalf("rebound click action = %d, want job", action)
	}
	if next.JobVisible() {
		t.Fatal("rebound notification remained visible after activation")
	}
	if !original.JobVisible() {
		t.Fatal("rebound callback still mutated the previous owner")
	}
}
