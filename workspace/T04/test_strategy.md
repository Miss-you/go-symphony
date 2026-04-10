# T04 Test Strategy

## Goal

`T04 Internal Config Model` must prove that `internal/config` can turn the raw `WORKFLOW.md` payload from `T03` into a typed, provider-neutral `Settings` model without regressing Symphony compatibility. The important proof is not just "the parser still works"; it is that defaults, env/path resolution, semantic validation, and last-known-good reload behavior now apply to typed settings as one coherent contract.

## What Must Be Proven

1. Legacy external workflow input still works, but downstream code can consume typed `Settings` instead of reparsing `Workflow.Config`.
2. Typed defaults match the Symphony reference for the fields that later runtime packages will rely on.
3. Env/path normalization preserves observable behavior for `LINEAR_API_KEY`, `LINEAR_ASSIGNEE`, `$VAR`, `~`, and missing-vs-empty workspace-root resolution.
4. Semantic validation fails early for unsupported provider kinds, missing required Linear fields, and invalid numeric/state-limit values.
5. `LoadSettings` and `CurrentSettings` are the supported typed entry points and return typed config without callers reparsing `Workflow.Config`.
6. Startup and reload never expose mismatched raw and typed config state; a bad typed reload keeps the previous full snapshot active.

## Verification Matrix

### 1. Package-scoped unit tests

Command:

```bash
go test ./internal/config/...
```

Why this matters:

- This is the task-board gate for `T04`.
- It is the only boundary that can directly prove typed defaults, env/path normalization, startup fail-fast behavior, and atomic raw+typed snapshot replacement.
- It must exercise `ParseSettings`, `LoadSettings`, and `CurrentSettings` directly so the typed API surface is proven instead of only being covered indirectly.
- It should assert concrete defaults, not just that "some default exists", for representative fields such as `Provider.Endpoint`, `Polling.IntervalMS`, `Workspace.Root`, `Agent.MaxTurns`, `Codex.Command`, and `Server.Host`.
- It should explicitly cover the `linear`/`memory` provider matrix, unsupported provider rejection, and reload behavior when raw parsing succeeds but typed validation fails.

### 2. Repository-wide compile safety

Command:

```bash
go test ./...
```

Why this matters:

- `T04` changes the foundational config contract that later packages will consume.
- A green `internal/config` package is not enough if the new typed API breaks imports, causes name collisions, or leaves the broader module in a bad compile state.

### 3. Canonical build and lint checks

Commands:

```bash
make build
make lint
```

Why this matters:

- `make build` proves the repository can still build through the canonical entrypoint after the config surface grows.
- `make lint` catches structural issues that narrow unit tests may miss, especially around new types, exported APIs, and dead code in a core package.

### 4. E2E applicability check

Command:

```bash
make test-e2e
```

Why this matters:

- `T04` does not yet expose a full end-to-end runtime path, so the meaningful proof still lives in `internal/config`.
- Running the repository-standard e2e command still checks whether the command contract remains healthy.
- If it is only acting as a command-contract check rather than a behavior proof for `T04`, that limitation must be recorded explicitly in `workspace/T04/todo.md` and the task board instead of being silently skipped.

## Acceptance Threshold

`T04` is ready to leave verification only if:

- `go test ./internal/config/...` proves typed defaults, env/path handling, startup failure, and atomic reload fallback behavior
- `go test ./...`, `make build`, and `make lint` pass, or any non-applicable verification is recorded explicitly
- the observed behavior matches the OpenSpec requirements for typed settings, typed validation, and reload-safe raw+typed snapshots
