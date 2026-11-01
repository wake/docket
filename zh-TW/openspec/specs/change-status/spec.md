# change-status Specification

## Purpose
定義 Docket 如何回報每個進行中的 change 的狀態，不論它放在哪裡，讓人與工具（例如網頁工作台）
不必逐一打開 worktree，就能看到每個 change 的階段、task 進度與近期活動。

## Requirements

### Requirement: Every active change listed
`dkt status` SHALL 依 slug 排序、每個只列一次，列出 `workspace-scan` 所定義的掃描找到的每個進行中的 change，附上找到它的每個 source 與該 source 的 commit。它 SHALL 從掃描列在最前面的 source 讀取每個 change 的文件與 commit。它 SHALL NOT 列出已封存的 change，也 SHALL NOT 列出在任何 source 中有已封存用法的 slug。

#### Scenario: Changes in worktrees and branches
- **WHEN** worktree A 持有 `add-x`，`add-y` 存在於 worktree B 與 `refs/remotes/origin/add-y`，且指令在 worktree A 執行
- **THEN** 它依序列出 `add-x` 與 `add-y`，`add-y` 附上兩個 source 與其 commit

#### Scenario: Archived change not listed
- **WHEN** `add-x` 已封存在預設分支的 tip，而某個未合併的分支仍持有 `openspec/changes/add-x/`
- **THEN** 不列出 `add-x`

#### Scenario: No active changes
- **WHEN** 沒有任何 source 持有進行中的 change
- **THEN** 指令不列出任何 change 並以 0 結束

### Requirement: Phase
每個 change SHALL 有一個階段：沒有 `plan.md` 或其 plan 沒有 task 時為 `design`；沒有 task 開始時為 `ready`；每個未取消的 task 都完成時為 `done`；其餘為 `implementing`。task 有步驟且步驟全部勾選時為完成；有步驟被勾選，或有該 change 的 commit 在 `Task` trailer 中指名它時，為已開始。

#### Scenario: Plan not written
- **WHEN** `add-x` 有 `proposal.md` 與 delta spec，但沒有 `plan.md`
- **THEN** 其階段是 `design`

#### Scenario: Nothing started
- **WHEN** `add-x` 有含三個 task 的 plan，沒有勾選任何步驟，也沒有 commit 指名任何 task
- **THEN** 其階段是 `ready`

#### Scenario: Commit names a task
- **WHEN** `add-x` 的 plan 沒有勾選任何步驟，而 `add-x` 的某個 commit 帶有 `Task: 1`
- **THEN** 其階段是 `implementing`，且 Task 1 是 `started`

#### Scenario: Every task done
- **WHEN** `add-x` 的 plan 中每個 task 的每個步驟都已勾選
- **THEN** 其階段是 `done`

#### Scenario: Cancelled task left out
- **WHEN** `add-x` 的 plan 把 Task 3 移到 Cancelled Tasks，且 Task 1 與 Task 2 的每個步驟都已勾選
- **THEN** 其階段是 `done`，Task 3 是 `cancelled`，且 task 進度計為二之二

### Requirement: Task progress
每個 change SHALL 依 plan 順序列出其 `plan.md` 的 task，接著列出已取消的 task，各附其編號、名稱、狀態（`pending`、`started`、`done` 或 `cancelled`）、其步驟與已勾選的步驟，以及指名它的 commit；commit 的取法與 `plan-sync` 相同，但改從第一個 source 的 commit 而非 HEAD 出發。有這類 commit 但仍有未勾選步驟的 task SHALL 標記為未同步。change 也 SHALL 給出完成與全部的 task 數，以及已勾選與全部的步驟數，皆不含已取消的 task。

#### Scenario: Commits not yet synced
- **WHEN** `add-x` 的某個 commit 帶有 `Task: 2`，而 Task 2 有未勾選的步驟
- **THEN** Task 2 列出該 commit 並標記為未同步，且 `plan.md` 不變

### Requirement: Activity
每個 change SHALL 給出它的最近活動：從第一個 source 的 commit 可達、且不是從 change 的 base 可達的 commit 中，在 `Change` trailer 指名該 change、或更動其資料夾內某個檔案的最新一個，附上該 commit 的時間，以及從它到本次執行開始的整日數。當第一個 source 是 worktree 時，change 也 SHALL 給出其資料夾內有多少檔案有未提交的變更。

#### Scenario: Idle change
- **WHEN** `add-x` 最新的 commit 是在執行前十天半建立的
- **THEN** 其最近活動指出該 commit，並給出閒置 10 天

