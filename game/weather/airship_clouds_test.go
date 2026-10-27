package weather

import (
	"math"
	"reflect"
	"testing"
	"time"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/game/effects"
	"github.com/kivutar/goro/game/internal/testutil"
	"github.com/kivutar/goro/game/scene"
	"github.com/kivutar/goro/render"
	"github.com/kivutar/goro/res"
	worldstate "github.com/kivutar/goro/world"
)

func TestAirshipCloudsDriftForwardForBothPhaseSigns(t *testing.T) {
	now := time.Unix(100, 0)
	for _, effectID := range []int{effects.Cloud5, effects.Cloud2, effects.Cloud4} {
		params, ok := weatherCloudParamsForEffect(effectID)
		if !ok {
			t.Fatalf("missing cloud profile %d", effectID)
		}
		for _, sign := range []float64{-1, 1} {
			state := mapWeatherCloudState{lastUpdate: now, clouds: []mapWeatherCloud{{
				x: 10, y: 20, phaseX: sign * math.Pi / 2, phaseY: -math.Pi / 2, rotStart: time.Minute,
			}}}
			state.update(params, nil, 0, 0, now.Add(100*time.Millisecond))
			wantX, wantY := sign*0.06, -0.06
			switch effectID {
			case effects.Cloud5:
				// Original Cloud(4): +0.20*abs(sin(x)) and 0.05*sin(y) per frame.
				wantX = 0.24
			case effects.Cloud4:
				wantX, wantY = sign*0.018, -0.018
			}
			if dx, dy := state.clouds[0].x-10, state.clouds[0].y-20; math.Abs(dx-wantX) > 1e-9 || math.Abs(dy-wantY) > 1e-9 {
				t.Fatalf("effect %d phase sign %.0f: drift %f,%f, want %f,%f", effectID, sign, dx, dy, wantX, wantY)
			}
		}
	}
}

func TestAirshipCloudsUseClassicDensityAndPlacement(t *testing.T) {
	params, ok := weatherCloudParamsForEffect(effects.Cloud5)
	if !ok {
		t.Fatal("airship cloud profile missing")
	}
	state := mapWeatherCloudState{}
	state.ensure("airplane_01.rsw", params, nil, 239.5, 62.5, time.Unix(100, 0))
	if len(state.clouds) != 320 {
		t.Fatalf("cloud count = %d, want 320 (80 groups of four)", len(state.clouds))
	}
	for _, cloud := range state.clouds {
		dx, dy := math.Abs(cloud.x-239.5), math.Abs(cloud.y-62.5)
		if dx < 5 || dx > 45 || dy < 5 || dy > 45 || cloud.z < -10 || cloud.z > -8 {
			t.Fatalf("cloud outside classic sky placement: %+v", cloud)
		}
	}
	if params.tint.R != 255 || params.tint.G != 255 || params.tint.B != 255 || params.overlay || params.useGround || params.screenHaze.A != 0 {
		t.Fatalf("airship should use white sky clouds with depth testing: %+v", params)
	}
}

func TestAirshipCloudsRenderAndResetRealData(t *testing.T) {
	manager := testutil.RealDataManager(t)
	mode := &System{}
	fx := &effects.System{}
	for _, mapName := range []string{"airplane", "airplane_01"} {
		t.Run(mapName, func(t *testing.T) {
			gndData, err := manager.ReadFile("data/" + mapName + ".gnd")
			if err != nil {
				t.Fatal(err)
			}
			gnd, err := res.ParseGND(gndData)
			if err != nil {
				t.Fatal(err)
			}
			gatData, err := manager.ReadFile("data/" + mapName + ".gat")
			if err != nil {
				t.Fatal(err)
			}
			gat, err := res.ParseGAT(gatData)
			if err != nil {
				t.Fatal(err)
			}
			world := worldstate.New()
			world.MapName, world.GND, world.GAT = mapName, gnd, gat
			world.Player = worldstate.Actor{X: 239, Y: 62}
			ctx := client.Context{World: world, Resources: manager}
			projection := testutil.Projection(800, 600, 239.5, 62.5, scene.TerrainHeightAt(world, 239, 62))
			frame := render.NewFrame(800, 600)
			now := time.Unix(100, 0)
			mode.Draw(frame, ctx, projection, now, fx, nil)
			mode.Draw(frame, ctx, projection, now.Add(100*time.Millisecond), fx, nil)
			if mode.cloud.effectID != effects.Cloud5 || mode.cloud.key != mapName+".rsw" {
				t.Fatalf("wrong map cloud state: key=%s effect=%d", mode.cloud.key, mode.cloud.effectID)
			}
			commands := reflect.ValueOf(frame).Elem().FieldByName("worldBillboards")
			if commands.Len() == 0 {
				t.Fatal("airship cloud textures produced no draw commands")
			}
			t.Logf("%s rendered %d cloud billboards", mapName, commands.Len())
			world.MapName = "prontera"
			frame.BeginFrame()
			mode.Draw(frame, ctx, projection, now.Add(200*time.Millisecond), fx, nil)
			if len(mode.cloud.clouds) != 0 || commands.Len() != 0 {
				t.Fatal("airship clouds survived a change to a map without clouds")
			}
		})
	}
}
