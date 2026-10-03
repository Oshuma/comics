package server

import (
	"encoding/json"
	"errors"
	"io/fs"
	"log/slog"
	"net/http"
	"path"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Oshuma/comics/internal/config"
	"github.com/Oshuma/comics/internal/media"
	"github.com/Oshuma/comics/internal/store"
)

type Server struct {
	cfg    *config.Config
	store  *store.Store
	media  *media.Storage
	static fs.FS

	// Serializes initial admin setup.
	setupMu sync.Mutex
}

func New(cfg *config.Config, st *store.Store, m *media.Storage, static fs.FS) *Server {
	return &Server{cfg: cfg, store: st, media: m, static: static}
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /api/status", s.handleStatus)
	mux.HandleFunc("POST /api/setup", s.handleSetup)
	mux.HandleFunc("POST /api/session", s.handleLogin)
	mux.HandleFunc("DELETE /api/session", s.handleLogout)

	user := func(pattern string, h http.HandlerFunc) { mux.Handle(pattern, s.requireUser(h)) }
	admin := func(pattern string, h http.HandlerFunc) { mux.Handle(pattern, s.requireUser(s.requireAdmin(h))) }

	user("GET /api/groups", s.handleListGroups)
	user("POST /api/groups", s.handleCreateGroup)
	user("GET /api/groups/{id}", s.handleGetGroup)
	user("PATCH /api/groups/{id}", s.handleUpdateGroup)
	user("DELETE /api/groups/{id}", s.handleDeleteGroup)
	user("DELETE /api/groups/{id}/comics", s.handleDeleteGroupComics)
	user("DELETE /api/groups/{id}/comics/read", s.handleDeleteGroupReadComics)

	user("POST /api/comics", s.handleUploadComic)
	user("DELETE /api/comics/read", s.handleDeleteReadComics)
	user("GET /api/comics/{id}", s.handleGetComic)
	user("DELETE /api/comics/{id}", s.handleDeleteComic)
	user("PUT /api/comics/{id}/group", s.handleMoveComic)
	user("PUT /api/comics/{id}/finish", s.handleFinishComic)
	user("GET /api/comics/{id}/resume", s.handleResumeComic)
	user("PUT /api/comics/{id}/pages/{pageId}/current", s.handleViewPage)

	user("GET /api/pages/{id}/image", s.handlePageImage)
	user("GET /api/pages/{id}/thumb", s.handlePageThumb)

	user("GET /api/history", s.handleHistory)
	user("GET /api/stats", s.handleStats)

	admin("GET /api/admin/users", s.handleListUsers)
	admin("POST /api/admin/users", s.handleCreateUser)
	admin("PUT /api/admin/users/{id}/admin", s.handleSetAdmin)
	admin("DELETE /api/admin/users/{id}", s.handleDeleteUser)

	mux.HandleFunc("/api/", func(w http.ResponseWriter, r *http.Request) {
		writeError(w, http.StatusNotFound, "Not found")
	})
	mux.Handle("/", s.spaHandler())

	return logRequests(recoverPanics(csrfProtect(mux)))
}

// csrfProtect requires a custom header on state-changing API requests.
// Browsers can't send custom headers cross-origin without a CORS preflight,
// which this server never approves.
func csrfProtect(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet, http.MethodHead, http.MethodOptions:
		default:
			if strings.HasPrefix(r.URL.Path, "/api/") && r.Header.Get("X-Requested-With") == "" {
				writeError(w, http.StatusForbidden, "Missing X-Requested-With header")
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

func recoverPanics(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				if err == http.ErrAbortHandler {
					panic(err)
				}
				slog.Error("panic", "err", err, "path", r.URL.Path)
				writeError(w, http.StatusInternalServerError, "Internal server error")
			}
		}()
		next.ServeHTTP(w, r)
	})
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}

func (r *statusRecorder) Unwrap() http.ResponseWriter { return r.ResponseWriter }

func logRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rec, r)
		slog.Info("request", "method", r.Method, "path", r.URL.Path, "status", rec.status,
			"duration", time.Since(start).Round(time.Millisecond))
	})
}

// spaHandler serves the embedded frontend, falling back to index.html for
// client-side routes.
func (s *Server) spaHandler() http.Handler {
	fileServer := http.FileServerFS(s.static)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		name := strings.TrimPrefix(path.Clean(r.URL.Path), "/")
		if name != "" {
			if info, err := fs.Stat(s.static, name); err == nil && !info.IsDir() {
				if strings.HasPrefix(name, "assets/") {
					w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
				}
				fileServer.ServeHTTP(w, r)
				return
			}
		}

		index, err := fs.ReadFile(s.static, "index.html")
		if err != nil {
			http.Error(w, "Frontend not built. Run `make build`.", http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "no-cache")
		w.Write(index)
	})
}

// JSON helpers

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]any{"error": msg})
}

func writeOK(w http.ResponseWriter) {
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

// writeStoreError maps store errors to HTTP responses.
func writeStoreError(w http.ResponseWriter, err error) {
	var verr *store.ValidationError
	switch {
	case errors.As(err, &verr):
		writeJSON(w, http.StatusUnprocessableEntity, map[string]any{"error": verr.Error(), "messages": verr.Messages})
	case errors.Is(err, store.ErrNotFound):
		writeError(w, http.StatusNotFound, "Not found")
	default:
		slog.Error("request failed", "err", err)
		writeError(w, http.StatusInternalServerError, "Internal server error")
	}
}

func decodeJSON(w http.ResponseWriter, r *http.Request, v any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	if err := json.NewDecoder(r.Body).Decode(v); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON body")
		return false
	}
	return true
}

func pathID(w http.ResponseWriter, r *http.Request, name string) (int, bool) {
	id, err := strconv.Atoi(r.PathValue(name))
	if err != nil || id <= 0 {
		writeError(w, http.StatusNotFound, "Not found")
		return 0, false
	}
	return id, true
}

func pageParam(r *http.Request) int {
	p, err := strconv.Atoi(r.URL.Query().Get("page"))
	if err != nil || p < 1 {
		return 1
	}
	return p
}

type paginated[T any] struct {
	Items   []T `json:"items"`
	Total   int `json:"total"`
	Page    int `json:"page"`
	PerPage int `json:"perPage"`
}

func newPaginated[T any](items []T, total, page int) paginated[T] {
	if items == nil {
		items = []T{}
	}
	return paginated[T]{Items: items, Total: total, Page: page, PerPage: store.PerPage}
}
