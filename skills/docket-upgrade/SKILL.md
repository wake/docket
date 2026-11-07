---
name: docket-upgrade
description: >-
  Runs only when the user invokes it. Upgrades Docket on this machine to the
  latest release, then adjusts the Docket files in the current project.
disable-model-invocation: true
---

# Docket upgrade

Upgrade `dkt` and the Docket plugin on this machine to the latest release, then adjust the Docket files in the current project.

0. With the user's consent, you may make the installs and writes below; otherwise skip that step and go on to the next.

1. Check the versions all at once and show them to the user:
   - `dkt --version`
   - the plugin: in Claude Code, `claude plugin list --json`, and read `version` of `docket@docket`
   - the latest release: `curl -fsSL https://api.github.com/repos/wake/docket/releases/latest`, and read `tag_name`

   When the latest release cannot be found, say why and stop. When there is a newer release, read the notes of each release after the current version in `https://api.github.com/repos/wake/docket/releases`, and tell the user in a few sentences what changed.

2. Update `dkt`, unless it is already the latest:
   - `curl -fsSL https://raw.githubusercontent.com/wake/docket/main/install.sh | sh` (installs `~/.local/bin/dkt`)
   - Run `dkt --version` again. When the version has not changed, find the one in use with `command -v dkt`, tell the user, and stop.

3. Update the plugin, unless it is already the latest:
   - In Claude Code: run `claude plugin marketplace update docket`, then `claude plugin update docket@docket`.
   - In another coding agent: tell the user to update the plugin themselves.

   These instructions are still the old version's; the new plugin takes effect once it is reloaded (`/reload-plugins` in Claude Code).

4. Go on only when all three hold: this is a git directory, it has a `docket/workflow.md` file, and `dkt` is the latest or the user agreed to update it. Otherwise skip the rest of this step, tell the user the project was not adjusted, and go to step 5.

   Run `dkt inspect --json`; it is a read-only command. When it exits with 2 or 4, it did not check properly and has no result: say what else it reported and go to step 5. Otherwise act on the result:
   - `schema-outdated`, `skill-outdated`, `guidance-outdated`: name the files that will be updated, run `dkt init --json` once the user agrees, and help reload the skills as its next steps say. When these files are committed, triage them by the docket skill.
   - Any other finding, or `unreadable`: list them and do not fix them here. Suggest that the user run docket-doctor; when the plugin was just updated, it has to be reloaded first.
   - None of these: tell the user the project is up to date.

5. Finally, tell the user what was updated, what was skipped, and what is left to do: reloading the plugin, committing, running docket-doctor when needed.
