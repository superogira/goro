package social

import (
	"encoding/binary"
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/uitest"
	"github.com/gogpu/ui/widget"
	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/game/internal/testutil"
	"github.com/kivutar/goro/network"
	"github.com/kivutar/goro/session"
	gameui "github.com/kivutar/goro/ui"
	"github.com/kivutar/goro/ui/rotheme"
)

type mailTest struct {
	t      *testing.T
	m      *System
	ctx    client.Context
	server net.Conn
	now    time.Time
}

func newMailTest(t *testing.T) *mailTest {
	t.Helper()
	netClient, server := testutil.NewConnection(t, 20080910)
	h := &mailTest{t: t, m: newTestSystem(), server: server, now: time.Now(), ctx: client.Context{
		Network: netClient, Session: &session.Session{Inventory: session.Inventory{Zeny: 1000,
			Items: []session.InventoryItem{{Index: 7, ItemID: 501, Amount: 10, Identified: true}}}},
		ScreenW: 1024, ScreenH: 768,
	}}
	h.m.bindings.Windows.MailWindow.Open(h.ctx, h.m.MailError)
	return h
}

func (h *mailTest) action(action gameui.MailAction) { h.m.HandleMailAction(h.ctx, action, h.now) }

func (h *mailTest) expect(packets ...[]byte) {
	h.t.Helper()
	var want []byte
	for _, p := range packets {
		want = append(want, p...)
	}
	testutil.ReadPackets(h.t, h.server, want)
}

func (h *mailTest) quiet() {
	h.t.Helper()
	if err := h.server.SetReadDeadline(time.Now().Add(20 * time.Millisecond)); err != nil {
		h.t.Fatal(err)
	}
	n, err := h.server.Read(make([]byte, 1))
	if n != 0 || err == nil {
		h.t.Fatalf("unexpected packet: bytes=%d err=%v", n, err)
	}
	if netErr, ok := err.(net.Error); !ok || !netErr.Timeout() {
		h.t.Fatalf("checking for packet: %v", err)
	}
}

func (h *mailTest) result(opcode uint16, result network.MailResult) {
	h.t.Helper()
	n := 3
	if opcode == network.PacketZCMailAddAttachment {
		n = 5
	}
	if opcode == network.PacketZCMailDelete || opcode == network.PacketZCMailReturn {
		n = 8
	}
	p := make([]byte, n)
	binary.LittleEndian.PutUint16(p, opcode)
	switch n {
	case 3:
		p[2] = byte(result.Result)
	case 5:
		binary.LittleEndian.PutUint16(p[2:], result.Index)
		p[4] = byte(result.Result)
	case 8:
		binary.LittleEndian.PutUint32(p[2:], result.ID)
		binary.LittleEndian.PutUint16(p[6:], result.Result)
	}
	if !h.m.HandleMailPacket(h.ctx, network.Packet{ID: opcode, Data: p}, h.now, false) {
		h.t.Fatal("mail ACK was not handled")
	}
}

func (h *mailTest) compose() {
	h.action(gameui.MailAction{Kind: gameui.MailActionCompose, Recipient: "Erika", Title: "Hello"})
	h.expect(network.BuildMailResetPacket(network.MailResetAll))
}

func (h *mailTest) readReply(id, zeny uint32) {
	h.t.Helper()
	p := make([]byte, 101)
	binary.LittleEndian.PutUint16(p, network.PacketZCMailRead)
	binary.LittleEndian.PutUint16(p[2:], uint16(len(p)))
	binary.LittleEndian.PutUint32(p[4:], id)
	copy(p[8:48], "Gift")
	copy(p[48:72], "Erika")
	binary.LittleEndian.PutUint32(p[76:], zeny)
	h.m.HandleMailPacket(h.ctx, network.Packet{ID: network.PacketZCMailRead, Data: p}, h.now, false)
}

