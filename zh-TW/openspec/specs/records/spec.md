# records Specification

## Purpose
定義 record：輕審路徑上每一件工作一個檔案，說明做了什麼、為什麼、如何檢查，以及審查發現了什麼；`dkt record` 如何寫入 record、`dkt lint` 如何檢查它，以及 `docket` skill 如何使用它。

## Requirements

### Requirement: Record file
record SHALL 是 `docket/records/<slug>.md`：YAML front matter 含 `title`（採 `docket-metadata` 定義的形式）、`kind`（`fix`、`refactor`、`test`、`prompt`、`checks` 之一）與非空的 `source`，接著依序是二級段落 `## What`、`## Why`、`## Verification` 與 `## Review`，其後可選擇性地有 `## Tasks`。

#### Scenario: Record written
- **WHEN** 使用者在 2026-10-30 執行 `dkt record fix-lint-timeout --kind fix --title "Lint timeout" --source "review of #40"`
- **THEN** `docket/records/2026-10-30-fix-lint-timeout.md` 的 front matter 含有該 title、kind 與 source，並有所附範本的四個段落，`result.slug` 為 `2026-10-30-fix-lint-timeout`，且指令以 0 結束

### Requirement: Record creation
`dkt record <name>` SHALL 要求 `--kind`（五種之一）、非空的 `--source` 與 `--title`，缺少或無效時 SHALL 以 2 結束且不寫任何東西。名稱 SHALL 遵守 `dkt new` 的 slug 規則；record 的 slug SHALL 是今天的日期（`YYYY-MM-DD`）、一個連字號，再加上名稱。

#### Scenario: Unknown kind
- **WHEN** 使用者執行 `dkt record fix-x --kind chore --title X --source x`
- **THEN** 指令以 2 結束、列出五種 kind，且不寫任何東西

#### Scenario: Name with a date
- **WHEN** 使用者執行 `dkt record 2026-10-30-fix-x --kind fix --title X --source x`
- **THEN** 指令以 1 結束、說明 `dkt record` 會自己加上日期，且不寫任何東西

### Requirement: Record slug never reused
`dkt record` SHALL NOT 寫入 slug 已被任何掃描到的 source 用於 record 的 record。slug 已被使用時，它 SHALL 把 record 寫在 `<slug>-2`、`<slug>-3`、… 中第一個沒有被任何掃描到的 source 使用、且在 worktree 中不存在者之下，並 SHALL 回報該 slug。任何 source 無法讀取時，它 SHALL 什麼都不寫並以 3 結束。

#### Scenario: Same name same day
- **WHEN** `docket/records/2026-10-30-fix-x.md` 存在於某個分支，而使用者在 2026-10-30 於另一個 worktree 執行 `dkt record fix-x --kind fix --title X --source x`
- **THEN** 指令寫入 `docket/records/2026-10-30-fix-x-2.md`，`result.slug` 為 `2026-10-30-fix-x-2`，並以 0 結束

#### Scenario: Unreadable source blocks the record
- **WHEN** 某個列出的 worktree 無法讀取，而使用者執行 `dkt record fix-x --kind fix --title X --source x`
- **THEN** 什麼都沒寫，且指令以 3 結束

#### Scenario: Concurrent records
- **WHEN** 兩個 `dkt record fix-x --kind fix --title X --source x` 在 2026-10-30 於同一個 worktree 同時開始
- **THEN** 一個寫入 `2026-10-30-fix-x.md`，另一個寫入 `2026-10-30-fix-x-2.md`，且兩者都以 0 結束

### Requirement: Records folder not written through a link
`dkt record` SHALL NOT 透過符號連結寫入：當 `docket`、`docket/records` 或 record 的路徑是符號連結時，它 SHALL 什麼都不寫、指名該連結並以 1 結束。

#### Scenario: Symlinked records folder refused
- **WHEN** `docket/records` 是指向 repository 外某個目錄的符號連結，而使用者執行 `dkt record fix-x --kind fix --title X --source x`
- **THEN** 連結兩側都沒有寫入任何東西，結果指名該連結，且指令以 1 結束

### Requirement: Record front matter checked
`dkt lint` SHALL 對 front matter 缺少、不是 YAML mapping、缺少有效 `title` 或缺少非空 `source` 的 record 回報 `record-front-matter`，並對不屬於這五種的 `kind` 回報 `record-kind`。

