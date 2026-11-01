# setup-inspection Specification

## Purpose
定義 `dkt inspect` 回報專案的 Docket 設定的哪些內容，以及使用者叫用以執行它、檢查需要判斷的事項並修正所發現問題的 `docket-doctor` 指令。

## Requirements

### Requirement: Inspect writes nothing
`dkt inspect` SHALL 不變更任何檔案，無論在 repository 內或其他地方。它 SHALL 啟動的行程只有 git 與 OpenSpec CLI 的版本查詢。沒有回報任何 finding 時它 SHALL 以 0 結束，回報任何 finding 時以 1 結束。

#### Scenario: Nothing written
- **WHEN** 使用者在下列 repository 中執行 `dkt inspect`：每個已安裝的檔案都與所附檔案不同、專案持有所附 schema 的副本、`openspec/config.yaml` 指名 `spec-driven`、流程檔不存在，且兩個使用者層級的 `docket-init` 資料夾都不存在
- **THEN** repository 中、使用者的設定目錄下、每台機器的 schema 目錄下，以及 `~/.claude/` 或 `~/.agents/` 下，都沒有任何檔案被建立、變更或移除

#### Scenario: Clean setup
- **WHEN** repository 是由執行中版本的 `dkt init` 設定的，OpenSpec CLI 回報 1.14.3，且 `dkt workflow` 會回報 `ok`
- **THEN** `dkt inspect --json` 不回報任何 finding，並以 0 結束

### Requirement: Installed files inspected
`dkt inspect` SHALL 依 `dkt init` 使用的規則，檢查 `dkt init` 會安裝或保留的每個檔案（也包括每台機器的 schema、使用者層級的 `docket-init` 檔案與 `config.yaml`），並 SHALL 在 `result.files` 中把每一個回報為 `same`、`missing` 或 `differs`。缺少的檔案 SHALL 給出 `schema-missing`、`user-skill-missing`、`skill-missing` 或 `agents-block-missing`；內容不同者給出 `schema-differs`、`user-skill-differs`、`skill-differs` 或 `agents-block-differs`；格式錯誤的區塊給出 `agents-block-malformed`；`dkt init` 會拒絕的路徑給出 `symlink-refused`。

#### Scenario: Before init
- **WHEN** 使用者在從未執行過 `dkt init` 的 git repository 中、於沒有每台機器 schema 的機器上執行 `dkt inspect --json`
- **THEN** finding 含有指名每台機器檔案的 `schema-missing`、`skill-missing`、`agents-block-missing` 與 `config-schema-default`，且指令以 1 結束

#### Scenario: Edited skill
- **WHEN** `.claude/skills/docket/SKILL.md` 與所附檔案不同
- **THEN** `result.files` 把它列為 `differs`，且 finding 含有指名它的 `skill-differs`

#### Scenario: Machine copy differs
- **WHEN** 每台機器的 `schema.yaml` 與所附檔案不同
- **THEN** `result.files` 以其絕對路徑把它列為 `differs`、kind 為 `schema`，且 finding 含有指名它的 `schema-differs`，並說明 `dkt init --force` 會覆寫它

#### Scenario: Command not installed
- **WHEN** `~/.agents/skills/docket-init/` 不存在
- **THEN** `result.files` 把它所附的每個檔案的絕對路徑列為 `missing`、kind 為 `user-skill`，且 finding 含有指名每一個的 `user-skill-missing`

#### Scenario: CLAUDE.md imports AGENTS.md
- **WHEN** `CLAUDE.md` 含有 `@AGENTS.md` 這一行，且 `AGENTS.md` 含有所附區塊
- **THEN** 沒有任何 finding 指名 `CLAUDE.md`

#### Scenario: No home directory
- **WHEN** `XDG_DATA_HOME` 與 `CLAUDE_CONFIG_DIR` 都未設定，且找不到家目錄
- **THEN** finding 含有附上該原因的 `schema-missing` 與 `user-skill-missing`，且指令不以 4 結束