func (h *mailTest) listReply(ids ...uint32) {
	p := make([]byte, 8+len(ids)*73)
	binary.LittleEndian.PutUint16(p, network.PacketZCMailList)
	binary.LittleEndian.PutUint16(p[2:], uint16(len(p)))
	binary.LittleEndian.PutUint32(p[4:], uint32(len(ids)))
	for i, id := range ids {
		o := 8 + i*73
		binary.LittleEndian.PutUint32(p[o:], id)
		copy(p[o+4:o+44], "Gift")
		copy(p[o+45:o+69], "Erika")
	}
	h.m.HandleMailPacket(h.ctx, network.Packet{ID: network.PacketZCMailList, Data: p}, h.now, false)
}

func TestMailServerOpenListReadAndNewMail(t *testing.T) {
	h := newMailTest(t)
	h.m.bindings.Windows.MailWindow.CloseFromServer(h.ctx)
	open := network.Packet{ID: network.PacketZCMailWindow, Data: []byte{0x60, 2, 0, 0, 0, 0}}
	h.m.HandleMailPacket(h.ctx, open, h.now, false)
	h.expect(network.BuildMailResetPacket(network.MailResetAll), network.BuildMailRefreshPacket())
	h.m.HandleMailPacket(h.ctx, open, h.now, false)
	h.quiet()
	h.listReply(42, 43)
	if h.m.Mail.Pending.Kind != gameui.MailActionNone || h.m.Mail.inboxDirty || !h.m.bindings.Windows.MailWindow.IsOpen() {
		t.Fatal("opening/listing the inbox did not finish")
	}
	h.action(gameui.MailAction{Kind: gameui.MailActionRead, ID: 42})
	h.expect(network.BuildMailReadPacket(42))
	h.readReply(43, 0)
	if h.m.Mail.Pending.ID != 42 {
		t.Fatal("read reply for another mail consumed request")
	}
	h.readReply(42, 0)
	if msg, ok := h.m.bindings.Windows.MailWindow.ReadMessage(); !ok || msg.ID != 42 {
		t.Fatal("read reply did not open requested mail")
	}
	h.compose()
	newMail := network.Packet{ID: network.PacketZCMailNew, Data: make([]byte, 70)}
	binary.LittleEndian.PutUint16(newMail.Data, newMail.ID)
	h.m.HandleMailPacket(h.ctx, newMail, h.now, false)
	if h.m.Mail.inboxDirty {
		t.Fatal("unread-count notification treated as new mail")
	}
	binary.LittleEndian.PutUint32(newMail.Data[2:], 44)
	h.m.HandleMailPacket(h.ctx, newMail, h.now, false)
	h.m.UpdateMail(h.ctx, h.now)
	h.quiet()
	if !h.m.Mail.inboxDirty || !h.m.bindings.Windows.MailWindow.IsComposing() {
		t.Fatal("incoming mail changed the draft")
	}
}

func TestMailRefreshReplyAfterReopenDoesNotReplaceNewInbox(t *testing.T) {
	h := newMailTest(t)
	h.action(gameui.MailAction{Kind: gameui.MailActionRefresh})
	h.expect(network.BuildMailResetPacket(network.MailResetAll), network.BuildMailRefreshPacket())
	h.action(gameui.MailAction{Kind: gameui.MailActionClose})
	h.expect(network.BuildMailResetPacket(network.MailResetAll))
	h.m.HandleMailPacket(h.ctx, network.Packet{ID: network.PacketZCMailWindow, Data: []byte{0x60, 2, 0, 0, 0, 0}}, h.now, false)
	h.listReply(42)
	if !h.m.Mail.inboxDirty {
		t.Fatal("cancelled refresh consumed the new inbox request")
	}
	h.m.UpdateMail(h.ctx, h.now)
	h.expect(network.BuildMailResetPacket(network.MailResetAll), network.BuildMailRefreshPacket())
	h.listReply(43)
	if h.m.Mail.inboxDirty || h.m.Mail.Pending.Kind != gameui.MailActionNone {
		t.Fatal("new refresh did not finish")
	}
}

