# T03 Original Implementation Research

## Scope

This note covers how the current Symphony implementation in `/Users/lihui/Documents/GitHub/symphony` handles `WORKFLOW.md` discovery, parsing, prompt/template loading, hot reload, and last-known-good fallback behavior.

## Relevant Source Files

- `elixir/lib/symphony_elixir/workflow.ex`: workflow file path resolution, file loading, front-matter splitting, and the returned workflow shape.
- `elixir/lib/symphony_elixir/workflow_store.ex`: cache, polling, file-stamp detection, and last-known-good reload behavior.
- `elixir/lib/symphony_elixir/config.ex`: config accessors and the blank-prompt fallback template.
- `elixir/lib/symphony_elixir/prompt_builder.ex`: template parsing and strict rendering behavior.
- `elixir/lib/symphony_elixir/cli.ex`: explicit workflow-path startup behavior and default path selection.
- `elixir/WORKFLOW.md`: the real repository workflow contract example.
- `elixir/README.md` and `SPEC.md`: documented contract for path precedence, parsing, reload, and fallback rules.
- `elixir/test/symphony_elixir/core_test.exs` and `elixir/test/symphony_elixir/extensions_test.exs`: edge-case coverage for parsing, reload, and fallback behavior.

## Behavior Summary

- `Workflow.workflow_file_path/0` uses the application env override when present; otherwise it defaults to `File.cwd!()/WORKFLOW.md` (`elixir/lib/symphony_elixir/workflow.ex:10-14`).
- The CLI accepts an explicit `WORKFLOW.md` path or defaults to `./WORKFLOW.md`; the file must exist before startup continues (`elixir/lib/symphony_elixir/cli.ex:32-70`).
- `Workflow.load/1` reads the file, returning `{:error, {:missing_workflow_file, path, reason}}` on read failure (`elixir/lib/symphony_elixir/workflow.ex:52-60`).
- Parsing splits an initial `---` block into YAML front matter and prompt body. If no front matter exists, the whole file becomes the prompt body and config is `{}` (`elixir/lib/symphony_elixir/workflow.ex:63-114`).
- The prompt body is trimmed before use, and the loaded workflow returns both `prompt` and `prompt_template` with the trimmed body (`elixir/lib/symphony_elixir/workflow.ex:66-75`).
- YAML front matter must decode to a map. Non-map YAML returns `:workflow_front_matter_not_a_map`; YAML decode failures are wrapped as `{:workflow_parse_error, reason}` (`elixir/lib/symphony_elixir/workflow.ex:102-112`).
- `WorkflowStore` is the runtime cache for the loaded workflow. It loads on init, then polls every 1000 ms using a stamp derived from file mtime, size, and content hash (`elixir/lib/symphony_elixir/workflow_store.ex:11-17`, `:49-59`, `:82-94`, `:141-148`).
- When the workflow file changes, `WorkflowStore` reloads from the current path. If reload succeeds, the cache is replaced; if reload fails, the previous workflow stays active and the error is logged (`elixir/lib/symphony_elixir/workflow_store.ex:96-152`).
- `Workflow.current/0` and `WorkflowStore.current/0` fall back to direct file loading when the store is not running (`elixir/lib/symphony_elixir/workflow.ex:36-44`, `elixir/lib/symphony_elixir/workflow_store.ex:24-33`).
- `Workflow.set_workflow_file_path/1` and `clear_workflow_file_path/0` both trigger an immediate store reload when the store is running (`elixir/lib/symphony_elixir/workflow.ex:16-27`).
- `Config.workflow_prompt/0` returns the current prompt body when present; if the body is blank or workflow loading fails, it falls back to an internal default prompt template (`elixir/lib/symphony_elixir/config.ex:75-83`).
- The default prompt template in `Config` includes the issue identifier, title, and body, and the body section uses a conditional fallback when description is missing (`elixir/lib/symphony_elixir/config.ex:9-21`).
- `PromptBuilder` parses the workflow prompt with Solid, renders with strict variables and filters, and passes `issue` plus `attempt` into the template (`elixir/lib/symphony_elixir/prompt_builder.ex:8-25`).

## Edge Cases / Fallbacks

- Prompt-only files are accepted and become the prompt body with empty config (`elixir/test/symphony_elixir/core_test.exs:176-181`).
- Unterminated front matter is treated as front matter through EOF and yields an empty prompt body (`elixir/test/symphony_elixir/core_test.exs:184-189`).
- Non-map front matter is rejected (`elixir/test/symphony_elixir/core_test.exs:192-196`).
- Blank prompt bodies fall back to the built-in default prompt template rather than failing (`elixir/test/symphony_elixir/core_test.exs:872-913`).
- Template syntax errors are surfaced as `template_parse_error`, while missing variables fail strict rendering (`elixir/test/symphony_elixir/core_test.exs:836-870`).
- The workflow store keeps the last known good workflow when a reload breaks due to invalid content or a missing path (`elixir/test/symphony_elixir/extensions_test.exs:104-127`, `:137-170`).
- Startup stops if the workflow file is missing, rather than booting with an implicit fallback (`elixir/test/symphony_elixir/extensions_test.exs:130-135`).
- The documented contract also says reload failures must not crash the service and must keep the last known good workflow active (`elixir/README.md:148-150`, `SPEC.md:1948-1953`).

## Constraints for Go Port

- Preserve the same path precedence: explicit runtime path first, then cwd `WORKFLOW.md` (`SPEC.md:284-292`, `SPEC.md:1945-1948`).
- Preserve the same parse contract: optional YAML front matter, trimmed prompt body, map-only front matter, and distinct error classes for missing file, parse failure, and non-map front matter (`SPEC.md:296-316`, `SPEC.md:471-484`).
- Preserve the same hot-reload semantics: detect changes without restart, re-read and re-apply config, and keep last known good workflow on invalid reloads (`SPEC.md:507-523`).
- Preserve the same prompt split between loading and rendering. File/read failures must not silently fall back to a prompt, but a blank prompt body may use the built-in default prompt (`SPEC.md:446-469`, `elixir/lib/symphony_elixir/config.ex:75-83`).
- Preserve strict template rendering with `issue` and `attempt` variables, and surface template failures only on the affected run attempt (`SPEC.md:448-485`, `elixir/lib/symphony_elixir/prompt_builder.ex:8-25`).
- Preserve the runtime cache behavior: direct load when no store is running, immediate reload when the path changes, and poll-driven refresh using a stable file stamp (`elixir/lib/symphony_elixir/workflow_store.ex:24-47`, `:82-152`).
