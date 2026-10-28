---
title: Run CI once per pull request and test the Go floor
---

Found while reviewing `add-init-and-new`, 2026-10-24. Pick it up in the
change that touches the same code, or fix it directly when it changes no
spec sentence.

CI runs on both `push` and `pull_request`, so a pull request branch runs
twice, and it tests `go-version: stable` only, not the `go 1.24` floor in
go.mod.
