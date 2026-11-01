---
name: docket-doctor
description: >-
  檢查這個專案的 Docket 設定（`dkt init` 安裝的檔案、OpenSpec CLI、使用者的流程檔、
  遺失 commit trailer 的已合併 change，以及需要判斷的事項），然後協助使用者修正有問題的地方。
  只在使用者主動叫用時執行。
disable-model-invocation: true
---

# Docket doctor

檢查這個專案的 Docket 設定，列出每個問題，然後與使用者逐一修正。
沒有經使用者同意的事，一律不改。

## 1. Run `dkt inspect`

先執行 `dkt inspect --json`，其他都在它之後。它不寫入任何東西。它的 `result` 包含：

- `files`：`dkt init` 安裝的每個檔案，狀態為 `same`、`missing` 或 `differs`；
- `openspec`：OpenSpec CLI 的 `version`（無法啟動時為 null）與是否 `supported`；
- `workflow`：使用者的流程檔，如 `dkt workflow` 所回報；
- `default_branch`：檢查 trailer 的那個分支；找不到時為 null（例如沒有 commit 的 repository），
  此時 `changes` 為空；
- `changes`：在預設分支上、已檢查 commit trailer 的進行中 change，以及被略過的，附原因。

它的 `findings` 與 `unreadable` 是機械可查的問題。其中包括 `install-ignored`
（`dkt init` 安裝的路徑被 git 忽略）、`title-missing`（沒有 title 的主 spec）與
`other-schema`（用 `docket` 以外的 schema 建立的進行中 change）。它以結束碼 2 或 4 結束時，
表示什麼都沒檢查：告訴使用者它說了什麼，然後停下。

## 2. Check what needs judgment

流程檔存在時，讀 `result.workflow.path` 的檔案，然後檢查：

- 每個步驟需要什麼：對流程檔的每個步驟，這個環境是否具備它點名的東西，例如可供審查的其他模型、
  subagent，或 command、skill、plugin。必要審查的審查者缺少時，說明檔案 `## Reviews` 下的
  退路適用哪一級：不同模型的審查者，否則是新 context 中的同一個模型，否則是在 PR 中註明
  這項工作沒有獨立審查。
- 偏離：流程檔與你 context 裡已有的使用者指示（他們的全域 agent 指示、記憶、這個專案的
  `AGENTS.md` 與 `CLAUDE.md`）對同一步驟說法不同時，例如不同的審查者、模型或 effort。
  兩邊都引用。
- 與框架的衝突：你 context 裡 Docket 的規則不允許的指示，例如 squash 合併
  （`dkt sync` 讀每個 commit 的 trailer），或在分流不會要求的地方開 change 或發版的規則。
- superpowers：它的 skill（brainstorming、writing-plans、subagent-driven-development）
  是否可用；全審路徑會用到它們。

除非使用者同意，不要讀取、列出或搜尋不在你 context 裡的使用者層級檔案，
例如全域指示檔或其他 agent 的設定。

## 3. List every problem

修正任何東西之前，列出步驟 1 與 2 的每個問題，各附它是什麼、為什麼重要，以及你建議的修正。
沒有問題時，說明沒有，然後停下。

## 4. Fix them one at a time

依列出的順序處理問題。每一個先說你要做什麼，使用者同意後才做；他們拒絕時，換下一個。

- 流程檔（`workflow-missing`、`workflow-incomplete`、`workflow-stale`，或這個環境做不到的步驟）：
  執行 `docket-setup` skill。`result.workflow.path` 為 null 時，表示找不到存放檔案的家目錄：
  就說明這點；`HOME` 設好之前，設定無法執行。
- 偏離：問使用者要保留哪一邊。流程檔透過 `docket-setup` 變更；對於他們自己的指示，
  提出建議的措辭。
