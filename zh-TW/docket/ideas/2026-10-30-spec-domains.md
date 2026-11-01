---
title: spec 領域
---

2026-10-30 記錄，當時 `add-doctor` 正在審查。維護者問 Docket 是否幫助專案依領域切分 spec（例如一般產品中的帳號、文章與商品），因為 OpenSpec 接受巢狀的 spec 資料夾。

## 問題

OpenSpec 讀取相對於 `openspec/specs/` 的 capability 路徑，路徑可以有多段（`account/login`）。Docket 也讀這種路徑（`delta.CapabilityPath`），但：

- 沒有任何東西提議領域的版面配置。隨附的 schema 保留了 OpenSpec 的指示：遵循專案現有的組織方式，不要替扁平的版面加上領域層級，所以一開始扁平的專案，不論長出多少區塊都維持扁平。
- 只有部分指令用巢狀路徑測過：`delta`、`dkt status` 與封存的標題檢查。lint 的情境引用、`dkt conflicts`、封存的 base 檢查，以及與 OpenSpec 的交叉檢查都沒有。
- 領域資料夾對 Docket 沒有意義：它沒有標題，`dkt status` 也無法依它分組或篩選。

## 已決定

- 維護者希望產品有不同區塊時，spec 依領域切分。Docket 自己的 spec 維持扁平；它不需要切分。
- 是否引入領域，以及某個 capability 屬於哪個領域，是判斷。這放在 prompt 文字裡（schema 的 proposal 與 specs 指示、`docket` skill），絕不做成 `dkt` 的旗標，依「涉及 agent 的工作絕不放進 CLI」這條規則。況且 `dkt new` 目前還不知道 change 的 capability。
- 機械的部分歸 `dkt`：巢狀路徑在每個指令中都能運作、領域資料夾的標題、報告中的分組，以及檢查新 capability 是否遵循專案版面配置的 lint。

## 相關

- `2026-10-29-ideas-and-order.md`：idea 的 `domain:` 欄位，與該檔所記的分類欄位一起設計。

## 待釐清問題

- OpenSpec 1.14.3 在 `list --specs`、`validate`、`show` 與 `archive` 中如何處理巢狀 spec 資料夾。設計前先在沙箱驗證。
- prompt 何時應提議第一層領域，以及專案把版面配置記在哪裡：現有資料夾，或宣告的清單。
- 領域資料夾是否像 capability 一樣有帶標題的 `docket.yaml`。
- `dkt status` 與 `dkt conflicts` 如何依領域分組。
