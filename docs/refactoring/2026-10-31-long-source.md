# Long source decomposition

Baseline: `3bd4375d5aafab32948f145f7535d5ef6670ef67`.

## Scope and structure

The production-source inventory identified eight Go files above 350 lines and
four Vue pages above 300 lines. These are review thresholds, not style limits.
Tests, dependencies and generated build outputs are outside this inventory.

The eight Go files now separate routes, queries, mutations, profile credentials,
telemetry parsing, SOS, notification enqueueing, delivery, scheduling and care
dispatch. The original packages and exported interfaces are retained.

Two long orchestration functions have also been decomposed:

- `rules.Engine.Device`: lock the device, read policy, collect observations and
  baselines, evaluate and persist candidates, upgrade night incidents, commit.
  Input collection and the night upgrade use the same transaction as before.
- `notify.Worker.Drain`: claim and commit a bounded batch, deliver external
  messages, persist results guarded by the original lease token. Provider calls
  still start only after the claim commits. Retry order and concurrency are unchanged.

The person, family and threshold pages delegate create/edit forms to record
components. Pages retain drafts, requests, dialogs and refresh ownership. Form
components preserve validation callbacks and reset behavior through parent refs.
Resident writes and health alert methods follow the existing Options API method
module pattern. The health alert dialog emits the original resolution kind.

Following the redundant-source cleanup request, create/edit fields share three
domain components rather than six duplicated forms. Explicit `editing` props retain
hidden identifiers, create-only elder phone input, read-only bound devices and
disabled threshold device identifiers. Six duplicate component files are removed.
An unreferenced `nullableCoordinate` helper and orphaned/repeated comments are also
removed. Frontend import reachability found no unused production source file;
the images and font resources reviewed still have live references.

OpenAPI regeneration changes only 12 implementation-file descriptions following
the capability route move. HTTP paths, methods, authentication and DTOs are unchanged.
There are no database migrations or dependency updates in this batch.

## Verification

- AST comparison: 82 of the 84 declarations in the eight original Go files remain
  identical, ignoring positions/comments. The two intentional exceptions are
  `Engine.Device` and `Worker.Drain`, whose extracted steps received independent
  behavior review. The earlier migration-only intermediate state matched all 84.
- `go vet` passed for handlers, notification, care and rules packages.
- Seven existing MySQL integration regressions passed for short notification
  deadlines, stale leases, bulk fallback, retry ordering, baseline recovery,
  concurrent offline deduplication and idempotent night upgrades.
- Four existing handler regressions passed for guardian integrity, resident edit,
  signed telemetry replay and atomic SOS rollback.
- Frontend type checking and production build passed. The existing large-bundle
  advisory remains; this source-structure change does not optimize bundle size.
- Existing Chromium forms acceptance: 30 checks passed. Eight focused production
  bundle checks also passed for validation/reset, threshold limits, alert display
  and resolution identity. HTTP responses in these browser checks were mocked.
- OpenAPI consistency and native API boundary checks passed.
- Independent standards and behavior reviews found no blocking issue.

PR #55 completed 26 checks (23 successful, two neutral, one skipped). Two
post-CI MySQL regressions and eight component-path checks passed. It was merged
as `77b1cf487b1c8ccf026620e04296326157c2ac18`; the merge tree matched the
tested head `6675f05c748cc3935cb38d3867095ffd5e13c81f` exactly.

## Second batch: normalization, aggregation and statistics reads

The next batch separates four additional long functions into domain steps:

- `Sample.Normalize`: record validation, metric/unit contract, and measurement
  normalization. Private helpers operate on the public method's local copy;
  partial normalized values returned with errors remain unchanged. A duplicate
  interval-duration guard is removed: its earlier identical check already rejects
  invalid intervals, and the time fields cannot change between those checks.
- `Aggregate`: sample selection remains in the entry point, while interval step
  allocation and total reduction have named helpers. Sorting, accumulation order,
  overflow checks and partial statistics on errors are retained.
- `healthStatistics`: request-range validation and bounded SQL sample loading
  have separate helpers. Authorization precedes cache reads; the revision checks,
  candidate limit, byte-exact source predicates and window filtering are retained.
  The named request type preserves the original JSON field order for cache keys.
- `appendEvidenceFast`: eligibility checks move to the evidence value, with a
  pointer receiver to avoid copying the candidate. The encoding sequence remains
  together, including byte order, canonical fallbacks and borrowed-buffer lifetime.

Ten distinct existing regressions passed across normalization, aggregation,
evidence encoding, MySQL cache identity/consistency and submicrosecond windows.
Affected-package vet and OpenAPI consistency passed. Independent review found no
behavior issue; its suggestion to avoid new Sample value copies was applied and
the resulting local-copy/error semantics were reviewed again.