func TestMailMissingReadTargetRefreshesInbox(t *testing.T) {
	h := newMailTest(t)
	h.action(gameui.MailAction{Kind: gameui.MailActionRead, ID: 42})
	h.expect(network.BuildMailReadPacket(42))
	h.result(network.PacketZCMailReturn, network.MailResult{ID: 42, Result: 1})
	if h.m.Mail.Pending.Kind != gameui.MailActionNone || !h.m.Mail.inboxDirty {
		t.Fatal("missing read target left mailbox stuck")
	}
	h.m.UpdateMail(h.ctx, h.now)
	h.expect(network.BuildMailResetPacket(network.MailResetAll), network.BuildMailRefreshPacket())
}

func (h *mailTest) attach() session.InventoryItem {
	h.t.Helper()
	item := h.ctx.Session.Inventory.Items[0]
	item.Amount = 3
	h.action(gameui.MailAction{Kind: gameui.MailActionAttach, Item: item})
	h.expect(network.BuildMailAddAttachmentPacket(7, 3))
	return item
}

func TestMailComposerLayoutIsStableDuringRequestsAndErrors(t *testing.T) {
	h := newMailTest(t)
	h.compose()
	var editor *rotheme.TextAreaWidget
	layout := func() geometry.Rect {
		t.Helper()
		root := h.m.bindings.Windows.MailWindow.Widget()
		ctx := widget.NewContext()
		root.Layout(ctx, geometry.Tight(geometry.Sz(1024, 768)))
		root.Draw(ctx, &uitest.MockCanvas{})
		editor = nil
		var findEditor func(widget.Widget)
		findEditor = func(w widget.Widget) {
			if field, ok := w.(*rotheme.TextAreaWidget); ok {
				editor = field
			}
			for _, child := range w.Children() {
				findEditor(child)
			}
		}
		findEditor(root)
		if editor == nil {
			t.Fatal("composer has no body editor")
		}
		return editor.ScreenBounds()
	}
	want := layout()
	original := editor
	h.attach()
	if got := layout(); got != want || editor != original || editor.IsEnabled() {
		t.Fatalf("pending attachment changed editor layout: got %v, want %v (enabled=%t)", got, want, editor.IsEnabled())
	}
	h.m.UpdateMail(h.ctx, h.now.Add(time.Second))
	if got := layout(); got != want {
		t.Fatalf("waiting for attachment changed editor layout: got %v, want %v", got, want)
	}
	h.m.UpdateMail(h.ctx, h.now.Add(time.Minute))
	if got := layout(); got != want || editor != original || editor.IsEnabled() {
		t.Fatalf("timeout changed editor layout: got %v, want %v (enabled=%t)", got, want, editor.IsEnabled())
	}
	if messages := h.m.bindings.Console.Messages(); len(messages) != 1 || messages[0].Text != "No mail reply from the server. Reconnect before retrying." {
		t.Fatalf("timeout console messages = %+v", messages)
	}
	h.result(network.PacketZCMailAddAttachment, network.MailResult{Index: 7})
	if got := layout(); got != want || editor != original || !editor.IsEnabled() {
		t.Fatalf("attachment reply changed editor layout: got %v, want %v (enabled=%t)", got, want, editor.IsEnabled())
	}
	action := gameui.MailAction{Kind: gameui.MailActionSend, Recipient: "Missing", Title: "Hello"}
	h.action(action)
	packet, _ := network.BuildMailSendPacket(action.Recipient, action.Title, action.Body)
	h.expect(network.BuildMailResetPacket(network.MailResetZeny), packet)
	h.result(network.PacketZCMailSend, network.MailResult{Result: 1})
	h.expect(network.BuildMailResetPacket(network.MailResetAll))
	if got := layout(); got != want || editor != original || !editor.IsEnabled() {
		t.Fatalf("send failure changed editor layout: got %v, want %v (enabled=%t)", got, want, editor.IsEnabled())
	}
	if messages := h.m.bindings.Console.Messages(); len(messages) != 2 || messages[1].Text != "Mail was not sent. Check the recipient and try again." {
		t.Fatalf("send failure console messages = %+v", messages)
	}
}

