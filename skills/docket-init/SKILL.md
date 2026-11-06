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

4. When `dkt init` reports `docket/workflow.md` as created, integrate the user's instructions and habits into it:
   - Sources: the user's instructions and memory already in your context (such as global instruction files), where they say how planning, reviews, verification and merging are done, and the tools you can use (other models, subagents, plugins, skills, commands). Read user-level files that are not in your context only with the user's consent.
   - Keep every step of the template: where the user's instructions say how to do a step, rewrite it their way, naming the tools and settings; add the user's extra steps to the matching section, without adding sections; where they contradict the template, keep the template's step and mark the conflict for the user to decide. Instructions that conflict with the docket skill's steps (such as asking for squash merges) are listed as conflicts too.
   - Keep the headings and the `template` value in the front matter as they are, and write in the file's own language.
   - Rather than showing the user the whole file, tell them the main points of the workflow and of the integration, listing where it differs from the template and the conflicts to decide; revise it until the user agrees, and only then write it and commit. When there is nothing to integrate, keep the default content.

   When `docket/workflow.md` already existed before `dkt init` (for example, a teammate set it up), it holds the project's shared steps: do not change it to fit the user's personal habits; when the user wants to adjust it, run docket-doctor.

5. Run `dkt inspect --json` to confirm the setup is complete, or whether the docket-doctor skill is needed.
