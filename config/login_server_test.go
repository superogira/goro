package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadConfigServerSlots(t *testing.T) {
	isolateUserConfig(t)
	cfg, err := LoadConfig(nil)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Login.ServerSlot != 0 || cfg.Login.CharServerSlot != 0 {
		t.Fatal("server selection must default to the first entry")
	}
	path := filepath.Join(t.TempDir(), "goro.ini")
	if err := os.WriteFile(path, []byte("[login]\nserver_slot = 2\nchar_server_slot = 1\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg, err = LoadConfig([]string{"--config", path})
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Login.ServerSlot != 2 || cfg.Login.CharServerSlot != 1 {
		t.Fatalf("INI server slots = %d, %d; want 2, 1", cfg.Login.ServerSlot, cfg.Login.CharServerSlot)
	}
	cfg, err = LoadConfig([]string{"--config", path, "--server-slot=0", "--char-server-slot=3"})
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Login.ServerSlot != 0 || cfg.Login.CharServerSlot != 3 {
		t.Fatalf("CLI server slots = %d, %d; want 0, 3", cfg.Login.ServerSlot, cfg.Login.CharServerSlot)
	}
}

func TestLoadConfigRejectsNegativeServerSlots(t *testing.T) {
	for _, name := range []string{"server-slot", "char-server-slot"} {
		t.Run(name, func(t *testing.T) {
			isolateUserConfig(t)
			if _, err := LoadConfig([]string{"--" + name + "=-1"}); err == nil || !strings.Contains(err.Error(), name) {
				t.Fatalf("negative CLI slot error = %v", err)
			}
			path := filepath.Join(t.TempDir(), "goro.ini")
			text := "[login]\n" + strings.ReplaceAll(name, "-", "_") + " = -1\n"
			if err := os.WriteFile(path, []byte(text), 0o600); err != nil {
				t.Fatal(err)
			}
			if _, err := LoadConfig([]string{"--config", path}); err == nil || !strings.Contains(err.Error(), name) {
				t.Fatalf("negative INI slot error = %v", err)
			}
		})
	}
}
