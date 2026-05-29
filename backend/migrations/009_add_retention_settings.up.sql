INSERT INTO system_settings (key, value) VALUES ('audit_retention_days', '90'), ('login_history_retention_days', '90') ON CONFLICT (key) DO NOTHING;
