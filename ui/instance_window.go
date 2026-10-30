package ui

import (
	"fmt"
	"time"

	"github.com/gogpu/ui/primitives"
	"github.com/gogpu/ui/state"
	"github.com/gogpu/ui/widget"
	"github.com/kivutar/goro/session"
	"github.com/kivutar/goro/ui/rotheme"
)

const (
	instanceWindowW = 320
	instanceWindowH = 178
)

type InstanceWindow struct {
	Window
	Confirm    ConfirmModal
	instance   session.Instance
	canDestroy bool
	onDestroy  func()
	lifetime   state.Signal[string]
	idleTime   state.Signal[string]
	lastSecond int64
}

func (w *InstanceWindow) Rebind(ctx Context, onDestroy func()) {
	w.onDestroy = onDestroy
	w.closeConfirmation(ctx)
	w.Sync(ctx)
	if w.IsOpen() {
		w.RebindContent(ctx, w.widgetTree(ctx))
	}
}

func (w *InstanceWindow) Toggle(ctx Context) {
	if w.IsOpen() {
		w.Close()
	} else if ctx.Session != nil && ctx.Session.Instance.Name != "" {
		w.instance = ctx.Session.Instance
		w.canDestroy = ctx.Session.CanDestroyInstance()
		w.openWindow(ctx)
	}
}

// Apply every server transition, including destruction followed by another
// reservation in the same network read. A hidden window stays hidden on refresh.
func (w *InstanceWindow) Sync(ctx Context) {
	if ctx.Session == nil {
		return
	}
	previous := w.instance
	i := ctx.Session.Instance
	allowed := ctx.Session.CanDestroyInstance()
	transition := i.Name != previous.Name || i.Phase() != previous.Phase()
	changed := i != previous || allowed != w.canDestroy
	if transition || allowed != w.canDestroy {
		w.closeConfirmation(ctx)
	}
	w.instance, w.canDestroy = i, allowed
	if i.Name == "" {
		if w.IsOpen() || w.Confirm.IsOpen() {
			w.Close()
		}
		return
	}
	if transition {
		w.openWindow(ctx)
	} else if changed && w.IsOpen() {
		w.SetContent(w.widgetTree(ctx))
		w.Publish(ctx)
	}
}

func (w *InstanceWindow) openWindow(ctx Context) {
	w.EnsureWindow(instanceWindowW, instanceWindowH)
	w.CloseOnEsc = false
	w.Window.Open(ctx, w.widgetTree(ctx))
	w.Publish(ctx)
}

func (w *InstanceWindow) Close() {
	w.closeConfirmation(w.ctx)
	w.Window.Close()
}

func (w *InstanceWindow) closeConfirmation(ctx Context) {
	w.Confirm.Close(ctx)
	w.Confirm.onOK = nil
}

func (w *InstanceWindow) UpdatePresentation(ctx Context, now time.Time) {
	w.Sync(ctx)
	if w.IsOpen() && now.Unix() != w.lastSecond {
		w.lastSecond = now.Unix()
		w.updateTimers(now)
	}
}

func (w *InstanceWindow) Update(ctx Context) bool {
	if !w.IsOpen() {
		return false
	}
	consumed := w.Window.Update(ctx)
	w.Publish(ctx)
	return consumed
}

func (w *InstanceWindow) requestDestroy(ctx Context) {
	if !ctx.Session.CanDestroyInstance() || w.onDestroy == nil {
		return
	}
	requested := ctx.Session.Instance
	message := "Destroy this memorial dungeon?"
	if requested.Queued {
		message = "Cancel this memorial dungeon reservation?"
	}
	w.Confirm.Open(ctx, "Memorial Dungeon", message, func() {
		// State or party leadership may change while confirmation is open.
		current := ctx.Session.Instance
		if current.Name == requested.Name && current.Phase() == requested.Phase() && ctx.Session.CanDestroyInstance() && w.onDestroy != nil {
			w.onDestroy()
		}
	}, nil)
}

func (w *InstanceWindow) widgetTree(ctx Context) widget.Widget {
	if w.lifetime == nil {
		w.lifetime = state.NewSignal("")
		w.idleTime = state.NewSignal("")
	}
	w.updateTimers(time.Now())
	i := w.instance
	status := "In progress"
	switch i.Phase() {
	case session.InstanceWaiting:
		status = fmt.Sprintf("Waiting — queue position: %d", i.QueuePosition)
	case session.InstanceReady:
		status = "Ready to enter"
	}
	rows := []widget.Widget{rotheme.Label(i.Name).MaxLines(1), rotheme.Text(status).MaxLines(1)}
	if !i.Queued {
		rows = append(rows, rotheme.Text("").ContentSignal(w.lifetime).MaxLines(1))
		if i.IdleExpiresAt != 0 {
			rows = append(rows, rotheme.Text("").ContentSignal(w.idleTime).MaxLines(1))
		}
	}
	label := "Destroy"
	if i.Queued {
		label = "Cancel reservation"
	}
	return Win(Title("Memorial Dungeon"), CloseButton(true), OnClose(w.Close), Size(instanceWindowW, instanceWindowH),
		Content(primitives.Box(rows...).Padding(12).Gap(6)),
		Footer(primitives.Expanded(primitives.Box()),
			rotheme.ButtonDisabled(label, !w.canDestroy, func() { w.requestDestroy(ctx) }),
			rotheme.Button("Close", w.Close),
		),
	)
}

func (w *InstanceWindow) updateTimers(now time.Time) {
	w.lifetime.Set("Time left: " + instanceTimeRemaining(w.instance.ExpiresAt, now))
	w.idleTime.Set("Entry time left: " + instanceTimeRemaining(w.instance.IdleExpiresAt, now))
}

func instanceTimeRemaining(deadline uint32, now time.Time) string {
	if deadline == 0 {
		return "--:--:--"
	}
	seconds := max(int64(deadline)-now.Unix(), 0)
	return fmt.Sprintf("%02d:%02d:%02d", seconds/3600, seconds/60%60, seconds%60)
}
