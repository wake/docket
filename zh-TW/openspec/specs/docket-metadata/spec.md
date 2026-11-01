# docket-metadata Specification

## Purpose
定義 Docket 把它加到 OpenSpec 的 change 與 spec 上的資料放在哪裡、
如何讀取在這之前建立的 change，以及什麼是 title。

## Requirements

### Requirement: Change metadata file
Docket SHALL 從 change 資料夾中的 `docket.yaml` 讀取 change 的 title、kind、source 與 base：這是一個 YAML mapping，鍵為 `title`、`kind`、`source` 與 `base`。建立日期 SHALL 維持在 `.openspec.yaml`，如 OpenSpec 所寫。

#### Scenario: Metadata read from docket.yaml
- **WHEN** `openspec/changes/add-x/docket.yaml` 含 `title: Booking waitlist`、`kind: feature`、`source: issue 12` 與一個 base SHA
- **THEN** Docket 為 `add-x` 讀取該 title、kind、source 與 base

### Requirement: Changes made before docket.yaml
對沒有 `docket.yaml` 的 change 資料夾，Docket SHALL 從 `.openspec.yaml` 的 `x-kind`、`x-source` 與 `x-base` 鍵讀取 kind、source 與 base，且該 change SHALL 沒有 title。`docket.yaml` 存在時，Docket SHALL 忽略那些鍵。Docket SHALL NOT 為了搬移而改寫已封存 change 的檔案。

#### Scenario: Change made before docket.yaml
- **WHEN** `add-x` 沒有 `docket.yaml`，且其 `.openspec.yaml` 含 `x-kind`、`x-source` 與 `x-base`
- **THEN** Docket 從那些鍵讀取它的 kind、source 與 base，且不給它 title

#### Scenario: Both files present
- **WHEN** `add-x` 有一個含某個 base 的 `docket.yaml`，以及一個 `x-base` 為另一個值的 `.openspec.yaml`
- **THEN** Docket 使用 `docket.yaml` 中的 base

### Requirement: Spec titles
主 spec 的 title SHALL 是 `openspec/specs/<capability>/docket.yaml` 中的 `title`。change SHALL 在 `openspec/changes/<slug>/specs/<capability>/docket.yaml`（與該 capability 的 delta 並列）中為 capability 提供其 title。Docket SHALL NOT 從標題取得 title。

#### Scenario: Title beside a main spec
- **WHEN** `openspec/specs/booking/docket.yaml` 含 `title: Booking`
- **THEN** `booking` spec 的 title 是 "Booking"，不論其 `spec.md` 的 H1 寫什麼

### Requirement: Title form
title SHALL 是一行非空、且不只有空白的文字。

#### Scenario: Title on two lines
- **WHEN** 某個 `docket.yaml` 的 `title` 中含有換行
- **THEN** Docket 將該 title 視為缺少

### Requirement: Change schema
Docket SHALL 從 change 的 `.openspec.yaml` 的 `schema` 鍵讀取其 schema，如 OpenSpec 所寫。`.openspec.yaml` 沒有 `schema` 鍵的 change SHALL 的 schema 為 `docket`。schema 不是 `docket` 的 change 是其他 schema 的 change；`dkt` SHALL NOT 寫入它的 `.openspec.yaml`。

#### Scenario: Schema read
- **WHEN** `openspec/changes/add-x/.openspec.yaml` 含 `schema: spec-driven`
- **THEN** Docket 將 `add-x` 的 schema 讀為 `spec-driven`，即其他 schema 的 change

#### Scenario: Schema missing
- **WHEN** `openspec/changes/add-x/.openspec.yaml` 只含 `created: 2026-01-02`
- **THEN** Docket 將 `add-x` 的 schema 讀為 `docket`

### Requirement: Path a source names
change 或 record 的 `source` 所指名的路徑 SHALL 是：去掉前後空白的 source，取到第一個空白為止，並去掉結尾的一個逗號。依路徑尋找 issue 或 idea 的規則 SHALL 使用這個路徑，因此其後可以接一段備註。

#### Scenario: Path alone
- **WHEN** 預設分支的 tip 持有 idea `docket/ideas/2026-11-02-x.md` 與 record `2026-11-02-fix-x`，後者的 source 是 `docket/ideas/2026-11-02-x.md`
- **THEN** 該 record 是該 idea 的 handler

#### Scenario: Path with a note
- **WHEN** 預設分支的 tip 持有 idea `docket/ideas/2026-11-02-x.md`、一個 source 為 `docket/ideas/2026-11-02-x.md, item 1 of docket/order.md` 的 record，以及一個 source 為 `docket/ideas/2026-11-02-x.md (order 5)` 的已封存 change
- **THEN** 兩者都是該 idea 的 handler
