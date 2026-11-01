# issues Specification

## Purpose
定義 issue：預期行為已經確定、但現在不做的工作。涵蓋 Docket 為它們附帶的範本、
未指定外部工具的專案放在 `docket/issues/` 底下的檔案 issue、`dkt issue` 如何寫入、
`dkt lint` 如何檢查、`dkt status` 如何列出，以及 `docket` skill 如何使用 issue。

## Requirements

### Requirement: Issue template
Docket SHALL 附帶一份 issue 範本，依序含二級章節 `## Current`、`## Expected`、`## Unchanged` 與 `## Found`，每個章節各有一段 HTML 註解說明該寫什麼；`## Expected` 底下的註解 SHALL 詢問預期行為在哪裡確定。`dkt issue template` SHALL 印出這些章節，加上 `--json` 時以 `result.text` 給出。

#### Scenario: Template printed
- **WHEN** 使用者執行 `dkt issue template --json`
- **THEN** `result.text` 依序含 `## Current`、`## Expected`、`## Unchanged` 與 `## Found`，且指令以 0 結束

### Requirement: Issue file
檔案 issue SHALL 是 `docket/issues/<slug>.md`：YAML front matter 含 `title`（符合 `docket-metadata` 所定義的形式）與非空的 `source`，可選的 `dropped: true`，以及指名 `docket/ideas/` 底下某檔案的 `idea`；其後是範本的四個章節。

#### Scenario: Issue written
- **WHEN** 使用者在 2026-11-02 執行 `dkt issue lint-timeout --title "Lint timeout" --source "review of #40"`
- **THEN** `docket/issues/2026-11-02-lint-timeout.md` 的 front matter 含該 title 與 source，並含範本的四個章節，`result.slug` 是 `2026-11-02-lint-timeout`，且指令以 0 結束

### Requirement: Issue creation
`dkt issue <name>` SHALL 要求 `--title` 與非空的 `--source`，缺少或無效時 SHALL 以 2 結束且不寫入任何東西。name SHALL 遵循 `dkt new` 的 slug 規則；slug SHALL 是今天的日期 `YYYY-MM-DD`、一個連字號與 name，而以日期開頭的 name SHALL 以 1 結束。name 為 `template` 時 SHALL 改為印出範本。

#### Scenario: Missing source writes nothing
- **WHEN** 使用者執行 `dkt issue lint-timeout --title "Lint timeout"`
- **THEN** 指令以 2 結束且不寫入任何東西

#### Scenario: Issue name with a date
- **WHEN** 使用者執行 `dkt issue 2026-11-02-x --title X --source x`
- **THEN** 指令以 1 結束、說明 `dkt issue` 會自己加上日期，且不寫入任何東西

### Requirement: Issue slug never reused
`dkt issue` SHALL NOT 寫入 slug 已被任何掃描到的來源用於某個 issue 的 issue。slug 已被使用時，它 SHALL 將 issue 寫在 `<slug>-2`、`<slug>-3`、… 中第一個沒有被任何掃描到的來源使用、且在 worktree 中不存在的 slug 之下，並 SHALL 回報該 slug。任何來源無法讀取時，它 SHALL 不寫入任何東西並以 3 結束。

#### Scenario: Same name same day
- **WHEN** `docket/issues/2026-11-02-x.md` 存在於某個分支，且使用者在 2026-11-02 於另一個 worktree 執行 `dkt issue x --title X --source x`
- **THEN** 指令寫入 `docket/issues/2026-11-02-x-2.md`，`result.slug` 是 `2026-11-02-x-2`，並以 0 結束

#### Scenario: Unreadable source blocks the issue
- **WHEN** 列出的某個 worktree 無法讀取，且使用者執行 `dkt issue x --title X --source x`
- **THEN** 不寫入任何東西，指令以 3 結束

### Requirement: Issues folder not written through a link
`dkt issue` SHALL NOT 透過符號連結寫入：當 `docket`、`docket/issues` 或該 issue 的路徑是符號連結時，它 SHALL 不寫入任何東西、指出該連結並以 1 結束。

#### Scenario: Symlinked issues folder refused
- **WHEN** `docket/issues` 是指向儲存庫外某目錄的符號連結，且使用者執行 `dkt issue x --title X --source x`
- **THEN** 連結兩側都不寫入任何東西，結果指出該連結，指令以 1 結束