### Requirement: OpenSpec CLI inspected
`dkt inspect` SHALL 如其他指令那樣找到 OpenSpec CLI 並讀取其版本。無法啟動任何 CLI 時它 SHALL 回報 `openspec-missing`；版本超出可接受範圍時回報 `openspec-unsupported`，指名版本與可接受的範圍。兩者都 SHALL NOT 讓它以 4 結束。

#### Scenario: Unsupported OpenSpec
- **WHEN** OpenSpec CLI 回報版本 1.3.0，而使用者執行 `dkt inspect --json`
- **THEN** `result.openspec.version` 為 `1.3.0`，`result.openspec.supported` 為 false，finding 含有 `openspec-unsupported`，且指令以 1 結束

#### Scenario: No OpenSpec
- **WHEN** 無法啟動任何 OpenSpec CLI
- **THEN** `result.openspec.version` 為 null，且 finding 含有 `openspec-missing`

### Requirement: Workflow file inspected
`dkt inspect` SHALL 在 `result.workflow` 中回報 `dkt workflow --json` 在其 `result` 中回報的內容，並 SHALL 給出相同的 finding。找不到家目錄時，它 SHALL 回報附上該原因的 `workflow-missing`，而不是以 4 結束。無法讀取的流程檔 SHALL 列在 `unreadable` 中。

#### Scenario: Workflow reported
- **WHEN** 流程檔缺少 `## Check`，而使用者執行 `dkt inspect --json`
- **THEN** `result.workflow.status` 為 `incomplete`，且 finding 含有指名 `Check` 的 `workflow-incomplete`

### Requirement: Trailers on the default branch
對每個其資料夾在預設分支的 tree 中、且其在該處的 plan 有 task 的進行中 change，當從預設分支的 tip 可到達、從該 change 記錄的 base 不可到達的 commit 中，沒有任何一個同時帶有 `Change: <slug>` 與 `Task:` trailer 時，`dkt inspect` SHALL 回報 `no-task-commits`。base 缺少或不是 tip 的祖先的 change，或其他 schema 的 change，SHALL 被略過。`result.changes` SHALL 列出每個這類 change，被略過時附上原因。

#### Scenario: Squashed change
- **WHEN** `add-x` 在預設分支的 tree 中，plan 有兩個 task，而預設分支上自其 base 之後唯一的 commit 沒有 `Change: add-x` trailer
- **THEN** finding 含有 `add-x` 的 `no-task-commits`

#### Scenario: Merged with a merge commit
- **WHEN** `add-x` 經由 merge commit 進入預設分支，而該 merge commit 的第二個 parent 的歷史中有帶有 `Change: add-x` 與 `Task: 1` 的 commit
- **THEN** 沒有任何 finding 指名 `add-x`

#### Scenario: Base not an ancestor
- **WHEN** `add-x` 在預設分支的 tree 中，而其記錄的 base 不是預設分支 tip 的祖先
- **THEN** 沒有任何 finding 指名 `add-x`，且 `result.changes` 把 `add-x` 列為被略過並附上原因

#### Scenario: Change of another schema skipped
- **WHEN** `add-x` 在預設分支的 tree 中，帶有 `schema: spec-driven` 與有 task 的 `plan.md`，且沒有任何 commit 帶有 `Change: add-x`
- **THEN** 沒有任何 finding 指名 `add-x`，且 `result.changes` 把 `add-x` 列為被略過，說明它是以 `spec-driven` schema 做出的

### Requirement: Doctor runs only when invoked
Docket SHALL 附上一個只在使用者叫用時才執行的 `docket-doctor` 指令。它的 `SKILL.md` SHALL 在 frontmatter 中有 `disable-model-invocation: true`，且它的資料夾 SHALL 含有 `agents/openai.yaml`，其中 `policy: allow_implicit_invocation: false`。

#### Scenario: Doctor not run on its own
- **WHEN** 在由 `dkt init` 設定的 repository 中，agent 被要求新增功能，或被問到某個函式做什麼
- **THEN** 它不載入 `docket-doctor`，也不執行 `dkt inspect`

