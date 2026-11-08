-- Reverting the receipt store disables replay acknowledgements.
DROP TABLE IF EXISTS device_report_receipt;
DELETE FROM schema_migrations WHERE name='0019_report_receipts.sql';
