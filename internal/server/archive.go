package server

import (
	"errors"
	"log/slog"
	"net/http"
	"os"
	"path"
	"path/filepath"

	"github.com/Oshuma/comics/internal/archiveorg"
)

type archiveSearchResponse struct {
	*archiveorg.Results
	// Files larger than this can't be imported.
	MaxBytes int64 `json:"maxBytes"`
}

func (s *Server) handleArchiveSearch(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	if q == "" {
		writeError(w, http.StatusBadRequest, "Enter something to search for.")
		return
	}
	results, err := s.archive.Search(r.Context(), q, pageParam(r))
	if err != nil {
		slog.Warn("archive.org search", "q", q, "err", err)
		writeError(w, http.StatusBadGateway, "Searching archive.org failed. Try again in a moment.")
		return
	}
	writeJSON(w, http.StatusOK, archiveSearchResponse{Results: results, MaxBytes: s.cfg.MaxUploadMB << 20})
}

// handleArchiveImport downloads a CBR/CBZ file from archive.org and adds it
// to a group, the same way as an upload.
func (s *Server) handleArchiveImport(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Identifier string `json:"identifier"`
		Name       string `json:"name"`
		GroupID    int    `json:"groupId"`
		GroupName  string `json:"groupName"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	userID := currentUser(r).ID

	// Check the group before downloading anything.
	groupID, err := s.resolveGroup(r.Context(), userID, req.GroupID, req.GroupName)
	if err != nil {
		writeGroupError(w, err)
		return
	}

	// Only files listed in the item can be imported, so arbitrary URLs
	// can't be fetched.
	file, err := s.archive.ItemFile(r.Context(), req.Identifier, req.Name)
	if errors.Is(err, archiveorg.ErrNotFound) {
		writeError(w, http.StatusNotFound, "That file isn't available on archive.org.")
		return
	}
	if err != nil {
		slog.Warn("archive.org metadata", "identifier", req.Identifier, "err", err)
		writeError(w, http.StatusBadGateway, "Couldn't reach archive.org. Try again in a moment.")
		return
	}
	maxBytes := s.cfg.MaxUploadMB << 20
	if file.Size > maxBytes {
		writeError(w, http.StatusRequestEntityTooLarge, "File is too large.")
		return
	}

	tmpDir, err := s.media.TempDir()
	if err != nil {
		writeStoreError(w, err)
		return
	}
	defer os.RemoveAll(tmpDir)

	archivePath := filepath.Join(tmpDir, "archive")
	out, err := os.Create(archivePath)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	err = s.archive.Download(r.Context(), file, out, maxBytes)
	if cerr := out.Close(); err == nil {
		err = cerr
	}
	if errors.Is(err, archiveorg.ErrTooLarge) {
		writeError(w, http.StatusRequestEntityTooLarge, "File is too large.")
		return
	}
	if err != nil {
		slog.Warn("archive.org download", "identifier", file.Identifier, "file", file.Name, "err", err)
		writeError(w, http.StatusBadGateway, "Downloading from archive.org failed. Try again in a moment.")
		return
	}

	s.importArchive(w, r, userID, groupID, path.Base(file.Name), tmpDir, archivePath)
}
