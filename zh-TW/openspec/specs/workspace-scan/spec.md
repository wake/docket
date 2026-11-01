# workspace-scan Specification

## Purpose
定義 Docket 到哪裡找 change，以及如何回報，讓在另一個 worktree、另一個分支或另一台機器上進行中的
change 被看見，而 Docket 讀不到的地方絕不被當成空的。

## Requirements

### Requirement: Sources scanned
掃描 SHALL 在以下位置找出 change：`git worktree list` 回報的每個 worktree（讀取其檔案，含未提交與未追蹤的）；預設分支 tip 未包含其 tip 的每個本地分支與每個遠端追蹤分支；以及預設分支的 tip（含其封存）。

#### Scenario: Change only in another worktree
- **WHEN** worktree B 持有未提交的 `openspec/changes/add-x/`，且掃描從 worktree A 執行
- **THEN** 掃描回報 `add-x`，其 source 為 worktree B

#### Scenario: Change only on a remote-tracking branch
- **WHEN** `refs/remotes/origin/feature-y` 持有 `openspec/changes/add-y/`，而沒有任何 worktree 或本地分支持有
- **THEN** 掃描回報 `add-y`，其 source 為 `refs/remotes/origin/feature-y`

#### Scenario: Merged branch skipped
- **WHEN** 某個本地分支的 tip 已包含在預設分支的 tip 之中
- **THEN** 掃描不把該分支當成獨立的 source 來讀取

### Requirement: Default branch
Docket SHALL 在給定 `--default-branch` 時以它作為預設分支；否則取 `refs/remotes/origin/HEAD` 的指向；否則若 `main` 存在就用 `main`，再其次 `master`。其 tip SHALL 在該分支的遠端追蹤 ref 存在時取它，否則取本地分支。找不到預設分支時，指令 SHALL 以 4 結束。

#### Scenario: origin HEAD set
- **WHEN** `refs/remotes/origin/HEAD` 指向 `refs/remotes/origin/trunk`，且未給旗標
- **THEN** Docket 以 `trunk` 為預設分支，以 `refs/remotes/origin/trunk` 為其 tip

#### Scenario: No default branch
- **WHEN** 未給旗標、`refs/remotes/origin/HEAD` 未設定，且 `main` 與 `master` 都不存在
- **THEN** 指令以 4 結束並建議使用 `--default-branch`

### Requirement: One change per slug
change SHALL 以其 slug 識別。同一個 slug 出現在多個 source 時，Docket SHALL 只回報一次並列出每個 source，且 SHALL 依以下順序使用第一個 source 的內容：HEAD 不被預設分支 tip 包含的 worktree、本地分支、遠端追蹤分支、HEAD 被預設分支 tip 包含的 worktree、預設分支的 tip。

#### Scenario: Merged change also on a branch
- **WHEN** `add-x` 存在於預設分支的 tip、檢出預設分支的主 worktree，以及一個編輯過它的未合併分支
- **THEN** 掃描只回報一次 `add-x`、列出全部三個 source，並使用該分支的內容

#### Scenario: Worktree with unmerged work comes first
- **WHEN** worktree B 在一個有預設分支 tip 所沒有的 commit 的分支上，對 `add-x` 有未提交的編輯，而 `add-x` 也存在於 B 的分支與預設分支的 tip
- **THEN** 掃描使用 worktree B 的內容

### Requirement: Archived slugs
掃描到的 source 中，每個目錄 `openspec/changes/archive/<YYYY-MM-DD>-<slug>/` SHALL 算作 `<slug>` 的一次已封存使用。

#### Scenario: Archived slug found
- **WHEN** 預設分支的 tip 持有 `openspec/changes/archive/2026-10-24-add-x/`
- **THEN** 掃描將 `add-x` 回報為已封存

### Requirement: Unreadable sources
git 有列出、但其目錄不存在或無法讀取的 worktree，以及其樹無法讀取的分支，SHALL 連同原因被回報為無法讀取，且指令 SHALL 依 `cli-output` 對無法讀取的 source 的要求結束。整個無法讀取的 source SHALL 只列一次，不為其中的資料夾另列項目。

#### Scenario: Worktree on an unmounted volume
- **WHEN** 列出的某個 worktree 的目錄因其磁碟區未掛載而不存在
- **THEN** 掃描將該 worktree 列為無法讀取，且指令以 3 結束

#### Scenario: Worktree listed once
- **WHEN** 某個 worktree 的 `docket/records` 無法列出，且 git 無法判斷預設分支 tip 是否包含該 worktree 的 HEAD
- **THEN** 掃描將該 worktree 只列為無法讀取一次，且指令以 3 結束

### Requirement: No network
掃描 SHALL NOT 從 remote fetch 或以其他方式連線到 remote。遠端追蹤分支 SHALL 以最後一次 fetch 時的樣子讀取。

#### Scenario: Scan offline
- **WHEN** 網路無法使用且執行掃描
- **THEN** 掃描以現有的遠端追蹤 ref 完成

### Requirement: Records found
掃描 SHALL 在讀取 change 的 source 中找出 record：每個 slug 以日期開頭的一般檔案 `docket/records/<slug>.md`。record SHALL 以其 slug 識別，只回報一次並列出持有它的每個 source，並依 `One change per slug` 所定義的順序從第一個 source 讀取。缺少 `docket/records` SHALL 視為沒有 record，且 SHALL NOT 使該 source 變成無法讀取。

#### Scenario: Record on a branch
- **WHEN** `refs/heads/fix-x` 持有 `docket/records/2026-11-01-fix-x.md`，位於該分支的 worktree B 也持有它，且掃描從 worktree A 執行
- **THEN** 掃描只回報一次 `2026-11-01-fix-x`，worktree B 在前、分支在後

### Requirement: Issues found
掃描 SHALL 在讀取 change 的 source 中找出檔案 issue：每個 slug 以日期開頭的一般檔案 `docket/issues/<slug>.md`。issue SHALL 以其 slug 識別，只回報一次並列出持有它的每個 source，並依 `One change per slug` 所定義的順序從第一個 source 讀取。缺少 `docket/issues` SHALL 視為沒有 issue，且 SHALL NOT 使該 source 變成無法讀取。

#### Scenario: Issue on a branch
- **WHEN** `refs/heads/fix-x` 持有 `docket/issues/2026-11-02-x.md`，位於該分支的 worktree B 也持有它，且掃描從 worktree A 執行
- **THEN** 掃描只回報一次 `2026-11-02-x`，worktree B 在前、分支在後

#### Scenario: No issues folder
- **WHEN** 某個 source 沒有 `docket/issues` 資料夾
- **THEN** 掃描對它不回報任何 issue，且該 source 可讀取
