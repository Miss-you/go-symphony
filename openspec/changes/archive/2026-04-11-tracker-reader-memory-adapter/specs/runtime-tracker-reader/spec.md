## ADDED Requirements

### Requirement: Core tracker boundary stays provider-neutral and read-only
The core runtime MUST define a provider-neutral `TrackerReader` contract that exposes only the tracker reads the Symphony runtime actually depends on. The contract MUST support candidate listing, state-based listing, and refresh-by-id reads, and it MUST not expose tracker write operations in the core.

#### Scenario: Core code depends on tracker reads only
- **WHEN** a later core package compiles against `internal/tracker`
- **THEN** it can request candidate items, items by normalized state, and refreshed items by ID through one `TrackerReader`
- **AND** it does not gain comment-creation, state-mutation, or provider-specific query APIs from the core tracker boundary

#### Scenario: Restart-cleanup read is part of the frozen contract
- **WHEN** later runtime work needs to list terminal-state items for startup cleanup behavior
- **THEN** the core tracker contract already exposes a state-based read
- **AND** later tasks do not need to widen `TrackerReader` to add that capability

### Requirement: Tracker reads preserve deterministic semantics for later runtime use
`TrackerReader` reads MUST preserve deterministic, runtime-safe semantics. State-based reads MUST normalize incoming state names by trimming whitespace and comparing case-insensitively. Refresh-by-id reads MUST accept tracker-internal item IDs, preserve request order for visible matches, omit missing IDs without error, and return empty results for empty input.

#### Scenario: State-based query normalizes requested state names
- **WHEN** a caller requests items by states such as `" In Progress "` and `"done"`
- **THEN** the tracker reader compares those state names after trimming whitespace and case-folding
- **AND** matching items are returned without requiring the caller to normalize state names first

#### Scenario: Refresh-by-id preserves request order and skips missing items
- **WHEN** a caller refreshes item IDs in the order `["item-2", "missing", "item-1"]`
- **THEN** the reader returns only the visible matches
- **AND** the returned slice preserves the requested visible order as `item-2`, then `item-1`

#### Scenario: Empty input is a no-op read
- **WHEN** a caller passes an empty state list or empty ID list
- **THEN** the reader returns an empty slice with no error

### Requirement: Memory tracker reader is deterministic and caller-isolated
The memory-backed tracker reader MUST implement `TrackerReader` for local and test use. It MUST keep a private copy of seeded `domain.WorkItem` data and return deep copies on every read so caller mutations cannot leak back into adapter-internal state.

#### Scenario: Candidate reads do not expose shared mutable data
- **WHEN** a caller reads seeded candidate items from the memory reader and mutates returned labels, blockers, or pointer-backed fields
- **THEN** those mutations affect only the caller-owned copies
- **AND** a subsequent memory-reader call returns the original seeded values

#### Scenario: Memory reader supports exact ID refresh
- **WHEN** a caller refreshes seeded item IDs through the memory reader
- **THEN** the reader matches items by exact item ID
- **AND** it returns normalized `domain.WorkItem` values that still preserve the runtime fields later scheduler logic depends on
