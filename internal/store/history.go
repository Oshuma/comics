package store

import (
	"context"
	"time"
)

type History struct {
	ID        int       `json:"id"`
	GroupName string    `json:"groupName"`
	ComicName string    `json:"comicName"`
	CreatedAt time.Time `json:"createdAt"`
	// Set when the group/comic still exists.
	GroupID *int `json:"groupId"`
	ComicID *int `json:"comicId"`
}

// RecordHistory adds a history entry unless an identical one already exists.
func (s *Store) RecordHistory(ctx context.Context, userID int, groupName, comicName string) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT OR IGNORE INTO histories (user_id, group_name, comic_name) VALUES (?, ?, ?)`,
		userID, groupName, comicName)
	return err
}

func (s *Store) ListHistory(ctx context.Context, userID, page int) ([]History, int, error) {
	var total int
	if err := s.db.QueryRowContext(ctx, `SELECT count(*) FROM histories WHERE user_id = ?`, userID).Scan(&total); err != nil {
		return nil, 0, err
	}

	rows, err := s.db.QueryContext(ctx, `
		SELECT h.id, h.group_name, h.comic_name, h.created_at,
			(SELECT g.id FROM groups g WHERE g.user_id = h.user_id AND g.name = h.group_name),
			(SELECT c.id FROM comics c WHERE c.user_id = h.user_id AND c.filename = h.comic_name ORDER BY c.id LIMIT 1)
		FROM histories h
		WHERE h.user_id = ?
		ORDER BY h.created_at DESC, h.id DESC
		LIMIT ? OFFSET ?`, userID, PerPage, offset(page))
	items, err := collect(rows, err, func(row scanner) (History, error) {
		var h History
		err := row.Scan(&h.ID, &h.GroupName, &h.ComicName, &h.CreatedAt, &h.GroupID, &h.ComicID)
		return h, err
	})
	return items, total, err
}
