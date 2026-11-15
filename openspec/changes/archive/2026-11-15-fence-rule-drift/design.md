# Design

## Context

`markdown.FenceMask` follows the fence rule of OpenSpec 1.14.3: a line opens fenced code when it starts, after optional space, with three or more backticks or tildes. Every reader of change files in dkt masks with it, and `fence-in-comment` follows the same fence lines. OpenSpec 1.16.0 added one condition to its opener (`getFenceMarker` in `dist/core/parsers/code-fence.js`): a backtick fence is not an opener when the rest of the line, as its `(.*)` captures it, holds a backtick. The closing rule is unchanged. OpenSpec 2.0.0, the version checked here, keeps both rules as 1.16.0 has them. See proposal.md - Why for the effect.

## Goals / Non-Goals

**Goals:**
- Name lines the 1.14 rule, which dkt follows, and the 1.16 rule read differently, so that a file with no finding is read the same by both.

**Non-Goals:**
- Changing `FenceMask` or any reader built on it.
- Comparing dkt with OpenSpec's readers that use a third rule; see the last decision.

## Decisions

### Report the lines OpenSpec 1.16 reads outside fenced code

Call a line that starts, after optional space, with three or more backticks and holds another backtick after them a backtick-info line. The check reads the file with the 1.16 rule, as `FenceMask` does but with the 1.16 opener, and reports each backtick-info line that this reading leaves outside fenced code.

Each finding is a line the two rules read differently: the 1.14 rule reads every backtick-info line as fenced code, as an opener when it is outside code and as content when it is inside. And a file with no finding is read the same way by both: up to the first line where the readings differ, both are in the same state; inside code they close on the same lines, and outside code they differ only on a backtick-info line, which 1.16 then reads outside code. So the first difference is always a reported line.

A reported line may not be the cause: after an earlier reported line, the two readings can swap state, and a later backtick-info line that 1.16 reads outside code while dkt is inside a fence is reported too. Rewriting it is still right: 1.16 and CommonMark already read it as inline code, so the rewrite changes nothing for them and only removes an opener that 1.14 alone can see.

The opener is matched with the expression `FenceMask` uses, so the check and the readers cannot drift apart; the 1.16 reading shares `FenceMask`'s walk and closing rule and differs only in the opener. The text after the marker is cut at the first U+2028 or U+2029, as JavaScript's `.` in `getFenceMarker`'s `(.*)` stops there; dkt has already split lines at CR and LF.

Alternatives considered:
- Walking only dkt's reading and reporting each opener it takes that 1.16 would not. Simpler, but after an earlier finding it reports a line 1.16 also reads inside code, and misses a line 1.16 reads outside code while dkt is inside a fence, so a file with no finding could still be read differently.
- Reporting every line where the two masks differ. The first difference is the same line, but the rest are mostly ordinary lines far from any marker; the finding should name a line the author can rewrite.
- Reporting only the first difference in a file. Fewer findings, but the author then needs one lint run per line.

### Checked where fence-in-comment is checked

The rule runs on the same files as `fence-in-comment`: each `.md` file of a `docket` change, each record and issue lint reads, and, when no change is named, each main spec in the walk that checks titles. These are the files dkt reads with the mask. The two fence rules are applied together at each of those places, so a new place gets both.

A line inside an HTML comment that is reported by `fence-in-comment` may also be reported by this rule. Each finding is true, and one rewrite with single backticks settles both.

### OpenSpec's readers with a simpler rule are left out

Two readers in OpenSpec 2.0.0 do not use `buildCodeFenceMask` as the requirement parsers do:

- `extractFirstPurposeLine` in `core/references.js` toggles a fence on any line starting with three backticks or tildes, closing it by marker kind only, with no check of length or trailing text. It agrees with neither rule, but it reads only the first line of a spec's Purpose, for the index of referenced specs, which dkt does not read.
- `extractPurposeSection` in `core/specs-apply.js` blanks HTML comments before it masks fences, with the 1.16 rule. It reads only the Purpose that archive copies into a new main spec. Outside comments it reads fences as 1.16 does; its comment-first order is the case `fence-in-comment` covers.

Neither gives a line a third reading that changes the requirements, steps, headings or tables dkt reads, so the rule compares only the fence rules of OpenSpec 1.14 and 1.16.

## Risks / Trade-offs

- [A finding after an earlier one may be a consequence, not a cause] → Every finding is a line the two rules read differently and that reads correctly once rewritten; a file with no finding is read the same under both rules.
- [OpenSpec changes its fence rule again] → The check is tied to the 1.16 condition named in `Context`; a later rule change would be a new idea, found the way this one was.
