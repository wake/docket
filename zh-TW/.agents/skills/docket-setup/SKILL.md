---
name: docket-setup
description: >-
  撰寫或修訂使用者的 Docket 流程檔。使用者要求設定、更新或變更他們的 Docket 流程時使用，
  或 docket skill 因 `dkt workflow` 回報 `missing` 或 `incomplete` 而把你送來這裡時使用。
---

# Workflow setup

設定會寫出使用者的流程檔：Docket 的範本，每個步驟都寫成這位使用者的做法。在以下時機執行：

- `docket` skill 因 `dkt workflow` 在開發工作開始時回報 `missing` 或 `incomplete` 而把你送來這裡；
  先完成設定，再做那項工作；
- 使用者要求更新 `stale` 的流程檔；
- 使用者要求變更他們的流程，不論流程檔的狀態。

1. 執行 `dkt workflow --json` 取得檔案的路徑（`result.path`）與狀態（`result.status`），
   執行 `dkt workflow template --json` 取得範本（`result.text`）與它的 hash（`result.template`）。
2. 依範本逐步起草檔案，每個步驟都說明這位使用者怎麼做：
   - 先看你 context 裡已有的使用者指示，例如他們的全域 agent 指示與記憶：
     他們為規劃、審查、驗證與合併指名的審查者、模型、指令與習慣。
   - 再看你能用的工具：可呼叫來審查的其他模型、subagent 類型、plugin、skill 與 command。
   - 除非使用者同意，不要讀取、列出或搜尋不在你 context 裡的使用者層級檔案，
     例如全域指示檔或其他 agent 的設定。
   - 保留範本的每個步驟。使用者的指示說明某步驟怎麼做時，照他們的方式寫，點出工具與其設定；
     他們新增步驟時，就加上；他們與某步驟矛盾時，保留範本的步驟，並註記衝突讓使用者決定。
   - 標題完全照範本：`## Full review`、`## Light review`、`## Check`、`## Direct merge` 與 `## Reviews`。
   - 對 `incomplete`，保留檔案已有的段落，只起草缺少的；草稿仍是整份檔案。步驟 1 的
     `result.recorded` 不是範本的 hash 時，保留的段落也已過時：照 `stale` 處理。
     對 `stale`，保留使用者的工具，並帶入目前範本不同的地方。
   - 使用者要求的變更，從檔案現況出發，只改他們要求的，加上檔案處於 `stale` 或 `incomplete`
     時該處理的部分。
3. 把整份檔案依它將被寫出的樣子給使用者看，放在一個區塊中、每一段完整呈現，
   不要摘要，也不要只給新增的部分。然後列出它與範本或與他們的指示不同的每一處，
   以及待決定的衝突。修訂到他們同意為止。
4. 使用者同意之後，才把檔案寫到 `result.path`，目錄不存在時先建立。路徑是符號連結時，
   透過它寫入並保留連結。檔案以步驟 1 的 hash 開頭：

   ```
   ---
   template: "<hash>"
   ---
   ```

5. 執行 `dkt workflow`。它必須回報 `ok`；然後繼續工作（如果有的話）。

使用者現在不想設定時，就對手上的工作（如果有的話）依範本的步驟走，並在下一項工作時再問一次。
