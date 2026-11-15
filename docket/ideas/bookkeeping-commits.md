---
title: Ideas, the order and a record's review each cost a pull request or a commit of their own
---

Of the twelve pull requests from #138 to #149, six were bookkeeping: three version bumps, two for ideas and the order, one fixing a pointer. Two places add steps without adding a check:

- The skill says a new idea and any edit to the order are each "a direct merge of its own". Read literally, adding ideas and ordering them are two pull requests, as #145 and #146 were. Nothing reads the two apart.
- A record's `## Review` can only be written after the review, so it is a commit after the work's own; the length limit then cost `openspec-2-skill-names` a second one to shorten it.

Direction: say that ideas and the order may go in one direct merge; for the record, either let the pull request hold the review or say that the review section is filled in the commit that answers the review.

Found in the maintainer's review of the workflow, 2026-11-15.
