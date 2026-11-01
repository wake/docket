# change-lint Specification

## Purpose
定義 `dkt lint` 對 change 資料夾強制執行的結構規則，讓串接 superpowers 與 OpenSpec 的規則在被破壞時明確失敗，而不是只存在於 prompt 裡。

## Requirements

### Requirement: What is linted
`dkt lint` SHALL 檢查命令列上指名的 change 與 record；未指名任何東西時，則檢查目前 worktree 的 `openspec/changes/` 中每個進行中的 change、分支新增的 `docket/records/` 下每個 record（自與預設分支的 merge base 之後已 commit 的，或尚未 commit 的），以及 `openspec/specs/` 下主 spec 的 title。以日期開頭的名稱指的是 record。它 SHALL 照原樣讀取 worktree 的檔案（含未 commit 的修改），且 SHALL NOT 需要 OpenSpec CLI。

#### Scenario: Lint every active change
- **WHEN** worktree 有進行中的 change `add-x` 與 `add-y`，而使用者執行 `dkt lint`
- **THEN** 兩個 change 都被檢查，封存資料夾則不檢查

#### Scenario: Named change missing
- **WHEN** 使用者執行 `dkt lint add-z`，而 worktree 沒有 change `add-z`
- **THEN** 指令回報 `add-z` 不存在，並以 1 結束

#### Scenario: No OpenSpec installed
- **WHEN** 沒有安裝 OpenSpec CLI，而使用者執行 `dkt lint`
- **THEN** 指令照常檢查

#### Scenario: Lint records the branch adds
- **WHEN** 預設分支含有 record `2026-10-01-fix-a`，分支新增了 record `2026-10-30-fix-b`，而使用者執行 `dkt lint`
- **THEN** `2026-10-30-fix-b` 被檢查，`2026-10-01-fix-a` 不檢查

### Requirement: Proposal sections
proposal.md SHALL 有 Why、What Changes、Non-Goals、Builds On、Capabilities、Impact、Implementation Contract 與 Change Log 這些段落，且 Implementation Contract SHALL 有 Observable Behavior、Interfaces、Failure Modes 與 Acceptance Criteria 這些子段落。Non-Goals、Builds On 與這四個子段落 SHALL 各自含有空白與 HTML 註解以外的文字。

#### Scenario: Non-Goals missing
- **WHEN** proposal.md 沒有 Non-Goals 段落
- **THEN** lint 回報指名 Non-Goals 的 `proposal-section` finding

#### Scenario: Empty Failure Modes
- **WHEN** Failure Modes 子段落只有範本的 HTML 註解
- **THEN** lint 回報指名 Failure Modes 的 `proposal-section` finding

#### Scenario: Builds On left empty
- **WHEN** Builds On 段落沒有文字
- **THEN** lint 回報指名 Builds On 的 `proposal-section` finding

### Requirement: Change Log rows
Change Log 表格在表頭之後的每一列 SHALL 有 YYYY-MM-DD 格式的日期與非空的 Source。只有表頭的 Change Log 表格 SHALL 通過。

#### Scenario: Change Log row without source
- **WHEN** Change Log 的某一列有日期，但 Source 儲存格是空的
- **THEN** lint 在該列所在行回報 `change-log` finding

#### Scenario: Header-only Change Log
- **WHEN** Change Log 表格只有表頭，沒有任何列
- **THEN** lint 不回報任何有關 Change Log 的事

### Requirement: Leftover template text
lint SHALL 在 change 的每個檔案中，回報逐字出現在所附 `docket` schema 範本中的任何 HTML 註解或角括號佔位符，程式碼片段（code span）與圍欄程式碼區塊（fenced code block）內除外。

#### Scenario: Template comment left
- **WHEN** proposal.md 的 Why 底下仍留著範本註解
- **THEN** lint 在該行回報 `placeholder` finding

### Requirement: Plan rules only with a plan
plan.md 與情境宣告的規則 SHALL 只在 change 有 plan.md 時適用。沒有 plan.md 的 change SHALL 只就其 proposal 與 delta 檢查。

#### Scenario: Plan not written yet
- **WHEN** change 有 proposal.md 與 delta，但沒有 plan.md
- **THEN** lint 不對它回報 plan 或 `scenario-unclaimed` 的 finding

