# T08 Code Review

## Findings

1. [high] `internal/runner/execution_host.go` did not parse `user@[ipv6]:port`.
   - Impact: valid SSH worker host values using a user-prefixed bracketed IPv6 literal would be passed as one destination string and the configured port would be lost.
   - Resolution: added a red/green test for `user@[2001:db8::1]:2200` and updated host parsing to split an optional user prefix before bracketed IPv6 parsing.

2. [high] `internal/workspace/manager.go` remote workspace creation script did not fail fast.
   - Impact: a failing intermediate command could be masked by a later successful `printf`, making remote workspace prepare look successful when it was not.
   - Resolution: added a red/green test requiring `set -eu` and prefixed the remote lifecycle script with fail-fast shell settings.

3. [medium] Runner-backed host selection was only used when `serviceDeps.hostSelection` was injected explicitly.
   - Impact: the selector could remain a test helper instead of the default service admission path.
   - Resolution: added a red/green service-constructor test and wired `newService` to build a default `runner.HostSelection` from `settings.Worker` when no test override is supplied.

## Verdict

All review findings were fixed with targeted tests before rerunning verification.
