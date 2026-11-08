-- Offline cutover: stop every old-version writer before applying this change.
-- Revisions remain globally unique. health_clock is the reserved high-water mark.
CREATE TABLE IF NOT EXISTS health_owner_clock (
 resident_id bigint PRIMARY KEY,
 revision bigint NOT NULL,
 range_end bigint NOT NULL,
 CHECK(revision>=0 AND range_end>=revision AND range_end<=9007199254740991)
) ENGINE=InnoDB;
INSERT INTO health_owner_clock(resident_id,revision,range_end)
 SELECT o.resident_id,c.revision,c.revision FROM resident o CROSS JOIN health_clock c WHERE c.id=1
 ON DUPLICATE KEY UPDATE resident_id=health_owner_clock.resident_id;
CREATE INDEX idx_message_lease ON message_outbox(status,last_attempt_at);
CREATE INDEX idx_message_channel_ready ON message_outbox(status,channel,retry_at);
