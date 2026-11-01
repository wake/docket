---
title: 只在 --json 是獨立旗標時才比對
source: review of add-init-and-new, 2026-10-24; moved from docket/ideas/2026-10-24-json-requested-match.md
---

## Current

`jsonRequested`（internal/cli/run.go:166）會比對到作為其他旗標之值的 `--json`（`--default-branch --json --bogus`），以及單獨的位置參數 `json`，所以這類失敗的呼叫會以 JSON 回應。

## Expected

`cli-output`（spec 第 10 行）：`--json` 是旗標；只有它能選擇 JSON 輸出。名為 `json` 的值或位置參數不應該。

## Unchanged

解析失敗後或遇到未知命令時的 `--json`、`-json` 與 `--json=true`。

## Found

`add-init-and-new` 的審查，2026-10-24。main 上仍存在。
