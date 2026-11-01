# project-setup Specification

## Purpose
定義 `dkt init` 如何讓 repository 準備好使用 Docket 流程：安裝 `docket` OpenSpec schema 與 `docket` skill，並維護 `AGENTS.md` 中的 Docket 區塊，不動使用者其餘的 agent 或 OpenSpec 設定。

## Requirements

### Requirement: Install the schema
`dkt init` SHALL 把執行中的 Docket 版本所附的 `docket` schema 寫到 `openspec-runtime` 指定的每台機器目錄，並 SHALL 以絕對路徑列出它寫入的每個檔案。它 SHALL NOT 把 schema 寫進專案。`openspec/` 不存在時，它 SHALL 先執行 `openspec init --tools none`。

#### Scenario: Fresh repository
- **WHEN** 使用者在沒有 `openspec/` 的 git repository 中執行 `dkt init`
- **THEN** `openspec/` 存在、`openspec/schemas/docket/` 不存在、每台機器的副本與所附 schema 相符，且指令以 0 結束

#### Scenario: Existing OpenSpec project
- **WHEN** 使用者在 `openspec/` 已存在的地方執行 `dkt init`
- **THEN** Docket 不執行 `openspec init`

#### Scenario: Fresh machine
- **WHEN** 每台機器目錄中沒有 `docket` schema，而使用者在專案中執行 `dkt init`
- **THEN** 所附的每個 schema 檔案都寫入該處，結果把每個檔案連同絕對路徑列為 `written`，且指令以 0 結束

#### Scenario: Linked data directory followed
- **WHEN** `XDG_DATA_HOME` 是指向另一個目錄的符號連結，而使用者執行 `dkt init`
- **THEN** schema 寫在連結所指的目錄之下，且指令以 0 結束

### Requirement: Existing installed files
對每個已存在的 schema 或 skill 檔案，`dkt init` SHALL 不動內容相同的檔案。對內容不同的檔案，它 SHALL 留在原處並回報，除非給了 `--force`；給了 `--force` 時 SHALL 覆寫該檔案並回報已覆寫。

#### Scenario: Locally edited template
- **WHEN** 每台機器的 `templates/proposal.md` 與所附檔案不同，且沒有 `--force`
- **THEN** 檔案不變，結果回報 `schema-differs` 並指名其絕對路徑，且指令以 1 結束

#### Scenario: Machine copy forced
- **WHEN** 每台機器的 `templates/proposal.md` 與 `.agents/skills/docket/SKILL.md` 都與所附檔案不同，而使用者執行 `dkt init --force`
- **THEN** 兩個檔案都與所附檔案相符，結果把兩者列為 `overwritten`，且指令以 0 結束

#### Scenario: Edited skill file
- **WHEN** `.agents/skills/docket/SKILL.md` 與所附檔案不同，且沒有 `--force`
- **THEN** 檔案不變，結果對它回報 `skill-differs`，且指令以 1 結束

#### Scenario: Re-run with no differences
- **WHEN** 使用者第二次執行 `dkt init`，期間沒有任何變動
- **THEN** 第二次執行什麼都不寫，並以 0 結束

### Requirement: Files init writes
除了 `openspec init --tools none` 建立的內容，`dkt init` SHALL 只寫入：每台機器的 schema、使用者層級的 `docket-init` 資料夾、`openspec/config.yaml` 的 `schema` 鍵、skill 資料夾、`AGENTS.md` 與 `CLAUDE.md` 的 Docket 區塊，以及缺少的 `AGENTS.md` 或 `CLAUDE.md`；並 SHALL 只移除所附 schema 的專案副本。當它要變更的路徑經過 repository 中的符號連結，或位於每台機器的 `docket` 資料夾或使用者層級的 `docket-init` 資料夾（含其下）時，它 SHALL 什麼都不變更、指名該連結，並以 1 結束。

#### Scenario: Other configuration unchanged
- **WHEN** `dkt init` 在有 `.claude/settings.json`、`.claude/skills/` 下自己的 skill，以及指名 `docket` 且有 `context` 的 `openspec/config.yaml` 的 repository 中成功
- **THEN** 這些檔案逐位元組不變，且輸出中沒有要貼上的區段

