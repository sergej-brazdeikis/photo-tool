CREATE TABLE IF NOT EXISTS schema_meta (
	singleton INTEGER PRIMARY KEY CHECK (singleton = 1),
	version INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS assets (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	content_hash TEXT NOT NULL,
	rel_path TEXT NOT NULL,
	capture_time_unix INTEGER NOT NULL,
	width INTEGER,
	height INTEGER,
	mime TEXT,
	rejected INTEGER NOT NULL DEFAULT 0 CHECK (rejected IN (0, 1)),
	rejected_at_unix INTEGER,
	deleted_at_unix INTEGER,
	created_at_unix INTEGER NOT NULL
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_assets_content_hash ON assets (content_hash);

CREATE UNIQUE INDEX IF NOT EXISTS idx_assets_rel_path_active
ON assets (rel_path)
WHERE deleted_at_unix IS NULL;

CREATE INDEX IF NOT EXISTS idx_assets_capture ON assets (capture_time_unix);

CREATE INDEX IF NOT EXISTS idx_assets_rejected ON assets (rejected);
