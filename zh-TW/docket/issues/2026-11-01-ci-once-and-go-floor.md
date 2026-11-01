---
title: 每個 pull request 只跑一次 CI，並測試 Go 的下限版本
source: review of add-init-and-new, 2026-10-24; moved from docket/ideas/2026-10-24-ci-triggers-and-go-floor.md
---

## Current

.github/workflows/ci.yml 在 `push` 與 `pull_request` 都會執行（第 3-5 行），所以有 pull request 的分支會跑兩次。每個 job 都用 `go-version: stable`（第 45、57、88 行）。

## Expected

go.mod 宣告 `go 1.24` 為下限，所以 CI 應該測試它。有 pull request 的分支應該只跑一次。

## Unchanged

僅文件則跳過的規則、macOS 與 Linux 的矩陣，以及會執行的檢查。

## Found

`add-init-and-new` 的審查，2026-10-24。main 上仍存在。這是檢查設定。
