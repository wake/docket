# materialized-workflow Specification

## Purpose
定義 Docket 附帶的流程範本、使用者實體化的流程檔放在哪裡與其內容，
以及 `dkt workflow` 對它回報什麼，讓 agent 能在每個專案中，對每條路徑依循同一位使用者的步驟。

## Requirements

### Requirement: Workflow file location
使用者實體化的流程檔 SHALL 是檔案 `docket/workflow.md`，位於 `XDG_CONFIG_HOME` 為絕對路徑時該變數所指的目錄之下，否則位於使用者家目錄的 `.config` 之下。它 SHALL NOT 放在儲存庫中。

#### Scenario: Workflow in the config home
- **WHEN** `XDG_CONFIG_HOME` 未設定、家目錄是 `/h`，且使用者執行 `dkt workflow --json`
- **THEN** `result.path` 是 `/h/.config/docket/workflow.md`

#### Scenario: Workflow under XDG_CONFIG_HOME
- **WHEN** `XDG_CONFIG_HOME` 是 `/x`，且使用者執行 `dkt workflow --json`
- **THEN** `result.path` 是 `/x/docket/workflow.md`

#### Scenario: XDG config home ignored when relative
- **WHEN** `XDG_CONFIG_HOME` 是 `rel`、家目錄是 `/h`，且使用者執行 `dkt workflow --json`
- **THEN** `result.path` 是 `/h/.config/docket/workflow.md`

### Requirement: Shipped template
`dkt workflow template` SHALL 印出所執行的 Docket 版本附帶的流程範本。加上 `--json` 時，其 `result` SHALL 以 `text` 存放文字，並以 `template` 存放其雜湊：文字的 SHA-256 的前 12 個十六進位數字。

#### Scenario: Template printed
- **WHEN** 使用者執行 `dkt workflow template --json`
- **THEN** `result.text` 是附帶的範本，`result.template` 是其 SHA-256 的前 12 個十六進位數字，且指令以 0 結束

### Requirement: Template content
範本 SHALL 在四條路徑（全審、輕審、查核、直接合併）各自的標題之下，給出該路徑的審查與驗證步驟，只指名能力，絕不指名產品。必要的審查 SHALL 依序退而求其次：先是不同模型的審查者，再是全新 context 中的同一個模型，最後是在 PR 中註明該工作沒有獨立審查。

#### Scenario: Template names capabilities
- **WHEN** 印出範本
- **THEN** 它有標題 `## Full review`、`## Light review`、`## Check` 與 `## Direct merge`，且不指名任何產品或模型

### Requirement: Workflow status
`dkt workflow` SHALL 回報檔案的路徑與一個狀態：檔案不存在時為 `missing`；缺少 `## Full review`、`## Light review`、`## Check` 或 `## Direct merge` 其中一行時為 `incomplete`；其 frontmatter 沒有 `template`，或其值與附帶範本的雜湊不同時為 `stale`；其餘為 `ok`。`incomplete` SHALL 優先於 `stale`。只有 `ok` SHALL 以 0 結束；其他 SHALL 以 1 結束。

#### Scenario: No workflow
- **WHEN** 流程檔不存在，且使用者執行 `dkt workflow --json`
- **THEN** `result.status` 是 `missing`，findings 含 `workflow-missing`，且指令以 1 結束

#### Scenario: Complete workflow
- **WHEN** 檔案有這四個標題，且 `template` 等於附帶範本的雜湊
- **THEN** `result.status` 是 `ok`，沒有 finding，且指令以 0 結束

#### Scenario: A path missing
- **WHEN** 檔案的 `template` 等於較舊的雜湊，且缺少 `## Check`
- **THEN** `result.status` 是 `incomplete`，`result.missing_sections` 是 `["Check"]`，一個 `workflow-incomplete` finding 指出它，且指令以 1 結束

#### Scenario: Made from an older template
- **WHEN** 檔案有這四個標題與 `template: 0123456789ab`，而這不是附帶的雜湊
- **THEN** `result.status` 是 `stale`，`result.recorded` 是 `0123456789ab`，`result.template` 是附帶的雜湊，findings 含 `workflow-stale`，且指令以 1 結束

#### Scenario: No frontmatter
- **WHEN** 檔案有這四個標題，且沒有 frontmatter
- **THEN** `result.status` 是 `stale`，且 `result.recorded` 為 null

#### Scenario: Unreadable workflow
- **WHEN** 流程檔存在但無法讀取
- **THEN** 它被列在 `unreadable` 中，且指令以 3 結束
