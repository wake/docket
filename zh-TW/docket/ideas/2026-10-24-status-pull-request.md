---
title: 顯示 change 的 PR
---

2026-10-24 `add-status` 刻意不放進 `dkt status` 的項目。

找 PR 需要託管服務的 API，而 `dkt status` 從不對外取得或呼叫。
workbench 可以把 PR 接到它從 `status --json` 讀到的 `slug` 上。
