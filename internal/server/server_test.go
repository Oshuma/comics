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
	"path/filepath"
	"strconv"
	"testing"
	"testing/fstest"

	"github.com/Oshuma/comics/internal/config"
	"github.com/Oshuma/comics/internal/database"
	"github.com/Oshuma/comics/internal/media"
	"github.com/Oshuma/comics/internal/store"
)

type client struct {
	t    *testing.T
	http *http.Client
	base string
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
	static := fstest.MapFS{"index.html": {Data: []byte("<html></html>")}}
	ts := httptest.NewServer(New(cfg, store.New(db, m), m, static, "1.2.3+abc").Handler())
	t.Cleanup(ts.Close)

	jar, _ := cookiejar.New(nil)
	return &client{t: t, http: &http.Client{Jar: jar}, base: ts.URL}
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
	var comic store.Comic
	if code := c.do("POST", "/api/comics", &body, mw.FormDataContentType(), &comic); code != http.StatusCreated {
		t.Fatalf("upload: %d", code)
	}
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
