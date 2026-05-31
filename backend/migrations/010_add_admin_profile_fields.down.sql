UPDATE admins SET email = username || '@localhost' WHERE email IS NULL;
ALTER TABLE admins ALTER COLUMN email SET NOT NULL;
DROP INDEX IF EXISTS admins_email_key;
CREATE UNIQUE INDEX admins_email_key ON admins (email);
ALTER TABLE admins DROP COLUMN IF EXISTS social_links;
ALTER TABLE admins DROP COLUMN IF EXISTS last_name;
ALTER TABLE admins DROP COLUMN IF EXISTS first_name;
