# T09 Test Strategy

## Goal

`T09 Codex App-Server Protocol` must prove that `internal/codex` can drive the current Symphony app-server protocol shape without becoming a runner, scheduler, or provider-specific tool layer. The tests should show more than compilation. They must prove that session bootstrap, turn execution, approval handling, dynamic tool dispatch, malformed traffic handling, timeout enforcement, and deterministic shutdown all work through a protocol boundary that later packages can trust.

## What Must Be Proven

1. Workspace validation blocks invalid launch contexts before the protocol session starts, including the workspace root, out-of-root paths, and symlink escapes.
2. Session bootstrap sends the expected protocol sequence, stores thread identity, and advertises dynamic tool specs at `thread/start`.
3. Turn execution keeps one session reusable across multiple turns, normalizes protocol facts, and ends cleanly on terminal results.
4. Approval handling and user-input prompts behave deterministically under the current Symphony policies, especially non-interactive `approval_policy == "never"`.
5. Dynamic tool calls are dispatched through an injected handler, while unsupported tools return structured failures instead of hanging the session.
6. Malformed and unknown protocol input are surfaced as protocol events rather than crashing the loop.
7. Read timeout and whole-turn timeout are distinguishable so callers do not need string matching to classify failure mode.
8. The package boundary stays narrow: `internal/codex` owns protocol behavior only, while orchestration, runner, workflow, and Linear write behavior remain outside the package.

## Verification Matrix

### 1. Package-scoped protocol contract tests

Command:

```bash
go test ./internal/codex/...
```

Why this matters:

- This is the task-board gate for `T09`.
- The package is the protocol engine, so the tests need to prove transcript behavior directly instead of only proving that the package compiles.
- The right test shape is a scripted transport harness that records outgoing JSON, feeds inbound lines, simulates slow or missing responses, and closes cleanly.

What this proves:

- `initialize` is sent before `thread/start`
- `thread/start` includes dynamic tool specifications
- the session stores thread identity after bootstrap
- `turn/start` carries approval and sandbox settings parsed from `internal/config`
- terminal turn events close the turn without extra state transitions
- repeated turns reuse the same session until the caller stops it
- malformed lines do not crash the receive loop
- unknown messages are emitted as protocol facts
- approval requests are answered non-interactively when policy is `never`
- user-input prompts return deterministic answers through the approval boundary
- tool calls reach the injected handler
- unsupported tools return structured failures
- read timeout and turn timeout remain distinguishable failures
- shutdown is deterministic and does not leak transport ownership

### 2. Repository-wide regression sweep

Command:

```bash
go test ./...
```

Why this matters:

- `internal/codex` sits in the core path and must compile cleanly with the rest of the repository.
- This check catches interface drift between protocol types, orchestrator expectations, runner boundaries, and workspace validation helpers.
- It also proves the new package does not accidentally import provider-specific behavior that should stay in compatibility layers.

What this proves:

- package boundaries stay compilable after the Codex protocol engine lands
- the core runtime still links with the new protocol surface
- no hidden dependency on Linear tool execution or workflow selection appears in `internal/codex`

### 3. Canonical build and lint gates

Commands:

```bash
make build
make lint
```

Why this matters:

- `make build` proves the canonical CLI build path still succeeds after the app-server protocol package lands.
- `make lint` catches formatting, vet, and static issues that unit tests will not expose, especially around protocol models, cancellation paths, and error handling.

What this proves:

- the repository still produces a working binary through the normal entrypoint
- the code style and static checks stay healthy after protocol plumbing is added
- the Codex package did not introduce compile-only fixes that fail under build or vet

### 4. E2E applicability check

Command:

```bash
make test-e2e
```

Why this matters:

- T09 lands before T14 wires the full runtime path, so `make test-e2e` is not the primary proof of Codex protocol correctness yet.
- Running it still validates the repository command contract and surfaces accidental coupling to later integration surfaces.
- If the command is still low-signal for T09, that limitation must be recorded in `workspace/T09/todo.md` and the task-board notes before closure.

What this proves:

- the repo’s e2e entrypoint still runs cleanly
- the protocol work did not break the broader command contract
- the current limitation is explicit instead of hidden behind a green checkmark

## Acceptance Threshold

`T09` is ready to leave verification only if:

- `go test ./internal/codex/...` proves the bootstrap, turn, approval, tool dispatch, timeout, malformed-input, and shutdown behaviors in `workspace/T09/final_impl.md`
- `go test ./...`, `make build`, and `make lint` pass
- `make test-e2e` either passes or its current low-signal applicability is recorded explicitly
- the tests show `internal/codex` owns protocol mechanics only, while orchestration, runner behavior, workflow selection, and Linear writes remain outside the package
