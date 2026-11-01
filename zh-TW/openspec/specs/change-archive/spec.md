# change-archive Specification

## Purpose
定義 `dkt archive` 在讓 OpenSpec 把 change 合併進現行 spec 之前所做的檢查、它留下的佐證，
以及 `dkt rebase` 如何把較新的 spec 接受為 change 的起點。

## Requirements

### Requirement: Preconditions
`dkt archive` SHALL 以結束碼 1 拒絕且不更動任何東西，除非：工作樹沒有未提交的變更、HEAD 包含預設分支的 tip、每個指名的 change 都存在於預設分支的 tip，且依 `openspec-runtime` 所述能找到 `docket` schema。找不到 schema 時，它 SHALL 回報 `schema-missing`，告訴使用者執行 `dkt init`。

#### Scenario: Uncommitted changes
- **WHEN** 工作樹有未提交的編輯，且使用者執行 `dkt archive add-x`
- **THEN** 指令拒絕並以 1 結束

#### Scenario: Branch behind the default branch
- **WHEN** HEAD 不包含預設分支的 tip
- **THEN** `dkt archive add-x` 拒絕並以 1 結束

#### Scenario: Change not merged yet
- **WHEN** `add-x` 存在於 worktree，但不在預設分支的 tip
- **THEN** `dkt archive add-x` 拒絕並以 1 結束

#### Scenario: Schema not installed
- **WHEN** 專案的 `openspec/schemas/docket/` 與每台機器的目錄都沒有該 schema，且使用者執行 `dkt archive add-x`
- **THEN** 不執行任何 OpenSpec 指令、不更動任何東西，findings 含指名 `dkt init` 的 `schema-missing`，且指令以 1 結束

### Requirement: Steps in order
對每個指名的 change，依給定的順序，`dkt archive` SHALL 執行 plan 檢查、lint、base 檢查、`openspec validate <slug> --strict`、讀取交叉核對、`openspec archive <slug> -y --json`、spec title 步驟與佐證步驟。它 SHALL 在第一個失敗某個步驟的 change 停下並回報該步驟，同一次執行中較早封存的 change 仍保持封存在工作樹中。

#### Scenario: Second of two fails
- **WHEN** 使用者執行 `dkt archive add-x add-y`，`add-x` 通過每個步驟，而 `add-y` 的 lint 失敗
- **THEN** `add-x` 已封存在工作樹中、`add-y` 未被更動、回報 lint 的 finding，且指令以 1 結束

### Requirement: Plan check
plan 檢查 SHALL 在以下情況失敗：`dkt sync <slug> --check` 會以 1 結束、plan.md 中某個 `### Task` 章節有未勾選的步驟，或從 HEAD 可達、且不是從 change 的 base 可達的 commit 中，沒有任何一個同時帶有 `Change: <slug>` 與 `Task:` trailer。

#### Scenario: Unticked task
- **WHEN** `add-x` 的 Task 5 有一個未勾選的步驟，且沒有 commit 指名它
- **THEN** `dkt archive add-x` 在 plan 檢查停下並指出 Task 5

#### Scenario: Squashed pull request
- **WHEN** `add-x` 的每個步驟都已勾選，但自其 base 以來沒有 commit 帶有 `Change: add-x` 與 `Task:` trailer
- **THEN** `dkt archive add-x` 在 plan 檢查停下

### Requirement: Base check
對 change 所修改、移除或更名的每個需求，base 檢查 SHALL 比較該需求在 change 的 base 時主 spec 中的區塊，與它在工作樹中（含同一次執行中較早封存的 change）的區塊，兩者不同時 SHALL 失敗並顯示差異。對 change 所新增的每個需求，若工作樹的主 spec 中有同名需求而 base 時的主 spec 沒有，它 SHALL 失敗。

#### Scenario: Requirement changed since base
- **WHEN** `add-y` 修改 `booking` / "Cancel booking"，而在 `add-y` 的 base 之後封存的 `add-x` 更動了該需求
- **THEN** `dkt archive add-y` 在 base 檢查停下，並顯示該需求在 base 時與現在的文字

#### Scenario: Two changes in one run touch one requirement
- **WHEN** `add-x` 與 `add-y` 共用同一個 base，兩者都修改 `booking` / "Cancel booking"，且使用者執行 `dkt archive add-x add-y`
- **THEN** `add-x` 已封存，`add-y` 在 base 檢查停下

#### Scenario: Other requirement in the same capability
- **WHEN** `add-y` 修改 `booking` / "Cancel booking"，而自其 base 以來只有 `booking` / "Refund points" 有變動
- **THEN** base 檢查通過

#### Scenario: Added name now taken
- **WHEN** `add-y` 新增 `booking` / "Waitlist cap"，而該需求在 `add-y` 的 base 之後才出現在主 spec
- **THEN** `dkt archive add-y` 在 base 檢查停下

### Requirement: Reading cross-check
執行 `openspec archive` 之前，`dkt archive` SHALL 將 Docket 對 change 的 delta 的讀取結果，與 `openspec show <slug> --type change --json --deltas-only` 的輸出比較，並在 capability、操作、需求名稱或情境名稱不同時失敗。

#### Scenario: Readings disagree
- **WHEN** Docket 讀到 `add-x` 中有一個需求，而 OpenSpec 的輸出沒有列出
- **THEN** `dkt archive add-x` 在 `openspec archive` 之前停下，並回報兩邊的讀取結果

### Requirement: Evidence
spec title 步驟之後，`dkt archive` SHALL 在已封存的 change 資料夾中寫入 `evidence.md`，記錄日期、Docket 與 OpenSpec 的版本、`origin` remote 的 URL（沒有則記為 none）、HEAD 的 SHA、change 的 title（沒有則記為 none）、它的 kind、source 與 base、它的 schema、為它略過的步驟、每個 task 的 trailer 指名它的 commit，以及 validate 與 base 檢查步驟的結果。

