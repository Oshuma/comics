package store

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/jackc/pgx/v5"

	"github.com/Oshuma/comics/internal/archive"
	"github.com/Oshuma/comics/internal/media"
)

type Comic struct {
	ID          int    `json:"id"`
	Filename    string `json:"filename"`
	Name        string `json:"name"`
	GroupID     int    `json:"groupId"`
	PageCount   int    `json:"pageCount"`
	CoverPageID *int   `json:"coverPageId"`
	Read        bool   `json:"read"`
	Reading     bool   `json:"reading"`
}

type ComicDetail struct {
	Comic
	GroupName string `json:"groupName"`
	Pages     []Page `json:"pages"`
}

type Page struct {
	ID     int  `json:"id"`
	Number int  `json:"number"`
	Read   bool `json:"read"`
}

// PrettyName strips the extension from a comic's filename.
func PrettyName(filename string) string {
	return strings.TrimSuffix(filename, filepath.Ext(filename))
}

const comicSelect = `
	SELECT c.id, COALESCE(c.filename, ''), c.group_id,
		(SELECT count(*) FROM pages p WHERE p.comic_id = c.id),
		(SELECT p.id FROM pages p WHERE p.comic_id = c.id ORDER BY p.number LIMIT 1),
		` + readComicCondition + `,
		EXISTS (SELECT 1 FROM pages p WHERE p.comic_id = c.id AND p.read)
	FROM comics c`

func scanComic(row pgx.Row) (Comic, error) {
	var c Comic
	err := row.Scan(&c.ID, &c.Filename, &c.GroupID, &c.PageCount, &c.CoverPageID, &c.Read, &c.Reading)
	c.Name = PrettyName(c.Filename)
	return c, err
}

func (s *Store) ListComics(ctx context.Context, userID, groupID int) ([]Comic, error) {
	rows, err := s.db.Query(ctx,
		comicSelect+` WHERE c.user_id = $1 AND c.group_id = $2 ORDER BY c.filename ASC, c.id ASC`, userID, groupID)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (Comic, error) { return scanComic(row) })
}

func (s *Store) GetComic(ctx context.Context, userID, id int) (*ComicDetail, error) {
	c, err := scanComic(s.db.QueryRow(ctx, comicSelect+` WHERE c.user_id = $1 AND c.id = $2`, userID, id))
	if err != nil {
		return nil, notFound(err)
	}
	d := &ComicDetail{Comic: c}

	if err := s.db.QueryRow(ctx, `SELECT COALESCE(name, '') FROM groups WHERE id = $1`, c.GroupID).Scan(&d.GroupName); err != nil {
		return nil, err
	}

	rows, err := s.db.Query(ctx,
		`SELECT id, number, COALESCE(read, false) FROM pages WHERE comic_id = $1 ORDER BY number`, id)
	if err != nil {
		return nil, err
	}
	d.Pages, err = pgx.CollectRows(rows, func(row pgx.CollectableRow) (Page, error) {
		var p Page
		err := row.Scan(&p.ID, &p.Number, &p.Read)
		return p, err
	})
	if err != nil {
		return nil, err
	}
	return d, nil
}

// CreateComic stores a comic and its pages from extracted archive entries.
// Entries that aren't images are skipped.
func (s *Store) CreateComic(ctx context.Context, userID, groupID int, filename string, entries []archive.Entry) (*Comic, error) {
	if strings.TrimSpace(filename) == "" {
		return nil, invalid("Filename can't be blank")
	}
	if _, err := s.GetGroup(ctx, userID, groupID); err != nil {
		return nil, err
	}

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	c := &Comic{Filename: filename, Name: PrettyName(filename), GroupID: groupID}
	err = tx.QueryRow(ctx, `
		INSERT INTO comics (filename, user_id, group_id, created_at, updated_at)
		VALUES ($1, $2, $3, `+now+`, `+now+`) RETURNING id`, filename, userID, groupID,
	).Scan(&c.ID)
	if err != nil {
		return nil, err
	}

	var imported []int
	cleanup := func() { s.removePageFiles(imported) }

	for _, e := range entries {
		contentType, ok := media.DetectImageType(e.Path)
		if !ok {
			continue
		}
		var size int64
		if info, err := os.Stat(e.Path); err == nil {
			size = info.Size()
		}
		fileName := media.SanitizeFileName(e.Name)

		var pageID int
		err := tx.QueryRow(ctx, `
			INSERT INTO pages (number, read, comic_id, image_file_name, image_content_type, image_file_size,
				image_updated_at, created_at, updated_at)
			VALUES ($1, false, $2, $3, $4, $5, `+now+`, `+now+`, `+now+`) RETURNING id`,
			len(imported)+1, c.ID, fileName, contentType, size,
		).Scan(&pageID)
		if err != nil {
			cleanup()
			return nil, err
		}
		imported = append(imported, pageID)
		if err := s.media.Import(pageID, fileName, e.Path); err != nil {
			cleanup()
			return nil, err
		}
	}

	if len(imported) == 0 {
		return nil, invalid("No images found in archive")
	}
	if err := tx.Commit(ctx); err != nil {
		cleanup()
		return nil, err
	}

	c.PageCount = len(imported)
	c.CoverPageID = &imported[0]
	return c, nil
}

