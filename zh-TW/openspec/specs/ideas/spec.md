# ideas Specification

## Purpose
定義 idea（變成 change 之前還需要設計的工作）與 `docket/order.md`（取用它們的順序）：兩者的檔案格式、
`dkt lint` 如何檢查、idea 的狀態如何由指名它的工作推得、`dkt status` 如何回報 order 與 idea，
以及 `docket` skill 如何使用它們。

## Requirements

### Requirement: Idea file
idea SHALL 是一個一般檔案 `docket/ideas/<name>.md`：YAML front matter 含符合 `docket-metadata` 所定義形式的 `title`，可選的 `dropped: true`，以及可選的 `needs`（它所依賴的 idea 的儲存庫根目錄路徑清單）；其後是自由文字。

#### Scenario: Idea file read
- **WHEN** `docket/ideas/2026-11-02-x.md` 的 front matter 有 `title: X` 與 `needs: [docket/ideas/2026-11-01-y.md]`，`docket/ideas/2026-11-01-y.md` 有 `title: Y`，且使用者執行 `dkt lint`
- **THEN** lint 對這兩個 idea 都不回報 finding

### Requirement: Order file
`docket/order.md` SHALL 只存放即將進行的工作的順序，別無其他：標題 `# Order`，其後是形式為 ``N. `<path>` `` 的行，從 1 起編號、不得跳號，每行以儲存庫根目錄路徑指名一個 idea。行與行之間 MAY 有空行。idea 完成後，該行仍保留。

#### Scenario: Order file read
- **WHEN** `docket/order.md` 含 `# Order`、一個空行、``1. `docket/ideas/2026-11-01-a.md` `` 與 ``2. `docket/ideas/2026-11-02-b.md` ``，兩個 idea 都存在，且使用者執行 `dkt lint`
- **THEN** lint 對 order 不回報 finding

### Requirement: Order checked
`dkt lint` SHALL 在以下情況回報 `order-format`：`docket/order.md` 不是一般檔案、第一個非空行不是 `# Order`、某行既非空行也不符合行的形式、或各行編號不是 1、2、3 依序遞增；並 SHALL 對以下的行回報 `order-idea`：路徑在 worktree 中指不到一般 idea 檔案、指名了較前面某行已指名的 idea、或指名了帶 `dropped: true` 的 idea。每個 finding SHALL 指出它所在的行。

#### Scenario: Order line with a checkbox
- **WHEN** `docket/order.md` 的第 3 行是 ``3. [ ] `docket/ideas/2026-11-02-c.md` ``
- **THEN** `dkt lint` 對該行回報 `order-format`，並以 1 結束

#### Scenario: Gap in the order
- **WHEN** `docket/order.md` 各行編號為 1、2、4
- **THEN** `dkt lint` 對編號 4 的行回報 `order-format`

#### Scenario: Order names a missing idea
- **WHEN** `docket/order.md` 的某行指名 `docket/ideas/2026-11-02-none.md`，而 worktree 沒有這個檔案
- **THEN** `dkt lint` 對該行回報 `order-idea`

#### Scenario: Same idea twice
- **WHEN** `docket/order.md` 的第 2 行與第 5 行指名同一個 idea
- **THEN** `dkt lint` 對第 5 行回報 `order-idea`，對第 2 行不回報

#### Scenario: Dropped idea in the order
- **WHEN** `docket/order.md` 的某行指名一個 front matter 帶 `dropped: true` 的 idea
- **THEN** `dkt lint` 對該行回報 `order-idea`

#### Scenario: Empty order file
- **WHEN** `docket/order.md` 為空，或只有空行
- **THEN** `dkt lint` 對第 1 行回報 `order-format`，並以 1 結束

#### Scenario: Linked order file
- **WHEN** `docket/order.md` 是指向儲存庫外某檔案的符號連結
- **THEN** `dkt lint` 回報 `order-format` 並指出該連結，且不透過連結讀取

### Requirement: Idea front matter checked
`dkt lint` SHALL 對以下的 idea 回報 `idea-front-matter`：front matter 缺少或不是 YAML mapping、缺少有效的 `title`、`dropped` 的值不是 `true`、或 `needs` 不是路徑清單，或其中的路徑並未各自指到 worktree 中另一個一般 idea 檔案。`docket/ideas/` 底下不是一般檔案的項目 SHALL 略過。當 `docket/ideas` 本身是符號連結時，lint SHALL 將它列為無法讀取，且不透過它讀取任何 idea。

