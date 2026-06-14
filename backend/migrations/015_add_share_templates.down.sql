ALTER TABLE waffles DROP COLUMN IF EXISTS share_template_id;
ALTER TABLE waffles DROP COLUMN IF EXISTS share_message;

DROP TABLE IF EXISTS message_templates;
