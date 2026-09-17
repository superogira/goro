package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestConfigPrecedence(t *testing.T) {
	isolateUserConfig(t)
	localPath, err := filepath.Abs("goro.ini")
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	dataPath := filepath.Join(dir, "goro.ini")
	explicitPath := filepath.Join(t.TempDir(), "custom.ini")
	for path, contents := range map[string]string{
		localPath:    "[window]\nwidth = 800\nheight = 700\n[packet]\nprofile = 29\n",
		dataPath:     "[window]\nwidth = 900\nheight = 600\n[render]\nvsync = false\n",
		explicitPath: "[window]\nwidth = 1000\n[render]\nvsync = true\n",
	} {
		if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	for _, tc := range []struct {
		name   string
		args   []string
		width  int
		height int
		vsync  bool
		path   string
	}{
		{"working directory", nil, 800, 700, true, localPath},
		{"flags override working directory", []string{"--width=1100", "--vsync=false"}, 1100, 700, false, localPath},
		{"data directory", []string{"--data-dir", dir}, 900, 600, false, dataPath},
		{"explicit config", []string{"--config", explicitPath}, 1000, 700, true, explicitPath},
		{"both files", []string{"--config", explicitPath, "--data-dir", dir}, 1000, 600, true, explicitPath},
		{"flags override both", []string{"--vsync=false", "--width=1100", "--data-dir", dir, "--config", explicitPath}, 1100, 600, false, explicitPath},
		{"equals syntax", []string{"--data-dir=" + dir, "--config=" + explicitPath}, 1000, 600, true, explicitPath},
		{"last config wins", []string{"--config", "missing.ini", "--config", explicitPath}, 1000, 700, true, explicitPath},
		{"last data dir wins", []string{"--data-dir", "missing", "--data-dir", dir}, 900, 600, false, dataPath},
		{"single dash", []string{"-data-dir", dir, "-config", explicitPath, "-vsync=false"}, 1000, 600, false, explicitPath},
		{"flags after terminator ignored", []string{"--", "--config", explicitPath, "--data-dir", dir}, 800, 700, true, localPath},
		{"flag-like value", []string{"--title", "--config=missing.ini"}, 800, 700, true, localPath},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cfg, err := LoadConfig(tc.args)
			if err != nil {
				t.Fatal(err)
			}
			if cfg.Window.Width != tc.width || cfg.Window.Height != tc.height || cfg.Render.VSync != tc.vsync || cfg.ConfigPath != tc.path {
				t.Fatalf("width=%d height=%d vsync=%v path=%q; want %d %d %v %q",
					cfg.Window.Width, cfg.Window.Height, cfg.Render.VSync, cfg.ConfigPath, tc.width, tc.height, tc.vsync, tc.path)
			}
			if cfg.Packet.Profile != 29 {
				t.Fatal("lost a working-directory setting absent from the higher-priority files")
			}
		})
	}
}

