---
title: 從封存搬移之後的失敗中復原
---

2026-11-02 記錄，來自兩次 codex 審查：`adopt-existing-projects` 的審查（一個 high 的 finding，因為 `main` 的順序相同而判定在範圍之外）以及較早的一次審查，在 spec 標題步驟中發現同樣的事。

## 問題

`dkt archive` 在 spec 標題步驟與 evidence 步驟之前先執行 `openspec archive`（`internal/archive/run.go:219-246`）。當其中任何一步失敗時，change 已經搬到 `openspec/changes/archive/`，而且 OpenSpec 可能已把它的 delta 合併進主 spec：

- spec 標題步驟依序寫入每個 capability 的 `docket.yaml`，並在第一次失敗時停止，所以較早的 capability 保有新標題。
- `evidence.md` 沒有寫入。
- 進行中的 change 資料夾已不存在，所以再執行一次 `dkt archive <slug>` 無法完成這次執行。

spec 只說執行會在失敗的步驟停止。沒有任何東西說明使用者能拿回什麼，所以這是 idea，不是 issue。

目前使用者可以手動復原：`dkt archive` 除非工作樹乾淨否則拒絕執行，而且不 commit 任何東西，所以用 git 還原工作樹就會回到執行前的狀態。

## 方向，尚未決定

- 在搬移之前，先檢查後面的步驟需要什麼：每個 delta 標題、每個目標 `docket.yaml`，以及 `evidence.md` 是否能寫入。
- 搬移後失敗時，還原這次執行改動的東西，就像 `dkt issue` 移除不完整的檔案。
- 讓第二次執行能完成沒有 `evidence.md` 的已封存 change。
- 在失敗訊息中說明如何還原工作樹，其他什麼都不改。

## 待釐清問題

- 既然執行前工作樹是乾淨的，用 git 手動還原是否足夠。
