---
title: 直接合併只跑會讀取所改內容的檢查
kind: prompt
source: 'maintainer, 2026-11-01: a direct merge runs only the checks related to what changed'
---

## What

流程檔範本的直接合併步驟現在只跑會讀取所改內容的檢查：沒有東西建置、嵌入或測試的文件就不跑；程式碼、測試或 manifest 則跑建置與測試。兩份 README 說法一致。CI 新增一個 `changes` job：diff 只動到那些文件時，test、openspec 與 duplicates 會跳過；lint 一律執行。

## Why

「通過自動化檢查」讓 agent 為了修改 `docket/order.md`（沒有任何測試會讀它）而跑完整個測試套件，約五分半鐘，而 CI 對每個僅文件的 pull request 又再跑一次。

## Verification

範本可由 `dkt workflow template` 印出；兩份 README 與它相符。CI 的判斷從 `ci.yml` 取出，對十六個 git diff 執行，只有 Markdown 文件會跳過；程式碼、嵌入的檔案、移進 `docs/` 的檔案、其中的 Go 檔案、被 `go:embed` 讀取的文件、空的 diff 與缺少 base 都會全部執行。取消它三條規則中的任何一條，都會讓一個案例失敗。

## Review

另一個模型的審查者發現，`docs/` 底下新增的 Go 檔案會讓測試被跳過。已修正：只有 Markdown 算作文件，而根套件嵌入的文件算作程式碼。未再審查。
