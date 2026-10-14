package server

import (
	"errors"
	"log/slog"
	"net/http"
	"path"

	"github.com/Oshuma/comics/internal/archiveorg"
	"github.com/Oshuma/comics/internal/importer"
	"github.com/Oshuma/comics/internal/store"
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

// handleArchiveImport queues a CBR/CBZ file on archive.org to be downloaded
// and added to a group in the background.
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

	job, err := s.store.CreateImport(r.Context(), store.NewImport{
		UserID: userID, GroupID: groupID, Filename: path.Base(file.Name), Source: store.SourceArchive,
		ArchiveIdentifier: file.Identifier, ArchiveName: file.Name,
	})
	if err != nil {
		writeStoreError(w, err)
		return
	}
	s.importer.Notify()
	writeJSON(w, http.StatusAccepted, job)
}

type importView struct {
	store.Import
	// Set while the job is being processed.
	Progress *importer.Progress `json:"progress,omitempty"`
}

func (s *Server) handleListImports(w http.ResponseWriter, r *http.Request) {
	jobs, err := s.store.ListImports(r.Context(), currentUser(r).ID)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	views := make([]importView, len(jobs))
	for i, j := range jobs {
		views[i].Import = j
		if p, ok := s.importer.Progress(j.ID); ok && j.Status == store.ImportProcessing {
			views[i].Progress = &p
		}
	}
	writeJSON(w, http.StatusOK, views)
}

// handleDeleteImport cancels a queued or running job, or dismisses a
// finished one.
func (s *Server) handleDeleteImport(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r, "id")
	if !ok {
		return
	}
	job, err := s.store.DeleteImport(r.Context(), currentUser(r).ID, id)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	s.importer.Cancel(job.ID)
	if job.UploadPath != "" {
		if err := s.media.RemoveSpool(job.UploadPath); err != nil {
			slog.Warn("removing spooled upload", "import", job.ID, "err", err)
		}
	}
	writeOK(w)
}
