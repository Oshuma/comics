package archiveorg

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"sync/atomic"
	"testing"
)

const fakeMetadata = `{
	"metadata": {"title": ["Batman", "Collection"]},
	"files": [
		{"name": "Batman 001.cbz", "format": "Comic Book ZIP", "size": "11"},
		{"name": "extras/Batman 002.CBR", "format": "Comic Book RAR", "size": "22"},
		{"name": "Batman 003.cbr", "format": "Comic Book RAR", "size": "33", "private": "true"},
		{"name": "Batman 001.pdf", "format": "Text PDF", "size": "44"},
		{"name": "__ia_thumb.jpg", "format": "Item Tile", "size": "5"}
	]
}`

// The query parameters of the last search the fake archive received.
var lastSearch atomic.Value

func searchParam(name string) string {
	return lastSearch.Load().(url.Values).Get(name)
}

func fakeArchive(t *testing.T) (*Client, *atomic.Int32) {
	t.Helper()
	var metadataCalls atomic.Int32
	mux := http.NewServeMux()
	mux.HandleFunc("GET /advancedsearch.php", func(w http.ResponseWriter, r *http.Request) {
		lastSearch.Store(r.URL.Query())
		w.Write([]byte(`{"response": {"numFound": 2, "docs": [
			{"identifier": "batman-collection", "title": "Batman Collection"},
			{"identifier": "missing-item", "title": "Gone"}
		]}}`))
	})
	mux.HandleFunc("GET /metadata/{id}", func(w http.ResponseWriter, r *http.Request) {
		metadataCalls.Add(1)
		switch r.PathValue("id") {
		case "batman-collection":
			w.Write([]byte(fakeMetadata))
		case "missing-item":
			w.Write([]byte(`{}`))
		default:
			http.Error(w, "boom", http.StatusInternalServerError)
		}
	})
	// Like archive.org, downloads redirect to the server holding the file.
	mux.HandleFunc("GET /download/batman-collection/{name...}", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/items/batman-collection/"+r.PathValue("name"), http.StatusFound)
	})
	mux.HandleFunc("GET /items/batman-collection/{name...}", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("contents of " + r.PathValue("name")))
	})
	mux.HandleFunc("GET /download/evil-item/{name...}", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "http://169.254.169.254/latest/meta-data", http.StatusFound)
	})
	ts := httptest.NewServer(mux)
	t.Cleanup(ts.Close)
	return New(ts.URL, "comics-test"), &metadataCalls
}

func TestSearch(t *testing.T) {
	c, metadataCalls := fakeArchive(t)
	res, err := c.Search(context.Background(), " batman ", 1, "")
	if err != nil {
		t.Fatal(err)
	}
	if q := searchParam("q"); q != BuildQuery("batman") {
		t.Fatalf("unexpected query %q", q)
	}
	if s := searchParam("sort[]"); s != "" {
		t.Fatalf("relevance search sent sort %q", s)
	}
	if res.TotalItems != 2 || res.Page != 1 || res.PerPage != PerPage {
		t.Fatalf("unexpected paging: %+v", res)
	}
	want := []File{
		{Identifier: "batman-collection", ItemTitle: "Batman, Collection", Name: "Batman 001.cbz", Size: 11},
		{Identifier: "batman-collection", ItemTitle: "Batman, Collection", Name: "extras/Batman 002.CBR", Size: 22},
	}
	if len(res.Files) != len(want) {
		t.Fatalf("got %+v, want %+v", res.Files, want)
	}
	for i := range want {
		if res.Files[i] != want[i] {
			t.Fatalf("file %d: got %+v, want %+v", i, res.Files[i], want[i])
		}
	}

	// Item metadata is cached.
	before := metadataCalls.Load()
	if _, err := c.ItemFile(context.Background(), "batman-collection", "Batman 001.cbz"); err != nil {
		t.Fatal(err)
	}
	if metadataCalls.Load() != before {
		t.Fatal("expected cached metadata to be reused")
	}

	if _, err := c.Search(context.Background(), "  ", 1, ""); err == nil {
		t.Fatal("expected empty query to fail")
	}
}

func TestItemFile(t *testing.T) {
	c, _ := fakeArchive(t)
	ctx := context.Background()
	for _, tc := range []struct{ identifier, name string }{
		{"batman-collection", "Batman 003.cbr"}, // private
		{"batman-collection", "Batman 001.pdf"}, // not a comic
		{"batman-collection", "nope.cbz"},
		{"missing-item", "x.cbz"},
		{"../etc", "passwd.cbz"},
	} {
		if _, err := c.ItemFile(ctx, tc.identifier, tc.name); !errors.Is(err, ErrNotFound) {
			t.Errorf("%s/%s: expected ErrNotFound, got %v", tc.identifier, tc.name, err)
		}
	}
	if _, err := c.ItemFile(ctx, "server-error", "x.cbz"); err == nil || errors.Is(err, ErrNotFound) {
		t.Errorf("expected server error to be reported, got %v", err)
	}
}