### Requirement: Failed issue write leaves nothing
`dkt issue` 在開始寫入後寫入 issue 失敗時，SHALL 移除寫到一半的檔案與該次執行建立的任何資料夾，並 SHALL 回報錯誤；無法移除其中任何一項時，錯誤 SHALL 指出留下了什麼。

#### Scenario: Failed write leaves nothing
- **WHEN** `docket/issues` 不存在，且因磁碟已滿而寫入新 issue 失敗
- **THEN** 之後 issue 檔案與 `docket/issues` 都不存在，且指令回報錯誤

### Requirement: Issue front matter checked
`dkt lint` SHALL 對以下的檔案 issue 回報 `issue-front-matter`：front matter 缺少、不是 YAML mapping、缺少有效的 `title` 或非空的 `source`、`dropped` 的值不是 `true`、或 `idea` 指不到 worktree 中 `docket/ideas/` 底下的檔案。

#### Scenario: Issue front matter invalid
- **WHEN** 分支新增的某個檔案 issue 有 `idea: docket/ideas/2026-11-02-none.md`，而 worktree 沒有這個檔案
- **THEN** `dkt lint` 對該 issue 回報 `issue-front-matter`，並以 1 結束

### Requirement: Issue sections checked
`dkt lint` SHALL 對以下的檔案 issue 回報 `issue-section`：缺少四個章節之一、章節順序錯誤、有其他二級標題、或某章節除了空白與 HTML 註解外沒有文字。對於範本遺留的文字，它 SHALL 回報 `placeholder`。

#### Scenario: Issue section empty
- **WHEN** 分支新增的某個檔案 issue 的 `## Unchanged` 底下什麼都沒有
- **THEN** `dkt lint` 回報 `issue-section` 並指出 `Unchanged`，並以 1 結束

#### Scenario: Issue template text left
- **WHEN** 某個檔案 issue 的 `## Found` 底下仍留著範本的註解
- **THEN** `dkt lint` 對該行回報 `placeholder`

### Requirement: Issue section word limits
檔案 issue 的四個章節，每個在 HTML 註解之外 SHALL 最多 60 個字，計算方式與 `records` 計算 record 字數的方式相同。`dkt lint` SHALL 回報 `issue-too-long` 並指出該章節與其字數。

#### Scenario: Issue section over the limit
- **WHEN** 某個檔案 issue 的 `## Current` 有 61 個英文字
- **THEN** `dkt lint` 回報 `issue-too-long` 並指出 `Current` 與 61

### Requirement: Issue sources checked
`dkt lint` SHALL 對它所檢查的每個 change，以及分支新增的每個 record，只要其 source 依 `docket-metadata` 的定義指名 `docket/issues/` 底下的路徑，而該路徑在 worktree 中不是檔案 issue，就回報 `issue-source` 並指出該 source。

#### Scenario: Source names no issue
- **WHEN** 分支新增的某個 record 有 `source: docket/issues/2026-11-02-x.md`，而 worktree 沒有這個檔案
- **THEN** `dkt lint` 回報 `issue-source` 並指出該 source，並以 1 結束

#### Scenario: Source names an issue
- **WHEN** 分支新增的某個 record 有 `source: docket/issues/2026-11-02-x.md`，且 worktree 持有該檔案
- **THEN** `dkt lint` 不回報 `issue-source`

#### Scenario: Source with a note names no issue
- **WHEN** 分支新增的某個 record 有 `source: docket/issues/2026-11-02-x.md, from review of #40`，而 worktree 沒有這個檔案
- **THEN** `dkt lint` 回報 `issue-source` 並指出該 source

### Requirement: Issue state
檔案 issue 的 handler SHALL 是 source 依 `docket-metadata` 的定義指名該 issue 路徑的 record 與 change（不論進行中或已封存），預設分支的 tip 持有它時從 tip 讀取，否則從它的第一個 source 讀取。其狀態 SHALL 在 `dropped: true` 時為 `dropped`，有 `idea` 時為 `moved`；其餘情況：沒有 handler 時為 `pending`，tip 並未持有某個 handler 時為 `in-progress`，tip 持有所有 handler 時為 `handled`。

#### Scenario: Handler on a branch
- **WHEN** 預設分支的 tip 持有 `docket/issues/2026-11-02-x.md`，而某個未合併的分支持有 source 為該路徑的 record
- **THEN** 該 issue 的狀態是 `in-progress`

