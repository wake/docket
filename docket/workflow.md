---
template: 1
---

The `template` value in the front matter is the template's version. Do not
change it by hand: when the template is updated, this is detected and the
update starts on its own.

# Development workflow

The steps of the Docket workflow that this project can customize. When the
docket skill's workflow names a section of `docket/workflow.md`, follow the
section here with that heading. The headings are the names the sections are
looked up by: do not rename or translate them; their contents can be changed
for the project. In this project, unless the user asks otherwise in this
conversation, these steps take precedence over personal usual instructions
and habits.

## Full review

### Plan review

Before any code, review the plan once, together with the proposal, the delta
specs and design.md (see "Reviewers"), and fix what the review finds.

### Task execution

Use superpowers:subagent-driven-development, with a fresh implementer agent
for each task, test first. Each implementer gets the Reuse rows for its task
and reports every exported symbol it added, with why no existing one fit.
Structural changes and behavior changes go in separate commits.

### Task review

Only a task marked `**Review:** judgment` gets its own reviewer, in a fresh
context. Every other task that changes code gets the mechanical checks
instead: its new tests are seen failing before the code and passing after,
and mutating the lines it changed makes a test fail. A task that changes no
code, such as one that only edits docs, is checked by its own verification
steps.

### Pre-merge review

Review the whole change once, by two reviewers in parallel: one on a
different model, asked to attack (wrong results, edge cases, partial
failure, the same defect elsewhere in the code), and one in a fresh context,
checking the code against the proposal and the delta specs (see "Reviewers"
when no other model is available). After fixes, only a large fix is reviewed
again.

### Splitting pull requests

A change is one pull request; the order of its commits shows its layers.
Split it only when it mixes unrelated concerns, when a structural refactor
is mixed with behavior (the refactor goes first, in its own pull request),
or when product code and tests together run far past about 1000 lines. Do
not stack pull requests.

### When to archive

Until this repository has a release process, archive each change in its own pull request, as its last commit.

## Light review

### Verification

Verify by the kind of edit:

- A fix: write a regression test and see it fail before the fix.
- A restructuring: the tests that cover the code pass before and after, and
  mutating the changed lines makes one fail; when none fails, first add the
  tests as a test-only edit.
- Prompt text: check it mechanically, confirming that it still loads and
  that every copy says the same; do not write tests that only check that
  strings are present.
- Tests: mutate the code they cover; the tests must fail.
- Check settings: run them; the pull request's own CI run uses the new
  settings, and the review looks for a check that got weaker.

### Review

Before merging, have the whole diff reviewed once (see "Reviewers").

## Checked merge

### Comment check

Have a fresh agent, given only the diff and the triage's definitions,
confirm that every changed line is a comment or whitespace and that no
comment changes how the code builds or runs.

## Direct merge

### Checks

Run only the checks that read what the edit changed: none when it changes
only docs that nothing builds, embeds or tests; the build and the tests when
it changes code, test files or dependency manifests. No review.

## Shared

### Where to work

Do each piece of work in its own git worktree.

### How to merge

Open a pull request and merge it with a merge commit or a rebase.

### Reviewers

A required review is done by a reviewer on a different model. If there is
none, by the same model in a fresh context. If that is not possible either,
the pull request says it had no independent review and asks a person to
review it. The author's own reading is not a review.

### Parallel work

When sessions work in parallel, each change belongs to one session, and the
session that implements a change does not review it. Merge one change at a
time: rebase it on the default branch, run the full test suite, and review
again only where the rebase had conflicts.
