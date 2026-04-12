# T18 Verification Evidence

Fresh verification after implementation and code-review follow-up:

- `go test -count=1 ./internal/cli/... ./cmd/symphony/...` passed.
- `go test -count=1 -tags=e2e ./internal/cli/...` passed.
- `go test -count=1 ./internal/web/... ./internal/httpapi/... ./internal/dashboard/... ./internal/observability/...` passed.
- `go test -count=1 ./...` passed.
- `make build` passed.
- `make lint` initially found one unused test-only type; the type was removed, then `make lint` passed with `0 issues`.
- `make test` passed.
- `make test-e2e` passed and ran the e2e-tagged no-network runtime smoke test under `internal/cli`.
- `make verify` passed after the review fixes.
- `openspec validate --type change cli-full-parity-test-sweep` passed.
- `openspec validate --specs` passed for 16 specs before sync/archive.
- `git diff --check` passed.
- After syncing `cli-bootstrap-parity` into main specs and archiving the change, `openspec validate --specs` passed for 17 specs.
- After syncing and archiving, `make verify` passed.
- After syncing and archiving, `git diff --check` passed.

The gates prove the CLI/bootstrap change compiles, covers acknowledgement and argument behavior, preserves existing dashboard/API/observability contracts, starts and shuts down the memory runtime in an e2e-tagged no-network path, keeps the repo-wide command gates healthy, and validates the OpenSpec artifacts.
