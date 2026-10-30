package skills

import (
	"testing"
	"time"

	"github.com/kivutar/goro/db"
	"github.com/kivutar/goro/game/internal/testutil"
	maps "github.com/kivutar/goro/game/map"
	"github.com/kivutar/goro/network"
	"github.com/kivutar/goro/res"
	"github.com/kivutar/goro/session"
	worldstate "github.com/kivutar/goro/world"
)

func TestTrapTargetCell(t *testing.T) {
	ctx := testutil.ArcherEffectContext()
	m := &System{}
	m.ApplySkillUnitModelEntry(network.SkillUnitEntry{ID: 500, UnitID: 153, X: 15, Y: 20, Visible: true})
	skill := session.Skill{ID: db.SkillHTRemovetrap, Type: SkillTargetTrap, Level: 1, Range: 2}
	pending := PendingTarget{Skill: skill, TargetID: 500}
	if x, y, ok := m.targetCell(ctx, pending, time.Now()); !ok || x != 15 || y != 20 {
		t.Fatal("trap chase lost target")
	}
	m.RemoveSkillUnitModelOnly(500)
	if _, _, ok := m.targetCell(ctx, pending, time.Now()); ok {
		t.Fatal("disappeared trap remains a cast target")
	}
}

func TestLegacySkillUnitTrapRSMModelsLoadFromRealDataWhenConfigured(t *testing.T) {
	manager := testutil.RealDataManager(t)
	checked := 0
	for unitID, spec := range db.SkillUnitModels {
		if len(spec.FallbackModelPaths) == 0 {
			continue
		}
		checked++
		var rsm *res.RSM
		var loadedPath string
		paths := append([]string{spec.ModelPath}, spec.FallbackModelPaths...)
		for _, modelPath := range paths {
			loaded, err := maps.LoadRSMModel(manager, modelPath)
			if err != nil {
				continue
			}
			rsm = loaded
			loadedPath = modelPath
			break
		}
		if rsm == nil {
			t.Fatalf("load trap model unit=%d paths=%v", unitID, paths)
		}
		if len(rsm.Nodes) == 0 {
			t.Fatalf("trap model unit=%d model=%s has no nodes", unitID, loadedPath)
		}
	}
	if checked == 0 {
		t.Fatal("no legacy trap RSM models were checked")
	}
}

func TestSkillUnitRSMPlacementUsesCellCenterAndTerrainHeight(t *testing.T) {
	world := &worldstate.World{
		GAT: &res.GAT{
			Width:  2,
			Height: 2,
			Cells: []res.GATCell{
				{Heights: [4]float32{1, 1, 1, 1}},
				{Heights: [4]float32{2, 2, 2, 2}},
				{Heights: [4]float32{3, 3, 3, 3}},
				{Heights: [4]float32{7, 7, 7, 7}},
			},
		},
	}
	unit := skillUnitModel{UnitID: 145, X: 1, Y: 1, ModelPath: db.SkillUnitModels[145].ModelPath}

	placement := skillUnitRSMPlacement(world, 9205, unit)

	if placement.BaseX != 1.5 || placement.BaseY != 1.5 {
		t.Fatalf("base = %.1f,%.1f, want cell center 1.5,1.5", placement.BaseX, placement.BaseY)
	}
	if placement.Model.Position.Y != 7 {
		t.Fatalf("model y = %.1f, want terrain height 7", placement.Model.Position.Y)
	}
	if placement.Model.Scale != (res.RSWVector3{X: 1, Y: 1, Z: 1}) {
		t.Fatalf("scale = %+v, want 1,1,1", placement.Model.Scale)
	}
}

func TestTrapSkillUnitRSMPlacementUsesEffectScale(t *testing.T) {
	world := worldstate.New()
	unit := skillUnitModel{UnitID: 145, X: 1, Y: 1, ModelPath: db.SkillUnitModels[145].ModelPath, Scale: db.SkillUnitModels[145].Scale}

	placement := skillUnitRSMPlacement(world, 9206, unit)

	want := res.RSWVector3{X: 0.15, Y: 0.15, Z: 0.15}
	if placement.Model.Scale != want {
		t.Fatalf("scale = %+v, want skill-unit trap scale %+v", placement.Model.Scale, want)
	}
}

func TestTrapSkillUnitRSMFrameUsesClassicClientFixedFrame(t *testing.T) {
	rsm := &res.RSM{
		AnimLength: 12,
		Nodes: []res.RSMNode{{
			Name: "root",
			PositionKeyframes: []res.RSMPositionKeyframe{
				{Frame: 0},
				{Frame: 12, Pos: res.RSMVector3{X: 12}},
			},
		}},
	}
	unit := skillUnitModel{HasFixedFrame: true, FixedFrame: 3}

	frame, animated := SkillUnitRSMFrame(rsm, res.RSWModel{}, unit, time.UnixMilli(1000))
	if animated || frame != 3 {
		t.Fatalf("trap RSM frame = %d animated=%t, want fixed frame 3", frame, animated)
	}
}
