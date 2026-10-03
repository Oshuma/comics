package config

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

const fileName = "config.yaml"

type Config struct {
	// Address the HTTP server listens on.
	Listen string `yaml:"listen"`
	// PostgreSQL connection URL.
	DatabaseURL string `yaml:"database_url"`
	// Directory where extracted page images are stored.
	StorageDir string `yaml:"storage_dir"`
	// Maximum upload size per comic archive, in megabytes.
	MaxUploadMB int64 `yaml:"max_upload_mb"`
	// Set the Secure flag on session cookies (enable when served over HTTPS).
	SecureCookies bool `yaml:"secure_cookies"`

	// Path of the config file that was loaded, if any.
	Path string `yaml:"-"`
}

func defaults() Config {
	return Config{
		Listen:      ":3000",
		DatabaseURL: "postgres://postgres:postgres@localhost:5432/comics_development?sslmode=disable",
		StorageDir:  "storage",
		MaxUploadMB: 2048,
	}
}

// SearchPaths returns the locations checked for config.yaml, in order.
func SearchPaths() []string {
	var paths []string
	if wd, err := os.Getwd(); err == nil {
		paths = append(paths, filepath.Join(wd, fileName))
	}
	if dir, err := os.UserConfigDir(); err == nil {
		paths = append(paths, filepath.Join(dir, "comics", fileName))
	}
	return paths
}

// Load reads the config from explicitPath if given, otherwise from the first
// existing file in SearchPaths. Defaults are used when no file is found.
func Load(explicitPath string) (*Config, error) {
	cfg := defaults()

	path := explicitPath
	if path == "" {
		for _, p := range SearchPaths() {
			if _, err := os.Stat(p); err == nil {
				path = p
				break
			} else if !errors.Is(err, fs.ErrNotExist) {
				return nil, fmt.Errorf("checking %s: %w", p, err)
			}
		}
	}

	if path != "" {
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("reading config: %w", err)
		}
		if err := yaml.Unmarshal(data, &cfg); err != nil {
			return nil, fmt.Errorf("parsing %s: %w", path, err)
		}
		cfg.Path, _ = filepath.Abs(path)
	}

	if v := os.Getenv("COMICS_DATABASE_URL"); v != "" {
		cfg.DatabaseURL = v
	}

	// Relative storage paths are resolved against the config file's directory.
	if !filepath.IsAbs(cfg.StorageDir) && cfg.Path != "" {
		cfg.StorageDir = filepath.Join(filepath.Dir(cfg.Path), cfg.StorageDir)
	}
	abs, err := filepath.Abs(cfg.StorageDir)
	if err != nil {
		return nil, err
	}
	cfg.StorageDir = abs

	if cfg.MaxUploadMB <= 0 {
		cfg.MaxUploadMB = defaults().MaxUploadMB
	}

	return &cfg, nil
}
