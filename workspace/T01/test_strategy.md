# T01 Test Strategy

## Goal

`T01 Compatibility Contract` is a documentation-and-spec task. The purpose of verification is not to prove runtime behavior; it is to prove that the repository now contains a durable, validated compatibility contract that downstream tasks can implement against without reopening the design doc.

## What Must Be Proven

1. The OpenSpec change is structurally valid and archiveable.
2. The contract captures the required parity surfaces, terminology mapping, boundary rules, and explicit V1 non-goals.
3. The task stayed inside `T01` scope and did not absorb `T02` code or repo-skeleton work.
4. After sync/archive, the durable contract exists in main specs.

## Verification Matrix

### 1. OpenSpec validation

Command in landed repo state:

```bash
openspec validate compatibility-contract
```

Historical pre-archive gate:

```bash
openspec validate freeze-compatibility-contract
```

Why this matters:

- Proves the landed main spec is internally consistent and still valid in the merged repo state.
- Preserves the exact change-level validation that was run before archive during task execution.

### 2. Contract-content consistency review

Command:

```bash
rg -n "WORKFLOW.md|default unattended Linear workflow|WorkItem|provider|linear_graphql|no universal tracker write interface|no universal workpad abstraction|observability" \
  docs/plans/2026-04-10-go-symphony-design.md \
  workspace/T01/final_impl.md \
  openspec/specs/compatibility-contract/spec.md \
  openspec/changes/archive/2026-04-10-freeze-compatibility-contract/specs/compatibility-contract/spec.md
```

Why this matters:

- Proves the change spec actually contains the contract points the design and accepted implementation plan require.
- Prevents a formally valid OpenSpec change that still misses the real parity or boundary obligations.

### 3. Scope guard against `T02` leakage

Command:

```bash
test -z "$(git diff --cached --name-only -- cmd internal go.mod)"
```

Why this matters:

- Proves `T01` remained documentation-only.
- Prevents accidental repo-skeleton or code bootstrap work from being mixed into the contract-freeze task, even when the repository already contains `T02` baseline files.

### 4. Sync/archive landing check

Commands:

```bash
test -f openspec/specs/compatibility-contract/spec.md
openspec list --specs
```

Why this matters:

- Proves the contract is no longer trapped inside a transient change directory.
- Confirms the durable spec landing point exists for downstream tasks.

## Intentionally Not Run In T01

### Build / compile

Not applicable because `T01` introduces no Go code and intentionally must not add or modify repo-skeleton/code paths such as `go.mod`, `cmd/`, or `internal/`.

### Lint

Not applicable as a primary gate because this task is about OpenSpec/document consistency, not code style or static analysis of runtime sources.

### Unit tests

Not applicable because there is no runtime behavior implemented in this task.

### E2E tests

Not applicable because no runnable service or compatibility surface is implemented yet.

## Acceptance Threshold

`T01` passes verification only if:

- `openspec validate compatibility-contract` passes
- the content check shows the required contract points are present
- the diff-based scope guard confirms no `T02` skeleton/code paths are part of this task's change set
- the synced main spec exists after sync/archive

If any of those fail, `T01` is not done even if the prose looks correct.
