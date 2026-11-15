---
title: "Let ideas and the order share a direct merge, and say when a record's review is written"
kind: prompt
source: "docket/ideas/bookkeeping-commits.md"
---

## What

The docket skill (`shipped/skill/SKILL.md` and the two copies installed here): a new idea and an edit to the order still go in a direct merge apart from the work's pull request, but may share one; and the light review says a record's `## Review` is written after the review, in the commit that answers it, with `dkt lint` run again then.

## Why

"A direct merge of its own" read as one pull request for the ideas and another for the order (#145, #146). Nothing said when the review section is written, though `dkt lint` reports it as empty until the review is done.

## Verification

The shipped skill and its two installed copies are identical; `dkt inspect` of this build reports nothing; `gofmt -l .`, `go vet ./...` and `go test ./...` pass. Not verified: an agent following the new wording.

## Review

