// Package archive extracts comic book archives (CBZ/CBR).
package archive

import (
	"archive/zip"
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/nwaples/rardecode/v2"
)

var ErrUnsupported = errors.New("unsupported archive format (expected CBZ or CBR)")

// Entry is a file extracted from an archive.
type Entry struct {
	// Name is the path of the file inside the archive.
	Name string
	// Path is where the file was extracted to.
	Path string
}

var (
	zipMagic = []byte("PK\x03\x04")
	rarMagic = []byte("Rar!\x1a\x07")
)

// Extract unpacks every regular file in the archive at src into destDir and
// returns the entries in natural sort order of their names. The format is
// detected from the file contents, since .cbr files are often zips.
func Extract(src, destDir string) ([]Entry, error) {
	f, err := os.Open(src)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	header := make([]byte, 8)
	n, _ := io.ReadFull(f, header)
	header = header[:n]
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		return nil, err
	}

	var entries []Entry
	switch {
	case bytes.HasPrefix(header, zipMagic):
		entries, err = extractZip(f, destDir)
	case bytes.HasPrefix(header, rarMagic):
		entries, err = extractRar(f, destDir)
	default:
		return nil, ErrUnsupported
	}
	if err != nil {
		return nil, err
	}

	sort.SliceStable(entries, func(i, j int) bool {
		return naturalLess(strings.ToLower(entries[i].Name), strings.ToLower(entries[j].Name))
	})
	return entries, nil
}

func extractZip(f *os.File, destDir string) ([]Entry, error) {
	info, err := f.Stat()
	if err != nil {
		return nil, err
	}
	zr, err := zip.NewReader(f, info.Size())
	if err != nil {
		return nil, fmt.Errorf("reading zip: %w", err)
	}

	var entries []Entry
	for _, zf := range zr.File {
		if zf.FileInfo().IsDir() || skipName(zf.Name) {
			continue
		}
		rc, err := zf.Open()
		if err != nil {
			return nil, fmt.Errorf("opening %s: %w", zf.Name, err)
		}
		entry, err := writeEntry(destDir, len(entries), zf.Name, rc)
		rc.Close()
		if err != nil {
			return nil, err
		}
		entries = append(entries, entry)
	}
	return entries, nil
}

func extractRar(f *os.File, destDir string) ([]Entry, error) {
	rr, err := rardecode.NewReader(f)
	if err != nil {
		return nil, fmt.Errorf("reading rar: %w", err)
	}

	var entries []Entry
	for {
		hdr, err := rr.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("reading rar: %w", err)
		}
		if hdr.IsDir || skipName(hdr.Name) {
			continue
		}
		entry, err := writeEntry(destDir, len(entries), hdr.Name, rr)
		if err != nil {
			return nil, err
		}
		entries = append(entries, entry)
	}
	return entries, nil
}

// writeEntry writes to a numbered file so archive paths never touch the
// filesystem (avoids path traversal via crafted names).
func writeEntry(destDir string, index int, name string, r io.Reader) (Entry, error) {
	path := filepath.Join(destDir, fmt.Sprintf("%06d", index))
	out, err := os.Create(path)
	if err != nil {
		return Entry{}, err
	}
	if _, err := io.Copy(out, r); err != nil {
		out.Close()
		return Entry{}, fmt.Errorf("extracting %s: %w", name, err)
	}
	if err := out.Close(); err != nil {
		return Entry{}, err
	}
	return Entry{Name: strings.ReplaceAll(name, "\\", "/"), Path: path}, nil
}

// skipName filters out OS metadata files commonly found in archives.
func skipName(name string) bool {
	name = strings.ReplaceAll(name, "\\", "/")
	base := filepath.Base(name)
	return strings.HasPrefix(name, "__MACOSX/") ||
		strings.HasPrefix(base, "._") ||
		strings.EqualFold(base, "Thumbs.db") ||
		strings.EqualFold(base, ".DS_Store")
}

// naturalLess compares strings treating digit runs as numbers, so
// "page2" sorts before "page10".
func naturalLess(a, b string) bool {
	for a != "" && b != "" {
		ad, bd := isDigit(a[0]), isDigit(b[0])
		if ad && bd {
			na, ra := splitDigits(a)
			nb, rb := splitDigits(b)
			ta, tb := strings.TrimLeft(na, "0"), strings.TrimLeft(nb, "0")
			if len(ta) != len(tb) {
				return len(ta) < len(tb)
			}
			if ta != tb {
				return ta < tb
			}
			if len(na) != len(nb) {
				return len(na) < len(nb)
			}
			a, b = ra, rb
			continue
		}
		if a[0] != b[0] {
			return a[0] < b[0]
		}
		a, b = a[1:], b[1:]
	}
	return len(a) < len(b)
}

func isDigit(c byte) bool { return c >= '0' && c <= '9' }

func splitDigits(s string) (string, string) {
	i := 0
	for i < len(s) && isDigit(s[i]) {
		i++
	}
	return s[:i], s[i:]
}