#### Scenario: Evidence written
- **WHEN** `dkt archive add-x` 在 2026-11-02 成功，且 `add-x` 的 title 是 "Booking waitlist"
- **THEN** `openspec/changes/archive/2026-11-02-add-x/evidence.md` 存在、給出 title "Booking waitlist"，並列出每個 task 的 commit

### Requirement: No commit
`dkt archive` SHALL NOT commit、push 或建立分支。它的結果 SHALL 留在工作樹中，由使用者自行 commit。

#### Scenario: Nothing committed
- **WHEN** `dkt archive add-x` 成功
- **THEN** HEAD 與之前是同一個 commit，且 `git status` 將封存的變更顯示為未提交

### Requirement: Rebase
`dkt rebase <slug>` SHALL 把 change 的 base 設為 HEAD 與預設分支 tip 的 merge base 的完整 SHA，寫在 `docket-metadata` 存放它的地方，不更動其他行，並 SHALL 列出 change 所涉及、且其主 spec 區塊在舊 base 與新 base 之間不同的每個需求。它 SHALL NOT 要求 OpenSpec CLI。

#### Scenario: Rebase after merging the default branch
- **WHEN** 預設分支已合併進 change 的分支，且其已封存的 change 修改了 `add-y` 所涉及的某個需求
- **THEN** `dkt rebase add-y` 把 `openspec/changes/add-y/docket.yaml` 中的 `base` 移到新的 merge base，並列出該需求

#### Scenario: Rebase a change made before docket.yaml
- **WHEN** `add-y` 沒有 `docket.yaml`，而其 `.openspec.yaml` 含有 `x-base`
- **THEN** `dkt rebase add-y` 把 `x-base` 移到新的 merge base，且不建立 `docket.yaml`

#### Scenario: Nothing to move
- **WHEN** HEAD 與預設分支 tip 的 merge base 等於 change 目前的 base
- **THEN** `dkt rebase add-y` 不更動任何東西並以 0 結束

### Requirement: Spec titles carried
`openspec archive` 成功後，對每個其 delta 旁有 `docket.yaml` 的 capability，spec title 步驟 SHALL 把 `openspec/specs/<capability>/docket.yaml` 的 `title` 設為該 delta 的 title，必要時建立該檔案，且不更動其他行。無法讀取的檔案 SHALL 列為無法讀取並以 3 結束；任何其他失敗 SHALL 對該 capability 回報 `spec-title-failed` 並以 1 結束。兩種情況執行都會停下。

#### Scenario: Title carried to a new spec
- **WHEN** `add-x` 新增 capability `booking`，`openspec/changes/add-x/specs/booking/docket.yaml` 含 `title: Booking`，且使用者執行 `dkt archive add-x`
- **THEN** `openspec/specs/booking/docket.yaml` 含 `title: Booking`

#### Scenario: Title replaced on an existing spec
- **WHEN** `openspec/specs/booking/docket.yaml` 含 `title: Bookings` 與另一行，而 `add-x` 對 `booking` 的 delta 有含 `title: Booking` 的 `docket.yaml`
- **THEN** `dkt archive add-x` 之後該檔案含 `title: Booking`，其另一行不變

### Requirement: Schema reported on archive
`dkt archive` SHALL 在 `result.schema` 中回報它找到 schema 的目錄（絕對路徑），以及它是否與附帶的 schema 逐位元組相同。不同的副本 SHALL 被回報，且 SHALL NOT 使封存停下。

#### Scenario: Archive with a differing schema
- **WHEN** 每台機器的 `schema.yaml` 與附帶的檔案不同，且 `add-x` 通過每個步驟
- **THEN** `add-x` 已封存、`result.schema.matches` 為 false、輸出說明副本不同，且指令以 0 結束

### Requirement: Changes of another schema archived
對其他 schema 的 change，`dkt archive` SHALL 略過 plan 檢查與 base 檢查，並依序執行其他每個步驟，其 lint 步驟套用 lint 對這類 change 所用的規則。它 SHALL NOT 要求這類 change 有 `plan.md`、`docket.yaml` 或 commit trailer。

#### Scenario: Change of another schema archived
- **WHEN** `add-x` 有 `schema: spec-driven`、一份 `tasks.md`，沒有 `plan.md` 或 `docket.yaml`，位於預設分支的 tip，且使用者執行 `dkt archive add-x`
- **THEN** `add-x` 經由 `openspec archive` 封存，其 `evidence.md` 指出 schema `spec-driven` 並說明略過了 plan 與 base 檢查，且指令以 0 結束

#### Scenario: New capability without a title stops the archive
- **WHEN** 同一個 `add-x` 新增 capability `waitlist`，其 delta 旁沒有 `docket.yaml`
- **THEN** `dkt archive add-x` 在 lint 步驟以 `waitlist` 的 `title-missing` 停下，且早於任何會寫入的 OpenSpec 指令

### Requirement: Rebase of a change of another schema
對其他 schema 的 change，`dkt rebase <slug>` SHALL 回報 `other-schema`，指出該 change 的 `.openspec.yaml` 與其 schema，不更動任何東西，並以 1 結束。

#### Scenario: Rebase refused for another schema
- **WHEN** `add-x` 有 `schema: spec-driven`，且使用者執行 `dkt rebase add-x`
- **THEN** findings 含指名 `openspec/changes/add-x/.openspec.yaml` 的 `other-schema`、沒有檔案被更動，且指令以 1 結束
