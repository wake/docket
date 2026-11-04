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
   - not a git directory: name the folder where you will run `git init` and have the user confirm it. If it is a home directory, a temporary directory, or does not look like a project, first ask where the project is.
   - dkt: `curl -fsSL https://raw.githubusercontent.com/wake/docket/main/install.sh | sh` (installs `~/.local/bin/dkt`)
   - OpenSpec: follow https://github.com/Fission-AI/OpenSpec/blob/main/install.md
   - superpowers: follow https://github.com/obra/superpowers/blob/main/README.md#installation

3. Run `dkt init --json`, follow its instructions for the rest, commit the files involved, and help reload the skills.

4. Run `dkt inspect --json` to confirm the setup is complete, or whether the docket-doctor skill is needed.
