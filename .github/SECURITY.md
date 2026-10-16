# Security Policy

## Supported branch

Active engineering and preview hardening currently happen on `preview`. Published preview binaries must remain traceable to an exact source commit.

## Reporting a vulnerability

Do not open a public issue containing an exploitable vulnerability, real user records, credentials, database files, private keys, SMS credentials, device secrets or personally identifying health data.

Prefer GitHub's private security-advisory reporting channel for this repository. If private reporting is unavailable, contact a repository maintainer privately and share only the minimum information needed to reproduce the issue.

A useful report includes:

- affected source commit / preview version;
- affected component (Renderer, Electron Main, Spring service, installer, device API, etc.);
- prerequisites and realistic impact;
- minimal reproduction steps;
- whether the issue crosses a documented trust boundary;
- suggested mitigation if known.

## Security boundaries that must not be weakened silently

- Electron Renderer has no Node.js access.
- `contextIsolation`, sandbox and web security remain enabled.
- external navigation, new windows, WebView, downloads and permission requests are denied by default.
- the Spring service binds to loopback by default.
- the desktop bootstrap token is not exposed to Renderer JavaScript or localStorage.
- business authorization remains server-side; the desktop token is not a substitute for user authorization.
- user data is not stored in the installation directory.
- ordinary PR/CI jobs do not receive signing private keys.

See `docs/ARCHITECTURE.md` and `docs/decisions/` for the full invariants.

## Scope notes

The project has automated backend, UI/a11y, build-contract and static-security checks, but these are not a claim of complete security. Real-watch integration, real SMS, formal performance testing and a complete independent penetration test remain outside the currently published evidence.
