package render

import (
	"os"
	"path/filepath"
	"testing"
)

func writeBatteryTree(t *testing.T, root, supply string, capacity, status string, withCapacity bool) {
	t.Helper()
	dir := filepath.Join(root, supply)
	if err := os.MkdirAll(filepath.Join(dir), 0o755); err != nil {
		t.Fatal(err)
	}
	if withCapacity {
		if err := os.WriteFile(filepath.Join(dir, "capacity"), []byte(capacity), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if status != "" {
		if err := os.WriteFile(filepath.Join(dir, "status"), []byte(status), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

func TestDiscoverBatteryPathsPrefersBatteryName(t *testing.T) {
	root := t.TempDir()
	// A different supply appears first alphabetically; "battery" still wins.
	writeBatteryTree(t, root, "axp-charger", "1\n", "Charging\n", false)
	writeBatteryTree(t, root, "battery", "80\n", "Discharging\n", true)

	paths, ok := discoverBatteryPaths(root)
	if !ok || filepath.Base(filepath.Dir(paths.capacity)) != "battery" {
		t.Fatalf("discovery = %+v ok=%v, want the battery supply", paths, ok)
	}
}

func TestDiscoverBatteryPathsFallsBackToFirstWithCapacity(t *testing.T) {
	root := t.TempDir()
	writeBatteryTree(t, root, "bms", "42\n", "", true)

	paths, ok := discoverBatteryPaths(root)
	if !ok || paths.capacity == "" {
		t.Fatal("expected the fallback supply to be discovered")
	}
}

func TestDiscoverBatteryPathsNoSysfs(t *testing.T) {
	if _, ok := discoverBatteryPaths(filepath.Join(t.TempDir(), "missing")); ok {
		t.Fatal("a missing sysfs root must not discover anything")
	}
}

func TestReadBattery(t *testing.T) {
	root := t.TempDir()
	writeBatteryTree(t, root, "battery", "85\n", "Charging\n", true)
	paths, _ := discoverBatteryPaths(root)

	state := readBattery(paths)
	if !state.ok || state.percent != 85 || !state.charging {
		t.Fatalf("state = %+v, want 85%% charging", state)
	}

	writeBatteryTree(t, root, "battery", "garbage\n", "", true)
	if state := readBattery(paths); state.ok {
		t.Fatal("malformed capacity must hide the badge, not show garbage")
	}
}
