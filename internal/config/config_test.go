package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadPrefersWorkingDir(t *testing.T) {
	wd := t.TempDir()
	home := t.TempDir()
	t.Chdir(wd)
	t.Setenv("XDG_CONFIG_HOME", home)
	t.Setenv("HOME", home)
	t.Setenv("COMICS_DATABASE_URL", "")

	os.MkdirAll(filepath.Join(home, "comics"), 0o755)
	os.WriteFile(filepath.Join(home, "comics", "config.yaml"), []byte("listen: ':2'\n"), 0o644)

	cfg, err := Load("")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Listen != ":2" {
		t.Fatalf("expected user config dir to be used, got %q", cfg.Listen)
	}
	if cfg.StorageDir != filepath.Join(home, "comics", "storage") {
		t.Fatalf("storage dir not resolved relative to config: %s", cfg.StorageDir)
	}

	os.WriteFile(filepath.Join(wd, "config.yaml"), []byte("listen: ':1'\ndatabase_url: postgres://x\n"), 0o644)
	cfg, err = Load("")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Listen != ":1" || cfg.DatabaseURL != "postgres://x" {
		t.Fatalf("expected local config to win, got %+v", cfg)
	}
}

func TestLoadDefaults(t *testing.T) {
	t.Chdir(t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("HOME", t.TempDir())
	t.Setenv("COMICS_DATABASE_URL", "postgres://env")

	cfg, err := Load("")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Path != "" || cfg.Listen != ":3000" || cfg.DatabaseURL != "postgres://env" {
		t.Fatalf("unexpected config: %+v", cfg)
	}
}
