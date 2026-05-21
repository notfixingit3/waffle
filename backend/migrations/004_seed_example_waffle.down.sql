DELETE FROM spots WHERE waffle_id = (SELECT id FROM waffles WHERE slug = 'welcome-example-waffle');
DELETE FROM waffles WHERE slug = 'welcome-example-waffle';
