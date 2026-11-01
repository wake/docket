# cli-output Specification

## Purpose
定義每個 `dkt` 指令印出與回傳什麼，讓腳本、CI 工作與 agent 能依賴任何指令相同的輸出形狀與結束碼。

## Requirements

### Requirement: JSON output
每個指令 SHALL 接受 `--json`。有 `--json` 時，指令 SHALL 在標準輸出只寫出恰好一個 JSON 物件，不寫其他東西，成功與失敗皆然，並 SHALL 把診斷訊息寫到標準錯誤。沒有 `--json` 時，輸出 SHALL 是人類可讀的文字。

#### Scenario: JSON on success
- **WHEN** 使用者在 `dkt init --json` 會成功的 repository 中執行它
- **THEN** 標準輸出可解析為單一 JSON 物件，其 `ok` 為 true

#### Scenario: JSON on failure
- **WHEN** 使用者執行 `dkt new Bad_Slug --kind feature --source x --title X --json`
- **THEN** 標準輸出可解析為單一 JSON 物件，其 `ok` 為 false，且其 finding 指名該無效的 slug

### Requirement: Result envelope
每個 JSON 結果 SHALL 含有欄位 `tool`（永遠是 "dkt"）、`version`、`command`、`repo`、`ok`、`findings`、`unreadable` 與 `result`，使用錯誤與環境錯誤的結果也一樣。無法辨識 repository 時，`repo` SHALL 為 null。每個 finding SHALL 帶有 `rule` 與 `message`，並在適用時帶有 `change`、`file` 與 `line`。

#### Scenario: Envelope fields present
- **WHEN** 任何指令帶 `--json` 執行
- **THEN** 該物件有全部八個欄位，且 `findings` 與 `unreadable` 是陣列，沒有東西可回報時為空

### Requirement: Repository identity
每個結果 SHALL 辨明它描述的對象：repository 根目錄路徑、HEAD 的完整 SHA、目前的分支名稱（HEAD 為 detached 時為 null），以及 `origin` remote 的 URL（沒有時為 null）。

#### Scenario: Detached HEAD
- **WHEN** 指令在 HEAD 為 detached 的狀態下執行
- **THEN** `repo.branch` 為 null，且 `repo.head` 是已 checkout 的 commit 的完整 SHA

#### Scenario: No origin remote
- **WHEN** repository 沒有 `origin` remote
- **THEN** `repo.origin` 為 null，指令其餘行為照常

### Requirement: Exit codes
每個指令 SHALL 在成功且沒有東西可回報時以 0 結束；在執行了並發現問題、或拒絕了請求時以 1 結束；使用錯誤時以 2 結束；無法讀取它所需的 source 時以 3 結束；環境無法使用時以 4 結束，即沒有 git repository、缺少 `git`，或需要 OpenSpec CLI 的指令沒有可用的 OpenSpec CLI。

#### Scenario: Usage error
- **WHEN** 使用者執行 `dkt new` 而沒有給 slug
- **THEN** 指令以 2 結束並印出其用法；帶 `--json` 時則印出一個 JSON 物件，其 finding 的 rule 為 `usage`

#### Scenario: Not a git repository
- **WHEN** 使用者在 git repository 之外帶 `--json` 執行任何指令
- **THEN** 指令以 4 結束並印出一個 JSON 物件，其 `repo` 為 null，且其 finding 的 rule 為 `environment`

### Requirement: Unreadable is not empty
指令無法讀取它所需的 source 時，SHALL 把該 source 連同原因列在 `unreadable` 中，SHALL NOT 把該 source 回報為沒有任何 change，且即使在別處也發現了問題，仍 SHALL 以 3 結束。

#### Scenario: Unreadable source with other findings
- **WHEN** 指令在一個 worktree 中發現問題，而無法讀取另一個
- **THEN** 它兩者都回報，並以 3 結束
