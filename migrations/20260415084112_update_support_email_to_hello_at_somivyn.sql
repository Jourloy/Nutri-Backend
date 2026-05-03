-- Updates already persisted translation values to the new support e-mail.

UPDATE translations
SET
	value = REPLACE(value, 'hello@nutri02.com', 'hello@nutri02.com'),
	updated_at = NOW()
WHERE deleted_at IS NULL
	AND value LIKE '%hello@nutri02.com%';
