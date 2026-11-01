---
title: 有狀態的 idea，與 order 檔
needs:
  - docket/ideas/2026-10-29-records.md
---

2026-10-29 記錄。以下各點已與維護者定案；採用這個 idea 的 change 會把它們寫成 proposal 與 delta。

## 問題

idea 以前放在後續筆記裡，每份筆記有許多項目，而即將進行的工作的順序是某個 idea 裡的一份清單。沒有東西說明哪個 idea 變成哪個 change、哪些已排入佇列、哪些被放棄，所以狀態得靠人工去讀。

## 已決定

- 一個 idea 一個檔案：`docket/ideas/<date>-<slug>.md`。它的 front matter 只放人決定的東西：`title`；工作開始後的 `to: change:<slug>` 或
  `to: record:<slug>`；不會做的 idea 用 `dropped: true`；以及它所依賴的 idea 用 `needs:`。內文說明這個 idea 是什麼與為什麼。
- `docket/order.md` 放即將進行工作的順序，其餘什麼都不放：標題 `# Order`，然後一份從 1 開始、沒有缺號的編號清單，每行是一個核取方塊，加上指向 idea 的一個放在反引號裡的 repository 根路徑。沒有 idea 標題、版本、相依或備註。
- 第一個未勾選的行就是下一件工作。當它的 idea 所指向的 change 或 record 合併時，`dkt sync` 勾選該行，而 `dkt sync --check` 回報不同步的行。完成的行留著。
- `dkt lint` 檢查每行的格式、路徑存在於 `docket/ideas/` 之下、編號從 1 起沒有缺號、沒有重複，以及 order 裡沒有被放棄的 idea。
- 狀態是推導出來的，不是寫下來的：pool（不在 order 中、沒有 `to`）、queued（在 order 中、沒有 `to`）、in progress（它的 `to` 存在）、done（已合併或已封存）、dropped。出貨某個 idea 的版本來自 `git tag --contains`。

## 平行工作

2026-10-31 記錄，來自與維護者的討論。這是提議，不是定案；它會取代上面已決定的兩點，在採用這個 idea 時再定案。

已決定的兩點，各自讓每個 change 的 worktree 寫入所有 change 共用的檔案：

- `dkt sync` 勾選 order 的行。sync 在 change 的 worktree 中、PR 開啟之前執行，所以平行做的兩個 change 編輯 `docket/order.md` 的相鄰行，git 會回報衝突，而且一行在它的 PR 合併之前就被勾選了。
- `to:` 在工作開始時寫進那個 worktree。變成好幾件工作的 idea，如 records（`add-records`，然後是讀取 record 的 change），沒有單一的值可寫。

plan 兩個問題都沒有，因為它的勾選存在 change 自己的資料夾裡。提議把同樣的規則套到 idea：

- worktree 只寫自己那件工作的檔案：它的 change 資料夾或它的 record。`docket/order.md` 與 idea 檔只因維護者的決定而改變，從同一個地方寫入。
- `docket/order.md` 放順序。沒有東西勾選它；`dkt` 推導每行是否完成，依據是 `source` 指名該 idea 的 change 或 record 已合併或已封存。下一件工作是第一個未完成的行，由一個指令回報（見下面關於哪個指令回報 idea 狀態的待釐清問題）。AGENTS.md 的「第一個未勾選的行」隨之改變。
- `to:` 從 change 與 record 的 `source` 推導，不寫入。
- 一個 idea 就是一件工作。當 idea 有進一步的部分，那個部分就成為自己的 idea，其 `needs:` 指名第一個。

沒有 worktree 寫入它們，idea 與 order 就只有一個寫入者。作為從 2026-10-31 起的試行，這個 repository 中對 idea 的編輯直接推到預設分支，不開 PR。

## 待釐清問題

- 哪個指令回報 idea 的狀態。
- `needs:` 的確切形式。
- idea 如何分類，讓 order 讀起來像優先順序清單。
  2026-10-30 記錄。維護者發現佇列混雜了使用者的請求、開發中發現的修正與最佳化。提議的兩個軸，在採用這個 idea 時定案：
  - 來源：使用者的請求、開發中發現的（審查 finding、試驗、建置時碰到的 bug）、維護；
  - 性質：新 capability、修正、最佳化、文件。
  `domain:` 欄位與它們一起設計
  （`2026-10-30-spec-domains.md`）。`dkt` 依這些欄位列出與分組；order 本身仍是維護者的決定。

## 狀態

由 change `add-idea-state`（v0.15.0）完成，它採用了「平行工作」之下的提議，取代已決定的兩點：

- 沒有東西勾選 order，也沒有 `to:`。idea 的狀態來自 source 指名它的 record 與 change，和 issue 的做法一樣。
  `docket/order.md` 沒有核取方塊，`dkt sync` 也沒有 order 檢查。
- `dkt status` 回報狀態：`result.order`、`result.next` 與
  `result.ideas`。idea 與 order 從預設分支的 tip 讀取。
- source 在第一個欄位中指名路徑，所以在這個 change 之前寫的 source 仍然算數。
- `needs:` 是 idea 路徑的清單；status 顯示某行仍在等待的那些。
- 分類欄位被省略：issue 把修正、最佳化與測試從 idea 中拿走了。領域留在
  `2026-10-30-spec-domains.md`。
