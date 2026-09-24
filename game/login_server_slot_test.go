package game

import (
	"fmt"
	"strings"
	"testing"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/config"
	"github.com/kivutar/goro/network"
	"github.com/kivutar/goro/res"
	"github.com/kivutar/goro/session"
)

func TestAutologinRejectsUnavailableServerSlots(t *testing.T) {
	for _, headless := range []bool{false, true} {
		for _, count := range []int{0, 2} {
			t.Run(fmt.Sprintf("headless=%t/servers=%d", headless, count), func(t *testing.T) {
				for _, slot := range []int{-1, count, count + 10} {
					ctx := client.Context{
						Config:    config.Config{Headless: headless, Login: config.LoginConfig{AutoLogin: true, ServerSlot: slot}},
						Resources: loginTestResources(make([]res.Connection, count)...),
						Session:   &session.Session{},
					}
					// A nil network also ensures an invalid slot never falls back
					// to sending credentials to another server.
					mode := NewLoginMode()
					_, err := mode.Update(ctx)
					if err == nil || !strings.Contains(err.Error(), fmt.Sprintf("--server-slot %d is unavailable", slot)) {
						t.Fatalf("slot %d error = %v", slot, err)
					}
					ctx.Config.Login.CharServerSlot = slot
					err = mode.applyAccountAcceptLogin(ctx, network.AccountAcceptLogin{CharServer: make([]network.CharServer, count)})
					if err == nil || !strings.Contains(err.Error(), fmt.Sprintf("--char-server-slot %d is unavailable", slot)) {
						t.Fatalf("character server slot %d error = %v", slot, err)
					}
				}
			})
		}
	}
}
