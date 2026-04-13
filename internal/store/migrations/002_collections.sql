-- Collections and many-to-many membership (FR-18, FR-20).
-- display_date: calendar date as ISO TEXT 'YYYY-MM-DD' (not capture timestamp).
CREATE TABLE IF NOT EXISTS collections (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	name TEXT NOT NULL,
	display_date TEXT NOT NULL,
	created_at_unix INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS asset_collections (
	asset_id INTEGER NOT NULL,
	collection_id INTEGER NOT NULL,
	created_at_unix INTEGER NOT NULL,
	PRIMARY KEY (asset_id, collection_id),
	FOREIGN KEY (asset_id) REFERENCES assets (id),
	FOREIGN KEY (collection_id) REFERENCES collections (id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_asset_collections_collection ON asset_collections (collection_id);

CREATE INDEX IF NOT EXISTS idx_asset_collections_asset ON asset_collections (asset_id);