### Requirement: Task headings
在 plan.md 中，每個文字以 Task 一詞加數字開頭的標題 SHALL 是形式嚴格為 `### Task N: <name>` 的三級標題，N 為正整數。task 編號 SHALL 在各 task 與 `## Cancelled Tasks` 下的項目之間唯一。

#### Scenario: Task heading at the wrong level
- **WHEN** plan.md 有標題 `## Task 3: Parser`
- **THEN** lint 在該行回報 `task-heading` finding

#### Scenario: Duplicate task number
- **WHEN** 兩個 task 的編號都是 2
- **THEN** lint 回報指名兩行的 `task-heading` finding

### Requirement: Reuse table
plan.md SHALL 有 `## Reuse` 段落，內含一個表格，欄位為 Need、Existing (path:symbol)、Decision、Reason 與 Evidence，且至少有一列。Decision SHALL 是 `reuse`、`extend` 或 `new`。對 `reuse` 與 `extend`，Existing 中第一個冒號之前的路徑 SHALL 存在於 worktree，但 Existing 以 `external:` 開頭者除外，它指的是 repository 之外的相依套件；對 `new`，Reason SHALL 非空。每一列 SHALL 有非空的 Evidence。

#### Scenario: Reuse row with a missing path
- **WHEN** 某個 `reuse` 列指名 `internal/gitx/run.go:Run`，而該檔案不存在
- **THEN** lint 在該列回報 `reuse-table` finding

#### Scenario: New row without a reason
- **WHEN** 某個 `new` 列的 Reason 是空的
- **THEN** lint 在該列回報 `reuse-table` finding

#### Scenario: External dependency row
- **WHEN** 某個 `reuse` 列的 Existing 是 `external: git CLI`
- **THEN** lint 不去找路徑，也不回報該列的任何事

#### Scenario: Empty reuse table
- **WHEN** Reuse 表格只有表頭，沒有任何列
- **THEN** lint 對該表格回報 `reuse-table` finding

#### Scenario: Row without evidence
- **WHEN** 某個 `reuse` 列的 Evidence 儲存格是空的
- **THEN** lint 在該列回報 `reuse-table` finding

### Requirement: Scenario references
每個 task SHALL 有一行 `**Scenarios:**`，內容為 `none` 或以逗號分隔的 `<capability-path>#<scenario-slug>` 清單。每個參照 SHALL 指名 change 的 delta 或目前主 spec 中的某個情境。change 的 ADDED 與 MODIFIED 需求底下的每個情境 SHALL 至少被一個 task 參照。

#### Scenario: Scenario nobody claims
- **WHEN** delta 在 `booking` 新增了情境「Join when full」，而沒有任何 task 列出 `booking#join-when-full`
- **THEN** lint 對它回報 `scenario-unclaimed` finding

#### Scenario: Reference to an unknown scenario
- **WHEN** 某個 task 列出 `booking#join-when-ful`
- **THEN** lint 在該 task 回報 `scenario-ref` finding

#### Scenario: Scaffolding task
- **WHEN** 某個 task 的該行是 `**Scenarios:** none`
- **THEN** lint 不回報該 task 的情境的任何事

### Requirement: Scenario slugs
情境的 slug SHALL 是其名稱的小寫形式，名稱中任何文字的字母與數字以外的每一段連續字元都換成一個連字號，且開頭與結尾沒有連字號。在同一個 capability 中，取主 spec 的需求、去掉 delta 修改或移除的、再加上 delta 新增與修改的，其中任兩個情境 SHALL NOT 共用 slug。

#### Scenario: Two scenarios named Success
- **WHEN** `booking` 的 delta 中有兩個需求，各有一個名為「Success」的情境
- **THEN** lint 回報指名兩者的 `scenario-duplicate` finding

### Requirement: Cancelled and compensating tasks
`## Cancelled Tasks` 底下的每個項目 SHALL 是 `- Task N: <name> - <reason>` 的形式，且 reason 非空。每一行 `**Compensates:** Task N` SHALL 指名同一份 plan 中屬於 `### Task` 標題的 task。

#### Scenario: Cancelled task without a reason
- **WHEN** Cancelled Tasks 底下的項目寫成 `- Task 5: Front desk screen`
- **THEN** lint 在該行回報 `cancelled-task` finding

