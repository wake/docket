# 架構

Docket（`dkt`）是個 Go CLI。撰寫 proposal 或 plan 之前先讀這份。

## 程式碼地圖

| 路徑 | 角色 |
|---|---|
| `schema.go` | 根套件 `docket`：嵌入 OpenSpec schema（`schema/docket`）、`docket`、`docket-setup` 與 `docket-doctor` skill、使用者層級的 `docket-init` 命令（`user-skills/docket-init`）、`AGENTS.md` 的 Docket 區塊、流程檔範本（`workflow/template.md`）與 record 範本（`record/template.md`） |
| `cmd/dkt` | 進入點；唯一讀取行程環境與工作目錄的地方 |
| `internal/cli` | 命令表、參數處理、文字與 JSON 輸出、結束碼 |
| `internal/cli/dated.go` | `writeDated`，`dkt record` 與 `dkt issue` 共用的流程：挑一個可用的 slug 並建立檔案 |
| `internal/gitx` | 所有 git 呼叫：ref、worktree、merge-base、某個 revision 的檔案內容 |
| `internal/openspec` | 找到 OpenSpec CLI、檢查版本並執行它 |
| `internal/scan` | 在各 worktree 與分支間找出 change；解析預設分支 |
| `internal/setup` | 每台機器安裝一次 schema，並安裝 skill、使用者層級的 `docket-init` 命令與 Docket 區塊（`dkt init`）、移除專案內出貨 schema 的副本，以及在不寫入的情況下判斷它會做什麼（`dkt inspect`） |
| `internal/change` | change 資料夾的規則：slug 驗證、`docket.yaml` 的中繼資料、plan 步驟與勾選 |
| `internal/archive` | 結束 change：`dkt sync`（plan 與 record）、`dkt archive`、`dkt rebase`、base 檢查、Docket 的讀法與 OpenSpec 的交叉核對、`dkt inspect` 的 trailer 檢查，以及 `evidence.md` |
| `internal/markdown` | 逐行、標題與表格的讀取器，會跳過程式碼區塊與註解 |
| `internal/delta` | 讀取 delta spec 與主 spec；產生情境的 slug |
| `internal/status` | 摘要每個進行中的 change 與每個 record：階段、由 commit trailer 得出的 task 狀態、進度、最後活動 |
| `internal/conflicts` | 找出被多個進行中的 change 動到的需求 |
| `internal/record` | Record：kind、名稱與 slug、範本渲染、以可用的名稱建立檔案、解析與字數統計 |
| `internal/idea` | Idea 與 order：front matter、idea 路徑、order 檔案的各行及其問題 |
| `internal/issue` | Issue：名稱與 slug、範本渲染、解析會隱藏 issue 的 front matter |
| `internal/lint` | 透過 `fs.FS` 對 change 資料夾執行 lint 規則 |
| `internal/userdirs` | 由 `cmd/dkt` 傳入的值找出家目錄、資料目錄（`XDG_DATA_HOME`）與 Claude Code 設定目錄（`CLAUDE_CONFIG_DIR`） |
| `internal/safepath` | 拒絕經過符號連結的寫入路徑 |
| `internal/workflow` | 找到使用者的流程檔，並對照出貨的範本回報其狀態；從不寫入 |
| `internal/testutil` | 僅供測試的輔助工具：暫時的 repository、假的 OpenSpec |

## Capability 索引

