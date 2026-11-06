---
title: Verify Docket on Codex and ship its plugin
---

The plugin and the docket skill have been checked on Claude Code only. On 2026-11-04 the maintainer chose to leave Codex untested and not to write a Codex plugin yet. What is open, all of it needs a real install:

- How a plugin skill is invoked in Codex (the name and form that load `docket-init` and `docket-doctor`).
- Whether the Codex CLI reads `.claude-plugin/marketplace.json`, or needs its own marketplace or plugin files.
- Whether superpowers, installed by `docket-init`, can be used in the same Codex session or needs a restart.

`dkt init` already writes the docket skill to `.agents/skills/docket/` for Codex.
