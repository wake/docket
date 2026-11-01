# zh-TW：繁體中文原稿

這個目錄放 Docket 文件的繁體中文原稿。英文檔是從這裡翻出去的譯本。

## 對應

路徑鏡射原檔：`zh-TW/<路徑>` ↔ `<路徑>`。例如 `zh-TW/openspec/specs/ideas/spec.md` ↔ `openspec/specs/ideas/spec.md`。
README 例外，繁中版是根目錄的 `README.zh-TW.md`。封存的 change（`openspec/changes/archive/`）不翻。

## 流程

1. 先寫或改 `zh-TW/<路徑>`。
2. 維護者確認。
3. 翻成英文，寫回 `<路徑>`。
4. 兩邊放在同一個 PR。

程式只讀英文檔；這個目錄不被 `dkt` 或 OpenSpec 讀取。

## 保持英文

- 程式碼、指令、旗標、路徑、檔名、JSON 欄位、YAML 鍵、lint 規則名稱、狀態值、commit trailer、反引號裡的一切。
- 程式解析的標記：
  - spec：`## Purpose`、`## Requirements`、`## ADDED Requirements` 等，`### Requirement: <名稱>` 與 `#### Scenario: <名稱>` 整行（名稱會被其他檔引用），`SHALL`、`MUST`、`MAY`，`**WHEN**`、`**THEN**`、`**AND**`。
  - front matter 的鍵。`title` 與 skill 的 `description` 的值要翻。
  - lint 檢查的固定標題：issue 的 `## Current` 等四段、record 的 `## What` 等、proposal 的各段、plan 的 `### Task N:` 整行與 `**Scenarios:**`、`**Review:**`、`**Files:**`，流程檔的五個路徑標題。
  - 標記用的 HTML 註解。給撰寫者的說明註解要翻。
- commit、PR、程式碼註解一律英文。

## 用語

| 英文 | 繁中 |
|---|---|
| change、spec、main spec、delta spec | change、spec、主 spec、delta spec |
| requirement、scenario、capability | 需求、情境、capability |
| proposal、design、plan、task、step | proposal、design、plan、task、步驟 |
| record、issue、idea、the order | record、issue、idea、order |
| Full review／Light review／Check／Direct merge | 全審／輕審／查核／直接合併 |
| triage | 分流 |
| default branch、tip | 預設分支、tip |
| worktree、branch、commit、pull request | worktree、分支、commit、PR |
| finding、unreadable | finding、無法讀取 |
| workflow file | 流程檔 |
| template | 範本 |
| exit code | 結束碼 |

台灣用語：程式碼、檔案、資料夾、預設、設定、支援、執行、範本。
