package npc

import (
	"fmt"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/game/scene"
	"github.com/kivutar/goro/glog"
	"github.com/kivutar/goro/network"
	"github.com/kivutar/goro/session"
	gameui "github.com/kivutar/goro/ui"
)

const storagePasswordMaximumFailures = 3

func (m *System) HandleStoragePasswordPrompt(ctx client.Context, prompt network.StoragePasswordPrompt) {
	if ctx.Session == nil {
		return
	}
	ctx.Session.Storage.PasswordPending = true
	ctx.Session.Storage.Open = false
	m.bindings.Storage.SetOpen(false)

	switch prompt.State {
	case network.StoragePasswordNotSet:
		m.bindings.Windows.StoragePassword.OpenPrompt(ctx, gameui.StoragePasswordModeSet, "Set a password before opening storage.")
	case network.StoragePasswordSet:
		m.bindings.Windows.StoragePassword.OpenPrompt(ctx, gameui.StoragePasswordModeEnter, "Enter your storage password.")
	case network.StoragePasswordLocked:
		m.bindings.Windows.StoragePassword.OpenPrompt(ctx, gameui.StoragePasswordModeLocked, "Storage is locked after too many failed attempts.")
	default:
		glog.Warnf("unknown storage password prompt state=%d", prompt.State)
		m.bindings.Windows.StoragePassword.OpenPrompt(ctx, gameui.StoragePasswordModeLocked, "Storage authentication is unavailable.")
	}
}

func (m *System) HandleStoragePasswordResult(ctx client.Context, result network.StoragePasswordResult) {
	if ctx.Session == nil || !ctx.Session.Storage.PasswordPending {
		glog.Debugf("ignored storage password result=%d without a pending prompt", result.Code)
		return
	}

	switch result.Code {
	case network.StoragePasswordChangeSucceeded:
		m.bindings.Windows.StoragePassword.OpenPrompt(ctx, gameui.StoragePasswordModeEnter, "Password changed. Enter it to open storage.")
	case network.StoragePasswordChangeFailed:
		m.bindings.Windows.StoragePassword.OpenPrompt(ctx, gameui.StoragePasswordModeSet, "The password could not be changed.")
	case network.StoragePasswordCheckSucceeded:
		ctx.Session.Storage.PasswordPending = false
		ctx.Session.Storage.Open = true
		m.bindings.Windows.StoragePassword.CloseFromServer(ctx)
		m.bindings.Storage.OpenWindow(ctx)
	case network.StoragePasswordCheckFailed:
		remaining := scene.MaxInt(0, storagePasswordMaximumFailures-int(result.ErrorCount))
		message := fmt.Sprintf("Wrong password. %d attempt(s) left.", remaining)
		m.bindings.Windows.StoragePassword.OpenPrompt(ctx, gameui.StoragePasswordModeEnter, message)
	case network.StoragePasswordTooManyFailures:
		m.bindings.Windows.StoragePassword.OpenPrompt(ctx, gameui.StoragePasswordModeLocked, "Storage is locked after too many failed attempts.")
	default:
		glog.Warnf("unknown storage password result=%d errors=%d", result.Code, result.ErrorCount)
		m.bindings.Windows.StoragePassword.ShowMessage(ctx, "Storage authentication failed.")
	}
}

func (m *System) UpdateStoragePassword(ctx client.Context) bool {
	if !m.bindings.Windows.StoragePassword.Update(ctx) {
		return false
	}
	action := m.bindings.Windows.StoragePassword.PopAction()
	switch action.Kind {
	case gameui.StoragePasswordActionCheck:
		m.sendStoragePasswordReply(ctx, network.StoragePasswordCheck, action.Password, "")
	case gameui.StoragePasswordActionChange:
		m.sendStoragePasswordReply(ctx, network.StoragePasswordChange, "", action.Password)
	case gameui.StoragePasswordActionCancel:
		if ctx.Network != nil {
			if err := ctx.Network.SendCloseStorage(); err != nil {
				glog.Warnf("storage password cancel failed: %v", err)
			}
		}
		session.CloseStorage(ctx.Session)
	}
	return true
}

func (m *System) sendStoragePasswordReply(ctx client.Context, replyType network.StoragePasswordReplyType, password, newPassword string) {
	if ctx.Network == nil {
		m.bindings.Windows.StoragePassword.ShowMessage(ctx, "Unable to contact the server.")
		return
	}
	if err := ctx.Network.SendStoragePasswordReply(replyType, password, newPassword); err != nil {
		glog.Warnf("storage password reply failed: %v", err)
		m.bindings.Windows.StoragePassword.ShowMessage(ctx, "Unable to contact the server.")
	}
}
