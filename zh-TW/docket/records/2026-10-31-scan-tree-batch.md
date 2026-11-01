---
title: 批次讀取未合併 ref 的 tree
kind: refactor
source: docket/ideas/2026-10-31-scan-tree-batch.md, measured on a copy of the Purdex repository
---

## What

掃描逐層讀取每個未合併 ref 與預設分支 tip 的 tree：一次 `cat-file --batch` 讀根 tree，接著是 `openspec` 與 `docket`，再來是 `changes` 與 `records`，最後是封存；每個 commit 與 tree 只讀一次。物件遺失的 commit 或格式錯誤的 tree 改走舊的 `ls-tree`。

## Why

在 Purdex repository 的副本上，`dkt status` 啟動了 322 個 git 行程，其中 308 個是每個未合併 ref 一次的 `ls-tree`，耗時 21.9 秒；這些 ref 沒有一個含有 change。

## Verification

先用測試固定，且在舊程式碼上通過：非資料夾的項目、record 的 mode、共用同一個 commit 的 ref、每一層各缺一個物件、預設分支 tip、SHA-256 repository。對改動的行做突變都會讓測試失敗，除了三個等價突變。Purdex 副本：改動前 322 個 git 行程、22.1 秒，改動後 18 個、1.7 秒；`dkt status --json` 完全相同。

## Review

Codex (gpt-5.6-sol) 對整個 diff 攻擊一次：兩個 medium finding，原始 tree 項目的名稱含斜線，以及重複的名稱，ls-tree 對它們的讀法不同。含有這種 tree 的 commit 現在改走舊的 `ls-tree`，並附以字面 tree 建立的測試。
