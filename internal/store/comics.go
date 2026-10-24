package store

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"strings"
	"unicode/utf8"

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

var archiveExts = map[string]bool{".cbz": true, ".cbr": true, ".zip": true, ".rar": true}

// archiveExt returns the filename's extension if it's an archive type, so
// names like "Vol. 1" aren't mistaken for having one.
func archiveExt(filename string) string {
	ext := filepath.Ext(filename)
	if archiveExts[strings.ToLower(ext)] {
		return ext
	}
	return ""
}

// PrettyName strips the archive extension from a comic's filename.
func PrettyName(filename string) string {
	return strings.TrimSuffix(filename, archiveExt(filename))
}

// Comics are considered read when none of their pages are unread.
const readComicCondition = `NOT EXISTS (SELECT 1 FROM pages p WHERE p.comic_id = c.id AND NOT p.read)`

const comicOrder = `c.filename COLLATE NOCASE, c.id`

const comicSelect = `
	SELECT c.id, c.filename, c.group_id,
		(SELECT count(*) FROM pages p WHERE p.comic_id = c.id),
		(SELECT p.id FROM pages p WHERE p.comic_id = c.id ORDER BY p.number LIMIT 1),
		` + readComicCondition + `,
		EXISTS (SELECT 1 FROM pages p WHERE p.comic_id = c.id AND p.read)
	FROM comics c`

func scanComic(row scanner) (Comic, error) {
	var c Comic
	err := row.Scan(&c.ID, &c.Filename, &c.GroupID, &c.PageCount, &c.CoverPageID, &c.Read, &c.Reading)
	c.Name = PrettyName(c.Filename)
	return c, err
}

func (s *Store) ListComics(ctx context.Context, userID, groupID int) ([]Comic, error) {
	rows, err := s.db.QueryContext(ctx,
		comicSelect+` WHERE c.user_id = ? AND c.group_id = ? ORDER BY `+comicOrder, userID, groupID)
	return collect(rows, err, scanComic)
}

func (s *Store) GetComic(ctx context.Context, userID, id int) (*ComicDetail, error) {
	c, err := scanComic(s.db.QueryRowContext(ctx, comicSelect+` WHERE c.user_id = ? AND c.id = ?`, userID, id))
	if err != nil {
		return nil, notFound(err)
	}
	d := &ComicDetail{Comic: c}

	if err := s.db.QueryRowContext(ctx, `SELECT name FROM groups WHERE id = ?`, c.GroupID).Scan(&d.GroupName); err != nil {
		return nil, err
	}

	rows, err := s.db.QueryContext(ctx, `SELECT id, number, read FROM pages WHERE comic_id = ? ORDER BY number`, id)
	d.Pages, err = collect(rows, err, func(row scanner) (Page, error) {
		var p Page
		err := row.Scan(&p.ID, &p.Number, &p.Read)
		return p, err
	})
	if err != nil {
		return nil, err
	}
	return d, nil
}