#### Scenario: Doctor invoked
- **WHEN** 使用者叫用 `docket-doctor`
- **THEN** agent 在 doctor 的任何其他步驟之前先執行 `dkt inspect --json`

### Requirement: Doctor judges the setup
在 `dkt inspect` 之後，doctor SHALL 檢查需要判斷的事項：這個環境是否具備流程中每個步驟所指名的東西，因此適用哪個審查的備案；流程檔與其所在情境中使用者的其他指示在哪裡不一致；與框架衝突的指示，例如 squash merge；以及 superpowers 是否可用。除非使用者先同意，它 SHALL 不讀取其他使用者層級的檔案。

#### Scenario: Drift found
- **WHEN** 流程檔為跨模型審查者指名某個 effort，而 agent 情境中使用者的指示指名另一個
- **THEN** doctor 把兩者各自引用後列為一個問題

### Requirement: Doctor lists, then guides
doctor SHALL 在修正任何問題之前，先列出它找到的每個問題，各附原因與建議的修正。接著它 SHALL 一次處理一個，且只在使用者同意後才動作：流程檔透過 `docket-setup`；已安裝的檔案先展示差異，再執行 `dkt init` 或 `dkt init --force`；使用者自己的指示檔絕不編輯，只提議措辭。它 SHALL 以再次執行 `dkt inspect` 作結。

#### Scenario: Problems listed first
- **WHEN** 使用者叫用 `docket-doctor`，而 `dkt inspect` 回報 `skill-differs` 與 `workflow-stale`
- **THEN** agent 在更動任何東西之前，先列出這兩個問題及其原因

#### Scenario: Edited skill fixed with consent
- **WHEN** doctor 處理到 `skill-differs` 問題
- **THEN** 它展示該檔案與所附檔案的差異，並只在使用者同意後才執行 `dkt init --force`

### Requirement: Project copy and schema key inspected
`dkt inspect` SHALL 依 `dkt init` 的處理方式回報專案的 `openspec/schemas/docket/`：與某個所附 schema 相同時回報 `schema-project-copy`，說明 `dkt init` 會移除它；與任何一個都不同時回報 `schema-edited`，說明它會取代每台機器的副本被使用。`openspec/config.yaml` 不存在、沒有 `schema` 鍵或指名 `spec-driven` 時，它 SHALL 回報 `config-schema-default`；指名其他 schema 時回報 `config-schema-other`。

#### Scenario: Project copy reported
- **WHEN** 專案的 `openspec/schemas/docket/` 含有 dkt v0.9.1 所附的 schema
- **THEN** finding 含有指名 `openspec/schemas/docket` 的 `schema-project-copy`，且指令以 1 結束

#### Scenario: Default schema key reported
- **WHEN** `openspec/config.yaml` 以 `schema: spec-driven` 開頭
- **THEN** `result.files` 把 `openspec/config.yaml` 列為 `differs`、kind 為 `config`，且 finding 含有指名它的 `config-schema-default`

### Requirement: Repository without a default branch
`dkt inspect` SHALL 在 `result.default_branch` 中回報它檢查 trailer 所用的分支。找不到預設分支時（例如沒有任何 commit 的 repository），它 SHALL 略過 trailer 檢查，把 `result.default_branch` 回報為 null、`result.changes` 為空，其餘照常回報，且 SHALL NOT 因此以 4 結束。

#### Scenario: Repository without commits
- **WHEN** 使用者在沒有任何 commit 的 git repository 中執行 `dkt inspect --json`
- **THEN** `result.default_branch` 為 null，`result.changes` 為空，`result.openspec` 與 `result.files` 照常回報，沒有任何 finding 與預設分支有關，且指令不以 4 結束

#### Scenario: Default branch found
- **WHEN** 使用者在預設分支為 `main` 的 repository 中執行 `dkt inspect --json`
- **THEN** `result.default_branch` 為 `main`

