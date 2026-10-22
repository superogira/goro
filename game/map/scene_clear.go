package maps

import (
	"image/color"
	"strings"

	"github.com/kivutar/goro/render"
)

func ClearWorldScene(screen *render.Frame, mapName string) {
	screen.Fill(worldSceneClearColor(mapName))
}

func worldSceneClearColor(mapName string) color.RGBA {
	if skyColor, ok := robrSkyClearColors[normalizeMapNameForSceneClear(mapName)]; ok {
		return skyColor
	}
	return color.RGBA{A: 255}
}

var robrSkyClearColors = map[string]color.RGBA{
	"airplane.rsw":    {R: 0x66, G: 0x99, B: 0xcc, A: 255},
	"airplane_01.rsw": {R: 0x66, G: 0x99, B: 0xcc, A: 255},
	"gonryun.rsw":     {R: 0x66, G: 0x99, B: 0xcc, A: 255},
	"gon_dun02.rsw":   {R: 0x66, G: 0x99, B: 0xcc, A: 255},
	"himinn.rsw":      {R: 0x66, G: 0x99, B: 0xcc, A: 255},
	"ra_temsky.rsw":   {R: 0x66, G: 0x99, B: 0xcc, A: 255},
	"rwc01.rsw":       {R: 0x66, G: 0x99, B: 0xcc, A: 255},
	"sch_gld.rsw":     {R: 0x66, G: 0x99, B: 0xcc, A: 255},
	"valkyrie.rsw":    {R: 0x66, G: 0x99, B: 0xcc, A: 255},
	"yuno.rsw":        {R: 0x66, G: 0x99, B: 0xcc, A: 255},
	"5@tower.rsw":     {R: 0x33, G: 0x00, B: 0x33, A: 255},
	"thana_boss.rsw":  {R: 0xe0, G: 0xd4, B: 0xc2, A: 255},
}

func normalizeMapNameForSceneClear(name string) string {
	name = strings.ToLower(strings.TrimSpace(name))
	if index := strings.LastIndexAny(name, `\/`); index >= 0 {
		name = name[index+1:]
	}
	switch {
	case strings.HasSuffix(name, ".gat"):
		return strings.TrimSuffix(name, ".gat") + ".rsw"
	case name != "" && !strings.Contains(name, "."):
		return name + ".rsw"
	default:
		return name
	}
}
