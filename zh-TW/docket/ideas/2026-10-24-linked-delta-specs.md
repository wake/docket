---
title: 回報屬於連結的 delta spec
---

2026-10-24 審查 `add-conflicts` 時發現。

屬於連結的 delta spec 會被略過而不回報；OpenSpec 可能會跟隨連結，
所以經由連結 delta spec 造成的重疊不會被看見。要決定是否回報它們。
