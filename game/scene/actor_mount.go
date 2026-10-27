package scene

import (
	"github.com/kivutar/goro/db"
	worldstate "github.com/kivutar/goro/world"
)

func ActorVisualJob(actor worldstate.Actor) int {
	return VisualJobForEffectState(int(actor.Job), actor.EffectState)
}

func VisualJobForEffectState(job int, effectState uint32) int {
	if effectState&db.EffectStateWedding != 0 {
		return db.JobMarried
	}
	if effectState&db.EffectStateRiding != 0 {
		if mounted, ok := db.MountJob(job); ok {
			return mounted
		}
	}
	switch job {
	case db.JobStar2:
		job = db.JobStar
	case db.JobStar2B:
		job = db.JobStarB
	}
	if effectState&db.EffectStateFlying != 0 {
		switch job {
		case db.JobStar:
			return db.JobStar2
		case db.JobStarB:
			return db.JobStar2B
		}
	}
	return job
}
