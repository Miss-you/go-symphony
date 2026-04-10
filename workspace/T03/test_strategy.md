# T03 Test Strategy

## Goal

`T03 WORKFLOW Loader` must prove that `internal/config` can load the repository-owned `WORKFLOW.md` contract, preserve the raw workflow payload needed by later tasks, and survive invalid reloads without discarding the last known good workflow.

## What Must Be Proven

1. Workflow path resolution matches the current Symphony precedence rules.
2. The loader parses prompt-only files, YAML front matter, unterminated front matter, and typed error cases correctly.
3. The reload store replaces cached state on valid changes and preserves cached state on invalid changes.
4. The blank-prompt compatibility helper preserves Symphony's fallback behavior without requiring prompt rendering in `T03`.
5. `T03` stops at the raw loader/store boundary and does not require full runtime wiring to be verified.

## Verification Matrix

### 1. Package-scoped unit tests

Command:

```bash
go test ./internal/config/...
```

Why this matters:

- This is the task-board gate for `T03`.
- It proves the loader/store contract in isolation, which is the correct boundary for this task.
- It can directly exercise missing-file, invalid-YAML, and reload-fallback behavior without waiting for orchestrator or CLI integration.
- It should use injected tick/filesystem seams so reload behavior is proven deterministically without sleeping on real 1-second polls.

### 2. Repository-wide compile safety

Command:

```bash
go test ./...
```

Why this matters:

- `T03` will add new code and likely at least one dependency.
- A package-local green result is not enough if the new loader API breaks the rest of the skeleton or introduces module issues.

### 3. Build and lint confirmation

Commands:

```bash
make build
make lint
```

Why this matters:

- `make build` proves the new package compiles under the canonical repository entrypoint.
- `make lint` catches structural mistakes that unit tests may miss.
- These checks matter even before the full runtime is wired because the loader becomes a foundational core package.

### 4. E2E applicability check

Command:

```bash
make test-e2e
```

Why this matters:

- `T03` itself does not introduce a runnable end-to-end surface yet.
- Running the canonical command still proves whether the repository contract remains intact.
- If the command is not meaningful for this task beyond compile coverage, that limitation must be recorded explicitly in `workspace/T03/todo.md` and the task board instead of being silently skipped.

## Acceptance Threshold

`T03` is ready to close only if:

- `go test ./internal/config/...` passes
- the broader compile/build/lint checks requested above pass, or any non-applicable verification is explicitly recorded
- the observed behavior matches the OpenSpec scenarios for path resolution, parsing, blank-prompt fallback, and last-known-good reload semantics
