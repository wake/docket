# change-creation Specification

## Purpose
定義 `dkt new` 如何開啟 change：一個從未用過的有效名稱、`docket` schema，
以及一份記錄：這是哪種 change、請求從哪裡來、從哪個 commit 開始。

## Requirements

### Requirement: Slug rules
`dkt new` SHALL 只接受符合以下條件的 slug：小寫 kebab-case（字母 a-z 與數字，以單一連字號連接成組）、最長 200 個字元、不以 `YYYY-MM-DD` 形式的日期開頭，且不是 `archive`（OpenSpec 封存資料夾的名稱）。

#### Scenario: Date prefix rejected
- **WHEN** 使用者執行 `dkt new 2026-10-24-add-x --kind feature --source x --title X`
- **THEN** 指令以 1 結束，並說明封存時會自己加上日期

#### Scenario: Uppercase rejected
- **WHEN** 使用者執行 `dkt new Add-X --kind feature --source x --title X`
- **THEN** 指令以 1 結束

#### Scenario: Reserved name rejected
- **WHEN** 儲存庫有已封存的 change，且使用者執行 `dkt new archive --kind feature --source x --title X`
- **THEN** 指令以 1 結束，且 `openspec/changes/archive/` 不變

### Requirement: Slug never reused
`dkt new` SHALL NOT 在任何掃描到的 source 已使用的 slug 下建立 change，不論它是進行中或已封存。slug 已被使用時，它 SHALL 在 `<slug>-2`、`<slug>-3`、… 中第一個沒有被任何掃描到的 source 使用的 slug 下建立 change，並 SHALL 回報該 slug 以及給定的 slug 被使用的每個地方。任何 source 無法讀取時，它 SHALL 不建立任何東西並以 3 結束，因為它無法證明該 slug 未被使用。同時在同一個 worktree 建立同一個 slug 的兩次執行中，一次 SHALL 建立該 slug，另一次 SHALL 取下一個後綴，且不動到第一次建立的東西。

#### Scenario: Slug active in another worktree
- **WHEN** worktree B 持有 `add-x`，且使用者在 worktree A 執行 `dkt new add-x --kind feature --source x --title X`
- **THEN** 指令建立 `openspec/changes/add-x-2/`，`result.slug` 是 `add-x-2`，輸出指出 worktree B，並以 0 結束

#### Scenario: Slug archived
- **WHEN** 預設分支的 tip 持有 `openspec/changes/archive/2025-01-02-add-x/`，且使用者執行 `dkt new add-x --kind feature --source x --title X`
- **THEN** 指令建立 `openspec/changes/add-x-2/`，輸出指出已封存的資料夾，並以 0 結束

#### Scenario: Unreadable worktree blocks creation
- **WHEN** 列出的某個 worktree 無法讀取，且使用者執行 `dkt new add-x --kind feature --source x --title X`
- **THEN** 不建立任何東西，指令以 3 結束

#### Scenario: Concurrent runs keep the first
- **WHEN** 兩次 `dkt new add-x --kind feature --source x --title X` 同時在同一個 worktree 開始
- **THEN** 一次建立 `openspec/changes/add-x/`，另一次建立 `openspec/changes/add-x-2/`，兩者都以 0 結束，且 `add-x` 保留第一次執行寫入的所有東西

### Requirement: Kind and source
`dkt new` SHALL 要求 `--kind`，其值為 `feature`、`change-request` 或 `baseline` 之一，並 SHALL 要求非空的 `--source`，說明請求從哪裡來。恢復已知答案的修正是 record，不是 change。

#### Scenario: Missing source
- **WHEN** 使用者執行 `dkt new add-x --kind feature`
- **THEN** 指令以 2 結束

#### Scenario: Unknown kind
- **WHEN** 使用者執行 `dkt new add-x --kind fix --source x`
- **THEN** 指令以 2 結束並列出三種 kind

### Requirement: Change created with metadata
`dkt new` SHALL 執行 `openspec new change <slug> --schema docket` 來建立 change，再寫入 change 的 `docket.yaml`，含 title、kind、source 與 base，並讓 `.openspec.yaml` 維持 OpenSpec 寫入的樣子。base SHALL 是 HEAD 與預設分支 tip 的 merge base 的完整 SHA。change 資料夾的路徑經過 `openspec/` 底下的符號連結時，`dkt new` SHALL 既不執行 OpenSpec 也不寫入任何東西，並 SHALL 以 1 結束且指出該連結。

#### Scenario: Change created
- **WHEN** 使用者在從預設分支建出的分支上執行 `dkt new add-x --kind feature --source "issue 12" --title "Booking waitlist"`
- **THEN** `openspec/changes/add-x/docket.yaml` 含 `title: Booking waitlist`、`kind: feature`、`source: issue 12`，以及等於該 merge base 的 `base`，且 `.openspec.yaml` 只含 OpenSpec 寫入的內容

#### Scenario: Schema not installed
- **WHEN** 專案的 `openspec/schemas/docket/` 與每台機器的目錄都沒有該 schema，且使用者執行 `dkt new add-x --kind feature --source x --title X`
- **THEN** 不執行 OpenSpec、不建立任何東西，findings 含指名 `dkt init` 的 `schema-missing`，且指令以 1 結束

#### Scenario: Symlinked changes folder refused
- **WHEN** `openspec/changes` 是指向儲存庫外某目錄的符號連結，且使用者執行 `dkt new add-x --kind feature --source x --title X`
- **THEN** 不執行 OpenSpec、連結兩側都不寫入任何東西，且指令以 1 結束並指出該連結

### Requirement: Title
`dkt new` SHALL 要求 `--title`，其 title 須符合 `docket-metadata` 所定義的形式，缺少或不符該形式時 SHALL 不建立任何東西。

#### Scenario: Missing title
- **WHEN** 使用者執行 `dkt new add-x --kind feature --source x`
- **THEN** 指令以 2 結束且不建立任何東西

#### Scenario: Blank title
- **WHEN** 使用者執行 `dkt new add-x --kind feature --source x --title "   "`
- **THEN** 指令以 2 結束且不建立任何東西

### Requirement: Schema found before creating
執行 OpenSpec 之前，`dkt new` SHALL 依 `openspec-runtime` 所述尋找 `docket` schema。兩處都沒有時，`dkt new` SHALL 不建立任何東西、回報 `schema-missing` 告訴使用者執行 `dkt init`，並以 1 結束。否則它 SHALL 在 `result.schema` 中回報找到的目錄（絕對路徑），以及它是否與附帶的 schema 逐位元組相同；不同的副本 SHALL 被回報，且 SHALL NOT 使 change 無法建立。

#### Scenario: Schema from the machine
- **WHEN** 只有每台機器的目錄持有 schema，且與附帶的相同，使用者執行 `dkt new add-x --kind feature --source x --title X`
- **THEN** 建立 change，`result.schema.path` 是每台機器的目錄，`result.schema.matches` 為 true，且指令以 0 結束

#### Scenario: Schema differs
- **WHEN** 每台機器的 `schema.yaml` 與附帶的檔案不同，且使用者執行 `dkt new add-x --kind feature --source x --title X`
- **THEN** 建立 change，`result.schema.matches` 為 false，輸出說明副本不同且 `dkt init --force` 會覆寫它，並以 0 結束
