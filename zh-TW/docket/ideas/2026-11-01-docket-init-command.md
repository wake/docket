---
title: 讓專案準備好使用 Docket 的使用者層級指令
---

2026-11-01 記錄，來自設定一個新專案時。在 change `install-schema-per-machine` 之後討論。

## 問題

`docket`、`docket-setup` 與 `docket-doctor` 這幾個 skill 只透過 `dkt init` 進到專案，所以從未執行過它的專案沒有 Docket 指令可以開始。目前使用者在終端機執行 `dkt init`，或請 agent 執行。

## 方向，尚未決定

- 一個使用者叫用的指令（不由模型自己載入），安裝在使用者層級，例如 `~/.claude/skills/docket-init/`，這樣它存在於這台機器的每個專案。
- 單單執行 `dkt init`，比起請 agent 執行並沒有多少好處。這個指令的價值在於之後需要 agent 的部分：流程檔缺少時設定它，然後像 `docket-doctor` 那樣檢查設定。
- 它建立在 `install-schema-per-machine` 之上，該 change 已經讓 `dkt init` 每台機器寫一次東西。

## 來自維護者，2026-11-01

- 它的運作像 Claude Code 自己的 `/init`：可以在任何專案中輸入。環境是否支援，由指令自己檢查。
- 它是在 session 內使用的指令，但它的步驟不限於 agent：不需要判斷的部分也可以從 CLI 執行。
- 它先檢查 superpowers 與 OpenSpec 是否已安裝；沒有它們，設定 Docket 沒有用。
- 在既有專案中，它調查程式碼並建立程式碼地圖或索引。可能有幫助的工具正整理成一份報告，讓維護者先讀；見 `2026-10-31-codebase-decomposition.md`。
- 之後，專案準備好開始開發，或開始重構（`2026-10-31-refactoring.md`）。

## 步驟，2026-11-01 同意

1. 檢查環境，不寫入任何東西：OpenSpec 1.14.x（`dkt` 已經會檢查；`dkt inspect` 也會回報）、superpowers（只有 agent 看得到它有哪些 skill，而 `dkt` 不讀使用者的 agent 設定，所以由指令檢查），以及 git repository。缺少什麼時，停下來並說明如何安裝；安裝會寫到專案之外，所以指令不做。
2. `dkt init`：專案的檔案與每台機器的 schema。
3. 流程檔缺少時執行 `docket-setup`。
4. 在既有專案中，調查它：介面清單
   （`2026-10-31-codebase-decomposition.md`、`2026-10-24-add-index.md`）、採用檢查（`2026-10-31-existing-project-adoption.md`），以及供使用者確認的 capability 草稿，作為 baseline 的開頭。
5. 像 `docket-doctor` 那樣檢查設定。
6. 回報要 commit 什麼，以及開發或重構可以開始了。

拆分：這個 idea 是指令本身，即步驟 1、2、3、5 與 6，大多是把既有的東西接起來。步驟 4 來自它點名的三個 idea，各自作為獨立的一件工作完成；指令在每一部分出貨時再接上。

## 2026-11-01 已決定

`dkt init` 把指令安裝在使用者層級，Claude Code 是 `~/.claude/skills/docket-init/`（`/docket-init`），Codex 是 `~/.agents/skills/docket-init/`（`$docket-init`），做法和安裝每台機器的 schema 一樣：缺少的副本就寫入並印出路徑，不同的就回報，`--force` 覆寫。只有 `dkt init` 寫它。放棄了 plugin：它仍然需要另外安裝 `dkt`（把建置好的執行檔放進 plugin 的 git 來源並不實際），而且 Codex 會需要第二種格式。

## 2026-11-01 發現

來自 Claude Code 2.1.316 與 Codex 0.153.4 的文件；沒有實際嘗試。

- 使用者層級的 skill，Claude Code 在 `~/.claude/skills/`，Codex 在 `~/.agents/skills/`。Codex 沒有自訂的 `/name`；skill 以 `$name` 輸入，或從 `/skills` 挑選。
- 在 Claude Code 中，個人 skill 優先於同名的專案 skill，所以使用者層級的項目不能叫 `docket`，否則會遮住每個專案自己的 `docket` skill。
- Claude Code 會撿起 session 執行期間加進 `.claude/skills/` 的 skill，但如果 session 開始時那個資料夾不存在就不會：這時使用者要輸入 `/reload-skills`。Codex 偵測得到變動，但沒出現時會說要重新啟動。所以在新專案中，指令改為讀取 `dkt init` 剛寫下的 `SKILL.md` 檔並照著做，而不是用名稱交給 `/docket-setup`。
- Claude Code plugin 由使用者安裝
  （`claude plugin install <plugin> --marketplace wake/docket`，預設是使用者範圍），它的 skill 以 `/<plugin>:<skill>` 輸入，而它的 `bin/` 裡的檔案在 agent shell 的 PATH 上。第三方 marketplace 除非使用者開啟，否則不會自動更新。Codex 也有 plugin（`codex plugin marketplace add owner/repo`）；它們的版面配置沒有檢查。

## 待釐清問題

- 誰來安裝它：`dkt` 依使用者要求安裝，像安裝每台機器的 schema 那樣，還是 plugin。使用者自己的 agent 設定比 OpenSpec 的資料目錄更私人，所以寫到那裡可能需要單獨的同意。預先建置的執行檔（`2026-11-01-prebuilt-binaries.md`）改變了權衡，因為 plugin 可以把 `dkt` 一起帶來。
- 一個 skill 資料夾能否同時服務兩個 agent：Codex 是否忽略 Claude Code 的 frontmatter 鍵，文件沒有記載。
- 它的第一次安裝仍需要終端機；它只是在之後的專案省掉那個步驟。

## 相關

- `2026-10-31-existing-project-adoption.md`：既有專案如何開始使用 Docket。
- `2026-10-31-global-schema.md`：它所建立在其上的每台機器安裝。

## 狀態

由 change `add-docket-init` 完成：`docket-init` 指令（步驟 1、2、3、5 與 6），由 `dkt init` 每台機器安裝一次到 `~/.claude/skills/`（或 `CLAUDE_CONFIG_DIR`）與 `~/.agents/skills/`，由 `dkt inspect` 回報成 `user-skill-*`，以及讓 `dkt inspect` 在沒有預設分支的 repository 中執行。步驟 4，也就是既有專案的調查，留給它點名的那些 idea。
