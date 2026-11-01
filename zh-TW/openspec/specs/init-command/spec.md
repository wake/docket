# init-command Specification

## Purpose
定義 `docket-init` 指令：`dkt init` 每台機器安裝一次，使用者在任何專案中叫用它，讓該專案準備好使用 Docket；包括它先檢查什麼、執行什麼，以及留什麼給使用者。

## Requirements

### Requirement: Init command runs only when invoked
Docket SHALL 附上一個只在使用者叫用時才執行的 `docket-init` skill。它的 `SKILL.md` SHALL 以 frontmatter 開頭，其 `name` 為 `docket-init`，其 `description` 只說明何時使用、至多 1,024 個字元，並有 `disable-model-invocation: true`。它的資料夾 SHALL 含有 `agents/openai.yaml`，其中 `policy: allow_implicit_invocation: false`。

#### Scenario: Not run on its own
- **WHEN** 安裝了 `docket-init` 的 agent 被要求新增功能，或設定測試框架
- **THEN** 它不載入 `docket-init`，也不執行 `dkt init`

#### Scenario: Invoked
- **WHEN** 使用者在 Claude Code 輸入 `/docket-init`，或在 Codex 輸入 `$docket-init`
- **THEN** agent 依 `docket-init` 進行

### Requirement: Environment checked first
在任何會寫入檔案的動作之前，指令 SHALL 在不寫任何東西的情況下檢查：工作目錄位於 git work tree 中、`dkt` 可執行、`dkt` 找到的 OpenSpec CLI 在其支援範圍內，以及 agent 有 superpowers skill。任何一項缺少時，它 SHALL 停下來、指名每個缺少的項目、說明如何安裝，且 SHALL NOT 代為安裝。

#### Scenario: Environment first
- **WHEN** 使用者在有 OpenSpec 1.14.3 且 superpowers 可用的 git repository 中叫用 `docket-init`
- **THEN** agent 在執行 `dkt init` 之前先檢查這四項

#### Scenario: Missing OpenSpec
- **WHEN** `dkt inspect --json` 回報 `openspec-missing` 或 `openspec-unsupported`
- **THEN** agent 在執行 `dkt init` 之前停下來，說明如何安裝 OpenSpec 1.14.x 或把 `DOCKET_OPENSPEC` 指向它，且沒有任何檔案被建立或變更

#### Scenario: Missing superpowers
- **WHEN** agent 沒有 superpowers skill
- **THEN** 它在執行 `dkt init` 之前停下來，說明如何在該 agent 中安裝 superpowers

#### Scenario: Not a git repository
- **WHEN** 使用者在任何 git work tree 之外的目錄叫用 `docket-init`
- **THEN** agent 停下來，說明要開啟專案的 repository 或建立一個

### Requirement: Setup through existing steps
檢查通過後，指令 SHALL 執行 `dkt init`；然後在 `dkt workflow` 回報 `missing` 或 `incomplete` 時，依 `docket-setup` 進行；接著如同 `docket-doctor` 那樣檢查設定。它 SHALL 依 `dkt init` 寫在專案中的檔案來執行這些 skill，因為 agent 可能不會載入工作階段中途新增的 skill。其所依循的步驟在動作前會先詢問時，它 SHALL 同樣詢問。

#### Scenario: New project
- **WHEN** 使用者在沒有 commit、沒有 `openspec/`、沒有流程檔的 git repository 中叫用 `docket-init`，且環境已就緒
- **THEN** agent 執行 `dkt init`，展示流程檔的草稿並只在使用者同意後才寫入，接著執行 `dkt inspect` 並列出它回報的任何問題

#### Scenario: Differing files
- **WHEN** `dkt init` 對 `.claude/skills/docket/SKILL.md` 回報 `skill-differs`
- **THEN** agent 展示該檔案與所附檔案的差異，並只在使用者同意後才執行 `dkt init --force`

### Requirement: Init command report
指令 SHALL 以下列作結：列出設定在 repository 中建立或變更的檔案供使用者 commit，並說明可以開始開發或重構。agent 可能沒有工作階段中途新增的 skill 時，它 SHALL 說明如何載入它們。它 SHALL NOT commit。

#### Scenario: Files to commit
- **WHEN** `docket-init` 已設定好一個新專案
- **THEN** agent 把 `openspec/`、skill 資料夾、`AGENTS.md` 與 `CLAUDE.md` 列為要 commit 的檔案，不做任何 commit，並在 `.claude/skills/` 先前不存在時，於 Claude Code 中說明要輸入 `/reload-skills`