#### Scenario: Symlinked schema file refused
- **WHEN** 每台機器的 `templates/proposal.md` 是指向另一個檔案的符號連結，而使用者執行 `dkt init --force`
- **THEN** 連結所指的檔案逐位元組不變、沒有寫入任何 schema 檔案、結果指名該連結，且指令以 1 結束

#### Scenario: Symlinked skill directory refused
- **WHEN** `.claude/skills/docket` 是指向另一個目錄的符號連結，而使用者執行 `dkt init --force`
- **THEN** 該目錄及其他任何位置都沒有寫入、結果指名該連結，且指令以 1 結束

#### Scenario: Symlinked command folder refused
- **WHEN** `~/.claude/skills/docket-init` 是指向另一個目錄的符號連結，而使用者執行 `dkt init --force`
- **THEN** 該目錄及其他任何位置都沒有寫入、結果指名該連結，且指令以 1 結束

#### Scenario: User skills directory linked
- **WHEN** `~/.claude/skills` 是指向另一個目錄的符號連結，而使用者執行 `dkt init`
- **THEN** Claude Code 的副本寫在連結所指的目錄之下，且指令以 0 結束

#### Scenario: Personal skills unchanged
- **WHEN** `~/.claude/skills/` 內有使用者自己的 skill、`~/.claude/settings.json` 存在，而使用者執行 `dkt init`
- **THEN** 兩者都逐位元組不變

### Requirement: Install the framework skill
`dkt init` SHALL 把執行中的 Docket 版本所附的每個 skill 資料夾（`docket`、`docket-setup` 與 `docket-doctor`）寫到 `.claude/skills/<name>/` 與 `.agents/skills/<name>/` 兩處。使用者在這些目錄下新增的檔案 SHALL 不動。

#### Scenario: Fresh repository gets the skill
- **WHEN** 使用者在沒有這些 skill 的 git repository 中執行 `dkt init`
- **THEN** `.claude/skills/docket/`、`.claude/skills/docket-setup/`、`.claude/skills/docket-doctor/` 以及 `.agents/skills/` 下相同的三個資料夾各自的 `SKILL.md` 都與所附檔案相符，且指令以 0 結束

#### Scenario: Skill copies agree
- **WHEN** `dkt init` 已執行
- **THEN** 每個 skill 在 `.claude/skills/` 下的資料夾與 `.agents/skills/` 下的資料夾，檔案逐位元組相同，且每個 `SKILL.md` 都以 frontmatter 開頭，其 `name` 是資料夾名稱、`description` 至多 1,024 個字元

### Requirement: Framework skill content
所附的 skill SHALL 包含把工作分流到四條路徑（全審、輕審、查核、直接合併）的規則，以及框架規則：`dkt` 讀取或檢查什麼、東西放在哪裡。對每條路徑的審查與驗證步驟，它 SHALL 讓 agent 去看使用者具體化後的流程檔，並透過 `dkt workflow` 找到它。它的 description SHALL 只說明何時使用。

#### Scenario: Skill loads for development work
- **WHEN** 裝了 superpowers 的 agent 在由 `dkt init` 設定的 repository 中被要求新增功能
- **THEN** 它在 brainstorm 或寫任何檔案之前先載入 `docket` skill

#### Scenario: Skill not loaded for a question
- **WHEN** 同一個 agent 被問到某個函式做什麼
- **THEN** 它不載入 `docket` skill 就直接回答

#### Scenario: Steps from the workflow
- **WHEN** `dkt workflow` 回報 `ok`，而 agent 走輕審路徑
- **THEN** 它執行流程檔中 `## Light review` 底下的步驟

### Requirement: Docket block
`dkt init` SHALL 在 `AGENTS.md` 中維護一個 Docket 區塊：從 `<!-- docket:start -->` 到 `<!-- docket:end -->` 的各行，每個標記各自獨佔一行。檔案不存在時，它 SHALL 建立只含該區塊的 `AGENTS.md`；檔案沒有該區塊時，SHALL 附加該區塊，標記以外的文字逐位元組不變。相同的區塊 SHALL 不動；不同的區塊 SHALL 回報並保留，除非給了 `--force`，此時才取代。

