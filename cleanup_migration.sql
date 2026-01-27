-- Remove failed migration record
DELETE FROM migrations WHERE name = '20260128120003_add_supplement_callback_mappings.sql';

-- Try to drop the table if it was partially created
DROP TABLE IF EXISTS supplement_callback_mappings;
