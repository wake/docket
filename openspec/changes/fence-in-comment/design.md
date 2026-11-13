## Context

`internal/markdown` finds fenced code with `FenceMask`, which follows the fence rule of OpenSpec 1.14.3, and then blanks HTML comments outside fenced lines with `BlankComments`. Every reader of change files masks this way: `change.ParsePlan`, `record.Parse`, `specdoc`, `setup` and the lint rules. OpenSpec 1.16.0 reads specs the same way: its requirement parser masks fences and never looks at comments, and its check for unapplied delta text blanks comments only outside fences. See proposal.md - Why for the effect of a fence marker inside a comment.

## Goals / Non-Goals

**Goals:**
- Find the lines where reading fences first and reading in document order disagree, so lint can name them.

**Non-Goals:**
- Changing `FenceMask`, `BlankComments` or any reader built on them.

## Decisions

### A second reading in document order, used only by the check

The check reads the file once more, in document order: outside a comment, a line that opens fenced code starts code, as `FenceMask` would; outside code, `<!--` opens a comment and the first `-->` or `--!>` after it closes it, as `BlankComments` does. Inside a comment it follows the fence lines the same way `FenceMask` does, starting from no open code. When the comment closes while such code is still open, the line that opened it is reported: there `FenceMask` runs the code past the comment.

The fence lines are matched with the same expressions `FenceMask` uses, so the check and the readers cannot drift apart.

The comments checked are the ones this reading finds, in document order, as CommonMark reads them; a line this reading finds inside code is not checked, even where `FenceMask` has fallen out of step and reads it inside a comment. Then a file the check passes is read the same way by both: at the start of each comment both are outside code, inside it `FenceMask` follows the same fence lines, and at its end no code is open, so `BlankComments` closes it on the same line. A marker `FenceMask` would only see after an earlier one is not reported, since fixing the earlier one changes what that later line is.

Alternatives considered:
- Reporting every fence line inside a comment. Simpler, but it would report a code block commented out whole, which hides nothing; a lint finding has to be fixed before archiving, so it should only name a real problem.
- Comparing the visible lines of both readings. It finds the same cases, but the first difference is often far from the marker that caused it, and the finding should name the marker.

### Checked where the placeholder rule is checked

The rule runs on the same files as `placeholder`: each `.md` file of a `docket` change, and each record and issue lint reads. These are the files whose headings, steps and tables dkt reads with the mask. A change of another schema is not checked, as no docket rule is.

## Risks / Trade-offs

- [After a reported marker, the two readings stay out of step, so a marker `FenceMask` would only see later is not reported in the same run] → Each comment this reading finds is checked on its own lines, starting from no open code, so every finding names a marker inside a comment; fixing them and running lint again settles the rest, and a file with no finding is read the same way by both.
- [A comment that only ends at the end of the file never closes, so a marker in it is not reported] → Both readings hide everything after such a marker anyway, so nothing is read differently.
