CREATE TABLE IF NOT EXISTS message_templates (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    body TEXT NOT NULL,
    is_default BOOLEAN NOT NULL DEFAULT false,
    created_by UUID REFERENCES admins(id),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

ALTER TABLE waffles ADD COLUMN IF NOT EXISTS share_template_id UUID REFERENCES message_templates(id);
ALTER TABLE waffles ADD COLUMN IF NOT EXISTS share_message TEXT;

INSERT INTO message_templates (name, body, is_default) VALUES (
    'Default Hype Drop',
    E'🧇 NEW WAFFLE DROP 🧇\n\n{item}\n\n${price}/spot • {spots_left} of {total_spots} left\n\nClaim your spot 👇\n{url}',
    true
);
