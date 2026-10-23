# Docket

[English](README.md) | 繁體中文

Docket 是一個小型命令列工具，負責檢查 [OpenSpec](https://github.com/Fission-AI/OpenSpec) 的 change 資料夾，並幫它們記帳。適用於用 [superpowers](https://github.com/obra/superpowers) 規劃與執行工作的專案。Docket 自己不存任何狀態，指令名稱是 `dkt`。

> **狀態：設計階段。**還沒有寫任何程式碼，下面列的指令都是草案，之後會調整。第一版會照這份 README 描述的流程來開發。

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
| 檢查（唯讀，失敗時回傳非零結束碼） | `lint`、`conflicts`、`check` | 必填段落、殘留的佔位字、任務標題格式、複用表裡不存在的路徑、被多個進行中 change 修改的需求、沒有測試的 scenario |
| 記帳 | `status`、`sync`、`log` | 從 commit trailer 算出進度、閒置天數、一條需求怎麼演變。`sync` 只改 `plan.md` 的勾選框，不動其他內容 |
| 組裝 | `brief`、`accept --export` | 把散在多個檔案的內容拼成給實作 agent 的 brief，或拼成給客戶確認的頁面 |
| 流程包裝 | `init`、`new`、`archive`、`rebase` | 安裝 schema，依固定順序呼叫 OpenSpec CLI 和上面的檢查，並記下 base commit 與證據 |

會寫檔的指令有：`init`（schema）、`new`（change 資料夾）、`sync`（勾選框）、`archive`（經由 OpenSpec，外加 `evidence.md`）、`rebase`（`.openspec.yaml` 的一行）。它們都不會 commit。

## Docket 不是什麼

Docket **不會**：

- **判斷內容。**它不呼叫語言模型，也不判斷內容對不對，那是審查者的工作。
- **保存狀態。**它回報的一切都能從 repo 和 git 重新算出來。
- **修改 agent 設定。**它不碰 `CLAUDE.md`、`AGENTS.md`、`.claude/settings.json`。
- **自己合併規格。**它不重寫 OpenSpec 的合併引擎，而是呼叫一個鎖定版本的 `openspec` CLI。

## 它假設的流程

由 superpowers 開場。OpenSpec 從不主動啟動任何步驟，只在三個時間點被呼叫：建立 change 資料夾、驗證、封存。

1. **用 brainstorming 分流。**superpowers 的 brainstorming 先讀專案，再把想法談清楚。
   - 如果這件事不會改到 `openspec/specs/` 裡的任何一句話，就到此為止，走輕量路徑，完全不碰 OpenSpec。
2. **開 change。**在 git worktree 裡執行 `dkt new <slug>`，它會呼叫 `openspec new change`。
3. **寫提案與 delta。**
   - brainstorming 直接把 `proposal.md` 和 `specs/**/spec.md` 寫進 change 資料夾，需要時再加 `design.md`。
   - 模板從 `openspec instructions` 取得。
4. **驗證。**執行 `openspec validate --strict` 和 `dkt lint`。
5. **寫計畫。**superpowers 的 writing-plans 把 `plan.md` 寫進 change 資料夾。
   - 這是唯一的任務清單，不另外寫 `tasks.md`。
6. **實作與審查。**
   - 用 superpowers 的 subagent-driven development 搭配 TDD。
   - 每個任務的 commit 都帶 trailer：

     ```
     Change: add-waitlist
     Task: 3
     ```

   - 審查 pull request，`dkt check` 必須通過。
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
dkt init [--force]                # 安裝 docket schema
dkt new <slug> --kind feature|change-request|fix|baseline --source "<出處>"
dkt lint [<slug>...]              # 文件規則
dkt conflicts [<slug>]            # 被多個進行中 change 碰到的需求
dkt sync <slug> [--check]         # 依 commit trailer 勾 plan.md
dkt archive <slug>...             # 發版步驟，在從預設分支開出的分支上
dkt rebase <slug>                 # 把較新的規格接受為 change 的 base
dkt status [--json]               # 階段、任務、worktree、負責人、PR、閒置天數、驗收輪
dkt brief <slug> --task N         # 給實作 agent 的 brief
dkt check <slug>                  # scenario 覆蓋與勾選一致性（PR 閘門）
dkt accept <slug> --round N [--export html]
dkt log <capability>
dkt index                         # 既有能力的複用索引
```

第一個里程碑是 `init`、`new`、`lint`、`conflicts`、`sync`、`archive`、`rebase`，拆成 `openspec/changes/` 底下的四個 OpenSpec change 來做。

## 設計守則

- **repo 是唯一的真相。**刪掉 Docket 不會少任何資料。
- **認得 worktree。**進行中的 change 會從所有 git worktree 和未合併的分支找出來，不只看目前所在的那一個。
- **「沒有」和「讀不到」是兩回事。**外接碟沒掛、checkout 不存在時，回報「無法讀取」，不會回報「0 個 change」。
- **輸出標明 commit。**每個結果都附上 repo 與 commit SHA，不同機器跑出來的結果才能互相比對。
- **每個指令都能輸出 JSON。**所以能放進 CI、測試工具，或 agent 的 session 管理工具裡跑。

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