#### Scenario: Compensating a cancelled task
- **WHEN** 某個 task 寫了 `**Compensates:** Task 5`，而 Task 5 列在 Cancelled Tasks 之下
- **THEN** lint 在該行回報 `compensates` finding

### Requirement: Routing
lint SHALL 回報目前分支在與預設分支 tip 的 merge base 之後新增的、位於 `docs/superpowers/specs/` 或 `docs/superpowers/plans/` 下的每個檔案，不論已 commit 或在 worktree 中尚未 commit，因為 design 與 plan 屬於 change 資料夾。找不到預設分支或 merge base 時，lint SHALL 把分支新增的檔案列為無法讀取一次，對 routing、record 與 `Record` trailer 都一樣，並仍執行其他規則。

#### Scenario: Design written to the superpowers folder
- **WHEN** worktree 有新的未追蹤檔案 `docs/superpowers/specs/2026-10-25-x-design.md`
- **THEN** lint 回報指名該檔案的 `routing` finding

#### Scenario: No default branch for routing
- **WHEN** repository 沒有 Docket 找得到的預設分支，而某個 change 有 `proposal-section` 問題
- **THEN** lint 回報該 `proposal-section` finding，把分支新增的檔案列為無法讀取，並以 3 結束

#### Scenario: One unreadable entry
- **WHEN** repository 沒有 Docket 找得到的預設分支，而使用者執行 `dkt lint`
- **THEN** lint 恰好列出一個無法讀取的項目

### Requirement: Unparseable structure
對 OpenSpec 1.14 不會套用的 delta 內容，lint SHALL 回報為 `unparseable`，不得略過：位於 ADDED、MODIFIED、REMOVED 與 RENAMED 段落之外的需求標題、這四個與 Purpose 以外的二級段落，以及沒有成對的 RENAMED FROM 或 TO 行。圍欄程式碼區塊內的標題 SHALL 被忽略，如同 OpenSpec 忽略它們。

#### Scenario: Requirement outside any delta section
- **WHEN** delta 檔案在 `## Notes` 底下有 `### Requirement: Cap`
- **THEN** lint 在該行回報 `unparseable` finding

#### Scenario: Header inside a code block
- **WHEN** 某個需求的文字含有一個圍欄程式碼區塊，其中有一行 `### Requirement:`
- **THEN** 該行不被當作需求讀取

#### Scenario: Unpaired rename
- **WHEN** RENAMED 段落有一行 FROM，其後沒有 TO 行
- **THEN** lint 在該 FROM 行回報 `unparseable` finding

### Requirement: Agreement with OpenSpec
對 OpenSpec 1.14 接受的每個 delta 檔案，Docket 的讀取結果 SHALL 產生與 `openspec show <change> --type change --json --deltas-only` 相同的 capability、操作、需求名稱與情境名稱，且順序相同。

#### Scenario: Examples agree with OpenSpec
- **WHEN** 契約測試以 OpenSpec 1.14.3 對共用範例執行
- **THEN** Docket 的讀取結果與 OpenSpec 的輸出對每個範例都相符

### Requirement: Titles
lint SHALL 在下列情況回報 `title-missing`：被檢查的 change 沒有 title（其他 schema 的 change 除外）；被檢查 change 的某個 capability 的 delta，而該 capability 在 worktree 中沒有主 spec，且該 delta 旁邊沒有 title；以及沒有指名 change 時，每個沒有 title 的主 spec。不是 YAML mapping 的 `docket.yaml` SHALL 被回報為 `metadata-invalid`。

#### Scenario: Change without title
- **WHEN** `add-x` 沒有 `docket.yaml`，而使用者執行 `dkt lint add-x`
- **THEN** lint 回報指名 `openspec/changes/add-x/docket.yaml` 的 `title-missing` finding

#### Scenario: New capability without title
- **WHEN** `add-x` 有 `booking` 的 delta，worktree 沒有 `openspec/specs/booking/spec.md`，且 `openspec/changes/add-x/specs/booking/docket.yaml` 不存在
- **THEN** lint 對 `booking` 回報 `title-missing` finding

