// Package importer turns uploaded and downloaded archives into comics in
// background workers, so requests don't wait on extraction or downloads.
package importer

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/Oshuma/comics/internal/archive"
	"github.com/Oshuma/comics/internal/archiveorg"
	"github.com/Oshuma/comics/internal/media"
	"github.com/Oshuma/comics/internal/store"
)

const (
	// Finished jobs are kept this long so users can see the outcome.
	keepFinished = 7 * 24 * time.Hour
	// Workers also check for jobs this often, in case a wake-up was missed.
	pollInterval = time.Minute
)

type Importer struct {
	store    *store.Store
	media    *media.Storage
	archive  *archiveorg.Client
	maxBytes int64
	workers  int

	wake chan struct{}
	wg   sync.WaitGroup

	mu       sync.Mutex
	running  map[int]context.CancelFunc
	progress map[int]Progress
}

// Progress of a job being processed.
type Progress struct {
	Phase string `json:"phase"`
	// Bytes processed out of the total; zero total means unknown.
	Done  int64 `json:"done"`
	Total int64 `json:"total"`
}

const (
	PhaseDownloading = "downloading"
	PhaseExtracting  = "extracting"
	PhaseSaving      = "saving"
)

func New(st *store.Store, m *media.Storage, archive *archiveorg.Client, maxBytes int64, workers int) *Importer {
	return &Importer{
		store:    st,
		media:    m,
		archive:  archive,
		maxBytes: maxBytes,
		workers:  max(workers, 1),
		wake:     make(chan struct{}, 1),
		running:  map[int]context.CancelFunc{},
		progress: map[int]Progress{},
	}
}

// Start resumes jobs interrupted by a previous run and starts the workers,
// which run until ctx is cancelled. Jobs in progress then are requeued.
func (im *Importer) Start(ctx context.Context) error {
	n, err := im.store.ResetImports(ctx)
	if err != nil {
		return err
	}
	if n > 0 {
		slog.Info("resuming interrupted imports", "count", n)
	}
	keep, err := im.store.PendingUploadPaths(ctx)
	if err != nil {
		return err
	}
	if err := im.media.CleanSpool(keep); err != nil {
		slog.Warn("cleaning import spool", "err", err)
	}
	if err := im.store.PruneImports(ctx, keepFinished); err != nil {
		slog.Warn("pruning imports", "err", err)
	}

	for range im.workers {
		im.wg.Add(1)
		go im.work(ctx)
	}
	return nil
}

// Wait blocks until the workers have stopped.
func (im *Importer) Wait() { im.wg.Wait() }

// Notify wakes a worker to look for queued jobs.
func (im *Importer) Notify() {
	select {
	case im.wake <- struct{}{}:
	default:
	}
}

// Progress returns how far along a job being processed is.
func (im *Importer) Progress(id int) (Progress, bool) {
	im.mu.Lock()
	defer im.mu.Unlock()
	p, ok := im.progress[id]
	return p, ok
}

func (im *Importer) setProgress(id int, phase string, done, total int64) {
	im.mu.Lock()
	im.progress[id] = Progress{Phase: phase, Done: done, Total: total}
	im.mu.Unlock()
}

// Cancel stops a job if it's being processed.
func (im *Importer) Cancel(id int) {
	im.mu.Lock()
	defer im.mu.Unlock()
	if cancel, ok := im.running[id]; ok {
		cancel()
	}
}

func (im *Importer) work(ctx context.Context) {
	defer im.wg.Done()
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()
	for {
		job, err := im.store.ClaimImport(ctx)
		if err != nil && ctx.Err() == nil {
			slog.Error("claiming import", "err", err)
		}
		if job != nil {
			// More may be queued; let another idle worker look.
			im.Notify()
			im.run(ctx, job)
			continue
		}
		select {
		case <-ctx.Done():
			return
		case <-im.wake:
		case <-ticker.C:
		}
	}
}

