# T01 Current Go State Research

## Scope

Task `T01 Compatibility Contract` asks what durable artifacts already exist in `go-symphony`, what is still missing, and what the approved design requires before downstream implementation starts.

## Existing Durable Artifacts

### `docs/plans/2026-04-10-go-symphony-design.md`

- This is the only real contract artifact in the repo today.
- It already contains the material T01 needs:
  - explicit parity scope
  - explicit V1 non-goals
  - terminology mapping from Symphony/Elixir to Go internals
  - core versus compatibility-shell boundary
  - anti-overdesign constraints
  - rough task breakdown and acceptance criteria for `T01`

### `docs/plans/2026-04-10-go-symphony-design-task.md`

- This task board is now the execution truth source.
- It is required for status tracking, but it is derivative of the design rather than the compatibility contract itself.

### `README.md`

- Establishes repository identity only.
- It does not define compatibility behavior, boundaries, or acceptance criteria.

### `openspec/config.yaml`

- Confirms OpenSpec is configured.
- The repo is structurally ready to express T01 as a change plus stable specs.

## Missing Artifacts

- No active OpenSpec change exists for `T01`.
- No stable spec exists under `openspec/specs/`.
- No implementation artifact currently turns the design into a normative contract for later tasks.
- No code exists yet, which means T01 should avoid leaking into `T02 Repo Skeleton`.

## Design Constraints That Must Govern T01

From `docs/plans/2026-04-10-go-symphony-design.md`:

- User-facing parity comes first; architecture may become more Go-native.
- The system is intentionally split into:
  - a provider-neutral core
  - a provider-specific compatibility shell
- The following constraints are mandatory and should be treated as normative:
  - no universal tracker write interface in core
  - no universal workpad abstraction in core
  - no fake provider-agnostic default workflow
  - no second runtime state machine for observability
  - no oversized `WorkItem`
- The terminology mapping is already stable enough to be promoted into spec-level contract language.
- T01 acceptance is intentionally narrow:
  - parity checklist exists
  - terminology mapping exists
  - in-scope and out-of-scope items are explicit

## Gap Analysis

The current repo has good design prose, but not yet an executable contract. Without a spec-backed compatibility contract:

- later tasks will keep reinterpreting the design
- scope boundaries can drift across task workspaces
- terminology can regress back to Elixir-specific names in core code
- parity obligations can be enforced socially instead of through repo artifacts

## Recommended Direction

- Keep T01 documentation-only and contract-focused.
- Create a single-task OpenSpec change for `T01`.
- Convert the approved design's normative parts into durable spec artifacts:
  - parity scope
  - terminology mapping
  - explicit non-goals
  - core versus compatibility-shell boundary rules
- Do not create repo skeleton or code in this task; that belongs to `T02`.

## Acceptance Proof For T01

T01 should be considered complete only when the repo contains durable artifacts that let a later task answer these questions without reopening the design document:

1. What user-facing parity is in scope?
2. Which Symphony/Elixir terms intentionally map to different Go internal names?
3. Which behaviors and abstractions are explicitly out of scope for V1?
4. Which boundary rules later tasks are not allowed to violate?
