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

4. When `dkt init` reports `docket/workflow.md` as created, look at the user template on this computer first, then integrate the user's instructions and habits into it:
   - The user template: `result.user_workflow` of `dkt inspect --json` gives its `path`, whether it `exists`, and its `version`. It has the same headings and `template` as Docket's template, and holds the user's own habits, not one project's tools, hosts or names.
     - Its version equals the `template` of `docket/workflow.md`: ask the user whether to use it. If yes, copy it to `docket/workflow.md`; if no, go on as when there is none.
     - Its version is older (`user-workflow-outdated`): ask whether to upgrade it first, as docket-doctor upgrades a project's `docket/workflow.md`, then go on as in the point above. When it is not upgraded, go on as when there is none, with the old template as one more reference.
     - Its version is newer than this dkt's (`user-workflow-newer`): ask the user to upgrade dkt first; never downgrade the template.
     - It has no version (`user-workflow-template-unknown`): say how `template` is written (a positive integer in the front matter) and let the user decide which version it is, then go on as for that version; do not guess. Until then, go on as when there is none, with the file as one more reference.
     - There is no user template: once the user agrees, run `dkt workflow refs --json` to read how the user's other registered projects do each step, as references beside the user's instructions: section by section, show the template's step and each way, naming which projects do it which way, and let the user decide each.
   - Sources: the user's instructions and memory already in your context (such as global instruction files), where they say how planning, reviews, verification and merging are done, and the tools you can use (other models, subagents, plugins, skills, commands). Read user-level files that are not in your context only with the user's consent.
   - Keep every step of the template: where the user's instructions say how to do a step, rewrite it their way, naming the tools and settings; add the user's extra steps to the matching section, without adding sections; where they contradict the template, keep the template's step and mark the conflict for the user to decide. Instructions that conflict with the docket skill's steps (such as asking for squash merges) are listed as conflicts too.
   - Keep the headings and the `template` value in the front matter as they are, and write in the file's own language.
   - Rather than showing the user the whole file, tell them the main points of the workflow and of the integration, listing where it differs from the template and the conflicts to decide; revise it until the user agrees, and only then write it and commit. When there is nothing to integrate, keep the default content.
   - When there was no user template at the start, finally ask whether to save the result as the user template, at `result.user_workflow.path`. Before saving, point out what belongs only to this project (tools, hosts, names), and leave it out of the saved version unless the user chooses to keep it.

   When `docket/workflow.md` already existed before `dkt init` (for example, a teammate set it up), it holds the project's shared steps: do not change it to fit the user's personal habits; when the user wants to adjust it, run docket-doctor.

5. Run `dkt inspect --json` to confirm the setup is complete, or whether the docket-doctor skill is needed.
