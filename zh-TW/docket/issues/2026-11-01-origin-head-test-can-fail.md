---
title: 讓 TestOriginHEADNotASource 有可能失敗
source: review of add-init-and-new, 2026-10-24; moved from docket/ideas/2026-10-24-origin-head-test.md
---

## Current

`TestOriginHEADNotASource`（internal/scan/scan_test.go:120）本身不可能失敗：`origin/HEAD` 指向預設分支的 tip，所以它從不會被列為額外的 source；實際被測試的只有 `internal/gitx` 裡的防護。

## Expected

`workspace-scan`（spec 第 38 行）列出掃描所讀的 source；`origin/HEAD` 不是其中之一。如果掃描列出了它，測試應該失敗。

## Unchanged

正式環境的程式碼；gitx 防護的測試。

## Found

`add-init-and-new` 的審查，2026-10-24。讓掃描列出它，並確認測試失敗。
