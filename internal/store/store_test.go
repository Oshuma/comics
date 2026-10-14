package store

import (
	"archive/zip"
	"context"
	"errors"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/Oshuma/comics/internal/archive"
	"github.com/Oshuma/comics/internal/database"
	"github.com/Oshuma/comics/internal/media"
)

func newTestStore(t *testing.T) (*Store, *media.Storage, string) {
	t.Helper()
	dir := t.TempDir()
	ctx := context.Background()
	db, err := database.Open(ctx, filepath.Join(dir, "comics.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	if err := database.Migrate(ctx, db); err != nil {
		t.Fatal(err)
	}
	m, err := media.New(filepath.Join(dir, "storage"))
	if err != nil {
		t.Fatal(err)
	}
	return New(db, m), m, filepath.Join(dir, "storage")
}

func createUser(t *testing.T, s *Store, email string) *User {
	t.Helper()
	u, err := s.CreateUser(context.Background(), NewUser{Email: email, Password: "password1", PasswordConfirmation: "password1"})
	if err != nil {
		t.Fatal(err)
	}
	return u
}

// createComic builds a CBZ with the given number of pages plus a non-image
// file, and imports it through the same path as an upload.
func createComic(t *testing.T, s *Store, m *media.Storage, userID, groupID int, filename string, pages int) *Comic {
	t.Helper()
	tmp, err := m.TempDir()
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmp)

	archivePath := filepath.Join(tmp, "archive")
	f, err := os.Create(archivePath)
	if err != nil {
		t.Fatal(err)
	}
	zw := zip.NewWriter(f)
	for i := range pages {
		w, _ := zw.Create(filepath.Join("pages", string(rune('a'+i))+".png"))
		png.Encode(w, image.NewRGBA(image.Rect(0, 0, 400, 600)))
	}
	w, _ := zw.Create("info.txt")
	w.Write([]byte("not an image"))
	zw.Close()
	f.Close()

	dir := filepath.Join(tmp, "pages")
	os.Mkdir(dir, 0o755)
	entries, err := archive.Extract(archivePath, dir, nil)
	if err != nil {
		t.Fatal(err)
	}
	staged, err := media.Stage(dir, entries)
	if err != nil {
		t.Fatal(err)
	}
	c, err := s.CreateComic(context.Background(), userID, groupID, filename, dir, staged)
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func TestUsersAndSessions(t *testing.T) {
	s, _, _ := newTestStore(t)
	ctx := context.Background()

	u := createUser(t, s, " Alice@Example.com ")
	if u.Email != "alice@example.com" || u.CreatedAt.IsZero() {
		t.Fatalf("unexpected user: %+v", u)
	}

	var verr *ValidationError
	if _, err := s.CreateUser(ctx, NewUser{Email: "alice@example.com", Password: "password1", PasswordConfirmation: "password1"}); !errors.As(err, &verr) {
		t.Fatalf("expected duplicate email validation error, got %v", err)
	}

	if _, err := s.Authenticate(ctx, "alice@example.com", "wrong"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected bad password to fail, got %v", err)
	}
	if _, err := s.Authenticate(ctx, "nobody@example.com", "password1"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected unknown email to fail, got %v", err)
	}
	if _, err := s.Authenticate(ctx, "ALICE@example.com", "password1"); err != nil {
		t.Fatal(err)
	}

	token, err := s.CreateSession(ctx, u.ID, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	got, err := s.SessionUser(ctx, token)
	if err != nil || got.ID != u.ID || got.CreatedAt.IsZero() {
		t.Fatalf("session lookup: %+v, %v", got, err)
	}
	expired, err := s.CreateSession(ctx, u.ID, -time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.SessionUser(ctx, expired); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected expired session to be rejected, got %v", err)
	}
	s.DeleteSession(ctx, token)
	if _, err := s.SessionUser(ctx, token); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected deleted session to be rejected, got %v", err)
	}

	if err := s.SetAdmin(ctx, u.ID, true); err != nil {
		t.Fatal(err)
	}
	if err := s.SetAdmin(ctx, 999, true); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected not found, got %v", err)
	}
	users, total, err := s.ListUsers(ctx, 1)
	if err != nil || total != 1 || !users[0].Admin {
		t.Fatalf("list users: %+v %d %v", users, total, err)
	}
}

