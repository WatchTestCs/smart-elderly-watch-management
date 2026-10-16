## What changed

<!-- Describe the user-visible or engineering behavior, not only the files changed. -->

## Why

<!-- State the problem and why this is the smallest safe solution. -->

## Verification

- [ ] Preview CI is green.
- [ ] I ran or reviewed the tests relevant to this change.
- [ ] I did not weaken an architecture invariant without updating ARCHITECTURE.md / an ADR.

### UI/UX changes

- [ ] Not applicable.
- [ ] UI Quality is green.
- [ ] Desktop, narrow/mobile, keyboard and large-text behavior were considered.
- [ ] Loading / empty / error / long-content / destructive states remain understandable.
- [ ] A visual baseline was changed only because the visual change is intentional.

### Data / security / desktop changes

- [ ] Not applicable.
- [ ] User data under `%APPDATA%\岁宁` is preserved unless the user explicitly requests deletion.
- [ ] Electron Renderer still has no Node.js access and external navigation remains denied by default.
- [ ] Authentication / authorization / CSRF effects are covered by tests.
- [ ] Build/sign/release boundaries remain separated.

## Evidence and remaining limits

<!-- Link workflow runs, screenshots, logs or test output. List anything not verified. -->
