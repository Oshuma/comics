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
	// Path to the SQLite database file.
	DatabasePath string `yaml:"database_path"`
	// Directory where extracted page images are stored.
	StorageDir string `yaml:"storage_dir"`
	// Maximum upload size per comic archive, in megabytes.
	MaxUploadMB int64 `yaml:"max_upload_mb"`
	// Number of uploads/imports processed at the same time in the background.
	ImportWorkers int `yaml:"import_workers"`
	// Set the Secure flag on session cookies (enable when served over HTTPS).
	SecureCookies bool `yaml:"secure_cookies"`

	// Path of the config file that was loaded, if any.
	Path string `yaml:"-"`
}

func defaults() Config {
	return Config{
		Listen:        ":3000",
		DatabasePath:  "comics.db",
		StorageDir:    "storage",
		MaxUploadMB:   2048,
		ImportWorkers: 2,
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

	if v := os.Getenv("COMICS_DATABASE_PATH"); v != "" {
		cfg.DatabasePath = v
	}

	var err error
	if cfg.DatabasePath, err = cfg.resolve(cfg.DatabasePath); err != nil {
		return nil, err
	}
	if cfg.StorageDir, err = cfg.resolve(cfg.StorageDir); err != nil {
		return nil, err
	}

	if cfg.MaxUploadMB <= 0 {
		cfg.MaxUploadMB = defaults().MaxUploadMB
	}
	if cfg.ImportWorkers <= 0 {
		cfg.ImportWorkers = defaults().ImportWorkers
	}

	return &cfg, nil
}

// resolve makes a relative path absolute against the config file's directory
// (or the working directory when no config file was loaded).
func (c *Config) resolve(path string) (string, error) {
	if !filepath.IsAbs(path) && c.Path != "" {
		path = filepath.Join(filepath.Dir(c.Path), path)
	}
	return filepath.Abs(path)
}
