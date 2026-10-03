package archive

import (
	"archive/zip"
	"os"
	"path/filepath"
	"sort"
	"testing"
)

func TestNaturalLess(t *testing.T) {
	names := []string{"page10.jpg", "page2.jpg", "Page1.jpg", "page02b.jpg", "cover.jpg", "page1.jpg"}
	sort.SliceStable(names, func(i, j int) bool { return naturalLess(names[i], names[j]) })
	want := []string{"Page1.jpg", "cover.jpg", "page1.jpg", "page2.jpg", "page02b.jpg", "page10.jpg"}
	for i := range want {
		if names[i] != want[i] {
			t.Fatalf("got %v, want %v", names, want)
		}
	}
}

func TestExtractZip(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "comic.cbr") // zip content with a .cbr name
	f, err := os.Create(src)
	if err != nil {
		t.Fatal(err)
	}
	zw := zip.NewWriter(f)
	for _, name := range []string{"b/10.jpg", "b/2.jpg", "../evil.jpg", "__MACOSX/._2.jpg", "Thumbs.db"} {
		w, _ := zw.Create(name)
		w.Write([]byte(name))
	}
	zw.Close()
	f.Close()

	out := filepath.Join(dir, "out")
	os.Mkdir(out, 0o755)
	entries, err := Extract(src, out)
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, e := range entries {
		got = append(got, e.Name)
		if filepath.Dir(e.Path) != out {
			t.Errorf("entry %q extracted outside dest: %s", e.Name, e.Path)
		}
	}
	want := []string{"../evil.jpg", "b/2.jpg", "b/10.jpg"}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v, want %v", got, want)
		}
	}
}

func TestExtractUnsupported(t *testing.T) {
	src := filepath.Join(t.TempDir(), "x.cbz")
	os.WriteFile(src, []byte("not an archive"), 0o644)
	if _, err := Extract(src, t.TempDir()); err != ErrUnsupported {
		t.Fatalf("got %v, want ErrUnsupported", err)
	}
}
