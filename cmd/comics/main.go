package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/Oshuma/comics/internal/archiveorg"
	"github.com/Oshuma/comics/internal/config"
	"github.com/Oshuma/comics/internal/database"
	"github.com/Oshuma/comics/internal/media"
	"github.com/Oshuma/comics/internal/server"
	"github.com/Oshuma/comics/internal/store"
	"github.com/Oshuma/comics/web"
)

// Set at build time with -ldflags "-X main.version=... -X main.commit=...".
var (
	version = "0.0.0-dev"
	commit  = ""
)

// fullVersion returns the semver version with the commit as build metadata,
// e.g. 0.1.0+271c7ec.
func fullVersion() string {
	if commit == "" {
		return version
	}
	return version + "+" + commit
}

func main() {
	configPath := flag.String("config", "", "path to config.yaml (overrides the default search locations)")
	showVersion := flag.Bool("version", false, "print the version and exit")
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Usage: comics [flags]\n\nConfig search order:\n")
		for _, p := range config.SearchPaths() {
			fmt.Fprintf(flag.CommandLine.Output(), "  %s\n", p)
		}
		fmt.Fprintln(flag.CommandLine.Output(), "\nFlags:")
		flag.PrintDefaults()
	}
	flag.Parse()

	if *showVersion {
		fmt.Println(fullVersion())
		return
	}

	if err := run(*configPath); err != nil {
		slog.Error(err.Error())
		os.Exit(1)
	}
}

func run(configPath string) error {
	cfg, err := config.Load(configPath)
	if err != nil {
		return err
	}
	if cfg.Path != "" {
		slog.Info("loaded config", "path", cfg.Path)
	} else {
		slog.Info("no config.yaml found, using defaults")
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	db, err := database.Open(ctx, cfg.DatabasePath)
	if err != nil {
		return err
	}
	defer db.Close()

	if err := database.Migrate(ctx, db); err != nil {
		return err
	}

	m, err := media.New(cfg.StorageDir)
	if err != nil {
		return err
	}
	// Leftovers from interrupted uploads.
	os.RemoveAll(filepath.Join(cfg.StorageDir, "tmp"))

	archive := archiveorg.New(archiveorg.DefaultBaseURL, "comics/"+fullVersion()+" (+https://github.com/Oshuma/comics)")

	srv := &http.Server{
		Addr:              cfg.Listen,
		Handler:           server.New(cfg, store.New(db, m), m, web.Dist(), archive, fullVersion()).Handler(),
		ReadHeaderTimeout: 10 * time.Second,
		IdleTimeout:       2 * time.Minute,
	}

	errCh := make(chan error, 1)
	go func() {
		slog.Info("listening", "addr", cfg.Listen, "database", cfg.DatabasePath, "storage", cfg.StorageDir, "version", fullVersion())
		errCh <- srv.ListenAndServe()
	}()

	select {
	case err := <-errCh:
		if !errors.Is(err, http.ErrServerClosed) {
			return err
		}
	case <-ctx.Done():
		slog.Info("shutting down")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		return srv.Shutdown(shutdownCtx)
	}
	return nil
}
