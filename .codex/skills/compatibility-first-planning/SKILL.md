---
name: compatibility-first-planning
description: Use when go-symphony work starts from a broad goal, rewrite, migration, or multi-surface feature and the scope, parity requirements, or abstraction boundaries are still unclear.
---

# Compatibility-First Planning

## Overview

Turn a broad goal into an implementation-ready design without letting compatibility questions, naming drift, or future-provider speculation leak into the runtime core.

**Core principle:** Freeze external contracts and terminology before introducing abstractions.

This skill is for planning. It does not replace detailed execution planning. Once the design is approved, hand off to `writing-plans`.

## When to Use

Use this skill when:

- the request starts as "rewrite", "port", "support X", "re-architecture", or another broad goal
- user-facing parity matters for more than one surface
- multiple surfaces are involved, such as workflow, API, dashboard, CLI, or provider integration
- future extensibility matters, but overdesign is a real risk
- naming is likely to drift between the current system and the new implementation

Do not use this skill for:

- a small isolated bugfix
- a single-file implementation with already-clear acceptance criteria
- task-by-task execution once the design is already approved

## Required Outputs

Your planning output must contain all of these, either as separate docs or as sections in one design doc:

1. Goal and non-goals
2. Current-system inventory
3. Compatibility contract
4. Terminology mapping
5. Core vs compatibility-shell boundary
6. Phase plan
7. Rough task breakdown
8. Verification and parity plan
9. Current-system evidence references

If any of these are missing, the plan is not implementation-ready.

## The Flow

### 1. Clarify the target

Define what success means in user-facing terms.

Lock these early:

- what must be preserved
- what may change internally
- what is explicitly out of scope for v1

If "1:1" is requested, ask which surfaces count:

- workflow semantics
- API routes and payloads
- terminal dashboard
- web dashboard
- CLI behavior
- tracker/provider behavior

### 2. Inventory the current system

Before proposing architecture, identify:

- real runtime behavior
- user-facing surfaces
- provider-specific behavior
- implementation accidents that should not be preserved blindly

Back inventory claims with evidence when possible:

- point to concrete files, interfaces, routes, configs, or runtime behaviors
- prefer a small number of high-signal references over broad code dumps
- do not claim something is "core" or "compatibility-facing" without showing where that behavior exists today

Separate:

- true core behavior
- compatibility shell behavior
- future extension ideas

### 3. Freeze compatibility contracts

Write down the external contracts before abstraction work starts.

Examples:

- `WORKFLOW.md` parsing and reload behavior
- HTTP API routes, DTOs, and error semantics
- dashboard output expectations
- CLI flags and shutdown behavior
- provider-specific workflow behavior

Do not leave compatibility as "we'll compare later".

### 4. Write terminology mapping

Create an explicit old-term to new-term map.

Always do this when:

- the existing system uses provider-specific names
- the new system introduces a provider-neutral domain model
- internal and external language may intentionally differ

Example questions:

- What is the internal replacement for `issue`?
- Which external surfaces still say `issue` for compatibility?
- Which provider-specific names must stay out of the core?

### 5. Draw the architectural boundary

Define the smallest possible provider-neutral core.

For go-symphony, the default posture is:

- keep orchestration, runtime state, runner, workspace lifecycle, Codex protocol handling, and tracker reads in the core
- keep provider-specific workflow behavior, tracker writes, workpad semantics, and provider tools in the compatibility shell

Only abstract what the current system already proves is common.

### 6. Phase the work around closed loops

The earliest meaningful milestone must close a real loop.

Prefer this shape:

1. boot
2. load config
3. poll stub or memory source
4. run one worker
5. emit snapshot
6. stop cleanly

Do not make the first phase "all abstractions" or "all provider integrations".

### 7. Define checks before implementation

Every phase needs explicit verification.

At minimum, include:

- scope check
- boundary check
- compatibility check
- operational closed-loop check
- extension check
- maintenance check

If the plan says "we will test later", it is incomplete.

### 8. Hand off to execution planning

Once the design is approved:

- freeze the design doc
- identify the implementation order
- then use `writing-plans` for the task-by-task execution plan

## Quick Reference

| Planning artifact | Question it answers |
| --- | --- |
| Goal and non-goals | What are we actually building in v1? |
| Current-system inventory | What behavior exists today? |
| Compatibility contract | What must remain stable for users? |
| Terminology mapping | How do old names map to new names? |
| Architecture boundary | What belongs in core vs shell? |
| Phase plan | What order closes risk earliest? |
| Rough tasks | What big chunks of work exist? |
| Verification plan | How will we prove parity and correctness? |
| Evidence references | What concrete current-system facts justify the plan? |

## Default Checks

Run these checks against every major design:

- `Scope`: Is v1 solving one concrete end-to-end user path?
- `Boundary`: Is each abstraction required now, or only hypothetical?
- `Compatibility`: Is every user-facing surface explicitly covered?
- `Operational`: Can the system boot, run, observe, and stop cleanly?
- `Extension`: Would adding another provider later require rewriting the core?
- `Maintenance`: Is the design understandable to the people who will actually maintain it?

## Red Flags

Stop and fix the plan if you notice any of these:

- "We'll figure out compatibility as we implement."
- "Let's make tracker writes generic now for future providers."
- "The default workflow should be provider-agnostic from day one."
- "Observability can keep its own runtime truth."
- "CLI or shutdown behavior is just edge polish."
- "Terminology mapping is obvious; we can skip it."
- "We'll add verification after the code exists."

## Common Rationalizations

| Excuse | Reality |
| --- | --- |
| "We should generalize now so Lark is easy later." | Premature generalization makes the core unstable. Preserve only proven seams. |
| "Tracker reads and writes should live in one clean interface." | That pollutes the core with provider-specific workflow behavior. |
| "The dashboard and API can be checked after the runtime works." | Those are part of the user-facing contract and must be planned explicitly. |
| "Term mapping is clerical, not architecture." | Missing term mapping causes naming drift and wrong abstractions. |
| "Hot reload, CLI flags, and shutdown semantics are minor." | These are real compatibility surfaces when parity matters. |
| "A second observability store will make projections cleaner." | Duplicate truth creates drift. Keep the orchestrator as the single runtime owner. |

## Common Mistakes

- Jumping from vague goal to package layout without freezing compatibility
- Mixing provider-specific writes and workpad semantics into the core
- Letting future provider support dictate v1 abstractions
- Treating Codex app-server as a thin transport instead of a protocol contract
- Creating phases that do not close a real operational loop
- Producing architecture notes without a verification plan

## Output Standard

A good planning result for go-symphony should leave the reader knowing:

- what the v1 target is
- what is intentionally deferred
- how Symphony terms map into Go terms
- what the provider-neutral core is
- what remains compatibility-facing
- what order the work should happen in
- how parity will be proven

If any of those are unclear, the plan is not ready.

## Next Step

After approval, use `writing-plans` to turn the approved design into an execution-ready implementation plan.
