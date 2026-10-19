package res

import (
	"maps"
	"testing"
	"testing/fstest"

	"github.com/kivutar/goro/db"
)

func TestParseSkillNameTable(t *testing.T) {
	nameToID := map[string]int{"sm_bash": 5}
	got := parseSkillNameTable([]byte("// comment\nSM_BASH#Bash#\nUNKNOWN#Nope#\n"), nameToID)
	if got[5] != "Bash" {
		t.Fatalf("skill name = %q", got[5])
	}
	if _, ok := got[0]; ok {
		t.Fatal("zero skill id should not be populated")
	}
}

func TestParseSkillDescriptionTable(t *testing.T) {
	nameToID := map[string]int{"sm_bash": 5}
	names, descriptions := parseSkillDescriptionTable([]byte("SM_BASH#\nBash\nStrike an enemy.\nConsumes SP.\n#\n"), nameToID)
	if names[5] != "Bash" {
		t.Fatalf("skill title = %q", names[5])
	}
	lines := descriptions[5]
	if len(lines) != 2 || lines[0] != "Strike an enemy." || lines[1] != "Consumes SP." {
		t.Fatalf("description = %#v", lines)
	}
}

func TestSkillMetadataLookupCopiesDescription(t *testing.T) {
	manager := &Manager{
		skillMetadataLoaded: true,
		skillDisplayNames: map[int]string{
			5: "Bash",
		},
		skillDescriptions: map[int][]string{
			5: []string{"Strike an enemy."},
		},
	}
	if got, ok := manager.SkillDisplayName(5); !ok || got != "Bash" {
		t.Fatalf("display name = %q ok=%v", got, ok)
	}
	lines, ok := manager.SkillDescription(5)
	if !ok || len(lines) != 1 || lines[0] != "Strike an enemy." {
		t.Fatalf("description = %#v ok=%v", lines, ok)
	}
	lines[0] = "mutated"
	if lines, _ := manager.SkillDescription(5); lines[0] != "Strike an enemy." {
		t.Fatalf("description was not copied: %#v", lines)
	}
}

func TestParseSkillSPAmountMaxLevels(t *testing.T) {
	nameToID := map[string]int{
		"sm_bash":    5,
		"pr_angelus": 33,
	}
	got := parseSkillSPAmountMaxLevels([]byte("SM_BASH#\n8#\n8#\n@\nUNKNOWN#\n1#\n@\nPR_ANGELUS#\n23#\n26#\n29#\n@\n"), nameToID)
	if got[5] != 2 {
		t.Fatalf("SM_BASH max level = %d, want 2", got[5])
	}
	if got[33] != 3 {
		t.Fatalf("PR_ANGELUS max level = %d, want 3", got[33])
	}
	if _, ok := got[0]; ok {
		t.Fatal("zero skill id should not be populated")
	}
}

func TestParseSkillTreePositionsInvertsClientTable(t *testing.T) {
	table := luaValue{kind: luaTable, table: map[interface{}]luaValue{
		18: {kind: luaTable, table: map[interface{}]luaValue{
			0:  {kind: luaNumber, num: 225},
			14: {kind: luaNumber, num: 228},
		}},
		"invalid": {kind: luaTable},
	}}

	positions := parseSkillTreePositions(table)
	if got := positions[18][225]; got != 0 {
		t.Fatalf("skill 225 position = %d, want 0", got)
	}
	if got := positions[18][228]; got != 14 {
		t.Fatalf("skill 228 position = %d, want 14", got)
	}
	if _, ok := positions[0]; ok {
		t.Fatal("invalid job key produced a position table")
	}
}

func TestSkillTreePositionsFromTextWithoutLua(t *testing.T) {
	for _, jobKey := range []string{"MONK", "15"} {
		t.Run(jobKey, func(t *testing.T) {
			manager, err := NewManagerFS(fstest.MapFS{
				"data/SkillTreeView.txt": {Data: []byte(jobKey + `
{
MO_IRONHAND#0#10@
MO_EXPLOSIONSPIRITS#14#7#5@
MO_EXTREMITYFIST#22#14#15#5@
MO_BODYRELOCATION#30#16#22#24#1@
}`)},
			})
			if err != nil {
				t.Fatal(err)
			}
			want := map[int]int{
				int(db.SkillMOIronhand): 0, int(db.SkillMOExplosionspirits): 14,
				int(db.SkillMOExtremityfist): 22, int(db.SkillMOBodyrelocation): 30,
			}
			got, ok := manager.SkillTreePositions(db.JobMonk)
			if !ok || !maps.Equal(got, want) {
				t.Fatalf("Monk positions = %v, %v; want %v", got, ok, want)
			}
		})
	}
}

func TestSkillTreeTextPositionValidation(t *testing.T) {
	data := []byte("\ufeff// header\r\nmonk\r\n{\r\n" + `
MO_IRONHAND#0#10@ // a real position of zero
MO_EXPLOSIONSPIRITS#14#7#5@
UNKNOWN#4#1@
MO_CALLSPIRITS#bad#5@
MO_DODGE#-1#10@
MO_BODYRELOCATION#30#1
}`)
	got := parseSkillTreeTextPositions(data, map[string]int{
		"mo_ironhand": 259, "mo_explosionspirits": 270,
		"mo_callspirits": 261, "mo_dodge": 265, "mo_bodyrelocation": 264,
	})
	if len(got) != 1 || !maps.Equal(got["MONK"], map[int]int{259: 0, 270: 14}) {
		t.Fatalf("positions include malformed entries: %v", got)
	}
}

func TestSkillTreeLayoutSourceSelection(t *testing.T) {
	manager := &Manager{
		skillTreePositionsLoaded: true,
		skillTreePositions:       map[int]map[int]int{db.JobMonk: {259: 2}},
		skillTreeTextPositions: map[string]map[int]int{
			"MONK": {259: 0}, "4": {28: 2}, "ACOLYTE": {28: 3},
			"STAR": {421: 0}, "LINKER": {445: 0},
		},
	}
	for _, tc := range []struct{ job, skill, position int }{
		{db.JobMonk, 259, 2}, {db.JobAcolyte, 28, 2},
		{db.JobStar, 421, 0}, {db.JobLinker, 445, 0},
	} {
		positions, ok := manager.SkillTreePositions(tc.job)
		position, found := positions[tc.skill]
		if !ok || !found || position != tc.position {
			t.Fatalf("job %d positions = %v, %v; want skill %d at %d", tc.job, positions, ok, tc.skill, tc.position)
		}
	}
}
