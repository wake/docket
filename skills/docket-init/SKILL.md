---
name: docket-init
description: Initialize the project directory with the Docket management tool
disable-model-invocation: true
---

# Docket init

Docket is a development workflow management framework built on superpowers and OpenSpec, working with git. This skill sets Docket up in the current project directory and initializes it.

0. With the user's consent, you may make the writes below; otherwise the user makes them.

1. Check the whole environment at once:
   - `git rev-parse --show-toplevel`
   - `dkt --version`
   - `openspec --version`
   - the superpowers skills: brainstorming, writing-plans, subagent-driven-development

2. Fill in what is missing:
   - not a git directory: find the folder where `git init` should run, have the user confirm it, run it there, and do the remaining steps in that folder. If it is a home directory, a temporary directory, or does not look like a project, first ask for the project's folder.
   - dkt: `curl -fsSL https://raw.githubusercontent.com/wake/docket/main/install.sh | sh` (installs `~/.local/bin/dkt`)
   - OpenSpec: follow https://github.com/Fission-AI/OpenSpec/blob/main/install.md
   - superpowers: in Claude Code, `claude plugin install superpowers@claude-plugins-official`; in Codex, `codex plugin add superpowers@openai-curated-remote`

   If anything is still missing (the user declined or an install failed), stop, and once the user has dealt with it, check again from step 1.

3. Run `dkt init --json --tools <coding agent>`, with `<coding agent>` the coding agent you are using, such as `claude` or `codex`. Then follow its instructions for the rest, commit only the files `dkt init` reports it wrote, and help reload the skills.

4. Run `dkt inspect --json` to confirm the setup is complete, or whether the docket-doctor skill is needed.
