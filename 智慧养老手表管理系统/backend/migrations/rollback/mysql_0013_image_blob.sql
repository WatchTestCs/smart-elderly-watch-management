-- Preserve catalog/tombstones and image bytes, never erase live image history.
-- Old binaries cannot read S3 images: materialize local images before rollback.
DELETE FROM schema_migrations WHERE name='mysql/0013_image_blob.sql';
