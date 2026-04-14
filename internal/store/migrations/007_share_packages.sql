-- Story 4.1: multi-asset package shares — link_kind, nullable package parent asset_id, member rows.
CREATE TABLE share_links_new (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	token_hash TEXT NOT NULL,
	asset_id INTEGER REFERENCES assets (id),
	created_at_unix INTEGER NOT NULL,
	payload TEXT,
	link_kind TEXT NOT NULL DEFAULT 'single' CHECK (link_kind IN ('single', 'package')),
	UNIQUE (token_hash),
	CHECK (
		(link_kind = 'single' AND asset_id IS NOT NULL)
		OR (link_kind = 'package')
	)
);

INSERT INTO share_links_new (id, token_hash, asset_id, created_at_unix, payload, link_kind)
SELECT id, token_hash, asset_id, created_at_unix, payload, 'single' FROM share_links;

DROP TABLE share_links;

ALTER TABLE share_links_new RENAME TO share_links;

CREATE INDEX IF NOT EXISTS idx_share_links_asset_id ON share_links (asset_id);
CREATE INDEX IF NOT EXISTS idx_share_links_link_kind ON share_links (link_kind);

CREATE TABLE share_link_members (
	share_link_id INTEGER NOT NULL REFERENCES share_links (id) ON DELETE CASCADE,
	position INTEGER NOT NULL,
	asset_id INTEGER NOT NULL REFERENCES assets (id),
	PRIMARY KEY (share_link_id, position)
);

CREATE INDEX IF NOT EXISTS idx_share_link_members_asset_id ON share_link_members (asset_id);
