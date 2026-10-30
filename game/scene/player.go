package scene

import (
	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/session"
)

func PlayerIsDead(ctx client.Context) bool {
	return ctx.Session != nil && ctx.Session.Dead
}

func SelectedCharacterName(s *session.Session) string {
	if s == nil {
		return ""
	}
	return s.SelectedCharacter().Name
}