#### Scenario: Linked ideas folder
- **WHEN** `docket/ideas` 是指向儲存庫內某個含 `b.md` 的資料夾的符號連結，且 `docket/order.md` 的某行指名 `docket/ideas/b.md`
- **THEN** `dkt lint` 將 `docket/ideas` 列為無法讀取、對該行回報 `order-idea`，並以 3 結束

#### Scenario: Idea without a title
- **WHEN** `docket/ideas/2026-11-02-x.md` 的 front matter 沒有 `title`
- **THEN** `dkt lint` 對它回報 `idea-front-matter`，並以 1 結束

#### Scenario: Need names no idea
- **WHEN** 某個 idea 有 `needs: [docket/ideas/2026-11-02-none.md]`，而 worktree 沒有這個檔案
- **THEN** `dkt lint` 回報 `idea-front-matter` 並指出該路徑

### Requirement: Idea sources checked
`dkt lint` SHALL 對它所檢查的每個 change，以及分支新增的每個 record，只要其 source 依 `docket-metadata` 的定義指名 `docket/ideas/` 底下的路徑，而該路徑在 worktree 中不是一般 idea 檔案，就回報 `idea-source` 並指出該 source。

#### Scenario: Source names no idea
- **WHEN** change `add-x` 的 source 是 `docket/ideas/2026-11-02-none.md, item 4`，而 worktree 沒有這個檔案
- **THEN** `dkt lint add-x` 回報 `idea-source` 並指出該 source，並以 1 結束

### Requirement: Idea state
idea 的 handler SHALL 是 source 依 `docket-metadata` 的定義指名該 idea 路徑的 record 與 change，其尋找與讀取方式與 issue 的 handler 相同。其狀態 SHALL 在 `dropped: true` 時為 `dropped`；其餘情況：沒有 handler 時為 `pending`，預設分支的 tip 並未持有某個 handler 時為 `in-progress`，tip 持有所有 handler 時為 `done`。idea SHALL 只從預設分支的 tip 讀取。

#### Scenario: Idea pending
- **WHEN** 預設分支的 tip 持有 `docket/ideas/2026-11-02-x.md`，且沒有任何 record 或 change 指名它
- **THEN** 該 idea 的狀態是 `pending`

#### Scenario: Handler on a branch
- **WHEN** 預設分支的 tip 持有該 idea，而某個未合併的分支持有 change `add-x`，其 `docket.yaml` 以該 idea 的路徑為 source
- **THEN** 該 idea 的狀態是 `in-progress`，`add-x` 是 tip 並未持有的 handler

#### Scenario: Every handler merged
- **WHEN** 預設分支的 tip 持有該 idea、一個 source 為其路徑的 record，以及一個 source 為其路徑後接 `, second part` 的已封存 change
- **THEN** 該 idea 的狀態是 `done`

#### Scenario: One handler of two merged
- **WHEN** 預設分支的 tip 持有該 idea 與一個指名它的 record，而 worktree B 持有一個指名它的未提交 change
- **THEN** 該 idea 的狀態是 `in-progress`

#### Scenario: Dropped idea
- **WHEN** 該 idea 有 `dropped: true`，且預設分支的 tip 持有一個指名它的 record
- **THEN** 該 idea 的狀態是 `dropped`

#### Scenario: Idea on a branch only
- **WHEN** 某個未合併的分支新增了 `docket/ideas/2026-11-02-z.md`，而預設分支的 tip 沒有這個檔案
- **THEN** `dkt status` 不回報 idea `docket/ideas/2026-11-02-z.md`

### Requirement: Order listed by status
`dkt status` SHALL 在 `result.order` 中依序列出預設分支 tip 的 `docket/order.md` 中每一個符合行形式的行，附上其編號、路徑、其 idea 的標題（無法讀取時為 null）、狀態（tip 不持有該 idea 時為 `missing`）、handler，以及 `waiting_on`：該 idea 的 `needs` 中，其 idea 尚未 `done` 的路徑。`result.next` SHALL 是第一個狀態為 `pending` 或 `in-progress` 的行的編號，否則為 null。

#### Scenario: Order listed
- **WHEN** tip 的 order 第 1 行指名一個 `done` 的 idea，第 2 行指名一個被分支上某 change 指名的 idea，且使用者執行 `dkt status --json`
- **THEN** `result.order` 將第 1 行列為 `done`，第 2 行列為 `in-progress` 並附上該 change，且 `result.next` 是 2

