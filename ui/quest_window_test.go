package ui

import (
	"testing"
	"time"

	uiapp "github.com/gogpu/ui/app"
	"github.com/gogpu/ui/event"
	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/uitest"
	"github.com/kivutar/goro/input"
	"github.com/kivutar/goro/session"
)

func questTestContext() Context {
	return Context{Session: &session.Session{}, Input: input.NewState(), UIManager: NewManager(), ScreenW: 1024, ScreenH: 768}
}

func TestQuestJournalTabsUpdatesAndIdle(t *testing.T) {
	ctx := questTestContext()
	ctx.Session.Quests.Replace([]session.Quest{{ID: 3, Active: true}, {ID: 1}, {ID: 2, Active: true}})
	w := &QuestWindow{}
	w.OpenWindow(ctx)
	if len(w.rows) != 2 || w.rows[0].ID != 2 || w.rows[1].ID != 3 {
		t.Fatalf("active rows = %+v", w.rows)
	}
	w.selectRow(1)
	ctx.Session.Quests.Set(session.Quest{ID: 4, Active: true})
	w.UpdatePresentation(ctx, time.Now())
	if w.selectedID != 3 || w.selectedRow.Get() != 1 {
		t.Fatal("update lost selection")
	}
	content := w.content
	w.UpdatePresentation(ctx, time.Now())
	if w.content != content {
		t.Fatal("idle journal rebuilt content")
	}
	w.tab = 1
	w.refreshRows(ctx)
	if len(w.rows) != 1 || w.rows[0].ID != 1 {
		t.Fatalf("inactive rows = %+v", w.rows)
	}
	w.tab = 2
	w.refreshRows(ctx)
	if len(w.rows) != 4 {
		t.Fatalf("all rows = %+v", w.rows)
	}
	w.openDetail(ctx, 3)
	ctx.Session.Quests.Remove(3)
	w.UpdatePresentation(ctx, time.Now())
	if w.detail.IsOpen() {
		t.Fatal("removed quest detail stayed open")
	}
	w.Close()
	version := w.listVersion
	ctx.Session.Quests.Set(session.Quest{ID: 10, Active: true})
	w.UpdatePresentation(ctx, time.Now())
	if w.listVersion != version {
		t.Fatal("closed journal did list work")
	}
	w.OpenWindow(ctx)
	if len(w.rows) != 4 {
		t.Fatal("reopening did not refresh journal")
	}
}

func TestQuestDetailKeepsUpdatingWithJournalClosed(t *testing.T) {
	ctx := questTestContext()
	now := time.Unix(1800000000, 0)
	ctx.Session.Quests.Set(session.Quest{ID: 1001, Active: true, ExpiresAt: uint32(now.Unix() + 90), Objectives: []session.QuestObjective{{MonsterID: 1002, Name: "Poring", Required: 10}}})
	w := &QuestWindow{}
	w.OpenWindow(ctx)
	w.openDetail(ctx, 1001)
	w.Close()
	content := w.detail.content
	w.UpdatePresentation(ctx, now)
	if w.deadlineText.Get() != "Time left: 00:01:30" || w.detail.content != content {
		t.Fatal("timer rebuilt detail or showed wrong deadline")
	}
	ctx.Session.Quests.UpdateHunt(1001, 1002, 1, 10)
	w.UpdatePresentation(ctx, now.Add(time.Second))
	if w.detail.content == content {
		t.Fatal("closed journal blocked detail update")
	}
	w.UpdatePresentation(ctx, now.Add(90*time.Second))
	if w.deadlineText.Get() != "Time limit expired" {
		t.Fatal("expired timer not displayed")
	}
	if _, ok := ctx.Session.Quests.Entries[1001]; !ok {
		t.Fatal("timer removed a server-owned quest")
	}
	ctx.Session.Quests.Remove(1001)
	w.UpdatePresentation(ctx, now)
	if w.detail.IsOpen() {
		t.Fatal("deletion ignored while journal closed")
	}
}

func TestQuestRightClickUsesPublishedInputAndWaitsForAck(t *testing.T) {
	ctx := questTestContext()
	a := uiapp.New()
	bridge := mailWindowTestApp{basicMenuTestApp{app: a}}
	ctx.UIManager.(*Manager).SetUIApp(bridge)
	ctx.UIApp = bridge
	ctx.Session.Quests.Set(session.Quest{ID: 1001, Active: true})
	w := &QuestWindow{}
	requests := 0
	w.Rebind(ctx, func(id uint32, active bool) {
		if id != 1001 || active {
			t.Fatalf("toggle = %d, %v", id, active)
		}
		requests++
	})
	w.OpenWindow(ctx)
	a.Frame()
	a.Window().DrawTo(&uitest.MockCanvas{})
	p := geometry.Pt(float32(w.x+questTabW+50), float32(w.y+ROWindowTitleHeight+16))
	a.Window().HandleEvent(event.NewMouseEvent(event.MousePress, event.ButtonRight, event.ButtonStateRight, p, p, event.ModNone))
	a.Window().HandleEvent(event.NewMouseEvent(event.MouseRelease, event.ButtonRight, 0, p, p, event.ModNone))
	if requests != 1 || !ctx.Session.Quests.Entries[1001].Active {
		t.Fatalf("requests = %d, active = %v", requests, ctx.Session.Quests.Entries[1001].Active)
	}
	ctx.Input.SetMousePosition(int(p.X), int(p.Y))
	if !w.Update(ctx) {
		t.Fatal("journal click leaked to map input")
	}
	ctx.Session.Quests.SetActive(1001, false)
	w.UpdatePresentation(ctx, time.Now())
	if len(w.rows) != 0 {
		t.Fatal("ACK did not remove quest from active tab")
	}
	a.Frame()
	a.Window().DrawTo(&uitest.MockCanvas{})
	// Switch tabs and open details through the published widgets too.
	tab := geometry.Pt(float32(w.x+questTabW/2), float32(w.y+ROWindowTitleHeight+105))
	a.Window().HandleEvent(event.NewMouseEvent(event.MousePress, event.ButtonLeft, event.ButtonStateLeft, tab, tab, event.ModNone))
	a.Window().HandleEvent(event.NewMouseEvent(event.MouseRelease, event.ButtonLeft, 0, tab, tab, event.ModNone))
	if w.tab != 1 || len(w.rows) != 1 || w.rows[0].ID != 1001 {
		t.Fatal("inactive tab did not show the deactivated quest")
	}
	a.Frame()
	a.Window().DrawTo(&uitest.MockCanvas{})
	a.Window().HandleEvent(event.NewMouseEvent(event.MouseDoubleClick, event.ButtonLeft, 0, p, p, event.ModNone))
	if !w.detail.IsOpen() || w.detailID != 1001 {
		t.Fatal("double click did not open quest details")
	}
}

func TestQuestMetadataFallbackAndObjectiveDisplay(t *testing.T) {
	meta := questMetadata(Context{}, 12345)
	if meta.Title != "Quest #12345" || meta.Description == "" {
		t.Fatalf("fallback = %+v", meta)
	}
	objective := session.QuestObjective{Name: "Poring", Current: 0, Required: 10}
	if got := questObjectiveText(objective); got != "Poring: 0 / 10" {
		t.Fatal(got)
	}
	objective.Required = 0
	if got := questObjectiveText(objective); got != "Poring: 0" {
		t.Fatal(got)
	}
}