func TestGroups(t *testing.T) {
	s, _, _ := newTestStore(t)
	ctx := context.Background()
	u := createUser(t, s, "a@example.com")
	other := createUser(t, s, "b@example.com")

	g, err := s.CreateGroup(ctx, u.ID, "Batman")
	if err != nil {
		t.Fatal(err)
	}
	var verr *ValidationError
	if _, err := s.CreateGroup(ctx, u.ID, "batman"); !errors.As(err, &verr) {
		t.Fatalf("expected duplicate name error, got %v", err)
	}
	if _, err := s.CreateGroup(ctx, other.ID, "Batman"); err != nil {
		t.Fatalf("names should be unique per user: %v", err)
	}

	id, err := s.FindOrCreateGroup(ctx, u.ID, "BATMAN")
	if err != nil || id != g.ID {
		t.Fatalf("expected existing group %d, got %d (%v)", g.ID, id, err)
	}
	id, err = s.FindOrCreateGroup(ctx, u.ID, "Superman")
	if err != nil || id == g.ID {
		t.Fatalf("expected new group, got %d (%v)", id, err)
	}

	if err := s.RenameGroup(ctx, u.ID, g.ID, "superman"); !errors.As(err, &verr) {
		t.Fatalf("expected rename conflict, got %v", err)
	}
	if err := s.RenameGroup(ctx, other.ID, g.ID, "Mine"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected not found for other user, got %v", err)
	}

	groups, err := s.ListGroups(ctx, u.ID)
	if err != nil || len(groups) != 2 || groups[0].Name != "Batman" || !groups[0].Read {
		t.Fatalf("list groups: %+v %v", groups, err)
	}
}

func TestComicLifecycle(t *testing.T) {
	s, m, root := newTestStore(t)
	ctx := context.Background()
	u := createUser(t, s, "a@example.com")
	g, _ := s.CreateGroup(ctx, u.ID, "Group")

	c2 := createComic(t, s, m, u.ID, g.ID, "issue 2.cbz", 2)
	c1 := createComic(t, s, m, u.ID, g.ID, "Issue 1.cbz", 3)
	if c1.PageCount != 3 || c1.CoverPageID == nil {
		t.Fatalf("unexpected comic: %+v", c1)
	}
	if _, err := os.Stat(filepath.Join(root, "comics", strconv.Itoa(c1.ID), "0003.png")); err != nil {
		t.Fatalf("page file not installed: %v", err)
	}
	if entries, _ := os.ReadDir(filepath.Join(root, "tmp")); len(entries) != 0 {
		t.Fatalf("staging dirs left behind: %v", entries)
	}

	comics, err := s.ListComics(ctx, u.ID, g.ID)
	if err != nil || len(comics) != 2 || comics[0].ID != c1.ID {
		t.Fatalf("expected case-insensitive ordering: %+v %v", comics, err)
	}
	next, err := s.NextComic(ctx, u.ID, c1.ID)
	if err != nil || next == nil || *next != c2.ID {
		t.Fatalf("next comic: %v %v", next, err)
	}

	d, err := s.GetComic(ctx, u.ID, c1.ID)
	if err != nil || len(d.Pages) != 3 || d.GroupName != "Group" {
		t.Fatalf("get comic: %+v %v", d, err)
	}

	v, err := s.ViewPage(ctx, u.ID, c1.ID, d.Pages[1].ID)
	if err != nil || v.Number != 2 || *v.PrevPageID != d.Pages[0].ID || *v.NextPageID != d.Pages[2].ID {
		t.Fatalf("view page: %+v %v", v, err)
	}
	d, _ = s.GetComic(ctx, u.ID, c1.ID)
	if !d.Pages[0].Read || d.Pages[1].Read || !d.Reading || d.Read {
		t.Fatalf("unexpected read state: %+v", d)
	}
	first, _ := s.FirstUnreadPage(ctx, u.ID, c1.ID)
	if first == nil || *first != d.Pages[1].ID {
		t.Fatalf("first unread: %v", first)
	}

	img, err := s.GetPageImage(ctx, u.ID, d.Pages[0].ID)
	if err != nil || img.ContentType != "image/png" {
		t.Fatalf("page image: %+v %v", img, err)
	}
	thumb, err := m.Thumb(img.ComicID, img.File)
	if err != nil {
		t.Fatal(err)
	}
	if f, err := os.Open(thumb); err != nil {
		t.Fatal(err)
	} else {
		cfg, format, err := image.DecodeConfig(f)
		f.Close()
		if err != nil || format != "jpeg" || cfg.Height != media.ThumbSize || cfg.Width != 200 {
			t.Fatalf("thumb: %+v %s %v", cfg, format, err)
		}
	}

	if err := s.FinishComic(ctx, u.ID, c1.ID); err != nil {
		t.Fatal(err)
	}
	if err := s.FinishComic(ctx, u.ID, c1.ID); err != nil {
		t.Fatal(err)
	}
	history, total, err := s.ListHistory(ctx, u.ID, 1)
	if err != nil || total != 1 || history[0].ComicID == nil || *history[0].ComicID != c1.ID || *history[0].GroupID != g.ID {
		t.Fatalf("history: %+v %d %v", history, total, err)
	}

	group, _ := s.GetGroup(ctx, u.ID, g.ID)
	if group.ComicCount != 2 || group.DiskSize == 0 || group.Read || !group.Reading || *group.CoverPageID != *c1.CoverPageID {
		t.Fatalf("group aggregates: %+v", group)
	}

	if err := s.DeleteReadComics(ctx, u.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := s.GetComic(ctx, u.ID, c1.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected read comic deleted, got %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "comics", strconv.Itoa(c1.ID))); !os.IsNotExist(err) {
		t.Fatalf("comic files not removed: %v", err)
	}
	if _, err := s.GetComic(ctx, u.ID, c2.ID); err != nil {
		t.Fatalf("unread comic should remain: %v", err)
	}

	// IDs must not be reused, since images are cached by page ID.
	c3 := createComic(t, s, m, u.ID, g.ID, "Issue 3.cbz", 1)
	if c3.ID <= c1.ID || *c3.CoverPageID <= *c1.CoverPageID+2 {
		t.Fatalf("IDs reused: %+v after %+v", c3, c1)
	}

	if err := s.DeleteUser(ctx, u.ID); err != nil {
		t.Fatal(err)
	}
	if entries, _ := os.ReadDir(filepath.Join(root, "comics")); len(entries) != 0 {
		t.Fatalf("comic dirs left after deleting user: %v", entries)
	}
	var n int
	s.db.QueryRow(`SELECT (SELECT count(*) FROM pages) + (SELECT count(*) FROM groups) + (SELECT count(*) FROM histories)`).Scan(&n)
	if n != 0 {
		t.Fatalf("expected cascade delete, %d rows remain", n)
	}
}

