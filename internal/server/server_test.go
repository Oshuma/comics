package server

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"image"
	"image/jpeg"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"github.com/Oshuma/comics/internal/archiveorg"
	"github.com/Oshuma/comics/internal/config"
	"github.com/Oshuma/comics/internal/database"
	"github.com/Oshuma/comics/internal/importer"
	"github.com/Oshuma/comics/internal/media"
	"github.com/Oshuma/comics/internal/store"
)

type client struct {
	t       *testing.T
	http    *http.Client
	base    string
	storage string
}

func (c *client) do(method, path string, body io.Reader, contentType string, out any) int {
	c.t.Helper()
	req, _ := http.NewRequest(method, c.base+path, body)
	req.Header.Set("X-Requested-With", "test")
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	res, err := c.http.Do(req)
	if err != nil {
		c.t.Fatal(err)
	}
	defer res.Body.Close()
	if out != nil {
		if err := json.NewDecoder(res.Body).Decode(out); err != nil {
			c.t.Fatalf("%s %s: decoding response: %v", method, path, err)
		}
	}
	return res.StatusCode
}

func (c *client) json(method, path string, in, out any) int {
	b, _ := json.Marshal(in)
	return c.do(method, path, bytes.NewReader(b), "application/json", out)
}

func newTestServer(t *testing.T) *client {
	// Nothing listens here; tests that need archive.org use newTestServerWithArchive.
	return newTestServerWithArchive(t, "http://127.0.0.1:1")
}

func newTestServerWithArchive(t *testing.T, archiveURL string) *client {
	dir := t.TempDir()
	ctx := context.Background()
	db, err := database.Open(ctx, filepath.Join(dir, "comics.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	if err := database.Migrate(ctx, db); err != nil {
		t.Fatal(err)
	}
	m, err := media.New(filepath.Join(dir, "storage"))
	if err != nil {
		t.Fatal(err)
	}
	cfg := &config.Config{MaxUploadMB: 10}
	st := store.New(db, m)
	archive := archiveorg.New(archiveURL, "comics-test")
	imp := importer.New(st, m, archive, cfg.MaxUploadMB<<20, 2)
	impCtx, stopImporter := context.WithCancel(context.Background())
	if err := imp.Start(impCtx); err != nil {
		t.Fatal(err)
	}
	static := fstest.MapFS{"index.html": {Data: []byte("<html></html>")}}
	ts := httptest.NewServer(New(cfg, st, m, static, archive, imp, "1.2.3+abc").Handler())
	t.Cleanup(func() {
		ts.Close()
		stopImporter()
		imp.Wait()
	})

	jar, _ := cookiejar.New(nil)
	return &client{t: t, http: &http.Client{Jar: jar}, base: ts.URL, storage: filepath.Join(dir, "storage")}
}

// waitImport polls until a queued import finishes, and returns it.
func (c *client) waitImport(id int) store.Import {
	c.t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		var jobs []store.Import
		c.do("GET", "/api/imports", nil, "", &jobs)
		for _, j := range jobs {
			if j.ID == id && (j.Status == store.ImportDone || j.Status == store.ImportFailed) {
				return j
			}
		}
		time.Sleep(20 * time.Millisecond)
	}
	c.t.Fatalf("import %d didn't finish", id)
	return store.Import{}
}

// importedComic waits for an import to succeed and returns its comic.
func (c *client) importedComic(id int) store.ComicDetail {
	c.t.Helper()
	job := c.waitImport(id)
	if job.Status != store.ImportDone || job.ComicID == nil {
		c.t.Fatalf("import failed: %+v", job)
	}
	var comic store.ComicDetail
	c.do("GET", "/api/comics/"+strconv.Itoa(*job.ComicID), nil, "", &comic)
	return comic
}

func cbz(t *testing.T, pages int) []byte {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for i := range pages {
		w, _ := zw.Create(string(rune('a'+i)) + ".jpg")
		jpeg.Encode(w, image.NewRGBA(image.Rect(0, 0, 100, 150)), nil)
	}
	zw.Close()
	return buf.Bytes()
}