// CreateComic stores a comic from a directory of staged pages, which is moved
// into the comic's storage location.
func (s *Store) CreateComic(ctx context.Context, userID, groupID int, filename, stagedDir string, pages []media.StagedPage) (*Comic, error) {
	if strings.TrimSpace(filename) == "" {
		return nil, invalid("Filename can't be blank")
	}
	if len(pages) == 0 {
		return nil, invalid("No images found in archive")
	}
	if _, err := s.GetGroup(ctx, userID, groupID); err != nil {
		return nil, err
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	c := &Comic{Filename: filename, Name: PrettyName(filename), GroupID: groupID, PageCount: len(pages)}
	err = tx.QueryRowContext(ctx,
		`INSERT INTO comics (filename, user_id, group_id) VALUES (?, ?, ?) RETURNING id`,
		filename, userID, groupID,
	).Scan(&c.ID)
	if err != nil {
		return nil, err
	}

	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO pages (comic_id, number, file, content_type, size)
		VALUES (?, ?, ?, ?, ?) RETURNING id`)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()
	for i, p := range pages {
		var pageID int
		if err := stmt.QueryRowContext(ctx, c.ID, i+1, p.File, p.ContentType, p.Size).Scan(&pageID); err != nil {
			return nil, err
		}
		if i == 0 {
			c.CoverPageID = &pageID
		}
	}

	if err := s.media.InstallComic(c.ID, stagedDir); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		s.media.DeleteComic(c.ID)
		return nil, err
	}
	return c, nil
}

// RenameComic sets the comic's name, keeping its archive extension. History
// entries for the comic follow the rename.
func (s *Store) RenameComic(ctx context.Context, userID, comicID int, name string) error {
	name = strings.TrimSpace(name)
	switch {
	case name == "":
		return invalid("Name can't be blank")
	case utf8.RuneCountInString(name) > 255:
		return invalid("Name is too long (maximum is 255 characters)")
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var oldFilename, groupName string
	err = tx.QueryRowContext(ctx, `
		SELECT c.filename, g.name FROM comics c JOIN groups g ON g.id = c.group_id
		WHERE c.user_id = ? AND c.id = ?`, userID, comicID,
	).Scan(&oldFilename, &groupName)
	if err != nil {
		return notFound(err)
	}
	filename := name + archiveExt(oldFilename)
	if filename == oldFilename {
		return nil
	}

	if _, err := tx.ExecContext(ctx, `UPDATE comics SET filename = ? WHERE id = ?`, filename, comicID); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `
		UPDATE OR IGNORE histories SET comic_name = ?
		WHERE user_id = ? AND group_name = ? AND comic_name = ?`,
		filename, userID, groupName, oldFilename); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Store) MoveComic(ctx context.Context, userID, comicID, groupID int) error {
	if _, err := s.GetGroup(ctx, userID, groupID); err != nil {
		return err
	}
	return requireAffected(s.db.ExecContext(ctx,
		`UPDATE comics SET group_id = ? WHERE user_id = ? AND id = ?`, groupID, userID, comicID))
}

func (s *Store) DeleteComic(ctx context.Context, userID, comicID int) error {
	return s.deleteComics(ctx, func(tx *sql.Tx) ([]int, error) {
		ids, err := deletedIDs(ctx, tx, `DELETE FROM comics WHERE user_id = ? AND id = ? RETURNING id`, userID, comicID)
		if err == nil && len(ids) == 0 {
			err = ErrNotFound
		}
		return ids, err
	})
}

// FinishComic marks every page read and records the comic in the history.
func (s *Store) FinishComic(ctx context.Context, userID, comicID int) error {
	c, err := s.GetComic(ctx, userID, comicID)
	if err != nil {
		return err
	}
	if _, err := s.db.ExecContext(ctx, `UPDATE pages SET read = 1 WHERE comic_id = ? AND NOT read`, comicID); err != nil {
		return err
	}
	return s.RecordHistory(ctx, userID, c.GroupName, c.Filename)
}

// MarkComicUnread marks every page of the comic unread.
func (s *Store) MarkComicUnread(ctx context.Context, userID, comicID int) error {
	if _, err := s.comicOwned(ctx, userID, comicID); err != nil {
		return err
	}
	_, err := s.db.ExecContext(ctx, `UPDATE pages SET read = 0 WHERE comic_id = ? AND read`, comicID)
	return err
}

// FirstUnreadPage returns the ID of the comic's first unread page, or nil.
func (s *Store) FirstUnreadPage(ctx context.Context, userID, comicID int) (*int, error) {
	if _, err := s.comicOwned(ctx, userID, comicID); err != nil {
		return nil, err
	}
	var id int
	err := s.db.QueryRowContext(ctx,
		`SELECT id FROM pages WHERE comic_id = ? AND NOT read ORDER BY number LIMIT 1`, comicID).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
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
	rows, err := s.db.QueryContext(ctx,
		`SELECT c.id FROM comics c WHERE c.user_id = ? AND c.group_id = ? ORDER BY `+comicOrder, userID, groupID)
	ids, err := collect(rows, err, scanInt)
	if err != nil {
		return nil, err
	}
	for i, id := range ids {
		if id == comicID && i+1 < len(ids) {
			return &ids[i+1], nil
		}
	}
	return nil, nil
}

// comicOwned checks ownership and returns the comic's group ID.
func (s *Store) comicOwned(ctx context.Context, userID, comicID int) (int, error) {
	var groupID int
	err := s.db.QueryRowContext(ctx,
		`SELECT group_id FROM comics WHERE user_id = ? AND id = ?`, userID, comicID).Scan(&groupID)
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
	err := s.db.QueryRowContext(ctx, `
		SELECT p.id, p.number, c.filename, c.group_id, g.name,
			(SELECT count(*) FROM pages x WHERE x.comic_id = c.id),
			(SELECT x.id FROM pages x WHERE x.comic_id = c.id AND x.number < p.number ORDER BY x.number DESC LIMIT 1),
			(SELECT x.id FROM pages x WHERE x.comic_id = c.id AND x.number > p.number ORDER BY x.number ASC LIMIT 1)
		FROM pages p
		JOIN comics c ON c.id = p.comic_id
		JOIN groups g ON g.id = c.group_id
		WHERE c.user_id = ? AND c.id = ? AND p.id = ?`, userID, comicID, pageID,
	).Scan(&v.ID, &v.Number, &filename, &v.GroupID, &v.GroupName, &v.PageCount, &v.PrevPageID, &v.NextPageID)
	if err != nil {
		return nil, notFound(err)
	}
	v.ComicName = PrettyName(filename)

	_, err = s.db.ExecContext(ctx,
		`UPDATE pages SET read = (number < ?1) WHERE comic_id = ?2 AND read <> (number < ?1)`, v.Number, comicID)
	if err != nil {
		return nil, err
	}
	return v, nil
}

type PageImage struct {
	ComicID     int
	File        string
	ContentType string
}

// GetPageImage returns the image info for a page owned by the user.
func (s *Store) GetPageImage(ctx context.Context, userID, pageID int) (*PageImage, error) {
	img := &PageImage{}
	err := s.db.QueryRowContext(ctx, `
		SELECT p.comic_id, p.file, p.content_type
		FROM pages p JOIN comics c ON c.id = p.comic_id
		WHERE c.user_id = ? AND p.id = ?`, userID, pageID,
	).Scan(&img.ComicID, &img.File, &img.ContentType)
	if err != nil {
		return nil, notFound(err)
	}
	return img, nil
}
