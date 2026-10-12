package store

import (
	"context"
	"database/sql"
	"strings"
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
	SELECT g.id, g.name,
		(SELECT count(*) FROM comics c WHERE c.group_id = g.id),
		(SELECT p.id FROM comics c JOIN pages p ON p.comic_id = c.id
			WHERE c.group_id = g.id ORDER BY ` + comicOrder + `, p.number LIMIT 1),
		NOT EXISTS (SELECT 1 FROM comics c JOIN pages p ON p.comic_id = c.id
			WHERE c.group_id = g.id AND NOT p.read),
		EXISTS (SELECT 1 FROM comics c JOIN pages p ON p.comic_id = c.id
			WHERE c.group_id = g.id AND p.read),
		(SELECT COALESCE(sum(p.size), 0) FROM comics c JOIN pages p ON p.comic_id = c.id
			WHERE c.group_id = g.id)
	FROM groups g`

func scanGroup(row scanner) (Group, error) {
	var g Group
	err := row.Scan(&g.ID, &g.Name, &g.ComicCount, &g.CoverPageID, &g.Read, &g.Reading, &g.DiskSize)
	return g, err
}

func (s *Store) ListGroups(ctx context.Context, userID int) ([]Group, error) {
	rows, err := s.db.QueryContext(ctx, groupSelect+` WHERE g.user_id = ? ORDER BY g.name, g.id`, userID)
	return collect(rows, err, scanGroup)
}

func (s *Store) GetGroup(ctx context.Context, userID, id int) (*Group, error) {
	g, err := scanGroup(s.db.QueryRowContext(ctx, groupSelect+` WHERE g.user_id = ? AND g.id = ?`, userID, id))
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

var errGroupTaken = invalid("Name has already been taken")

func (s *Store) CreateGroup(ctx context.Context, userID int, name string) (*Group, error) {
	name, err := validateGroupName(name)
	if err != nil {
		return nil, err
	}
	g := &Group{Name: name, Read: true}
	err = s.db.QueryRowContext(ctx,
		`INSERT INTO groups (name, user_id) VALUES (?, ?) RETURNING id`, name, userID,
	).Scan(&g.ID)
	if isUniqueViolation(err) {
		return nil, errGroupTaken
	}
	return g, err
}

// FindOrCreateGroup returns the user's group with the given name (ignoring
// case), creating it if needed.
func (s *Store) FindOrCreateGroup(ctx context.Context, userID int, name string) (int, error) {
	name, err := validateGroupName(name)
	if err != nil {
		return 0, err
	}
	var id int
	// The no-op update makes RETURNING yield the existing row on conflict.
	err = s.db.QueryRowContext(ctx, `
		INSERT INTO groups (name, user_id) VALUES (?, ?)
		ON CONFLICT (user_id, name) DO UPDATE SET name = name
		RETURNING id`, name, userID,
	).Scan(&id)
	return id, err
}

func (s *Store) RenameGroup(ctx context.Context, userID, id int, name string) error {
	name, err := validateGroupName(name)
	if err != nil {
		return err
	}
	err = requireAffected(s.db.ExecContext(ctx,
		`UPDATE groups SET name = ? WHERE user_id = ? AND id = ?`, name, userID, id))
	if isUniqueViolation(err) {
		return errGroupTaken
	}
	return err
}

func (s *Store) DeleteGroup(ctx context.Context, userID, id int) error {
	return s.deleteComics(ctx, func(tx *sql.Tx) ([]int, error) {
		comicIDs, err := deletedIDs(ctx, tx,
			`DELETE FROM comics WHERE user_id = ? AND group_id = ? RETURNING id`, userID, id)
		if err != nil {
			return nil, err
		}
		err = requireAffected(tx.ExecContext(ctx, `DELETE FROM groups WHERE user_id = ? AND id = ?`, userID, id))
		return comicIDs, err
	})
}

// DeleteGroupComics removes all comics in a group, or only the read ones.
func (s *Store) DeleteGroupComics(ctx context.Context, userID, groupID int, onlyRead bool) error {
	if _, err := s.GetGroup(ctx, userID, groupID); err != nil {
		return err
	}
	query := `DELETE FROM comics AS c WHERE c.user_id = ? AND c.group_id = ?`
	if onlyRead {
		query += ` AND ` + readComicCondition
	}
	return s.deleteComics(ctx, func(tx *sql.Tx) ([]int, error) {
		return deletedIDs(ctx, tx, query+` RETURNING id`, userID, groupID)
	})
}

// DeleteReadComics removes every read comic the user has.
func (s *Store) DeleteReadComics(ctx context.Context, userID int) error {
	return s.deleteComics(ctx, func(tx *sql.Tx) ([]int, error) {
		return deletedIDs(ctx, tx,
			`DELETE FROM comics AS c WHERE c.user_id = ? AND `+readComicCondition+` RETURNING id`, userID)
	})
}
