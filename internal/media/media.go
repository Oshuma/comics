// Package media manages page image files on disk.
//
// Files are laid out the same way Paperclip stored them in the original Rails
// app (pages/images/000/000/123/{original,thumb}/name.jpg), so an existing
// public/system directory can be used as the storage directory.
package media

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	"image/gif"
	"image/jpeg"
	"image/png"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/image/draw"
	_ "golang.org/x/image/webp"
	"golang.org/x/sync/singleflight"
)

const ThumbSize = 300

var allowedTypes = map[string]bool{
	"image/jpeg": true,
	"image/png":  true,
	"image/gif":  true,
	"image/webp": true,
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
// root, so imported files can be renamed into place instead of copied.
func (s *Storage) TempDir() (string, error) {
	base := filepath.Join(s.root, "tmp")
	if err := os.MkdirAll(base, 0o755); err != nil {
		return "", err
	}
	return os.MkdirTemp(base, "upload-")
}

func (s *Storage) pageDir(pageID int) string {
	id := fmt.Sprintf("%09d", pageID)
	return filepath.Join(s.root, "pages", "images", id[0:3], id[3:6], id[6:9])
}

func (s *Storage) OriginalPath(pageID int, fileName string) string {
	return filepath.Join(s.pageDir(pageID), "original", fileName)
}

func (s *Storage) thumbPath(pageID int, fileName string) string {
	return filepath.Join(s.pageDir(pageID), "thumb", fileName)
}

// DetectImageType returns the MIME type of the file if it's a supported image.
func DetectImageType(path string) (string, bool) {
	f, err := os.Open(path)
	if err != nil {
		return "", false
	}
	defer f.Close()

	buf := make([]byte, 512)
	n, _ := io.ReadFull(f, buf)
	ct := http.DetectContentType(buf[:n])
	return ct, allowedTypes[ct]
}

// SanitizeFileName mimics Paperclip's filename cleaning.
func SanitizeFileName(name string) string {
	name = filepath.Base(strings.ReplaceAll(name, "\\", "/"))
	return strings.Map(func(r rune) rune {
		if strings.ContainsRune(`&$+,/:;=?@<>[]{}|\^~%# `, r) || r < 32 {
			return '_'
		}
		return r
	}, name)
}

// Import moves src into place as the original image for a page.
func (s *Storage) Import(pageID int, fileName, src string) error {
	dst := s.OriginalPath(pageID, fileName)
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	if err := os.Rename(src, dst); err == nil {
		return nil
	}
	// Rename fails across filesystems; fall back to copying.
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return err
	}
	return out.Close()
}

// Thumb returns the thumbnail path for a page, generating it if needed.
func (s *Storage) Thumb(pageID int, fileName string) (string, error) {
	dst := s.thumbPath(pageID, fileName)
	if _, err := os.Stat(dst); err == nil {
		return dst, nil
	}

	_, err, _ := s.thumbs.Do(dst, func() (any, error) {
		if _, err := os.Stat(dst); err == nil {
			return nil, nil
		}
		return nil, generateThumb(s.OriginalPath(pageID, fileName), dst)
	})
	if err != nil {
		return "", err
	}
	return dst, nil
}

func (s *Storage) DeletePage(pageID int) error {
	err := os.RemoveAll(s.pageDir(pageID))
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}

func generateThumb(src, dst string) error {
	f, err := os.Open(src)
	if err != nil {
		return err
	}
	defer f.Close()

	img, format, err := image.Decode(f)
	if err != nil {
		return fmt.Errorf("decoding %s: %w", src, err)
	}

	// Equivalent of ImageMagick's "300x300>": only shrink, keep aspect ratio.
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
		dstImg := image.NewRGBA(image.Rect(0, 0, w, h))
		draw.CatmullRom.Scale(dstImg, dstImg.Bounds(), img, b, draw.Src, nil)
		img = dstImg
	}

	var buf bytes.Buffer
	switch format {
	case "png":
		err = png.Encode(&buf, img)
	case "gif":
		err = gif.Encode(&buf, img, nil)
	default:
		err = jpeg.Encode(&buf, img, &jpeg.Options{Quality: 85})
	}
	if err != nil {
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
