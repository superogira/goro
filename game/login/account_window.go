package login

import (
	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/glog"
	"github.com/kivutar/goro/res"
	gameui "github.com/kivutar/goro/ui"
)

func (m *Screen) updateAccountInput(ctx client.Context) bool {
	m.updateAccountWindow(ctx)
	if m.serviceWindow != nil {
		window := m.serviceWindow
		consumed := window.Update(ctx)
		if m.serviceWindow == window {
			window.Publish(ctx)
		}
		return consumed
	}
	if m.loginWindow != nil {
		consumed := m.loginWindow.Update(ctx)
		m.loginWindow.Publish(ctx)
		return consumed
	}
	return false
}

func (m *Screen) updateAccountWindow(ctx client.Context) {
	switch m.accountStep {
	case loginAccountConnection:
		if len(loginConnections(ctx)) > 1 && !ctx.Config.Login.AutoLogin {
			m.updateLoginServerWindow(ctx)
			return
		}
	case loginAccountCharacterService:
		m.updateCharacterServiceWindow(ctx)
		return
	case loginAccountCharacterConnecting:
		return
	}
	if m.accountStep == loginAccountConnection {
		if len(loginConnections(ctx)) == 0 {
			m.status = "no login servers discovered"
		} else {
			m.status = "enter account credentials"
		}
	}
	m.accountStep = loginAccountCredentials
	m.updateLoginWindow(ctx)
}

func (m *Screen) updateLoginWindow(ctx client.Context) {
	if ctx.Config.Headless {
		return
	}
	if m.loginWindow == nil {
		m.loginWindow = gameui.NewLoginWindow(ctx, m.username, m.password, m.keepID, gameui.LoginWindowCallbacks{
			OnSubmit: func() {
				// FieldValues reads the live widgets: the OSK types via
				// SetText, which never fires OnChange, so the Username/
				// Password backing fields can be stale until the next
				// submit sync — submitting those sent an empty login.
				m.username, m.password = m.loginWindow.FieldValues()
				m.keepID = m.loginWindow.KeepID
				m.saveLoginID(ctx)
				if conn, ok := m.selectedLoginConnection(ctx); ok {
					m.connectAndMaybeLogin(ctx, conn, true)
				}
			},
		})
		m.loginWindow.Publish(ctx)
		return
	}
	m.loginWindow.SetContext(ctx)
	m.username = m.loginWindow.Username
	m.password = m.loginWindow.Password
	m.loginWindow.Publish(ctx)
}

func (m *Screen) saveLoginID(ctx client.Context) {
	if ctx.Session != nil {
		ctx.Session.KeepLoginID = m.keepID
		ctx.Session.SavedUsername = ""
		if m.keepID {
			ctx.Session.SavedUsername = m.username
		}
	}
	if _, err := ctx.Config.SaveLoginID(m.username, m.keepID); err != nil {
		glog.Warnf("login ID save failed: %v", err)
	}
}

func (m *Screen) selectedLoginConnection(ctx client.Context) (res.Connection, bool) {
	connections := loginConnections(ctx)
	if len(connections) == 0 {
		return res.Connection{}, false
	}
	if m.selectedLoginServer < 0 || m.selectedLoginServer >= len(connections) {
		m.selectedLoginServer = 0
	}
	return connections[m.selectedLoginServer], true
}

func loginConnections(ctx client.Context) []res.Connection {
	if ctx.Resources == nil {
		return nil
	}
	return ctx.Resources.ClientInfo.Connections
}

// UpdateAccountInputOSK routes the handheld on-screen keyboard while the
// credential form is up: the OSK types straight into the focused field.
// It reports whether the OSK consumed the frame.
func (m *Screen) UpdateAccountInputOSK(ctx client.Context) bool {
	if m.phase != loginPhaseAccount || m.accountStep != loginAccountCredentials {
		return false
	}
	m.updateAccountInput(ctx)
	return true
}

// LoginWindowFieldFocus reports the credential form's focused field.
func (m *Screen) LoginWindowFieldFocus() (userFocused, passwordFocused bool) {
	if m.loginWindow == nil {
		return false, false
	}
	return m.loginWindow.FieldFocus()
}

// ToggleLoginKeep flips the Keep-ID checkbox row.
func (m *Screen) ToggleLoginKeep() {
	if m.loginWindow == nil {
		return
	}
	m.loginWindow.ToggleKeep()
	m.keepID = m.loginWindow.KeepID
}

// AdvanceLoginFocus swaps Account <-> Password.
func (m *Screen) AdvanceLoginFocus() {
	if m.loginWindow == nil {
		return
	}
	m.loginWindow.AdvanceFocus()
}

// TypeLoginFocused types one OSK stroke into the focused credential field.
func (m *Screen) TypeLoginFocused(ch, action string) {
	if m.loginWindow == nil {
		return
	}
	m.loginWindow.TypeIntoFocusedField(ch, action)
	m.username = m.loginWindow.Username
	m.password = m.loginWindow.Password
}

// LoginCredentials returns the current typed credential values.
func (m *Screen) LoginCredentials() (string, string) {
	return m.username, m.password
}

// LoginKeepID reports the Keep-ID state.
func (m *Screen) LoginKeepID() bool {
	return m.keepID
}