A bounded baseline/candidate benchmark of three changed hot paths found the same
allocation counts: evidence buffer 1, 1000 interval samples 16, and 1000 daily
samples 26 allocations per operation. These short measurements do not establish
a speed improvement. No frontend work or dependency changes are included here.

PR #56 completed 25 checks (23 successful, one neutral, one skipped). Three
post-CI checks passed for MySQL cache authorization, normalization and byte-exact
evidence. It was merged as `ab0699421684781f23f6ec979b7b04a1504912b9`, with the
same tree as tested head `718b2279c23f2f5c7a4521d4dbb5c73bdbc0c188`.

## Dead-code cleanup

Production-call searches and the complete command dependency graph identified
retired adapters alongside live test infrastructure. This cleanup removes the
production `Evaluate`, `marshalEvidenceFast`, `dispatchOne`, `hitWindow.allow`,
`Enqueue`, `SyncOne` and fragment-cache `valueRows` entry points. Their ordinary
regressions now call the active compact evaluator, buffer encoder, dispatcher,
budget limiter, channel enqueueing and batch projection paths directly.

The independent map-shaped rule oracle, single-event projection implementation
and fragment cache remain only in existing reference/A-B test files where they
provide comparison baselines. No new test files or test suites are introduced.
Cross-package test infrastructure (`db.Postgres`, `dbtest.MySQL`, `health.Save`)
is retained: those consumers are valid dependencies, not dead references.

`health/batch_sql.go` and its retired-cache test file are deleted. The live
`buildValueRows` implementation moves unchanged beside its sole production caller
in `batch_statements.go`. Its literal and maximum-size boundary assertions remain
in `TestBuildValueRows`. The two offline benchmark source lists and file indexes
are updated to remove references to deleted files; historical result manifests
retain their original source hashes.

All production commands build. Eleven distinct focused regressions passed for
rule/evidence equivalence, SQL construction, dual-database recovery/replay,
notification rollback, stale notification leases and sliding-window boundaries.
Affected-package vet, OpenAPI consistency and updated Python/index parsing passed.
Independent review caught omitted SQL-builder boundary assertions; those were
restored and rechecked. No review issue remains open.

Frontend entry reachability still accounts for all 72 production source files;
the reviewed images, fonts and font-check fixtures have live consumers. This pass
does not delete referenced resources merely because they are outside the runtime
entry point. PR #57 completed 25 checks (23 successful, one neutral, one skipped)
and three post-CI smoke checks. It was merged as
`fadd93416d1609a19163fa99a1b6fdab05faa216`, matching the tested head's tree.

## Handler decomposition

Six more initially inventoried functions now separate their domain steps inside
the existing source files:

- Device registration delegates defaults and input validation to its parser.
- Family creation delegates the locked, at-most-two-elders read to a target
  loader. It still inserts the account before that read and rolls back on zero,
  ambiguous or already-bound matches.
- Invitation registration parses and validates one input record, then delegates
  account creation/activation and conditional elder linking to one persistence
  step. The parent retains the original locks, password hashing, invitation
  consumption, failure responses and transaction commit/rollback ownership.
- Invitation generation separates expiration calculation and bounded uniqueness
  retries. Elder existence is still checked before expiration parsing.
- Device listing delegates row projection, coordinate decryption and online
  calculation, using the same captured timestamp for every row.
- Vital-alert detail delegates response presentation after the unchanged lookup.

Seven focused MySQL regressions passed: existing family onboarding, overwrite
rejection, ambiguous elder phone, concurrent invitation registration, invitation
expiry and configured threshold explanation tests, plus one registration/list
contract test added to the existing device validation test file. The latter covers
defaults, owner/profile linkage, stored-secret separation, filtered online counts,
decrypted coordinates and rejected input without extra records. No new source or
test file is added. Handler vet, OpenAPI consistency and independent behavior and
standards review passed. This batch still requires final-head CI and smoke.

## Remaining scope

This batch addresses all 12 inventoried long files, and leaves no file above the
inventory thresholds. It is not a claim that all long functions are resolved.
The initial function inventory contained 30 functions of at least 90 lines.
After the batches above, 18 original functions remain for responsibility review.
The 94-line evidence encoding sequence has been reviewed and retained as one
cohesive operation after extracting eligibility checks; the review threshold is
not a requirement to fragment a single encoding sequence. The decomposed 124-line
invitation registration and 91-line device registration retain their transaction
coordinator bodies to keep failure, commit and rollback ordering explicit. Next priorities are SOS and device binding,
legacy transfer, alert resolution and telemetry persistence. Their transaction
boundaries must remain intact while validation and business steps are separated.
