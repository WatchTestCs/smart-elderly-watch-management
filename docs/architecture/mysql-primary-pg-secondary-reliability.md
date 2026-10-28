# MySQL primary / PostgreSQL analytics-secondary isolation

MySQL remains the only business authority. Device, SOS, authentication and care transactions commit to MySQL first; MySQL triggers persist changes in `analytics_outbox` as part of the same business transaction. PostgreSQL is a disposable projection and never participates in synchronous device acknowledgements.

## Reliability improvements

- When `PG_ANALYTICS_URL` is absent, the analytics worker returns immediately and does not issue unnecessary `COUNT(*)` queries against MySQL.
- An enabled PG pool is capped at two connections, zero minimum idle connections, and a three-second connection timeout. It is not allowed to scale with the MySQL business pool.
- Consecutive secondary failures use bounded retries (2, 4, 8, 16, then 30 seconds), preventing repeated connection storms. Successful PG cycles reset the delay immediately; saturated backlogs still yield for only 10 milliseconds between full drains.
- MySQL outbox records are only acknowledged after a successful PostgreSQL projection write, using the existing idempotent event-ID check. A secondary outage leaves events durable in MySQL.
- No dual-primary configuration, cross-database transaction, schema migration, or device-protocol change is introduced.

## Verification

The new tests cover retry-delay boundaries, PG connection limits, the no-secondary fast path and a real MySQL commit with PostgreSQL unreachable. The existing end-to-end PG/MySQL tests also cover batch coalescing, tombstones, PG recovery/replay and 6,000-event backlog draining.

Operational limits: a long PostgreSQL outage still grows MySQL's durable Outbox; alert on queue age, storage utilization and last-successful-sync age. Connection caps and retries reduce load but cannot replace capacity planning or PostgreSQL recovery.
