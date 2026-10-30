package scene

import (
	"github.com/kivutar/goro/res"
	"github.com/kivutar/goro/world"
)

func ActorIsPlayerObject(actor world.Actor) bool {
	if actor.HasObjectType {
		return actor.ObjectType == ActorObjectTypePC
	}
	return res.HasPlayerJobToken(int(actor.Job))
}
