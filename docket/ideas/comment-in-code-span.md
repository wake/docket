---
title: A comment opener in a code span hides what follows
---

dkt's `markdown.BlankComments` and OpenSpec's comment masking start an HTML comment at every comment opener outside fenced code, code spans included, and a comment that is never closed runs to the end of the file. CommonMark, and so GitHub, reads the opener inside a code span as plain text. In

````markdown
Blank the comment opener `<!--` before finding fences.

### Task 2: Report it
````

GitHub shows the heading, but dkt and OpenSpec read everything after the opener as a comment, up to the next comment closer or the end of the file: headings, task steps and requirements disappear without a finding. The `fence-in-comment` rule does not see it.

Found while writing the plan of `fence-in-comment`, 2026-11-13: a comment opener written in a code span hid the plan's Reuse table and Task headings until the example moved into fenced code.

Preferred direction (discussed 2026-11-15): keep how dkt reads files, the same as OpenSpec, and have `dkt lint` report a comment opener inside a code span, in the files `fence-in-comment` checks. `markdown.CodeSpans`, which the placeholder rule uses, finds the spans.

## Open questions

- Whether to report, too, an opener outside a code span that is never closed and does not start a line: CommonMark reads it as text, while dkt and OpenSpec hide the rest of the file.
- Whether a comment opened in a code span and closed later in the file needs a different message, since it hides only the lines between.
