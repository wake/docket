# Docket

[English](README.md) | 繁體中文

Docket 是一個小型命令列工具，負責檢查 [OpenSpec](https://github.com/Fission-AI/OpenSpec) 的 change 資料夾，並幫它們記帳。適用於用 [superpowers](https://github.com/obra/superpowers) 規劃與執行工作的專案。Docket 自己不存任何狀態，指令名稱是 `dkt`。

> **狀態：早期。**`init`、`new`、`lint`、`conflicts`、`sync`、`archive`、`rebase`、`status`、`record`、`issue`、`workflow` 已經能用，前面七個指令完成了第一個里程碑。下面其餘的指令還是草案，之後會調整。Docket 本身就是照這份 README 描述的流程開發的。

## 為什麼要有它

兩個工具常見的分工是：

- **OpenSpec 管「做什麼」：**現行規格、變更提案、行為差異（delta）、封存。
- **superpowers 管「怎麼做」：**brainstorming、writing-plans、subagent-driven development、TDD。

這樣搭配會留下幾個缺口，光靠提示詞補不起來：

- **兩份任務清單。**OpenSpec 追蹤 `tasks.md`，superpowers 執行的是 `plan.md`，兩份要靠 agent 自己保持一致。
- **規則只寫在提示詞裡。**OpenSpec 的 schema 指示和設定檔裡的 guidance 都只是建議，agent 沒照做也不會有任何東西報錯。
- **靜默覆蓋。**兩個 change 改到同一條需求時，後封存的那個會直接蓋掉前一個的文字，沒有任何警告。
- **沒有測試的 scenario。**沒有東西檢查 delta 裡的每個 scenario 是否都有測試。
- **忘了封存。**做完的 change 一直沒封存，下一個 session 又把工作重做一遍。

Docket 用程式能跑的檢查和記帳來補這些缺口，不另外再加一套規劃系統。

## Docket 是什麼

Docket 讀 repo 裡的檔案和 git 紀錄，做四種工作：

| 角色 | 指令 | 做什麼 |
|---|---|---|
| 檢查（唯讀，失敗時回傳非零結束碼） | `lint`、`conflicts`、`check` | 必填段落（紀錄：front matter、段落、字數上限）、殘留的佔位字、任務標題格式、複用表裡不存在的路徑、被多個進行中 change 修改的需求、沒有測試的 scenario |
| 記帳 | `status`、`sync`、`log` | 從 commit trailer 算出進度、閒置天數、一條需求怎麼演變。`sync` 只改 `plan.md` 或紀錄的勾選框，不動其他內容 |
| 組裝 | `brief`、`accept --export` | 把散在多個檔案的內容拼成給實作 agent 的 brief，或拼成給客戶確認的頁面 |
| 流程包裝 | `init`、`new`、`archive`、`rebase` | 安裝 schema，依固定順序呼叫 OpenSpec CLI 和上面的檢查，並記下 base commit 與證據 |

會寫檔的指令有：`init`（schema、skill、`docket-init` 指令與 Docket 區塊）、`new`（change 資料夾）、`record`（`docket/records/` 底下的紀錄）、`sync`（勾選框）、`archive`（經由 OpenSpec，外加 spec 標題與 `evidence.md`）、`rebase`（`docket.yaml` 的 base 那一行；舊版 dkt 建立的 change 則是 `.openspec.yaml` 的那一行）。它們都不會 commit。

## Docket 不是什麼

Docket **不會**：

- **判斷內容。**它不呼叫語言模型，也不判斷內容對不對，那是審查者的工作。
- **保存狀態。**它回報的一切都能從 repo 和 git 重新算出來。它唯一會讀的 repo 外檔案是使用者的 workflow 檔，而且絕不寫入。
- **修改 agent 設定。**它不碰 `.claude/settings.json`；在 `AGENTS.md` 和 `CLAUDE.md` 裡只寫有標記的 Docket 區塊，兩個檔案缺哪個就建哪個。
- **自己合併規格。**它不重寫 OpenSpec 的合併引擎，而是呼叫一個鎖定版本的 `openspec` CLI。

## 它假設的流程

由 superpowers 開場。OpenSpec 從不主動啟動任何步驟，只在三個時間點被呼叫：建立 change 資料夾、驗證、封存。這些規則由 `docket` skill 承載，`dkt init` 會把它裝進專案。每條路徑的審查與驗證步驟，則取自使用者的 workflow 檔（`~/.config/docket/workflow.md`，或在 `XDG_CONFIG_HOME` 底下）。agent 在第一件工作開始時，依 Docket 的範本與使用者一起寫出它；`dkt workflow` 會回報它是缺少、少了某條路徑，還是依舊版範本做的。在任何專案裡，使用者都可以輸入 `/docket-init`（Codex 用 `$docket-init`）來檢查環境並設定該專案。設定本身是獨立的 skill `docket-setup`，使用者也可以主動呼叫它來修訂這個檔案（Claude Code 用 `/docket-setup`，Codex 用 `$docket-setup`）。要檢查專案的設定，使用者呼叫 `docket-doctor`：它會執行 `dkt inspect`、檢查需要判斷的項目、列出所有問題，再與使用者逐一修正。

1. **用 brainstorming 分流。**superpowers 的 brainstorming 先讀專案，再把想法談清楚。同時符合好幾條時，取最重的：
   - 會改到 `openspec/specs/` 裡的句子：開 change，照下面的步驟走。任何設下新預期的改動也一樣，例如說不出「原本該怎樣」的 bug，或斷言新行為的測試：規格沒寫、也沒有既有行為可對照，改它就是在定義行為。
   - 讓程式回到規格行為的 bug 修正，或不改行為的重構（搬移程式碼也是重構）：修正要寫回歸測試，先看它在修正前失敗；重構要讓涵蓋它的測試前後都通過，而且突變改動的行會讓測試失敗。合併前讓整份 diff 審一次。會改變建置或執行的註解（例如編譯指令）算程式碼。完全不碰 OpenSpec。
   - prompt 文字（agent 讀了會照做的文字，例如 `AGENTS.md`、`docket` skill、schema 裡的說明；路徑列在 `AGENTS.md`）、只改測試而且測的是已經規定或已經存在的行為，以及 CI 這類檢查設定：prompt 用機械方式檢查，測試用突變它所測的程式碼來檢查，檢查設定靠 PR 自己跑的 CI 來檢查，合併前讓整份 diff 審一次。完全不碰 OpenSpec。
   - 給人讀的文件、程式碼或測試檔裡只改註解或空白（這不算改程式碼或測試）、版本號、只改相依套件清單的升級：不審查，只跑會讀到這次改動的檢查（只改沒有建置、嵌入或測試會讀的文件時，一項都不跑）。改到程式碼或測試檔時，先讓一個只讀 diff 的新 agent 確認每一行改動都是註解或空白，而且沒有會改變建置或執行的註解。依改到的東西歸類：prompt 裡的錯字算 prompt，搬移程式碼算重構。完全不碰 OpenSpec。

   第二、第三條路徑的工作不開 change，但會在 `docket/records/` 留一份紀錄，由 `dkt record` 寫出、`dkt lint` 檢查，它的 commit 帶 `Record: <slug>`，trailer 指不到任何紀錄時 `dkt lint` 會回報。`dkt status` 列出未合併的紀錄，`dkt sync` 勾選它們的任務。`dkt new` 和 `dkt record` 在 slug 已被用掉時，會改用空著的 `-2`（再來 `-3`，依此類推）後綴。

   預期行為已經確定、現在不做的工作是 issue，不是 idea。issue 有四節（`## Current`、`## Expected`、`## Unchanged`、`## Found`，`dkt issue template` 會印出）。專案若把 issue 放在外部工具，就在 `AGENTS.md` 的 `## Issues` 一節寫明，agent 經使用者同意後把 issue 開在那裡；否則 `dkt issue` 寫出 `docket/issues/<日期>-<name>.md`。來源是該 issue 路徑的紀錄或 change 就是處理它的工作，`dkt status` 依這些紀錄與 change 列出待處理或進行中的檔案 issue。`dkt lint` 會檢查分支新增的檔案 issue（front matter、四節、字數上限），並回報來源指向不存在的檔案 issue 的紀錄或 change；`dkt status --json` 把未結的 issue 列在 `result.issues`。`dkt` 不會呼叫外部工具。
2. **開 change。**在 git worktree 裡執行 `dkt new <slug>`，它會呼叫 `openspec new change`。
3. **寫提案與 delta。**
   - brainstorming 直接把 `proposal.md` 和 `specs/**/spec.md` 寫進 change 資料夾，需要時再加 `design.md`。
   - 模板從 `openspec instructions` 取得。
4. **驗證。**執行 `openspec validate --strict` 和 `dkt lint`。
5. **寫計畫。**superpowers 的 writing-plans 把 `plan.md` 寫進 change 資料夾。
   - 這是唯一的任務清單，不另外寫 `tasks.md`。
   - 需要審查者判斷的任務標 `**Review:** judgment`，其餘標 `**Review:** checks`。
   - 動手寫程式之前，plan 連同提案與 delta 審一次。
6. **實作與審查。**
   - 用 superpowers 的 subagent-driven development 搭配 TDD，但只有標 `**Review:** judgment` 的任務另派審查者。
   - 每個改程式碼的任務都改做機械檢查：新測試在程式寫好前失敗、寫好後通過，而且改動它動過的那幾行會讓某個測試失敗。只改文件的任務，用它自己的驗證步驟檢查。
   - 每個任務的 commit 都帶 trailer：

     ```
     Change: add-waitlist
     Task: 3
     ```

   - 一個 change 就是一個 pull request。合併前整個審一次，兩個審查者平行：一個用不同的模型專門挑錯，一個在全新的 context 裡對照提案與 delta 核對。`dkt check` 必須通過。
   - 用 merge commit 合併，不要 squash，每個 commit 的 trailer 才留得下來。
7. **封存。**在發版步驟，從預設分支最新版開出的分支上執行 `dkt archive <slug>...`。

### 一份任務清單

做法是 fork 一份 OpenSpec schema，把 `tasks` 這個 artifact 換成 `plan`，並把 `apply.tracks` 指向 `plan.md`。在 OpenSpec 1.14.3 的沙箱裡，`status`、`list`、`validate --strict`、`archive` 都接受這種寫法。

有兩個要注意的地方：

- **OpenSpec 數的是步驟，不是任務。**它會數所有勾選框，所以顯示的進度是步驟層級。`dkt status` 會改用任務層級回報。
- **superpowers 不會勾 `plan.md`。**superpowers 7.0 把進度記在 agent 工具的 todo 清單和一份暫存的進度帳裡。沒有 `dkt sync` 依 commit trailer 補勾的話，OpenSpec 的進度會一直是零。

我們沒有找到任何公開專案這樣只用一份清單，大多數都同時保留兩份。在 pilot 驗證之前，請把它當成實驗。

## 規劃中的指令（草案）

```
dkt init [--force]                # 安裝 docket schema、各 skill 與 Docket 區塊
dkt new <slug> --kind feature|change-request|baseline --source "<出處>" --title "<標題>"
dkt record <name> --kind fix|refactor|test|prompt|checks --source "<出處>" --title "<標題>"   # 輕審工作的紀錄
dkt issue <name> --title "<標題>" --source "<出處>"   # 預期行為已定、現在不做的工作，寫成檔案 issue；`dkt issue template` 印出各節
dkt lint [<slug>|<record>...]     # 文件規則
dkt conflicts [<slug>]            # 被多個進行中 change 碰到的需求
dkt sync <slug>|<record> [--check]   # 依 commit trailer 勾 plan.md 或紀錄的任務
dkt archive <slug>...             # 發版步驟，在從預設分支開出的分支上
dkt rebase <slug>                 # 把較新的規格接受為 change 的 base
dkt status [<slug>|<record>...] [--merged [--since <rev>] [--limit <n>]] [--json]   # 階段、任務與其 commit、進度、最後活動、未提交的改動；未合併的紀錄，加 --merged 連已合併的一起列
dkt workflow [template] [--json]  # 使用者的 workflow 檔，或範本
dkt inspect [--json]              # 專案的 Docket 設定；不寫入任何檔案
dkt brief <slug> --task N         # 給實作 agent 的 brief
dkt check <slug>                  # scenario 覆蓋與勾選一致性（PR 閘門）
dkt accept <slug> --round N [--export html]
dkt log <capability>
dkt index                         # 既有能力的複用索引
```

第一個里程碑是 `init`、`new`、`lint`、`conflicts`、`sync`、`archive`、`rebase`，拆成四個 OpenSpec change 來做，現已封存在 `openspec/changes/archive/`。

## 設計守則

- **repo 是唯一的真相。**刪掉 Docket 不會少任何資料。
- **認得 worktree。**進行中的 change 會從所有 git worktree 和未合併的分支找出來，不只看目前所在的那一個。
- **「沒有」和「讀不到」是兩回事。**外接碟沒掛、checkout 不存在時，回報「無法讀取」，不會回報「0 個 change」。
- **輸出標明 commit。**每個結果都附上 repo 與 commit SHA，不同機器跑出來的結果才能互相比對。
- **每個指令都能輸出 JSON。**所以能放進 CI、測試工具，或 agent 的 session 管理工具裡跑。

## 安裝

```
go install github.com/wake/docket/cmd/dkt@latest
```

`dkt init` 和 `dkt new` 需要 OpenSpec CLI 1.14.x 在 `PATH` 上，或用 `DOCKET_OPENSPEC` 指定要執行的指令。

`dkt init` 會把 `docket` schema 寫到這台機器的 `$XDG_DATA_HOME/openspec/schemas/docket/`（預設是 `~/.local/share/openspec/schemas/docket/`），每台機器只需要寫一次，OpenSpec 會到那裡找它。`dkt init` 也會把 `openspec/config.yaml` 設成 `schema: docket`。如果用包裝指令替 OpenSpec 換了 `XDG_DATA_HOME` 或 `HOME`，執行 `dkt` 時也要帶相同的值。在 CI 裡對 change 執行 `openspec` 之前，要先跑 `dkt init`。它也會把 `docket-init` 指令寫到 `~/.claude/skills/docket-init/`（設了 `CLAUDE_CONFIG_DIR` 時寫在它底下）與 `~/.agents/skills/docket-init/`。

### 升級

舊版 dkt 建立的 change 仍可使用：它們的 kind、source、base 會從 `.openspec.yaml` 的 `x-kind`、`x-source`、`x-base` 讀取。`dkt lint` 會要求為每一個補上帶 title 的 `docket.yaml`，每份主 spec 也各需要一個帶 title 的 `docket.yaml`。

舊版 dkt 設定過的專案，請再跑一次 `dkt init`：它會加上 `docket` skill 和 Docket 區塊。把先前版本貼進 `AGENTS.md` 的「Change workflow」章節刪掉，並提交 `.claude/skills/docket/`、`.agents/skills/docket/`、`AGENTS.md` 與 `CLAUDE.md`。

dkt v0.6.0 設定過的專案，請執行 `dkt init --force` 換上新的 Docket 區塊與 skill（在那之前 `dkt init` 會回報 `agents-block-differs`），然後提交。之後第一件開發工作開始時，agent 會和你一起設定 workflow 檔。

dkt v0.9.2 以前設定過的專案，schema 是放在 `openspec/schemas/docket/` 的一份副本。請執行 `dkt init`，確認它刪掉了 `openspec/schemas/docket/`、並在 `openspec/config.yaml` 設好 `schema: docket`，然後把兩者一起提交。專案裡每個人都要升級 dkt，並在各自的機器上執行一次 `dkt init`。改過的副本會原樣保留，`dkt init` 會回報 `schema-edited`。

### 在既有專案採用 Docket

執行 `/docket-init`，或先跑 `dkt init` 再用 `docket-doctor` skill。doctor 會讀 `dkt inspect`，逐項提出修正，經你同意後才會動手編輯。

- `dkt init` 要安裝、但被忽略規則擋在外面的檔案（`install-ignored`）：加上最窄的規則讓它們能被納入，`.claude/` 的做法是 `.claude/*` 加 `!.claude/skills/`，然後提交。`dkt init` 與 `dkt inspect` 都不會編輯忽略檔。
- 沒有 title 的主 spec（`title-missing`）：doctor 會為每一份擬好 title，經你同意後寫入。
- 用其他 OpenSpec schema 建立的 change（`.openspec.yaml` 的 `schema:`，例如 `spec-driven`）保留自己的檔案。`dkt lint` 只檢查它新增的 capability 的 title。`dkt status` 會顯示它的 schema 與 `tasks.md` 進度。`dkt archive` 封存它時不做 plan 與 base 檢查。`dkt sync` 與 `dkt rebase` 會拒絕它。一個步驟都還沒勾選的 change，可在你同意後由 agent 依 `docket-doctor/convert.md` 轉成 `docket` schema。
- 專案刻意不納入版本控制的 OpenSpec 內容，要先由專案自己提交、改寫或捨棄，才能倚賴它。doctor 不會編輯那裡的任何東西。

## 需求（規劃中）

- git
- 鎖定版本的 [OpenSpec](https://github.com/Fission-AI/OpenSpec) CLI，目前測過的是 1.14.3
- Go，從原始碼建置時需要
- [superpowers](https://github.com/obra/superpowers) 7.0.1 以上，如果你採用上面的流程

## 相關專案

Docket 與 OpenSpec、superpowers 及任何 OpenSpec 周邊 app 都沒有隸屬關係。

社群裡已有幾個專案用自訂 schema 或橋接 skill 串接 OpenSpec 和 superpowers，例如 [openspec-schemas](https://github.com/JiangWay/openspec-schemas)、[superspec](https://github.com/danielhanold/superspec)。Docket 的不同之處，是把程式能檢查的規則全部從提示詞搬進程式碼。

Docket 的設計可以和 agent session 管理工具 [Purdex](https://github.com/wake/purdex) 搭配使用，但不依賴它。

## 授權

[MIT](LICENSE)
