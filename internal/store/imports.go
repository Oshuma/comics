package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

const (
	ImportQueued     = "queued"
	ImportProcessing = "processing"
	ImportDone       = "done"
	ImportFailed     = "failed"

	SourceUpload  = "upload"
	SourceArchive = "archive.org"
)

// Import is a background job that turns an uploaded or downloaded archive
// into a comic.
type Import struct {
	ID                int       `json:"id"`
	GroupID           int       `json:"groupId"`
	GroupName         string    `json:"groupName"`
	Filename          string    `json:"filename"`
	Name              string    `json:"name"`
	Source            string    `json:"source"`
	ArchiveIdentifier string    `json:"archiveIdentifier,omitempty"`
	ArchiveName       string    `json:"archiveName,omitempty"`
	Status            string    `json:"status"`
	Error             string    `json:"error,omitempty"`
	ComicID           *int      `json:"comicId"`
	CreatedAt         time.Time `json:"createdAt"`
	UpdatedAt         time.Time `json:"updatedAt"`

	UserID     int    `json:"-"`
	UploadPath string `json:"-"`
}

type NewImport struct {
	UserID            int
	GroupID           int
	Filename          string
	Source            string
	UploadPath        string
	ArchiveIdentifier string
	ArchiveName       string
}

const importSelect = `
	SELECT i.id, i.user_id, i.group_id, g.name, i.filename, i.source, COALESCE(i.upload_path, ''),
		COALESCE(i.archive_identifier, ''), COALESCE(i.archive_name, ''), i.status, COALESCE(i.error, ''),
		i.comic_id, i.created_at, i.updated_at
	FROM imports i JOIN groups g ON g.id = i.group_id`

func scanImport(row scanner) (Import, error) {
	var i Import
	err := row.Scan(&i.ID, &i.UserID, &i.GroupID, &i.GroupName, &i.Filename, &i.Source, &i.UploadPath,
		&i.ArchiveIdentifier, &i.ArchiveName, &i.Status, &i.Error, &i.ComicID, &i.CreatedAt, &i.UpdatedAt)
	i.Name = PrettyName(i.Filename)
	return i, err
}

// CreateImport queues a job for one of the user's groups.
func (s *Store) CreateImport(ctx context.Context, n NewImport) (*Import, error) {
	if _, err := s.GetGroup(ctx, n.UserID, n.GroupID); err != nil {
		return nil, err
	}
	var id int
	err := s.db.QueryRowContext(ctx, `
		INSERT INTO imports (user_id, group_id, filename, source, upload_path, archive_identifier, archive_name)
		VALUES (?, ?, ?, ?, NULLIF(?, ''), NULLIF(?, ''), NULLIF(?, '')) RETURNING id`,
		n.UserID, n.GroupID, n.Filename, n.Source, n.UploadPath, n.ArchiveIdentifier, n.ArchiveName,
	).Scan(&id)
	if err != nil {
		return nil, err
	}
	return s.GetImport(ctx, n.UserID, id)
}

func (s *Store) GetImport(ctx context.Context, userID, id int) (*Import, error) {
	i, err := scanImport(s.db.QueryRowContext(ctx, importSelect+` WHERE i.user_id = ? AND i.id = ?`, userID, id))
	if err != nil {
		return nil, notFound(err)
	}
	return &i, nil
}

// ListImports returns the user's most recent jobs, newest first.
func (s *Store) ListImports(ctx context.Context, userID int) ([]Import, error) {
	rows, err := s.db.QueryContext(ctx, importSelect+` WHERE i.user_id = ? ORDER BY i.id DESC LIMIT 100`, userID)
	return collect(rows, err, scanImport)
}

// ClaimImport marks the oldest queued job as processing and returns it, or
// nil when nothing is queued. It's a single statement, so a job is never left
// claimed without being returned. GroupName isn't set.
func (s *Store) ClaimImport(ctx context.Context) (*Import, error) {
	var i Import
	err := s.db.QueryRowContext(ctx, `
		UPDATE imports SET status = 'processing', updated_at = CURRENT_TIMESTAMP
		WHERE id = (SELECT id FROM imports WHERE status = 'queued' ORDER BY id LIMIT 1)
		RETURNING id, user_id, group_id, filename, source, COALESCE(upload_path, ''),
			COALESCE(archive_identifier, ''), COALESCE(archive_name, ''), status, comic_id, created_at, updated_at`,
	).Scan(&i.ID, &i.UserID, &i.GroupID, &i.Filename, &i.Source, &i.UploadPath,
		&i.ArchiveIdentifier, &i.ArchiveName, &i.Status, &i.ComicID, &i.CreatedAt, &i.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	i.Name = PrettyName(i.Filename)
	return &i, nil
}

// FinishImport, FailImport and RequeueImport only change jobs still being
// processed; they return ErrNotFound if the job was cancelled meanwhile.

func (s *Store) FinishImport(ctx context.Context, id, comicID int) error {
	return requireAffected(s.db.ExecContext(ctx, `
		UPDATE imports SET status = 'done', comic_id = ?, error = NULL, upload_path = NULL, updated_at = CURRENT_TIMESTAMP
		WHERE id = ? AND status = 'processing'`, comicID, id))
}

func (s *Store) FailImport(ctx context.Context, id int, message string) error {
	return requireAffected(s.db.ExecContext(ctx, `
		UPDATE imports SET status = 'failed', error = ?, upload_path = NULL, updated_at = CURRENT_TIMESTAMP
		WHERE id = ? AND status = 'processing'`, message, id))
}

func (s *Store) RequeueImport(ctx context.Context, id int) error {
	return requireAffected(s.db.ExecContext(ctx, `
		UPDATE imports SET status = 'queued', updated_at = CURRENT_TIMESTAMP
		WHERE id = ? AND status = 'processing'`, id))
}

// DeleteImport removes one of the user's jobs and returns it.
func (s *Store) DeleteImport(ctx context.Context, userID, id int) (*Import, error) {
	i, err := s.GetImport(ctx, userID, id)
	if err != nil {
		return nil, err
	}
	if err := requireAffected(s.db.ExecContext(ctx, `DELETE FROM imports WHERE user_id = ? AND id = ?`, userID, id)); err != nil {
		return nil, err
	}
	return i, nil
}

// ResetImports requeues jobs left processing by a previous run.
func (s *Store) ResetImports(ctx context.Context) (int, error) {
	res, err := s.db.ExecContext(ctx, `UPDATE imports SET status = 'queued' WHERE status = 'processing'`)
	if err != nil {
		return 0, err
	}
	n, err := res.RowsAffected()
	return int(n), err
}

// PendingUploadPaths returns the spooled files still needed by jobs.
func (s *Store) PendingUploadPaths(ctx context.Context) (map[string]bool, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT upload_path FROM imports WHERE upload_path IS NOT NULL AND status IN ('queued', 'processing')`)
	paths, err := collect(rows, err, func(row scanner) (string, error) {
		var p string
		err := row.Scan(&p)
		return p, err
	})
	if err != nil {
		return nil, err
	}
	set := map[string]bool{}
	for _, p := range paths {
		set[p] = true
	}
	return set, nil
}

// PruneImports deletes finished jobs older than age.
func (s *Store) PruneImports(ctx context.Context, age time.Duration) error {
	_, err := s.db.ExecContext(ctx,
		`DELETE FROM imports WHERE status IN ('done', 'failed') AND updated_at < datetime('now', ?)`,
		fmt.Sprintf("-%d seconds", int(age.Seconds())))
	return err
}
