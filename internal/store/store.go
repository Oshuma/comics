package store

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"strings"

	"modernc.org/sqlite"
	sqlite3 "modernc.org/sqlite/lib"

	"github.com/Oshuma/comics/internal/media"
)

const PerPage = 25

var ErrNotFound = errors.New("not found")

// ValidationError holds user-facing validation messages.
type ValidationError struct {
	Messages []string
}

func (e *ValidationError) Error() string { return strings.Join(e.Messages, ", ") }

func invalid(msgs ...string) error { return &ValidationError{Messages: msgs} }

type Store struct {
	db    *sql.DB
	media *media.Storage
}

func New(db *sql.DB, media *media.Storage) *Store {
	return &Store{db: db, media: media}
}

// scanner is implemented by *sql.Row and *sql.Rows.
type scanner interface {
	Scan(dest ...any) error
}

func notFound(err error) error {
	if errors.Is(err, sql.ErrNoRows) {
		return ErrNotFound
	}
	return err
}

func isUniqueViolation(err error) bool {
	var serr *sqlite.Error
	return errors.As(err, &serr) && serr.Code() == sqlite3.SQLITE_CONSTRAINT_UNIQUE
}

func requireAffected(res sql.Result, err error) error {
	if err != nil {
		return err
	}
	if n, err := res.RowsAffected(); err != nil {
		return err
	} else if n == 0 {
		return ErrNotFound
	}
	return nil
}

func offset(page int) int {
	if page < 1 {
		page = 1
	}
	return (page - 1) * PerPage
}

// collect scans every row of a query.
func collect[T any](rows *sql.Rows, err error, scan func(scanner) (T, error)) ([]T, error) {
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []T
	for rows.Next() {
		item, err := scan(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func scanInt(row scanner) (int, error) {
	var n int
	err := row.Scan(&n)
	return n, err
}

// deleteComics runs a DELETE ... RETURNING id statement for comics in a
// transaction (pages cascade), then removes the deleted comics' files.
func (s *Store) deleteComics(ctx context.Context, fn func(tx *sql.Tx) ([]int, error)) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	comicIDs, err := fn(tx)
	if err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	for _, id := range comicIDs {
		if err := s.media.DeleteComic(id); err != nil {
			slog.Error("removing comic files", "comic", id, "err", err)
		}
	}
	return nil
}

func deletedIDs(ctx context.Context, tx *sql.Tx, query string, args ...any) ([]int, error) {
	rows, err := tx.QueryContext(ctx, query, args...)
	return collect(rows, err, scanInt)
}
