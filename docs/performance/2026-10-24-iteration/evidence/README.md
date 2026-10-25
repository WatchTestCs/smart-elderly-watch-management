# Reproducible performance evidence

All records are synthetic. These files retain favorable and unfavorable measurements from the original local experiment; they are not production traces or CI results.

- `http-runs.jsonl`: four complete fixture reports in A/B/B/A order, including every measured latency. The legacy report `commit` field is a fixed baseline identifier; the outer run label distinguishes variants. Final production source is the reviewed patch based on daac58a9, not that legacy field.
- `microbenchmarks.txt`: original CPU and final pagination output, grouped by exact run label. The CPU candidate was measured before the additional decoder dependency, but its health/rules production source is unchanged in the final implementation.
- `rejected-pagination.txt`: the first joined-query regression and the later rejected scalar COUNT comparison, including all measured cases.
- `decoder-experiment.txt`: original library comparison and compatibility results, including the rejected Sonic signed-zero failure.
- `source-hashes.json`: SHA-256 of original local source reports/logs before JSONL packing or grouped text concatenation.

The complete fixture for the real HTTP comparison is committed as `backend/internal/handlers/read_latency_ab_test.go` under the application directory. It imports only production APIs and existing test helpers available at the pinned baseline, so the same file can be copied unchanged to the baseline checkout. The current fixture records caller variant and VCS build settings instead of the legacy fixed commit field; measurement loops and response assertions are unchanged.

See the parent README for the environment, commands and limitations. Raw legacy execution logs contained in the earlier local delivery bundle remain local; this directory contains the data needed to audit the published performance tables.

`production-source-hashes.json` verifies that the reviewed production Go/module files are byte-identical to the measured implementation; review fixes only change tests and documentation.
