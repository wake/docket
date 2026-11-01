# conflict-detection Specification

## Purpose
定義 Docket 如何找出動到同一個需求的進行中 change，讓在不同 worktree 平行進行的工作，
能在還可以避免的時候就得知重疊，遠早於封存時一個 change 覆蓋另一個。

## Requirements

### Requirement: Overlapping changes
當兩個或更多不同的進行中 change 動到同一個 capability 路徑與需求名稱時，`dkt conflicts` SHALL 將該需求回報為重疊。change 在其 delta 修改、移除、將需求從該名稱更名或更名為該名稱、或新增同名需求時，即算動到該需求。

#### Scenario: Both modify one requirement
- **WHEN** `add-x` 與 `add-y` 各有一個針對 `booking` / "Cancel booking" 的 MODIFIED 區塊
- **THEN** 指令回報 `booking` / "Cancel booking" 上的一個重疊，並指出兩個 change

#### Scenario: Rename against modify
- **WHEN** `add-x` 把 `booking` / "Cancel booking" 更名為 "Cancel a booking"，而 `add-y` 修改 "Cancel booking"
- **THEN** 指令回報一個重疊，指出兩個 change 與它們的操作

#### Scenario: Same capability, different requirements
- **WHEN** `add-x` 修改 `booking` / "Cancel booking"，而 `add-y` 新增 `booking` / "Refund points"
- **THEN** 指令不回報重疊

#### Scenario: Both add the same name
- **WHEN** `add-x` 與 `add-y` 各新增 `booking` / "Waitlist cap"
- **THEN** 指令回報一個重疊，指出兩個 change

### Requirement: Where changes are found
`dkt conflicts` SHALL 考慮 `workspace-scan` 所定義的掃描從其所有 source 找到的每個進行中的 change，且 SHALL NOT 考慮已封存的 change。change SHALL 只與其他 change 比較，絕不與它在其他 source 中的副本比較。

#### Scenario: Overlap with another worktree
- **WHEN** worktree A 持有 `add-x`，worktree B 持有未提交的 `add-y`，兩者修改同一個需求，且指令在 worktree A 執行
- **THEN** 回報該重疊，`add-y` 的 source 為 worktree B

#### Scenario: Archived change left on a branch
- **WHEN** `add-x` 已封存在預設分支的 tip，某個未合併的分支仍持有 `openspec/changes/add-x/`，而 `add-y` 修改了該副本所修改的需求
- **THEN** 指令不回報重疊

### Requirement: One change in focus
指定 slug 時，`dkt conflicts` SHALL 只回報牽涉該 change 的重疊。該 slug SHALL 是進行中的 change。

#### Scenario: Unrelated overlap hidden
- **WHEN** `add-x` 與 `add-y` 重疊，`add-z` 與任何東西都不重疊，且使用者執行 `dkt conflicts add-z`
- **THEN** 指令不回報重疊並以 0 結束

#### Scenario: Slug not active
- **WHEN** 使用者執行 `dkt conflicts add-q`，而沒有名為 `add-q` 的進行中 change
- **THEN** 指令回報 `add-q` 不在進行中，並以 1 結束

### Requirement: Overlap report
每個重疊 SHALL 指出 capability 與需求，並對牽涉的每個 change，指出它的 slug、它對該需求套用的每個操作，以及找到它的每個 source 與該 source 的 commit。指令回報任何重疊時 SHALL 以 1 結束，沒有重疊時 SHALL 以 0 結束，但無法讀取的 source 依 `cli-output` 的要求處理。它 SHALL NOT 要求 OpenSpec CLI。

#### Scenario: Sources listed
- **WHEN** `add-y` 同時存在於 worktree B 與 `refs/remotes/origin/add-y`
- **THEN** 重疊中它的條目列出兩個 source 與其 commit

#### Scenario: No overlaps
- **WHEN** 沒有任何兩個進行中的 change 動到同一個需求
- **THEN** 指令不回報重疊並以 0 結束

#### Scenario: Unreadable worktree
- **WHEN** 列出的某個 worktree 無法讀取，而另外兩個 change 重疊
- **THEN** 回報該重疊，該 worktree 被列為無法讀取，且指令以 3 結束
