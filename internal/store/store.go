package store

import (
	"context"
	"errors"
	"log"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Oshuma/comics/internal/media"
)

// Postgres expression for the current UTC time, matching how Rails stored
// timestamps in "timestamp without time zone" columns.
const now = "(now() AT TIME ZONE 'utc')"

const PerPage = 25

var ErrNotFound = errors.New("not found")

// ValidationError holds user-facing validation messages.
type ValidationError struct {
	Messages []string
}

func (e *ValidationError) Error() string { return strings.Join(e.Messages, ", ") }

func invalid(msgs ...string) error { return &ValidationError{Messages: msgs} }

type Store struct {
	db    *pgxpool.Pool
	media *media.Storage
}

func New(db *pgxpool.Pool, media *media.Storage) *Store {
	return &Store{db: db, media: media}
}

func notFound(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	return err
}

func offset(page int) int {
	if page < 1 {
		page = 1
	}
	return (page - 1) * PerPage
}

// deleteComics removes comics and their pages, returning the deleted page IDs.
func (s *Store) deleteComics(ctx context.Context, tx pgx.Tx, comicIDs []int) ([]int, error) {
	if len(comicIDs) == 0 {
		return nil, nil
	}
	rows, err := tx.Query(ctx, `DELETE FROM pages WHERE comic_id = ANY($1) RETURNING id`, comicIDs)
	if err != nil {
		return nil, err
	}
	pageIDs, err := pgx.CollectRows(rows, pgx.RowTo[int])
	if err != nil {
		return nil, err
	}
	if _, err := tx.Exec(ctx, `DELETE FROM comics WHERE id = ANY($1)`, comicIDs); err != nil {
		return nil, err
	}
	return pageIDs, nil
}

func (s *Store) removePageFiles(pageIDs []int) {
	for _, id := range pageIDs {
		if err := s.media.DeletePage(id); err != nil {
			log.Printf("removing files for page %d: %v", id, err)
		}
	}
}

// deleteTx runs fn in a transaction. fn returns the IDs of deleted pages,
// whose files are removed once the transaction commits.
func (s *Store) deleteTx(ctx context.Context, fn func(tx pgx.Tx) ([]int, error)) error {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	pageIDs, err := fn(tx)
	if err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return err
	}
	s.removePageFiles(pageIDs)
	return nil
}

func collectIDs(ctx context.Context, tx pgx.Tx, sql string, args ...any) ([]int, error) {
	rows, err := tx.Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowTo[int])
}
