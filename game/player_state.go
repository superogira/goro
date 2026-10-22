package game

import (
	"github.com/kivutar/goro/client"
)

func playerIsDead(ctx client.Context) bool {
	return ctx.Session != nil && ctx.Session.Dead
}
