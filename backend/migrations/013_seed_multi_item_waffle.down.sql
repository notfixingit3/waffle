DELETE FROM spots WHERE waffle_id = (SELECT id FROM waffles WHERE slug = 'multi-item-example-waffle');
DELETE FROM waffles WHERE slug = 'multi-item-example-waffle';
