---
name: docket-init
description: >-
  為這個專案設定 Docket：檢查 git、dkt、OpenSpec CLI 與 superpowers 是否就緒，執行 dkt init，
  在使用者的流程檔缺少時寫出它，檢查設定，並列出要 commit 的檔案。只在使用者主動叫用時執行。
disable-model-invocation: true
---

# Docket init

逐步為這個專案設定 Docket。這個 command 只透過 `dkt init`、`docket-setup` 與 `docket-doctor`
寫入，各自取得它所要求的同意。它從不安裝任何東西，也從不 commit。

## 1. Check the environment, writing nothing

先執行下面每一項檢查，把缺少的都收集起來，再說話。因前面的檢查失敗而無法執行的檢查，
列為尚未檢查，並附原因。

- git：執行 `git rev-parse --show-toplevel`。失敗時，使用者應開啟專案的 repository，
  或用 `git init` 建立一個。
- `dkt`：無法啟動時，使用者應以 `go install github.com/wake/docket/cmd/dkt@latest` 安裝。
- OpenSpec，只在 work tree 內：執行 `dkt inspect --json`。出現 `openspec-missing` 或
  `openspec-unsupported` 時，使用者應安裝 OpenSpec 1.14.x
  （`npm install -g @fission-ai/openspec@1.14`），或把 `DOCKET_OPENSPEC` 設為能執行它的指令，
  例如 `npx -y @fission-ai/openspec@1.14.3`。以結束碼 2 或 4 結束時，它所說的也算作問題。
- superpowers：檢查你自己的 skill 清單有 brainstorming、writing-plans 與
  subagent-driven-development（Claude Code 把它們列為 `superpowers:brainstorming` 等）。
  缺少任何一個時：在 Claude Code 用 `/plugin install superpowers@claude-plugins-official`；
  在 Codex 用 `/plugins` 並搜尋 superpowers；其他 agent 見 https://github.com/obra/superpowers。
  然後開新的 session。

同時記下 repository 裡現在是否已有 `.claude/skills/` 與 `.agents/skills/`，要在任何東西執行之前記；
步驟 5 會用到這份記錄。

有任何東西缺少時，一次回報所有缺少的，附上各自的安裝方式，並在步驟 2 之前停下。什麼都不要安裝。

## 2. Run `dkt init --json`

它寫出專案的檔案、每台機器一份的 schema，以及這個 command 的使用者層級副本。對它回報的任何 finding，
照 `docket-doctor` 對該 finding 所說的做（讀它的 `SKILL.md`，見步驟 4）：顯示不同的檔案哪裡不同，
並在使用者同意後才執行 `dkt init --force`。它以結束碼 2、3 或 4 結束時，告訴使用者它說了什麼，然後停下。

## 3. The workflow file

執行 `dkt workflow --json`。

- `missing` 或 `incomplete`：從這個 repository 的 `.claude/skills/`（Claude Code）或
  `.agents/skills/`（Codex）讀 `docket-setup/SKILL.md` 並照做。說明 `dkt init` 在這個 session
  期間新增的 skill 可能尚未載入，所以才從檔案讀取。
- `stale`：說明這點並提議執行設定。
- `ok`：繼續。

## 4. Check the setup

以同樣方式讀 `docket-doctor/SKILL.md` 並照做。

## 5. Report

列出 `git status --short` 顯示由設定建立或變更的東西，供使用者 commit
（`openspec/`、`.claude/skills/`、`.agents/skills/`、`AGENTS.md`、`CLAUDE.md`）。
說明流程檔、每台機器一份的 schema 與 `docket-init` 的副本在 repository 之外。不要 commit。

- 在沒有 commit 的 repository，說明第一個 commit 會給它 `dkt new` 需要的預設分支。
- 在 Claude Code，步驟 1 的記錄說 `.claude/skills/` 原本不存在時，說明要輸入 `/reload-skills`。
  在 Codex，`$docket` 沒有顯示時，說明要重新啟動。

最後說明可以開始開發或重構了：使用者描述工作，由 `docket` skill 分流。
