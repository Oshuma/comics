package store

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"
)

type Group struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	ComicCount  int    `json:"comicCount"`
	CoverPageID *int   `json:"coverPageId"`
	Read        bool   `json:"read"`
	Reading     bool   `json:"reading"`
	DiskSize    int64  `json:"diskSize"`
}

// Shared select for groups with their aggregate reading state.
const groupSelect = `
	SELECT g.id, COALESCE(g.name, ''),
		(SELECT count(*) FROM comics c WHERE c.group_id = g.id),
		(SELECT p.id FROM comics c JOIN pages p ON p.comic_id = c.id
			WHERE c.group_id = g.id ORDER BY c.filename, c.id, p.number LIMIT 1),
		NOT EXISTS (SELECT 1 FROM comics c JOIN pages p ON p.comic_id = c.id
			WHERE c.group_id = g.id AND NOT COALESCE(p.read, false)),
		EXISTS (SELECT 1 FROM comics c JOIN pages p ON p.comic_id = c.id
			WHERE c.group_id = g.id AND p.read),
		(SELECT COALESCE(sum(p.image_file_size), 0) FROM comics c JOIN pages p ON p.comic_id = c.id
			WHERE c.group_id = g.id)::bigint
	FROM groups g`

func scanGroup(row pgx.Row) (Group, error) {
	var g Group
	err := row.Scan(&g.ID, &g.Name, &g.ComicCount, &g.CoverPageID, &g.Read, &g.Reading, &g.DiskSize)
	return g, err
}

func (s *Store) ListGroups(ctx context.Context, userID int) ([]Group, error) {
	rows, err := s.db.Query(ctx, groupSelect+` WHERE g.user_id = $1 ORDER BY g.name ASC, g.id ASC`, userID)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (Group, error) { return scanGroup(row) })
}

func (s *Store) GetGroup(ctx context.Context, userID, id int) (*Group, error) {
	g, err := scanGroup(s.db.QueryRow(ctx, groupSelect+` WHERE g.user_id = $1 AND g.id = $2`, userID, id))
	if err != nil {
		return nil, notFound(err)
	}
	return &g, nil
}

func validateGroupName(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", invalid("Name can't be blank")
	}
	return name, nil
}

func (s *Store) CreateGroup(ctx context.Context, userID int, name string) (*Group, error) {
	name, err := validateGroupName(name)
	if err != nil {
		return nil, err
	}
	g := &Group{Name: name, Read: true}
	err = s.db.QueryRow(ctx, `
		INSERT INTO groups (name, user_id, created_at, updated_at)
		VALUES ($1, $2, `+now+`, `+now+`) RETURNING id`, name, userID,
	).Scan(&g.ID)
	return g, err
}

// FindOrCreateGroup returns the user's group with the given name, creating it if needed.
func (s *Store) FindOrCreateGroup(ctx context.Context, userID int, name string) (int, error) {
	name, err := validateGroupName(name)
	if err != nil {
		return 0, err
	}
	var id int
	err = s.db.QueryRow(ctx,
		`SELECT id FROM groups WHERE user_id = $1 AND name = $2 ORDER BY id LIMIT 1`, userID, name,
	).Scan(&id)
	if err == nil {
		return id, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return 0, err
	}
	g, err := s.CreateGroup(ctx, userID, name)
	if err != nil {
		return 0, err
	}
	return g.ID, nil
}

func (s *Store) RenameGroup(ctx context.Context, userID, id int, name string) error {
	name, err := validateGroupName(name)
	if err != nil {
		return err
	}
	tag, err := s.db.Exec(ctx,
		`UPDATE groups SET name = $3, updated_at = `+now+` WHERE user_id = $1 AND id = $2`, userID, id, name)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) DeleteGroup(ctx context.Context, userID, id int) error {
	return s.deleteTx(ctx, func(tx pgx.Tx) ([]int, error) {
		comicIDs, err := collectIDs(ctx, tx, `SELECT id FROM comics WHERE user_id = $1 AND group_id = $2`, userID, id)
		if err != nil {
			return nil, err
		}
		pageIDs, err := s.deleteComics(ctx, tx, comicIDs)
		if err != nil {
			return nil, err
		}
		tag, err := tx.Exec(ctx, `DELETE FROM groups WHERE user_id = $1 AND id = $2`, userID, id)
		if err != nil {
			return nil, err
		}
		if tag.RowsAffected() == 0 {
			return nil, ErrNotFound
		}
		return pageIDs, nil
	})
}

// Comics are considered read when none of their pages are unread.
const readComicCondition = `NOT EXISTS (SELECT 1 FROM pages p WHERE p.comic_id = c.id AND NOT COALESCE(p.read, false))`

// DeleteGroupComics removes all comics in a group, or only the read ones.
func (s *Store) DeleteGroupComics(ctx context.Context, userID, groupID int, onlyRead bool) error {
	if _, err := s.GetGroup(ctx, userID, groupID); err != nil {
		return err
	}
	sql := `SELECT c.id FROM comics c WHERE c.user_id = $1 AND c.group_id = $2`
	if onlyRead {
		sql += ` AND ` + readComicCondition
	}
	return s.deleteTx(ctx, func(tx pgx.Tx) ([]int, error) {
		comicIDs, err := collectIDs(ctx, tx, sql, userID, groupID)
		if err != nil {
			return nil, err
		}
		return s.deleteComics(ctx, tx, comicIDs)
	})
}

// DeleteReadComics removes every read comic the user has.
func (s *Store) DeleteReadComics(ctx context.Context, userID int) error {
	return s.deleteTx(ctx, func(tx pgx.Tx) ([]int, error) {
		comicIDs, err := collectIDs(ctx, tx,
			`SELECT c.id FROM comics c WHERE c.user_id = $1 AND `+readComicCondition, userID)
		if err != nil {
			return nil, err
		}
		return s.deleteComics(ctx, tx, comicIDs)
	})
}
