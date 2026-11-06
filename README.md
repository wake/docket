# Docket

Docket is a spec-driven development workflow for coding agents. It combines [superpowers](https://github.com/obra/superpowers) and [OpenSpec](https://github.com/Fission-AI/OpenSpec), and `dkt` is the CLI that supports it.

An agent triages each piece of work into one of four paths, by what the work changes: a **full review** (spec, plan, reviews), a **light review** (a record and one review), a **checked merge** (only comments or whitespace), or a **direct merge** (docs, version bumps, dependency upgrades).

The fixed workflow is in the [docket skill](shipped/skill/SKILL.md). The steps a project can customize, such as how it reviews and who does the work, are in the project's own `docket/workflow.md`, which starts from [this template](shipped/workflow/workflow.md).

## Requirements

- git
- [OpenSpec](https://github.com/Fission-AI/OpenSpec/blob/main/install.md) 1.14.0 or later
- [superpowers](https://github.com/obra/superpowers)
- Claude Code. Codex has not been verified yet.
- macOS or Linux, on Intel or ARM. There is no Windows executable yet.

## Install

Install the plugin:

```sh
claude plugin marketplace add wake/docket
claude plugin install docket@docket
```

Then, in your project, run `/docket:docket-init` in Claude Code. It checks the requirements and offers to install what is missing, including `dkt`, then sets the project up and fits `docket/workflow.md` to how you work. See [what it does](skills/docket-init/SKILL.md).

To install only `dkt`:

```sh
curl -fsSL https://raw.githubusercontent.com/wake/docket/main/install.sh | sh
```

This puts the latest release at `~/.local/bin/dkt`, after verifying its SHA-256. To choose a version, set `DKT_VERSION`:

```sh
curl -fsSL https://raw.githubusercontent.com/wake/docket/main/install.sh | DKT_VERSION=v0.16.0 sh
```

## Commands

Once a project is set up, the agent loads the docket skill when development work starts, and runs these commands as the workflow calls for them.

| Command | What it does |
|---|---|
| `/docket:docket-init` | Sets Docket up in the current project (plugin skill, run by hand) |
| `/docket:docket-doctor` | Checks the project's Docket setup and helps fix it (plugin skill, run by hand) |
| `dkt init` | Writes Docket's files into the project, keeping existing files that differ |
| `dkt inspect` | Reports on the project's Docket setup and writes nothing |
| `dkt new` | Opens a change on the full review path |
| `dkt record` | Opens a record on the light review path |
| `dkt sync` | Ticks the tasks of a plan or a record from commit trailers |
| `dkt lint` | Checks the structure of changes, records and main spec titles |
| `dkt conflicts` | Finds active changes that touch the same requirement |

Run `dkt --help` for the arguments. Every command takes `--json`.

## Developing Docket

This repository uses Docket's own workflow: see [AGENTS.md](AGENTS.md). Releases are described in [docs/release.md](docs/release.md).

## Previous version

Docket was rebuilt for v0.16.0. The previous version is on the [`legacy`](https://github.com/wake/docket/tree/legacy) branch, and its last release is v0.15.0; both are kept for reference only.

## License

[MIT](LICENSE)
