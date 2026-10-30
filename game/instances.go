package game

import (
	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/glog"
	"github.com/kivutar/goro/network"
	"github.com/kivutar/goro/session"
)

func (m *WorldMode) handleInstancePacket(ctx client.Context, pkt network.Packet) bool {
	if pkt.ID < network.PacketZCInstanceQueue || pkt.ID > network.PacketZCInstanceNotify {
		return false
	}
	if ctx.Session == nil || m.ui.disconnectDialog.IsOpen() {
		return true
	}
	var err error
	message := ""
	i := &ctx.Session.Instance
	switch pkt.ID {
	case network.PacketZCInstanceQueue:
		var queue network.InstanceQueue
		queue, _, err = network.ParseInstanceQueue(pkt)
		if err == nil {
			*i = session.Instance{Name: queue.Name, Queued: true, QueuePosition: queue.Position}
			if queue.Position < 0 {
				*i = session.Instance{}
				message = "Memorial dungeon reservation cancelled."
			}
		}
	case network.PacketZCInstanceQueuePosition:
		var position int16
		position, _, err = network.ParseInstanceQueuePosition(pkt)
		if err == nil {
			if position < 0 {
				*i = session.Instance{}
				message = "Memorial dungeon reservation cancelled."
			} else if i.Name != "" {
				i.Queued, i.QueuePosition = true, position
			}
		}
	case network.PacketZCInstanceInfo:
		var info network.InstanceInfo
		info, _, err = network.ParseInstanceInfo(pkt)
		if err == nil {
			*i = session.Instance{Name: info.Name, ExpiresAt: info.ExpiresAt, IdleExpiresAt: info.IdleExpiresAt}
		}
	case network.PacketZCInstanceNotify:
		var notify network.InstanceNotify
		notify, _, err = network.ParseInstanceNotify(pkt)
		if err == nil {
			switch notify.Type {
			case network.InstanceNotifyTime:
				if i.Name != "" {
					i.Queued, i.IdleExpiresAt = false, notify.IdleExpiresAt
				}
			case network.InstanceNotifyExpired:
				message = "The memorial dungeon's time limit expired. It has been destroyed."
			case network.InstanceNotifyIdleExpired:
				message = "The memorial dungeon's entry time limit expired. It has been destroyed."
			case network.InstanceNotifyDestroyed:
				message = "The memorial dungeon has been removed."
			case network.InstanceNotifyCreateFailed:
				// Also used by Hercules to hide the window when leaving a party.
				// Like the original client, close it without a failure message.
			}
			if notify.Type != network.InstanceNotifyTime {
				*i = session.Instance{}
			}
		}
	default:
		return false
	}
	if err != nil {
		glog.Warnf("parse instance packet 0x%04X: %v", pkt.ID, err)
		return true
	}
	if message != "" {
		m.ui.console.AddSystemMessage("%s", message)
	}
	if !ctx.Config.Headless {
		m.ui.instanceWindow.Sync(ctx)
	}
	return true
}

func (m *WorldMode) destroyInstance(ctx client.Context) {
	if !ctx.Session.CanDestroyInstance() || m.ui.social.TradeWindow.IsOpen() {
		return
	}
	if ctx.Network == nil {
		m.ui.console.AddErrorMessage("Destroy instance failed: not connected.")
		return
	}
	if err := ctx.Network.SendInstanceDestroy(); err != nil {
		m.ui.console.AddErrorMessage("Destroy instance failed.")
		glog.Warnf("instance destroy request: %v", err)
	}
}

func (m *WorldMode) resetInstance(ctx client.Context) {
	if ctx.Session != nil {
		ctx.Session.Instance = session.Instance{}
	}
	m.ui.instanceWindow.Close()
}
