ALTER TABLE waffles ADD COLUMN IF NOT EXISTS item_count INTEGER NOT NULL DEFAULT 1;
ALTER TABLE waffles ADD COLUMN IF NOT EXISTS winning_spot_numbers JSONB DEFAULT '[]';
ALTER TABLE waffles ADD COLUMN IF NOT EXISTS winning_instagram_handles JSONB DEFAULT '[]';

-- Backfill existing winners into the new JSONB arrays
UPDATE waffles 
SET winning_spot_numbers = jsonb_build_array(winning_spot_number),
    winning_instagram_handles = jsonb_build_array(winning_instagram_handle)
WHERE winning_spot_number IS NOT NULL;
