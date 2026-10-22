package testutil

import (
	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/db"
	"github.com/kivutar/goro/session"
	"github.com/kivutar/goro/world"
)

func ArcherEffectContext() client.Context {
	w := world.New()
	w.Player = world.Actor{ID: 100, Job: db.JobArcher, Weapon: db.WeaponBow, X: 10, Y: 20}
	w.Actors[300] = world.Actor{ID: 300, Job: 1002, X: 20, Y: 20}
	return client.Context{World: w, Session: &session.Session{
		AccountID: 2000000, CharID: 100,
		Selected: session.Character{ID: 100, Job: db.JobArcher, Weapon: db.WeaponBow},
	}}
}
