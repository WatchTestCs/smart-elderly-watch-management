# Suining-Core

Rust analytics accelerator for Suining. MySQL remains the only authoritative
store for business data, sessions, audit, blobs, transactions and recovery.
This crate contains no SQL client, persistent store, network listener or
authorization implementation. Its projections are disposable.

## Initial implementation

`HeartRateSeries` is an immutable structure-of-arrays projection of one
resident/source/device's instantaneous heart-rate series. It accepts at most
10,000 eligible observations, retains the original revision order, and uses two
binary searches to locate a half-open time window. Querying performs no heap
allocation. An empty covered range differs from a cache miss. Unsupported
inputs, out-of-order timestamps, unknown revisions and windows outside the
snapshot's coverage are not eligible.

The projection cannot establish that the data it receives is complete or
authorized. The Go adapter must establish that before admission. In particular:

- Resolve the resident and run current MySQL authorization on every request,
  including hits. Keep the existing API's source, metric and semantics filters,
  candidate limit, exact timestamp boundaries and error behavior.
- Obtain a strictly positive committed `health_owner_clock` revision and
  read a consistent snapshot; revalidate that revision before publishing it.
  Reject incomplete snapshots, mixed units or identities, legacy time basis,
  candidate-count discrepancies and unsupported timestamp ranges. Never sort
  values into a different floating-point accumulation order.
- Serve only the same identity, revision and a fully covered window. Never
  return a cache hit when MySQL validation fails. Do not cache errors.
- Bound total projection memory, evict cold entries, and use the existing Go/MySQL
  implementation when admission or lookup fails. No user write is replayed.
- Keep native binaries under the existing build integrity and release gates.
  Do not load arbitrary libraries from a writable search path.

The initial crate is not wired into the HTTP service or desktop packaging.
There is no cross-language ABI, production cache, scheduler, SIMD kernel,
durable queue or new database. The integration requires Go/Rust differential
tests, isolated MySQL 8.4 tests of concurrent correction/delete/revocation,
resource limits, and end-to-end measurements before activation.

## Verification

From this directory:

```powershell
cargo fmt --check
cargo test --locked --offline
cargo clippy --locked --offline --all-targets -- -D warnings
cargo run --locked --offline --release --example range_bench
```

The benchmark compares a full scan of the same 4,320 synthetic observations
with an already-built in-memory index. It excludes index construction,
authorization, MySQL, decryption and Go/Rust boundary costs. It is not a
claim about HTTP throughput.
