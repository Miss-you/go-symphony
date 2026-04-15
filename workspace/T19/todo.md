# T19 Residual Notes

- Live Linear/Codex smoke was not executed during automated verification. It requires `LINEAR_API_KEY`, a disposable Linear issue, and consent to launch real `codex app-server`.
- `symphony-verify run` scopes terminal cleanup through the selected issue filter. This is intentional for smoke safety and does not replace normal unfiltered `cmd/symphony` startup validation.
