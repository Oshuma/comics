-- Background import jobs for uploads and archive.org downloads. The comic is
-- only created once a job succeeds.
CREATE TABLE imports (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    group_id INTEGER NOT NULL REFERENCES groups (id) ON DELETE CASCADE,
    -- Becomes the comic's filename.
    filename TEXT NOT NULL,
    -- 'upload' or 'archive.org'.
    source TEXT NOT NULL,
    -- Uploads: the spooled archive waiting to be processed.
    upload_path TEXT,
    -- archive.org: the file to download.
    archive_identifier TEXT,
    archive_name TEXT,
    -- 'queued', 'processing', 'done' or 'failed'.
    status TEXT NOT NULL DEFAULT 'queued',
    error TEXT,
    comic_id INTEGER REFERENCES comics (id) ON DELETE SET NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX imports_user_id ON imports (user_id);
CREATE INDEX imports_status ON imports (status, id);
