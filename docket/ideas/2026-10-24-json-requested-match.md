---
title: Match --json only as a flag of its own
---

Found while reviewing `add-init-and-new`, 2026-10-24. Pick it up in the
change that touches the same code, or fix it directly when it changes no
spec sentence.

`jsonRequested` (`internal/cli/run.go`), used only after a parse failure or
for an unknown command, also matches `--json` given as another flag's value
(`--default-branch --json --bogus`) and a bare `json` positional, so such a
failing call can answer in JSON.
