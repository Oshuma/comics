package server

import (
	"errors"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/Oshuma/comics/internal/archive"
)

// handleUploadComic accepts a multipart upload with a "comic" archive and
// either "group_id" or "group_name" (which come before the file part).
// The archive is streamed to disk rather than buffered in memory.
func (s *Server) handleUploadComic(w http.ResponseWriter, r *http.Request) {
	userID := currentUser(r).ID
	r.Body = http.MaxBytesReader(w, r.Body, s.cfg.MaxUploadMB<<20)

	mr, err := r.MultipartReader()
	if err != nil {
		writeError(w, http.StatusBadRequest, "Expected a multipart upload")
		return
	}

	tmpDir, err := s.media.TempDir()
	if err != nil {
		writeStoreError(w, err)
		return
	}
	defer os.RemoveAll(tmpDir)

	var groupID int
	var groupName, filename, archivePath string

	for {
		part, err := mr.NextPart()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			writeUploadError(w, err)
			return
		}

		switch part.FormName() {
		case "group_id":
			v, _ := readSmallPart(part)
			groupID, _ = strconv.Atoi(strings.TrimSpace(v))
		case "group_name":
			groupName, _ = readSmallPart(part)
		case "comic":
			filename = filepath.Base(part.FileName())
			archivePath = filepath.Join(tmpDir, "archive")
			if err := saveUpload(part, archivePath); err != nil {
				writeUploadError(w, err)
				return
			}
		}
		part.Close()
	}

	if archivePath == "" || filename == "" || filename == "." {
		writeError(w, http.StatusBadRequest, "No comic file was uploaded.")
		return
	}

	if groupID == 0 {
		if strings.TrimSpace(groupName) == "" {
			writeError(w, http.StatusUnprocessableEntity, "Please select a group.")
			return
		}
		if groupID, err = s.store.FindOrCreateGroup(r.Context(), userID, groupName); err != nil {
			writeStoreError(w, err)
			return
		}
	}

	extractDir := filepath.Join(tmpDir, "pages")
	if err := os.Mkdir(extractDir, 0o755); err != nil {
		writeStoreError(w, err)
		return
	}
	entries, err := archive.Extract(archivePath, extractDir)
	if err != nil {
		slog.Warn("extracting upload", "file", filename, "err", err)
		msg := "Could not read archive: " + err.Error()
		if errors.Is(err, archive.ErrUnsupported) {
			msg = "Unsupported file type. Upload a CBZ or CBR archive."
		}
		writeError(w, http.StatusUnprocessableEntity, msg)
		return
	}
	// The archive itself is no longer needed; free the space early.
	os.Remove(archivePath)

	comic, err := s.store.CreateComic(r.Context(), userID, groupID, filename, entries)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, comic)
}

func readSmallPart(p *multipart.Part) (string, error) {
	b, err := io.ReadAll(io.LimitReader(p, 4096))
	return string(b), err
}

func saveUpload(r io.Reader, path string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	if _, err := io.Copy(f, r); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}

func writeUploadError(w http.ResponseWriter, err error) {
	var maxErr *http.MaxBytesError
	if errors.As(err, &maxErr) {
		writeError(w, http.StatusRequestEntityTooLarge, "File is too large.")
		return
	}
	writeError(w, http.StatusBadRequest, "Upload failed: "+err.Error())
}
