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

	"github.com/Oshuma/comics/internal/config"
	"github.com/Oshuma/comics/internal/database"
	"github.com/Oshuma/comics/internal/media"
	"github.com/Oshuma/comics/internal/server"
	"github.com/Oshuma/comics/internal/store"
	"github.com/Oshuma/comics/web"
)

// Set at build time with -ldflags "-X main.version=...".
var version = "dev"

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
		fmt.Println(version)
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

	pool, err := database.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()

	if err := database.Migrate(ctx, pool); err != nil {
		return err
	}

	m, err := media.New(cfg.StorageDir)
	if err != nil {
		return err
	}
	// Leftovers from interrupted uploads.
	os.RemoveAll(filepath.Join(cfg.StorageDir, "tmp"))

	srv := &http.Server{
		Addr:              cfg.Listen,
		Handler:           server.New(cfg, store.New(pool, m), m, web.Dist()).Handler(),
		ReadHeaderTimeout: 10 * time.Second,
		IdleTimeout:       2 * time.Minute,
	}

	errCh := make(chan error, 1)
	go func() {
		slog.Info("listening", "addr", cfg.Listen, "storage", cfg.StorageDir, "version", version)
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
