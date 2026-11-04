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
   - not a git directory: find the folder where `git init` should run and have the user confirm it. If it is a home directory, a temporary directory, or does not look like a project, first ask for the project's folder.
   - dkt: `curl -fsSL https://raw.githubusercontent.com/wake/docket/main/install.sh | sh` (installs `~/.local/bin/dkt`)
   - OpenSpec: follow https://github.com/Fission-AI/OpenSpec/blob/main/install.md
   - superpowers: in Claude Code, `claude plugin install superpowers@claude-plugins-official`; in Codex, `codex plugin add superpowers@openai-curated-remote`

3. Run `dkt init --json --tools <AI tool>`, with `<AI tool>` the AI coding tool you are using: `claude` for Claude Code, `codex` for Codex. Follow its instructions for the rest, commit the files involved, and help reload the skills.

4. Run `dkt inspect --json` to confirm the setup is complete, or whether the docket-doctor skill is needed.
