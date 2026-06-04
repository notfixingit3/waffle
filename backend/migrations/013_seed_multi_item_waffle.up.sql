DO $$
DECLARE
    wid UUID;
BEGIN
    -- Insert the multi-item example waffle, idempotent via slug conflict
    INSERT INTO waffles (slug, title, description, total_spots, spot_price, status, instagram_media_links, item_count, created_at)
    VALUES (
        'multi-item-example-waffle',
        'Multi-Item Example Waffle',
        'Welcome! This is an example waffle with 3 items up for grabs, each with its own Instagram post.',
        10,
        1000,
        'active',
        '["https://www.instagram.com/p/C-example1/", "https://www.instagram.com/p/C-example2/", "https://www.instagram.com/p/C-example3/"]'::jsonb,
        3,
        NOW()
    )
    ON CONFLICT (slug) DO NOTHING
    RETURNING id INTO wid;

    -- If already existed, get the existing ID
    IF wid IS NULL THEN
        SELECT id INTO wid FROM waffles WHERE slug = 'multi-item-example-waffle';
    END IF;

    -- Insert 10 spots
    INSERT INTO spots (waffle_id, number, status, claimed_by_handle)
    SELECT wid, n, 
        CASE 
            WHEN n = 1 THEN 'paid'
            WHEN n = 2 THEN 'pending'
            WHEN n = 3 THEN 'paid'
            WHEN n = 4 THEN 'pending'
            WHEN n = 5 THEN 'paid'
            ELSE 'available'
        END,
        CASE 
            WHEN n IN (1, 2, 3) THEN 'buyer_dani'
            WHEN n IN (4, 5) THEN 'buyer_boo'
            ELSE NULL
        END
    FROM generate_series(1, 10) AS n
    WHERE NOT EXISTS (
        SELECT 1 FROM spots s WHERE s.waffle_id = wid AND s.number = n
    );
END $$;
