package server

import (
	"net/http"
	"os"

	"github.com/Oshuma/comics/internal/store"
)

// Groups

func (s *Server) handleListGroups(w http.ResponseWriter, r *http.Request) {
	groups, err := s.store.ListGroups(r.Context(), currentUser(r).ID)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	if groups == nil {
		groups = []store.Group{}
	}
	writeJSON(w, http.StatusOK, groups)
}

func (s *Server) handleGetGroup(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r, "id")
	if !ok {
		return
	}
	userID := currentUser(r).ID
	group, err := s.store.GetGroup(r.Context(), userID, id)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	comics, err := s.store.ListComics(r.Context(), userID, id)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	if comics == nil {
		comics = []store.Comic{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"group": group, "comics": comics})
}

type groupRequest struct {
	Name string `json:"name"`
}

func (s *Server) handleCreateGroup(w http.ResponseWriter, r *http.Request) {
	var req groupRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	group, err := s.store.CreateGroup(r.Context(), currentUser(r).ID, req.Name)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, group)
}

func (s *Server) handleUpdateGroup(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r, "id")
	if !ok {
		return
	}
	var req groupRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	if err := s.store.RenameGroup(r.Context(), currentUser(r).ID, id, req.Name); err != nil {
		writeStoreError(w, err)
		return
	}
	writeOK(w)
}

func (s *Server) handleDeleteGroup(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r, "id")
	if !ok {
		return
	}
	if err := s.store.DeleteGroup(r.Context(), currentUser(r).ID, id); err != nil {
		writeStoreError(w, err)
		return
	}
	writeOK(w)
}

func (s *Server) handleDeleteGroupComics(w http.ResponseWriter, r *http.Request) {
	s.deleteGroupComics(w, r, false)
}

func (s *Server) handleDeleteGroupReadComics(w http.ResponseWriter, r *http.Request) {
	s.deleteGroupComics(w, r, true)
}

func (s *Server) deleteGroupComics(w http.ResponseWriter, r *http.Request, onlyRead bool) {
	id, ok := pathID(w, r, "id")
	if !ok {
		return
	}
	if err := s.store.DeleteGroupComics(r.Context(), currentUser(r).ID, id, onlyRead); err != nil {
		writeStoreError(w, err)
		return
	}
	writeOK(w)
}

// Comics

func (s *Server) handleGetComic(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r, "id")
	if !ok {
		return
	}
	comic, err := s.store.GetComic(r.Context(), currentUser(r).ID, id)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, comic)
}

func (s *Server) handleDeleteComic(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r, "id")
	if !ok {
		return
	}
	if err := s.store.DeleteComic(r.Context(), currentUser(r).ID, id); err != nil {
		writeStoreError(w, err)
		return
	}
	writeOK(w)
}

func (s *Server) handleDeleteReadComics(w http.ResponseWriter, r *http.Request) {
	if err := s.store.DeleteReadComics(r.Context(), currentUser(r).ID); err != nil {
		writeStoreError(w, err)
		return
	}
	writeOK(w)
}

