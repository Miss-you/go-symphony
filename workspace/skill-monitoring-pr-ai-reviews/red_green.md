# RED/GREEN

## RED

Before this change:

- `find .codex/skills -maxdepth 2 -type d | sort | rg -n "pr|review|monitor|follow|post" -n -S || true` returned no project-level skill for post-task PR AI review follow-up.
- `find .github/workflows -maxdepth 1 -type f | sort | rg -n "pr-ai|review|copilot|monitor" -n -S || true` returned no scheduled workflow for PR AI review scanning.

Conclusion: the repo had no local skill and no scheduled monitor for the "task done -> create PR -> keep watching AI review -> triage/fix/resolve" workflow.

## GREEN

After this change, these checks passed on 2026-04-10:

- `python3 scripts/pr_ai_review_monitor.py --repo Miss-you/go-symphony --pr 3`
- `python3 scripts/pr_ai_review_monitor.py --repo Miss-you/go-symphony --output /tmp/pr-ai-review-report.md`
- `ruby -e 'require "yaml"; YAML.load_file(".github/workflows/pr-ai-review-monitor.yml"); puts "ok"'`
- `rg -n "Copilot|AI review|resolveReviewThread|non-draft PR|public API" .codex/skills/monitoring-pr-ai-reviews/SKILL.md`

Observed result:

- the script reports zero unresolved AI review threads for PR `#3` after thread resolution
- the all-open-PR scan currently reports `PRs scanned: 0`, which matches the current repo state because PR `#3` is already `MERGED`
- the workflow YAML parses successfully
- the skill is discoverable by search and contains the evaluation rules needed for future use
