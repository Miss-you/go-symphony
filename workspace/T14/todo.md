# T14 Residual Notes

## Accepted Limits

- Prompt rendering is limited to V1 issue placeholders plus the default description conditional. Broader template-engine parity is deferred outside T14.
- Memory runtime coverage proves no Linear tool advertisement or Linear workflow bundle usage. It does not attempt outbound network interception because the memory path is constructed without a provider client.
- Linear runtime coverage verifies workflow-selected `linear_graphql` advertisement through `thread/start`. The Linear bridge behavior itself remains covered by `internal/toolbridge/linear`.
- Full process UX, API, dashboard, and web surfaces remain deferred to T15-T18.

## E2E Note

`make test-e2e` is meaningful for the current repository shape and passed after T14. It currently runs the Go test suite with the `e2e` tag; broader product-level end-to-end scenarios can be added once the API/dashboard surfaces land.

## Follow-Up Tasks

No T14 follow-up is required before T15.
