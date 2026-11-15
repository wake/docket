# Design

## Context

`markdown.FenceMask` follows the fence rule of OpenSpec 1.14.3: a line opens fenced code when it starts, after optional space, with three or more backticks or tildes. Every reader of change files in dkt masks with it, and `fence-in-comment` follows the same fence lines. OpenSpec 1.16.0 added one condition to its opener (`getFenceMarker` in `dist/core/parsers/code-fence.js`): a backtick fence is not an opener when the rest of the line, as its `(.*)` captures it, holds a backtick. The closing rule is unchanged. OpenSpec 2.0.0, the version checked here, keeps both rules as 1.16.0 has them. See proposal.md - Why for the effect.

## Goals / Non-Goals

**Goals:**
- Name each line where the 1.14 rule, which dkt follows, opens fenced code and the 1.16 rule does not.

**Non-Goals:**
- Changing `FenceMask` or any reader built on it.
- Comparing dkt with OpenSpec's readers that use a third rule; see the last decision.

## Decisions

### Report the opener, in dkt's own reading

The check walks the lines as `FenceMask` does and, at each line `FenceMask` reads as an opener, reports it when its marker is a backtick and the text after the marker holds a backtick. The new rule is strictly narrower than the old one, so in any file the first line where the two readings differ is such a line; reporting the opener names the cause, not a symptom many lines below.

Lines after a reported opener are read as `FenceMask` reads them. A later reported line may fall where OpenSpec 1.16 is already inside code because of the earlier one; the finding is still true of the line itself, and rewriting it is harmless. A file with no finding is read the same way by both rules, since they then take the same openers and share the closing rule.

The opener is matched with the expression `FenceMask` uses, so the check and the readers cannot drift apart. The text after the marker is cut at the first U+2028 or U+2029, as JavaScript's `.` stops there; dkt has already split lines at CR and LF.

Alternatives considered:
- Computing a second mask under the 1.16 rule and reporting where the masks differ. The first difference is the same line, but later ones are often far from any marker; the finding should name the marker.
- Reporting only the first such line in a file. Fewer findings, but the author then needs one lint run per line.

### Checked where fence-in-comment is checked

The rule runs on the same files as `fence-in-comment`: each `.md` file of a `docket` change, each record and issue lint reads, and, when no change is named, each main spec in the walk that checks titles. These are the files dkt reads with the mask. The two fence rules are applied together at each of those places, so a new place gets both.

A line inside an HTML comment that is reported by `fence-in-comment` may also be reported by this rule. Each finding is true, and one rewrite with single backticks settles both.

### OpenSpec's readers with a simpler rule are left out

Two readers in OpenSpec 2.0.0 do not use `buildCodeFenceMask` as the requirement parsers do:

- `extractFirstPurposeLine` in `core/references.js` takes any line starting with three backticks or tildes as a fence and closes it by marker kind only. It reads the first line of a spec's Purpose for the index of referenced specs. For the lines this rule reports it agrees with OpenSpec 1.14 and dkt, so the rewrite brings it in line too; its other differences concern the Purpose summary, which dkt does not read.
- `extractPurposeSection` in `core/specs-apply.js` blanks HTML comments before it masks fences, with the 1.16 rule. It reads only the Purpose that archive copies into a new main spec. For a reported line it agrees with 1.16; its comment-first order is the case `fence-in-comment` covers.

Neither gives a line a third reading that changes the requirements, steps, headings or tables dkt reads, so the rule compares only the two `buildCodeFenceMask` rules.

## Risks / Trade-offs

- [A line that only becomes a reported opener once an earlier finding is fixed is not reported in the same run] → Every finding names a real opener; fixing them and running lint again settles the rest, and a file with no finding reads the same under both rules.
- [OpenSpec changes its fence rule again] → The check is tied to the 1.16 condition named in `Context`; a later rule change would be a new idea, found the way this one was.
