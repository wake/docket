# Design

## Context

See `proposal.md` - Why. What was checked with OpenSpec 1.16.0 before choosing:

- A change folder cannot be nested: OpenSpec refuses a slug holding `/`, and calls a wrapping folder `not a change`.
- A spec can be nested, and `dkt lint`, `dkt conflicts` and archive all follow the path. Moving a main spec needs its own change with `skip_specs: true`, and a change left on the old path either aborts at archive (MODIFIED) or creates a second main spec there (only ADDED).
- Front matter at the top of a main spec survives `openspec archive`, but a delta's does not reach the main spec, and `dkt lint` then reports `title-missing`, since it reads the title from the first line.
- A file `docket/domains.md` affects none of `dkt lint`, `status`, `inspect` or `conflicts`, and `dkt init --force` leaves it alone.

## Goals / Non-Goals

**Goals:**

- A viewer gets one JSON tree holding a project's areas, capabilities, titles and active changes, without inferring anything.
- A coding agent can ask which areas exist when placing a new capability.
- Reclassifying a capability touches one file and no spec.

**Non-Goals:**

- The proposal's non-goals. At design level, also: normalizing node names beyond removing surrounding whitespace, and a length limit on them.

## Approaches

Three ways to record the classification were compared. The maintainer chose the tree.

1. **The storage path** (`openspec/specs/user/payment/refund/spec.md`). No code at all, but one spec has one place, node names are limited to path rules, and every reclassification moves a file, with the risk above.
2. **Front matter in each main spec** (`domain: user/payment`). The value travels with the spec, but it needs the title rule changed, a new way for new capabilities to carry it past archive, and a triage exception, since editing `openspec/specs/` is a full review. It depends on OpenSpec keeping front matter it does not define. It also cannot hold an empty node, and renaming a node means editing every spec under it.
3. **A separate tree file holding the mapping** (`docket/domains.md`). Chosen. It needs none of the above, holds empty nodes, and a rename is one line. The costs: reading a spec does not show its node, parallel changes may conflict in this one file, and a capability that is renamed or removed leaves a stale line, which `domain-unknown` reports.

A `domain` field on a change was also considered and dropped: a change can touch capabilities in several nodes, so one value cannot describe it, and where a change belongs follows from the capabilities in `changes`.

## Decisions

**A Markdown list, not YAML.** The repository has no third-party dependency and writes its own small parsers, as for `docket/order.md`. A list reads well as is, shows empty nodes, and takes any language. A plain item is a node and a code span is a capability, which is the same distinction a reader sees.

**A bad line stops the answer.** A tree with a line that cannot be placed could put a capability under the wrong node, so `result` is null on `domain-format` and `domain-duplicate`, as `dkt status` gives no next item when the order has a problem. Every bad line is reported at once, not only the first: a bad item still takes its place in the stack, so its children do not report a false level skip.

**A stale capability does not stop the answer.** `domain-unknown` means one line is out of date, and the rest of the tree is right. The entry stays, with `main` false and `changes` empty, which a viewer can show as it is.

**An unreadable source skips `domain-unknown`.** Without the full list of capabilities, a line could be reported as stale when it is not. The rest of the result stands, as `dkt status` reports `next` from what it could read.

**The worktree, not the default branch.** `dkt status` reads the default branch because an order edit counts once merged. Here a coding agent working on a change must see the tree its own branch is editing, and `dkt lint` already reads the worktree. Scanning other worktrees and branches, as `dkt conflicts` does, would add a source of conflicting copies with no need in this change.

**One node per capability.** `domain-duplicate` keeps the one-to-one reading of a leaf. It can be relaxed later without breaking any file already valid.

**`dkt lint <slug>` skips the tree.** Same reason as main spec titles: a problem in an area the change does not touch should not stop it. The skill tells an agent editing the file to run `dkt domains`, which reports the same findings.

**Authorization lives in the skill, not the CLI.** `dkt` does not ask or decide. The rule that an agent suggests a node and adds, renames or moves one only with the user's agreement is prompt text. No tool could tell an authorized node from another, so the review of the pull request is the check.

**Shared code.** `internal/domain` holds the parser and `Load`, which `dkt domains` and `dkt lint` both call, so both report the same findings.

## Risks / Trade-offs

- [Two changes add lines at the same place in `docket/domains.md`] → A git conflict in a small list, resolved at rebase; `docket/workflow.md` already merges one change at a time.
- [A capability is renamed or removed by a change that only ran `dkt lint <slug>`] → The stale line surfaces at the next `dkt lint` or `dkt domains` as `domain-unknown`. Nothing depends on the tree to run, so a late report costs little.
- [Visually identical names with different code points] → Compared exactly; both would stay as two nodes, visible in the tree. The standard library has no Unicode normalization, and adding a dependency is not worth it for this.
- [An agent adds a node without asking] → It shows in the pull request diff. `dkt` cannot detect it.

## Migration Plan

None. The file is optional, and a project without it sees only one new command.
