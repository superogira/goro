package social

import (
	"fmt"
	"time"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/glog"
	"github.com/kivutar/goro/network"
	"github.com/kivutar/goro/session"
	gameui "github.com/kivutar/goro/ui"
)

type MailState struct {
	Pending          gameui.MailAction
	waitingZeny      bool
	readingAfterTake bool
	cancelled        bool
	inboxDirty       bool
	since            time.Time
	warned           bool
}

func (m *System) UpdateMail(ctx client.Context, now time.Time) {
	for {
		action := m.bindings.Windows.MailWindow.PopAction()
		if action.Kind == gameui.MailActionNone {
			break
		}
		m.HandleMailAction(ctx, action, now)
	}
	if m.Mail.Pending.Kind != gameui.MailActionNone {
		// Anonymous legacy ACKs cannot safely be forgotten on timeout: a late
		// reply could otherwise complete the next send/take operation instead.
		if !m.Mail.warned && now.Sub(m.Mail.since) > 10*time.Second {
			m.Mail.warned = true
			m.MailError("No mail reply from the server. Reconnect before retrying.")
		}
		return
	}
	if m.Mail.inboxDirty && m.bindings.Windows.MailWindow.IsOpen() && !m.bindings.Windows.MailWindow.IsComposing() {
		m.HandleMailAction(ctx, gameui.MailAction{Kind: gameui.MailActionRefresh}, now)
	}
}

func (m *System) HandleMailAction(ctx client.Context, action gameui.MailAction, now time.Time) {
	if action.Kind == gameui.MailActionClose {
		m.CloseMail(ctx)
		return
	}
	if !m.bindings.Windows.MailWindow.IsOpen() || m.Mail.Pending.Kind != gameui.MailActionNone {
		return
	}
	if ctx.Network == nil {
		m.bindings.Windows.MailWindow.SetBusy(false)
		m.MailError("Unable to contact the mail server.")
		return
	}
	var err error
	switch action.Kind {
	case gameui.MailActionCompose:
		err = ctx.Network.SendMailReset(network.MailResetAll)
		if err == nil {
			m.bindings.Windows.MailWindow.BeginCompose(action.Recipient, action.Title)
		}
	case gameui.MailActionRemoveAttachment:
		err = ctx.Network.SendMailReset(network.MailResetItem)
		if err == nil {
			m.bindings.Windows.MailWindow.SetAttachment(session.InventoryItem{})
		}
	case gameui.MailActionRefresh:
		err = ctx.Network.SendMailReset(network.MailResetAll)
		if err == nil {
			err = ctx.Network.SendMailRefresh()
		}
		if err == nil {
			m.bindings.Windows.MailWindow.ShowInbox()
		}
	case gameui.MailActionRead:
		if action.ID == 0 {
			m.bindings.Windows.MailWindow.SetBusy(false)
			return
		}
		err = ctx.Network.SendMailRead(action.ID)
	case gameui.MailActionTake, gameui.MailActionDelete, gameui.MailActionReturn:
		message, ok := m.bindings.Windows.MailWindow.ReadMessage()
		if !ok || message.ID != action.ID || action.ID == 0 {
			m.bindings.Windows.MailWindow.SetBusy(false)
			return
		}
		switch action.Kind {
		case gameui.MailActionTake:
			if !message.HasAttachment() {
				m.bindings.Windows.MailWindow.SetBusy(false)
				return
			}
			err = ctx.Network.SendMailGetAttachment(action.ID)
		case gameui.MailActionDelete:
			if message.HasAttachment() {
				m.bindings.Windows.MailWindow.SetBusy(false)
				m.MailError("Take the attachments before deleting this mail.")
				return
			}
			err = ctx.Network.SendMailDelete(action.ID)
		case gameui.MailActionReturn:
			err = ctx.Network.SendMailReturn(action.ID, message.Sender)
		}
	case gameui.MailActionAttach:
		item, ok := session.FindInventoryItem(ctx.Session, action.Item.Index)
		if !ok || item.Index < 2 || item.ItemID != action.Item.ItemID || item.Equipped || action.Item.Amount <= 0 || action.Item.Amount > min(item.Amount, 32767) {
			m.bindings.Windows.MailWindow.SetBusy(false)
			m.MailError("This item is no longer available to attach.")
			return
		}
		err = ctx.Network.SendMailAddAttachment(item.Index, uint32(action.Item.Amount))
	case gameui.MailActionSend:
		// Validate before reserving Zeny or transmitting any part of a send.
		_, err = network.BuildMailSendPacket(action.Recipient, action.Title, action.Body)
		if err == nil && (ctx.Session == nil || uint64(action.Zeny) > uint64(max(0, ctx.Session.Inventory.Zeny))) {
			err = fmt.Errorf("not enough Zeny")
		}
		if err == nil {
			err = ctx.Network.SendMailReset(network.MailResetZeny)
		}
		if err == nil {
			if action.Zeny > 0 {
				err = ctx.Network.SendMailAddAttachment(0, action.Zeny)
			} else {
				err = ctx.Network.SendMail(action.Recipient, action.Title, action.Body)
			}
		}
	default:
		return
	}
	if err != nil {
		m.bindings.Windows.MailWindow.SetBusy(false)
		m.MailError("Mail request failed: " + err.Error())
		return
	}
	if action.Kind == gameui.MailActionCompose || action.Kind == gameui.MailActionRemoveAttachment {
		m.bindings.Windows.MailWindow.SetBusy(false)
		return
	}
	m.Mail.Pending = action
	m.Mail.cancelled = false
	m.Mail.waitingZeny = action.Kind == gameui.MailActionSend && action.Zeny > 0
	m.Mail.since = now
	m.Mail.warned = false
	m.bindings.Windows.MailWindow.SetBusy(true)
}

