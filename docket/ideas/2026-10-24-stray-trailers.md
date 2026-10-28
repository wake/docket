---
title: Find task commits whose trailers git cannot read
---

Recorded 2026-10-24 at the first release.

## Problem

Docket reads `Change:` and `Task:` with git's trailer parser, which only
looks at the last paragraph of a commit message. A commit made with one `-m`
per trailer puts each one in its own paragraph:

```
fix(archive): refuse a read-only target

Change: add-archive

Task: 3

Co-Authored-By: ...
```

git then reads only `Co-Authored-By` as a trailer. `dkt sync` and
`dkt archive` see no task commit, and nothing reports it as long as the task
has another commit with good trailers. At the first release, ten commits of
`add-archive` (all from its review fix rounds) were left out of its
`evidence.md` this way; they were listed by hand under "Added by hand at
release".

## Direction

Report a commit in the change's range whose message has a line
`Change: <slug>` that git does not read as a trailer, naming the commit and
the fix (keep the trailers in one final paragraph). Candidates for where it
runs: `dkt sync --check` and the plan check of `dkt archive`, which already
read the range, or the planned `dkt check` pull request gate. This changes
the `plan-sync` or `change-archive` spec, so it needs a change.

## Open questions

- Whether a stray trailer is a failure (exit 1) or a warning.
- Whether to look for `Task:` without `Change:` too.
