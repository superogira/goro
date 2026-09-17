package game

import "github.com/kivutar/goro/client"

func playerIsDead(ctx client.Context) bool {
	return ctx.Session != nil && ctx.Session.Dead
}

// hudHidden reports the small-screen layout mode: the permanent HUD
// (basic info + menu, minimap, shortcut bar, chat console) stays
// unpublished while a handheld-specific layout is being designed.
func hudHidden(ctx client.Context) bool {
	return ctx.Config.UI.HideHUD
}