#### Scenario: Block added to an existing AGENTS.md
- **WHEN** 使用者在 `AGENTS.md` 存在但沒有該區塊的地方執行 `dkt init`
- **THEN** 區塊被附加、先前的內容逐位元組不變，且指令以 0 結束

#### Scenario: AGENTS.md created
- **WHEN** 使用者在 `AGENTS.md` 不存在的地方執行 `dkt init`
- **THEN** `AGENTS.md` 只含該區塊，沒有其他內容

#### Scenario: Edited block
- **WHEN** `AGENTS.md` 中的區塊與所附區塊不同，且沒有 `--force`
- **THEN** `AGENTS.md` 不變，結果對它回報 `agents-block-differs`，且指令以 1 結束；有 `--force` 時，只取代從起始標記到結束標記的各行

### Requirement: Docket block content
所附的區塊 SHALL 告訴 agent：在開發工作開始時、brainstorm、規劃或編輯之前，使用 `docket` skill；skill 與其他 skill 衝突時以 skill 為準，並點名 skill 取代的每一項 superpowers 預設；四條路徑的步驟依使用者具體化後的流程檔進行，即使使用者的其他指示不同也一樣；除此之外，該區塊不凌駕使用者自己的指示。

#### Scenario: Skill wins over superpowers
- **WHEN** 裝了 superpowers 的 agent 在由 `dkt init` 設定的 repository 中，對一個需要 change 的功能進行 brainstorm
- **THEN** design 寫進 change 資料夾，而不是 `docs/superpowers/specs/`

#### Scenario: Workflow wins over user instructions
- **WHEN** 使用者自己的指示為輕審指名一位審查者，而具體化後的流程檔指名另一位
- **THEN** 走輕審路徑的 agent 使用流程檔所指名的那一位

### Requirement: Malformed block
當 `AGENTS.md` 或 `CLAUDE.md` 有起始標記而沒有結束標記、結束標記出現在起始標記之前，或有多於一個區塊時，`dkt init` SHALL 不寫任何檔案、對該檔案回報 `agents-block-malformed`，並以 1 結束。

#### Scenario: Malformed block
- **WHEN** `AGENTS.md` 有 `<!-- docket:start -->` 而沒有結束標記
- **THEN** 沒有寫入任何檔案，結果對 `AGENTS.md` 回報 `agents-block-malformed`，且指令以 1 結束

### Requirement: CLAUDE.md reaches the block
當 `CLAUDE.md` 有一行 `@AGENTS.md` 或 `@./AGENTS.md`，或是指向 `AGENTS.md` 的符號連結時，`dkt init` SHALL 不動 `CLAUDE.md`。`CLAUDE.md` 不存在時，它 SHALL 建立內容為 `@AGENTS.md` 的檔案。其他情況下，它 SHALL 依 `AGENTS.md` 的規則，也在 `CLAUDE.md` 中維護 Docket 區塊。

#### Scenario: CLAUDE.md imports AGENTS.md
- **WHEN** `CLAUDE.md` 含有 `@AGENTS.md` 這一行，而使用者執行 `dkt init`
- **THEN** `CLAUDE.md` 逐位元組不變，且 `AGENTS.md` 含有該區塊

#### Scenario: CLAUDE.md created
- **WHEN** 使用者在 `CLAUDE.md` 不存在的地方執行 `dkt init`
- **THEN** `CLAUDE.md` 只含 `@AGENTS.md`，沒有其他內容

#### Scenario: CLAUDE.md without the import
- **WHEN** `CLAUDE.md` 存在但沒有匯入 `AGENTS.md`，而使用者執行 `dkt init`
- **THEN** 區塊被附加到 `CLAUDE.md`，其先前的內容逐位元組不變

### Requirement: Workflow setup
設定 SHALL 是 `docket-setup` skill，使用者可隨時叫用它來寫入或修訂流程檔。當 `dkt workflow` 在開發工作開始時回報 `missing` 或 `incomplete`，`docket` skill SHALL 讓 agent 在工作之前先執行設定。設定 SHALL 依 `dkt workflow template`、agent 已有的使用者指示與它能找到的工具起草，讀取其他使用者層級的檔案前先詢問，展示完整草稿，並且只在使用者同意後才寫入（連同範本的 hash）。

