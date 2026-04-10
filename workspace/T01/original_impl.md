# T01 Original Implementation Research

## Scope

Task `T01 Compatibility Contract` asks what the current Symphony implementation already treats as the compatibility contract that a Go port must preserve.

## Primary Source Artifacts

### `/Users/lihui/Documents/GitHub/symphony/SPEC.md`

- Defines Symphony as a language-agnostic service specification rather than an Elixir-only architecture.
- Establishes the key service boundary:
  - Symphony is a scheduler/runner and tracker reader.
  - Ticket writes, comments, PR links, and workflow-side tracker mutations are usually performed by the coding agent through tools available at runtime.
- Freezes top-level goals:
  - poll tracker work on a cadence
  - bounded concurrency
  - deterministic per-issue workspaces
  - single authoritative orchestrator state
  - retries and reconciliation
  - repo-owned `WORKFLOW.md`
  - operator-visible observability
  - restart recovery without a database
- Freezes top-level non-goals:
  - rich web UI or multi-tenant control plane
  - general-purpose workflow engine
  - built-in ticket/PR/comment business logic
  - mandated sandbox/approval posture
- Defines the strongest language-agnostic source for the boundary that the Go port should preserve.

### `/Users/lihui/Documents/GitHub/symphony/elixir/README.md`

- Documents the concrete user-facing behavior of the current reference implementation:
  - poll Linear for work
  - create a workspace per issue
  - launch Codex in app-server mode
  - send the workflow prompt
  - keep Codex working until the issue is done
- Makes explicit that app-server sessions inject the `linear_graphql` tool for Linear-specific operations.
- Documents the current compatibility surfaces:
  - `WORKFLOW.md` loading contract
  - optional observability web dashboard at `/`
  - JSON API routes `/api/v1/state`, `/api/v1/<issue_identifier>`, `/api/v1/refresh`
  - CLI flags such as `--logs-root` and `--port`
  - local and SSH e2e coverage expectations
- Documents important runtime semantics:
  - invalid startup `WORKFLOW.md` prevents boot
  - invalid later reload keeps last known good workflow
  - terminal issue states trigger agent stop and workspace cleanup

### `/Users/lihui/Documents/GitHub/symphony/elixir/WORKFLOW.md`

- Freezes the in-repo workflow contract shape:
  - YAML front matter for config
  - Markdown body for prompt template
- Shows the current prompt vocabulary is issue-centric:
  - `issue.identifier`
  - `issue.title`
  - `issue.state`
  - `issue.labels`
  - `issue.url`
- Makes Linear-specific tool availability explicit:
  - Linear MCP or injected `linear_graphql` must be available
- Encodes workflow and handoff expectations that later compatibility work must preserve even if internal architecture changes.
- Encodes the default unattended Linear workflow semantics around `Backlog`, `Todo`, `In Progress`, `Human Review`, `Rework`, `Merging`, and `Done`.

### `/Users/lihui/Documents/GitHub/symphony/elixir/AGENTS.md`

- Reinforces that the Elixir implementation should stay aligned with the language-agnostic `SPEC.md`.
- Calls out critical invariants:
  - `WORKFLOW.md` front matter drives runtime config
  - workspace safety is non-negotiable
  - orchestrator behavior is stateful and concurrency-sensitive
  - behavior/config changes must update docs alongside code
- Makes explicit that the Elixir implementation may be a superset of the root spec, but must not conflict with it.

### `/Users/lihui/Documents/GitHub/symphony/elixir/lib/symphony_elixir/orchestrator.ex`

- Confirms the orchestrator owns runtime truth for polling, claimed/running/retry bookkeeping, continuation retry, failure retry, reconciliation, and operator-visible updates.

### `/Users/lihui/Documents/GitHub/symphony/elixir/lib/symphony_elixir/tracker.ex`

- Shows the current Elixir implementation exposes tracker writes at the adapter boundary.
- This is precisely the behavior the Go design intentionally narrows: tracker reads belong to the core contract, but generic tracker writes should not expand the Go core interface.

### `/Users/lihui/Documents/GitHub/symphony/elixir/lib/symphony_elixir/prompt_builder.ex`

- Confirms prompt rendering is built from issue data plus retry attempt context.
- Reinforces that prompt rendering is part of the compatibility surface, even if internal implementation changes.

### `/Users/lihui/Documents/GitHub/symphony/elixir/lib/symphony_elixir_web/router.ex`

- Confirms the concrete web/API compatibility surface:
  - `/`
  - `/api/v1/state`
  - `/api/v1/refresh`
  - `/api/v1/:issue_identifier`
- Reinforces that observability routes are real user-facing product behavior, not optional internal details.

## What The Old System Treats As Normative

The old system does not centralize the full contract in one file. The contract is spread across:

1. `SPEC.md` for system boundaries and non-goals.
2. `elixir/README.md` for concrete runtime surfaces and operational semantics.
3. `elixir/WORKFLOW.md` for workflow/config and prompt vocabulary.
4. `elixir/AGENTS.md` for implementation invariants that cannot drift quietly.
5. `orchestrator.ex`, `tracker.ex`, `prompt_builder.ex`, and `router.ex` for concrete proof of runtime truth ownership, tracker boundary, prompt inputs, and API/dashboard surfaces.

## Terminology That Matters For The Go Port

- The external/user-facing vocabulary is still issue-centric:
  - issue
  - issue state
  - issue identifier
  - tracker kind
- The root spec exposes these source-language concepts that the Go contract should map intentionally rather than accidentally:
  - `Issue`
  - `Workflow Definition`
  - `Service Config`
  - `Workspace`
  - `Run Attempt`
  - `Live Session`
  - `Retry Entry`
  - `Orchestrator Runtime State`
- The implementation boundary is explicit:
  - tracker reading is part of Symphony core behavior
  - tracker writes belong in workflow/runtime tooling, not in a universal core interface
- `linear_graphql` is not a generic tracker feature; it is a Linear-specific compatibility behavior.

## Implications For T01

- T01 should not invent new behavior.
- T01 should extract and freeze the compatibility contract that is already implicit in the old system.
- The Go repo needs one durable place where later tasks can look up:
  - what user-facing parity must be preserved
  - what terminology mapping is intentional
  - what boundaries and non-goals later tasks must not violate
- T01 should preserve external behavior while avoiding a blind port of Elixir-internal abstractions that exist only because the current implementation is a superset of the language-agnostic spec.

## Recommended Capture In Go

- Promote the approved Go design's parity checklist and terminology mapping into a stable OpenSpec-backed contract.
- Preserve the old system's boundary that Symphony is a runner/orchestrator plus tracker reader, while provider-specific write behavior lives outside the core.
- Preserve the fact that observability surfaces are compatibility targets, but do not turn them into a second runtime state owner.