func TestUploadAndRead(t *testing.T) {
	c := newTestServer(t)

	var status struct {
		SetupRequired bool
		Version       string
	}
	c.do("GET", "/api/status", nil, "", &status)
	if !status.SetupRequired || status.Version != "1.2.3+abc" {
		t.Fatalf("unexpected status: %+v", status)
	}
	user := map[string]string{"email": "admin@example.com", "password": "password1", "passwordConfirmation": "password1"}
	if code := c.json("POST", "/api/setup", user, nil); code != http.StatusCreated {
		t.Fatalf("setup: %d", code)
	}
	if code := c.json("POST", "/api/setup", user, nil); code != http.StatusConflict {
		t.Fatalf("second setup: %d", code)
	}

	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	mw.WriteField("group_name", "Saga")
	fw, _ := mw.CreateFormFile("comic", "Saga 01.cbz")
	fw.Write(cbz(t, 2))
	mw.Close()
	var job store.Import
	if code := c.do("POST", "/api/comics", &body, mw.FormDataContentType(), &job); code != http.StatusAccepted {
		t.Fatalf("upload: %d", code)
	}
	if job.Status != store.ImportQueued || job.Source != store.SourceUpload || job.GroupName != "Saga" {
		t.Fatalf("unexpected job: %+v", job)
	}
	comic := c.importedComic(job.ID)
	if comic.Name != "Saga 01" || comic.PageCount != 2 {
		t.Fatalf("unexpected comic: %+v", comic)
	}

	var detail store.ComicDetail
	c.do("GET", "/api/comics/"+strconv.Itoa(comic.ID), nil, "", &detail)
	var view store.PageView
	path := "/api/comics/" + strconv.Itoa(comic.ID) + "/pages/" + strconv.Itoa(detail.Pages[1].ID) + "/current"
	if code := c.do("PUT", path, nil, "", &view); code != http.StatusOK || view.PrevPageID == nil || view.GroupName != "Saga" {
		t.Fatalf("view page: %d %+v", code, view)
	}

	for _, suffix := range []string{"image", "thumb"} {
		res, err := c.http.Get(c.base + "/api/pages/" + strconv.Itoa(detail.Pages[0].ID) + "/" + suffix)
		if err != nil {
			t.Fatal(err)
		}
		res.Body.Close()
		if res.StatusCode != http.StatusOK || res.Header.Get("Content-Type") != "image/jpeg" {
			t.Fatalf("%s: %d %s", suffix, res.StatusCode, res.Header.Get("Content-Type"))
		}
	}

	var finish struct{ GroupID int }
	if code := c.do("PUT", "/api/comics/"+strconv.Itoa(comic.ID)+"/finish", nil, "", &finish); code != http.StatusOK || finish.GroupID != comic.GroupID {
		t.Fatalf("finish: %d %+v", code, finish)
	}

	path = "/api/comics/" + strconv.Itoa(comic.ID)
	if code := c.json("PUT", path+"/read", map[string]bool{"read": false}, nil); code != http.StatusOK {
		t.Fatalf("mark unread: %d", code)
	}
	c.do("GET", path, nil, "", &detail)
	if detail.Read || detail.Reading {
		t.Fatalf("after mark unread: %+v", detail.Comic)
	}
	if code := c.json("PUT", path+"/read", map[string]bool{"read": true}, nil); code != http.StatusOK {
		t.Fatalf("mark read: %d", code)
	}
	c.do("GET", path, nil, "", &detail)
	if !detail.Read {
		t.Fatalf("after mark read: %+v", detail.Comic)
	}
	if code := c.json("PUT", "/api/comics/99999/read", map[string]bool{"read": false}, nil); code != http.StatusNotFound {
		t.Fatalf("mark unread missing comic: %d", code)
	}

	if code := c.json("PATCH", path, map[string]string{"name": "Saga #1"}, nil); code != http.StatusOK {
		t.Fatalf("rename: %d", code)
	}
	if code := c.json("PATCH", path, map[string]string{"name": " "}, nil); code != http.StatusUnprocessableEntity {
		t.Fatalf("blank rename: %d", code)
	}
	c.do("GET", path, nil, "", &detail)
	if detail.Name != "Saga #1" || detail.Filename != "Saga #1.cbz" {
		t.Fatalf("after rename: %+v", detail.Comic)
	}

	c.do("DELETE", "/api/session", nil, "", nil)
	if code := c.do("GET", "/api/groups", nil, "", nil); code != http.StatusUnauthorized {
		t.Fatalf("expected signed out, got %d", code)
	}
}