func (m *System) CloseMail(ctx client.Context) {
	// A user close has already hidden the window; its reservation still needs
	// releasing even when there is no outstanding acknowledgement.
	if ctx.Network != nil {
		if err := ctx.Network.SendMailReset(network.MailResetAll); err != nil {
			glog.Warnf("mail reset on close failed: %v", err)
		}
	}
	m.bindings.Windows.MailWindow.CloseFromServer(ctx)
	m.Mail.cancelled = true
	m.Mail.inboxDirty = false
	// Preserve any outstanding ACK until it arrives, even if another mailbox
	// is opened on this same connection. Character selection creates a new
	// WorldMode; map changes on the same connection carry this ledger forward.
}

func (m *System) finishMailOperation() bool {
	visible := !m.Mail.cancelled && m.bindings.Windows.MailWindow.IsOpen()
	m.Mail.Pending = gameui.MailAction{}
	m.Mail.waitingZeny = false
	m.Mail.readingAfterTake = false
	m.Mail.cancelled = false
	m.Mail.warned = false
	m.bindings.Windows.MailWindow.SetBusy(false)
	return visible
}

func (m *System) MailError(message string) {
	m.bindings.Console.AddErrorMessage("%s", message)
}

func (m *System) HandleMailPacket(ctx client.Context, pkt network.Packet, now time.Time, disconnected bool) bool {
	var err error
	switch pkt.ID {
	case network.PacketZCMailWindow:
		if disconnected {
			return true
		}
		var open bool
		open, _, err = network.ParseMailWindow(pkt)
		if err != nil {
			break
		}
		if !open {
			m.CloseMail(ctx)
			return true
		}
		if m.bindings.Windows.MailWindow.IsOpen() {
			return true
		}
		m.bindings.Windows.MailWindow.Open(ctx, m.MailError)
		m.Mail.inboxDirty = true
		m.bindings.Windows.MailWindow.SetBusy(m.Mail.Pending.Kind != gameui.MailActionNone)
		m.UpdateMail(ctx, now)
	case network.PacketZCMailList:
		var entries []network.MailEntry
		entries, _, err = network.ParseMailList(pkt)
		if err != nil {
			break
		}
		if m.bindings.Windows.MailWindow.IsOpen() && !m.Mail.cancelled {
			m.bindings.Windows.MailWindow.SetInbox(entries)
			m.Mail.inboxDirty = false
		}
		if m.Mail.Pending.Kind == gameui.MailActionRefresh {
			m.finishMailOperation()
		}
	case network.PacketZCMailRead:
		var message network.MailMessage
		message, _, err = network.ParseMailRead(pkt)
		if err != nil {
			break
		}
		if (m.Mail.Pending.Kind == gameui.MailActionRead || m.Mail.readingAfterTake) && message.ID == m.Mail.Pending.ID {
			refresh := m.Mail.readingAfterTake
			if m.finishMailOperation() {
				if refresh {
					m.bindings.Windows.MailWindow.UpdateReadMessage(message)
				} else {
					m.bindings.Windows.MailWindow.ShowMail(message)
				}
			}
		}
	case network.PacketZCMailNew:
		var entry network.MailEntry
		entry, _, err = network.ParseMailNew(pkt)
		if err != nil {
			break
		}
		// ID zero is a legacy unread-status refresh, not a new message.
		if entry.ID != 0 {
			m.bindings.Console.AddBlueMessage("You have new mail.")
			m.Mail.inboxDirty = true
		}
	case network.PacketZCMailAddAttachment, network.PacketZCMailGetAttachment, network.PacketZCMailSend, network.PacketZCMailDelete, network.PacketZCMailReturn:
		var result network.MailResult
		result, _, err = network.ParseMailResult(pkt)
		if err == nil {
			m.handleMailResult(ctx, pkt.ID, result, now)
		}
	default:
		return false
	}
	if err != nil {
		glog.Errorf("parse legacy mail 0x%04X: %v", pkt.ID, err)
	}
	return true
}

