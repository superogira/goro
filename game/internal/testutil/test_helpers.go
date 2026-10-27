package testutil

import (
	"github.com/kivutar/goro/db"
	"github.com/kivutar/goro/game/scene"
)

func EffectTableSize(value float64) float64 {
	return value * scene.PixelRatio
}

type effectCoverage struct {
	Implemented     int
	ReferenceActive int
	ReferenceAll    int
	ActivePercent   float64
	AllPercent      float64
}

func EffectCoverageSnapshot() effectCoverage {
	implemented := len(db.EffectSpecs)
	return effectCoverage{
		Implemented:     implemented,
		ReferenceActive: db.ReferenceActiveEffectTableEntries,
		ReferenceAll:    db.ReferenceNumericEffectConstants,
		ActivePercent:   float64(implemented) / float64(db.ReferenceActiveEffectTableEntries) * 100,
		AllPercent:      float64(implemented) / float64(db.ReferenceNumericEffectConstants) * 100,
	}
}
