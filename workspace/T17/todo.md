# T17 Residual Notes

## Accepted Residuals

- Phoenix LiveView transport is not recreated. The Go implementation keeps the visible dashboard, route behavior, static asset paths, and later-request snapshot freshness without adding a browser socket layer.
- Pixel-perfect browser parity is not asserted because the source research found route/text coverage rather than a browser snapshot contract.
- CLI `--port`, startup acknowledgement copy, and shutdown rendering remain T18 scope.

## Follow-up Candidates

- Add browser polling or server-sent updates if parity hardening later requires automatic refresh without a full page reload.
- Add browser screenshot fixtures if a future design freezes pixel-level dashboard appearance.