#### Scenario: Next skips done, dropped and missing
- **WHEN** tip 的 order 第 1 行是 `done`，第 2 行指名一個 dropped 的 idea，第 3 行指名 tip 不持有的 idea，第 4 行是 `pending`
- **THEN** 第 3 行的狀態是 `missing`、標題為 null，且 `result.next` 是 4

#### Scenario: Waiting on a need
- **WHEN** tip 的 order 第 1 行指名的 idea，其 `needs` 列出 `docket/ideas/2026-11-01-b.md`，而該 idea 是 `pending`
- **THEN** 第 1 行的 `waiting_on` 是 `["docket/ideas/2026-11-01-b.md"]`

#### Scenario: No order file
- **WHEN** 預設分支的 tip 沒有 `docket/order.md`，且使用者執行 `dkt status --json`
- **THEN** `result.order` 為空、`result.next` 為 null，且不對 order 回報 finding

#### Scenario: Unparsed order line
- **WHEN** tip 的 `docket/order.md` 含 ``1. `docket/ideas/2026-11-01-a.md` `` 與一行 `- later`
- **THEN** `result.order` 只列出第 1 行

### Requirement: Ideas outside the order listed by status
`dkt status` SHALL 在 `result.ideas` 中依路徑排序，列出預設分支 tip 持有、order 的任何一行都沒有指名、且狀態為 `pending` 或 `in-progress` 的每個 idea，各附其路徑、標題、狀態與 handler。指定 slug 時，它 SHALL 不列出任何 idea 與 order 行，且 `result.next` SHALL 為 null。

#### Scenario: Ideas outside the order
- **WHEN** tip 持有 idea `a`（被 order 指名）、`b`（pending）、`c`（done）與 `d`（dropped），且使用者執行 `dkt status --json`
- **THEN** `result.ideas` 只列出 `b`

#### Scenario: Order not listed with slugs
- **WHEN** tip 的 order 有一個 pending 的行，且使用者執行 `dkt status add-x --json`
- **THEN** `result.order` 與 `result.ideas` 為空，且 `result.next` 為 null

### Requirement: Order and ideas in the text output
`dkt status` 的文字輸出 SHALL 在 issue 之後，為每個不是 `done` 的 order 行輸出一行，含其編號、狀態、路徑與標題；為 `result.ideas` 中每個 `in-progress` 的 idea 輸出一行；並在 `result.next` 不為 null 時，輸出一行指出它。

#### Scenario: Order in the text output
- **WHEN** tip 的 order 第 1 行是 `done`、第 2 行是 `in-progress`，且使用者執行 `dkt status`
- **THEN** 輸出有一行對應第 2 行、沒有對應第 1 行的行，並有一行指出下一個是 2

### Requirement: Ideas and the order have one writer
`docket` skill SHALL 讓 agent 在 change 或 record 的 worktree 中，不更動 `docket/order.md` 與 `docket/ideas/` 底下的檔案。工作中發現的 idea，以及對 order 的任何編輯，SHALL 各自以一次直接合併進行。

#### Scenario: Idea found during a change
- **WHEN** 正在做 change `add-x` 的 agent 發現還有需要設計的工作
- **THEN** 它將該 idea 以一次獨立的直接合併寫入，且 `add-x` 的分支不更動 `docket/ideas/` 底下的任何檔案

### Requirement: Work names its idea
當工作取用某個 idea 時，`docket` skill SHALL 讓 agent 只傳該 idea 的路徑，作為 change 或 record 的 `--source`。

#### Scenario: Change takes an idea
- **WHEN** agent 為 idea `docket/ideas/2026-11-02-x.md` 開啟 change
- **THEN** 它執行 `dkt new`，並帶 `--source docket/ideas/2026-11-02-x.md`

### Requirement: One idea, one piece of work
`docket` skill SHALL 讓 agent 將 idea 的後續部分（該 idea 的 change 或 record 留待日後的部分）寫成新的 idea，其 `needs` 指名第一個 idea。

#### Scenario: Idea with a further part
- **WHEN** `docket/ideas/2026-11-02-x.md` 的 change 把第二部分留待日後
- **THEN** agent 將該部分寫成新的 idea，帶 `needs: [docket/ideas/2026-11-02-x.md]`，並以一次獨立的直接合併進行

### Requirement: Next work from status
`docket` skill SHALL 讓 agent 從 `dkt status --json` 的 `result.next` 取得下一件工作，而不是讀 `docket/order.md` 的核取方塊或前幾行。

#### Scenario: Agent picks the next work
- **WHEN** 使用者要求下一件工作
- **THEN** agent 執行 `dkt status --json`，取 `result.next` 所指的那一行的 idea
