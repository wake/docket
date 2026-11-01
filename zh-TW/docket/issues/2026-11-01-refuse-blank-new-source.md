---
title: dkt new 拒絕只含空白的 --source
source: review of add-init-and-new, 2026-10-24; moved from docket/ideas/2026-10-24-blank-source.md
---

## Current

`dkt new --source " "` 會被接受：internal/cli/cmd_new.go:45 只拒絕 `source == ""`。

## Expected

`change-creation`（spec 第 45 行）要求 `--source` 不得為空。`dkt record` 與 `dkt issue` 已用 `strings.TrimSpace` 拒絕空白的 source（cmd_record.go:35、cmd_issue.go:41）；`dkt new` 應該一致，結束碼 2。

## Unchanged

真正的 source 照原樣儲存；其他 usage 錯誤。

## Found

`add-init-and-new` 的審查，2026-10-24。main 上仍存在。
