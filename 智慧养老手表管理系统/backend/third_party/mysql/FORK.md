# Bounded parameter scratch patch

This is go-sql-driver/mysql v1.9.3, retained under its original MPL-2.0 license.
The original top-level source, tests and metadata are retained. Upstream GitHub
workflow metadata is omitted. UPSTREAM.json records original and current LF
source fingerprints and the authenticated Go module sums.

Only connection.go and utils.go change production behavior. A physical connection
reuses at most 4,096 driver.Value slots (64 KiB of slots on amd64). Go's
database/sql/driver.Conn contract serializes calls on a connection. The four
context execution/query methods clear borrowed references before returning,
including named-argument errors, cancellation and successful live-row returns.
No connection-global cleanup is added because cancellation cleanup can run
concurrently. Larger argument lists retain the original allocation path.

The SQL encoder, type converter, escaping, authentication, cancellation watcher,
statement/result protocol, transaction handling and retry behavior are unchanged.
The new named_values_scratch_test.go exercises the reference conversion, bounded
retention, all four early-return paths, separate connections, live rows, binary
parameters and cancellation with 2,800 real prepared parameters.
That real acceptance test fails when its explicit test address/database are
missing or unreachable; it cannot silently inherit the upstream availability skip.

The backend keeps v1.9.3 in its requirements and uses a local replace. Because
govulncheck follows replacement module identities, the existing current-source
scan is supplemented by scripts/performance/check-mysql-driver.py: it checks the
reviewed file inventory/fingerprints and actually scans the pinned, unreplaced
upstream package with govulncheck. Both scans fail closed. The backend CI runs the
new real MySQL scratch contracts separately because this is a nested Go module.

Upstream's complete suite has a pre-existing local failure:
TestConnectorReturnsTimeout directly dials 1.1.1.1:1234 expecting a 10 ms dial
timeout. This host instead connects and later panics in handshake logging with
the test's nil Logger. Both the original module and this copy reproduce it.
No test or assertion is removed or relaxed. Windows also retains upstream's
existing TestContextCancelBegin platform skip. Targeted real MySQL/race checks
and backend acceptance are reported separately from that failed full suite.

Changes, measurements, raw outputs and limitations are recorded in the
watch-sim driver scratch record; that evidence is not distributed with this
repository and must be requested separately.
Do not update these files without refreshing UPSTREAM.json, preserving the
upstream package scan, and re-running the real database/lifetime regressions.
