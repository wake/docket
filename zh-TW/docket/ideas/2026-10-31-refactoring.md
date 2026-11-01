---
title: Docket 中的重構
---

2026-10-31 記錄，當時正在實作 `add-records`。維護者問 Docket 如何支援重構，以及它屬於 Docket 還是 superpowers。Docket 是否該多做些什麼，等採用這個 idea 時再研究；以下沒有任何一項已決定。

## 現況

- 如何重整程式碼是 agent 的手藝。superpowers 沒有自己的重構 skill；它的測試驅動開發 skill 有 red、green、refactor 中的 refactor 步驟。
- Docket 涵蓋流程：
  - 分流把不改變行為的結構調整送到輕審。改變他人所依賴介面的調整改變了行為，走全審。
  - 流程範本的輕審來驗證：涵蓋該程式碼的測試在前後都通過，而對改動的行做突變會讓其中一個失敗。沒有任何一個失敗時，先補測試，作為只改測試的修改。
  - 流程範本的全審把結構改動與行為改動放在不同的 commit，並拆開把結構重構和行為混在一起的 PR，重構在前。輕審路徑對這兩件事都沒有說。
  - `add-records` 給這件工作一份類型為 `refactor` 的 record、大型工作可選用的 task，以及 `Record:` 與 `Task:` trailer。
  - 讓新邊界維持住的規則，例如 arch-go 的設定，屬於檢查設定：類型 `checks`。

## 線索

- 確定性的工具可以在不讓 agent 手動編輯的情況下改程式碼：`gopls` 的 rename、extract 與 inline；`gopatch`；Comby；Java 的 OpenRewrite；`rsc.io/rf`。prompt 可以在適合的地方偏好其中之一，並把指令寫進 record 的 Verification。
- `dkt` 不判斷內容就無法分辨結構 commit 與行為 commit，所以任何對它們的規則都留在 prompt 文字與審查裡。
- 依 `2026-10-31-codebase-decomposition.md` 的事實（哪些檔案一起改動）規劃的大型結構調整，是這兩個 idea 交會的地方。

## 相關

- `2026-10-31-codebase-decomposition.md`
- `2026-10-29-records.md`

## 待釐清問題

- 有 task 的重構是否需要比 record 更多的東西，例如讓每個 task 之後測試都維持綠燈的 task 順序。
- `docket` skill 或流程範本是否該提到重構工具，以及如何提而不點名某一種語言的工具。
- 輕審路徑是否也該讓結構 commit 與行為 commit 分開。
- 大到要把 capability 在領域之間移動的重構，如何分流。
