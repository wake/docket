---
title: 每台機器安裝一次隨附的 schema
---

2026-10-31 記錄，來自在既有專案中採用 Docket 的討論。已決定：Docket 遵循 OpenSpec 的版面配置。隨附的 schema 不複製進專案；專案指名它，想要其他規則的專案則以新名稱 fork 它。同樣已決定：除非使用者執行相關指令或同意，否則沒有東西會寫入每台機器的副本，而那個指令是 `dkt init`；其他指令只回報它缺少或過期。fork 留待以後：`2026-10-31-schema-fork.md`。

## 現況

- `dkt init` 把 schema 複製進每個專案的
  `openspec/schemas/docket/`。`dkt` 從不讀它的內容；讀的是 OpenSpec，當 agent 執行 `openspec instructions` 或 `openspec status`，以及 `dkt new` 和 `dkt archive` 呼叫它時。`dkt lint` 檢查的規則在執行檔裡。
- `openspec init` 讓 `openspec/config.yaml` 維持 `schema: spec-driven`，`dkt init` 也不改它。每個 change 在它的 `.openspec.yaml` 中指名 `docket`，因為 `dkt new` 傳了 `--schema docket`。

## 為什麼

- 同一套規則被分成兩半：專案裡的 schema 說明 agent 寫什麼，執行檔裡的 lint 檢查它。升級 `dkt` 後，專案的副本維持舊的，直到有人執行 `dkt init --force`。只有 `dkt inspect` 會發現；`lint` 與 `status` 不會。
- `dkt inspect` 拿副本與自己的比較，分不出較舊的副本和較新的副本。在 `dkt` 較舊的機器上，執行 `dkt init --force` 的建議會覆寫較新的已提交副本。
- 每次 schema 改動都讓每個專案多一個 PR，當成 prompt 文字審查。v0.1.0 之後的十二個版本中，有三個改了 schema。

## OpenSpec 怎麼做

- 它內建的 `spec-driven` schema 留在它的套件裡；專案只在 `config.yaml` 中指名它。
- 它依序在專案的 `openspec/schemas/`、`$XDG_DATA_HOME/openspec/schemas/`（預設是
  `~/.local/share/openspec/schemas/`），然後在它的套件中找 schema。
- `openspec schema fork <source> [name]` 把 schema 以新名稱複製進專案，預設是 `<source>-custom`。
- 它的獨立執行檔在同一個資料目錄中保留內建 schema 的副本。

在沙箱中以 OpenSpec 1.14.3 試過：

- schema 只在使用者目錄時，`openspec schema which
  docket` 回報 `Source: user`，`status` 與 `instructions` 都能運作。沒有它則是：`Unknown schema 'docket'. Available: spec-driven`。
- `openspec schema fork docket` 從那裡讀取並寫出
  `openspec/schemas/docket-custom/`：名稱改了、兩行說明重新換行；instructions 與範本沒變。fork 不記錄它是從哪裡 fork 的。

## 提議的形狀

在 change 中定案：

1. `dkt` 擁有 `$XDG_DATA_HOME/openspec/schemas/docket/`，並讓它與內嵌的 schema 保持一致。
2. 只有 `dkt init` 寫入它（已決定）。缺少時，`dkt init` 寫入並印出路徑。不同時，`dkt init` 回報，而 `dkt init --force` 會連同專案的 skill 與 Docket 區塊一起覆寫它。`dkt new`、`dkt archive` 與 `dkt inspect` 在它缺少或不同時回報，並指名 `dkt init`。考慮過並放棄的：
   - 每當過期就由 `dkt new` 寫入：沒有使用者的動作或同意，任何東西都不得寫入；
   - 為每台機器的副本另設一個指令：一個指令讓採用只需一個步驟。

   沒有人原地編輯它。
3. `dkt init` 在 `openspec/config.yaml` 設定 `schema: docket`。這也標示該專案使用 Docket。
4. 專案的 `openspec/schemas/docket/` 優先於使用者目錄，所以目前為止做出的副本要移除：
   - 與某個隨附版本相同的，由 `dkt init` 移除，這需要每個隨附版本的雜湊；
   - 與任何版本都不同的，視為原地編輯過：保留並繼續優先，和現在一樣，`dkt inspect` 會回報；
   - Docket 自己的 repository 保留它的副本，那是來源。

接受的成本：

- 每台機器使用自己 `dkt` 的 schema；專案不釘版本。
- 對 Docket change 執行 `openspec` 的 CI，要先安裝 schema。只執行 `dkt lint` 的 CI 什麼都不需要，因為 lint 不呼叫 OpenSpec。

## 相關

- `2026-10-31-existing-project-adoption.md` 建立在這之上：它的 `init` 與 `inspect` 檢查，以及讀取 change 的 `schema` 的 lint。
- `2026-10-31-schema-fork.md`：專案自己的 schema fork，留待以後。

## 待釐清問題

- `config.yaml` 已經指名另一個 schema 時，`dkt init` 怎麼做，例如有 `spec-driven` change 的專案。

## 狀態

由 change `install-schema-per-machine` 完成：`dkt init` 每台機器安裝一次 schema，並移除專案中隨附 schema 的副本。待釐清問題在那裡解決了：缺少、沒有 `schema` 鍵或指名 `spec-driven` 的 `config.yaml` 會得到 `schema: docket`；其他名稱則保留並回報。fork 在 `2026-10-31-schema-fork.md`。
