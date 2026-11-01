# plan-sync Specification

## Purpose
定義 `dkt sync` 如何依 commit trailer 勾選 plan.md，讓 plan.md 能成為唯一的 task 清單，
即使 superpowers 把進度記在別處、從不勾選 plan。

## Requirements

### Requirement: Ticks from trailers
對不以日期開頭的 slug，`dkt sync <slug>` SHALL 收集從 HEAD 可達、且不是從 change 的 base 可達、trailer 含 `Change: <slug>` 的 commit，取出這些 commit 的每個 `Task: N` trailer，並對取得的每個 N，勾選 plan.md 中 `### Task N` 章節裡每個未勾選的 `- [ ]` 步驟。trailer SHALL 以 git 讀取的方式讀取。以日期開頭的 slug 指名的是 record，由 `records` 涵蓋。

#### Scenario: Task with commits ticked
- **WHEN** 兩個 commit 帶有 `Change: add-x` 與 `Task: 3`，而 Task 3 有四個未勾選的步驟
- **THEN** `dkt sync add-x` 之後 Task 3 的四個步驟都已勾選

#### Scenario: Task without commits left alone
- **WHEN** 沒有任何 commit 對 `add-x` 帶有 `Task: 4`
- **THEN** `dkt sync add-x` 之後 Task 4 的步驟維持原樣

#### Scenario: Two tasks in one commit
- **WHEN** 一個 commit 帶有 `Change: add-x`、`Task: 1` 與 `Task: 2`
- **THEN** `dkt sync add-x` 勾選 Task 1 與 Task 2 兩者的步驟

#### Scenario: Commits counted from the base
- **WHEN** `add-x` 的 `docket.yaml` 記錄了 base，而一個帶有 `Change: add-x` 與 `Task: 1` 的 commit 可從該 base 到達
- **THEN** `dkt sync add-x` 不因該 commit 勾選 Task 1

### Requirement: Only checkboxes change
`dkt sync` SHALL 只更動 plan.md 的步驟核取方塊中的 `[ ]` 為 `[x]`；對於 `records` 所定義的指名 record 的 slug，則是該 record 檔案的步驟核取方塊。它 SHALL 絕不取消勾選步驟、絕不動其他檔案、絕不 commit。它 SHALL NOT 要求 OpenSpec CLI。

#### Scenario: Nothing else changes
- **WHEN** `dkt sync add-x` 勾選 plan.md 中的步驟
- **THEN** plan.md 與之前只在那些核取方塊字元上不同，且 `git status` 沒有顯示其他被修改的檔案

#### Scenario: Record file only
- **WHEN** `dkt sync 2026-11-01-fix-x` 勾選該 record 中的步驟
- **THEN** 該 record 與之前只在那些核取方塊字元上不同，且 `git status` 沒有顯示其他被修改的檔案

### Requirement: Trailer problems
`dkt sync` SHALL 將本 change 的 commit 上每個其 N 不是 plan.md 中某個 `### Task` 標題（含已移到 Cancelled Tasks 的 task）的 `Task: N` trailer，回報為 finding，並仍 SHALL 勾選它能勾選的 task。

#### Scenario: Trailer for an unknown task
- **WHEN** 某個 commit 帶有 `Change: add-x` 與 `Task: 9`，而 plan.md 沒有 Task 9
- **THEN** `dkt sync add-x` 回報 Task 9 與該 commit、勾選其他 task，並以 1 結束

### Requirement: Check mode
`dkt sync <slug> --check` SHALL 不寫入任何東西，並在有任何步驟會被勾選時 SHALL 列出那些 task 並以 1 結束。

#### Scenario: Check finds unticked work
- **WHEN** Task 2 有 commit 與未勾選的步驟，且使用者執行 `dkt sync add-x --check`
- **THEN** plan.md 不變、列出 Task 2，且指令以 1 結束

#### Scenario: Check on a synced plan
- **WHEN** `dkt sync add-x` 剛執行過，且使用者執行 `dkt sync add-x --check`
- **THEN** 指令以 0 結束

### Requirement: Sync of a change of another schema
對其他 schema 的 change，`dkt sync <slug>` SHALL 回報 `other-schema`，指出該 change 的 `.openspec.yaml` 與其 schema，不更動任何東西，並以 1 結束，不論有沒有 `--check`。

#### Scenario: Sync refused for another schema
- **WHEN** `add-x` 有 `schema: spec-driven`，且使用者執行 `dkt sync add-x`
- **THEN** findings 含指名 `openspec/changes/add-x/.openspec.yaml` 與 `spec-driven` 的 `other-schema`、沒有檔案被更動，且指令以 1 結束
