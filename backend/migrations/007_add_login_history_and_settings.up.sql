CREATE TABLE IF NOT EXISTS login_history (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    admin_id UUID NOT NULL REFERENCES admins(id) ON DELETE CASCADE,
    ip_address TEXT NOT NULL DEFAULT '',
    user_agent TEXT DEFAULT '',
    browser TEXT DEFAULT '',
    os TEXT DEFAULT '',
    device_type TEXT NOT NULL DEFAULT 'unknown' CHECK (device_type IN ('mobile', 'desktop', 'tablet', 'unknown')),
    ip_org TEXT NOT NULL DEFAULT '',
    ip_country TEXT NOT NULL DEFAULT '',
    ip_city TEXT NOT NULL DEFAULT '',
    ip_asn TEXT NOT NULL DEFAULT '',
    whois_server TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS system_settings (
    key TEXT PRIMARY KEY,
    value TEXT NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_by UUID REFERENCES admins(id)
);

CREATE INDEX IF NOT EXISTS idx_login_history_admin_id_created_at ON login_history(admin_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_login_history_created_at ON login_history(created_at DESC);

INSERT INTO system_settings (key, value) VALUES ('whois_server', 'whois.pwhois.org')
ON CONFLICT (key) DO NOTHING;
