-- Preserve receipts and binding history when reverting application code.
DELETE FROM schema_migrations WHERE name='mysql/0021_report_binding_receipts.sql';
