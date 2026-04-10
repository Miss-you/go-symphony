## Overview

`T02` uses the smallest possible bootstrap that satisfies the approved design and task gate:

- one executable entrypoint at `cmd/symphony`
- flat placeholder packages under `internal/...`
- no early runtime abstractions or provider-specific behavior in core packages

## Key Decisions

1. Keep the package tree limited to the layout named in the approved design.
2. Use `doc.go` placeholder packages so later tasks can land code in stable package homes without inventing behavior now.
3. Use a minimal `main()` for build/test closure.
4. Treat this as an internal repo-bootstrap change, not a durable capability-spec change.

## Risks

- Placeholder packages can invite premature implementation if later tasks treat them as design freedom.
- Guard against that by keeping comments narrow and by relying on the task board and approved design as the source of truth.