func TestMailCloseReleasesAcknowledgedAttachment(t *testing.T) {
	h := newMailTest(t)
	h.compose()
	h.attach()
	h.result(network.PacketZCMailAddAttachment, network.MailResult{Index: 7})
	// The UI hides first, then hands the close intent to the game next frame.
	h.m.bindings.Windows.MailWindow.CloseFromServer(h.ctx)
	h.action(gameui.MailAction{Kind: gameui.MailActionClose})
	h.expect(network.BuildMailResetPacket(network.MailResetAll))
	if h.m.bindings.Windows.MailWindow.IsOpen() {
		t.Fatal("closed mailbox still open")
	}
}

func TestMailSendWaitsForZenyAcknowledgement(t *testing.T) {
	for _, cancel := range []bool{false, true} {
		t.Run(map[bool]string{false: "send", true: "cancel"}[cancel], func(t *testing.T) {
			h := newMailTest(t)
			h.compose()
			action := gameui.MailAction{Kind: gameui.MailActionSend, Recipient: "Erika", Title: "Hello", Body: "First line\nSecond line", Zeny: 500}
			h.action(action)
			h.expect(network.BuildMailResetPacket(network.MailResetZeny), network.BuildMailAddAttachmentPacket(0, 500))
			h.action(action)
			h.quiet()
			if cancel {
				h.action(gameui.MailAction{Kind: gameui.MailActionClose})
				h.expect(network.BuildMailResetPacket(network.MailResetAll))
			}
			h.result(network.PacketZCMailAddAttachment, network.MailResult{Index: 0})
			if cancel {
				h.quiet()
				return
			}
			want, _ := network.BuildMailSendPacket(action.Recipient, action.Title, action.Body)
			h.expect(want)
			h.result(network.PacketZCMailSend, network.MailResult{})
			if h.m.bindings.Windows.MailWindow.IsComposing() || h.m.Mail.Pending.Kind != gameui.MailActionNone {
				t.Fatal("successful send did not finish")
			}
			if h.ctx.Session.Inventory.Zeny != 1000 {
				t.Fatal("send fabricated a Zeny status update")
			}
		})
	}
}

func TestMailSendWithoutZenyAndFailure(t *testing.T) {
	h := newMailTest(t)
	h.compose()
	h.attach()
	h.result(network.PacketZCMailAddAttachment, network.MailResult{Index: 7})
	action := gameui.MailAction{Kind: gameui.MailActionSend, Recipient: "Missing", Title: "Hello", Body: "Keep this draft"}
	h.action(action)
	want, _ := network.BuildMailSendPacket(action.Recipient, action.Title, action.Body)
	h.expect(network.BuildMailResetPacket(network.MailResetZeny), want)
	h.result(network.PacketZCMailSend, network.MailResult{Result: 1})
	h.expect(network.BuildMailResetPacket(network.MailResetAll))
	if !h.m.bindings.Windows.MailWindow.IsComposing() || h.m.Mail.Pending.Kind != gameui.MailActionNone {
		t.Fatal("failed send discarded composer or stayed pending")
	}
	if h.ctx.Session.Inventory.Items[0].Amount != 7 {
		t.Fatal("send result applied a second inventory change")
	}
}

func TestMailGetDeleteReturnAndErrors(t *testing.T) {
	for _, kind := range []gameui.MailActionKind{gameui.MailActionTake, gameui.MailActionDelete, gameui.MailActionReturn} {
		for _, code := range []uint16{0, 1, 2} {
			t.Run(fmt.Sprintf("%d/result%d", kind, code), func(t *testing.T) {
				h := newMailTest(t)
				msg := network.MailMessage{ID: 42, Sender: "Erika", Title: "Gift"}
				if kind != gameui.MailActionDelete {
					msg.Zeny = 100
				}
				h.m.bindings.Windows.MailWindow.ShowMail(msg)
				h.action(gameui.MailAction{Kind: kind, ID: 42})
				opcode := network.PacketZCMailGetAttachment
				switch kind {
				case gameui.MailActionTake:
					h.expect(network.BuildMailGetAttachmentPacket(42))
				case gameui.MailActionDelete:
					opcode = network.PacketZCMailDelete
					h.expect(network.BuildMailDeletePacket(42))
				case gameui.MailActionReturn:
					opcode = network.PacketZCMailReturn
					h.expect(network.BuildMailReturnPacket(42, "Erika"))
				}
				h.result(opcode, network.MailResult{ID: 42, Result: code})
				if kind == gameui.MailActionTake && code == 0 {
					h.expect(network.BuildMailReadPacket(42))
					h.readReply(42, 0)
				}
				got, exists := h.m.bindings.Windows.MailWindow.ReadMessage()
				if code != 0 && (!exists || got != msg) {
					t.Fatalf("error modified message: %+v", got)
				}
				if code == 0 && kind == gameui.MailActionTake && got.Zeny != 0 {
					t.Fatal("attachment not cleared")
				}
				if code == 0 && kind != gameui.MailActionTake && exists {
					t.Fatal("removed mail still readable")
				}
			})
		}
	}
}

