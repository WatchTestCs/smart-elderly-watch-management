# Go-native backend audit — preview (2026-10-28)

## Verified implementation boundary

The current `preview` backend is a Go module (`careold-go`) built from `cmd/server` using Go's standard `net/http` and `database/sql` (MySQL primary) plus `pgx` (optional PostgreSQL analytics secondary). `deploy/Dockerfile` uses a Go builder and emits a CGO-disabled Go binary. Its runtime image invokes `/app/suining`; no JRE/JDK/Maven or Java process is part of this path.

A recursive inspection of the current 997-file `preview` tree found **no** Java sources, class files, JAR/WAR, Gradle, Maven wrapper or pom.xml. Searching GitHub without a branch filter can return Java sources from older branches or commits; that does not establish their presence on `preview`.

New `internal/runtimeaudit` tests fail CI if a JVM artifact or Java build configuration enters the Go backend, or if the deployed image stops compiling and invoking the Go binary.

## Contract migration implemented here

- HTTP browser sessions are created under `SUINING_SESSION`, not `JSESSIONID`.
- Existing **valid** predecessor sessions are accepted only for a transition and are reissued under the new cookie name at the authenticated session endpoint. The underlying server-side session and CSRF token do not rotate during that migration.
- A modern session cookie, even an invalid one, takes precedence over the predecessor cookie. No silent downgrade is allowed when both are supplied.
- Login rotation and logout invalidate both browser-session IDs; new responses expire the old browser cookie. Cookie attributes remain HttpOnly, SameSite=Lax and Secure on TLS-enabled deployments.
- Vue's login, setup login, session lookup and logout use the already-supported `/api/v1/auth/*` aliases, preserving the same Go middleware and authentication checks.
- The Go response writer no longer mutates caller-owned response maps. Its legacy JSON envelope remains unchanged for currently deployed clients.

## Remaining wire compatibility, NOT Java runtime dependencies

- `/restful/*` paths and `returnCode` values are used by the watch firmware and current Vue pages. Removing them is an explicit, coordinated client/protocol version migration, not a necessary step to run a native Go backend.
- `JSESSIONID` is *read only* during the compatibility phase and is never issued as a live cookie. Once outstanding browsers have migrated or expired, the fallback can be removed in a later API release.
- The module and directory names `careold-go` and `oldIot` are historical repository identifiers. They neither run Java nor require it.
- This claim is about **the backend**. The desktop uses Rust/Tauri and the UI uses TypeScript/Vue, so the entire application is not literally written only in Go.

## Scope and verification

No database migrations, SOS delivery changes, MySQL/PostgreSQL role changes, hardware payload changes, or new Java binaries are introduced. Verify with the full Go 1.26/1.27 CI matrix, real MySQL/PostgreSQL integration tests, Go race/build/vet/gofmt, frontend Vitest/typecheck, static device protocol tests, and deployment/security workflows. A complete removal of legacy HTTP schemas requires a planned firmware and frontend cutover with monitoring and rollbacks.

## CodeQL security finding fixed

A PR CodeQL annotation identified `securestore/box.go`'s prior nil-Box SHA-256 blind-index fallback as high risk for low-entropy values. `Box.Index` now requires a configured keyed HMAC and returns an error otherwise. Login audit, account/IP brute-force controls and care-notification recipients explicitly reject missing keys; no new plaintext/digest fallback is emitted. Historical plaintext notification records may still be read during data migration, but newly encrypted recipient writes require the configured key. Test fixtures create a dedicated ephemeral test-only key rather than relying on the former insecure nil fallback.

The baseline CodeQL comparison retains an obsolete `/language:java-kotlin` category from earlier history even though `security.yml` on `preview` already scans Go and JavaScript/TypeScript only. This is scanning metadata, not evidence of Java runtime in the current branch.
