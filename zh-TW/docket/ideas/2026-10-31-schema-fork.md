---
title: 讓專案 fork 隨附的 schema
---

2026-10-31 記錄，從 `2026-10-31-global-schema.md` 拆出：那個 change 讓隨附的 schema 每台機器安裝一次，而使用者把 fork 留待以後。

## 已決定

- `dkt lint` 讀取的標題是固定的英文關鍵字，就像 OpenSpec 的 delta 標記（`## ADDED Requirements`、`### Requirement:`、
  `#### Scenario:`）。它們底下的文字可以是任何語言。fork 可以改 schema 的其餘部分，但不能改那些標題。
- 考慮過並放棄其他語言的標題：像 Gherkin 的 `# language:` 標頭那樣，為每種語言設一張別名表，或用隱藏標記取代標題。別名表之後仍可加入，而不必改變檔案格式。

## 已知的事

- `openspec schema fork <source> [name]` 把 schema 以新名稱複製進專案的 `openspec/schemas/`，預設是 `<source>-custom`，並且不記錄它是從哪裡 fork 的。
- 專案會在 `openspec/config.yaml` 中指名它的 fork，而 `dkt new` 會使用 `config.yaml` 指名的 schema，而不是傳 `--schema docket`。
- 專案裡與任何隨附版本都不同的 `docket` 副本，是原地編輯過的；`docket-doctor` 可以幫忙把它變成 fork。

## 相關

- `2026-10-31-global-schema.md`：每台機器安裝一次的隨附 schema。

## 待釐清問題

- `dkt` 如何判斷某個 fork 是 Docket 的，因為 fork 不說明它從哪裡來。
- 專案如何得知隨附的 schema 在它 fork 之後改了。
- 已在執行檔裡的 record 與流程範本，是否也讓專案覆寫。
