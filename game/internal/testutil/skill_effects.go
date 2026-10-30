package testutil

import (
	"reflect"
	"testing"
)

func ExpectEffectIDs(t *testing.T, label string, got []int, want ...int) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("%s effects = %v, want %v", label, got, want)
	}
}
