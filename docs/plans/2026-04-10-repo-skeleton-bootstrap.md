# Repo Skeleton Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Initialize the Go module and lay down the V1 repo skeleton for `T02` so the repository can build, lint, and test through the canonical `make` targets.

**Architecture:** Keep the scaffold flat and provider-neutral. Add a minimal `cmd/symphony` entrypoint plus placeholder `doc.go` packages for the approved `internal/...` layout so `go test ./...` passes without inventing runtime abstractions early.

**Tech Stack:** Go module, standard library, `golangci-lint`, `make`

---

### Task 1: Claim T02 And Record Implementation Intent

**Files:**
- Modify: `docs/plans/2026-04-10-go-symphony-design-task.md`
- Create: `docs/plans/2026-04-10-repo-skeleton-bootstrap.md`
- Create: `workspace/T02/final_impl.md`
- Create: `workspace/T02/test_strategy.md`
- Create: `workspace/T02/todo.md`

**Step 1: Claim T02 in the task board**

Update the task board row so `T02` moves from `todo` to `research`, assign `Owner=Codex`, set `Workspace=workspace/T02`, and record the claim in `Change Log`.

**Step 2: Capture the implementation boundary**

Write `workspace/T02/final_impl.md` describing the minimal bootstrap approach:

- `go.mod` at repo root
- `cmd/symphony/main.go` as the only executable entrypoint
- `internal/...` placeholder packages only
- no provider-specific behavior beyond the approved package layout

**Step 3: Record verification gates**

Write `workspace/T02/test_strategy.md` and `workspace/T02/todo.md` with the intended gates:

- `go test ./cmd/symphony`
- `go test ./...`
- `make build`
- `make lint`
- `make test`
- `make test-e2e`

### Task 2: Create The Minimal Executable Skeleton With TDD

**Files:**
- Create: `go.mod`
- Create: `cmd/symphony/main_test.go`
- Create: `cmd/symphony/main.go`
- Create: `internal/cli/doc.go`
- Create: `internal/codex/doc.go`
- Create: `internal/config/doc.go`
- Create: `internal/dashboard/doc.go`
- Create: `internal/domain/doc.go`
- Create: `internal/httpapi/doc.go`
- Create: `internal/observability/doc.go`
- Create: `internal/orchestrator/doc.go`
- Create: `internal/runner/doc.go`
- Create: `internal/tracker/doc.go`
- Create: `internal/trackers/linear/doc.go`
- Create: `internal/trackers/memory/doc.go`
- Create: `internal/web/doc.go`
- Create: `internal/workflow/doc.go`
- Create: `internal/workspace/doc.go`

**Step 1: Initialize the Go module**

Set the module path to `github.com/Miss-you/go-symphony`.

**Step 2: Write the failing entrypoint test**

Create `cmd/symphony/main_test.go` with a minimal test that calls `main()`.

Run:

```bash
go test ./cmd/symphony
```

Expected: compile failure because `main` is not defined yet.

**Step 3: Add the minimal executable**

Create `cmd/symphony/main.go` with an empty `main()` function.

**Step 4: Add approved placeholder packages**

Create `doc.go` files only for the approved `internal/...` packages from the design doc. Keep package comments brief and descriptive.

**Step 5: Format the scaffold**

Run:

```bash
gofmt -w cmd/symphony/main.go cmd/symphony/main_test.go internal
```

### Task 3: Prove The Skeleton Works Through Canonical Entrypoints

**Files:**
- Modify: `docs/plans/2026-04-10-go-symphony-design-task.md`
- Modify: `workspace/T02/test_strategy.md`
- Modify: `workspace/T02/todo.md`

**Step 1: Verify the red-green cycle**

Run:

```bash
go test ./cmd/symphony
```

Expected: PASS

**Step 2: Verify repository-wide gates**

Run:

```bash
go test ./...
make build
make lint
make test
make test-e2e
make verify
```

Expected: all commands exit `0`.

**Step 3: Update task board state**

Move `T02` through `implementing` and `verifying`, then mark it `done` only if the gates above pass and the repo layout matches the approved design.