func (s *Server) handleMoveComic(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r, "id")
	if !ok {
		return
	}
	var req struct {
		GroupID int `json:"groupId"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	if err := s.store.MoveComic(r.Context(), currentUser(r).ID, id, req.GroupID); err != nil {
		writeStoreError(w, err)
		return
	}
	writeOK(w)
}

// handleFinishComic marks the comic read and tells the client where to go next.
func (s *Server) handleFinishComic(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r, "id")
	if !ok {
		return
	}
	userID := currentUser(r).ID
	if err := s.store.FinishComic(r.Context(), userID, id); err != nil {
		writeStoreError(w, err)
		return
	}
	comic, err := s.store.GetComic(r.Context(), userID, id)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	next, err := s.store.NextComic(r.Context(), userID, id)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"groupId": comic.GroupID, "nextComicId": next})
}

// handleResumeComic returns the first unread page (null if the comic is read).
func (s *Server) handleResumeComic(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r, "id")
	if !ok {
		return
	}
	pageID, err := s.store.FirstUnreadPage(r.Context(), currentUser(r).ID, id)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"pageId": pageID})
}

func (s *Server) handleViewPage(w http.ResponseWriter, r *http.Request) {
	comicID, ok := pathID(w, r, "id")
	if !ok {
		return
	}
	pageID, ok := pathID(w, r, "pageId")
	if !ok {
		return
	}
	view, err := s.store.ViewPage(r.Context(), currentUser(r).ID, comicID, pageID)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, view)
}

// Page images

func (s *Server) handlePageImage(w http.ResponseWriter, r *http.Request) {
	s.servePage(w, r, false)
}

func (s *Server) handlePageThumb(w http.ResponseWriter, r *http.Request) {
	s.servePage(w, r, true)
}

func (s *Server) servePage(w http.ResponseWriter, r *http.Request, thumb bool) {
	id, ok := pathID(w, r, "id")
	if !ok {
		return
	}
	img, err := s.store.GetPageImage(r.Context(), currentUser(r).ID, id)
	if err != nil {
		writeStoreError(w, err)
		return
	}

	path := s.media.OriginalPath(img.ID, img.FileName)
	if thumb {
		if path, err = s.media.Thumb(img.ID, img.FileName); err != nil {
			writeError(w, http.StatusNotFound, "Image not available")
			return
		}
	}

	f, err := os.Open(path)
	if err != nil {
		writeError(w, http.StatusNotFound, "Image not available")
		return
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		writeStoreError(w, err)
		return
	}

	// Sniff the type rather than trusting the extension.
	buf := make([]byte, 512)
	n, _ := f.Read(buf)
	if _, err := f.Seek(0, 0); err != nil {
		writeStoreError(w, err)
		return
	}
	w.Header().Set("Content-Type", http.DetectContentType(buf[:n]))
	// Page IDs are never reused, so images can be cached indefinitely.
	w.Header().Set("Cache-Control", "private, max-age=31536000, immutable")
	http.ServeContent(w, r, "", info.ModTime(), f)
}

// History & stats

func (s *Server) handleHistory(w http.ResponseWriter, r *http.Request) {
	page := pageParam(r)
	items, total, err := s.store.ListHistory(r.Context(), currentUser(r).ID, page)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, newPaginated(items, total, page))
}

func (s *Server) handleStats(w http.ResponseWriter, r *http.Request) {
	groups, err := s.store.ListGroups(r.Context(), currentUser(r).ID)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	var total int64
	for _, g := range groups {
		total += g.DiskSize
	}
	if groups == nil {
		groups = []store.Group{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"totalSize": total, "groups": groups})
}

// Admin

func (s *Server) handleListUsers(w http.ResponseWriter, r *http.Request) {
	page := pageParam(r)
	users, total, err := s.store.ListUsers(r.Context(), page)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, newPaginated(users, total, page))
}

func (s *Server) handleCreateUser(w http.ResponseWriter, r *http.Request) {
	var req store.NewUser
	if !decodeJSON(w, r, &req) {
		return
	}
	user, err := s.store.CreateUser(r.Context(), req)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, user)
}

func (s *Server) handleSetAdmin(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r, "id")
	if !ok {
		return
	}
	var req struct {
		Admin bool `json:"admin"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	if !req.Admin && id == currentUser(r).ID {
		writeError(w, http.StatusUnprocessableEntity, "Can't remove yourself as admin.")
		return
	}
	if err := s.store.SetAdmin(r.Context(), id, req.Admin); err != nil {
		writeStoreError(w, err)
		return
	}
	writeOK(w)
}

func (s *Server) handleDeleteUser(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r, "id")
	if !ok {
		return
	}
	if id == currentUser(r).ID {
		writeError(w, http.StatusUnprocessableEntity, "Can't delete your own account.")
		return
	}
	if err := s.store.DeleteUser(r.Context(), id); err != nil {
		writeStoreError(w, err)
		return
	}
	writeOK(w)
}