#### Scenario: Setup before the first work
- **WHEN** 在由 `dkt init` 設定、且沒有流程檔的 repository 中，agent 被要求新增功能
- **THEN** 它在工作的任何其他步驟之前先執行 `dkt workflow` 與 `dkt workflow template`，並展示四個段落的草稿，使用者同意前不寫任何流程檔

#### Scenario: Setup invoked by the user
- **WHEN** `dkt workflow` 回報 `ok`，而使用者叫用 `docket-setup` 來更改輕審路徑上的審查者
- **THEN** agent 展示修訂後的完整檔案，使用者同意前不寫入任何東西

### Requirement: Stale workflow
當 `dkt workflow` 回報 `stale`，skill SHALL 讓 agent 繼續依流程檔進行，並向使用者說明一次：流程檔是由較舊的範本做出的、目前的範本有哪些不同，以及設定可以更新它。

#### Scenario: Stale workflow kept
- **WHEN** `dkt workflow` 回報 `stale`，而 agent 走輕審路徑
- **THEN** 它執行流程檔中 `## Light review` 底下的步驟，並向使用者說明一次流程已過時

### Requirement: Project copy of the schema
`dkt init` SHALL 把專案的 `openspec/schemas/docket/` 與截至執行中版本為止、每個 Docket 版本所附的 schema 逐一比較。當它恰好含有其中一個的全部檔案、逐位元組相同，且本次執行後每台機器的每個檔案都與所附檔案相符時，它 SHALL 移除這些檔案及因此變空的資料夾，並把每個檔案列為 `removed`。否則它 SHALL 保留該副本，與某個所附 schema 相同時回報 `schema-project-copy`，與任何一個都不同時回報 `schema-edited`。

#### Scenario: Shipped project copy removed
- **WHEN** `openspec/schemas/docket/` 含有 dkt v0.9.1 所附的 schema，每台機器的副本不存在，而使用者執行 `dkt init`
- **THEN** 每台機器的副本被寫入、專案副本的每個檔案被列為 `removed`、`openspec/schemas/docket/` 不再存在，且指令以 0 結束

#### Scenario: Edited project copy kept
- **WHEN** 專案的 `openspec/schemas/docket/templates/proposal.md` 與每個所附 schema 中的該檔案都不同，而使用者執行 `dkt init --force`
- **THEN** 專案副本逐位元組不變，結果回報 `schema-edited` 並指名 `openspec/schemas/docket`，且指令以 1 結束

#### Scenario: Folder without a schema
- **WHEN** 專案的 `openspec/schemas/docket/` 含有 `templates/proposal.md` 但沒有 `schema.yaml`（OpenSpec 不把它當 schema 讀），而使用者執行 `dkt init`
- **THEN** 該資料夾逐位元組不變，且沒有任何 finding 指名它

#### Scenario: Removal waits for the machine copy
- **WHEN** 專案副本與所附 schema 相同，每台機器的 `schema.yaml` 與它不同，且沒有 `--force`
- **THEN** 專案副本不變，結果回報 `schema-differs` 與 `schema-project-copy`，且指令以 1 結束

### Requirement: Schema key in config.yaml
`dkt init` SHALL 讓 `openspec/config.yaml` 指名 `docket`：檔案不存在時建立內容為 `schema: docket` 的檔案；檔案沒有頂層 `schema` 鍵時在最前面加上該行；並取代 `schema: spec-driven` 這一行，其餘每個位元組不變。其他 schema 名稱 SHALL 保留（無論有無 `--force`），並回報為 `config-schema-other`。`schema` 鍵不是單獨一行的純頂層寫法時，SHALL 什麼都不寫，並把該檔案列為無法讀取。

#### Scenario: Default schema replaced
- **WHEN** `openspec/config.yaml` 是 `openspec init --tools none` 寫出的內容，以 `schema: spec-driven` 開頭，而使用者執行 `dkt init`
- **THEN** 它的第一行是 `schema: docket`、其他每一行逐位元組不變，且結果把它列為 `written`