#### Scenario: Handler merged
- **WHEN** 預設分支的 tip 持有該 issue 與一個 source 為其路徑的 record，且沒有其他 source 持有 tip 所沒有的 handler
- **THEN** 該 issue 的狀態是 `handled`

#### Scenario: Merged change not yet archived
- **WHEN** 預設分支的 tip 持有進行中的 change `add-x`，其 `docket.yaml` 以該 issue 的路徑為 source
- **THEN** 該 issue 的狀態是 `handled`

#### Scenario: Merged record edited elsewhere
- **WHEN** 預設分支的 tip 持有一個 source 為 `review of #40` 的 record，而 worktree B 把該 record 的 source 改成該 issue 的路徑
- **THEN** 該 record 不是該 issue 的 handler

#### Scenario: Handler with a note
- **WHEN** 預設分支的 tip 持有該 issue 與一個 source 為其路徑後接 `, found while fixing y` 的 record
- **THEN** 該 issue 的狀態是 `handled`

### Requirement: Issues listed by status
`dkt status` SHALL 在 `result.issues` 中依 slug 排序，列出掃描找到的每個狀態為 `pending` 或 `in-progress` 的檔案 issue，各附其 slug、取自第一個 source 的 front matter 的 title 與 source（無法讀取時為 null）、其狀態、其 handler（種類 `record` 或 `change`、slug，以及預設分支的 tip 是否持有它）與其 source 及對應 commit，並在 record 之後為每個 issue 輸出一行。指定 slug 時，它 SHALL 不列出任何 issue。

#### Scenario: Pending issue listed
- **WHEN** 預設分支的 tip 持有 `docket/issues/2026-11-02-x.md`，沒有任何 record 或 change 指名它，且使用者執行 `dkt status --json`
- **THEN** `result.issues` 列出 `2026-11-02-x`，狀態為 `pending`，附其 title 與 source，沒有 handler

#### Scenario: Issue in progress
- **WHEN** worktree B 持有一個 source 為 `docket/issues/2026-11-02-x.md` 的未提交 record，且使用者在 worktree A 執行 `dkt status --json`
- **THEN** `result.issues` 列出 `2026-11-02-x`，狀態為 `in-progress`，並以該 record 為未合併的 handler

#### Scenario: Handled issue hidden
- **WHEN** 預設分支的 tip 持有該 issue 與一個指名它的已合併 record，且沒有其他 source 持有 handler
- **THEN** `result.issues` 不列出它

#### Scenario: Dropped and moved hidden
- **WHEN** 一個檔案 issue 有 `dropped: true`，另一個有 `idea: docket/ideas/2026-11-02-y.md`，且兩者都沒有 handler
- **THEN** `result.issues` 兩者都不列出

#### Scenario: Issues not listed with slugs
- **WHEN** 某個檔案 issue 是 pending，且使用者執行 `dkt status add-x --json`
- **THEN** `result.issues` 為空

### Requirement: Issues for settled work
`docket` skill SHALL 讓 agent 把預期行為已確定、但現在不做的工作變成 issue：專案的 `AGENTS.md` 有 `## Issues` 章節指名工具時，使用者同意在該處提出後，以 `dkt issue template` 印出的章節在該工具中建立；否則用 `dkt issue`。指不出預期行為的工作 SHALL 改成 idea。

#### Scenario: Bug found out of scope
- **WHEN** 某專案的 `AGENTS.md` 沒有 `## Issues` 章節，其中的 agent 在其他工作中發現一個 spec 有載明預期行為的 bug
- **THEN** 它用 `dkt issue` 開一個檔案 issue，而不是寫 idea

#### Scenario: Issue in the project tool
- **WHEN** 專案的 `AGENTS.md` 有 `## Issues` 指名 GitHub Issues，且 agent 發現這樣的 bug
- **THEN** 它把依 `dkt issue template` 印出的章節寫成的 issue 給使用者看，使用者同意後才在該處提出，且不執行 `dkt issue`

### Requirement: Work names its issue
當工作處理某個 issue 時，`docket` skill SHALL 讓 agent 把該 issue 作為 record 或 change 的 `--source` 傳入：檔案 issue 傳其路徑，外部工具的 issue 傳其位址。

#### Scenario: Record names its issue
- **WHEN** 走輕審路徑的 agent 修正 `docket/issues/2026-11-02-x.md` 的 bug
- **THEN** 它執行 `dkt record`，並帶 `--source docket/issues/2026-11-02-x.md`
