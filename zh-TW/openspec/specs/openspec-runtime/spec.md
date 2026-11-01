# openspec-runtime Specification

## Purpose
定義 Docket 如何找到它委派的 OpenSpec CLI，以及接受哪些版本，讓 repository 不會被 Docket 未測試過的 OpenSpec 版本變更。

## Requirements

### Requirement: Locating the CLI
`DOCKET_OPENSPEC` 環境變數有設定且不為空時，Docket SHALL 執行它所指名的 OpenSpec 指令，把值依空白切成程式與其前置引數；否則 SHALL 執行 `PATH` 上的 `openspec`。

#### Scenario: Override with npx
- **WHEN** `DOCKET_OPENSPEC` 是 `npx -y @fission-ai/openspec@1.14.3`，而使用者執行 `dkt new`
- **THEN** Docket 執行 `npx`，引數為 `-y @fission-ai/openspec@1.14.3`，其後接 OpenSpec 的引數

#### Scenario: Default from PATH
- **WHEN** `DOCKET_OPENSPEC` 未設定，而使用者執行 `dkt new`
- **THEN** Docket 執行 `PATH` 上找到的 `openspec`

### Requirement: Supported versions
在指令第一次呼叫 OpenSpec 之前，Docket SHALL 讀取 CLI 的版本，除非該版本在這個 Docket 版本所支援的範圍內（第一個版本為 1.14.x），否則 SHALL 以 4 結束且不變更任何東西。錯誤 SHALL 指名找到的版本、可接受的範圍，以及以 `DOCKET_OPENSPEC` 指向另一個 CLI 的方式。

#### Scenario: Unsupported version
- **WHEN** OpenSpec CLI 回報版本 1.3.0，而使用者執行 `dkt new add-x --kind feature --source x --title X`
- **THEN** Docket 以 4 結束、不建立任何東西，並指名 1.3.0 與可接受的範圍

#### Scenario: CLI missing
- **WHEN** 無法啟動任何 OpenSpec CLI，而使用者執行 `dkt init`
- **THEN** Docket 以 4 結束，並說明如何安裝一個或設定 `DOCKET_OPENSPEC`

### Requirement: Failures reported
OpenSpec 的呼叫以非零狀態結束時，Docket SHALL 回報它執行的指令、結束狀態與 OpenSpec 的標準錯誤，且 SHALL NOT 執行任何依賴該次呼叫的後續步驟。

#### Scenario: OpenSpec command fails
- **WHEN** `dkt new` 期間 `openspec new change` 失敗
- **THEN** Docket 回報 OpenSpec 的結束狀態與錯誤輸出、不寫入 `docket.yaml`，並以 1 結束

### Requirement: Schema location
Docket SHALL 在 OpenSpec 1.14 尋找的地方尋找 `docket` schema：先是專案的 `openspec/schemas/docket/`，然後是每台機器的目錄，即 `$XDG_DATA_HOME` 下的 `openspec/schemas/docket/`，或 `XDG_DATA_HOME` 未設定或為空時 `<home>/.local/share/` 下的。家目錄 SHALL 以與流程檔相同的方式找出。`XDG_DATA_HOME` 是相對路徑或找不到家目錄時，會讀取或寫入每台機器目錄的指令 SHALL 以 4 結束並指名原因。

#### Scenario: Schema from the data directory
- **WHEN** `XDG_DATA_HOME` 是 `/d`，而使用者執行 `dkt init`
- **THEN** `/d/openspec/schemas/docket/schema.yaml` 與所附檔案相符

#### Scenario: Default under home
- **WHEN** `XDG_DATA_HOME` 未設定、家目錄是 `/h`，而使用者執行 `dkt init`
- **THEN** `/h/.local/share/openspec/schemas/docket/schema.yaml` 與所附檔案相符

#### Scenario: Project copy first
- **WHEN** 專案的 `openspec/schemas/docket/` 與每台機器的目錄都有該 schema，而使用者執行 `dkt new add-x --kind feature --source x --title X`
- **THEN** `result.schema.path` 是專案的 `openspec/schemas/docket`（絕對路徑）

#### Scenario: Relative data directory
- **WHEN** `XDG_DATA_HOME` 是 `data`，而使用者執行 `dkt init`
- **THEN** 指令以 4 結束並指名 `XDG_DATA_HOME`，且 repository 內外都沒有寫入任何東西