#### Scenario: Existing capability needs no delta title
- **WHEN** `add-x` 有 `booking` 的 delta，主 spec `booking` 存在，且 delta 旁邊沒有 `docket.yaml`
- **THEN** lint 不對 `booking` 回報 `title-missing` finding

#### Scenario: Main spec without title
- **WHEN** `openspec/specs/booking/spec.md` 存在，旁邊沒有 `docket.yaml`，而使用者執行 `dkt lint`
- **THEN** lint 回報指名 `openspec/specs/booking/docket.yaml` 的 `title-missing` finding

#### Scenario: Named change skips main specs
- **WHEN** 某個主 spec 沒有 title，而使用者執行 `dkt lint add-x`，其中 `add-x` 有 title
- **THEN** lint 不回報關於該主 spec 的 finding

#### Scenario: Broken docket.yaml
- **WHEN** `openspec/changes/add-x/docket.yaml` 內容是 YAML list
- **THEN** lint 對該檔案回報 `metadata-invalid` finding

#### Scenario: Change of another schema has no title
- **WHEN** `add-x` 是其他 schema 的 change，沒有 `docket.yaml`
- **THEN** lint 不回報指名 `openspec/changes/add-x/docket.yaml` 的 `title-missing` finding

### Requirement: Issues linted
未給名稱時，`dkt lint` SHALL 也檢查分支新增的 `docket/issues/` 下每個檔案型 issue（自與預設分支的 merge base 之後已 commit 的，或尚未 commit 的），並照原樣從 worktree 讀取。分支新增的檔案無法讀取時，`Routing` 所列出的那一個無法讀取的項目 SHALL 同時代表這些 issue。命令列上給的名稱 SHALL NOT 指名 issue。

#### Scenario: Lint issues the branch adds
- **WHEN** 預設分支含有檔案型 issue `2026-10-01-a`，分支新增了檔案型 issue `2026-11-02-b`，而使用者執行 `dkt lint`
- **THEN** `2026-11-02-b` 被檢查，`2026-10-01-a` 不檢查

#### Scenario: Issues without a default branch
- **WHEN** repository 沒有 Docket 找得到的預設分支，而 worktree 有一個未 commit 的檔案型 issue
- **THEN** lint 恰好列出一個無法讀取的項目，且不對該 issue 回報 finding

### Requirement: Changes of another schema
lint SHALL 把每個被檢查的其他 schema 的 change，連同其 slug 與 schema，列在 `result.other_schema` 與文字輸出中。對這類 change，它 SHALL 只適用「其新增的 capability 沒有主 spec」的 title 規則，而不適用其 proposal、delta、plan 或中繼資料的其他任何規則。

#### Scenario: Change of another schema
- **WHEN** `add-x` 的 `.openspec.yaml` 有 `schema: spec-driven`，`proposal.md` 沒有 Non-Goals，有 `tasks.md`，沒有 `docket.yaml`，有既有 capability `booking` 的 delta，而使用者執行 `dkt lint --json`
- **THEN** `result.other_schema` 含有 `add-x` 與 `spec-driven`，沒有任何 finding 指名 `add-x`，且指令以 0 結束

#### Scenario: New capability of another schema
- **WHEN** 同一個 `add-x` 有 `waitlist` 的 delta，worktree 沒有 `openspec/specs/waitlist/spec.md`，且該 delta 旁邊沒有 `docket.yaml`
- **THEN** lint 對 `waitlist` 回報 `title-missing` finding，並且不對 `add-x` 回報其他 finding

### Requirement: Order and ideas linted
未給名稱時，`dkt lint` SHALL 也檢查 `docket/order.md` 與 `docket/ideas/` 下的每個 idea，並照原樣從 worktree 讀取（含未 commit 的修改）。兩者都沒有的 worktree SHALL 不得到任何關於它們的 finding。命令列上給了名稱時，它 SHALL 兩者都不檢查。

#### Scenario: Order and ideas linted
- **WHEN** worktree 的 `docket/order.md` 的行編號為 1、2、4，沒有指名 change 或 record，而使用者執行 `dkt lint`
- **THEN** lint 回報 `order-format`，並以 1 結束

#### Scenario: Order not linted with names
- **WHEN** worktree 的 `docket/order.md` 的行編號為 1、2、4，而使用者執行 `dkt lint add-x`
- **THEN** lint 不回報關於 order 的 finding
