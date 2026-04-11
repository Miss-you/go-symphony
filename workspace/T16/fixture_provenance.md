# T16 Fixture Provenance

## Source Copies

Unmodified Elixir terminal dashboard fixture copies are stored under:

`internal/dashboard/testdata/status_dashboard_snapshots/source/`

| Go fixture | Source fixture | Notes |
| --- | --- | --- |
| `idle.snapshot.txt` | `idle.snapshot.txt` | Same frame skeleton and empty-state labels. |
| `idle_with_dashboard_url.snapshot.txt` | `idle_with_dashboard_url.snapshot.txt` | Same frame skeleton plus dashboard URL line. |
| `super_busy.snapshot.txt` | `super_busy.snapshot.txt` | Same header/rate-limit skeleton; Go fixture uses Go event text and row widths from the Go renderer. |
| `backoff_queue.snapshot.txt` | `backoff_queue.snapshot.txt` | Same retry queue coverage with four rows and matching reset values. |
| `credits_unlimited.snapshot.txt` | `credits_unlimited.snapshot.txt` | Same credits-unlimited rate-limit variant. |

## Derived Fixtures

| Go fixture | Reason |
| --- | --- |
| `snapshot_unavailable.snapshot.txt` | Go-specific unavailable view for snapshot failure state. |
| `orchestrator_snapshot_unavailable.snapshot.txt` | Go-specific unavailable view preserving source label. |
| `offline.snapshot.txt` | Minimal offline frame equivalent to Elixir `render_offline_status`. |

## Executable Gate

`TestFixtureProvenance` reads `internal/dashboard/testdata/status_dashboard_snapshots/provenance.json`, verifies every Go fixture is covered, rejects provenance entries for missing Go fixtures, verifies mapped source fixtures exist, requires derived fixtures to record a reason, and compares normalized source/Go frame skeletons plus running and retry rows for mapped fixtures. The test prevents accidental fixture transcription drift from becoming local truth.