func (s *Store) MoveComic(ctx context.Context, userID, comicID, groupID int) error {
	if _, err := s.GetGroup(ctx, userID, groupID); err != nil {
		return err
	}
	tag, err := s.db.Exec(ctx,
		`UPDATE comics SET group_id = $3, updated_at = `+now+` WHERE user_id = $1 AND id = $2`,
		userID, comicID, groupID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) DeleteComic(ctx context.Context, userID, comicID int) error {
	return s.deleteTx(ctx, func(tx pgx.Tx) ([]int, error) {
		ids, err := collectIDs(ctx, tx, `SELECT id FROM comics WHERE user_id = $1 AND id = $2`, userID, comicID)
		if err != nil {
			return nil, err
		}
		if len(ids) == 0 {
			return nil, ErrNotFound
		}
		return s.deleteComics(ctx, tx, ids)
	})
}

// FinishComic marks every page read and records the comic in the history.
func (s *Store) FinishComic(ctx context.Context, userID, comicID int) error {
	c, err := s.GetComic(ctx, userID, comicID)
	if err != nil {
		return err
	}
	if _, err := s.db.Exec(ctx,
		`UPDATE pages SET read = true, updated_at = `+now+` WHERE comic_id = $1`, comicID); err != nil {
		return err
	}
	return s.RecordHistory(ctx, userID, c.GroupName, c.Filename)
}

// FirstUnreadPage returns the ID of the comic's first unread page, or nil.
func (s *Store) FirstUnreadPage(ctx context.Context, userID, comicID int) (*int, error) {
	if _, err := s.comicOwned(ctx, userID, comicID); err != nil {
		return nil, err
	}
	var id int
	err := s.db.QueryRow(ctx, `
		SELECT id FROM pages WHERE comic_id = $1 AND NOT COALESCE(read, false)
		ORDER BY number LIMIT 1`, comicID).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &id, nil
}

// NextComic returns the ID of the comic following comicID in its group, or nil.
func (s *Store) NextComic(ctx context.Context, userID, comicID int) (*int, error) {
	groupID, err := s.comicOwned(ctx, userID, comicID)
	if err != nil {
		return nil, err
	}
	comics, err := s.ListComics(ctx, userID, groupID)
	if err != nil {
		return nil, err
	}
	for i, c := range comics {
		if c.ID == comicID && i+1 < len(comics) {
			return &comics[i+1].ID, nil
		}
	}
	return nil, nil
}

// comicOwned checks ownership and returns the comic's group ID.
func (s *Store) comicOwned(ctx context.Context, userID, comicID int) (int, error) {
	var groupID int
	err := s.db.QueryRow(ctx,
		`SELECT group_id FROM comics WHERE user_id = $1 AND id = $2`, userID, comicID).Scan(&groupID)
	return groupID, notFound(err)
}

type PageView struct {
	Page
	ComicID    int    `json:"comicId"`
	ComicName  string `json:"comicName"`
	GroupID    int    `json:"groupId"`
	GroupName  string `json:"groupName"`
	PageCount  int    `json:"pageCount"`
	PrevPageID *int   `json:"prevPageId"`
	NextPageID *int   `json:"nextPageId"`
}

// ViewPage marks the page as the current reading position (earlier pages
// read, this and later pages unread) and returns its reader context.
func (s *Store) ViewPage(ctx context.Context, userID, comicID, pageID int) (*PageView, error) {
	v := &PageView{ComicID: comicID}
	var filename string
	err := s.db.QueryRow(ctx, `
		SELECT p.id, p.number, COALESCE(c.filename, ''), c.group_id, COALESCE(g.name, ''),
			(SELECT count(*) FROM pages x WHERE x.comic_id = c.id),
			(SELECT x.id FROM pages x WHERE x.comic_id = c.id AND x.number < p.number ORDER BY x.number DESC LIMIT 1),
			(SELECT x.id FROM pages x WHERE x.comic_id = c.id AND x.number > p.number ORDER BY x.number ASC LIMIT 1)
		FROM pages p
		JOIN comics c ON c.id = p.comic_id
		JOIN groups g ON g.id = c.group_id
		WHERE c.user_id = $1 AND c.id = $2 AND p.id = $3`, userID, comicID, pageID,
	).Scan(&v.ID, &v.Number, &filename, &v.GroupID, &v.GroupName, &v.PageCount, &v.PrevPageID, &v.NextPageID)
	if err != nil {
		return nil, notFound(err)
	}
	v.ComicName = PrettyName(filename)

	_, err = s.db.Exec(ctx, `
		UPDATE pages SET read = (number < $2), updated_at = `+now+`
		WHERE comic_id = $1 AND COALESCE(read, false) <> (number < $2)`, comicID, v.Number)
	if err != nil {
		return nil, err
	}
	return v, nil
}

type PageImage struct {
	ID       int
	FileName string
}

// GetPageImage returns the image info for a page owned by the user.
func (s *Store) GetPageImage(ctx context.Context, userID, pageID int) (*PageImage, error) {
	img := &PageImage{ID: pageID}
	err := s.db.QueryRow(ctx, `
		SELECT COALESCE(p.image_file_name, '')
		FROM pages p JOIN comics c ON c.id = p.comic_id
		WHERE c.user_id = $1 AND p.id = $2`, userID, pageID,
	).Scan(&img.FileName)
	if err != nil {
		return nil, notFound(err)
	}
	return img, nil
}