func (c *client) upload(groupName, filename string, data []byte, out any) int {
	c.t.Helper()
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	if groupName != "" {
		mw.WriteField("group_name", groupName)
	}
	fw, _ := mw.CreateFormFile("comic", filename)
	fw.Write(data)
	mw.Close()
	return c.do("POST", "/api/comics", &body, mw.FormDataContentType(), out)
}

func TestImportJobs(t *testing.T) {
	c := newTestServer(t)
	user := map[string]string{"email": "admin@example.com", "password": "password1", "passwordConfirmation": "password1"}
	c.json("POST", "/api/setup", user, nil)

	var ok, broken store.Import
	c.upload("Saga", "Saga 01.cbz", cbz(t, 1), &ok)
	c.upload("Saga", "Notes.cbz", []byte("not an archive"), &broken)
	c.importedComic(ok.ID)
	if j := c.waitImport(broken.ID); j.Status != store.ImportFailed || !strings.Contains(j.Error, "Unsupported file type") {
		t.Fatalf("broken upload: %+v", j)
	}

	// Rejected before queueing: nothing is left in the spool.
	var res struct{ Error string }
	if code := c.upload("", "x.cbz", cbz(t, 1), &res); code != http.StatusUnprocessableEntity || !strings.Contains(res.Error, "select a group") {
		t.Fatalf("upload without group: %d %q", code, res.Error)
	}
	if entries, _ := os.ReadDir(filepath.Join(c.storage, "imports")); len(entries) != 0 {
		t.Fatalf("spool not cleaned up: %v", entries)
	}

	var jobs []store.Import
	c.do("GET", "/api/imports", nil, "", &jobs)
	if len(jobs) != 2 || jobs[0].ID != broken.ID {
		t.Fatalf("list: %+v", jobs)
	}

	// Another user can't see or delete them.
	other := map[string]string{"email": "b@example.com", "password": "password1", "passwordConfirmation": "password1"}
	c.json("POST", "/api/admin/users", other, nil)
	c.do("DELETE", "/api/session", nil, "", nil)
	c.json("POST", "/api/session", map[string]string{"email": "b@example.com", "password": "password1"}, nil)
	c.do("GET", "/api/imports", nil, "", &jobs)
	if len(jobs) != 0 {
		t.Fatalf("other user sees jobs: %+v", jobs)
	}
	if code := c.do("DELETE", "/api/imports/"+strconv.Itoa(broken.ID), nil, "", nil); code != http.StatusNotFound {
		t.Fatalf("other user deleted a job: %d", code)
	}

	c.do("DELETE", "/api/session", nil, "", nil)
	c.json("POST", "/api/session", map[string]string{"email": "admin@example.com", "password": "password1"}, nil)
	if code := c.do("DELETE", "/api/imports/"+strconv.Itoa(broken.ID), nil, "", nil); code != http.StatusOK {
		t.Fatalf("dismiss: %d", code)
	}
	c.do("GET", "/api/imports", nil, "", &jobs)
	if len(jobs) != 1 || jobs[0].ID != ok.ID {
		t.Fatalf("after dismiss: %+v", jobs)
	}
}

func TestChangePassword(t *testing.T) {
	c := newTestServer(t)
	user := map[string]string{"email": "admin@example.com", "password": "password1", "passwordConfirmation": "password1"}
	c.json("POST", "/api/setup", user, nil)

	change := map[string]string{"currentPassword": "wrong", "password": "newpassword", "passwordConfirmation": "newpassword"}
	if code := c.json("PUT", "/api/account/password", change, nil); code != http.StatusUnprocessableEntity {
		t.Fatalf("wrong current password: %d", code)
	}
	change["currentPassword"] = "password1"
	if code := c.json("PUT", "/api/account/password", change, nil); code != http.StatusOK {
		t.Fatalf("change password: %d", code)
	}

	// The session that changed the password stays signed in.
	if code := c.do("GET", "/api/groups", nil, "", nil); code != http.StatusOK {
		t.Fatalf("session after change: %d", code)
	}

	c.do("DELETE", "/api/session", nil, "", nil)
	if code := c.json("POST", "/api/session", map[string]string{"email": "admin@example.com", "password": "password1"}, nil); code != http.StatusUnauthorized {
		t.Fatalf("login with old password: %d", code)
	}
	if code := c.json("POST", "/api/session", map[string]string{"email": "admin@example.com", "password": "newpassword"}, nil); code != http.StatusOK {
		t.Fatalf("login with new password: %d", code)
	}
}