| 工作 | 使用 |
|---|---|
| 執行 git | `gitx.Git.Run`（及其有型別的輔助函式，例如 `Git.Show`、`Git.Worktrees`） |
| 判斷忽略規則涵蓋哪些路徑 | `gitx.Git.Ignored` |
| 在各處找出 change | `scan.Scan`，然後 `Inventory.Uses` |
| 解析預設分支 | `scan.ResolveDefaultBranch` |
| 從掃描到的 source 讀取檔案 | `scan.ReadFile` |
| 啟動 OpenSpec | `openspec.Runner.Run`（以 `openspec.New` 建立） |
| 驗證 slug | `change.ValidateSlug` |
| 記錄 title、kind、source 與 base | `change.CreateDocket` |
| 安裝 schema | `setup.Init` |
| 找出家目錄 | `userdirs.Home` |
| 找出資料目錄 | `userdirs.DataHome` |
| 找出每台機器的 schema 目錄 | `setup.SchemaDir` |
| 找出 `docket-init` 的目錄 | `setup.UserSkillDirs` |
| 找出 Claude Code 設定目錄 | `userdirs.ClaudeConfig` |
| 以 OpenSpec 的方式找出 schema | `setup.FindSchema` |
| 對 schema tree 或目錄計算雜湊 | `setup.TreeHash`、`setup.HashDir` |
| 判斷某個 tree 是否為出貨的 schema | `setup.IsShipped` |
| 讀取或設定 `openspec/config.yaml` 的 `schema:` | `setup.SchemaKey` |
| 寫入前檢查路徑 | `safepath.Check` |
| 讀取 Markdown 的行、標題與表格 | `markdown.Lines`、`markdown.Headings`、`markdown.FindTable` |
| 清空程式碼區塊以外的 HTML 註解 | `markdown.BlankComments` |
| 讀取 delta 或主 spec | `delta.Parse`、`delta.ParseSpec` |
| 產生情境的 slug | `delta.Slug` |
| 讀取 proposal.md 或 plan.md | `change.ParseProposal`、`change.ParsePlan` |
| 對 change 資料夾做 lint | `lint.Run` |
| 找出沒有標題的主 spec | `lint.SpecTitles` |
| 找出屬於其他 schema 的 change | `lint.OtherSchemas` |
| 讀取 change 的中繼資料（`docket.yaml`，或較舊 change 的 `.openspec.yaml`） | `change.ReadMetadata`、`change.ParseDocket` |
| 設定一個已記錄的鍵，例如 base | `change.SetKey` |
| 勾選 plan 步驟 | `change.Tick` |
| 讀取 change 的 schema | `change.ReadSchema` |
| 計算檔案中已勾選與全部的核取方塊 | `change.Checkboxes` |
| 列出兩個 revision 之間的 commit | `Git.Commits` |
| 列出未 commit 的路徑 | `Git.Uncommitted` |
| 若檔案存在，讀取某個 revision 的檔案 | `Git.ShowIfExists` |
| 依 commit trailer 勾選 plan | `archive.Sync` |
| 移動 change 記錄的 base | `archive.Rebase` |
| 檢查並封存 change，然後記錄 evidence | `archive.Archive` |
| 比對 Docket 與 OpenSpec 對 delta 的讀法 | `archive.CrossCheck` |
| 分支新增的檔案 | `Git.AddedFiles`、`Git.UncommittedAdded` |
| 列出某個 revision 下某目錄內的檔案 | `Git.ListFiles` |
| 從掃描到的 source 讀取多個檔案 | `scan.ReadFiles` |
| 將 delta spec 路徑對應到它的 capability | `delta.CapabilityPath` |
| 載入每個進行中 change 的 delta | `conflicts.Load` |
| 找出被多個 change 動到的需求 | `conflicts.Find` |
| 保留涉及某個 change 的重疊 | `conflicts.Involving` |
| 載入每個進行中 change 的狀態 | `status.Load` |
| 算出 change 的階段、task 與進度 | `status.Summarize` |
| 由 `tasks.md` 得出其他 schema 的 change 的階段與進度 | `status.SummarizeChecklist` |
| 依 trailer 將 commit 對應到 plan 的 task | `change.NamedTasks` |
| 列出指定路徑下未 commit 的路徑 | `Git.UncommittedUnder` |
| 列出 commit 與其時間，限定於某些路徑 | `Git.Commits`（`Commit.Time`、`paths`） |
| 找到使用者的流程檔 | `workflow.Locate` |
| 讀取流程檔 | `workflow.Read` |
| 回報流程檔的狀態 | `workflow.Inspect` |
| 在不寫入的情況下判斷 init 會安裝什麼 | `setup.Inspect` |
| 檢查預設分支上 change 的 trailer | `archive.Trailers` |
| 對範本計算雜湊 | `workflow.Hash` |
| 挑選可用的 slug | `change.Suffixed`，搭配 `scan.Inventory.Uses` 或 `RecordSources` |
| record slug 用在何處 | `scan.Inventory.RecordSources` |
| 計算 record 區段的字數 | `record.Words` |
| 由指名它們的 record 與 change 得出 issue 與 idea 的狀態、order 與下一項工作 | `status.LoadBacklog` |
| 一次找出指名任一組路徑的 record 與 change | `status.findHandlers` |
| source 所指名的路徑（它的第一個欄位） | `change.SourcePath` |
| 讀取 idea 的 front matter；判斷是否為 idea 路徑；讀取 order 檔案 | `idea.Parse`；`idea.IsIdeaPath`；`idea.ParseOrder` |
| 以 source 表示的預設分支 | `scan.DefaultBranch.Source` |
| issue slug 用在何處，以及其 handler 的 source | `scan.Inventory.Issues` |
| 由範本渲染檔案 issue；讀取它的 front matter | `issue.Render`；`issue.ParseFrontMatter` |
| 載入 record 及其狀態、merge commit 與 tag | `status.LoadRecords` |
| 為 change 或 record 將 trailer 對應到 task | `change.MatchTasks` |
| HEAD 上帶有 record trailer 的 commit | `record.CommitsFor` |
| 每個 first-parent commit 在某目錄下新增的檔案 | `Git.Additions` |
| 列出 tag；包含各 record 的第一個 tag | `Git.Tags`；`scan.RecordTags` |
| 依 commit trailer 勾選 record 的 task | `archive.SyncRecord` |
| 檢查任何檔案中的 task 標題 | `lint.checkTaskHeadings` |
| 回報結果 | `cli.Outcome`；以 `cli.ExitCode` 對應到結束碼 |
| 建立測試用 repository | `testutil.NewRepo` |
| 代替 OpenSpec | `testutil.FakeOpenSpec` |

## 不變式

- 只有 `internal/openspec` 啟動 OpenSpec。
- 只有 `internal/gitx` 啟動 git（測試輔助工具除外）。
- `cmd/dkt` 以下的程式碼都不讀取行程環境；環境透過 `cli.Options` 傳入，時鐘（`Options.Now`）與使用者查詢（`Options.UserHome`）也包含在內（測試輔助工具除外）。
- 讀取器接收 bytes 而不是路徑，讓內容可以來自任何 revision。
- 任何東西都不透過 repository 內的符號連結寫入：先用 `safepath.Check` 檢查路徑。
