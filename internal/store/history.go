package store

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
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
	_, err := s.db.Exec(ctx, `
		INSERT INTO histories (user_id, group_name, comic_name, created_at, updated_at)
		SELECT $1::integer, $2::varchar, $3::varchar, `+now+`, `+now+`
		WHERE NOT EXISTS (
			SELECT 1 FROM histories WHERE user_id = $1::integer AND group_name = $2::varchar AND comic_name = $3::varchar
		)`, userID, groupName, comicName)
	return err
}

func (s *Store) ListHistory(ctx context.Context, userID, page int) ([]History, int, error) {
	var total int
	if err := s.db.QueryRow(ctx, `SELECT count(*) FROM histories WHERE user_id = $1`, userID).Scan(&total); err != nil {
		return nil, 0, err
	}

	rows, err := s.db.Query(ctx, `
		SELECT h.id, COALESCE(h.group_name, ''), COALESCE(h.comic_name, ''), h.created_at,
			(SELECT g.id FROM groups g WHERE g.user_id = h.user_id AND g.name = h.group_name ORDER BY g.id LIMIT 1),
			(SELECT c.id FROM comics c WHERE c.user_id = h.user_id AND c.filename = h.comic_name ORDER BY c.id LIMIT 1)
		FROM histories h
		WHERE h.user_id = $1
		ORDER BY h.created_at DESC, h.id DESC
		LIMIT $2 OFFSET $3`, userID, PerPage, offset(page))
	if err != nil {
		return nil, 0, err
	}
	items, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (History, error) {
		var h History
		err := row.Scan(&h.ID, &h.GroupName, &h.ComicName, &h.CreatedAt, &h.GroupID, &h.ComicID)
		return h, err
	})
	return items, total, err
}
