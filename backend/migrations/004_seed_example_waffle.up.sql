DO $$
DECLARE
    wid UUID;
BEGIN
    -- Insert the example waffle, idempotent via slug conflict
    INSERT INTO waffles (slug, title, description, total_spots, spot_price, status, instagram_media_links, created_at)
    VALUES (
        'welcome-example-waffle',
        'Example Waffle',
        'Welcome! This is an example waffle showing how spots work. Feel free to delete it and create your own.',
        10,
        500,
        'active',
        '[]'::jsonb,
        NOW()
    )
    ON CONFLICT (slug) DO NOTHING
    RETURNING id INTO wid;

    -- If already existed, get the existing ID
    IF wid IS NULL THEN
        SELECT id INTO wid FROM waffles WHERE slug = 'welcome-example-waffle';
    END IF;

    -- Insert 10 available spots if they don't already exist (idempotent)
    INSERT INTO spots (waffle_id, number, status)
    SELECT wid, n, 'available'
    FROM generate_series(1, 10) AS n
    WHERE NOT EXISTS (
        SELECT 1 FROM spots s WHERE s.waffle_id = wid AND s.number = n
    );
END $$;
