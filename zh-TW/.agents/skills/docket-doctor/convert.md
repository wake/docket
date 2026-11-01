# 將 change 轉換為 docket schema

只用於還沒有任何步驟被勾選的 change，並且要在使用者同意之後。沒有使用者在場，不 commit 任何東西。

1. 在該 change 自己的 worktree 中作業。
2. 先在 `openspec/changes/<slug>/.openspec.yaml` 設定 `schema: docket`。
3. 寫 `openspec/changes/<slug>/docket.yaml`，含 `title`、`kind` 與 `source`（各自與使用者確認），
   以及 `base`：HEAD 與預設分支的 merge base。
4. 執行 `openspec instructions proposal --change <slug> --json`，補上 proposal 缺少的段落。
5. 為 change 新增的每個 capability，在其 delta 旁加一個帶 `title` 的 `docket.yaml`
   （`specs/<capability>/docket.yaml`）。
6. 用 superpowers:writing-plans 依 `tasks.md` 寫 `plan.md`，依
   `openspec instructions plan --change <slug>`，然後刪除 `tasks.md`。
7. 執行 `dkt lint <slug>`，直到它什麼都不回報。

該 change 接著從它的 plan 階段繼續：流程中的 plan 審查是下一步。
