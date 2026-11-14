---
title: OpenSpec 1.16 and later find fenced code by a rule dkt does not follow
---

Since OpenSpec 1.16.0, `buildCodeFenceMask` (`dist/core/parsers/code-fence.js`) follows CommonMark: a backtick fence's info string cannot hold a backtick, so a line such as

````markdown
```npm test```
````

is inline code, not a fence opener. OpenSpec 1.14.3, and `markdown.FenceMask`, which follows it (`internal/markdown/markdown.go`), take the line as an opener and read every line after it as code until a closing fence. Under OpenSpec 1.16 or later, OpenSpec reads the requirements after such a line, while `dkt snapshot`, `dkt sync` and dkt's lint rules do not; under 1.14 neither does. OpenSpec 2.0.0 keeps the 1.16 rule.

dkt supports OpenSpec 1.14.0 and later, and CI runs both 1.14.0 and the latest, so dkt cannot read such a line the way every supported version does.

OpenSpec 1.16 is not uniform either: `core/references.js` finds fences by a simpler rule (three backticks or tildes at the start of the line, by marker kind only), and `extractPurposeSection` in `core/specs-apply.js` blanks HTML comments before it finds fences.

Found while checking how OpenSpec reads a fence marker inside a comment, for `fence-in-comment`, 2026-11-13.

Preferred direction (discussed 2026-11-15): keep how dkt reads files and have `dkt lint` report a line that the two rules read differently, so the author rewrites it, for example with single backticks, as `fence-in-comment` did for its case. The alternatives are following the newer rule, which leaves users of 1.14 and 1.15 with the drift the other way, and raising the minimum OpenSpec version.

## Open questions

- Which files the rule checks; the ones `fence-in-comment` checks are the natural set.
- Whether the readers of OpenSpec that use the simpler rule change the answer.
