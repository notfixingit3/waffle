DROP INDEX IF EXISTS idx_waffles_archived;
ALTER TABLE waffles DROP COLUMN IF EXISTS instagram_media_links;
ALTER TABLE waffles DROP COLUMN IF EXISTS archived;