#### Scenario: Record kind not known
- **WHEN** 分支新增的某個 record 有 `kind: chore`
- **THEN** `dkt lint` 對該 record 回報 `record-kind`，並以 1 結束

### Requirement: Record sections checked
`dkt lint` SHALL 對缺少四個段落之一、順序錯誤、有其他二級標題，或某個段落除了空白與 HTML 註解之外沒有文字的 record 回報 `record-section`。對遺留自所附 record 範本的文字，它 SHALL 回報 `placeholder`。

#### Scenario: Record section empty
- **WHEN** 分支新增的某個 record 的 `## Review` 底下什麼都沒有
- **THEN** `dkt lint` 回報指名 `Review` 的 `record-section`，並以 1 結束

#### Scenario: Template text left
- **WHEN** record 的 `## Why` 底下仍留著範本的註解
- **THEN** `dkt lint` 在該行回報 `placeholder`

### Requirement: Section word limits
四個段落各自 SHALL 含有至多 60 個字（HTML 註解之外）。一段連續的字母或數字算一個字，但每個中文、日文或韓文字元（包括日文的長音符 `ー`）SHALL 算半個字。`dkt lint` SHALL 回報 `record-too-long`，指名段落與其字數。

#### Scenario: Section over the limit
- **WHEN** record 的 `## What` 含有 61 個英文字
- **THEN** `dkt lint` 回報指名 `What` 與 61 的 `record-too-long`

#### Scenario: Chinese text counted
- **WHEN** record 的 `## Why` 含有 110 個中文字元與 4 個英文字
- **THEN** 它算 59 個字，且 `dkt lint` 不對它回報 `record-too-long`

#### Scenario: Prolonged sound mark counted
- **WHEN** record 的某個段落只含 `コーヒー`
- **THEN** 它算 2 個字

### Requirement: Record tasks
record 的 `## Tasks` 段落存在時，SHALL 以 plan 的格式列出 task，且 `dkt lint` SHALL 依 plan 的 task 標題規則檢查它們，不要求 `**Scenarios:**` 或 `**Review:**` 行。

#### Scenario: Record tasks checked
- **WHEN** record 的 `## Tasks` 含有 `### Task 1: Move` 與另一個 `### Task 1: Rename`
- **THEN** `dkt lint` 回報指名兩行的 `task-heading`

#### Scenario: Record without tasks
- **WHEN** record 沒有 `## Tasks` 段落
- **THEN** `dkt lint` 不對它回報任何 task finding

### Requirement: Light review keeps a record
`docket` skill SHALL 讓 agent 在輕審路徑上：在第一個 commit 之前用 `dkt record` 開一個 record、依工作與審查填寫其段落、為它的每個 commit 在訊息的最後一段加上 trailer `Record: <slug>`（做 record 某個 task 的 commit 另加 `Task: <N>`）、record 有 task 時執行 `dkt sync <slug>`，並在開 PR 之前執行 `dkt lint`。

#### Scenario: Light review opens a record
- **WHEN** 在由 `dkt init` 設定、流程檔為 `ok` 的 repository 中，agent 被要求修正某個 prompt 檔案裡的錯字
- **THEN** 它說明走輕審路徑、在第一個 commit 之前執行 `dkt record`，且它的 commit 帶有指名該 record 的 `Record: <slug>`

#### Scenario: Light review syncs record tasks
- **WHEN** 輕審路徑上的 agent 已在 record 的 `## Tasks` 下列出 task，並以 `Task` trailer commit 了它們
- **THEN** 它在 `dkt lint` 之前執行 `dkt sync <slug>`，且那些 task 在 record 中的步驟在 PR 裡被勾選

### Requirement: Records listed by status
`dkt status` SHALL 在 `result.records` 中列出 `workspace-scan` 所定義的掃描找到、而預設分支的 tip 未持有的每個 record，狀態為 `open`，依 slug 排序。每個 record SHALL 給出其 slug、從它第一個 source 的 front matter 讀到的 title、kind 與 source（front matter 無法讀取時各為 null），以及它被找到的每個 source 與該 source 的 commit。人類可讀的輸出 SHALL 在 change 之後，每個 record 一行。

#### Scenario: Open records listed
- **WHEN** worktree B 有未 commit 的 record `2026-11-01-fix-x`，分支 `refs/heads/fix-y` 有 `2026-11-01-fix-y`，而使用者在 worktree A 執行 `dkt status --json`
- **THEN** `result.records` 依該順序列出 `2026-11-01-fix-x` 與 `2026-11-01-fix-y`，各自狀態為 `open`，並附有 title、kind 與 source，以及其 source

