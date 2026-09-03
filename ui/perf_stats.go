package ui

import "fmt"

type perfMem struct {
	goHeapMB float64
	goSysMB  float64
	goGC     uint32
	extra    string
}

func formatMB(v float64) string {
	return fmt.Sprintf("%.1f", v)
}