func (m *System) handleMailResult(ctx client.Context, opcode uint16, result network.MailResult, now time.Time) {
	pending := m.Mail.Pending
	switch opcode {
	case network.PacketZCMailAddAttachment:
		if pending.Kind == gameui.MailActionSend && m.Mail.waitingZeny && result.Index == 0 {
			if result.Result != 0 {
				m.finishMailOperation()
				m.MailError("The Zeny could not be attached.")
				return
			}
			if m.Mail.cancelled {
				m.finishMailOperation()
				return
			}
			m.Mail.waitingZeny = false
			m.Mail.since = now
			if err := ctx.Network.SendMail(pending.Recipient, pending.Title, pending.Body); err != nil {
				m.finishMailOperation()
				m.MailError("Sending mail failed: " + err.Error())
			}
			return
		}
		if pending.Kind != gameui.MailActionAttach || result.Index != pending.Item.Index {
			return
		}
		if result.Result == 0 {
			// Legacy ACK_MAIL_ADD_ITEM removes the reserved stack from the
			// client inventory. Reset/failure returns it via normal add-item
			// packets; successful delivery does not send a second deletion.
			session.RemoveInventoryItem(ctx.Session, pending.Item.Index, pending.Item.Amount)
			if m.finishMailOperation() {
				m.bindings.Windows.MailWindow.SetAttachment(pending.Item)
			}
		} else {
			m.finishMailOperation()
			m.MailError("This item cannot be attached.")
		}
	case network.PacketZCMailGetAttachment:
		if pending.Kind != gameui.MailActionTake || m.Mail.readingAfterTake {
			return
		}
		if result.Result == 0 {
			// Some legacy servers send a success for the item and another for
			// Zeny. Re-read before accepting another operation: the ID-bearing
			// reply refreshes attachments and separates those anonymous ACKs
			// from the next retrieval, even if TCP splits them across frames.
			m.Mail.readingAfterTake = true
			m.Mail.since = now
			if err := ctx.Network.SendMailRead(pending.ID); err != nil {
				m.MailError("Unable to refresh the attachment: " + err.Error())
			}
			return
		}
		m.finishMailOperation()
		if result.Result == 1 {
			m.MailError("Cannot receive the attachment: the Zeny limit would be exceeded.")
		} else {
			m.MailError("Cannot receive the attachment: not enough inventory space or weight capacity.")
		}
	case network.PacketZCMailSend:
		if pending.Kind != gameui.MailActionSend || m.Mail.waitingZeny {
			return
		}
		visible := m.finishMailOperation()
		if result.Result == 0 {
			m.bindings.Console.AddBlueMessage("Mail sent.")
			if visible {
				m.bindings.Windows.MailWindow.ShowInbox()
				m.Mail.inboxDirty = true
			}
		} else {
			// The server restores attachments using normal inventory/status
			// packets. Preserve the message text, but do not reuse reservations.
			if ctx.Network != nil {
				_ = ctx.Network.SendMailReset(network.MailResetAll)
			}
			m.bindings.Windows.MailWindow.SetAttachment(session.InventoryItem{})
			m.MailError("Mail was not sent. Check the recipient and try again.")
		}
	case network.PacketZCMailDelete, network.PacketZCMailReturn:
		if (pending.Kind == gameui.MailActionRead || m.Mail.readingAfterTake) && opcode == network.PacketZCMailReturn && result.ID == pending.ID && result.Result != 0 {
			// rAthena uses ACK_MAIL_RETURN to report a missing read target.
			m.finishMailOperation()
			m.MailError("This mail is no longer available.")
			m.Mail.inboxDirty = true
			return
		}
		kind := gameui.MailActionDelete
		if opcode == network.PacketZCMailReturn {
			kind = gameui.MailActionReturn
		}
		if pending.Kind != kind || result.ID != pending.ID {
			return
		}
		m.finishMailOperation()
		if result.Result == 0 {
			m.bindings.Windows.MailWindow.RemoveMail(result.ID)
			m.Mail.inboxDirty = true
		} else if kind == gameui.MailActionDelete {
			m.MailError("Mail could not be deleted. Take its attachments first.")
		} else {
			m.MailError("This mail could not be returned to its sender.")
		}
	}
}
