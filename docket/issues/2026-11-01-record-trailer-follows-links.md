---
title: Count only regular files as records in record-trailer
source: review of add-record-tracking, 2026-11-01
---

## Current

`dkt lint`'s `record-trailer` check (internal/lint/record.go:237) uses `fs.Stat`, which follows symbolic links. A `Record:` trailer that names `docket/records/<slug>.md` passes when that path is a link to a regular file, though `dkt status` lists no such record.

## Expected

A record is "every regular file `docket/records/<slug>.md`" (workspace-scan, Records found, spec line 74), and lint's other record checks skip a link (`dkt lint <slug>` reports it as `record-missing`, internal/lint/record.go:39). A trailer that names a link names no record, so `record-trailer` should report it.

## Unchanged

A trailer that names a regular record file passes; one finding per commit and value, with the same message; `dkt record` and `dkt sync` still refuse to write through a link.

## Found

Reviews of `add-record-tracking` judged it able to wait. Trial with dkt v0.13.0: on a branch where `docket/records/2026-11-01-fix-x.md` is a link to a regular file, a commit with `Record: 2026-11-01-fix-x` gets no finding, and `dkt status --json` lists no record.
