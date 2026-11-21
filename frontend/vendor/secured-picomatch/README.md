# Secured picomatch

This development-only package replaces picomatch 4.0.7 for CVE-2026-112366. It comes from the MIT-licensed `@ryotsu/picomatch@4.0.7-a` release, with source at [3ab5a762](https://github.com/ryotsu-kankichi/secured-picomatch/tree/3ab5a7627d480259e3d9ca67fbc14c125d374f0e). The registry signature and release attestation were verified. The initial vendor import matched that source; the local security patch below is a deliberate deviation. The local manifest excludes the fork's test and release tooling.

The npm lockfile links to this checked-in package. GitHub's dependency review associates the registry alias with the original vulnerable package. The existing dependency review and high-severity npm audit still apply. Remove this directory and the override when the official package publishes a verified fix.

The local copy additionally replaces REGEX_REMOVE_BACKSLASH with a linear-time scanner and regression tests in rontend/tests/vendorPicomatch.test.ts. This intentionally differs from the attested upstream source; compare this patch before refreshing the vendored fork.
