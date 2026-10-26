# Reliability remediation plan (2026-10-26)

## Scope and authority

User request: resolve the handoff issues using plan, execution, refinement and acceptance.
The supplied handoff is a requirements/evidence document, not authority to contact people,
alter production, merge existing pull requests, or change repository protection silently.
User confirmed that validation should use an isolated local environment.

Starting preview: `41b0f1d8752d1cf023872b067af8655f7a2580fd`.
Historical handoff baseline: `50b9cc094002ca691244b7d504cdeedf117787dd`.
Keep results for different commits separate. Preserve the existing extracted directories.
Work in this independent checkout and an isolated database cluster, never the existing
MySQL Windows service. Never use real recipients or production records.

## Execution sequence

1. Record repository/PR/CI state and tool versions; save raw evidence with source SHA.
2. Integrate existing PR #27, #29 and #32 changes locally, resolve against current preview,
   and preserve their transaction, audit and notification semantics. Inspect #33 separately.
3. TASK-01: fail-closed current-source CI with real MySQL/PG, race/build/vet, migration
   replay/rollback, frontend test/typecheck/build, OpenAPI, Windows contract, security,
   explicit skipped-test accounting, and SHA-bound artifacts/release checks.
4. TASK-02/03: exercise receipts, retries, old-worker races, human role separation,
   independent monitoring, database/app/worker/gateway outages and actual restore.
5. TASK-04: bounded limiter admission, trusted-proxy parsing, cross-instance boundary,
   high-cardinality/256-client/IPv4/IPv6/NAT/forged-header regressions and memory evidence.
6. TASK-05: data classification based on integrated #32, key version/rotation/fail-closed
   design and executable rotation regressions. Do not claim full-database encryption.
7. TASK-06: assess image storage; implement authenticated local/object storage boundary,
   integrity, compensation/orphan handling, two-instance and restore checks where feasible.
8. TASK-07: prepare enforceable release/provenance and repository protection configuration;
   do not weaken checks or forge independent review/signatures.
9. Refine from failures, rerun affected checks, then produce HANDOFF_RESULT.md,
   validation-matrix.md, raw logs, environment manifest, rollback and production gates.

## Acceptance rules

- Tests bind to an immutable commit or a manifest identifying a dirty candidate tree.
- Missing database configuration or skipped database suites cannot count as passing.
- Durable MySQL commit precedes SOS acceptance. Gateway acceptance is not delivered;
  delivered is not acknowledged/resolved/verified. Duplicate signed receipts produce
  no duplicate event, and stale workers cannot overwrite delivery.
- Fault tests operate only on newly created disposable local processes/data directories.
- Monitoring remains independent of the app; local test receiver evidence is explicitly
  distinguished from a human receiving an external alert.
- Store failures and limitations alongside successes. No manual task-state editing,
  test deletion, audit suppression, mock-only DB validation or false recovery claims.
- Real hardware/provider/human response, external paging, production recovery and GitHub
  admin settings require evidence unavailable from an isolated local environment; mark
  these pending with concrete next steps, never accepted by assumption.

This plan was written before implementation or test execution.
