package server

import (
	"context"
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/Oshuma/comics/internal/store"
)

// handleUploadComic accepts a multipart upload with a "comic" archive and
// either "group_id" or "group_name" (which come before the file part).
// The archive is streamed to disk and queued for a background import.
func (s *Server) handleUploadComic(w http.ResponseWriter, r *http.Request) {
	userID := currentUser(r).ID
	r.Body = http.MaxBytesReader(w, r.Body, s.cfg.MaxUploadMB<<20)

	mr, err := r.MultipartReader()
	if err != nil {
		writeError(w, http.StatusBadRequest, "Expected a multipart upload")
		return
	}

	spoolDir, err := s.media.SpoolDir()
	if err != nil {
		writeStoreError(w, err)
		return
	}
	queued := false
	defer func() {
		if !queued {
			os.RemoveAll(spoolDir)
		}
	}()

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
			archivePath = filepath.Join(spoolDir, "archive")
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

	groupID, err = s.resolveGroup(r.Context(), userID, groupID, groupName)
	if err != nil {
		writeGroupError(w, err)
		return
	}
	job, err := s.store.CreateImport(r.Context(), store.NewImport{
		UserID: userID, GroupID: groupID, Filename: filename, Source: store.SourceUpload, UploadPath: archivePath,
	})
	if err != nil {
		writeStoreError(w, err)
		return
	}
	queued = true
	s.importer.Notify()
	writeJSON(w, http.StatusAccepted, job)
}

var errNoGroup = errors.New("no group")

// resolveGroup returns groupID, or the user's group named groupName
// (creating it if needed) when groupID is zero.
func (s *Server) resolveGroup(ctx context.Context, userID, groupID int, groupName string) (int, error) {
	if groupID != 0 {
		if _, err := s.store.GetGroup(ctx, userID, groupID); err != nil {
			return 0, err
		}
		return groupID, nil
	}
	if strings.TrimSpace(groupName) == "" {
		return 0, errNoGroup
	}
	return s.store.FindOrCreateGroup(ctx, userID, groupName)
}

func writeGroupError(w http.ResponseWriter, err error) {
	if errors.Is(err, errNoGroup) {
		writeError(w, http.StatusUnprocessableEntity, "Please select a group.")
		return
	}
	writeStoreError(w, err)
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
