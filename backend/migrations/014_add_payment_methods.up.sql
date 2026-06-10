CREATE TABLE IF NOT EXISTS payment_methods (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    type TEXT NOT NULL CHECK (type IN ('venmo', 'paypal', 'cashapp', 'zelle')),
    display_name TEXT NOT NULL,
    handle_or_url TEXT NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS waffle_payment_methods (
    waffle_id UUID NOT NULL REFERENCES waffles(id) ON DELETE CASCADE,
    payment_method_id UUID NOT NULL REFERENCES payment_methods(id) ON DELETE RESTRICT,
    PRIMARY KEY (waffle_id, payment_method_id)
);

CREATE INDEX IF NOT EXISTS idx_waffle_payment_methods_waffle_id ON waffle_payment_methods(waffle_id);