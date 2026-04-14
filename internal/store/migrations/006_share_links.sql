-- Story 3.1: default share mint — snapshot row (token hash only, never raw token).
CREATE TABLE IF NOT EXISTS share_links (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	token_hash TEXT NOT NULL,
	asset_id INTEGER NOT NULL REFERENCES assets (id),
	created_at_unix INTEGER NOT NULL,
	payload TEXT,
	UNIQUE (token_hash)
);

CREATE INDEX IF NOT EXISTS idx_share_links_asset_id ON share_links (asset_id);
