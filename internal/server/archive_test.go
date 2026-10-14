package server

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/Oshuma/comics/internal/store"
)

// fakeArchiveOrg serves one item with a real CBZ, an oversized CBZ and a
// broken one.
func fakeArchiveOrg(t *testing.T) *httptest.Server {
	t.Helper()
	comic := cbz(t, 2)
	mux := http.NewServeMux()
	mux.HandleFunc("GET /advancedsearch.php", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"response": {"numFound": 1, "docs": [{"identifier": "saga", "title": "Saga"}]}}`))
	})
	mux.HandleFunc("GET /metadata/saga", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"metadata": {"title": "Saga"}, "files": [
			{"name": "issues/Saga 01.cbz", "format": "Comic Book ZIP", "size": "` + itoa(len(comic)) + `"},
			{"name": "Saga Omnibus.cbz", "format": "Comic Book ZIP", "size": "999999999"},
			{"name": "Broken.cbz", "format": "Comic Book ZIP", "size": "10"}
		]}`))
	})
	mux.HandleFunc("GET /download/saga/{name...}", func(w http.ResponseWriter, r *http.Request) {
		switch r.PathValue("name") {
		case "issues/Saga 01.cbz":
			w.Write(comic)
		case "Broken.cbz":
			w.Write([]byte("not a zip!"))
		default:
			http.NotFound(w, r)
		}
	})
	ts := httptest.NewServer(mux)
	t.Cleanup(ts.Close)
	return ts
}

func TestArchiveSearchAndImport(t *testing.T) {
	c := newTestServerWithArchive(t, fakeArchiveOrg(t).URL)
	user := map[string]string{"email": "admin@example.com", "password": "password1", "passwordConfirmation": "password1"}
	if code := c.json("POST", "/api/setup", user, nil); code != http.StatusCreated {
		t.Fatalf("setup: %d", code)
	}

	var results struct {
		Files []struct {
			Identifier, Name, ItemTitle string
			Size                        int64
		}
		TotalItems int
		MaxBytes   int64
	}
	if code := c.do("GET", "/api/archive/search?q=saga", nil, "", &results); code != http.StatusOK {
		t.Fatalf("search: %d", code)
	}
	if len(results.Files) != 3 || results.TotalItems != 1 || results.MaxBytes != 10<<20 || results.Files[0].Name != "issues/Saga 01.cbz" {
		t.Fatalf("unexpected results: %+v", results)
	}
	if code := c.do("GET", "/api/archive/search?q=", nil, "", nil); code != http.StatusBadRequest {
		t.Fatalf("empty search: %d", code)
	}

	importFile := func(req map[string]any, out any) int {
		return c.json("POST", "/api/archive/import", req, out)
	}

	var comic store.Comic
	if code := importFile(map[string]any{"identifier": "saga", "name": "issues/Saga 01.cbz", "groupName": "Saga"}, &comic); code != http.StatusCreated {
		t.Fatalf("import: %d", code)
	}
	if comic.Name != "Saga 01" || comic.PageCount != 2 {
		t.Fatalf("unexpected comic: %+v", comic)
	}

	for _, tc := range []struct {
		name string
		req  map[string]any
		code int
		msg  string
	}{
		{"no group", map[string]any{"identifier": "saga", "name": "issues/Saga 01.cbz"}, http.StatusUnprocessableEntity, "select a group"},
		{"other user's group", map[string]any{"identifier": "saga", "name": "issues/Saga 01.cbz", "groupId": 999}, http.StatusNotFound, ""},
		{"not in item", map[string]any{"identifier": "saga", "name": "../../etc/passwd", "groupId": comic.GroupID}, http.StatusNotFound, "isn't available"},
		{"too large", map[string]any{"identifier": "saga", "name": "Saga Omnibus.cbz", "groupId": comic.GroupID}, http.StatusRequestEntityTooLarge, "too large"},
		{"broken archive", map[string]any{"identifier": "saga", "name": "Broken.cbz", "groupId": comic.GroupID}, http.StatusUnprocessableEntity, "Unsupported file type"},
	} {
		var res struct{ Error string }
		if code := importFile(tc.req, &res); code != tc.code || !strings.Contains(res.Error, tc.msg) {
			t.Errorf("%s: got %d %q, want %d containing %q", tc.name, code, res.Error, tc.code, tc.msg)
		}
	}

	var group struct{ Comics []store.Comic }
	c.do("GET", "/api/groups/"+itoa(comic.GroupID), nil, "", &group)
	if len(group.Comics) != 1 {
		t.Fatalf("failed imports created comics: %+v", group.Comics)
	}
}

func TestArchiveSearchUnavailable(t *testing.T) {
	c := newTestServer(t) // archive.org is unreachable
	user := map[string]string{"email": "admin@example.com", "password": "password1", "passwordConfirmation": "password1"}
	c.json("POST", "/api/setup", user, nil)
	var res struct{ Error string }
	if code := c.do("GET", "/api/archive/search?q=saga", nil, "", &res); code != http.StatusBadGateway || !strings.Contains(res.Error, "archive.org") {
		t.Fatalf("got %d %q", code, res.Error)
	}
}

func itoa(n int) string { return strconv.Itoa(n) }
