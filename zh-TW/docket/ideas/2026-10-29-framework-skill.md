---
title: 把框架以 skill 隨附，並提供流程範本
---

2026-10-29 記錄。以下各點已與維護者定案；採用這個 idea 的 change 會把它們寫成 proposal 與 delta。

## 問題

`dkt init` 把它的指引放在 `AGENTS.md` 的「Change workflow」一節，以及 `openspec/schemas/docket/schema.yaml` 的 instructions 裡。agent 每個 session 都載入 `AGENTS.md`，所以每個 task 它們都分流，連問題與調查也是。而且這份指引描述的是一種工作方式：已經有自己工具與習慣的使用者，沒有地方把它們帶進來。

## 已決定

- 兩層：
  - **框架**，由 Docket 隨附：把工作分流到四條路徑（全審、輕審、查核、直接合併）與背後的工作種類、`dkt` 讀取或檢查什麼，以及每條路徑的**範本**。範本以 capability 的方式書寫，例如「跨模型審查」或「新開的 agent」，絕不用產品名稱；目前隨附的文字是它們的起點。
  - **使用者的流程**：使用者已有的工具與習慣，保存在使用者自己放的地方，例如他的全域 agent 指示。Docket 不把它複製進 repository。
- **具體化的流程**把兩者接起來：每條路徑，把範本調整到使用者的流程與工具允許的最接近程度。它既不是照搬的範本，也不是照搬的使用者流程。它放在使用者層級，絕不放在專案 repository（下面的定案說明放在哪裡）。
- setup：當 agent 開始第一件開發工作而找不到具體化的流程時，它檢查環境、讀範本、引導使用者，並在寫入前顯示結果。`dkt` 本身只回報事實並檢查結構：流程是否存在、是否缺少某條路徑，以及它是由哪份範本做出來的（雜湊），所以範本改動會顯示它已過期。
- 範本中的審查後備：跨模型審查；否則 subagent 審查（同一個模型、全新的 context）；否則把這件工作記錄成沒有獨立審查，由人審查 PR。自我審查不算審查步驟。
- 查找順序：專案範本（選用、提交、給團隊用）優先於 Docket 的範本；具體化的流程一律在使用者層級。第一版只建立 Docket 的範本與使用者層級的具體化流程。
- `docket-doctor` skill 檢查設定，其中包括審查流程：具體化的流程是否涵蓋每條路徑、在這個環境適用哪個審查後備，以及與框架的衝突，例如 squash merge 會丟掉 `dkt sync` 讀的 trailer。
- 框架以 `dkt init` 安裝的 skill 隨附。分流從永遠載入的指示移進它，所以只適用於 Docket 專案上的工作。不讀 skill 的 agent 不支援：`AGENTS.md` 不為它們保留指標或摘要。
- Docket 只在 `dkt init` 之後、且已安裝 OpenSpec 與 superpowers 的 repository 中提供支援。Docket 自己的 repository 與其他一樣設定。
- 由 change `add-framework-skill` 定案：框架 skill 安裝在專案中（`.claude/skills/docket/` 與
  `.agents/skills/docket/`，提交）；`AGENTS.md` 中有標記的 Docket 區塊指向它並讓它優先於 superpowers 的預設，`CLAUDE.md` 連到該區塊；Reuse 表留在框架中。
- 由 change `add-workflow` 定案：
  - 具體化的流程只有一種模式。參照模式捨棄：使用者自己的指示可能太鬆散而無法指進去，doctor 會找出漂移。
  - 它放在 `$XDG_CONFIG_HOME/docket/workflow.md` 或
    `~/.config/docket/workflow.md`。
  - 範本放在 `dkt` 裡，由 `dkt workflow template` 印出，並以雜湊辨識。
  - 對這四條路徑，流程優先於使用者的其他指示。
  - 沒有流程時，setup 在第一件工作之前執行，`dkt init` 對此什麼都不說。
  - 過期的流程仍可運作，agent 說一次就好。

## 待釐清問題

- 在這次拆分前寫好的已排入佇列的 idea，各自採用時再更新：
  `2026-10-29-records.md` 保留它的機制與類型，不含審查那幾節；`2026-10-29-commit-gate.md` 保留它對工作落在正確容器的檢查；`2026-10-28-lint-review-and-files.md` 變成對專案所宣告欄位的檢查，或被放棄。`add-records` 之後已經定案了 records 這個 idea（一半：追蹤仍在，見該檔）。

## 狀態

三個 change 都已完成：`add-framework-skill`（skill 與區塊）、`add-workflow`（具體化的流程）與 `add-doctor`（`dkt inspect` 與 `docket-doctor` 指令）。

## 備註

- setup 的既有做法：spec-kit 的 constitution 指令（agent 詢問或推斷，然後寫一個檔案）與 Claude Code 的互動式 `/init`（寫入前顯示提議）。沒有找到任何工具同時偵測既有的流程並依使用者的習慣提議一份。
- 產生的 context 檔可能降低 agent 的成功率並增加成本，長的指示檔也較少被遵循：具體化的流程要保持簡短，並讓使用者審閱。
- git worktree 不帶 git 忽略的檔案，所以在專案裡被忽略的具體化流程，會在每個新的 worktree 中缺席。
- skill 的 description 說明何時使用它，而不是它說什麼，否則 agent 會照著摘要做；description 與 `when_to_use` 合計在 1,536 個字元處被截斷。compaction 之後，載入的 skill 只保留前 5,000 個 token。用應該與不應該載入它的 prompt 來檢查載入。
