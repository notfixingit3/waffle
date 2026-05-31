ALTER TABLE admins ADD COLUMN first_name TEXT;
ALTER TABLE admins ADD COLUMN last_name TEXT;
ALTER TABLE admins ADD COLUMN social_links JSONB DEFAULT '[]'::jsonb;
ALTER TABLE admins ALTER COLUMN email DROP NOT NULL;
ALTER TABLE admins DROP CONSTRAINT IF EXISTS admins_email_key;
CREATE UNIQUE INDEX admins_email_key ON admins (email) WHERE email IS NOT NULL;
