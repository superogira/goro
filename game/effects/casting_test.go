package effects

import (
	"testing"
)

func TestCastRingEffectSpecUsesMagicRingCylinder(t *testing.T) {
	spec, ok := Lookup(CastRing)
	if !ok {
		t.Fatal("cast ring effect missing")
	}
	if len(spec.Components) != 1 {
		t.Fatalf("components = %d, want 1", len(spec.Components))
	}
	component := spec.Components[0]
	if component.Kind != ComponentFUNC || component.FuncAdapter != EffectFuncCastRing || component.FuncName != "CastRing" || component.TextureName != "ring_yellow" {
		t.Fatalf("component = %+v", component)
	}
	if component.BottomSize != 0.8 || component.TopSize != 2.45 || component.Height != 2.8 {
		t.Fatalf("magic ring dimensions = bottom %.2f top %.2f height %.2f", component.BottomSize, component.TopSize, component.Height)
	}
	if component.Duration != 0 {
		t.Fatalf("cast ring component duration = %s, want inherited cast duration", component.Duration)
	}
}

func TestBeginSpellEffectsInheritCastDuration(t *testing.T) {
	for _, effectID := range []int{BeginSpell, BeginSpell2, BeginSpell3, BeginSpell4, BeginSpell5, BeginSpell6, BeginSpell7} {
		spec, ok := Lookup(effectID)
		if !ok {
			t.Fatalf("begin spell effect %d missing", effectID)
		}
		for i, component := range spec.Components {
			if component.Duration != 0 {
				t.Fatalf("effect %d component %d duration = %s, want inherited cast duration", effectID, i, component.Duration)
			}
		}
	}
}