func TestRenameComic(t *testing.T) {
	s, m, _ := newTestStore(t)
	ctx := context.Background()
	u := createUser(t, s, "a@example.com")
	other := createUser(t, s, "b@example.com")
	g, _ := s.CreateGroup(ctx, u.ID, "Group")
	c := createComic(t, s, m, u.ID, g.ID, "issue1.CBZ", 1)
	if err := s.FinishComic(ctx, u.ID, c.ID); err != nil {
		t.Fatal(err)
	}

	if err := s.RenameComic(ctx, u.ID, c.ID, "  Vol. 1  "); err != nil {
		t.Fatal(err)
	}
	d, _ := s.GetComic(ctx, u.ID, c.ID)
	if d.Filename != "Vol. 1.CBZ" || d.Name != "Vol. 1" {
		t.Fatalf("after rename: %+v", d.Comic)
	}
	history, _, _ := s.ListHistory(ctx, u.ID, 1)
	if len(history) != 1 || history[0].ComicName != "Vol. 1.CBZ" || history[0].ComicID == nil {
		t.Fatalf("history not updated: %+v", history)
	}

	var verr *ValidationError
	if err := s.RenameComic(ctx, u.ID, c.ID, " "); !errors.As(err, &verr) {
		t.Fatalf("expected blank name error, got %v", err)
	}
	if err := s.RenameComic(ctx, u.ID, c.ID, strings.Repeat("x", 256)); !errors.As(err, &verr) {
		t.Fatalf("expected long name error, got %v", err)
	}
	if err := s.RenameComic(ctx, other.ID, c.ID, "Mine"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected not found for other user, got %v", err)
	}

	// A name without an archive extension isn't truncated at its last dot.
	if PrettyName("Vol. 1") != "Vol. 1" || PrettyName("Saga 01.cbr") != "Saga 01" {
		t.Fatal("unexpected PrettyName result")
	}
}

func TestHistoryOutlivesComicsAndGroups(t *testing.T) {
	s, m, _ := newTestStore(t)
	ctx := context.Background()
	u := createUser(t, s, "a@example.com")
	g, _ := s.CreateGroup(ctx, u.ID, "Saga")
	c1 := createComic(t, s, m, u.ID, g.ID, "Saga 01.cbz", 1)
	c2 := createComic(t, s, m, u.ID, g.ID, "Saga 02.cbz", 1)
	for _, c := range []*Comic{c1, c2} {
		if err := s.FinishComic(ctx, u.ID, c.ID); err != nil {
			t.Fatal(err)
		}
	}

	check := func(step string, wantGroup bool, wantComics map[string]bool) {
		t.Helper()
		items, total, err := s.ListHistory(ctx, u.ID, 1)
		if err != nil || total != 2 || len(items) != 2 {
			t.Fatalf("%s: expected both entries kept, got %d %+v %v", step, total, items, err)
		}
		for _, h := range items {
			if h.GroupName != "Saga" || (h.GroupID != nil) != wantGroup || (h.ComicID != nil) != wantComics[h.ComicName] {
				t.Fatalf("%s: unexpected entry %+v", step, h)
			}
		}
	}
	check("before", true, map[string]bool{"Saga 01.cbz": true, "Saga 02.cbz": true})

	if err := s.DeleteComic(ctx, u.ID, c1.ID); err != nil {
		t.Fatal(err)
	}
	check("after deleting a comic", true, map[string]bool{"Saga 02.cbz": true})

	if err := s.DeleteReadComics(ctx, u.ID); err != nil {
		t.Fatal(err)
	}
	if err := s.DeleteGroup(ctx, u.ID, g.ID); err != nil {
		t.Fatal(err)
	}
	check("after deleting the group", false, map[string]bool{})
}
