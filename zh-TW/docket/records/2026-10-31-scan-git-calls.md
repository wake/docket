---
title: 掃描時啟動更少的 git 行程
kind: refactor
source: docket/ideas/2026-10-24-scan-git-calls.md, with measurements from the workbench session
---

## What

ref 改用三次 `for-each-ref` 列表加一次 `rev-list` 來判定，不再每個 ref 各跑一次 `merge-base`；每個 tree source 用一次 `ls-tree` 讀取，不再三次；分支的 change 檔案用一次 `cat-file --batch` 讀取。批次對受損歷史或奇怪的項目回報不同時，改跑舊的呼叫。

## Why

`dkt status` 在這裡啟動了 46 個 git 行程（3.3 秒），每個 ref 一個或更多；在有數千個 ref 的 repository 中，兩分鐘內跑不完。

## Verification

先用測試固定每個結果（已合併與未合併的 ref、受損歷史、tree 項目、檔案讀取）；改動前後都全數通過。對改動的行做突變都會讓測試失敗，除了四個等價突變，以及兩個只有故障注入才碰得到的 fallback。這裡的 `dkt status`：改動前 46 個 git 行程、3.3 秒，改動後 21 個、1.6 秒。

## Review

Codex (gpt-5.6-sol) 對整個 diff 攻擊一次：一個 medium finding，`ShowAll` 在沒有送出路徑時不檢查它的 revision。它的文件現已說明這點，並附測試。實作者在受損歷史上找到兩處批次的差異；fallback 保留現有的結果。

## Tasks

### Task 1: Settle refs without a merge-base each

- [x] 在沒有測試涵蓋的地方，用測試固定目前的行為：已合併的 ref 被跳過，未合併的被讀取，tip 或歷史無法讀取的 ref 單獨算作無法讀取。
- [x] 用 `for-each-ref --merged` 與 `--no-merged` 列出已合併與未合併的 ref，用 `rev-list` 走一次未合併的歷史，列表或走訪無法判定的 ref 則照舊逐一詢問。

### Task 2: One ls-tree per tree source

- [x] 在沒有測試涵蓋的地方，用測試固定 `readTree` 目前的行為：進行中與已封存的 change 資料夾、record、巢狀與非 record 的檔案、不是資料夾的 `docket/records`。
- [x] 用一次 `ls-tree` 讀取三個資料夾，失敗時改用原本的三次呼叫。

### Task 3: One cat-file for a branch's change files

- [x] 在沒有測試涵蓋的地方，用測試固定 `scan.ReadFiles` 在分支上目前的行為。
- [x] 用一次 `cat-file --batch` 讀取保留的檔案，它沒有回傳的檔案則用 `git show`。