func TestMailDoesNotDeleteAttachmentsOrSendInvalidDraft(t *testing.T) {
	h := newMailTest(t)
	h.m.bindings.Windows.MailWindow.ShowMail(network.MailMessage{ID: 42, Zeny: 100})
	h.action(gameui.MailAction{Kind: gameui.MailActionDelete, ID: 42})
	h.action(gameui.MailAction{Kind: gameui.MailActionSend, Recipient: "Erika", Title: "Hi", Zeny: 1001})
	h.action(gameui.MailAction{Kind: gameui.MailActionSend, Recipient: "", Title: "Hi"})
	h.quiet()
	if h.m.Mail.Pending.Kind != gameui.MailActionNone {
		t.Fatal("invalid request became pending")
	}
}

func TestMailTimeoutKeepsAnonymousAcknowledgementLedger(t *testing.T) {
	h := newMailTest(t)
	h.m.bindings.Windows.MailWindow.ShowMail(network.MailMessage{ID: 42, Zeny: 100})
	h.action(gameui.MailAction{Kind: gameui.MailActionTake, ID: 42})
	h.expect(network.BuildMailGetAttachmentPacket(42))
	h.m.UpdateMail(h.ctx, h.now.Add(time.Minute))
	if !h.m.Mail.warned || h.m.Mail.Pending.ID != 42 {
		t.Fatal("timeout forgot anonymous ACK correlation")
	}
	h.action(gameui.MailAction{Kind: gameui.MailActionTake, ID: 42})
	h.quiet()
	h.result(network.PacketZCMailGetAttachment, network.MailResult{})
	h.expect(network.BuildMailReadPacket(42))
	h.readReply(42, 0)
	if h.m.Mail.Pending.Kind != gameui.MailActionNone {
		t.Fatal("late result did not release pending request")
	}
}

func TestMailGetItemAndZenyRepliesCannotCompleteNextRetrieval(t *testing.T) {
	h := newMailTest(t)
	h.m.bindings.Windows.MailWindow.ShowMail(network.MailMessage{ID: 42, Zeny: 100, Attachment: network.MailAttachment{ItemID: 501, Amount: 3}})
	h.action(gameui.MailAction{Kind: gameui.MailActionTake, ID: 42})
	h.expect(network.BuildMailGetAttachmentPacket(42))
	h.result(network.PacketZCMailGetAttachment, network.MailResult{})
	h.expect(network.BuildMailReadPacket(42))
	h.action(gameui.MailAction{Kind: gameui.MailActionRead, ID: 43})
	h.result(network.PacketZCMailGetAttachment, network.MailResult{})
	if !h.m.Mail.readingAfterTake || h.m.Mail.Pending.ID != 42 {
		t.Fatal("duplicate retrieval ACK completed the read barrier")
	}
	h.quiet()
	h.readReply(42, 0)
	msg, _ := h.m.bindings.Windows.MailWindow.ReadMessage()
	if msg.Zeny != 0 || msg.Attachment.ItemID != 0 || h.m.Mail.Pending.Kind != gameui.MailActionNone {
		t.Fatal("read did not finish retrieval")
	}
}