func TestConfigIgnoresUnselectedFiles(t *testing.T) {
	isolateUserConfig(t)
	userDir, err := os.UserConfigDir()
	if err != nil {
		t.Fatal(err)
	}
	legacyPath := filepath.Join(userDir, "goro", "goro.ini")
	if err := os.MkdirAll(filepath.Dir(legacyPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(legacyPath, []byte("this file must not be read\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	explicitPath := filepath.Join(t.TempDir(), "custom.ini")
	if err := os.WriteFile(explicitPath, []byte("data_dir = "+dir+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "goro.ini"), []byte("not selected by an INI data_dir value\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{nil, {"--vsync=false"}, {"--config", explicitPath}, {"--data-dir", t.TempDir()}} {
		if _, err := LoadConfig(args); err != nil {
			t.Fatalf("%q: %v", args, err)
		}
	}
}

func TestConfigRequiresExplicitFileButAllowsMissingDataDirFile(t *testing.T) {
	isolateUserConfig(t)
	dir := t.TempDir()
	if _, err := LoadConfig([]string{"--config", filepath.Join(dir, "missing.ini")}); err == nil {
		t.Fatal("accepted a missing explicit config")
	}
	for _, args := range [][]string{{"--config"}, {"--data-dir"}, {"--config="}, {"--data-dir="}, {"--data-dir", dir, "--config="}} {
		if _, err := LoadConfig(args); err == nil {
			t.Fatalf("accepted missing or empty path: %q", args)
		}
	}
	cfg, err := LoadConfig([]string{"--data-dir", dir})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(cfg.ConfigPath); !os.IsNotExist(err) {
		t.Fatalf("loading created a config file: %v", err)
	}
	path, err := cfg.SaveUserSettings(UserSettings{VSync: false})
	if err != nil || path != filepath.Join(dir, "goro.ini") {
		t.Fatalf("save path=%q err=%v", path, err)
	}
	restarted, err := LoadConfig([]string{"--data-dir", dir})
	if err != nil || restarted.Render.VSync {
		t.Fatalf("saved VSync not restored: %v", err)
	}
	if _, err := os.Stat("goro.ini"); !os.IsNotExist(err) {
		t.Fatalf("saving to --data-dir unexpectedly created ./goro.ini: %v", err)
	}
}

func TestConfigDefaultsToWorkingDirectoryForPersistence(t *testing.T) {
	isolateUserConfig(t)
	cfg, err := LoadConfig(nil)
	if err != nil {
		t.Fatal(err)
	}
	wantPath, err := filepath.Abs("goro.ini")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.ConfigPath != wantPath || cfg.Window.Width != 1280 || !cfg.Render.VSync {
		t.Fatalf("unexpected default config: %+v", cfg)
	}
	if _, err := os.Stat(wantPath); !os.IsNotExist(err) {
		t.Fatalf("loading created a config file: %v", err)
	}
	commands := cfg.ChatShortcuts
	commands[0] = "/sit"
	for name, save := range map[string]func() (string, error){
		"settings":  func() (string, error) { return cfg.SaveUserSettings(UserSettings{}) },
		"login":     func() (string, error) { return cfg.SaveLoginID("Tester", true) },
		"shortcuts": func() (string, error) { return cfg.SaveChatShortcuts(commands) },
	} {
		if path, err := save(); path != wantPath || err != nil {
			t.Fatalf("%s: path=%q err=%v", name, path, err)
		}
	}
	if got, err := cfg.LoadChatShortcuts(); err != nil || got != commands {
		t.Fatalf("reloaded shortcuts = %q, %v", got, err)
	}
	restarted, err := LoadConfig(nil)
	if err != nil {
		t.Fatal(err)
	}
	if restarted.Render.VSync || !restarted.Login.KeepID || restarted.Login.SavedUsername != "Tester" || restarted.ChatShortcuts != commands {
		t.Fatalf("preferences lost after restart: %+v", restarted)
	}
	userDir, err := os.UserConfigDir()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(userDir, "goro", "goro.ini")); !os.IsNotExist(err) {
		t.Fatalf("unexpected per-user config: %v", err)
	}
}

func TestConfigRelativeSavePathSurvivesWorkingDirectoryChange(t *testing.T) {
	isolateUserConfig(t)
	cfg, err := LoadConfig([]string{"--data-dir", "."})
	if err != nil {
		t.Fatal(err)
	}
	original := cfg.ConfigPath
	t.Chdir(t.TempDir())
	if path, err := cfg.SaveLoginID("Tester", true); err != nil || path != original {
		t.Fatalf("save path=%q err=%v; want %q", path, err, original)
	}
	if _, err := os.Stat("goro.ini"); !os.IsNotExist(err) {
		t.Fatalf("saved in the new working directory: %v", err)
	}
}

func TestHeadlessConfigCredentialsLoadedBeforeValidation(t *testing.T) {
	path := filepath.Join(t.TempDir(), "goro.ini")
	if err := os.WriteFile(path, []byte("[login]\nusername = Tester\npassword = test\nchar_slot = 0\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadConfig([]string{"--headless", "--config", path}); err != nil {
		t.Fatal(err)
	}
}
