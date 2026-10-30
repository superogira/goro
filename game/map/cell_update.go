package maps

import (
	"strings"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/glog"
	"github.com/kivutar/goro/network"
)

func ApplyCellUpdate(ctx client.Context, update network.MapCellUpdate) bool {
	if ctx.World == nil || ctx.World.GAT == nil {
		return false
	}
	if update.MapName != "" && ctx.World.MapName != "" && !strings.EqualFold(update.MapName, ctx.World.MapName) {
		glog.Debugf("map cell update ignored map=%s current=%s cell=%d,%d type=%d", update.MapName, ctx.World.MapName, update.X, update.Y, update.RawType)
		return false
	}
	if !ctx.World.GAT.SetCellRawType(update.X, update.Y, update.RawType) {
		glog.Debugf("map cell update out of bounds map=%s cell=%d,%d type=%d", update.MapName, update.X, update.Y, update.RawType)
		return false
	}
	glog.Debugf("map cell update map=%s cell=%d,%d type=%d walkable=%t", update.MapName, update.X, update.Y, update.RawType, ctx.World.GAT.Walkable(update.X, update.Y))
	return true
}
