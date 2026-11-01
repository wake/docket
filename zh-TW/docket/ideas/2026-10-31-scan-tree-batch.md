---
title: 批次讀取未合併 ref 的 tree
---

2026-10-31 記錄，來自在 Purdex repository 的副本上對 dkt v0.9.1 做的唯讀試驗。它接續 `2026-10-24-scan-git-calls.md`，該檔的 Status 留下了每個未合併 ref 一次 `ls-tree`。

## 量測

- 這個副本有 3,625 個 ref 與 13,608 個 commit。`dkt status` 花了 21.9 秒並啟動 322 個 git 行程，其中 308 個是 `ls-tree`，每個未合併 ref 一個；`dkt conflicts` 花的時間一樣。v0.9.1 之前，`dkt status` 五分鐘後還沒結束。
- 掃描讀 `refs/heads` 與 `refs/remotes`。其中 307 個 ref 未合併（146 個本地、161 個 remote-tracking），分布在 228 個不同的 commit，而且沒有一個含有 `openspec/changes` 或 `docket/records`：這 308 次呼叫什麼都沒找到。
- dkt 不 fetch：它讀的是上次 fetch 時的 remote-tracking ref。163 個未合併的 remote-tracking 分支中有 158 個在上游仍然存在，所以 `git fetch --prune` 沒有幫助。成本來自未合併的分支，不只是 squash merge。

## 線索

1. 用固定數量的 git 行程讀取每個未合併 ref 的 tree：每一層一個 `git cat-file --batch`（每個 ref 的根 tree；然後 `openspec` 與 `docket`；然後 `changes` 與 `records`；然後封存），不論有多少 ref 共用，每個 tree 只讀一次。沒有任何一個資料夾的 ref 就不再有額外成本。
2. 結果要完全保持不變：任何一層缺少的物件，對那個 ref 退回今天的 `ls-tree`，所以它得到今天的錯誤。
3. 單獨對 `<rev>:openspec/changes` 做 `cat-file --batch-check` 比較簡單，但它把不存在的路徑與缺少的物件回報成同一種情況，所以會改變損壞的 repository 所回報的內容。

## 相關

- `2026-10-24-scan-git-calls.md`：第一部分。
- `2026-10-31-existing-project-adoption.md`：同一次試驗。

## 狀態

由 record `docket/records/2026-10-31-scan-tree-batch.md` 完成（線索 1 與 2）：在 Purdex 副本上，`dkt status` 從 322 個 git 行程與 22.1 秒變成 18 個與 1.7 秒。不論有多少 ref，讀取 tree 最多需要四個 `cat-file` 行程；物件缺少或 tree 格式錯誤的 commit 仍然花一次 `ls-tree`。
