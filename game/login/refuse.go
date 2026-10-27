package login

import (
	"strings"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/glog"
	"github.com/kivutar/goro/network"
	"github.com/kivutar/goro/res"
)

var loginRefuseMessages = map[uint8]disconnectMessage{
	0:   {6, "Unregistered ID."},
	1:   {267, "Incorrect Password."},
	2:   {8, "This ID has expired."},
	3:   {3, "Rejected from Server."},
	4:   {266, "Login is currently unavailable. Please try again shortly."},
	5:   {310, "Your game's executable is not the latest version."},
	6:   {449, "You are prohibited from logging in until %s."},
	7:   {439, "Server is jammed due to overpopulation."},
	8:   {681, "This account cannot connect to this server."},
	9:   {703, "This account has been blocked by the database administrator."},
	10:  {704, "Email confirmation is required."},
	11:  {705, "This account has been blocked by the GM team."},
	12:  {706, "Login is temporarily unavailable for database work."},
	13:  {707, "This account is self-locked."},
	14:  {708, "This account is not permitted to connect."},
	15:  {709, "This account is not permitted to connect."},
	99:  {368, "This account has been erased."},
	100: {809, "Login information remains at %s."},
	101: {810, "This account has been locked for an investigation."},
	102: {811, "This account has been temporarily prohibited from logging in."},
	103: {859, "This character is being deleted. Login is temporarily unavailable."},
	104: {860, "This character is being deleted. Login is temporarily unavailable."},
}

func loginRefuseMessage(resources *res.Manager, refusal network.AccountRefuseLogin) string {
	message, ok := loginRefuseMessages[refusal.ErrorCode]
	if !ok {
		message = disconnectMessage{9, "Login failed."}
	}
	text := MessageText(resources, message.msgID, message.fallback)
	return strings.Replace(text, "%s", refusal.UnblockTime, 1)
}

func (m *Screen) applyAccountRefuseLogin(ctx client.Context, refusal network.AccountRefuseLogin) {
	m.loginPending = false
	if ctx.Network != nil {
		ctx.Network.Close()
	}
	message := loginRefuseMessage(ctx.Resources, refusal)
	m.status = message
	glog.Warnf("login refused code=%d", refusal.ErrorCode)
	if m.disconnectDialog.IsOpen() {
		return
	}
	m.disconnectDialog.OpenAlert(ctx, "Login failed", message, func() {
		m.status = "enter account credentials"
	})
}
