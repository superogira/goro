package testutil

import (
	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/db"
	"github.com/kivutar/goro/res"
	"github.com/kivutar/goro/session"
	"github.com/kivutar/goro/world"
)

func AssassinEffectContext() client.Context {
	w := world.New()
	w.Player = world.Actor{ID: 100, X: 10, Y: 20, Job: db.JobAssassin}
	w.Actors[300] = world.Actor{ID: 300, X: 16, Y: 20, Job: 1002}
	return client.Context{World: w, Resources: &res.Manager{}, Session: &session.Session{AccountID: 2000000, CharID: 100}}
}
