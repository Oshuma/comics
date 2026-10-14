// Package media manages page image files on disk.
//
// Each comic's pages live in their own directory:
//
//	comics/<comic id>/0001.jpg
//	comics/<comic id>/thumbs/0001.jpg
//
// Uploads waiting to be imported are kept in imports/, and scratch files in
// tmp/.
package media

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"golang.org/x/image/draw"
	_ "golang.org/x/image/webp"
	"golang.org/x/sync/singleflight"

	_ "image/gif"
	_ "image/png"

	"github.com/Oshuma/comics/internal/archive"
)

const ThumbSize = 300

// Supported image types and the extension pages are stored with.
var imageExts = map[string]string{
	"image/jpeg": ".jpg",
	"image/png":  ".png",
	"image/gif":  ".gif",
	"image/webp": ".webp",
}

type Storage struct {
	root   string
	thumbs singleflight.Group
}

func New(root string) (*Storage, error) {
	if err := os.MkdirAll(root, 0o755); err != nil {
		return nil, fmt.Errorf("creating storage dir: %w", err)
	}
	return &Storage{root: root}, nil
}

// TempDir creates a scratch directory on the same filesystem as the storage
// root, so staged comics can be renamed into place instead of copied.
func (s *Storage) TempDir() (string, error) {
	base := filepath.Join(s.root, "tmp")
	if err := os.MkdirAll(base, 0o755); err != nil {
		return "", err
	}
	return os.MkdirTemp(base, "upload-")
}

func (s *Storage) spoolRoot() string {
	return filepath.Join(s.root, "imports")
}

// SpoolDir creates a directory for an upload waiting to be imported. Unlike
// TempDir, it survives restarts.
func (s *Storage) SpoolDir() (string, error) {
	if err := os.MkdirAll(s.spoolRoot(), 0o755); err != nil {
		return "", err
	}
	return os.MkdirTemp(s.spoolRoot(), "upload-")
}

// RemoveSpool deletes the spool directory holding path.
func (s *Storage) RemoveSpool(path string) error {
	dir := filepath.Dir(path)
	if filepath.Dir(dir) != s.spoolRoot() {
		return fmt.Errorf("not a spool path: %s", path)
	}
	return os.RemoveAll(dir)
}

// CleanSpool removes spool directories that don't hold one of keep.
func (s *Storage) CleanSpool(keep map[string]bool) error {
	entries, err := os.ReadDir(s.spoolRoot())
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	needed := map[string]bool{}
	for p := range keep {
		needed[filepath.Dir(p)] = true
	}
	for _, e := range entries {
		dir := filepath.Join(s.spoolRoot(), e.Name())
		if !needed[dir] {
			if err := os.RemoveAll(dir); err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *Storage) comicDir(comicID int) string {
	return filepath.Join(s.root, "comics", strconv.Itoa(comicID))
}

func (s *Storage) PagePath(comicID int, file string) string {
	return filepath.Join(s.comicDir(comicID), filepath.Base(file))
}

func (s *Storage) thumbPath(comicID int, file string) string {
	name := strings.TrimSuffix(filepath.Base(file), filepath.Ext(file)) + ".jpg"
	return filepath.Join(s.comicDir(comicID), "thumbs", name)
}

// StagedPage is a page image ready to be installed.
type StagedPage struct {
	File        string
	ContentType string
	Size        int64
}

// Stage turns extracted archive entries in dir into numbered page files
// (0001.jpg, ...), deleting anything that isn't a supported image.
func Stage(dir string, entries []archive.Entry) ([]StagedPage, error) {
	var pages []StagedPage
	for _, e := range entries {
		contentType, ok := detectImageType(e.Path)
		if !ok {
			if err := os.Remove(e.Path); err != nil {
				return nil, err
			}
			continue
		}
		info, err := os.Stat(e.Path)
		if err != nil {
			return nil, err
		}
		file := fmt.Sprintf("%04d%s", len(pages)+1, imageExts[contentType])
		if err := os.Rename(e.Path, filepath.Join(dir, file)); err != nil {
			return nil, err
		}
		pages = append(pages, StagedPage{File: file, ContentType: contentType, Size: info.Size()})
	}
	return pages, nil
}

// InstallComic moves a staged directory into place as the comic's pages.
func (s *Storage) InstallComic(comicID int, stagedDir string) error {
	dst := s.comicDir(comicID)
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	return os.Rename(stagedDir, dst)
}

func (s *Storage) DeleteComic(comicID int) error {
	err := os.RemoveAll(s.comicDir(comicID))
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}

func detectImageType(path string) (string, bool) {
	f, err := os.Open(path)
	if err != nil {
		return "", false
	}
	defer f.Close()

	buf := make([]byte, 512)
	n, _ := io.ReadFull(f, buf)
	ct := http.DetectContentType(buf[:n])
	_, ok := imageExts[ct]
	return ct, ok
}

// Thumb returns the JPEG thumbnail path for a page, generating it if needed.
func (s *Storage) Thumb(comicID int, file string) (string, error) {
	dst := s.thumbPath(comicID, file)
	if _, err := os.Stat(dst); err == nil {
		return dst, nil
	}

	_, err, _ := s.thumbs.Do(dst, func() (any, error) {
		if _, err := os.Stat(dst); err == nil {
			return nil, nil
		}
		return nil, generateThumb(s.PagePath(comicID, file), dst)
	})
	if err != nil {
		return "", err
	}
	return dst, nil
}

func generateThumb(src, dst string) error {
	f, err := os.Open(src)
	if err != nil {
		return err
	}
	defer f.Close()

	img, _, err := image.Decode(f)
	if err != nil {
		return fmt.Errorf("decoding %s: %w", src, err)
	}

	// Only shrink, keeping aspect ratio.
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	if w > ThumbSize || h > ThumbSize {
		if w >= h {
			h = max(1, h*ThumbSize/w)
			w = ThumbSize
		} else {
			w = max(1, w*ThumbSize/h)
			h = ThumbSize
		}
	}
	// White background so transparent areas don't turn black in the JPEG.
	thumb := image.NewRGBA(image.Rect(0, 0, w, h))
	draw.Draw(thumb, thumb.Bounds(), image.NewUniform(color.White), image.Point{}, draw.Src)
	draw.CatmullRom.Scale(thumb, thumb.Bounds(), img, b, draw.Over, nil)

	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, thumb, &jpeg.Options{Quality: 85}); err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	tmp := dst + ".tmp"
	if err := os.WriteFile(tmp, buf.Bytes(), 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, dst)
}
