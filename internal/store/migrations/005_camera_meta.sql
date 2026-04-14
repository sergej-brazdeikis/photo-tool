-- Story 2.8: minimal FR-26 hook for collection grouping by camera (nullable EXIF fields + canonical label).
ALTER TABLE assets ADD COLUMN camera_make TEXT;
ALTER TABLE assets ADD COLUMN camera_model TEXT;
ALTER TABLE assets ADD COLUMN camera_label TEXT;
