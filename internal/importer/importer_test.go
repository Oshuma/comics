package importer

import (
	"archive/zip"
	"bytes"
	"context"
	"image"
	"image/jpeg"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/Oshuma/comics/internal/archiveorg"
	"github.com/Oshuma/comics/internal/database"
	"github.com/Oshuma/comics/internal/media"
	"github.com/Oshuma/comics/internal/store"
)

type env struct {
	t       *testing.T
	store   *store.Store
	media   *media.Storage
	root    string
	im      *Importer
	userID  int
	groupID int
	// Closed to let slow downloads finish.
	release chan struct{}
}

func cbz(t *testing.T, pages int) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for i := range pages {
		w, _ := zw.Create(strconv.Itoa(i) + ".jpg")
		jpeg.Encode(w, image.NewRGBA(image.Rect(0, 0, 40, 60)), nil)
	}
	zw.Close()
	return buf.Bytes()
}

func newEnv(t *testing.T) *env {
	t.Helper()
	ctx := context.Background()
	dir := t.TempDir()
	db, err := database.Open(ctx, filepath.Join(dir, "comics.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	if err := database.Migrate(ctx, db); err != nil {
		t.Fatal(err)
	}
	root := filepath.Join(dir, "storage")
	m, err := media.New(root)
	if err != nil {
		t.Fatal(err)
	}
	st := store.New(db, m)
	u, err := st.CreateUser(ctx, store.NewUser{Email: "a@example.com", Password: "password1", PasswordConfirmation: "password1"})
	if err != nil {
		t.Fatal(err)
	}
	g, _ := st.CreateGroup(ctx, u.ID, "Saga")

	e := &env{t: t, store: st, media: m, root: root, userID: u.ID, groupID: g.ID, release: make(chan struct{})}
	comic := cbz(t, 3)
	mux := http.NewServeMux()
	mux.HandleFunc("GET /metadata/saga", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"files": [
			{"name": "Saga 01.cbz", "size": "` + strconv.Itoa(len(comic)) + `"},
			{"name": "Slow.cbz", "size": "` + strconv.Itoa(len(comic)) + `"},
			{"name": "Broken.cbz", "size": "10"}
		]}`))
	})
	mux.HandleFunc("GET /download/saga/{name}", func(w http.ResponseWriter, r *http.Request) {
		switch r.PathValue("name") {
		case "Saga 01.cbz":
			w.Write(comic)
		case "Slow.cbz":
			select {
			case <-e.release:
				w.Write(comic)
			case <-r.Context().Done():
			}
		default:
			w.Write([]byte("not a zip!"))
		}
	})
	ts := httptest.NewServer(mux)
	t.Cleanup(func() {
		select {
		case <-e.release:
		default:
			close(e.release)
		}
		ts.Close()
	})
	e.im = New(st, m, archiveorg.New(ts.URL, "comics-test"), 10<<20, 2)
	return e
}

// start runs the workers until the test ends, or until the returned
// function is called.
func (e *env) start() (stop func()) {
	ctx, cancel := context.WithCancel(context.Background())
	if err := e.im.Start(ctx); err != nil {
		e.t.Fatal(err)
	}
	stopped := false
	stop = func() {
		if !stopped {
			stopped = true
			cancel()
			e.im.Wait()
		}
	}
	e.t.Cleanup(stop)
	return stop
}

func (e *env) archiveJob(name string) *store.Import {
	e.t.Helper()
	j, err := e.store.CreateImport(context.Background(), store.NewImport{
		UserID: e.userID, GroupID: e.groupID, Filename: name, Source: store.SourceArchive,
		ArchiveIdentifier: "saga", ArchiveName: name,
	})
	if err != nil {
		e.t.Fatal(err)
	}
	e.im.Notify()
	return j
}

func (e *env) uploadJob(name string, data []byte) *store.Import {
	e.t.Helper()
	dir, err := e.media.SpoolDir()
	if err != nil {
		e.t.Fatal(err)
	}
	path := filepath.Join(dir, "archive")
	os.WriteFile(path, data, 0o644)
	j, err := e.store.CreateImport(context.Background(), store.NewImport{
		UserID: e.userID, GroupID: e.groupID, Filename: name, Source: store.SourceUpload, UploadPath: path,
	})
	if err != nil {
		e.t.Fatal(err)
	}
	e.im.Notify()
	return j
}

// wait polls until the job reaches one of the statuses.
func (e *env) wait(id int, statuses ...string) *store.Import {
	e.t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		j, err := e.store.GetImport(context.Background(), e.userID, id)
		if err != nil {
			e.t.Fatal(err)
		}
		for _, s := range statuses {
			if j.Status == s {
				return j
			}
		}
		time.Sleep(20 * time.Millisecond)
	}
	e.t.Fatalf("import %d didn't reach %v", id, statuses)
	return nil
}

func (e *env) spoolEntries() []os.DirEntry {
	entries, _ := os.ReadDir(filepath.Join(e.root, "imports"))
	return entries
}

func TestImports(t *testing.T) {
	e := newEnv(t)
	e.start()

	up := e.uploadJob("Saga 01.cbz", cbz(t, 2))
	ia := e.archiveJob("Saga 01.cbz")
	broken := e.archiveJob("Broken.cbz")
	missing := e.archiveJob("Gone.cbz")

	for _, j := range []*store.Import{up, ia} {
		done := e.wait(j.ID, store.ImportDone, store.ImportFailed)
		if done.Status != store.ImportDone || done.ComicID == nil {
			t.Fatalf("import %d: %+v", j.ID, done)
		}
	}
	upComic, _ := e.store.GetComic(context.Background(), e.userID, *e.wait(up.ID, store.ImportDone).ComicID)
	iaComic, _ := e.store.GetComic(context.Background(), e.userID, *e.wait(ia.ID, store.ImportDone).ComicID)
	if upComic.PageCount != 2 || iaComic.PageCount != 3 || upComic.GroupID != e.groupID {
		t.Fatalf("unexpected comics: %+v %+v", upComic.Comic, iaComic.Comic)
	}

	if j := e.wait(broken.ID, store.ImportFailed); j.Error != "Unsupported file type. Upload a CBZ or CBR archive." {
		t.Fatalf("broken: %q", j.Error)
	}
	if j := e.wait(missing.ID, store.ImportFailed); j.Error != "That file isn't available on archive.org." {
		t.Fatalf("missing: %q", j.Error)
	}

	if entries := e.spoolEntries(); len(entries) != 0 {
		t.Fatalf("spooled uploads left behind: %v", entries)
	}
	if entries, _ := os.ReadDir(filepath.Join(e.root, "tmp")); len(entries) != 0 {
		t.Fatalf("temp files left behind: %v", entries)
	}
}

func TestCancel(t *testing.T) {
	e := newEnv(t)
	e.start()

	j := e.archiveJob("Slow.cbz")
	e.wait(j.ID, store.ImportProcessing)
	deadline := time.Now().Add(5 * time.Second)
	for {
		if p, ok := e.im.Progress(j.ID); ok && p.Phase == PhaseDownloading && p.Total > 0 && p.Done == 0 {
			break
		}
		if time.Now().After(deadline) {
			p, ok := e.im.Progress(j.ID)
			t.Fatalf("expected download progress, got %+v %v", p, ok)
		}
		time.Sleep(10 * time.Millisecond)
	}
	if _, err := e.store.DeleteImport(context.Background(), e.userID, j.ID); err != nil {
		t.Fatal(err)
	}
	e.im.Cancel(j.ID)

	// The worker moves on to the next job.
	next := e.archiveJob("Saga 01.cbz")
	e.wait(next.ID, store.ImportDone)
	if _, ok := e.im.Progress(j.ID); ok {
		t.Fatal("progress kept for a cancelled job")
	}
	if _, ok := e.im.Progress(next.ID); ok {
		t.Fatal("progress kept for a finished job")
	}
	comics, _ := e.store.ListComics(context.Background(), e.userID, e.groupID)
	if len(comics) != 1 {
		t.Fatalf("expected only the second import's comic, got %+v", comics)
	}
}

func TestShutdownAndResume(t *testing.T) {
	e := newEnv(t)
	stop := e.start()

	slow := e.archiveJob("Slow.cbz")
	e.wait(slow.ID, store.ImportProcessing)
	stop()
	if j, _ := e.store.GetImport(context.Background(), e.userID, slow.ID); j.Status != store.ImportQueued {
		t.Fatalf("expected interrupted job to be requeued, got %+v", j)
	}

	// Simulate a crash mid-job: a job left processing, plus a queued upload
	// and an orphaned spool directory from a deleted job.
	crashed := e.archiveJob("Saga 01.cbz")
	queued := e.uploadJob("Saga 02.cbz", cbz(t, 1))
	e.store.ClaimImport(context.Background()) // slow
	e.store.ClaimImport(context.Background()) // crashed
	orphan, _ := e.media.SpoolDir()

	close(e.release)
	e.im = New(e.store, e.media, e.im.archive, e.im.maxBytes, 2)
	e.start()

	for _, j := range []*store.Import{slow, crashed, queued} {
		if done := e.wait(j.ID, store.ImportDone, store.ImportFailed); done.Status != store.ImportDone {
			t.Fatalf("import %d after restart: %+v", j.ID, done)
		}
	}
	if _, err := os.Stat(orphan); !os.IsNotExist(err) {
		t.Fatalf("orphaned spool dir not removed: %v", err)
	}
}
