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
     - Its version equals the `template` of `docket/workflow.md`: ask the user whether to use it. If yes, copy it to `docket/workflow.md`, then settle the mode: `sync` (the project keeps following the user template, and `dkt inspect` reports, section by section, where the two drift apart) or `independent` (each goes its own way). When `result.user_workflow.default` is set, propose that mode; otherwise ask the user. Once the user agrees, run `dkt workflow mode <mode>`. If no, go on as when there is none.
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

6. With Claude Code, finally offer the session activity hooks, which let `dkt snapshot` show which works a session is on. When a hook in the user's `~/.claude/settings.json` already calls `dkt activity`, skip this step.
   - Explain, then ask: on this computer, in every Claude Code session, each prompt sent, each tool call and each turn end runs `dkt activity`, which writes a small file under the state folder (`$XDG_STATE_HOME/docket/sessions/`, or `~/.local/state/docket/sessions/` when that is not set) with the session's state, directory and time; nothing is written in any repository, a state not refreshed within 15 minutes counts as absent, and old files are removed on their own. When the user declines, skip it and write nothing.
   - Once the user agrees, get dkt's absolute path with `command -v dkt`; when it finds nothing, stop and ask the user to install dkt or put it on the PATH first. Write each hook's command as `"<path>" activity <state> 2>/dev/null || true`, so that it stays silent and never stops the session, even after dkt is removed.
   - In `hooks` of `~/.claude/settings.json`, add: `UserPromptSubmit` and `PreToolUse` with `working`, `Notification` and `Stop` with `waiting`, `SessionEnd` with `ended`. Each entry is `{"hooks": [{"type": "command", "command": "<command>"}]}`, and the `PreToolUse` one also has `"matcher": "*"`. Keep the settings and hooks already in the file, adding the new entries after each event's existing ones. When the file does not exist, create it with these hooks alone; when it is not valid JSON, or its `hooks` is not an object, write nothing and tell the user why.
   - Finally, tell the user how to remove them: delete the five hooks that call `dkt activity` from `~/.claude/settings.json`; the state folder can simply be deleted.
