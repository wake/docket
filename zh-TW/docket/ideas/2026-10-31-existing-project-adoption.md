---
title: 在既有專案中採用 Docket
---

2026-10-31 記錄，來自在 Purdex repository 的副本上對 dkt v0.9.1（OpenSpec 1.14.3）做的唯讀試驗：這個專案有自己的規則、一個不進 git 的 OpenSpec 資料夾，以及一個用 OpenSpec 的 `spec-driven` schema 做的 change。以下沒有任何一項已決定。

## 發現

1. **git 忽略的已安裝檔案。** Purdex 的 `.gitignore` 有
   `.claude/`，它的 `.git/info/exclude` 有 `openspec/`。`dkt init` 寫入了 skill 與 schema 並回報 ok，`dkt inspect` 也是。
   這些檔案沒有一個能被 commit，所以新的 worktree 或 clone 裡沒有 Docket。
2. **用另一個 schema 做的 change。** 對於 `spec-driven` 的 change
   （`tasks.md`、沒有 `plan.md`、沒有 `docket.yaml`）：
   - `dkt lint` 依 Docket 的規則回報六個 finding：缺少標題、四個 proposal 節，以及 capability 的 `docket.yaml`。標題的 finding 說由舊版 dkt 做的 change 會搬移它的欄位，這並不適用，而且沒有任何東西說明這個 change 使用另一個 schema。
   - `dkt status` 把它顯示成 `design 0/0 tasks`，因為它只讀 `plan.md`。

   這個 schema 的已封存 change 不會產生任何 finding。
3. **與流程不一致的專案規則。** Purdex 的 AGENTS.md 寫了自己的流程（兩輪審查、合併後 bump 版本）。`dkt init` 附加 Docket 區塊時沒有察覺。
   `docket-doctor` 會找出這類衝突，但採用步驟沒有說要去跑它。
4. **git 之外的 OpenSpec 內容。** Purdex 的 `openspec/`（一份 `spec-driven` 設定與一個已封存的 change）從未 commit。要 commit、重寫還是丟棄，是專案的步驟，不是 `dkt` 的。

## 線索

- `dkt init` 與 `dkt inspect` 用 `git check-ignore` 檢查它們安裝的每個路徑，並回報被忽略的路徑，指出可以讓它進來的規則（對 `.claude/` 是：`.claude/*` 與 `!.claude/skills/`）。
- `dkt lint` 與 `dkt status` 讀 `.openspec.yaml` 中的 `schema`。另一個 schema 的 change 得到一個說明這件事的 finding，Docket 的規則跳過它；`dkt status` 指出它的 schema。
- README 的採用步驟在 `dkt init` 之後點名 `docket-doctor`，並說明如何處理已存在的 OpenSpec 內容。

## 相關

- `2026-10-31-global-schema.md`：隨附的 schema 離開專案，改為每台機器安裝一次，所以發現 1 之後只涉及 skill 與 Docket 區塊。這個 idea 建立在它之上。
- `2026-10-31-scan-tree-batch.md`：大型 repository 中掃描的速度，來自同一次試驗。
- `2026-10-31-codebase-decomposition.md`：描述已存在的程式碼。

## 待釐清問題

- `dkt status` 是否根本會列出另一個 schema 的 change，以及 `dkt archive` 是否拒絕它。
- `dkt init` 是自己寫入忽略的例外規則，還是只回報。

## 狀態

由 change `adopt-existing-projects` 完成。`dkt init` 與 `dkt inspect` 回報 `install-ignored` 並指出規則，絕不編輯忽略檔（發現 1）。另一個 schema 的 change 由其 `.openspec.yaml` 中的 `schema` 辨認：`dkt lint` 只檢查它新增的 capability 的標題，`dkt status` 顯示它的 schema 與 `tasks.md` 進度，`dkt archive` 不做 plan 與 base 檢查就封存它，`dkt sync` 與 `dkt rebase` 拒絕它，`dkt inspect` 回報它（發現 2）。`dkt inspect` 也回報沒有標題的主 spec。`docket-doctor` 在使用者同意下處理這三個 finding，並用 `convert.md` 轉換沒有勾選任何步驟的 change；`/docket-init` 在 `dkt init` 之後執行 doctor（發現 3）。README 的「在既有專案中採用 Docket」把不進 git 的 OpenSpec 內容留給專案處理（發現 4）。兩個待釐清問題都已解決：status 會列出這種 change，archive 接受它，而 `dkt init` 只回報忽略的例外規則。
