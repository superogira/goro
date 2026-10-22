package weather

import (
	"image/color"

	"github.com/kivutar/goro/config"
	"github.com/kivutar/goro/game/effects"
	"github.com/kivutar/goro/game/scene"
	"github.com/kivutar/goro/res"
)

// FogFromMap combines the map fog table with weather-specific colors.
func FogFromMap(manager *res.Manager, mapName string, cfg config.FogConfig) scene.SceneFog {
	if manager == nil || !cfg.Enabled {
		return scene.SceneFog{}
	}
	parameter, ok := manager.FogParameter(mapName)
	if !ok || parameter.Far <= parameter.Near {
		return scene.SceneFog{}
	}
	// The table's factor column is parsed for compatibility, but the reference
	// clients apply the near/far fog ramp at full strength.
	fogColor := parameter.Color
	if weatherColor, ok := sceneFogColorForMapWeather(mapName); ok {
		fogColor = weatherColor
	}
	return scene.SceneFog{
		Enabled: true,
		Near:    parameter.Near * 240,
		Far:     parameter.Far * 240,
		Color:   fogColor,
	}
}

func sceneFogColorForMapWeather(mapName string) (color.RGBA, bool) {
	effectID := mapWeatherEffectIDForMap(mapName)
	switch effectID {
	case effects.Cloud4:
		params, ok := weatherCloudParamsForEffect(effectID)
		return params.tint, ok
	default:
		return color.RGBA{}, false
	}
}
