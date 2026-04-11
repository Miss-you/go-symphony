## Context

The approved Symphony port design requires a web observability surface at `/` and treats observability as projection-only. The web layer must present the same runtime truth that the API and terminal dashboard consume, but it must not become a second state machine or own mutable run state.

T17 sits after the HTTP API work and before CLI parity. That means the web dashboard needs to be compatible with the shared snapshot model already established by the runtime, while also preserving the dashboard shell and static assets expected by the current product.

## Goals / Non-Goals

**Goals:**
- Serve the web dashboard at `/` as a compatibility surface.
- Serve the static assets required by the dashboard with stable lookup semantics.
- Render the dashboard from `observability.DashboardView`.
- Preserve the original dashboard's title, heading, status affordances, four metric cards, running/retry tables, and JSON detail links.
- Delegate `/api/v1/*` to `internal/httpapi` without duplicating DTO or error behavior.
- Start the web handler from runtime assembly when `server.port` is configured.
- Keep web presentation code projection-only and out of runtime state ownership.

**Non-Goals:**
- No new API endpoints.
- No changes to orchestrator scheduling, tracker behavior, or workflow selection.
- No terminal dashboard work.
- No CLI flag parsing, startup acknowledgement copy, or shutdown rendering. Those remain T18.
- No second observability state machine or browser-side truth source.

## Decisions

1. **Use an embedded asset bundle with stable URLs.**
   - Rationale: The dashboard assets need to be available in every execution environment and easy to verify in tests. Embedding the bundle keeps the compatibility surface reproducible and avoids depending on a mutable filesystem layout.
   - Alternatives considered:
     - Serve assets from disk at runtime. Rejected because it couples the compatibility surface to deployment layout.
     - Generate assets on demand. Rejected because it blurs the contract and makes asset availability harder to test.

2. **Render the dashboard from the shared observability presenter model.**
   - Rationale: The approved architecture already defines observability as projection-only. Using the shared presenter keeps web rendering aligned with API and terminal surfaces and avoids creating a second runtime state owner.
   - Alternatives considered:
     - Let the web layer query orchestrator state directly. Rejected because that would duplicate runtime ownership.
     - Introduce web-specific state to cache presentation facts. Rejected because it would recreate a second observability machine.

3. **Keep root rendering and asset serving inside `internal/web`.**
   - Rationale: The web package should own browser-facing routing and response assembly, while the core runtime remains provider-neutral. This keeps the compatibility boundary clear and limits the blast radius of web-specific changes.
   - Alternatives considered:
     - Move presentation responsibilities into observability. Rejected because observability should stay projection-only.
     - Push asset handling into a more general HTTP layer. Rejected because T17 is specifically about the web dashboard compatibility surface.

4. **Treat missing assets as explicit not-found responses.**
   - Rationale: Compatibility tests need to detect broken asset lookup rather than silently falling back to another file or inline content. A clear not-found path makes asset regressions visible.
   - Alternatives considered:
     - Fall back to the root dashboard shell for unknown asset paths. Rejected because it masks missing-asset bugs.
     - Inline every asset in the HTML response. Rejected because it obscures cacheable asset behavior.

5. **Delegate JSON API routes instead of reimplementing them.**
   - Rationale: T15 already froze `/api/v1/state`, `/api/v1/refresh`, and `/api/v1/:issue_identifier`. The web handler should compose the browser surface with that handler so route behavior, error envelopes, and DTOs stay single-owned.
   - Alternatives considered:
     - Recreate API responses in `internal/web`. Rejected because it duplicates compatibility behavior and increases drift risk.
     - Let unknown routes fall through to the root dashboard. Rejected because Symphony returns explicit JSON 404/405 responses for non-dashboard routes.

6. **Start the handler from configured runtime server settings.**
   - Rationale: A route contract is not user-visible unless the runtime mounts it when `server.port` is configured, matching the original optional endpoint behavior.
   - Alternatives considered:
     - Leave all server lifecycle wiring to T18. Rejected because T17's `/` acceptance would only exist as an unmounted package.
     - Add CLI `--port` parsing now. Rejected because T18 owns CLI compatibility, acknowledgement copy, and startup/shutdown behavior.

## Risks / Trade-offs

- [Risk] Asset names or paths may drift from the current product. → Mitigation: keep asset URLs stable, cover them with fixtures, and make lookup failures explicit.
- [Risk] Embedded assets may diverge from local development expectations. → Mitigation: keep a single bundled source of truth and verify the served bytes in package tests.
- [Risk] Presentation code may accidentally grow stateful behavior. → Mitigation: keep the web package on the projection path only and test that rendering is deterministic for a given snapshot.

## Migration Plan

1. Land the web dashboard spec and task list.
2. Implement the root dashboard route and static asset serving in `internal/web`.
3. Wire the route to the shared observability presenter model.
4. Add compatibility tests for empty-state rendering, asset lookup, and deterministic output.
5. Validate the change and then implement T17 against this contract.

Rollback is straightforward: remove the new web bundle and route wiring, then keep the shared runtime projection intact for later retry.

## Open Questions

- The exact asset list should be finalized against implementation fixtures, but the compatibility contract should stay stable once the assets are pinned.
- Whether the dashboard shell is server-rendered template HTML or a minimal bootstrap wrapper can be decided during implementation, as long as the root route and asset contract remain intact.
