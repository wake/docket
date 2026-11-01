---
title: 用開發資料檢視流程檔
---

2026-10-30 記錄於設計 `add-doctor` 時。維護者問，用真實開發資料來檢視流程檔（這些步驟划不划算、哪裡可以更輕或更快）應該放在哪裡。

## 問題

流程檔說明每條路徑做什麼，但沒有東西顯示它值不值得：每個 change 有多少審查、審出什麼、修了什麼、工作花多久。doctor 回答的是目前的設定對不對；這個要回答的是流程在許多件工作上是否划算，它的輸出是對流程檔的一份修改提議。

## 已決定

- 不屬於 doctor（`docket-doctor`）：輸入（跨 change 的歷史，而非目前狀態）、頻率與輸出都不同。
- 分成兩層，依「涉及 agent 的工作絕不放進 CLI」這條規則切：
  - 蒐集數字是機械的，歸 `dkt`；
  - 讀數字並提出修改是使用者叫用的指令，它會顯示修改後的整份流程檔，並且只在使用者同意時才寫入，和 setup 一樣。
- 只讀 repository：已封存的 change、它們的 `evidence.md`、record 與 git 歷史。token 用量、模型配額與 session 時間在 repository 之外，不納入。

## 依賴

- `2026-10-28-evidence-findings-and-sizes.md`：`evidence.md` 中每個 change 的大小與審查 finding。
- `2026-10-29-records.md`：record，這樣輕審的工作也會被計入。

## 待釐清問題

- 審查先讀哪些數字，涵蓋多少個 change。
- 由 `dkt status` 還是新指令印出跨 change 的數字。