#### Scenario: Merged records hidden
- **WHEN** 預設分支的 tip 有 record `2026-10-31-fix-a`，沒有其他 source 有 record，而使用者執行 `dkt status --json`
- **THEN** `result.records` 是空的，且 `result.records_total` 為 0

#### Scenario: Record front matter unreadable
- **WHEN** 某個 open record 的檔案沒有 front matter
- **THEN** 它被列出，`title`、`kind` 與 `source` 為 null，且指令以 0 結束

### Requirement: Record task progress
每個列出的 record SHALL 給出其 `## Tasks` 的 task 與進度，方式同 `change-status` 對 change 的做法；每個 task 的 commit 取自「從第一個 source 的 commit 可到達、從預設分支 tip 不可到達」且帶有 `Record: <slug>` 與 `Task: <N>` 的 commit。沒有 `## Tasks` 的 record SHALL 不給 task，進度為零。

#### Scenario: Record tasks from trailers
- **WHEN** 分支上某個 open record 有兩個 task，沒有任何步驟被勾選，且該分支上有一個 commit 帶有 `Record: <slug>` 與 `Task: 1`
- **THEN** Task 1 為 `started`、列出該 commit 並標為未同步，Task 2 為 `pending`

### Requirement: Merged records
有 `--merged` 時，`dkt status` SHALL 也列出預設分支 tip 持有的每個 record，狀態為 `merged`，排在 open 的之後，合併時間新的在前，同時間者依 slug。其 `merge` SHALL 是預設分支 tip 的 first-parent 歷史上、最新一個新增該 record 檔案的 commit，連同其 committer 時間；歷史中看不出時為 null。其 `tag` SHALL 是其 commit 含有該檔案的第一個 tag（依建立日期再依名稱），或 null。

#### Scenario: Merged records with merge and tag
- **WHEN** `2026-10-30-fix-a` 在 2026-10-30 以 merge commit 合併，其後建立了 tag `v0.9.1`，而使用者執行 `dkt status --merged --json`
- **THEN** `2026-10-30-fix-a` 被列出，狀態為 `merged`，`merge.commit` 是該 merge commit，`merge.time` 是其 committer 時間，`tag` 是 `v0.9.1`

### Requirement: Merged record filters
`--since <rev>` SHALL 只保留其 `merge` commit 為 `<rev>` 所未到達的已合併 record，`merge` 為 null 的則排除。`--limit <n>` SHALL 最多列出 `n` 個已合併 record，新的在前。`result.records_total` SHALL 計入套用 `--limit` 之前篩選條件所符合的 record 數，人類可讀的輸出 SHALL 說明 `--limit` 略去了多少個。沒有 `--merged` 而使用 `--since` 與 `--limit`，或三者之任一與 slug 並用，SHALL 是使用錯誤。

#### Scenario: Merged since
- **WHEN** `2026-10-29-fix-a` 在 tag `v0.9.1` 之前合併，`2026-10-31-fix-b` 在其後合併，而使用者執行 `dkt status --merged --since v0.9.1 --json`
- **THEN** 已合併的 record 中只列出 `2026-10-31-fix-b`

#### Scenario: Limit reports total
- **WHEN** 預設分支有五個 record，沒有其他 record，而使用者執行 `dkt status --merged --limit 2 --json`
- **THEN** 列出最新的兩個已合併 record，`result.records_total` 為 5，且人類可讀的輸出說明另有三個被略去

#### Scenario: Filter flags need merged
- **WHEN** 使用者執行 `dkt status --limit 2`
- **THEN** 指令以 2 結束

### Requirement: Record in focus
給了 slug 時，`dkt status` SHALL 列出任何掃描到的 source 持有的每個被指名的 record，不論 open 或 merged，已合併者附上其 `merge` 與 `tag`，並 SHALL 對沒有任何 source 持有的被指名 record 回報 `record-missing`。

#### Scenario: Named record listed
- **WHEN** 預設分支 tip 有 `2026-10-30-fix-a`，而使用者執行 `dkt status 2026-10-30-fix-a --json`
- **THEN** `result.records` 列出它，狀態為 `merged`，附有其 `merge` 與 `tag`，且指令以 0 結束

