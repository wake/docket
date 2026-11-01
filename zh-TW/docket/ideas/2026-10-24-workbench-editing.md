---
title: Workbench 的編輯與指令
needs:
  - docket/ideas/2026-10-24-web-workbench.md
---

2026-10-24 記為舊 order 的第 8 項：唯讀 workbench
（`docket/ideas/2026-10-24-web-workbench.md`）留待之後的部分。

- 在 worktree 裡編輯文件，寫入前做比對檢查，讓編輯不會蓋掉 agent 剛才做的修改。
  只存在於分支上的 change 維持唯讀。
- 從 UI 執行 Docket 指令（lint、conflicts、sync、archive）並顯示其 finding。
