---
title: 出貨的 schema 不再指向只有 Docket 才有的檔案
kind: prompt
source: docket/ideas/2026-10-31-existing-project-adoption.md, the read-only trial on a copy of the Purdex repository
---

## What

出貨 schema 的 Builds On 指示與 proposal 範本的註解，不再把 agent 導向 `docs/ARCHITECTURE.md` 與 capability 索引：改為搜尋程式碼與 `openspec/specs/`，並在專案的指示有指名架構筆記時閱讀它們。Reuse 表的 Evidence 範例移除 `dkt index`。

## Why

在 Purdex repository 副本上的試用中，撰寫 baseline 的 agent 去找 `docs/ARCHITECTURE.md` 與 `dkt index`。只有 Docket 自己的 repository 有前者，後者仍在規劃中；它在 proposal 中回報兩者都不存在。

## Verification

在由 v0.9.1 設定的拋棄式 repository 中，分支的建置回報兩個檔案不同，而 `dkt init --force` 會更新它們。OpenSpec 1.14.3 印出新的 proposal 與 plan 指示和範本；出貨的 schema 檔案沒有一個提到 `ARCHITECTURE` 或 `dkt index`。`go vet` 與 `go test ./...` 通過。

## Review

Codex (gpt-5.6-sol) 對整個 diff 攻擊一次，無 finding 而核准：schema 與範本一致，沒有其他出貨的 prompt 指向只有 Docket 的 repository 才有的東西，而 Docket 自己的 AGENTS.md 仍指名 `docs/ARCHITECTURE.md`。