#### Scenario: Config file created
- **WHEN** `openspec/` 存在但沒有 `config.yaml`，而使用者執行 `dkt init`
- **THEN** `openspec/config.yaml` 只含 `schema: docket`，沒有其他內容

#### Scenario: Other schema left
- **WHEN** `openspec/config.yaml` 有 `schema: team-flow`，而使用者執行 `dkt init --force`
- **THEN** 檔案逐位元組不變，結果對它回報 `config-schema-other`，且指令以 1 結束

#### Scenario: Unreadable schema key
- **WHEN** `openspec/config.yaml` 有 `schema:`，值縮排後放在下一行，而使用者執行 `dkt init`
- **THEN** 沒有寫入或移除任何檔案，`unreadable` 列出 `openspec/config.yaml`，且指令以 3 結束

### Requirement: Install the init command
`dkt init` SHALL 把執行中的 Docket 版本所附的 `docket-init` skill 資料夾寫到 `<claude>/skills/docket-init/` 與 `<home>/.agents/skills/docket-init/`，其中 `<claude>` 在 `CLAUDE_CONFIG_DIR` 有設定且不為空時就是它，否則是 `<home>/.claude`；並 SHALL 以絕對路徑列出每個檔案。相對路徑的 `CLAUDE_CONFIG_DIR` SHALL 是錯誤，與相對路徑的 `XDG_DATA_HOME` 相同。內容不同的檔案 SHALL 回報為 `user-skill-differs`。

#### Scenario: Fresh machine gets the command
- **WHEN** `~/.claude/skills/docket-init/` 與 `~/.agents/skills/docket-init/` 都不存在，而使用者在專案中執行 `dkt init`
- **THEN** 兩個資料夾下的 `SKILL.md` 與 `agents/openai.yaml` 都與所附檔案相符，結果把這四個檔案連同絕對路徑列為 `written`，且指令以 0 結束

#### Scenario: Claude config dir
- **WHEN** `CLAUDE_CONFIG_DIR` 是 `/srv/claude-work`，而使用者執行 `dkt init`
- **THEN** Claude Code 的副本寫在 `/srv/claude-work/skills/docket-init/` 之下、`~/.claude/` 下沒有寫入任何東西，而 Codex 的副本仍寫在 `~/.agents/skills/docket-init/` 之下

#### Scenario: Edited command copy
- **WHEN** `~/.agents/skills/docket-init/SKILL.md` 與所附檔案不同，且沒有 `--force`
- **THEN** 檔案不變，結果回報 `user-skill-differs` 並指名其絕對路徑，且指令以 1 結束；有 `--force` 時，它與所附檔案相符，並列為 `overwritten`

#### Scenario: Copies agree
- **WHEN** `dkt init` 已執行
- **THEN** 兩個 `docket-init` 資料夾的檔案逐位元組相同

### Requirement: Ignored install paths
寫入之後，對它安裝或保留在 repository 中、被 git 忽略且未被追蹤的每個檔案，`dkt init` SHALL 回報 `install-ignored`，指名該檔案與忽略它的規則（忽略檔、行號與樣式），並 SHALL 以 1 結束。它仍 SHALL 寫入這類檔案，且 SHALL NOT 變更任何忽略檔。

#### Scenario: Ignored skill reported
- **WHEN** `.gitignore` 的第三行是 `.claude/`，而使用者執行 `dkt init`
- **THEN** `.claude/skills/` 下的每個檔案都被寫入、`install-ignored` finding 為每一個檔案指名 `.gitignore`、第 3 行與 `.claude/`、`.gitignore` 逐位元組不變，且指令以 1 結束

#### Scenario: Tracked file not reported
- **WHEN** `.gitignore` 含有 `AGENTS.md`，而 `AGENTS.md` 已被 git 追蹤
- **THEN** 沒有 `install-ignored` finding 指名 `AGENTS.md`

#### Scenario: Nothing ignored
- **WHEN** 沒有任何已安裝的檔案被忽略，且沒有其他差異
- **THEN** `dkt init` 不回報 `install-ignored` finding，並以 0 結束
