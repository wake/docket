---
title: Verify Docket on Codex and ship its plugin
---

The plugin and the docket skill have been checked on Claude Code only. On 2026-11-04 the maintainer chose to leave Codex untested and not to write a Codex plugin yet. What is open, all of it needs a real install:

- How a plugin skill is invoked in Codex (the name and form that load `docket-init` and `docket-doctor`).
- Whether the Codex CLI reads `.claude-plugin/marketplace.json`, or needs its own marketplace or plugin files.
- Whether superpowers, installed by `docket-init`, can be used in the same Codex session or needs a restart.
- Whether Codex reads the whole docket skill. On 2026-11-15, in four `codex exec` runs in a scratch repository (codex-cli 0.153.4), Codex read `SKILL.md` 240 lines at a time, and in two of the four it read no further. The skill is 254 lines; "Commit trailers" starts at line 233 and "Parallel work" at line 249, so both can be missed. A missed trailer leaves `dkt sync` unable to tick the plan.

`dkt init` already writes the docket skill to `.agents/skills/docket/` for Codex.
