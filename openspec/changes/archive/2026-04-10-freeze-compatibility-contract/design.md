## Context

`go-symphony` currently has an approved design and an execution task board, but it does not yet have a durable spec-level contract for compatibility work. The approved design already freezes the intended parity scope, terminology mapping, and core-versus-compatibility-shell split; however, downstream implementation tasks would still need to reopen the design document unless those rules are promoted into OpenSpec artifacts.

The root Symphony `SPEC.md` and the current Elixir implementation also establish important boundaries that the Go port should preserve: `WORKFLOW.md` is a real compatibility surface, observability surfaces are user-facing product behavior, the orchestrator owns runtime truth, and ticket writes are workflow/tooling behavior rather than a reason to widen the Go core tracker interface.

## Goals / Non-Goals

**Goals:**

- Freeze the approved parity surfaces as normative OpenSpec requirements.
- Freeze the approved Symphony/Elixir-to-Go terminology mapping as normative reference.
- Freeze the intended provider-neutral core and compatibility-shell boundary rules.
- Make later changes update the contract in-repo when they alter parity scope or boundaries.

**Non-Goals:**

- Creating repo skeleton or Go package layout.
- Adding runtime code, API handlers, workflow loaders, or tracker adapters.
- Replacing the approved design document as the narrative architecture reference.
- Expanding `README.md` or setup docs beyond a tiny pointer if one is ever needed.

## Decisions

### Use one new capability: `compatibility-contract`

This change introduces one new capability rather than scattering contract rules across multiple specs. The contract is narrow and foundational: parity scope, terminology mapping, boundary rules, and explicit V1 non-goals all belong together because later implementation tasks need to consult them as one decision surface.

Alternative considered:

- Leave the approved design as the only source of truth.
  - Rejected because downstream tasks would keep reinterpreting design prose and task-local scope could drift.

### Keep T01 documentation-only

`T01` stops before `T02 Repo Skeleton`. It deliberately does not create `cmd/`, `internal/`, `go.mod`, or runtime code. That keeps the task aligned with its acceptance bar and avoids turning the contract-freeze step into premature implementation.

Alternative considered:

- Start the repo skeleton while the contract is being written.
  - Rejected because it mixes contract definition with implementation and makes it harder to prove whether later work stayed inside the approved boundaries.

### Promote only normative content into the spec

The synced main spec should hold the rules later tasks must obey:

- required parity surfaces
- terminology mapping
- boundary rules
- explicit non-goals
- the requirement to update this contract when those rules change

The design document remains the place for broader rationale and architecture narrative.

Alternative considered:

- Duplicate large sections of the approved design into the spec.
  - Rejected because it creates two competing design documents instead of one design plus one normative contract.

### Preserve the root Symphony boundary for tracker writes

The Go core should remain read-focused even though the current Elixir implementation exposes some tracker writes at the adapter boundary. The contract will therefore freeze the narrower Go boundary: provider-neutral core depends on tracker reads, while ticket writes remain workflow/runtime or compatibility-shell behavior.

Alternative considered:

- Mirror the current Elixir tracker boundary in the Go core.
  - Rejected because it conflicts with the approved Go design and over-generalizes provider-specific behavior.

## Risks / Trade-offs

- Design/spec duplication risk -> Keep the spec strictly normative and concise; keep narrative reasoning in the approved design and this change design.
- Contract too abstract to guide later tasks -> Write scenarios that name concrete parity surfaces and the required update discipline.
- Scope creep into `T02` -> Verification must fail if `cmd/`, `internal/`, `go.mod`, or other code-bearing skeleton files appear in this task.