#### Scenario: Uncommitted edits
- **WHEN** `add-x` 是從 worktree 讀取，其中 `plan.md` 與 `design.md` 有未提交的編輯
- **THEN** 它給出 2 個未提交檔案

#### Scenario: No commit yet
- **WHEN** `add-x` 只以未追蹤檔案存在於某個 worktree
- **THEN** 它沒有最近活動，並給出其未提交檔案

### Requirement: Changes in focus
指定 slug 時，`dkt status` SHALL 只依 slug 順序列出那些 change 與 record。以日期開頭的 slug SHALL 指名一個 record，依 `records` 所定義的方式列出；其他 slug SHALL 是進行中的 change，不是進行中 change 的 slug SHALL 被回報，其餘仍照列。

#### Scenario: One change in focus
- **WHEN** `add-x` 與 `add-y` 都在進行中，且使用者執行 `dkt status add-y`
- **THEN** 只列出 `add-y`，且指令以 0 結束

#### Scenario: Slug not active
- **WHEN** 使用者執行 `dkt status add-q`，而沒有名為 `add-q` 的進行中 change
- **THEN** 指令回報 `add-q` 不在進行中，並以 1 結束

#### Scenario: Record in focus
- **WHEN** `add-x` 與 `add-y` 在進行中，record `2026-11-01-fix-x` 是開啟的，且使用者執行 `dkt status add-y 2026-11-01-fix-x --json`
- **THEN** `result.changes` 只列出 `add-y`，`result.records` 只列出 `2026-11-01-fix-x`，且指令以 0 結束

### Requirement: Read-only report
`dkt status` SHALL 不寫入任何東西、不取得鎖、從不 fetch，且 SHALL NOT 要求 OpenSpec CLI。它沒有 finding 可回報時 SHALL 以 0 結束，但無法讀取的 source 依 `cli-output` 的要求處理。

#### Scenario: Unreadable worktree
- **WHEN** 列出的某個 worktree 無法讀取，而另一個 worktree 持有 `add-x`
- **THEN** 列出 `add-x`，該 worktree 被列為無法讀取，且指令以 3 結束

#### Scenario: Writes nothing
- **WHEN** 使用者在有進行中 change 的儲存庫執行 `dkt status`
- **THEN** 任何 worktree 中都沒有檔案被更動，也沒有 ref 移動

### Requirement: Title
每個 change SHALL 給出它的 title，依 `docket-metadata` 的定義從第一個 source 讀取；沒有 title 時為 none。

#### Scenario: Title shown
- **WHEN** `add-x` 的 `docket.yaml` 含 `title: Booking waitlist`，且使用者執行 `dkt status`
- **THEN** `add-x` 那一行顯示 "Booking waitlist"，加上 `--json` 時其 `title` 是 `"Booking waitlist"`

#### Scenario: Change without title
- **WHEN** `add-x` 沒有 `docket.yaml`，且使用者執行 `dkt status --json`
- **THEN** 列出 `add-x`，其 `title` 為 null，且指令以 0 結束

### Requirement: Schema of each change
每個 change SHALL 給出它的 `schema`。對其他 schema 的 change，不套用 `plan.md` 的階段與 task 規則，改以其 `tasks.md` 的核取方塊作為步驟，給出已勾選與全部的數量，不列出任何 task，且完成的 task 為 0 之 0；其階段 SHALL 在沒有 `tasks.md` 或沒有核取方塊時為 `design`，沒有任何一個被勾選時為 `ready`，全部勾選時為 `done`，其餘為 `implementing`。

#### Scenario: Change of another schema shown
- **WHEN** `add-x` 有 `schema: spec-driven`、含三個核取方塊（其中一個已勾選）的 `tasks.md`，且沒有 `plan.md`
- **THEN** `dkt status --json` 給 `add-x` schema `spec-driven`、階段 `implementing`、3 步驟之 1、空的 task 清單與 0 之 0 個 task

#### Scenario: No tasks file
- **WHEN** `add-x` 有 `schema: spec-driven`、一份 `proposal.md`，沒有 `tasks.md`
- **THEN** 其階段是 `design`，且有 0 之 0 個步驟

#### Scenario: Docket change
- **WHEN** `add-y` 是由 `dkt new` 建立的
- **THEN** `dkt status --json` 給 `add-y` schema `docket`，其階段與 task 來自它的 `plan.md`
