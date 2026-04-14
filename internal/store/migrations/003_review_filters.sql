-- Review filters (Story 2.2): star rating for min-rating filter and Story 2.3 badges.
ALTER TABLE assets ADD COLUMN rating INTEGER
	CHECK (rating IS NULL OR (rating >= 1 AND rating <= 5));

CREATE INDEX IF NOT EXISTS idx_assets_rating ON assets (rating);
