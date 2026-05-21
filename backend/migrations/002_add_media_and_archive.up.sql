ALTER TABLE waffles ADD COLUMN IF NOT EXISTS instagram_media_links JSONB DEFAULT '[]';
ALTER TABLE waffles ADD COLUMN IF NOT EXISTS archived BOOLEAN NOT NULL DEFAULT FALSE;

CREATE INDEX IF NOT EXISTS idx_waffles_archived ON waffles(archived);
