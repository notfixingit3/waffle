DELETE FROM system_settings WHERE key IN ('jwt_expiration_hours', 'password_min_length');

DROP INDEX IF EXISTS idx_audit_log_created_at;
DROP INDEX IF EXISTS idx_audit_log_action_created_at;
DROP INDEX IF EXISTS idx_audit_log_admin_id_created_at;

ALTER TABLE admins DROP COLUMN IF EXISTS last_login_ip;

DROP TABLE IF EXISTS audit_log;
