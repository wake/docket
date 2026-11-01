---
title: 掃描時每個 ref 少啟動幾個 git 行程
---

2026-10-24 審查 `add-init-and-new` 時發現。在動到同一段程式碼的 change 裡一併處理，
或在不改任何 spec 句子時直接修。

掃描對每個 ref 啟動數個 git 行程（`merge-base --is-ancestor` 與兩次 `ls-tree`）；
`git for-each-ref --no-merged` 可以一次呼叫就排除已合併的 ref。`Toplevel` 會執行 git 兩次。

## 2026-10-31 實測

由建置 workbench 的 session 回報，在 `main` 的 68294e2 上：

- 在這個 repository（22 個 ref：4 個本地、18 個 remote-tracking；4 個 worktree），
  一次 `dkt status` 花 2.7 秒，啟動 44 個 git 行程。在那台機器上，
  啟動一個 git 行程約花 30 到 45 毫秒。
  - `merge-base --is-ancestor`：24 次呼叫，1.09 秒，約 40%。
  - `ls-tree -d`：6 次呼叫，0.27 秒。
  - `rev-parse`、`log`、`symbolic-ref`、`status` 等：14 次呼叫，約 0.6 秒。
- 在估計有數千個 ref 的 repository 裡，`dkt status` 兩分鐘後仍未跑完。
  以 squash 合併的 repository，舊分支永遠不會被視為已合併，所以每個 ref 還要多付 `ls-tree` 的呼叫。
- 自 v0.9.0 起，掃描還會列出每個未合併 ref 中的 `docket/records`，
  每個 ref 多一次 `ls-tree`。
- `status.Load` 對分支上 change 的每個檔案啟動一次 `git show`。

## 線索，依預期效果排序

1. 用一次 `git for-each-ref --no-merged=<default tip> refs/heads refs/remotes`，
   取代每個 ref 一次 `merge-base --is-ancestor`。
2. 每個 ref 對 `openspec/changes`、其 archive 與 `docket/records` 只列一次，
   或對所有未合併的 ref 用一次 `git cat-file --batch`。
3. 在 `status.Load` 批次讀取（`cat-file --batch`，或一次讀整個 change 資料夾）。
4. 對仍然每個 ref 一次的呼叫，加上有界的平行度。

依 ref SHA 快取掃描結果，屬於常駐的 workbench，不屬於 `dkt`。
結果必須維持不變，包括單一無法讀取的 ref 要單獨被回報為無法讀取；
承諾某個速度會是新的期待，也是 spec 變更。

## 狀態

已由 record `docket/records/2026-10-31-scan-git-calls.md` 完成（線索 1 到 3）：
這裡的 `dkt status` 從 46 個 git 行程與 3.3 秒，降到 21 個與 1.6 秒，
而且處理 ref 的成本不再隨 ref 數量增加。留待之後：
每個未合併的 ref 仍要一次 `ls-tree`，這在以 squash 合併的 repository 裡很要緊
（它的舊分支會一直維持未合併）；每個 worktree 仍要一次 `merge-base`；
`Toplevel` 仍執行 git 兩次；以及線索 4，有界平行度。
每個未合併 ref 一次的 `ls-tree` 已量測，並在 `2026-10-31-scan-tree-batch.md` 接手處理。