### Requirement: Ignored files inspected
對 repository 中 `dkt init` 會安裝或保留、而 git 忽略且未追蹤的每個檔案，無論該檔案是否已存在，`dkt inspect` SHALL 如 `dkt init` 在寫入後那樣回報 `install-ignored`。它 SHALL 不變更任何忽略檔。

#### Scenario: Ignored file inspected
- **WHEN** `.git/info/exclude` 的第七行是 `openspec/`，而使用者執行 `dkt inspect --json`
- **THEN** finding 含有指名 `openspec/config.yaml` 並附上 `.git/info/exclude`、第 7 行與 `openspec/` 的 `install-ignored`，且 `.git/info/exclude` 逐位元組不變

### Requirement: Main spec titles inspected
對 worktree 的 `openspec/specs/` 下每個沒有 title 的主 spec，`dkt inspect` SHALL 回報 `title-missing`，如同未指名 change 時 `dkt lint` 的做法。當 `openspec` 本身是符號連結（`symlink-refused`）時，它 SHALL 為這項檢查而不讀取其下任何東西。

#### Scenario: Main spec title inspected
- **WHEN** `openspec/specs/booking/spec.md` 存在，旁邊沒有 `docket.yaml`
- **THEN** `dkt inspect` 回報指名 `openspec/specs/booking/docket.yaml` 的 `title-missing`，並以 1 結束

### Requirement: Changes of another schema inspected
對 worktree 的 `openspec/changes/` 中每個屬於其他 schema 的進行中 change，`dkt inspect` SHALL 回報 `other-schema`，指名它的 `.openspec.yaml`、它的 schema，以及其 `tasks.md` 的核取方塊共幾個、已勾選幾個。當 `openspec` 本身是符號連結（`symlink-refused`）時，它 SHALL 為這項檢查而不讀取其下任何東西。

#### Scenario: Change of another schema inspected
- **WHEN** `openspec/changes/add-x/.openspec.yaml` 含有 `schema: spec-driven`，且其 `tasks.md` 有四個核取方塊，沒有任何一個被勾選
- **THEN** `dkt inspect` 回報指名 `openspec/changes/add-x/.openspec.yaml`、`spec-driven` 與 0 of 4 的 `other-schema`

### Requirement: Doctor fixes adoption problems
doctor SHALL 把每個 `install-ignored`、`title-missing` 與 `other-schema` 問題，依其列出的步驟處理，且只在使用者同意後才動作：對忽略規則，先展示這項變更會讓 git 看見哪些檔案，以及該忽略檔是共用的還是本地的；對 title，列出依每個 spec 的 Purpose 起草的 title，再一起寫入；對其他 schema 的 change，只有在沒有任何步驟被勾選時才提議轉換它。

#### Scenario: Ignore rule changed with consent
- **WHEN** doctor 處理到 `.claude/skills/` 下的檔案的 `install-ignored`，這些檔案被 `.gitignore` 中的 `.claude/` 忽略
- **THEN** 它提議一條放行 `.claude/skills/` 的規則、列出 git 之後會看見的檔案、說明 `.gitignore` 與每個 clone 這個專案的人共用，並只在使用者同意後才編輯它

#### Scenario: Titles written with consent
- **WHEN** doctor 處理到三個主 spec 的 `title-missing`
- **THEN** 它在寫入任何一個之前，先列出三者各自起草的 title，並只在使用者同意後才寫入三個 `docket.yaml` 檔案

#### Scenario: Unstarted change converted
- **WHEN** doctor 處理到 `add-x` 的 `other-schema`，0 of 4 個步驟被勾選，且使用者同意轉換它
- **THEN** agent 依 doctor 所附的轉換指南把 `add-x` 轉成 `docket` schema，之後 `dkt lint add-x` 不回報任何 finding

#### Scenario: Started change left
- **WHEN** doctor 處理到 `add-x` 的 `other-schema`，2 of 4 個步驟被勾選
- **THEN** 它不提議轉換，說明以 `add-x` 自己的檔案完成它並用 `dkt archive` 封存，並列出 `add-x` 新增而沒有 title 的任何 capability
