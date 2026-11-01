---
title: 找出 trailer 被 git 讀不到的 task commit
---

2026-10-24 於第一次發版時記下。

## 問題

Docket 用 git 的 trailer 解析器讀 `Change:` 與 `Task:`，
而它只看 commit 訊息的最後一段。每個 trailer 各用一個 `-m` 的 commit，
會讓每個 trailer 各自成一段：

```
fix(archive): refuse a read-only target

Change: add-archive

Task: 3

Co-Authored-By: ...
```

git 於是只把 `Co-Authored-By` 讀成 trailer。`dkt sync` 與
`dkt archive` 看不到這個 task commit，而只要該 task 另有 commit 的 trailer 正常，
就沒有任何東西會回報它。第一次發版時，`add-archive` 有十個 commit
（全是審查修正輪產生的）因此沒有進它的 `evidence.md`；
它們是在「Added by hand at release」底下手動列出的。

## 方向

回報 change 範圍內的 commit：其訊息含有 `Change: <slug>` 這一行，
但 git 不把它讀成 trailer；並指出該 commit 與修法（把 trailer 放在同一個最後的段落）。
可能執行的位置：`dkt sync --check` 與 `dkt archive` 的 plan 檢查（它們本來就讀這個範圍），
或規劃中的 `dkt check` PR 關卡。這會改到 `plan-sync` 或 `change-archive` spec，所以需要一個 change。

## 待釐清問題

- 游離的 trailer 是失敗（結束碼 1）還是警告。
- 是否也要找沒有 `Change:` 的 `Task:`。