func (im *Importer) run(ctx context.Context, job *store.Import) {
	jobCtx, cancel := context.WithCancel(ctx)
	im.mu.Lock()
	im.running[job.ID] = cancel
	im.mu.Unlock()
	defer func() {
		im.mu.Lock()
		delete(im.running, job.ID)
		delete(im.progress, job.ID)
		im.mu.Unlock()
		cancel()
	}()

	log := slog.With("import", job.ID, "file", job.Filename, "source", job.Source)
	log.Info("import started")
	start := time.Now()
	comicID, err := im.process(jobCtx, job)

	// Record the outcome even if we're shutting down.
	bg := context.WithoutCancel(ctx)
	switch {
	case ctx.Err() != nil:
		// Shutting down: pick the job up again on the next start, keeping
		// any spooled upload.
		if err := im.store.RequeueImport(bg, job.ID); err != nil && !errors.Is(err, store.ErrNotFound) {
			log.Error("requeueing import", "err", err)
		}
		log.Info("import interrupted; will resume on restart")
		return
	case jobCtx.Err() != nil:
		log.Info("import cancelled")
	case err != nil:
		msg := userMessage(err)
		log.Warn("import failed", "err", err)
		if err := im.store.FailImport(bg, job.ID, msg); err != nil && !errors.Is(err, store.ErrNotFound) {
			log.Error("recording failed import", "err", err)
		}
	default:
		log.Info("import finished", "comic", comicID, "duration", time.Since(start).Round(time.Millisecond))
		if err := im.store.FinishImport(bg, job.ID, comicID); err != nil && !errors.Is(err, store.ErrNotFound) {
			log.Error("recording finished import", "err", err)
		}
	}
	if job.UploadPath != "" {
		if err := im.media.RemoveSpool(job.UploadPath); err != nil {
			log.Warn("removing spooled upload", "err", err)
		}
	}
}

// process downloads (for archive.org jobs), extracts and stores a comic,
// returning its ID.
func (im *Importer) process(ctx context.Context, job *store.Import) (int, error) {
	tmpDir, err := im.media.TempDir()
	if err != nil {
		return 0, err
	}
	defer os.RemoveAll(tmpDir)

	archivePath := job.UploadPath
	if job.Source == store.SourceArchive {
		archivePath = filepath.Join(tmpDir, "archive")
		if err := im.download(ctx, job, archivePath); err != nil {
			return 0, err
		}
	}

	extractDir := filepath.Join(tmpDir, "pages")
	if err := os.Mkdir(extractDir, 0o755); err != nil {
		return 0, err
	}
	entries, err := archive.Extract(archivePath, extractDir, func(done, total int64) {
		im.setProgress(job.ID, PhaseExtracting, done, total)
	})
	if errors.Is(err, archive.ErrUnsupported) {
		return 0, failure("Unsupported file type. Upload a CBZ or CBR archive.", err)
	}
	if err != nil {
		return 0, failure("Could not read archive: "+err.Error(), err)
	}
	// Free the archive's space before pages are moved into place.
	if job.Source == store.SourceArchive {
		os.Remove(archivePath)
	}

	im.setProgress(job.ID, PhaseSaving, 0, 0)
	pages, err := media.Stage(extractDir, entries)
	if err != nil {
		return 0, err
	}
	comic, err := im.store.CreateComic(ctx, job.UserID, job.GroupID, job.Filename, extractDir, pages)
	if errors.Is(err, store.ErrNotFound) {
		return 0, failure("The group no longer exists.", err)
	}
	if err != nil {
		return 0, err
	}
	return comic.ID, nil
}

func (im *Importer) download(ctx context.Context, job *store.Import, dst string) error {
	file, err := im.archive.ItemFile(ctx, job.ArchiveIdentifier, job.ArchiveName)
	if errors.Is(err, archiveorg.ErrNotFound) {
		return failure("That file isn't available on archive.org.", err)
	}
	if err != nil {
		return failure("Couldn't reach archive.org. Try again in a moment.", err)
	}
	if file.Size > im.maxBytes {
		return failure("File is too large.", archiveorg.ErrTooLarge)
	}

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	im.setProgress(job.ID, PhaseDownloading, 0, file.Size)
	pw := &progressWriter{w: out, update: func(n int64) { im.setProgress(job.ID, PhaseDownloading, n, file.Size) }}
	err = im.archive.Download(ctx, file, pw, im.maxBytes)
	if cerr := out.Close(); err == nil {
		err = cerr
	}
	if errors.Is(err, archiveorg.ErrTooLarge) {
		return failure("File is too large.", err)
	}
	if err != nil {
		return failure("Downloading from archive.org failed. Try again in a moment.", err)
	}
	return nil
}

type progressWriter struct {
	w      io.Writer
	n      int64
	update func(n int64)
}

func (p *progressWriter) Write(b []byte) (int, error) {
	n, err := p.w.Write(b)
	p.n += int64(n)
	p.update(p.n)
	return n, err
}

// jobError carries a message suitable for showing to the user.
type jobError struct {
	msg string
	err error
}

func (e *jobError) Error() string { return e.msg + ": " + e.err.Error() }
func (e *jobError) Unwrap() error { return e.err }

func failure(msg string, err error) error { return &jobError{msg: msg, err: err} }

func userMessage(err error) string {
	var jerr *jobError
	if errors.As(err, &jerr) {
		return jerr.msg
	}
	var verr *store.ValidationError
	if errors.As(err, &verr) {
		return verr.Error()
	}
	return "Import failed due to a server error."
}
