package testutil

import (
	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/session"
	worldstate "github.com/kivutar/goro/world"
)

func Level99AuraTestContext(baseLevel int, lessEffects bool) client.Context {
	return client.Context{
		Session: &session.Session{
			AccountID:   2000000,
			CharID:      2000000,
			LessEffects: lessEffects,
			Progress: session.Progress{
				BaseLevel: baseLevel,
			},
			Selected: session.Character{
				ID:    2000000,
				Level: int16(baseLevel),
			},
		},
		World: &worldstate.World{
			Player: worldstate.Actor{ID: 2000000, X: 10, Y: 20},
			Actors: map[uint32]worldstate.Actor{},
		},
	}
}
