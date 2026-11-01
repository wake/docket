---
title: 用 setup.Init 建立測試 fixture
source: review of add-init-and-new, 2026-10-24; moved from docket/ideas/2026-10-24-install-schema-helper.md
---

## Current

`testutil.InstallSchema` 及其同類函式（internal/testutil/openspec.go:104-139）重複了 `setup.Init`（internal/setup/setup.go:151）走訪並寫入檔案的迴圈。

## Expected

`project-setup` 定義 `dkt init` 安裝什麼。fixture 應該由同一份程式碼產生，才不會與它產生偏差。

## Unchanged

fixture 最後得到的檔案；測試結果；舊 schema 的 fixture。

## Found

`add-init-and-new` 的審查，2026-10-24。main 上仍存在。