#### Scenario: Named record missing
- **WHEN** 沒有任何 source 持有 `2026-11-01-fix-q`，而使用者執行 `dkt status 2026-11-01-fix-q`
- **THEN** 指令回報指名它的 `record-missing`，並以 1 結束

### Requirement: Record tasks ticked by sync
給以日期開頭的 slug 的 `dkt sync <slug>` SHALL 在目前 worktree 的 `docket/records/<slug>.md` 中，勾選 `## Tasks` 底下每個被 commit 以 `Task` trailer 指名的 task 的步驟；所取的 commit 是從 HEAD 可到達、從預設分支 tip 不可到達、且帶有 `Record: <slug>` 者。`result.record` SHALL 是該 slug，`result.change` 為 null。`--check` SHALL 與對 change 時的行為相同。

#### Scenario: Record tasks ticked
- **WHEN** 分支上有一個 commit 帶有 `Record: 2026-11-01-fix-x` 與 `Task: 2`，而該 record 的 Task 2 有兩個未勾選的步驟
- **THEN** 執行 `dkt sync 2026-11-01-fix-x` 後兩個步驟都被勾選，且 `result.record` 為 `2026-11-01-fix-x`

#### Scenario: Record sync ignores merged commits
- **WHEN** 預設分支 tip 可到達的某個 commit 帶有 `Record: 2026-11-01-fix-x` 與 `Task: 1`
- **THEN** `dkt sync 2026-11-01-fix-x` 不會因該 commit 而勾選 Task 1

#### Scenario: Record sync check
- **WHEN** record 的 Task 2 有 commit 且有未勾選的步驟，而使用者執行 `dkt sync <slug> --check`
- **THEN** record 不變，Task 2 被列出，且指令以 1 結束

### Requirement: Record sync problems
record sync SHALL 對該 record 的某個 commit 上、指名了 record 中不存在的 task 的 `Task` trailer 回報 `task-unknown`，並仍勾選其他的；worktree 沒有該 record 時回報 `record-missing` 並以 1 結束；`docket`、`docket/records` 或 record 是符號連結時 SHALL 什麼都不寫、指名該連結並以 1 結束。沒有預設分支時 SHALL 以 4 結束。

#### Scenario: Record sync unknown task
- **WHEN** 某個 commit 帶有 `Record: <slug>` 與 `Task: 9`，而 record 沒有 Task 9
- **THEN** `dkt sync <slug>` 回報指名該 commit 的 `task-unknown`、勾選其他 task，並以 1 結束

#### Scenario: Record sync needs default branch
- **WHEN** 找不到預設分支，而使用者執行 `dkt sync 2026-11-01-fix-x`
- **THEN** 指令什麼都不寫，並以 4 結束

#### Scenario: Record sync through link
- **WHEN** `docket/records` 是符號連結，而使用者執行 `dkt sync 2026-11-01-fix-x`
- **THEN** 連結兩側都沒有寫入任何東西，結果指名該連結，且指令以 1 結束

### Requirement: Record trailers checked
未給名稱時，`dkt lint` SHALL 對自與預設分支的 merge base 之後的每個 commit，只要其 `Record` trailer 沒有指名 worktree 中 `docket/records/` 下的任何檔案，就回報 `record-trailer`，指名該 commit 與該值。

#### Scenario: Trailer names no record
- **WHEN** record 是 `2026-11-01-fix-x`，而分支新增的某個 commit 帶有 `Record: fix-x`
- **THEN** `dkt lint` 回報指名該 commit 與 `fix-x` 的 `record-trailer`，並以 1 結束

#### Scenario: Trailer names a record
- **WHEN** 分支新增的每個帶有 `Record` trailer 的 commit 都指名 worktree 所持有的 `2026-11-01-fix-x`
- **THEN** `dkt lint` 不回報 `record-trailer`

### Requirement: Failed record write leaves nothing
`dkt record` 在開始之後寫入 record 失敗時，SHALL 移除不完整的檔案與這次執行所建立的任何資料夾，並 SHALL 回報錯誤；無法移除其中之一時，錯誤 SHALL 指名留下的東西。

#### Scenario: Failed write leaves nothing
- **WHEN** `docket/records` 不存在，而寫入新 record 因磁碟已滿而失敗
- **THEN** 之後 record 檔案與 `docket/records` 都不存在，且指令回報錯誤
