-- Story 2.5: normalized tags + asset_tags (FR-07, FR-15).
--
-- Referential integrity (MVP):
--   asset_tags.asset_id → assets(id) ON DELETE CASCADE
--     Removing an asset row removes its tag links (future hard-delete / quarantine finalization).
--   asset_tags.tag_id → tags(id) ON DELETE RESTRICT
--     No tag-delete API in MVP; unlink rows are deleted explicitly. Prevents accidental mass wipe.
--
-- Label uniqueness: UNIQUE(label COLLATE NOCASE) folds ASCII case only (SQLite NOCASE);
-- full Unicode case-folding is out of scope for MVP.

CREATE TABLE IF NOT EXISTS tags (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	label TEXT NOT NULL UNIQUE COLLATE NOCASE
);

CREATE TABLE IF NOT EXISTS asset_tags (
	asset_id INTEGER NOT NULL,
	tag_id INTEGER NOT NULL,
	PRIMARY KEY (asset_id, tag_id),
	FOREIGN KEY (asset_id) REFERENCES assets(id) ON DELETE CASCADE,
	FOREIGN KEY (tag_id) REFERENCES tags(id) ON DELETE RESTRICT
);

CREATE INDEX IF NOT EXISTS idx_asset_tags_tag_id ON asset_tags (tag_id);
CREATE INDEX IF NOT EXISTS idx_asset_tags_asset_id ON asset_tags (asset_id);
