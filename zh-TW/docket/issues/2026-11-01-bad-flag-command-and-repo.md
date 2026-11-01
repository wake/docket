---
title: 錯誤旗標後仍要指出命令與 repository
source: review of add-init-and-new, 2026-10-24; moved from docket/ideas/2026-10-24-bad-flag-command-name.md
---

## Current

`dkt --bogus probe --json` 即使在 repository 內，也印出 `"command":""` 與 `"repo":null`（internal/cli/run.go:31 以空名稱且不帶 repo 呼叫 `usageFailure`）。

## Expected

`cli-output`（spec 第 21、28 行）：只有在無法辨識 repository 時，`repo` 才是 null；每個結果都要指出它所描述的對象。

## Unchanged

結束碼 2、usage finding、stdout 上一個 JSON 物件。

## Found

`add-init-and-new` 的審查，2026-10-24；在 main 上以建置好的 `dkt` 重現。
