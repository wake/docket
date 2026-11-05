# Proposal

## Why

The `docket` skill has the agent run `dkt lint <slug>` before opening a pull request, and the archive guidance runs it before archiving: the plan rules the schema states (task headings, the Reuse table, scenario claims, cancelled and compensating tasks) and the proposal's front matter only hold if something checks them. It is the last of the three commands this repository needs before it switches to Docket's own workflow.

## What Changes

- Add `dkt lint [<slug>...]`: it lints the changes named, or every active change in the worktree, and then, when none is named, the titles of the main specs.
- For a `docket` change it reports:
  - front matter of `proposal.md`: a missing or malformed `title`, `kind` or `source`, and a new capability with no name under `specs:`;
  - proposal sections: Why, What Changes, Non-Goals, Capabilities and Impact present, Non-Goals not empty;
  - template text left in any file of the change (an HTML comment or placeholder copied from the shipped templates);
  - with a plan.md: task headings (`### Task N: <name>`, unique numbers, Cancelled Tasks included), the Reuse table, `**Scenarios:**` lines that name existing scenarios, every ADDED or MODIFIED scenario claimed by a task, scenario names that clash once slugged, Cancelled Tasks bullets, and `**Compensates:**` lines;
  - what `openspec validate` reports as errors.
- For a change of another schema it checks only that each capability the change adds has a name, as docket-doctor needs.
- A main spec's title is its first line, `# <name> Specification`; a first line that still holds the capability's folder name, as OpenSpec writes it, is a missing title.
- Deltas are read through `openspec show --json`, not parsed by dkt, so `dkt lint` needs the OpenSpec CLI, 1.14.0 or later (the first release whose JSON names requirements and scenarios; the minimum was raised to it for this).
- The command table accepts any number of positional arguments for a command that asks for it.
- cli-output: `dkt lint` joins the commands that need a repository.

## Non-Goals

- Routing (designs and plans written under `docs/superpowers/`), the default branch, and `dkt inspect`'s trailer and title checks: the next change, which resolves the default branch.
- Records, issues, ideas and the order file: they come with `dkt record` and `dkt issue`.
- The `**Review:**` and `**Files:**` lines of tasks: no rule asks for them yet.
- Fixing anything: `dkt lint` only reports.

## Capabilities

### New Capabilities
- `change-lint`: what `dkt lint` checks in a change and in the main specs' titles, and how it reports.

### Modified Capabilities
- `change-metadata`: how the front matter is read back, including `specs:`, and the title of a main spec.
- `cli-output`: `dkt lint` needs a repository.

## Impact

- Adds `dkt lint`, front matter reading, Markdown tables and code spans, the OpenSpec `show` and `validate` readers, and their tests; extends the real-OpenSpec test.
- Dependencies: still only the Go standard library, the system `git` and the OpenSpec CLI.