- 已安裝的檔案（`schema-missing`、`skill-missing`、`agents-block-missing`、`schema-differs`、
  `skill-differs`、`agents-block-differs`、`user-skill-missing`、`user-skill-differs`、
  `schema-project-copy`、`config-schema-default`）：
  一起處理，因為一個指令就能全部修好。`dkt init` 會寫出每個缺少的檔案，在每台機器的
  副本與出貨版相符後移除出貨 schema 的專案副本，在 `openspec/config.yaml` 設定
  `schema: docket`，並且不動不同的檔案；沒有 `openspec` 目錄時，它先執行 `openspec init`。
  `dkt init --force` 也會取代每個不同的檔案。schema 的檔案與 `docket-init` 的副本是每台機器一份，
  以絕對路徑命名。先顯示每個不同的檔案與出貨版有何不同：在新的暫存目錄中執行 `git init`，
  建立空的 `openspec` 目錄，並在那裡執行 `dkt init`，將 `HOME`、`XDG_DATA_HOME` 與
  `CLAUDE_CONFIG_DIR` 設為新的暫存目錄，這樣你的機器上不會寫入任何東西；比較同一路徑的檔案，
  或那些目錄下每台機器的檔案。然後指名這個指令將寫入或移除的每個檔案，使用者同意後才執行。
- `schema-edited`：顯示專案的 `openspec/schemas/docket` 與出貨 schema 有何不同。說明 OpenSpec
  會用它而不是每台機器的那份，而且尚不支援 fork。移除它會失去那些編輯：使用者同意後才移除，
  並執行 `dkt init`。
- `config-schema-other`：說明 `openspec/config.yaml` 指名了其他 schema，而 `dkt new` 無論如何都用
  `docket`。建議設定 `schema: docket`；使用者同意後才編輯檔案。
- `symlink-refused` 或 `agents-block-malformed`：說明什麼擋住了 `dkt init` 並建議如何修正；
  不要自己改連結或標記。
- `openspec-missing` 或 `openspec-unsupported`：說明 Docket 支援哪些版本（finding 會指名），
  以及 `DOCKET_OPENSPEC` 可以指名執行其中一個的指令。什麼都不要安裝。
- `install-ignored`：依規則（訊息中的樣式）將 finding 分組。建議之前，列出這項修正會讓 git 看見的檔案，
  包括該目錄下今天被忽略的檔案
  （`git ls-files --others --ignored --exclude-standard -- <dir>`）。建議能放行已安裝路徑的
  最窄規則（對 `.claude/`：`.claude/*` 與 `!.claude/skills/`）。說明忽略檔是共用的
  （`.gitignore`）還是只在這台機器上（`.git/info/exclude`、`core.excludesFile`）。使用者同意後才編輯。
  `openspec/` 是刻意不放進 git 時，什麼都不要編輯：指向 README 的採用步驟。
- `title-missing`：讀每份 spec 的 Purpose 並草擬 title，一行非空的文字（`dkt lint` 檢查的形式）。
  列出所有草稿，使用者同意後才寫每個 `openspec/specs/<capability>/docket.yaml`
  （`title: <text>`），然後執行 `dkt lint`。
- `other-schema`：finding 寫 `0 of <total> steps ticked` 時，建議依這個 skill 資料夾裡的
  `convert.md` 轉換該 change。否則不建議轉換：說明它用自己的檔案完成，並用 `dkt archive` 封存，
  並列出它新增、但 delta 旁沒有 `docket.yaml` title 的 capability
  （`openspec/changes/<slug>/specs/<capability>/docket.yaml`）；主動提議像 `title-missing` 一樣草擬。
- 與框架的衝突：為使用者的指示建議措辭；絕不編輯他們的檔案。
- `no-task-commits`：說明該 change 在預設分支上的 commit 遺失了 `Change` 與 `Task` trailer
  （如 squash 的 PR 所致），而 `dkt archive` 會拒絕這個 change。決定留給使用者。
- 無法讀取的 source，或沒有 superpowers：說明哪裡有問題；修正是使用者的事。

## 5. Check again

再執行一次 `dkt inspect`，回報剩下的問題。