func TestDownload(t *testing.T) {
	c, _ := fakeArchive(t)
	ctx := context.Background()

	f, err := c.ItemFile(ctx, "batman-collection", "extras/Batman 002.CBR")
	if err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	if err := c.Download(ctx, f, &buf, 1<<20); err != nil {
		t.Fatal(err)
	}
	if buf.String() != "contents of extras/Batman 002.CBR" {
		t.Fatalf("unexpected contents %q", buf.String())
	}

	buf.Reset()
	if err := c.Download(ctx, f, &buf, 5); !errors.Is(err, ErrTooLarge) {
		t.Fatalf("expected ErrTooLarge, got %v", err)
	}

	evil := &File{Identifier: "evil-item", Name: "x.cbz"}
	if err := c.Download(ctx, evil, &buf, 1<<20); err == nil || !strings.Contains(err.Error(), "refusing redirect") {
		t.Fatalf("expected redirect off archive.org to be refused, got %v", err)
	}
}

// Runs against the real archive.org when COMICS_LIVE_TESTS=1.
func TestLive(t *testing.T) {
	if os.Getenv("COMICS_LIVE_TESTS") != "1" {
		t.Skip("set COMICS_LIVE_TESTS=1 to run against archive.org")
	}
	c := New(DefaultBaseURL, "comics-test (+https://github.com/Oshuma/comics)")
	ctx := context.Background()
	res, err := c.Search(ctx, "batman year one", 1, "")
	if err != nil {
		t.Fatal(err)
	}
	if res.TotalItems == 0 || len(res.Files) == 0 {
		t.Fatalf("expected results, got %+v", res)
	}
	t.Logf("%d items, %d files on page 1; first: %+v", res.TotalItems, len(res.Files), res.Files[0])

	// Download the smallest file found, to keep the test quick.
	f := res.Files[0]
	for _, x := range res.Files {
		if x.Size > 0 && x.Size < f.Size {
			f = x
		}
	}
	var n countingWriter
	if err := c.Download(ctx, &f, &n, 200<<20); err != nil {
		t.Fatal(err)
	}
	if int64(n) != f.Size {
		t.Fatalf("downloaded %d bytes, expected %d", n, f.Size)
	}
	t.Logf("downloaded %s/%s (%d bytes)", f.Identifier, f.Name, n)
}

type countingWriter int64

func (w *countingWriter) Write(p []byte) (int, error) {
	*w += countingWriter(len(p))
	return len(p), nil
}

func TestSearchSort(t *testing.T) {
	c, _ := fakeArchive(t)
	for sort, want := range map[string]string{
		"downloads": "downloads desc",
		"newest":    "addeddate desc",
		"title":     "titleSorter asc",
		"size":      "item_size desc",
		"relevance": "",
		"bogus":     "",
	} {
		if _, err := c.Search(context.Background(), "batman", 1, sort); err != nil {
			t.Fatal(err)
		}
		if got := searchParam("sort[]"); got != want {
			t.Errorf("sort %q: sent %q, want %q", sort, got, want)
		}
	}
}

func TestSearchSizeOrdersFiles(t *testing.T) {
	c, _ := fakeArchive(t)
	res, err := c.Search(context.Background(), "batman", 1, "size")
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Files) != 2 || res.Files[0].Size != 22 || res.Files[1].Size != 11 {
		t.Fatalf("files not ordered largest first: %+v", res.Files)
	}
}

func TestBuildQuery(t *testing.T) {
	const formats = ` AND format:("Comic Book ZIP" OR "Comic Book RAR")`
	weighted := func(q string) string {
		return "(title:(" + q + ")^5 OR subject:(" + q + ")^2 OR creator:(" + q + "))" + formats
	}
	for in, want := range map[string]string{
		"batman year one":                  weighted("batman year one"),
		"Batman: Year One":                 weighted(`Batman\: Year One`),
		"Spider-Man (2018) #1":             weighted(`Spider\-Man \(2018\) #1`),
		`"year one" batman`:                weighted(`"year one" batman`),
		`batman "year one`:                 weighted(`batman \"year one`),
		`creator:"Frank Miller"`:           `(creator:"Frank Miller")` + formats,
		`batman AND collection:comics`:     `(batman AND collection:comics)` + formats,
		`(title:batman OR title:superman)`: `((title:batman OR title:superman))` + formats,
	} {
		if got := BuildQuery(in); got != want {
			t.Errorf("BuildQuery(%q)\n got: %s\nwant: %s", in, got, want)
		}
	}
}
