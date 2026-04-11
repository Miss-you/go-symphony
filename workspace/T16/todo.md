# T16 Residual Notes

## Deferred By Design

- CLI startup/shutdown wiring for the live terminal dashboard remains in T18.
- Web dashboard rendering and static assets remain in T17.
- The Go snapshot model does not expose the Elixir `codex_app_server_pid`; dashboard fixtures keep the `PID` header and use available runtime identifiers or `n/a` instead of expanding core state.

## Verification Notes

- `make verify` ran successfully and includes the repository e2e-tagged test gate (`go test -count=1 -tags=e2e ./...`).
- `TestFixtureProvenance` is the executable guard tying every Go dashboard fixture to copied Elixir source fixtures or explicit derived reasons.
- Code review follow-up is recorded in `workspace/T16/code_review_followup.md`.
- Final source/goal comparison is recorded in `workspace/T16/final_compare.md`.
