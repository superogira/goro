package game

import (
	"fmt"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/glog"
	"github.com/kivutar/goro/network"
	gameui "github.com/kivutar/goro/ui"
)

const storagePasswordMaximumFailures = 3

func (m *WorldMode) handleStoragePasswordPrompt(ctx client.Context, prompt network.StoragePasswordPrompt) {
	if ctx.Session == nil {
		return
	}
	ctx.Session.Storage.PasswordPending = true
	ctx.Session.Storage.Open = false
	m.ui.storageWindow.SetOpen(false)

	switch prompt.State {
	case network.StoragePasswordNotSet:
		m.ui.storagePassword.OpenPrompt(ctx, gameui.StoragePasswordModeSet, "Set a password before opening storage.")
	case network.StoragePasswordSet:
		m.ui.storagePassword.OpenPrompt(ctx, gameui.StoragePasswordModeEnter, "Enter your storage password.")
	case network.StoragePasswordLocked:
		m.ui.storagePassword.OpenPrompt(ctx, gameui.StoragePasswordModeLocked, "Storage is locked after too many failed attempts.")
	default:
		glog.Warnf("unknown storage password prompt state=%d", prompt.State)
		m.ui.storagePassword.OpenPrompt(ctx, gameui.StoragePasswordModeLocked, "Storage authentication is unavailable.")
	}
}

func (m *WorldMode) handleStoragePasswordResult(ctx client.Context, result network.StoragePasswordResult) {
	if ctx.Session == nil || !ctx.Session.Storage.PasswordPending {
		glog.Debugf("ignored storage password result=%d without a pending prompt", result.Code)
		return
	}

	switch result.Code {
	case network.StoragePasswordChangeSucceeded:
		m.ui.storagePassword.OpenPrompt(ctx, gameui.StoragePasswordModeEnter, "Password changed. Enter it to open storage.")
	case network.StoragePasswordChangeFailed:
		m.ui.storagePassword.OpenPrompt(ctx, gameui.StoragePasswordModeSet, "The password could not be changed.")
	case network.StoragePasswordCheckSucceeded:
		ctx.Session.Storage.PasswordPending = false
		ctx.Session.Storage.Open = true
		m.ui.storagePassword.CloseFromServer(ctx)
		m.ui.storageWindow.OpenWindow(ctx)
	case network.StoragePasswordCheckFailed:
		remaining := maxInt(0, storagePasswordMaximumFailures-int(result.ErrorCount))
		message := fmt.Sprintf("Wrong password. %d attempt(s) left.", remaining)
		m.ui.storagePassword.OpenPrompt(ctx, gameui.StoragePasswordModeEnter, message)
	case network.StoragePasswordTooManyFailures:
		m.ui.storagePassword.OpenPrompt(ctx, gameui.StoragePasswordModeLocked, "Storage is locked after too many failed attempts.")
	default:
		glog.Warnf("unknown storage password result=%d errors=%d", result.Code, result.ErrorCount)
		m.ui.storagePassword.ShowMessage(ctx, "Storage authentication failed.")
	}
}

func (m *WorldMode) updateStoragePasswordWindow(ctx client.Context) bool {
	if !m.ui.storagePassword.Update(ctx) {
		return false
	}
	action := m.ui.storagePassword.PopAction()
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
		applyStorageClosed(ctx)
	}
	return true
}

func (m *WorldMode) sendStoragePasswordReply(ctx client.Context, replyType network.StoragePasswordReplyType, password, newPassword string) {
	if ctx.Network == nil {
		m.ui.storagePassword.ShowMessage(ctx, "Unable to contact the server.")
		return
	}
	if err := ctx.Network.SendStoragePasswordReply(replyType, password, newPassword); err != nil {
		glog.Warnf("storage password reply failed: %v", err)
		m.ui.storagePassword.ShowMessage(ctx, "Unable to contact the server.")
	}
}
