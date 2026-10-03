package server

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/netip"
	"time"

	"github.com/Oshuma/comics/internal/store"
)

const (
	sessionCookie = "comics_session"
	sessionTTL    = 24 * time.Hour
	rememberTTL   = 14 * 24 * time.Hour
)

type ctxKey struct{}

func currentUser(r *http.Request) *store.User {
	u, _ := r.Context().Value(ctxKey{}).(*store.User)
	return u
}

func (s *Server) sessionUser(r *http.Request) *store.User {
	c, err := r.Cookie(sessionCookie)
	if err != nil || c.Value == "" {
		return nil
	}
	u, err := s.store.SessionUser(r.Context(), c.Value)
	if err != nil {
		return nil
	}
	return u
}

func (s *Server) requireUser(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u := s.sessionUser(r)
		if u == nil {
			writeError(w, http.StatusUnauthorized, "You need to sign in before continuing.")
			return
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), ctxKey{}, u)))
	})
}

func (s *Server) requireAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !currentUser(r).Admin {
			writeError(w, http.StatusForbidden, "Admins only.")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) startSession(w http.ResponseWriter, r *http.Request, userID int, remember bool) error {
	ttl := sessionTTL
	if remember {
		ttl = rememberTTL
	}
	token, err := s.store.CreateSession(r.Context(), userID, ttl)
	if err != nil {
		return err
	}
	cookie := &http.Cookie{
		Name:     sessionCookie,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   s.cfg.SecureCookies,
		SameSite: http.SameSiteLaxMode,
	}
	if remember {
		cookie.MaxAge = int(ttl.Seconds())
	}
	http.SetCookie(w, cookie)
	return nil
}

func clientIP(r *http.Request) netip.Addr {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}
	addr, _ := netip.ParseAddr(host)
	return addr.Unmap()
}

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	count, err := s.store.CountUsers(r.Context())
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"setupRequired": count == 0,
		"user":          s.sessionUser(r),
	})
}

// handleSetup creates the initial admin user; only allowed when no users exist.
func (s *Server) handleSetup(w http.ResponseWriter, r *http.Request) {
	var req store.NewUser
	if !decodeJSON(w, r, &req) {
		return
	}

	s.setupMu.Lock()
	defer s.setupMu.Unlock()

	count, err := s.store.CountUsers(r.Context())
	if err != nil {
		writeStoreError(w, err)
		return
	}
	if count > 0 {
		writeError(w, http.StatusConflict, "Setup has already been completed.")
		return
	}

	req.Admin = true
	user, err := s.store.CreateUser(r.Context(), req)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	if err := s.startSession(w, r, user.ID, false); err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, user)
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
		Remember bool   `json:"remember"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}

	user, err := s.store.Authenticate(r.Context(), req.Email, req.Password, clientIP(r))
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusUnauthorized, "Invalid email or password.")
		return
	}
	if err != nil {
		writeStoreError(w, err)
		return
	}
	if err := s.startSession(w, r, user.ID, req.Remember); err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, user)
}

func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie(sessionCookie); err == nil {
		s.store.DeleteSession(r.Context(), c.Value)
	}
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookie,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   s.cfg.SecureCookies,
		SameSite: http.SameSiteLaxMode,
	})
	writeOK(w)
}
