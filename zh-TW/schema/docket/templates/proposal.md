# Proposal

## Why

<!-- 說明這個 change 的動機。這解決什麼問題？為什麼是現在？ -->

## What Changes

<!-- 描述將會改變什麼。具體寫出新 capability、修改或移除。 -->

## Non-Goals

<!-- 這個 change 刻意不做的事，每項一個項目。不可為空。 -->

## Builds On

<!-- 這個 change 所建立在、延伸或不可重複造的既有東西，以 path:symbol 或 capability 表示。先搜尋程式碼與 openspec/specs/，若專案的指示有指名架構筆記，也要讀。如果沒有相關的既有東西，就說明沒有，以及搜尋了什麼。 -->

## Capabilities

### New Capabilities
<!-- 新引入的 capability。以有內聚性的系統行為來命名每個 capability，
     讓它在系統演進時能擁有相關的需求。不要以實作 task 或 proposal 的段落命名。
     避免過於廣泛的萬用名稱。你新引入的路徑片段用
     kebab-case（例如 user-auth 或 identity/user-auth），
     並遵循專案既有的 spec 組織方式。每一個都會建立
     specs/<capability-path>/spec.md。 -->
- `<capability-path>`: <brief description of what this capability covers>

### Modified Capabilities
<!-- 需求（REQUIREMENTS）有變更的既有 capability（只是實作不算）。
     只在 spec 層級的行為改變時才列在這裡。每一個都需要一個 delta spec 檔案。
     使用 openspec/specs/ 底下確切的既有路徑。沒有需求變更時留空。
     完全沒有 capability 的 change（純重構、工具、文件）
     必須在它的 .openspec.yaml 設定 `skip_specs: true`，openspec validate 會拒絕
     沒有該標記的零 delta change。不要為了通過驗證而編造需求。 -->
- `<existing-capability-path>`: <what requirement is changing>

## Impact

<!-- 受影響的程式碼、API、依賴、系統 -->

## Implementation Contract

### Observable Behavior

<!-- 這個 change 完成後，使用者或呼叫端看得到什麼。 -->

### Interfaces

<!-- 這個 change 新增或改變的指令、旗標、API、事件、檔案格式、資料欄位。 -->

### Failure Modes

<!-- 出錯時會發生什麼：錯誤的輸入、缺少依賴、部分失敗。 -->

### Acceptance Criteria

<!-- 每個準則一個項目，各自點出它對應的 delta 情境，例如 `booking#join-when-full`。 -->

## Change Log

<!-- change 還是新的時候，只留表頭列。之後的每次修訂都加一列。 -->

| Date | Source | Change | Impact |
|---|---|---|---|
