# Docket workflow steps

The review and verification steps for each path the `docket` skill
triages work into. The skill holds the rest: the triage, the change
folder, the plan's format, commit trailers and merging.

## Full review

1. Review the plan once, together with the proposal, the delta specs and
   design.md, before any code (see Reviews), and fix what the review finds.
2. Execute the plan task by task, with a fresh implementer agent for each
   task, test first. Each implementer gets the Reuse rows for its task and
   reports every exported symbol it added, with why no existing one fit.
   Structural changes and behavior changes go in separate commits.
3. Only a task marked `**Review:** judgment` gets its own reviewer, in a
   fresh context. Every task that changes code gets the mechanical checks
   instead: its new tests are seen failing before the code and passing
   after, and mutating the lines it changed makes a test fail. A task that
   changes no code, such as one that only edits docs, is checked by its own
   verification steps.
4. Before merging, review the whole change once, by two reviewers in
   parallel: one on a different model, asked to attack (wrong results, edge
   cases, partial failure, the same defect elsewhere in the code), and one
   in a fresh context, checking the code against the proposal and the delta
   specs (see Reviews when no other model is available). After fixes, only
   a large fix is reviewed again.
5. Split the pull request only when it mixes unrelated concerns, when a
   structural refactor is mixed with behavior (the refactor goes first, in
   its own pull request), or when product code and tests together run far
   past about 1000 lines. Do not stack pull requests.

## Light review

1. Verify by the kind of edit:
   - a fix: write a regression test and see it fail before the fix;
   - a restructuring: the tests that cover the code pass before and after,
     and mutating the changed lines makes one fail; when none fails, first
     add the tests as a test-only edit;
   - prompt text: check it mechanically, not with tests that look for
     strings: it still loads, and every copy says the same;
   - tests: mutate the code they cover; the tests must fail;
   - check settings: run them; the pull request's own CI run uses the new
     settings, and the review looks for a check that got weaker.
2. Have the whole diff reviewed once before merging (see Reviews).

## Check

Have a fresh agent, given only the diff and the triage's definitions,
confirm before merging that every changed line is a comment or whitespace
and that no comment changes how the code builds or runs. If it does not
confirm, triage again.

## Direct merge

Pass the automated checks. No review.

## Reviews

A required review is done by a reviewer on a different model. If there is
none, by the same model in a fresh context. If that is not possible either,
the pull request says it had no independent review and asks a person to
review it. The author's own reading is not a review.
